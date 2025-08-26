package websocket

import (
	"context"
	"encoding/json"
	"time"

	"github.com/gorilla/websocket"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
)

type WebSocketCallbacks struct {
	OnASRPartial func(segmentID, text string, seq int, stability float64)
	OnASRFinal   func(segmentID, text string, seq int)
	OnTTSStart   func(segmentID, ttsID, encoding string)
	OnTTSChunk   func(segmentID string, data []byte)
	OnTTSEnd     func(segmentID, ttsID string)
	OnError      func(code, msg string)
	OnEvaluation func(segmentID string, score float64, comment string)
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
}

type webSocketClient struct {
	url string
	h   httpHeader
	cb  WebSocketCallbacks

	conn      *websocket.Conn
	connected bool
}

type httpHeader map[string]string

func NewWebSocketClient(url string, headers map[string]string, cb WebSocketCallbacks) WebSocketClient {
	return &webSocketClient{
		url:       url,
		h:         headers,
		cb:        cb,
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
	return c.conn.WriteJSON(map[string]any{
		"v": 1, "type": "segment_start", "segment_id": seg,
		"sample_rate": sr, "encoding": enc, "channels": ch,
	})
}

func (c *webSocketClient) SendAudio(ctx context.Context, seg string, buf []byte) error {
	return c.conn.WriteMessage(websocket.BinaryMessage, buf)
}

func (c *webSocketClient) SegmentEnd(ctx context.Context, seg string) error {
	return c.conn.WriteJSON(map[string]any{"v": 1, "type": "segment_end", "segment_id": seg})
}

func (c *webSocketClient) StopTTS(ctx context.Context, seg string) error {
	return c.conn.WriteJSON(map[string]any{"v": 1, "type": "stop_tts", "segment_id": seg})
}

func (c *webSocketClient) SendMessage(ctx context.Context, msgType string, data map[string]interface{}) error {
	if !c.connected || c.conn == nil {
		return websocket.ErrCloseSent
	}

	message := map[string]interface{}{
		"v":    1,
		"type": msgType,
	}

	for k, v := range data {
		message[k] = v
	}

	return c.conn.WriteJSON(message)
}

func (c *webSocketClient) SendSessionInfo(ctx context.Context, sessionID, userID string) error {
	return c.SendMessage(ctx, "session_info", map[string]interface{}{
		"session_id": sessionID,
		"user_id":    userID,
		"timestamp":  time.Now().Unix(),
	})
}

func (c *webSocketClient) IsConnected() bool {
	return c.connected && c.conn != nil
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
			}
			if json.Unmarshal(data, &base) != nil {
				continue
			}
			switch base.Type {
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

			case "error":
				var x struct{ Code, Message string }
				if json.Unmarshal(data, &x) == nil && c.cb.OnError != nil {
					c.cb.OnError(x.Code, x.Message)
				}
			}

		case websocket.BinaryMessage:
			if c.cb.OnTTSChunk != nil {
				c.cb.OnTTSChunk("", data)
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
