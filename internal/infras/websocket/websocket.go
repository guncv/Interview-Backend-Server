package websocket

import (
	"context"
	"encoding/json"
	"net"
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

type msgBase struct {
	V    int    `json:"v"`
	Type string `json:"type"`
}

type msgHello struct {
	msgBase
	SessionID string `json:"session_id"`
}

type msgSegmentStart struct {
	msgBase
	SegmentID  string `json:"segment_id"`
	SampleRate int    `json:"sample_rate"`
	Encoding   string `json:"encoding"`
	Channels   int    `json:"channels"`
	StartedAt  int64  `json:"started_at_ms,omitempty"`
}

type msgSegmentEnd struct {
	msgBase
	SegmentID string `json:"segment_id"`
	EndedAt   int64  `json:"ended_at_ms,omitempty"`
}

type msgStopTTS struct {
	msgBase
	SegmentID string `json:"segment_id"`
}

// type msgASR struct {
// 	msgBase
// 	SegmentID string  `json:"segment_id"`
// 	Text      string  `json:"text"`
// 	IsFinal   bool    `json:"is_final"`
// 	Seq       int     `json:"seq,omitempty"`
// 	Stability float64 `json:"stability,omitempty"`
// }

// type msgTTSStart struct {
// 	msgBase
// 	SegmentID string `json:"segment_id"`
// 	TTSID     string `json:"tts_id"`
// 	Encoding  string `json:"encoding"` // "OPUS_OGG" | "OPUS_WEBM" | "PCM16"
// }

// type msgTTSEnd struct {
// 	msgBase
// 	SegmentID string `json:"segment_id"`
// 	TTSID     string `json:"tts_id"`
// }

// type msgDBAck struct {
// 	msgBase
// 	TurnID string   `json:"turn_id"`
// 	Saved  []string `json:"saved"`
// }

type msgError struct {
	msgBase
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Client struct {
	conn             *websocket.Conn
	mu               sync.Mutex
	userID           string
	sessionID        string
	currentSegmentID string
}

type WebSocketServerInterface interface {
	HandleConnection(ctx context.Context, w http.ResponseWriter, r *http.Request, sessionToken entities.OpenWsConnectionRequest) error
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
}

func NewWebSocketServer(
	log *log.Logger,
	redisClient database.RedisClient,
	authContext middleware.AuthContext,
	interviewSessionService services.InterviewSessionService,
	interviewSessionRepo repositories.InterviewSessionRepository,
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
	}
}

func isTimeoutErr(err error) bool {
	if ne, ok := err.(net.Error); ok && ne.Timeout() {
		return true
	}
	if ce, ok := err.(*websocket.CloseError); ok {
		return ce.Code == websocket.CloseAbnormalClosure
	}
	return false
}

func isNormalClose(err error) bool {
	if ce, ok := err.(*websocket.CloseError); ok {
		return ce.Code == websocket.CloseNormalClosure || ce.Code == websocket.CloseGoingAway
	}
	return false
}

func (s *webSocketServer) HandleConnection(ctx context.Context, w http.ResponseWriter, r *http.Request, sessionToken entities.OpenWsConnectionRequest) error {
	s.log.InfoWithID(ctx, "[WebSocketServer: HandleConnection] Called")

	authCtx, err := s.authContext.GetAuthContext(ctx)
	if err != nil {
		s.log.ErrorWithID(ctx, "[WebSocketServer: HandleConnection] Error getting auth context", err)
		return app_error.New(constants.ErrInvalidToken, app_error.ErrCodeSessionInvalidToken)
	}

	redisSessionToken, err := s.redisClient.Get(ctx, sessionToken.SessionToken)
	if err != nil {
		s.log.ErrorWithID(ctx, "[WebSocketServer: HandleConnection] Error getting redis session token", err)
		return err
	}

	var sessionPayload entities.RedisSessionToken
	if err := json.Unmarshal([]byte(redisSessionToken), &sessionPayload); err != nil {
		s.log.ErrorWithID(ctx, "[WebSocketServer: HandleConnection] Error unmarshalling redis session token", err)
		return err
	}

	if sessionPayload.UserID != authCtx.Payload.UserID {
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
		conn:      conn,
		userID:    sessionPayload.UserID,
		sessionID: sessionPayload.SessionID,
	}

	_ = client.conn.SetReadDeadline(time.Now().Add(constants.WebSocketReadTimeout))
	client.conn.SetPongHandler(func(string) error {
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

	_ = s.writeJSON(client, map[string]any{
		"v": 1, "type": "connection_established", "session_id": client.sessionID,
	})

	go s.pingLoop(client, constants.WebSocketPingInterval)
	go s.readLoop(ctx, client)

	return nil
}

func (s *webSocketServer) readLoop(ctx context.Context, c *Client) {
	s.log.InfoWithID(ctx, "[WebSocketServer: readLoop] Called")
	defer s.disconnect(c)
	c.conn.SetReadLimit(1 << 20)

	for {
		mt, payload, err := c.conn.ReadMessage()
		if err != nil {
			s.log.ErrorWithID(ctx, "[WebSocketServer: readLoop] Error reading message", err)
			_ = s.writeJSON(c, msgError{msgBase{1, "error"}, string(app_error.ErrCodeWebSocketInvalidMessage), app_error.ErrCodeWebSocketInvalidMessage.Message()})
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
				var m msgHello
				if json.Unmarshal(payload, &m) != nil || m.SessionID == "" {
					_ = s.writeJSON(c, msgError{msgBase{1, "error"}, string(app_error.ErrCodeWebSocketInvalidHello), app_error.ErrCodeWebSocketInvalidHello.Message()})
					continue
				}

				c.sessionID = m.SessionID
				_ = s.writeJSON(c, map[string]any{"v": 1, "type": "hello", "session_id": c.sessionID})

			case constants.WebSocketMessageTypeSegmentStart:
				var m msgSegmentStart
				if json.Unmarshal(payload, &m) != nil || m.SegmentID == "" {
					_ = s.writeJSON(c, msgError{msgBase{1, "error"}, string(app_error.ErrCodeWebSocketInvalidSegmentStart), app_error.ErrCodeWebSocketInvalidSegmentStart.Message()})
					continue
				}
				c.currentSegmentID = m.SegmentID

			case constants.WebSocketMessageTypeSegmentEnd:
				var m msgSegmentEnd
				if json.Unmarshal(payload, &m) != nil || m.SegmentID == "" {
					_ = s.writeJSON(c, msgError{msgBase{1, "error"}, string(app_error.ErrCodeWebSocketInvalidSegmentEnd), app_error.ErrCodeWebSocketInvalidSegmentEnd.Message()})
					continue
				}
				if c.currentSegmentID != m.SegmentID {
					_ = s.writeJSON(c, msgError{msgBase{1, "error"}, string(app_error.ErrCodeWebSocketInvalidSegmentEnd), app_error.ErrCodeWebSocketInvalidSegmentEnd.Message()})
					continue
				}

				c.currentSegmentID = ""

			case constants.WebSocketMessageTypeStopTTS:
				var m msgStopTTS
				if json.Unmarshal(payload, &m) != nil || m.SegmentID == "" {
					_ = s.writeJSON(c, msgError{msgBase{1, "error"}, string(app_error.ErrCodeWebSocketInvalidStopTTS), app_error.ErrCodeWebSocketInvalidStopTTS.Message()})
					continue
				}

				c.currentSegmentID = ""

			default:
				_ = s.writeJSON(c, msgError{msgBase{1, "error"}, string(app_error.ErrCodeWebSocketInvalidMessage), app_error.ErrCodeWebSocketInvalidMessage.Message()})
				_ = s.writeJSON(c, map[string]any{"v": 1, "type": "echo", "content": json.RawMessage(payload)})
			}

		case websocket.BinaryMessage:
			if c.currentSegmentID == "" {
				_ = s.writeJSON(c, msgError{msgBase{1, "error"}, "NO_ACTIVE_SEGMENT", "binary audio with no active segment"})
				continue
			}
			// TODO: write to recorder/S3 multipart and/or forward to AI agent
			// Example: s.audioForwarder.Forward(c.sessionID, c.currentSegmentID, payload)
		default:
			// ignore other frame types
		}
	}
}

func (s *webSocketServer) pingLoop(c *Client, every time.Duration) {
	t := time.NewTicker(every)
	defer t.Stop()

	for range t.C {
		c.mu.Lock()
		err := c.conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(5*time.Second))
		c.mu.Unlock()

		if err != nil {
			s.log.ErrorWithID(context.Background(), "[WebSocketServer: pingLoop] Error writing ping message", err)
			_ = c.conn.Close()
			return
		}
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
	_ = c.conn.Close()
}

func (s *webSocketServer) writeJSON(c *Client, v any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn.WriteJSON(v)
}

// func (s *webSocketServer) writeBinary(c *Client, b []byte) error {
// 	c.mu.Lock()
// 	defer c.mu.Unlock()
// 	return c.conn.WriteMessage(websocket.BinaryMessage, b)
// }

// // Example helper you might call when AI agent returns text:
// func (s *webSocketServer) sendASRPartial(c *Client, segmentID, text string, seq int, stability float64) error {
// 	return s.writeJSON(c, msgASR{
// 		msgBase:   msgBase{1, "asr"},
// 		SegmentID: segmentID,
// 		Text:      text,
// 		IsFinal:   false,
// 		Seq:       seq,
// 		Stability: stability,
// 	})
// }

// func (s *webSocketServer) sendASRFinal(c *Client, segmentID, text string, seq int) error {
// 	return s.writeJSON(c, msgASR{
// 		msgBase:   msgBase{1, "asr"},
// 		SegmentID: segmentID,
// 		Text:      text,
// 		IsFinal:   true,
// 		Seq:       seq,
// 	})
// }

// func (s *webSocketServer) sendTTSEvents(c *Client, segmentID, ttsID string, start bool) error {
// 	if start {
// 		return s.writeJSON(c, msgTTSStart{msgBase{1, "tts_start"}, segmentID, ttsID, "OPUS_OGG"})
// 	}
// 	return s.writeJSON(c, msgTTSEnd{msgBase{1, "tts_end"}, segmentID, ttsID})
// }

// // Example error send
// func (s *webSocketServer) sendError(c *Client, code, message string) error {
// 	return s.writeJSON(c, msgError{msgBase{1, "error"}, code, message})
// }

// // Example guard
// func require(b bool, code, msg string) error {
// 	if !b {
// 		return errors.New(code + ": " + msg)
// 	}
// 	return nil
// }
