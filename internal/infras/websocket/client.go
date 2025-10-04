package websocket

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
)

type WebSocketClient interface {
	Start(ctx context.Context, url string) error
	Close(ctx context.Context) error

	StartSessionConversation(ctx context.Context, msg MsgInterviewTypeAndSessionID) error
	SegmentStart(ctx context.Context, msg MsgSegmentStart) error
	SendUserAudio(ctx context.Context, msg MsgUserAudioChunk, audioData []byte) error
	SegmentEnd(ctx context.Context, msg MsgSegmentEnd) error

	SendMessage(ctx context.Context, data map[string]interface{}) error
	SendBinaryMessage(ctx context.Context, data []byte) error
	SendSessionInfo(ctx context.Context, sessionID, userID, resumeID string) error
	IsConnected() bool
	SetCallbacks(callbacks WebSocketClientCallbacks)
}

type webSocketClient struct {
	cb               WebSocketClientCallbacks
	log              *log.Logger
	sessionID        string
	userID           string
	currentSegmentID string
	resumeID         string

	conn         *websocket.Conn
	connected    bool
	lastPongTime time.Time
	pongReceived chan struct{}
	cancelFunc   context.CancelFunc
	mu           sync.RWMutex
	disconnected bool
}

func NewWebSocketClient(log *log.Logger) WebSocketClient {
	cb := NewWebSocketClientCallbacks(nil, nil, nil, log)

	return &webSocketClient{
		cb:           cb,
		log:          log,
		conn:         nil,
		connected:    false,
		pongReceived: make(chan struct{}, 1),
	}
}

func (c *webSocketClient) Start(ctx context.Context, url string) error {
	c.log.InfoWithID(ctx, "[WebSocketClient: Start] Starting AI agent client connection", map[string]any{
		"url": url,
	})

	dialer := websocket.Dialer{
		Proxy:             http.ProxyFromEnvironment,
		HandshakeTimeout:  constants.WebSocketClientHandshakeTimeout,
		EnableCompression: true,
	}

	conn, _, err := dialer.DialContext(ctx, url, nil)
	if err != nil {
		c.log.ErrorWithID(ctx, "[WebSocketClient: Start] Error connecting to AI agent", map[string]any{
			"url":   url,
			"error": err,
		})
		return err
	}
	c.conn = conn
	c.connected = true
	c.lastPongTime = time.Now()

	c.log.InfoWithID(ctx, "[WebSocketClient: Start] AI agent connection established successfully", map[string]any{
		"url": url,
	})

	cancelCtx, cancel := context.WithCancel(context.Background())
	c.cancelFunc = cancel

	// _ = c.conn.SetReadDeadline(time.Now().Add(constants.WebSocketReadTimeout))
	// c.conn.SetPongHandler(func(string) error {
	// 	c.mu.Lock()
	// 	c.lastPongTime = time.Now()
	// 	c.mu.Unlock()

	// 	select {
	// 	case c.pongReceived <- struct{}{}:
	// 	default:
	// 	}

	// 	return c.conn.SetReadDeadline(time.Now().Add(constants.WebSocketReadTimeout))
	// })

	go c.readLoop(cancelCtx)
	// go c.pingLoop(cancelCtx)

	return nil
}

func (c *webSocketClient) Close(ctx context.Context) error {
	c.log.InfoWithID(ctx, "[WebSocketClient: Close] Called")

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.disconnected {
		c.log.InfoWithID(ctx, "[WebSocketClient: Close] Already disconnected")
		return nil
	}

	c.connected = false
	c.disconnected = true

	if c.cancelFunc != nil {
		c.cancelFunc()
		c.cancelFunc = nil
	}

	if c.conn != nil {
		c.log.InfoWithID(ctx, "[WebSocketClient: Close] Closing connection")
		return c.conn.Close()
	}
	return nil
}

