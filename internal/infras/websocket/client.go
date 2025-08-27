package websocket

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
)

type WebSocketCallbacks struct {
	OnASRPartial            func(segmentID, text string, seq int, stability float64)
	OnASRFinal              func(segmentID, text string, seq int)
	OnTTSStart              func(segmentID, ttsID, encoding string)
	OnTTSChunk              func(segmentID string, data []byte)
	OnTTSEnd                func(segmentID, ttsID string)
	OnError                 func(code, msg string)
	OnEvaluation            func(segmentID string, score float64, comment string)
	OnConnectionEstablished func(sessionID string)
	OnPing                  func(timestamp float64)
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

	conn      *websocket.Conn
	connected bool
}

func NewWebSocketClient(log *log.Logger) WebSocketClient {
	return &webSocketClient{
		cb:        WebSocketCallbacks{},
		log:       log,
		conn:      nil,
		connected: false,
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

	_ = c.conn.SetReadDeadline(time.Now().Add(constants.WebSocketReadTimeout))

	go c.readLoop()
	go c.pingLoop()
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

func (c *webSocketClient) readLoop() {
	ctx := context.Background()
	c.log.InfoWithID(ctx, "[WebSocketClient: readLoop] Called")

	for {
		mt, data, err := c.conn.ReadMessage()
		if err != nil {
			c.connected = false
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

			case "error":
				var x struct{ Code, Message string }
				if json.Unmarshal(data, &x) == nil && c.cb.OnError != nil {
					c.cb.OnError(x.Code, x.Message)
				}
			}

		case websocket.BinaryMessage:

		}
	}
}

func (c *webSocketClient) pingLoop() {
	ctx := context.Background()
	c.log.InfoWithID(ctx, "[WebSocketClient: pingLoop] Called")

	t := time.NewTicker(constants.WebSocketPingInterval)
	defer t.Stop()
	for range t.C {
		if c.conn != nil {
			_ = c.conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(5*time.Second))
		}
	}
}
