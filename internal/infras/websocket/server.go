package websocket

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	"gitlab.com/interview-simulation/interview-backend-server/internal/entities"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/database"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
	"gitlab.com/interview-simulation/interview-backend-server/internal/middleware"
	"gitlab.com/interview-simulation/interview-backend-server/internal/repositories"
	"gitlab.com/interview-simulation/interview-backend-server/internal/services"
)

type Client struct {
	conn             *websocket.Conn
	mu               sync.Mutex
	userID           string
	sessionID        string
	currentSegmentID string
	lastPongTime     time.Time
	pongReceived     chan struct{}
}

type WebSocketServerInterface interface {
	HandleConnection(ctx context.Context, w http.ResponseWriter, r *http.Request, payloadReq entities.OpenWsConnectionRequest) error
	Start(ctx context.Context) error
	Close() error
}

type webSocketServer struct {
	log                     *log.Logger
	sessions                map[string]*Client
	userSessions            map[string]map[string]bool
	mu                      sync.RWMutex
	upgrader                websocket.Upgrader
	redisClient             database.RedisClient
	authContext             middleware.AuthContext
	interviewSessionService services.InterviewSessionService
	interviewSessionRepo    repositories.InterviewSessionRepository
	webSocketClient         WebSocketClient
	aiAgentConnected        bool
}

func NewWebSocketServer(
	log *log.Logger,
	redisClient database.RedisClient,
	authContext middleware.AuthContext,
	interviewSessionService services.InterviewSessionService,
	interviewSessionRepo repositories.InterviewSessionRepository,
	webSocketClient WebSocketClient,
) WebSocketServerInterface {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  64 << 10,
		WriteBufferSize: 64 << 10,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	return &webSocketServer{
		log:                     log,
		upgrader:                upgrader,
		redisClient:             redisClient,
		sessions:                make(map[string]*Client),
		userSessions:            make(map[string]map[string]bool),
		authContext:             authContext,
		interviewSessionService: interviewSessionService,
		interviewSessionRepo:    interviewSessionRepo,
		webSocketClient:         webSocketClient,
		aiAgentConnected:        false,
	}
}

func (s *webSocketServer) Start(ctx context.Context) error {
	s.log.InfoWithID(ctx, "[WebSocketServer: Start] Starting WebSocket server")

	if err := s.webSocketClient.Start(ctx); err != nil {
		s.log.ErrorWithID(ctx, "[WebSocketServer: Start] Failed to connect to AI agent", err)
		return err
	}

	s.aiAgentConnected = true

	s.log.InfoWithID(ctx, "[WebSocketServer: Start] WebSocket server started successfully")
	return nil
}

func (s *webSocketServer) Close() error {
	s.log.InfoWithID(context.Background(), "[WebSocketServer: Close] Closing WebSocket server")

	s.mu.Lock()
	for _, client := range s.sessions {
		_ = client.conn.Close()
	}
	s.sessions = make(map[string]*Client)
	s.userSessions = make(map[string]map[string]bool)
	s.mu.Unlock()

	if s.webSocketClient != nil {
		_ = s.webSocketClient.Close()
	}

	s.aiAgentConnected = false
	return nil
}