func (c *webSocketClient) StartSessionConversation(ctx context.Context, msg MsgInterviewTypeAndSessionID) error {
	c.log.InfoWithID(ctx, "[WebSocketClient: StartSessionConversation] Called:", msg)

	if c.sessionID != msg.SessionID {
		c.log.ErrorWithID(ctx, "[WebSocketClient: StartSessionConversation] Session ID mismatch")
		return errors.New("session ID mismatch")
	}

	message := map[string]interface{}{
		"type":       msg.Type,
		"session_id": msg.SessionID,
	}

	if err := c.SendMessage(ctx, message); err != nil {
		c.log.ErrorWithID(ctx, "[WebSocketClient: StartSessionConversation] Error sending session info", err)
		return err
	}

	return nil
}

func (c *webSocketClient) SegmentStart(ctx context.Context, msg MsgSegmentStart) error {
	c.log.InfoWithID(ctx, "[WebSocketClient: SegmentStart] Called:", msg)

	if c.sessionID != msg.SessionID {
		c.log.ErrorWithID(ctx, "[WebSocketClient: SegmentStart] Session ID mismatch")
		return errors.New("session ID mismatch")
	}

	message := map[string]interface{}{
		"type":       msg.Type,
		"session_id": msg.SessionID,
		"segment_id": msg.SegmentID,
	}

	c.currentSegmentID = msg.SegmentID

	if err := c.SendMessage(ctx, message); err != nil {
		c.log.ErrorWithID(ctx, "[WebSocketClient: SegmentStart] Error sending session info", err)
		return err
	}

	return nil
}

func (c *webSocketClient) SendUserAudio(ctx context.Context, msg MsgUserAudioChunk, audioData []byte) error {
	c.log.InfoWithID(ctx, "[WebSocketClient: SendAudio] Called:", msg)

	if c.sessionID != msg.SessionID {
		c.log.ErrorWithID(ctx, "[WebSocketClient: SendAudio] Session ID mismatch")
		return errors.New("session ID mismatch")
	}

	if c.currentSegmentID != msg.SegmentID {
		c.log.ErrorWithID(ctx, "[WebSocketClient: SendAudio] Segment ID mismatch")
		return errors.New("segment ID mismatch")
	}

	header := map[string]interface{}{
		"type":       msg.Type,
		"session_id": msg.SessionID,
		"segment_id": msg.SegmentID,
	}

	headerBytes, err := json.Marshal(header)
	if err != nil {
		c.log.ErrorWithID(ctx, "[WebSocketClient: SendAudio] Error marshalling header", err)
		return err
	}

	headerLength := uint32(len(headerBytes))

	payload := make([]byte, 4+len(headerBytes)+len(audioData))

	binary.BigEndian.PutUint32(payload[:4], headerLength)

	copy(payload[4:4+headerLength], headerBytes)

	copy(payload[4+headerLength:], audioData)

	if err := c.SendBinaryMessage(ctx, payload); err != nil {
		c.log.ErrorWithID(ctx, "[WebSocketClient: SendAudio] Error sending audio", err)
		return err
	}

	return nil
}

func (c *webSocketClient) SegmentEnd(ctx context.Context, msg MsgSegmentEnd) error {
	c.log.InfoWithID(ctx, "[WebSocketClient: SegmentEnd] Called:", msg)

	if c.sessionID != msg.SessionID {
		c.log.ErrorWithID(ctx, "[WebSocketClient: SegmentEnd] Session ID mismatch")
		return errors.New("session ID mismatch")
	}

	if c.currentSegmentID != msg.SegmentID {
		c.log.ErrorWithID(ctx, "[WebSocketClient: SegmentEnd] Segment ID mismatch")
		return errors.New("segment ID mismatch")
	}

	message := map[string]interface{}{
		"type":       msg.Type,
		"session_id": msg.SessionID,
		"segment_id": msg.SegmentID,
	}

	if err := c.SendMessage(ctx, message); err != nil {
		c.log.ErrorWithID(ctx, "[WebSocketClient: SegmentEnd] Error sending session info", err)
		return err
	}

	return nil
}

