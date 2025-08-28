package websocket

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
)

type WebSocketCallbacks struct {
	OnConnectionEstablished func(sessionID string)
	OnDisconnect            func(sessionID string)
}

type WebSocketClient interface {
	Start(ctx context.Context, url string) error
	Close(ctx context.Context) error

	SegmentStart(ctx context.Context, segmentID string, sampleRate int, encoding string, channels int) error
	SendAudio(ctx context.Context, segmentID string, buf []byte) error
	SegmentEnd(ctx context.Context, segmentID string) error
	StopTTS(ctx context.Context, segmentID string) error

	SendMessage(ctx context.Context, msgType string, data map[string]interface{}) error
	SendSessionInfo(ctx context.Context, sessionID, userID string) error
	IsConnected() bool
	SetCallbacks(callbacks WebSocketCallbacks)
}

type webSocketClient struct {
	cb        WebSocketCallbacks
	log       *log.Logger
	sessionID string
	userID    string

	conn         *websocket.Conn
	connected    bool
	mu           sync.Mutex
	lastPongTime time.Time
	pongReceived chan struct{}
}

func NewWebSocketClient(log *log.Logger) WebSocketClient {
	return &webSocketClient{
		cb:           WebSocketCallbacks{},
		log:          log,
		conn:         nil,
		connected:    false,
		pongReceived: make(chan struct{}, 1),
	}
}

func (c *webSocketClient) Start(ctx context.Context, url string) error {
	c.log.InfoWithID(ctx, "[WebSocketClient: Start] Called:", url)

	dialer := websocket.Dialer{
		Proxy:             http.ProxyFromEnvironment,
		HandshakeTimeout:  constants.WebSocketClientHandshakeTimeout,
		EnableCompression: true,
	}

	conn, _, err := dialer.DialContext(ctx, url, nil)
	if err != nil {
		c.log.ErrorWithID(ctx, "[WebSocketClient: Start] Error dialing:", err)
		return err
	}
	c.conn = conn
	c.connected = true
	c.lastPongTime = time.Now()

	_ = c.conn.SetReadDeadline(time.Now().Add(constants.WebSocketReadTimeout))
	c.conn.SetPongHandler(func(string) error {
		c.log.InfoWithID(ctx, "[WebSocketClient] Pong received from server")
		c.mu.Lock()
		c.lastPongTime = time.Now()
		c.mu.Unlock()

		select {
		case c.pongReceived <- struct{}{}:
		default:
		}

		return c.conn.SetReadDeadline(time.Now().Add(constants.WebSocketReadTimeout))
	})

	go c.readLoop(ctx)
	go c.pingLoop(ctx)
	return nil
}

func (c *webSocketClient) Close(ctx context.Context) error {
	c.log.InfoWithID(ctx, "[WebSocketClient: Close] Called")

	c.connected = false
	if c.conn != nil {
		c.log.InfoWithID(ctx, "[WebSocketClient: Close] Closing connection")
		return c.conn.Close()
	}
	return nil
}

func (c *webSocketClient) SegmentStart(ctx context.Context, seg string, sr int, enc string, ch int) error {
	return nil
}

func (c *webSocketClient) SendAudio(ctx context.Context, seg string, buf []byte) error {
	return nil
}

func (c *webSocketClient) SegmentEnd(ctx context.Context, seg string) error {
	return nil
}

func (c *webSocketClient) StopTTS(ctx context.Context, seg string) error {
	return nil
}

func (c *webSocketClient) SendMessage(ctx context.Context, msgType string, data map[string]interface{}) error {
	if !c.connected || c.conn == nil {
		c.log.ErrorWithID(ctx, "[WebSocketClient: SendMessage] Not connected")
		return websocket.ErrCloseSent
	}

	message := map[string]interface{}{
		"type": msgType,
	}

	for k, v := range data {
		message[k] = v
	}

	return c.conn.WriteJSON(message)
}