func (s *webSocketServer) HandleConnection(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	payloadReq entities.OpenWsConnectionRequest,
) error {
	s.log.InfoWithID(ctx, "[WebSocketServer: HandleConnection] Called")

	redisSessionToken, err := s.redisClient.Get(ctx, payloadReq.SessionToken)
	if err != nil {
		s.log.ErrorWithID(ctx, "[WebSocketServer: HandleConnection] Error getting redis session token", err)
		return err
	}

	var sessionPayload entities.RedisSessionToken
	if err := json.Unmarshal([]byte(redisSessionToken), &sessionPayload); err != nil {
		s.log.ErrorWithID(ctx, "[WebSocketServer: HandleConnection] Error unmarshalling redis session token", err)
		return err
	}

	if sessionPayload.UserID != payloadReq.UserID {
		s.log.ErrorWithID(ctx, "[WebSocketServer: HandleConnection] User ID mismatch")
		return app_error.New(constants.ErrInvalidToken, app_error.ErrCodeSessionInvalidToken)
	}

	exists, err := s.interviewSessionRepo.CheckInterviewSessionExists(ctx, uuid.MustParse(sessionPayload.SessionID))
	if err != nil {
		s.log.ErrorWithID(ctx, "[WebSocketServer: HandleConnection] Error checking interview session exists", err)
		return err
	}
	if !exists {
		s.log.ErrorWithID(ctx, "[WebSocketServer: HandleConnection] Interview session not found")
		return app_error.New(constants.ErrInvalidToken, app_error.ErrCodeSessionNotFound)
	}

	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.log.ErrorWithID(ctx, "[WebSocketServer: HandleConnection] Error upgrading connection", err)
		return err
	}

	client := &Client{
		conn:         conn,
		userID:       sessionPayload.UserID,
		sessionID:    sessionPayload.SessionID,
		lastPongTime: time.Now(),
		pongReceived: make(chan struct{}, 1),
	}

	_ = client.conn.SetReadDeadline(time.Now().Add(constants.WebSocketReadTimeout))
	client.conn.SetPongHandler(func(string) error {
		client.mu.Lock()
		client.lastPongTime = time.Now()
		client.mu.Unlock()

		select {
		case client.pongReceived <- struct{}{}:
		default:
		}

		return client.conn.SetReadDeadline(time.Now().Add(constants.WebSocketReadTimeout))
	})

	s.mu.Lock()
	s.sessions[client.sessionID] = client
	if s.userSessions[client.userID] == nil {
		s.userSessions[client.userID] = map[string]bool{}
	}
	s.userSessions[client.userID][client.sessionID] = true
	s.mu.Unlock()

	interviewReq := &entities.UpdateInterviewSessionStatusReq{
		SessionID: client.sessionID,
		Status:    constants.StatusOnGoing,
	}

	if err := s.interviewSessionService.UpdateInterviewSessionStatus(ctx, interviewReq); err != nil {
		s.log.ErrorWithID(ctx, "[WebSocketServer: HandleConnection] Error starting interview session", err)
		s.disconnect(client)
		return err
	}

	if s.aiAgentConnected && s.webSocketClient.IsConnected() {
		if err := s.webSocketClient.SendSessionInfo(ctx, client.sessionID, client.userID); err != nil {
			s.log.ErrorWithID(ctx, "[WebSocketServer: HandleConnection] Error sending session info to AI agent", err)
			s.disconnect(client)
			return err
		}
	}

	_ = s.writeJSON(client, map[string]any{
		"v": 1, "type": "connection_established", "session_id": client.sessionID,
	})

	go s.pingLoop(client)
	go s.readLoop(ctx, client)

	return nil
}

func (s *webSocketServer) readLoop(ctx context.Context, c *Client) {
	s.log.InfoWithID(ctx, "[WebSocketServer: readLoop] Called")
	defer s.disconnect(c)
	c.conn.SetReadLimit(1 << 20)

	for {
		mt, payload, err := c.conn.ReadMessage()

		_ = c.conn.SetReadDeadline(time.Now().Add(constants.WebSocketReadTimeout))

		if err != nil {
			s.log.ErrorWithID(ctx, "[WebSocketServer: readLoop] Error reading message", err)
			_ = s.writeJSON(c, msgError{
				msgBase: msgBase{1, "error"},
				Code:    string(app_error.ErrCodeWebSocketInvalidMessage),
				Message: app_error.ErrCodeWebSocketInvalidMessage.Message(),
			})
			s.disconnect(c)
			return
		}

		switch mt {
		case websocket.TextMessage:

			var base struct {
				V    int    `json:"v"`
				Type string `json:"type"`
			}
			if json.Unmarshal(payload, &base) != nil {
				continue
			}

			switch base.Type {

			case constants.WebSocketMessageTypeHello:
				s.sendMessageTypeHello(ctx, c, payload)
				continue

			case constants.WebSocketMessageTypeSegmentStart:
				s.sendMessageTypeSegmentStart(ctx, c, payload)
				continue

			case constants.WebSocketMessageTypeSegmentEnd:
				s.sendMessageTypeSegmentEnd(ctx, c, payload)
				continue

			default:
				s.sendMessageTypeError(ctx, c, payload)
				continue
			}

		case websocket.BinaryMessage:
			var base struct {
				V    int    `json:"v"`
				Type string `json:"type"`
			}
			if json.Unmarshal(payload, &base) != nil {
				continue
			}

			switch base.Type {
			case constants.WebSocketBineryTypeAudioChunk:
				s.sendBinaryMessageTypeAudioChunk(ctx, c, payload)
				continue

			default:
				s.sendMessageTypeError(ctx, c, payload)
				continue
			}

		default:
			// ignore other frame types, like pong
		}
	}
}

