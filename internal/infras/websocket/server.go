package websocket

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"gitlab.com/interview-simulation/interview-backend-server/internal/config"
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
	aiAgentConnected        bool
	clientManager           *ClientManager
	cfg                     *config.Config
}

func NewWebSocketServer(
	log *log.Logger,
	redisClient database.RedisClient,
	authContext middleware.AuthContext,
	interviewSessionService services.InterviewSessionService,
	interviewSessionRepo repositories.InterviewSessionRepository,
	clientManager *ClientManager,
	cfg *config.Config,
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
		aiAgentConnected:        false,
		clientManager:           clientManager,
		cfg:                     cfg,
	}
}

func (s *webSocketServer) Start(ctx context.Context) error {
	s.log.InfoWithID(ctx, "[WebSocketServer: Start] Starting WebSocket server")

	s.aiAgentConnected = true
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
		return nil
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
		s.log.InfoWithID(ctx, "[WebSocketServer] Pong received from client")
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

	u, err := url.Parse(s.cfg.InterviewSessionConfig.WebSocketURL)
	if err != nil {
		s.log.ErrorWithID(ctx, "[WebSocketServer: HandleConnection] Invalid agent WS URL", err)
		s.disconnect(client)
		return nil
	}

	q := u.Query()
	q.Set("session_id", client.sessionID)
	q.Set("user_id", client.userID)
	u.RawQuery = q.Encode()

	agentClient := NewWebSocketClient(s.log)
	callbacks := NewWebSocketCallbacks().
		WithASRPartial(func(segmentID, text string, seq int, stability float64) {
			s.log.InfoWithID(ctx, "[AI-ASRPartial]", map[string]any{"text": text})
		}).
		WithASRFinal(func(segmentID, text string, seq int) {
			s.log.InfoWithID(ctx, "[AI-ASRFinal]", map[string]any{"text": text})
		})
	agentClient.SetCallbacks(*callbacks)
	if err := agentClient.Start(ctx, u.String()); err != nil {
		s.log.ErrorWithID(ctx, "[WebSocketServer: HandleConnection] Error starting agent client: ", err)
		s.disconnect(client)
		return nil
	}

	_ = agentClient.SendSessionInfo(ctx, client.sessionID, client.userID)
	s.clientManager.Set(client.sessionID, agentClient)

	interviewReq := &entities.UpdateInterviewSessionStatusReq{
		SessionID: client.sessionID,
		Status:    constants.StatusOnGoing,
	}

	if err := s.interviewSessionService.UpdateInterviewSessionStatus(ctx, interviewReq); err != nil {
		s.log.ErrorWithID(ctx, "[WebSocketServer: HandleConnection] Error starting interview session", err)

		s.disconnect(client)
		return nil
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
				Type:    "error",
				Code:    string(app_error.ErrCodeWebSocketInvalidMessage),
				Message: app_error.ErrCodeWebSocketInvalidMessage.Message(),
			})
			s.disconnect(c)
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
			s.handleAudioBinaryMessage(ctx, c, payload)

		case websocket.PingMessage:
			s.log.InfoWithID(ctx, "[WebSocketServer] PingMessage received — replying with Pong")
			// You must reply with Pong manually here
			_ = c.conn.WriteMessage(websocket.PongMessage, nil)

		case websocket.PongMessage:
			s.log.InfoWithID(ctx, "[WebSocketServer] PongMessage received (manual read path)")
			c.lastPongTime = time.Now()

		default:
			// ignore other frame types, like pong
		}
	}
}