func (c *webSocketClient) SendMessage(ctx context.Context, data map[string]interface{}) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected || c.conn == nil || c.disconnected {
		c.log.ErrorWithID(ctx, "[WebSocketClient: SendMessage] Not connected")
		return websocket.ErrCloseSent
	}

	message, err := json.Marshal(data)
	if err != nil {
		c.log.ErrorWithID(ctx, "[WebSocketClient: SendMessage] Error marshalling message", err)
		return err
	}

	return c.conn.WriteMessage(websocket.TextMessage, message)
}

func (c *webSocketClient) SendBinaryMessage(ctx context.Context, data []byte) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected || c.conn == nil || c.disconnected {
		c.log.ErrorWithID(ctx, "[WebSocketClient: SendBinaryMessage] Not connected")
		return websocket.ErrCloseSent
	}

	return c.conn.WriteMessage(websocket.BinaryMessage, data)
}

func (c *webSocketClient) SendSessionInfo(ctx context.Context, sessionID, userID, resumeID string) error {
	c.log.InfoWithID(ctx, "[WebSocketClient: SendSessionInfo] Called:", sessionID, userID)
	c.sessionID = sessionID
	c.userID = userID
	c.resumeID = resumeID

	return c.SendMessage(ctx, map[string]interface{}{
		"session_id": sessionID,
		"user_id":    userID,
		"timestamp":  time.Now().Unix(),
	})
}

func (c *webSocketClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	ctx := context.Background()
	c.log.InfoWithID(ctx, "[WebSocketClient: IsConnected] Called:", c.connected, c.conn != nil)
	return c.connected && c.conn != nil && !c.disconnected
}

func (c *webSocketClient) SetCallbacks(callbacks WebSocketClientCallbacks) {
	ctx := context.Background()
	c.log.InfoWithID(ctx, "[WebSocketClient: SetCallbacks] Called:", callbacks)
	c.cb = callbacks
}

func (c *webSocketClient) readLoop(ctx context.Context) {
	c.log.InfoWithID(ctx, "[WebSocketClient: readLoop] Called")

	for {
		mt, data, err := c.conn.ReadMessage()
		if err != nil {
			// Check if it's a normal close from AI agent
			if closeErr, ok := err.(*websocket.CloseError); ok {
				switch closeErr.Code {
				case websocket.CloseNormalClosure:
					c.log.InfoWithID(ctx, "[WebSocketClient: readLoop] AI agent closed connection normally", map[string]any{
						"session_id": c.sessionID,
						"code":       closeErr.Code,
						"text":       closeErr.Text,
					})
				case websocket.CloseGoingAway:
					c.log.InfoWithID(ctx, "[WebSocketClient: readLoop] AI agent going away", map[string]any{
						"session_id": c.sessionID,
						"code":       closeErr.Code,
						"text":       closeErr.Text,
					})
				default:
					c.log.ErrorWithID(ctx, "[WebSocketClient: readLoop] AI agent closed connection with error", map[string]any{
						"session_id": c.sessionID,
						"code":       closeErr.Code,
						"text":       closeErr.Text,
						"error":      err,
					})
				}
			} else if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
				c.log.ErrorWithID(ctx, "[WebSocketClient: readLoop] Unexpected AI agent connection close", map[string]any{
					"session_id": c.sessionID,
					"error":      err,
				})
			} else {
				c.log.ErrorWithID(ctx, "[WebSocketClient: readLoop] AI agent connection error", map[string]any{
					"session_id": c.sessionID,
					"error":      err,
				})
			}

			c.mu.Lock()
			if !c.disconnected {
				c.connected = false
				c.mu.Unlock()
				c.cb.OnDisconnect(ctx, c.sessionID)
				c.disconnect(ctx)
			} else {
				c.mu.Unlock()
			}
			return
		}

		switch mt {
		case websocket.TextMessage:
			var base struct {
				Type      string `json:"type"`
				SessionID string `json:"session_id"`
			}
			if json.Unmarshal(data, &base) != nil {
				continue
			}

			switch base.Type {

			case constants.WebSocketMessageTypeConnectionEstablished:
				var msg MsgInterviewTypeAndSessionID

				if json.Unmarshal(data, &msg) != nil {
					c.disconnect(ctx)
					return
				}
				c.cb.OnConnectionEstablished(ctx, msg.SessionID)

			case constants.WebSocketMessageTypeUserFullTranscript:
				var msg MsgUserFullTranscript

				if json.Unmarshal(data, &msg) != nil {
					c.disconnect(ctx)
					return
				}

				c.cb.OnUserFullTranscript(ctx, msg)

			case constants.WebSocketMessageTypeInterviewerResponse:
				var msg MsgInterviewerResp

				if json.Unmarshal(data, &msg) != nil {
					c.disconnect(ctx)
					return
				}
				c.cb.OnInterviewerResp(ctx, msg)

			case constants.WebSocketMessageTypeInterviewTurnStart:
				var msg MsgInterviewTypeAndSessionID

				if json.Unmarshal(data, &msg) != nil {
					c.disconnect(ctx)
					return
				}
				c.cb.OnInterviewTurnStart(ctx, msg)

			case constants.WebSocketMessageTypeInterviewTurnEnd:
				var msg MsgInterviewTypeAndSessionID

				if json.Unmarshal(data, &msg) != nil {
					c.disconnect(ctx)
					return
				}
				c.cb.OnInterviewTurnEnd(ctx, msg)

			case constants.WebSocketMessageTypeInterviewCompleted:
				var msg MsgInterviewTypeAndSessionID
				if json.Unmarshal(data, &msg) != nil {
					c.disconnect(ctx)
					return
				}
				c.cb.OnInterviewCompleted(ctx, msg)
			}

		case websocket.BinaryMessage:
			c.cb.OnInterviewerAudioChunk(ctx, data)
		}
	}
}