func (s *webSocketServer) sendMessageTypeHello(ctx context.Context, client *Client, payload []byte) {
	s.log.InfoWithID(ctx, "[WebSocketServer: sendMessageTypeHello] Called")
	var m msgHello
	if json.Unmarshal(payload, &m) != nil || m.SessionID == "" {
		_ = s.writeJSON(client, msgError{
			msgBase: msgBase{1, "error"},
			Code:    string(app_error.ErrCodeWebSocketInvalidHello),
			Message: app_error.ErrCodeWebSocketInvalidHello.Message(),
		})
		return
	}

	client.sessionID = m.SessionID
	_ = s.writeJSON(client, map[string]any{"v": 1, "type": "hello", "session_id": client.sessionID})

	if s.aiAgentConnected && s.webSocketClient.IsConnected() {
		if err := s.webSocketClient.SendMessage(ctx, "hello", map[string]interface{}{
			"session_id": client.sessionID,
		}); err != nil {
			s.log.ErrorWithID(ctx, "[WebSocketServer: readLoop] Error sending hello to AI agent", err)
		}
	}
}

func (s *webSocketServer) sendMessageTypeSegmentStart(ctx context.Context, client *Client, payload []byte) {
	s.log.InfoWithID(ctx, "[WebSocketServer: sendMessageTypeSegmentStart] Called")
	var m msgSegmentStart
	if json.Unmarshal(payload, &m) != nil || m.SegmentID == "" {
		_ = s.writeJSON(client, msgError{
			msgBase: msgBase{1, "error"},
			Code:    string(app_error.ErrCodeWebSocketInvalidSegmentStart),
			Message: app_error.ErrCodeWebSocketInvalidSegmentStart.Message(),
		})
		return
	}
	client.currentSegmentID = m.SegmentID

	if s.aiAgentConnected && s.webSocketClient.IsConnected() {
		if err := s.webSocketClient.SegmentStart(ctx, m.SegmentID, m.SampleRate, m.Encoding, m.Channels); err != nil {
			s.log.ErrorWithID(ctx, "[WebSocketServer: readLoop] Error forwarding segment start to AI agent", err)
		}
	}

	s.storeConversationTurn(ctx, client, &ConversationTurn{
		SessionID:  client.sessionID,
		SegmentID:  m.SegmentID,
		TurnType:   "user_audio",
		SampleRate: m.SampleRate,
		Encoding:   m.Encoding,
		Channels:   m.Channels,
		StartedAt:  time.Now(),
	})
}

func (s *webSocketServer) sendMessageTypeSegmentEnd(ctx context.Context, client *Client, payload []byte) {
	s.log.InfoWithID(ctx, "[WebSocketServer: sendMessageTypeSegmentEnd] Called")
	var m msgSegmentEnd
	if json.Unmarshal(payload, &m) != nil || m.SegmentID == "" {
		_ = s.writeJSON(client, msgError{
			msgBase: msgBase{1, "error"},
			Code:    string(app_error.ErrCodeWebSocketInvalidSegmentEnd),
			Message: app_error.ErrCodeWebSocketInvalidSegmentEnd.Message(),
		})
		return
	}
	if client.currentSegmentID != m.SegmentID {
		_ = s.writeJSON(client, msgError{
			msgBase: msgBase{1, "error"},
			Code:    string(app_error.ErrCodeWebSocketInvalidSegmentEnd),
			Message: app_error.ErrCodeWebSocketInvalidSegmentEnd.Message(),
		})
		return
	}

	if s.aiAgentConnected && s.webSocketClient.IsConnected() {
		if err := s.webSocketClient.SegmentEnd(ctx, m.SegmentID); err != nil {
			s.log.ErrorWithID(ctx, "[WebSocketServer: readLoop] Error forwarding segment end to AI agent", err)
		}
	}

	s.updateConversationTurnEndTime(ctx, m.SegmentID, time.Now())

	client.currentSegmentID = ""
}

