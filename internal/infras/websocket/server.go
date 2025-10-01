package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"gitlab.com/interview-simulation/interview-backend-server/internal/config"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	"gitlab.com/interview-simulation/interview-backend-server/internal/entities"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/database"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/queue/publisher"
	"gitlab.com/interview-simulation/interview-backend-server/internal/middleware"
	"gitlab.com/interview-simulation/interview-backend-server/internal/services"
	"gitlab.com/interview-simulation/interview-backend-server/internal/utils"
)

type Client struct {
	conn                     *websocket.Conn
	mu                       sync.Mutex
	userID                   string
	SessionID                string
	resumeID                 string
	CurrentSegmentID         string
	LastTurnID               string
	PreviousSegmentID        string
	PreviousSegmentExpiredAt time.Time
	StartSessionTime         time.Time
	lastPongTime             time.Time
	lastActivityTime         time.Time
	inactivityWarningSent    bool
	isStartedConversation    bool
	pongReceived             chan struct{}
	connected                bool
	cancelFunc               context.CancelFunc
	currentState             string
	currentStateID           string
	disconnecting            bool
	biasPrompt               string
}

type WebSocketServerInterface interface {
	HandleConnection(ctx context.Context, w http.ResponseWriter, r *http.Request, session *entities.IsSessionValidResp) error
	Start(ctx context.Context) error
	SendCloseMessage(ctx context.Context, sessionID string, reason string) error
	Disconnect(ctx context.Context, client *Client, status ...string)
}

type webSocketServer struct {
	log                     *log.Logger
	sessions                map[string]*Client
	userSessions            map[string]map[string]bool
	mu                      sync.RWMutex
	upgrader                websocket.Upgrader
	authContext             middleware.AuthContext
	interviewSessionService services.InterviewSessionService
	aiAgentConnected        bool
	clientManager           *ClientManager
	cfg                     *config.Config
	logic                   *WebSocketServerLogic
	jwtMaker                utils.JwtToken
	publisher               publisher.RedisTaskPublisher
}

func NewWebSocketServer(
	log *log.Logger,
	redisClient database.RedisClient,
	authContext middleware.AuthContext,
	interviewSessionService services.InterviewSessionService,
	clientManager *ClientManager,
	cfg *config.Config,
	jwtMaker utils.JwtToken,
	generator utils.Generator,
	publisher publisher.RedisTaskPublisher,
	evaluationService services.EvaluationService,
) WebSocketServerInterface {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  64 << 10,
		WriteBufferSize: 64 << 10,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	server := &webSocketServer{
		log:                     log,
		upgrader:                upgrader,
		sessions:                make(map[string]*Client),
		userSessions:            make(map[string]map[string]bool),
		authContext:             authContext,
		interviewSessionService: interviewSessionService,
		aiAgentConnected:        false,
		clientManager:           clientManager,
		cfg:                     cfg,
		logic:                   nil,
		jwtMaker:                jwtMaker,
		publisher:               publisher,
	}

	logic := NewWebSocketServerLogic(
		log,
		func(ctx context.Context, client *Client) {
			server.Disconnect(ctx, client)
		},
		server.writeJSON,
		clientManager,
		interviewSessionService,
		redisClient,
		generator,
		publisher,
		evaluationService,
	)

	server.logic = logic

	return server
}

func (s *webSocketServer) Start(ctx context.Context) error {
	s.log.InfoWithID(ctx, "[WebSocketServer: Start] Starting WebSocket server")

	s.aiAgentConnected = true
	return nil
}