func (s *webSocketServer) handleAudioBinaryMessage(ctx context.Context, client *Client, payload []byte) {
	s.log.InfoWithID(ctx, "[WebSocketServer: handleAudioBinaryMessage] Called")

	if len(payload) < 4 {
		s.log.ErrorWithID(ctx, "Invalid frame: too short")
		s.disconnect(client)
		return
	}

	headerLength := binary.BigEndian.Uint32(payload[:4])

	if int(headerLength)+4 > len(payload) {
		s.log.ErrorWithID(ctx, "Invalid frame: header length too large")
		s.disconnect(client)
		return
	}

	headerBytes := payload[4 : 4+headerLength]

	var header msgAudioChunk
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		s.log.ErrorWithID(ctx, "Invalid header JSON", err)
		s.disconnect(client)
		return
	}

	if header.SessionID != client.sessionID {
		s.log.ErrorWithID(ctx, "Security violation: Session ID mismatch", map[string]any{
			"expected_session_id": client.sessionID,
			"received_session_id": header.SessionID,
			"user_id":             client.userID,
		})
		_ = s.writeJSON(client, msgError{
			Type:    "error",
			Code:    string(app_error.ErrCodeWebSocketInvalidMessage),
			Message: "Session ID mismatch",
		})
		s.disconnect(client)
		return
	}

	if header.SegmentID != client.currentSegmentID {
		s.log.ErrorWithID(ctx, "Security violation: Segment ID mismatch", map[string]any{
			"expected_segment_id": client.currentSegmentID,
			"received_segment_id": header.SegmentID,
			"session_id":          client.sessionID,
			"user_id":             client.userID,
		})
		_ = s.writeJSON(client, msgError{
			Type:    "error",
			Code:    string(app_error.ErrCodeWebSocketInvalidMessage),
			Message: "Segment ID mismatch",
		})
		s.disconnect(client)
		return
	}

	audioData := payload[4+headerLength:]

	if agentClient, exists := s.clientManager.Get(client.sessionID); exists {
		if err := agentClient.SendAudio(ctx, header.SegmentID, audioData); err != nil {
			s.log.ErrorWithID(ctx, "[WebSocketServer: handleAudioBinaryMessage] Error forwarding audio to AI agent", err)
		}
	}
}

func (s *webSocketServer) sendMessageTypeSegmentStart(ctx context.Context, client *Client, payload []byte) {
	s.log.InfoWithID(ctx, "[WebSocketServer: sendMessageTypeSegmentStart] Called")

	var m msgSegmentStart
	if json.Unmarshal(payload, &m) != nil || m.SegmentID == "" {
		_ = s.writeJSON(client, msgError{
			Type:    "error",
			Code:    string(app_error.ErrCodeWebSocketInvalidSegmentStart),
			Message: app_error.ErrCodeWebSocketInvalidSegmentStart.Message(),
		})
		s.disconnect(client)
		return
	}

	if client.sessionID != m.SessionID {
		s.log.ErrorWithID(ctx, "Security violation: Session ID mismatch", map[string]any{
			"expected_session_id": client.sessionID,
			"received_session_id": m.SessionID,
			"user_id":             client.userID,
		})
		s.disconnect(client)
		return
	}

	client.currentSegmentID = m.SegmentID

	if agentClient, exists := s.clientManager.Get(client.sessionID); exists {
		if err := agentClient.SegmentStart(ctx, m.SegmentID, m.SampleRate, m.Encoding, m.Channels); err != nil {
			s.log.ErrorWithID(ctx, "[WebSocketServer: sendMessageTypeSegmentStart] Error forwarding segment start to AI agent", err)
		}
	}

}

func (s *webSocketServer) sendMessageTypeSegmentEnd(ctx context.Context, client *Client, payload []byte) {
	s.log.InfoWithID(ctx, "[WebSocketServer: sendMessageTypeSegmentEnd] Called")

	var m msgSegmentEnd
	if json.Unmarshal(payload, &m) != nil || m.SegmentID == "" {
		_ = s.writeJSON(client, msgError{
			Type:    "error",
			Code:    string(app_error.ErrCodeWebSocketInvalidSegmentEnd),
			Message: app_error.ErrCodeWebSocketInvalidSegmentEnd.Message(),
		})
		return
	}

	if client.sessionID != m.SessionID {
		s.log.ErrorWithID(ctx, "Security violation: Session ID mismatch", map[string]any{
			"expected_session_id": client.sessionID,
			"received_session_id": m.SessionID,
			"user_id":             client.userID,
		})
		return
	}

	if client.currentSegmentID != m.SegmentID {
		s.log.ErrorWithID(ctx, "Security violation: Segment ID mismatch", map[string]any{
			"expected_segment_id": client.currentSegmentID,
			"received_segment_id": m.SegmentID,
			"session_id":          client.sessionID,
			"user_id":             client.userID,
		})
		return
	}

	client.currentSegmentID = ""

	if agentClient, exists := s.clientManager.Get(client.sessionID); exists {
		if err := agentClient.SegmentEnd(ctx, m.SegmentID); err != nil {
			s.log.ErrorWithID(ctx, "[WebSocketServer: sendMessageTypeSegmentEnd] Error forwarding segment end to AI agent", err)
		}
	}
}

