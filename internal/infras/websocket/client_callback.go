package websocket

import (
	"context"
	"encoding/binary"
	"encoding/json"

	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
)

type WebSocketClientCallbacks interface {
	OnConnectionEstablished(ctx context.Context, sessionID string)
	OnDisconnect(ctx context.Context, sessionID string)
	OnUserFullTranscript(ctx context.Context, req MsgUserFullTranscript)
	OnInterviewerResp(ctx context.Context, req MsgInterviewerResp)
	OnInterviewerAudioChunk(ctx context.Context, data []byte)
	OnInterviewTurnStart(ctx context.Context, req MsgInterviewTurnStart)
	OnInterviewTurnEnd(ctx context.Context, req MsgInterviewTurnEnd)
}

type webSocketClientCallbacks struct {
	server WebSocketServerInterface
	logic  *WebSocketServerLogic
	client *Client
	log    *log.Logger
}

func NewWebSocketClientCallbacks(
	s WebSocketServerInterface,
	logic *WebSocketServerLogic,
	c *Client,
	l *log.Logger,
) WebSocketClientCallbacks {

	return &webSocketClientCallbacks{
		server: s,
		logic:  logic,
		client: c,
		log:    l,
	}
}

func (w *webSocketClientCallbacks) OnConnectionEstablished(ctx context.Context, sessionID string) {
	w.log.InfoWithID(ctx, "[WebSocketClientCallbacks: WithConnectionEstablished] Agent connected", map[string]any{
		"session_id": sessionID,
	})
}

func (w *webSocketClientCallbacks) OnDisconnect(ctx context.Context, sessionID string) {
	w.log.InfoWithID(ctx, "[WebSocketClientCallbacks: WithDisconnect] Agent disconnected", map[string]any{
		"session_id": sessionID,
	})

	// Check if client is already disconnecting to prevent recursive calls
	if w.client != nil {
		w.client.mu.Lock()
		if w.client.disconnecting {
			w.client.mu.Unlock()
			w.log.InfoWithID(ctx, "[WebSocketClientCallbacks: WithDisconnect] Client already disconnecting", map[string]any{
				"session_id": sessionID,
			})
			return
		}
		w.client.mu.Unlock()

		w.server.Disconnect(ctx, w.client)
	}
}

func (w *webSocketClientCallbacks) OnUserFullTranscript(ctx context.Context, req MsgUserFullTranscript) {
	w.log.InfoWithID(ctx, "[WebSocketClientCallbacks: OnUserFullTranscript] Agent sent full transcript", map[string]any{
		"session_id": req.SessionID,
		"transcript": req.Transcript,
	})

	if req.SessionID != w.client.SessionID {
		w.log.ErrorWithID(ctx, "[WebSocketClientCallbacks: OnUserFullTranscript] Security violation: Session ID mismatch", map[string]any{
			"session_id": req.SessionID,
		})
		w.server.Disconnect(ctx, w.client)
		return
	}

	if req.SegmentID != w.client.CurrentSegmentID {
		w.log.ErrorWithID(ctx, "[WebSocketClientCallbacks: OnUserFullTranscript] Security violation: Segment ID mismatch", map[string]any{
			"session_id": req.SessionID,
			"segment_id": req.SegmentID,
		})
		w.server.Disconnect(ctx, w.client)
		return
	}

	w.logic.sendMessageTypeUserFullTranscript(ctx, w.client, req)
}

func (w *webSocketClientCallbacks) OnInterviewerResp(ctx context.Context, req MsgInterviewerResp) {
	w.log.InfoWithID(ctx, "[WebSocketClientCallbacks: OnInterviewerResp] Called")

	if req.SessionID != w.client.SessionID {
		w.log.ErrorWithID(ctx, "[WebSocketClientCallbacks: OnInterviewerResp] Security violation: Session ID mismatch", map[string]any{
			"session_id": req.SessionID,
		})
		w.server.Disconnect(ctx, w.client)
		return
	}

	w.logic.sendMessageTypeInterviewerResp(ctx, w.client, req)
}

func (w *webSocketClientCallbacks) OnInterviewerAudioChunk(ctx context.Context, data []byte) {
	w.log.InfoWithID(ctx, "[WebSocketClientCallbacks: OnInterviewerAudioChunk] Called: ")

	if len(data) < 4 {
		w.log.ErrorWithID(ctx, "[WebSocketClientCallbacks: OnInterviewerAudioChunk] Invalid frame: too short")
		return
	}

	headerLength := binary.BigEndian.Uint32(data[:4])
	if int(headerLength)+4 > len(data) {
		w.log.ErrorWithID(ctx, "[WebSocketClientCallbacks: OnInterviewerAudioChunk] Invalid frame: header length too large")
		return
	}

	headerBytes := data[4 : 4+headerLength]
	var req MsgInterviewerAudioChunk
	if err := json.Unmarshal(headerBytes, &req); err != nil {
		w.log.ErrorWithID(ctx, "[WebSocketClientCallbacks: OnInterviewerAudioChunk] Invalid header JSON", err)
		return
	}

	if req.SessionID != w.client.SessionID {
		w.log.ErrorWithID(ctx, "[WebSocketClientCallbacks: OnInterviewerAudioChunk] Security violation: Session ID mismatch", map[string]any{
			"session_id": req.SessionID,
		})
		w.server.Disconnect(ctx, w.client)
		return
	}

	audioData := data[4+headerLength:]
	w.log.InfoWithID(ctx, "[WebSocketClientCallbacks: OnInterviewerAudioChunk] Audio chunk received", map[string]any{
		"session_id": req.SessionID,
		"audio_size": len(audioData),
	})

	w.logic.sendMessageTypeInterviewerAudioChunk(ctx, w.client, req, audioData)
}

func (w *webSocketClientCallbacks) OnInterviewTurnStart(ctx context.Context, req MsgInterviewTurnStart) {
	w.log.InfoWithID(ctx, "[WebSocketClientCallbacks: OnInterviewTurnStart] Called")

	if req.SessionID != w.client.SessionID {
		w.log.ErrorWithID(ctx, "[WebSocketClientCallbacks: OnInterviewTurnStart] Security violation: Session ID mismatch", map[string]any{
			"session_id": req.SessionID,
		})
	}

	w.logic.sendMessageTypeInterviewTurnStart(ctx, w.client, req)
}

func (w *webSocketClientCallbacks) OnInterviewTurnEnd(ctx context.Context, req MsgInterviewTurnEnd) {
	w.log.InfoWithID(ctx, "[WebSocketClientCallbacks: OnInterviewTurnEnd] Called")

	if req.SessionID != w.client.SessionID {
		w.log.ErrorWithID(ctx, "[WebSocketClientCallbacks: OnInterviewTurnEnd] Security violation: Session ID mismatch", map[string]any{
			"session_id": req.SessionID,
		})
	}

	w.logic.sendMessageTypeInterviewTurnEnd(ctx, w.client, req)
}