func (c *webSocketClient) SendSessionInfo(ctx context.Context, sessionID, userID string) error {
	c.log.InfoWithID(ctx, "[WebSocketClient: SendSessionInfo] Called:", sessionID, userID)
	c.sessionID = sessionID
	c.userID = userID

	return c.SendMessage(ctx, "session_info", map[string]interface{}{
		"session_id": sessionID,
		"user_id":    userID,
		"timestamp":  time.Now().Unix(),
	})
}

func (c *webSocketClient) IsConnected() bool {
	ctx := context.Background()
	c.log.InfoWithID(ctx, "[WebSocketClient: IsConnected] Called:", c.connected, c.conn != nil)
	return c.connected && c.conn != nil
}

func (c *webSocketClient) SetCallbacks(callbacks WebSocketCallbacks) {
	ctx := context.Background()
	c.log.InfoWithID(ctx, "[WebSocketClient: SetCallbacks] Called:", callbacks)
	c.cb = callbacks
}

func (c *webSocketClient) readLoop(ctx context.Context) {
	c.log.InfoWithID(ctx, "[WebSocketClient: readLoop] Called")

	for {
		mt, data, err := c.conn.ReadMessage()
		if err != nil {
			c.log.ErrorWithID(ctx, "[WebSocketClient: readLoop] Connection error, disconnecting", err)
			c.connected = false

			if c.cb.OnDisconnect != nil {
				c.cb.OnDisconnect(c.sessionID)
			}

			c.disconnect(ctx)
			return
		}

		switch mt {
		case websocket.TextMessage:
			var base struct {
				Type      string `json:"type"`
				SegmentID string `json:"segment_id"`
				SessionID string `json:"session_id"`
			}
			if json.Unmarshal(data, &base) != nil {
				continue
			}

			switch base.Type {
			case "connection_established":
				var x struct {
					SessionID string `json:"session_id"`
				}
				if json.Unmarshal(data, &x) == nil && c.cb.OnConnectionEstablished != nil {
					c.cb.OnConnectionEstablished(x.SessionID)
				}
			}

		case websocket.BinaryMessage:

		}
	}
}

func (c *webSocketClient) pingLoop(ctx context.Context) {
	c.log.InfoWithID(ctx, "[WebSocketClient: pingLoop] Starting ping loop")

	t := time.NewTicker(constants.WebSocketPingInterval)
	defer t.Stop()

	for range t.C {
		c.mu.Lock()
		timeSinceLastPong := time.Since(c.lastPongTime)
		c.mu.Unlock()

		if timeSinceLastPong > constants.WebSocketPongTimeout {
			c.log.ErrorWithID(ctx, "[WebSocketClient: pingLoop] Pong timeout - no pong received", map[string]interface{}{
				"session_id":           c.sessionID,
				"time_since_last_pong": timeSinceLastPong,
				"timeout":              constants.WebSocketPongTimeout,
			})
			c.connected = false
			if c.cb.OnDisconnect != nil {
				c.cb.OnDisconnect(c.sessionID)
			}
			c.disconnect(ctx)
			return
		}

		c.mu.Lock()
		err := c.conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(constants.WebSocketPingDuration))
		c.mu.Unlock()
		if err != nil {
			c.log.ErrorWithID(ctx, "[WebSocketClient: pingLoop] Error writing ping message", err)
			c.connected = false
			if c.cb.OnDisconnect != nil {
				c.cb.OnDisconnect(c.sessionID)
			}
			c.disconnect(ctx)
			return
		}
	}
}

func (c *webSocketClient) disconnect(ctx context.Context) {
	c.log.InfoWithID(ctx, "[WebSocketClient: disconnect] Disconnecting client")

	if err := c.conn.Close(); err != nil {
		c.log.ErrorWithID(ctx, "[WebSocketClient: disconnect] Error closing connection", err)
	} else {
		c.log.InfoWithID(ctx, "[WebSocketClient: disconnect] Connection closed successfully")
	}

	c.connected = false
	if c.cb.OnDisconnect != nil {
		c.cb.OnDisconnect(c.sessionID)
	}
}
