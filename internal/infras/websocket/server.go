package websocket

import (
	"context"
	"encoding/json"
	"fmt"
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
	PreviousSegmentID        string
	PreviousSegmentExpiredAt time.Time
	StartSessionTime         time.Time
	lastPongTime             time.Time
	pongReceived             chan struct{}
	connected                bool
	cancelFunc               context.CancelFunc
}

type WebSocketServerInterface interface {
	HandleConnection(ctx context.Context, w http.ResponseWriter, r *http.Request, session *entities.IsSessionValidResp) error
	Start(ctx context.Context) error
	SendCloseMessage(ctx context.Context, sessionID string, reason string) error
	Disconnect(ctx context.Context, client *Client)
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
	}

	logic := NewWebSocketServerLogic(
		log,
		server.Disconnect,
		server.writeJSON,
		clientManager,
		interviewSessionService,
		redisClient,
		generator,
		publisher,
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
		PreviousSegmentID:        "",
		PreviousSegmentExpiredAt: time.Now(),
		StartSessionTime:         time.Now(),
		lastPongTime:             time.Now(),
		pongReceived:             make(chan struct{}, 1),
		connected:                true,
	}

	s.log.InfoWithID(ctx, "[WebSocketServer: HandleConnection] Setting read deadline", map[string]any{
		"session_id": client.SessionID,
	})
	// _ = client.conn.SetReadDeadline(time.Now().Add(constants.WebSocketReadTimeout))
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
		"type": "connection_established", "session_id": client.SessionID,
	})

	s.logic.sendStartSessionConversationMessage(ctx, client)
	// go s.pingLoop(cancelCtx, client)
	go s.readLoop(cancelCtx, client)

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
	defer s.Disconnect(ctx, c)
	c.conn.SetReadLimit(1 << 20)

	for {
		mt, payload, err := c.conn.ReadMessage()
		// _ = c.conn.SetReadDeadline(time.Now().Add(constants.WebSocketReadTimeout))

		if err != nil {
			s.log.ErrorWithID(ctx, "[WebSocketServer: readLoop] Error reading message", err)
			s.logic.sendMessageTypeError(ctx, c, app_error.ErrCodeWebSocketInvalidMessage)
			return
		}

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

			case constants.WebSocketMessageTypeClose:
				s.log.InfoWithID(ctx, "[WebSocketServer] Received close message from client", map[string]any{
					"session_id": c.SessionID,
					"user_id":    c.userID,
				})
				s.Disconnect(ctx, c)
				return

			default:
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

func (s *webSocketServer) pingLoop(ctx context.Context, client *Client) {
	s.log.InfoWithID(ctx, "[WebSocketServer: pingLoop] Starting ping loop")

	t := time.NewTicker(constants.WebSocketPingInterval)
	defer t.Stop()

	for range t.C {
		client.mu.Lock()
		timeSinceLastPong := time.Since(client.lastPongTime)
		client.mu.Unlock()

		if timeSinceLastPong > constants.WebSocketPongTimeout {
			s.log.ErrorWithID(ctx, "[WebSocketServer: pingLoop] Pong timeout - no pong received", map[string]interface{}{
				"session_id":           client.SessionID,
				"time_since_last_pong": timeSinceLastPong,
				"timeout":              constants.WebSocketPongTimeout,
			})
			s.Disconnect(ctx, client)
			return
		}

		client.mu.Lock()
		err := client.conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(constants.WebSocketPingDuration))
		client.mu.Unlock()
		if err != nil {
			s.log.ErrorWithID(ctx, "[WebSocketServer: pingLoop] Error writing ping message", err)
			s.Disconnect(ctx, client)
			return
		}
	}
}

func (s *webSocketServer) Disconnect(ctx context.Context, client *Client) {
	s.log.InfoWithID(ctx, "[WebSocketServer: disconnect] Called")

	s.mu.Lock()
	delete(s.sessions, client.SessionID)
	if set := s.userSessions[client.userID]; set != nil {
		delete(set, client.SessionID)
		if len(set) == 0 {
			delete(s.userSessions, client.userID)
		}
	}
	s.mu.Unlock()

	s.writeJSON(ctx, client, map[string]any{
		"type":    "disconnect",
		"code":    string(app_error.ErrCodeWebSocketInvalidMessage),
		"message": app_error.ErrCodeWebSocketInvalidMessage.Message(),
	})

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
	if client.connected {
		client.connected = false
		close(client.pongReceived)
	}
	client.mu.Unlock()
}

func (s *webSocketServer) writeJSON(ctx context.Context, c *Client, v any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	err := c.conn.WriteJSON(v)
	if err != nil {
		s.log.ErrorWithID(ctx, "[WebSocketServer: writeJSON] Error writing JSON", err)
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