func (s *webSocketServer) HandleConnection(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	session *entities.IsSessionValidResp,
) error {
	s.log.InfoWithID(ctx, "[WebSocketServer: HandleConnection] Called")

	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.log.ErrorWithID(ctx, "[WebSocketServer: HandleConnection] Error upgrading connection", err)
		return nil
	}

	client := &Client{
		conn:                     conn,
		userID:                   session.UserID,
		SessionID:                session.SessionID,
		resumeID:                 session.ResumeID,
		CurrentSegmentID:         "",
		LastTurnID:               "",
		PreviousSegmentID:        "",
		PreviousSegmentExpiredAt: time.Now(),
		StartSessionTime:         time.Now(),
		lastPongTime:             time.Now(),
		lastActivityTime:         time.Now(),
		inactivityWarningSent:    false,
		pongReceived:             make(chan struct{}, 1),
		connected:                true,
		isStartedConversation:    false,
		currentState:             "",
		currentStateID:           "",
		biasPrompt:               "",
	}

	// client.conn.SetPongHandler(func(string) error {
	// 	client.mu.Lock()
	// 	client.lastPongTime = time.Now()
	// 	client.mu.Unlock()

	// 	select {
	// 	case client.pongReceived <- struct{}{}:
	// 	default:
	// 	}

	// 	return client.conn.SetReadDeadline(time.Now().Add(constants.WebSocketReadTimeout))
	// })

	s.mu.Lock()
	s.sessions[client.SessionID] = client
	if s.userSessions[client.userID] == nil {
		s.userSessions[client.userID] = map[string]bool{}
	}
	s.userSessions[client.userID][client.SessionID] = true
	s.mu.Unlock()

	resp, err := s.logic.getInterviewSessionState(ctx, client)
	if err != nil {
		s.log.ErrorWithID(ctx, "[WebSocketServer: HandleConnection] Error checking exists and initializing started at interview session", err)
		s.Disconnect(ctx, client)
		return nil
	}

	if resp.IsTimedOut {
		s.log.InfoWithID(ctx, "[WebSocketServer: HandleConnection] Interview session timed out")
		s.writeJSON(ctx, client, map[string]any{
			"type": constants.WebSocketMessageTypeInterviewSessionAlreadyTimedOut,
		})

		s.Disconnect(ctx, client, constants.StatusAlreadyTimedOut)
		return nil
	}

	client.biasPrompt = resp.BiasPrompt
	client.currentState = resp.CurrentState
	client.currentStateID = resp.CurrentStateID

	if err := s.initClient(ctx, client); err != nil {
		s.log.ErrorWithID(ctx, "[WebSocketServer: HandleConnection] Error initializing client", err)
		s.Disconnect(ctx, client)
		return nil
	}

	interviewReq := &entities.UpdateInterviewSessionStatusReq{
		SessionID: client.SessionID,
		Status:    constants.StatusOnGoing,
	}

	if err := s.interviewSessionService.UpdateInterviewSessionStatus(ctx, interviewReq); err != nil {
		s.log.ErrorWithID(ctx, "[WebSocketServer: HandleConnection] Error starting interview session", err)
		s.Disconnect(ctx, client)
		return nil
	}

	cancelCtx, cancel := context.WithCancel(context.Background())
	client.cancelFunc = cancel

	s.writeJSON(ctx, client, map[string]any{
		"type":       constants.WebSocketMessageTypeConnectionEstablished,
		"started_at": resp.StartedAt,
		"session_id": client.SessionID,
	})
	client.StartSessionTime = utils.ParseToTime(resp.StartedAt)

	if !resp.IsStartedConversation {
		s.logic.sendStartSessionConversationMessage(ctx, client)
	} else {
		client.isStartedConversation = true
		s.writeJSON(ctx, client, map[string]any{
			"type":       constants.WebSocketMessageTypeConversationStarted,
			"session_id": client.SessionID,
		})
	}

	// go s.pingLoop(cancelCtx, client)
	go s.readLoop(cancelCtx, client)
	go s.inactivityMonitor(cancelCtx, client)

	return nil
}

func (s *webSocketServer) initClient(ctx context.Context, client *Client) error {
	s.log.InfoWithID(ctx, "[WebSocketServer: initClient] Called")

	token, err := s.jwtMaker.CreateWebSocketSessionToken(ctx, &entities.WebSocketSessionReq{
		UserID:    client.userID,
		SessionID: client.SessionID,
		ResumeID:  client.resumeID,
		Duration:  s.cfg.InterviewSessionConfig.InterviewSessionTokenTTL,
	})
	if err != nil {
		s.log.ErrorWithID(ctx, "[WebSocketServer: HandleConnection] Error creating web socket session token", err)
		s.Disconnect(ctx, client)
		return err
	}

	u, err := url.Parse(s.cfg.InterviewSessionConfig.InterviewWebsocketPath)
	if err != nil {
		s.log.ErrorWithID(ctx, "[WebSocketServer: HandleConnection] Invalid agent WS URL", err)
		s.Disconnect(ctx, client)
		return err
	}

	q := u.Query()
	q.Set("token", token)
	u.RawQuery = q.Encode()

	agentClient := NewWebSocketClient(s.log)
	callbacks := NewWebSocketClientCallbacks(s, s.logic, client, s.log)

	agentClient.SetCallbacks(callbacks)
	if err := agentClient.Start(ctx, u.String()); err != nil {
		s.log.ErrorWithID(ctx, "[WebSocketServer: HandleConnection] Error starting agent client: ", err)
		s.Disconnect(ctx, client)
		return err
	}

	_ = agentClient.SendSessionInfo(ctx, client.SessionID, client.userID, client.resumeID)
	s.clientManager.SetClientBySessionID(ctx, client.SessionID, agentClient)

	return nil
}