func (s *webSocketServer) sendBinaryMessageTypeAudioChunk(ctx context.Context, client *Client, payload []byte) {
	s.log.InfoWithID(ctx, "[WebSocketServer: sendBinaryMessageTypeAudioChunk] Called")
	// var base struct {
	// 	V    int    `json:"v"`
	// 	Type string `json:"type"`
	// }
	// if json.Unmarshal(payload, &base) != nil {
	// 	_ = s.writeJSON(client, msgError{msgBase{1, "error"}, string(app_error.ErrCodeWebSocketInvalidAudioChunk), app_error.ErrCodeWebSocketInvalidAudioChunk.Message()})
	// 	return
	// }
}

func (s *webSocketServer) sendMessageTypeError(ctx context.Context, client *Client, payload []byte) {
	s.log.InfoWithID(ctx, "[WebSocketServer: sendMessageTypeError] Called")
	_ = s.writeJSON(client, msgError{
		msgBase: msgBase{1, "error"},
		Code:    string(app_error.ErrCodeWebSocketInvalidMessage),
		Message: app_error.ErrCodeWebSocketInvalidMessage.Message(),
	})
	_ = s.writeJSON(client, map[string]any{"v": 1, "type": "echo", "content": json.RawMessage(payload)})
}

func (s *webSocketServer) storeConversationTurn(ctx context.Context, client *Client, turn *ConversationTurn) {
	s.log.InfoWithID(ctx, "[WebSocketServer: storeConversationTurn] Storing turn", map[string]interface{}{
		"session_id": turn.SessionID,
		"segment_id": turn.SegmentID,
		"turn_type":  turn.TurnType,
		"started_at": turn.StartedAt,
	})
}

func (s *webSocketServer) updateConversationTurnEndTime(ctx context.Context, segmentID string, endTime time.Time) {
	s.log.InfoWithID(ctx, "[WebSocketServer: updateConversationTurnEndTime] Updating turn end time", map[string]interface{}{
		"segment_id": segmentID,
		"end_time":   endTime,
	})
}

func (s *webSocketServer) pingLoop(c *Client) {
	s.log.InfoWithID(context.Background(), "[WebSocketServer: pingLoop] Starting ping loop")

	t := time.NewTicker(constants.WebSocketPingInterval)
	defer t.Stop()

	for range t.C {
		c.mu.Lock()
		timeSinceLastPong := time.Since(c.lastPongTime)
		c.mu.Unlock()

		if timeSinceLastPong > constants.WebSocketPongTimeout {
			s.log.ErrorWithID(context.Background(), "[WebSocketServer: pingLoop] Pong timeout - no pong received", map[string]interface{}{
				"session_id":           c.sessionID,
				"time_since_last_pong": timeSinceLastPong,
				"timeout":              constants.WebSocketPongTimeout,
			})
			s.disconnect(c)
			return
		}

		c.mu.Lock()
		err := c.conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(constants.WebSocketPingDuration))
		c.mu.Unlock()

		if err != nil {
			s.log.ErrorWithID(context.Background(), "[WebSocketServer: pingLoop] Error writing ping message", err)
			s.disconnect(c)
			return
		}

		s.log.DebugWithID(context.Background(), "[WebSocketServer: pingLoop] Ping sent", map[string]interface{}{
			"session_id":           c.sessionID,
			"time_since_last_pong": timeSinceLastPong,
		})
	}
}

func (s *webSocketServer) disconnect(c *Client) {
	s.log.InfoWithID(context.Background(), "[WebSocketServer: disconnect] Called")
	s.mu.Lock()
	delete(s.sessions, c.sessionID)

	if set := s.userSessions[c.userID]; set != nil {
		delete(set, c.sessionID)
		if len(set) == 0 {
			delete(s.userSessions, c.userID)
		}
	}
	s.mu.Unlock()

	close(c.pongReceived)
	_ = c.conn.Close()
}

func (s *webSocketServer) writeJSON(c *Client, v any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn.WriteJSON(v)
}
