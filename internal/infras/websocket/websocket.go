package websocket

import (
	"context"
	"encoding/json"
	"errors"

	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"gitlab.com/interview-simulation/interview-backend-server/internal/entities"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
)

type Client struct {
	conn      *websocket.Conn
	mu        sync.Mutex
	userID    string
	sessionID string
}

type WebSocketServer struct {
	log          *log.Logger
	sessions     map[string]*Client
	userSessions map[string]map[string]bool
	mu           sync.RWMutex
	upgrader     websocket.Upgrader
}

func NewWebSocketServer(log *log.Logger) *WebSocketServer {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	return &WebSocketServer{
		log:          log,
		upgrader:     upgrader,
		sessions:     make(map[string]*Client),
		userSessions: make(map[string]map[string]bool),
	}
}

func (s *WebSocketServer) HandleConnection(w http.ResponseWriter, r *http.Request, redisSessionToken entities.RedisSessionToken) {
	s.log.InfoWithID(r.Context(), "[WebSocketServer: HandleConnection] Called")
	ctx := r.Context()

	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.log.ErrorWithID(ctx, "[WebSocketServer: HandleConnection] Error upgrading connection", err)
		return
	}

	sessionID := uuid.NewString()
	c := &Client{
		conn:      conn,
		userID:    redisSessionToken.UserID,
		sessionID: redisSessionToken.SessionID,
	}

	s.mu.Lock()
	s.sessions[sessionID] = c
	if s.userSessions[redisSessionToken.UserID] == nil {
		s.userSessions[redisSessionToken.UserID] = map[string]bool{}
	}
	s.userSessions[redisSessionToken.UserID][sessionID] = true
	s.mu.Unlock()

	successConnection := map[string]any{
		"type": "connection_established",
	}

	_ = s.writeJSON(c, successConnection)

	go s.readLoop(ctx, c)
}

func (s *WebSocketServer) readLoop(ctx context.Context, c *Client) {
	s.log.InfoWithID(ctx, "[WebSocketServer: readLoop] Called")
	defer s.disconnect(c)
	c.conn.SetReadLimit(1 << 20)

	for {
		_, payload, err := c.conn.ReadMessage()
		if err != nil {
			return
		}

		var msg struct {
			Type         string      `json:"type"`
			SessionToken string      `json:"session_token,omitempty"`
			Content      interface{} `json:"content"`
		}

		if json.Unmarshal(payload, &msg) != nil {
			continue
		}

		switch msg.Type {
		case "send":
			s.SendToSession(c.sessionID, msg.SessionToken, msg.Content)
		default:
			_ = s.writeJSON(c, map[string]any{"type": "echo", "content": msg.Content})
		}
	}
}

func (s *WebSocketServer) SendToSession(fromSession, toSession string, content any) error {
	s.log.InfoWithID(context.Background(), "[WebSocketServer: SendToSession] Called")
	s.mu.RLock()
	rcpt := s.sessions[toSession]
	s.mu.RUnlock()
	if rcpt == nil {
		return errors.New("receiver session not connected")
	}
	return s.writeJSON(rcpt, map[string]any{
		"type": "message", "from_session": fromSession, "to_session": toSession,
		"content": content, "ts": time.Now(),
	})
}

func (s *WebSocketServer) disconnect(c *Client) {
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

func (s *WebSocketServer) writeJSON(c *Client, v any) error {
	s.log.InfoWithID(context.Background(), "[WebSocketServer: writeJSON] Called")
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn.WriteJSON(v)
}