func (s *webSocketServer) sendMessageTypeError(ctx context.Context, client *Client, payload []byte) {
	s.log.InfoWithID(ctx, "[WebSocketServer: sendMessageTypeError] Called")

	_ = s.writeJSON(client, msgError{
		Type:    "error",
		Code:    string(app_error.ErrCodeWebSocketInvalidMessage),
		Message: app_error.ErrCodeWebSocketInvalidMessage.Message(),
	})
	_ = s.writeJSON(client, map[string]any{"v": 1, "type": "echo", "content": json.RawMessage(payload)})
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

func (s *webSocketServer) disconnect(client *Client) {
	s.log.InfoWithID(context.Background(), "[WebSocketServer: disconnect] Called")
	s.mu.Lock()
	delete(s.sessions, client.sessionID)

	if set := s.userSessions[client.userID]; set != nil {
		delete(set, client.sessionID)
		if len(set) == 0 {
			delete(s.userSessions, client.userID)
		}
	}

	_ = s.writeJSON(client, map[string]any{
		"v":       1,
		"type":    "error",
		"code":    string(app_error.ErrCodeWebSocketInvalidMessage),
		"message": app_error.ErrCodeWebSocketInvalidMessage.Message(),
	})

	s.clientManager.Delete(client.sessionID)
	s.mu.Unlock()
	_ = client.conn.Close()
	close(client.pongReceived)
}

func (s *webSocketServer) writeJSON(c *Client, v any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn.WriteJSON(v)
}

func (s *webSocketServer) HandleTTSResponse(ctx context.Context, sessionID, segmentID, ttsID, encoding string) error {
	s.mu.RLock()
	client, exists := s.sessions[sessionID]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("client not found for session %s", sessionID)
	}

	_ = s.writeJSON(client, map[string]any{
		"v": 1, "type": "tts_start", "segment_id": segmentID, "tts_id": ttsID, "encoding": encoding,
	})

	return nil
}

func (s *webSocketServer) HandleTTSChunk(ctx context.Context, sessionID, segmentID string, audioData []byte) error {
	s.mu.RLock()
	client, exists := s.sessions[sessionID]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("client not found for session %s", sessionID)
	}

	header := map[string]string{
		"type":       "tts_chunk",
		"session_id": sessionID,
		"segment_id": segmentID,
	}

	headerBytes, err := json.Marshal(header)
	if err != nil {
		return err
	}

	frame := make([]byte, 4+len(headerBytes)+len(audioData))
	binary.BigEndian.PutUint32(frame[:4], uint32(len(headerBytes)))
	copy(frame[4:], headerBytes)
	copy(frame[4+len(headerBytes):], audioData)

	client.mu.Lock()
	defer client.mu.Unlock()
	return client.conn.WriteMessage(websocket.BinaryMessage, frame)
}

func (s *webSocketServer) HandleTTSEnd(ctx context.Context, sessionID, segmentID, ttsID string) error {
	s.mu.RLock()
	client, exists := s.sessions[sessionID]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("client not found for session %s", sessionID)
	}

	_ = s.writeJSON(client, map[string]any{
		"v": 1, "type": "tts_end", "segment_id": segmentID, "tts_id": ttsID,
	})

	return nil
}