func (c *webSocketClient) pingLoop(ctx context.Context) {
	c.log.InfoWithID(ctx, "[WebSocketClient: pingLoop] Starting ping loop")

	t := time.NewTicker(constants.WebSocketPingInterval)
	defer t.Stop()

	for range t.C {
		timeSinceLastPong := time.Since(c.lastPongTime)

		if timeSinceLastPong > constants.WebSocketPongTimeout {
			c.log.ErrorWithID(ctx, "[WebSocketClient: pingLoop] Pong timeout - no pong received", map[string]interface{}{
				"session_id":           c.sessionID,
				"time_since_last_pong": timeSinceLastPong,
				"timeout":              constants.WebSocketPongTimeout,
			})
			c.connected = false
			c.cb.OnDisconnect(ctx, c.sessionID)
			c.disconnect(ctx)
			return
		}

		err := c.conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(constants.WebSocketPingDuration))
		if err != nil {
			c.log.ErrorWithID(ctx, "[WebSocketClient: pingLoop] Error writing ping message", err)
			c.connected = false
			c.cb.OnDisconnect(ctx, c.sessionID)
			c.disconnect(ctx)
			return
		}
	}
}

func (c *webSocketClient) disconnect(ctx context.Context) {
	c.log.InfoWithID(ctx, "[WebSocketClient: disconnect] Disconnecting AI agent client", map[string]any{
		"session_id": c.sessionID,
		"connected":  c.connected,
	})

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.disconnected {
		c.log.InfoWithID(ctx, "[WebSocketClient: disconnect] Already disconnected")
		return
	}

	c.disconnected = true
	c.connected = false

	if c.cancelFunc != nil {
		c.cancelFunc()
		c.cancelFunc = nil
	}

	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			c.log.ErrorWithID(ctx, "[WebSocketClient: disconnect] Error closing AI agent connection", map[string]any{
				"session_id": c.sessionID,
				"error":      err,
			})
		} else {
			c.log.InfoWithID(ctx, "[WebSocketClient: disconnect] AI agent connection closed successfully", map[string]any{
				"session_id": c.sessionID,
			})
		}
	}
}
