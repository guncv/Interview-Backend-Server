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
	OnInterviewTurnStart(ctx context.Context, req MsgInterviewTypeAndSessionID)
	OnInterviewTurnEnd(ctx context.Context, req MsgInterviewTypeAndSessionID)
	OnInterviewCompleted(ctx context.Context, req MsgInterviewTypeAndSessionID)
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
}

func (w *webSocketClientCallbacks) OnDisconnect(ctx context.Context, sessionID string) {

	if w.client != nil {
		w.client.mu.Lock()
		if w.client.disconnecting {
			w.client.mu.Unlock()
			return
		}
		w.client.mu.Unlock()

		w.server.Disconnect(ctx, w.client)
	}
}

func (w *webSocketClientCallbacks) OnUserFullTranscript(ctx context.Context, req MsgUserFullTranscript) {
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
	var req MsgInterviewTypeAndSessionID
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
	w.logic.sendMessageTypeInterviewerAudioChunk(ctx, w.client, req, audioData)
}

func (w *webSocketClientCallbacks) OnInterviewTurnStart(ctx context.Context, req MsgInterviewTypeAndSessionID) {
	if req.SessionID != w.client.SessionID {
		w.log.ErrorWithID(ctx, "[WebSocketClientCallbacks: OnInterviewTurnStart] Security violation: Session ID mismatch", map[string]any{
			"session_id": req.SessionID,
		})
		w.server.Disconnect(ctx, w.client)
		return
	}

	w.logic.sendMessageTypeInterviewTurnStart(ctx, w.client, req)
}

func (w *webSocketClientCallbacks) OnInterviewTurnEnd(ctx context.Context, req MsgInterviewTypeAndSessionID) {
	if req.SessionID != w.client.SessionID {
		w.log.ErrorWithID(ctx, "[WebSocketClientCallbacks: OnInterviewTurnEnd] Security violation: Session ID mismatch", map[string]any{
			"session_id": req.SessionID,
		})
		w.server.Disconnect(ctx, w.client)
		return
	}

	w.logic.sendMessageTypeInterviewTurnEnd(ctx, w.client, req)
}

func (w *webSocketClientCallbacks) OnInterviewCompleted(ctx context.Context, req MsgInterviewTypeAndSessionID) {
	if req.SessionID != w.client.SessionID {
		w.log.ErrorWithID(ctx, "[WebSocketClientCallbacks: OnInterviewCompleted] Security violation: Session ID mismatch", map[string]any{
			"session_id": req.SessionID,
		})
		w.server.Disconnect(ctx, w.client)
		return
	}

	w.logic.sendMessageTypeInterviewCompleted(ctx, w.client, req)
}