func (s *webSocketServer) readLoop(ctx context.Context, c *Client) {
	s.log.InfoWithID(ctx, "[WebSocketServer: readLoop] Called")

	defer func() {
		s.log.ErrorWithID(ctx, "[WebSocketServer: readLoop] Disconnecting client...")
		s.Disconnect(ctx, c)
	}()

	c.conn.SetReadLimit(1 << 20)

	for {
		mt, payload, err := c.conn.ReadMessage()

		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				s.log.ErrorWithID(ctx, "[WebSocketServer: readLoop] Connection timeout - ending session due to inactivity")

				s.writeJSON(ctx, c, map[string]any{
					"type":   constants.WebSocketMessageTypeInterviewSessionTimedOut,
					"reason": constants.TimeOutMessage,
				})

				s.Disconnect(ctx, c, constants.StatusTimedOut)
				return
			}

			if closeErr, ok := err.(*websocket.CloseError); ok {
				s.log.InfoWithID(ctx, "[WebSocketServer: readLoop] Client closed connection", map[string]any{
					"session_id": c.SessionID,
					"user_id":    c.userID,
					"code":       closeErr.Code,
					"text":       closeErr.Text,
				})
			} else if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
				s.log.ErrorWithID(ctx, "[WebSocketServer: readLoop] Unexpected connection close", err)
			} else {
				s.log.ErrorWithID(ctx, "[WebSocketServer: readLoop] Connection closed With Error", err)
			}

			c.mu.Lock()
			c.connected = false
			c.mu.Unlock()

			return
		}

		c.mu.Lock()
		c.lastActivityTime = time.Now()
		c.inactivityWarningSent = false
		c.mu.Unlock()

		switch mt {
		case websocket.TextMessage:
			var m struct {
				Type string `json:"type"`
			}
			if json.Unmarshal(payload, &m) != nil {
				continue
			}

			switch m.Type {

			case constants.WebSocketMessageTypeSegmentStart:
				s.logic.SendMessageTypeSegmentStart(ctx, c, payload)
				continue

			case constants.WebSocketMessageTypeSegmentEnd:
				s.logic.sendMessageTypeSegmentEnd(ctx, c, payload)
				continue

			case constants.WebSocketMessageTypeEndInterviewSession:
				s.logic.endInterviewSession(ctx, c, payload)
				continue

			case constants.WebSocketMessageTypeClose:
				s.log.InfoWithID(ctx, "[WebSocketServer] Received close message from client", map[string]any{
					"session_id": c.SessionID,
					"user_id":    c.userID,
				})
				s.Disconnect(ctx, c)
				return

			default:
				s.log.ErrorWithID(ctx, "[WebSocketServer: readLoop] Invalid message type", map[string]any{"message_type": m.Type})
				s.logic.sendMessageTypeError(ctx, c, app_error.ErrCodeWebSocketInvalidMessage)
				continue
			}

		case websocket.BinaryMessage:
			s.logic.handleUserAudioBinaryMessage(ctx, c, payload)

		default:
			s.log.InfoWithID(ctx, "[WebSocketServer] Ignoring frame type", map[string]any{"frame_type": mt})
		}
	}
}

func (s *webSocketServer) inactivityMonitor(ctx context.Context, client *Client) {
	s.log.InfoWithID(ctx, "[WebSocketServer: inactivityMonitor] Starting inactivity monitor")

	ticker := time.NewTicker(constants.WebSocketInactivityMonitorInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.log.InfoWithID(ctx, "[WebSocketServer: inactivityMonitor] Context cancelled, stopping inactivity monitor")
			return
		case <-ticker.C:
			client.mu.Lock()
			if !client.connected || client.disconnecting {
				client.mu.Unlock()
				return
			}

			timeSinceLastActivity := time.Since(client.lastActivityTime)
			warningSent := client.inactivityWarningSent
			client.mu.Unlock()

			if !warningSent && timeSinceLastActivity >= constants.WebSocketInactivityWarningTimeout {
				s.log.InfoWithID(ctx, "[WebSocketServer: inactivityMonitor] Sending inactivity warning")

				s.writeJSON(ctx, client, map[string]any{
					"type":    constants.WebSocketMessageTypeInactivityWarning,
					"message": constants.InactivityWarningMessage,
				})

				client.mu.Lock()
				client.inactivityWarningSent = true
				client.mu.Unlock()
			}

			if timeSinceLastActivity >= constants.WebSocketInactivityTimeoutDuration {
				s.log.InfoWithID(ctx, "[WebSocketServer: inactivityMonitor] Session timed out due to inactivity")

				s.writeJSON(ctx, client, map[string]any{
					"type":    constants.WebSocketMessageTypeInterviewSessionTimedOut,
					"message": constants.TimeOutMessage,
				})

				s.Disconnect(ctx, client, constants.StatusTimedOut)
				return
			}
		}
	}
}

// func (s *webSocketServer) pingLoop(ctx context.Context, client *Client) {
// 	s.log.InfoWithID(ctx, "[WebSocketServer: pingLoop] Starting ping loop")

