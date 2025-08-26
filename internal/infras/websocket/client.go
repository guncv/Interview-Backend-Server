package websocket

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gorilla/websocket"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
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
	Start(ctx context.Context) error
	Close() error

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
	url       string
	h         httpHeader
	cb        WebSocketCallbacks
	sessionID string
	userID    string

	conn      *websocket.Conn
	connected bool
}

type httpHeader map[string]string

func NewWebSocketClient(url string, headers map[string]string) WebSocketClient {
	return &webSocketClient{
		url:       url,
		h:         headers,
		cb:        WebSocketCallbacks{},
		conn:      nil,
		connected: false,
	}
}

func (c *webSocketClient) Start(ctx context.Context) error {
	d := websocket.Dialer{HandshakeTimeout: 10 * time.Second}
	var hdr = make(map[string][]string)
	for k, v := range c.h {
		hdr[k] = []string{v}
	}
	conn, _, err := d.DialContext(ctx, c.url, hdr)
	if err != nil {
		return err
	}
	c.conn = conn
	c.connected = true

	_ = c.conn.SetReadDeadline(time.Now().Add(constants.WebSocketReadTimeout))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(constants.WebSocketReadTimeout))
	})

	go c.readLoop()
	go c.pingLoop(constants.WebSocketPingInterval)
	return nil
}

func (c *webSocketClient) Close() error {
	c.connected = false
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *webSocketClient) SegmentStart(ctx context.Context, seg string, sr int, enc string, ch int) error {
	if c.sessionID == "" {
		return fmt.Errorf("session ID not set")
	}

	return c.conn.WriteJSON(map[string]any{
		"type":        "segment_start",
		"session_id":  c.sessionID,
		"segment_id":  seg,
		"sample_rate": sr,
		"encoding":    enc,
		"channels":    ch,
	})
}

func (c *webSocketClient) SendAudio(ctx context.Context, seg string, buf []byte) error {
	if c.sessionID == "" {
		return fmt.Errorf("session ID not set")
	}

	header := map[string]string{
		"type":       "audio_chunk",
		"session_id": c.sessionID,
		"segment_id": seg,
	}

	headerBytes, err := json.Marshal(header)
	if err != nil {
		return err
	}

	frame := make([]byte, 4+len(headerBytes)+len(buf))
	binary.BigEndian.PutUint32(frame[:4], uint32(len(headerBytes)))
	copy(frame[4:], headerBytes)
	copy(frame[4+len(headerBytes):], buf)

	return c.conn.WriteMessage(websocket.BinaryMessage, frame)
}

func (c *webSocketClient) SegmentEnd(ctx context.Context, seg string) error {
	if c.sessionID == "" {
		return fmt.Errorf("session ID not set")
	}

	return c.conn.WriteJSON(map[string]any{
		"type":       "segment_end",
		"session_id": c.sessionID,
		"segment_id": seg,
		"timestamp":  time.Now().Unix(),
	})
}

func (c *webSocketClient) StopTTS(ctx context.Context, seg string) error {
	if c.sessionID == "" {
		return fmt.Errorf("session ID not set")
	}

	return c.conn.WriteJSON(map[string]any{
		"type":       "stop_tts",
		"session_id": c.sessionID,
		"segment_id": seg,
	})
}

func (c *webSocketClient) SendMessage(ctx context.Context, msgType string, data map[string]interface{}) error {
	if !c.connected || c.conn == nil {
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
	c.sessionID = sessionID
	c.userID = userID

	return c.SendMessage(ctx, "session_info", map[string]interface{}{
		"session_id": sessionID,
		"user_id":    userID,
		"timestamp":  time.Now().Unix(),
	})
}

func (c *webSocketClient) IsConnected() bool {
	return c.connected && c.conn != nil
}

func (c *webSocketClient) SetCallbacks(callbacks WebSocketCallbacks) {
	c.cb = callbacks
}

func (c *webSocketClient) readLoop() {
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

			case "asr":
				var x struct {
					SegmentID string  `json:"segment_id"`
					Text      string  `json:"text"`
					IsFinal   bool    `json:"is_final"`
					Seq       int     `json:"seq"`
					Stability float64 `json:"stability"`
				}
				if json.Unmarshal(data, &x) != nil {
					continue
				}
				if x.IsFinal {
					if c.cb.OnASRFinal != nil {
						c.cb.OnASRFinal(x.SegmentID, x.Text, x.Seq)
					}
				} else {
					if c.cb.OnASRPartial != nil {
						c.cb.OnASRPartial(x.SegmentID, x.Text, x.Seq, x.Stability)
					}
				}

			case "evaluation":
				var x struct {
					SegmentID string  `json:"segment_id"`
					Score     float64 `json:"score"`
					Comment   string  `json:"comment"`
				}
				if json.Unmarshal(data, &x) == nil && c.cb.OnEvaluation != nil {
					c.cb.OnEvaluation(x.SegmentID, x.Score, x.Comment)
				}

			case "tts_start":
				var x struct {
					SegmentID string `json:"segment_id"`
					TTSID     string `json:"tts_id"`
					Encoding  string `json:"encoding"`
				}
				if json.Unmarshal(data, &x) == nil && c.cb.OnTTSStart != nil {
					c.cb.OnTTSStart(x.SegmentID, x.TTSID, x.Encoding)
				}

			case "tts_end":
				var x struct {
					SegmentID string `json:"segment_id"`
					TTSID     string `json:"tts_id"`
				}
				if json.Unmarshal(data, &x) == nil && c.cb.OnTTSEnd != nil {
					c.cb.OnTTSEnd(x.SegmentID, x.TTSID)
				}

			case "ping":
				var x struct {
					Timestamp float64 `json:"timestamp"`
				}
				if json.Unmarshal(data, &x) == nil {
					if c.cb.OnPing != nil {
						c.cb.OnPing(x.Timestamp)
					}
					c.conn.WriteJSON(map[string]interface{}{
						"type":      "pong",
						"timestamp": x.Timestamp,
					})
				}

			case "error":
				var x struct{ Code, Message string }
				if json.Unmarshal(data, &x) == nil && c.cb.OnError != nil {
					c.cb.OnError(x.Code, x.Message)
				}
			}

		case websocket.BinaryMessage:
			if len(data) >= 4 {
				headerLen := binary.BigEndian.Uint32(data[:4])
				if int(headerLen) <= len(data)-4 {
					headerData := data[4 : 4+headerLen]
					var header struct {
						SegmentID string `json:"segment_id"`
					}
					if json.Unmarshal(headerData, &header) == nil && c.cb.OnTTSChunk != nil {
						c.cb.OnTTSChunk(header.SegmentID, data[4+headerLen:])
					}
				}
			}
		}
	}
}

func (c *webSocketClient) pingLoop(every time.Duration) {
	t := time.NewTicker(every)
	defer t.Stop()
	for range t.C {
		if c.conn != nil {
			_ = c.conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(5*time.Second))
		}
	}
}