// 	t := time.NewTicker(constants.WebSocketPingInterval)
// 	defer t.Stop()

// 	for range t.C {
// 		client.mu.Lock()
// 		timeSinceLastPong := time.Since(client.lastPongTime)
// 		client.mu.Unlock()

// 		if timeSinceLastPong > constants.WebSocketPongTimeout {
// 			s.log.ErrorWithID(ctx, "[WebSocketServer: pingLoop] Pong timeout - no pong received", map[string]interface{}{
// 				"session_id":           client.SessionID,
// 				"time_since_last_pong": timeSinceLastPong,
// 				"timeout":              constants.WebSocketPongTimeout,
// 			})
// 			s.Disconnect(ctx, client)
// 			return
// 		}

// 		client.mu.Lock()
// 		err := client.conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(constants.WebSocketPingDuration))
// 		client.mu.Unlock()
// 		if err != nil {
// 			s.log.ErrorWithID(ctx, "[WebSocketServer: pingLoop] Error writing ping message", err)
// 			s.Disconnect(ctx, client)
// 			return
// 		}
// 	}
// }

func (s *webSocketServer) Disconnect(ctx context.Context, client *Client, status ...string) {
	s.log.InfoWithID(ctx, "[WebSocketServer: disconnect] Called")

	client.mu.Lock()
	if client.disconnecting {
		client.mu.Unlock()
		s.log.InfoWithID(ctx, "[WebSocketServer: disconnect] Already disconnecting")
		return
	}
	client.disconnecting = true
	client.connected = false
	client.mu.Unlock()

	sessionStatus := constants.StatusCancelled
	if len(status) > 0 && status[0] != "" {
		sessionStatus = status[0]
	}

	endInterviewReq := &entities.EndInterviewSessionReq{
		SessionId: client.SessionID,
		Status:    sessionStatus,
	}

	if sessionStatus != constants.StatusAlreadyTimedOut {
		s.log.InfoWithID(ctx, "[WebSocketServer: disconnect] Publishing end interview session task", endInterviewReq)

		endInterviewPayload := &entities.EndInterviewSessionPayload{
			SessionID: endInterviewReq.SessionId,
			Status:    endInterviewReq.Status,
		}

		if err := s.logic.publisher.PublishTaskEndInterviewSession(context.Background(), endInterviewPayload); err != nil {
			s.log.ErrorWithID(ctx, "[WebSocketServer: disconnect] Error publishing end interview session task", err)
			if fallbackErr := s.interviewSessionService.EndInterviewSession(context.Background(), endInterviewReq); fallbackErr != nil {
				s.log.ErrorWithID(ctx, "[WebSocketServer: disconnect] Error finalizing session phrase evaluation (fallback)", fallbackErr)
			}
		}
	}

	s.mu.Lock()
	delete(s.sessions, client.SessionID)
	if set := s.userSessions[client.userID]; set != nil {
		delete(set, client.SessionID)
		if len(set) == 0 {
			delete(s.userSessions, client.userID)
		}
	}
	s.mu.Unlock()

	if agentClient, exists := s.clientManager.GetClientBySessionID(ctx, client.SessionID); exists {
		s.log.InfoWithID(ctx, "[WebSocketServer: disconnect] Closing AI agent client", map[string]any{
			"session_id": client.SessionID,
		})
		_ = agentClient.Close(ctx)
	}
	s.clientManager.DeleteClientBySessionID(ctx, client.SessionID)

	_ = client.conn.Close()

	if client.cancelFunc != nil {
		client.cancelFunc()
		client.cancelFunc = nil
	}

	client.mu.Lock()
	if !client.disconnecting {
		close(client.pongReceived)
	}
	client.mu.Unlock()
}

func (s *webSocketServer) writeJSON(ctx context.Context, c *Client, v any) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected || c.conn == nil {
		s.log.ErrorWithID(ctx, "[WebSocketServer: writeJSON] Connection already closed, skipping write")
		return
	}

	err := c.conn.WriteJSON(v)
	if err != nil {
		// Check if it's a close error to avoid logging it as an error
		if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
			s.log.InfoWithID(ctx, "[WebSocketServer: writeJSON] Connection closed during write", map[string]any{
				"error": err.Error(),
			})
		} else {
			s.log.ErrorWithID(ctx, "[WebSocketServer: writeJSON] Error writing JSON", err)
		}
	}
}

func (s *webSocketServer) SendCloseMessage(ctx context.Context, sessionID string, reason string) error {
	s.mu.RLock()
	client, exists := s.sessions[sessionID]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("client not found for session %s", sessionID)
	}

	s.writeJSON(ctx, client, map[string]any{
		"v":      1,
		"type":   constants.WebSocketMessageTypeClose,
		"reason": reason,
	})

	time.Sleep(100 * time.Millisecond)
	s.Disconnect(ctx, client)

	return nil
}
