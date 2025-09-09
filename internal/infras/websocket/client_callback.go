package websocket

import (
	"context"

	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
)

type WebSocketClientCallbacks interface {
	OnConnectionEstablished(ctx context.Context, sessionID string)
	OnDisconnect(ctx context.Context, sessionID string)
	OnUserPartialTranscript(ctx context.Context, req MsgUserPartialTranscript)
	OnUserFullTranscript(ctx context.Context, req MsgUserFullTranscript)
	OnAIResponse(ctx context.Context, req MsgAIResponse)
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

	w.server.Disconnect(ctx, w.client)
}

func (w *webSocketClientCallbacks) OnUserPartialTranscript(ctx context.Context, req MsgUserPartialTranscript) {
	w.log.InfoWithID(ctx, "[WebSocketClientCallbacks: OnUserPartialTranscript] Agent sent partial transcript", map[string]any{
		"session_id": req.SessionID,
		"transcript": req.Transcript,
	})

	if req.SessionID != w.client.sessionID {
		w.log.ErrorWithID(ctx, "[WebSocketClientCallbacks: OnUserPartialTranscript] Security violation: Session ID mismatch", map[string]any{
			"session_id": req.SessionID,
		})
		w.server.Disconnect(ctx, w.client)
		return
	}

	if req.SegmentID != w.client.currentSegmentID {
		w.log.ErrorWithID(ctx, "[WebSocketClientCallbacks: OnUserPartialTranscript] Security violation: Segment ID mismatch", map[string]any{
			"session_id": req.SessionID,
			"segment_id": req.SegmentID,
		})
		w.server.Disconnect(ctx, w.client)
		return
	}

	w.logic.sendMessageTypeUserPartialTranscript(ctx, w.client, req)
}

func (w *webSocketClientCallbacks) OnUserFullTranscript(ctx context.Context, req MsgUserFullTranscript) {
	w.log.InfoWithID(ctx, "[WebSocketClientCallbacks: OnUserFullTranscript] Agent sent full transcript", map[string]any{
		"session_id": req.SessionID,
		"transcript": req.Transcript,
	})

	if req.SessionID != w.client.sessionID {
		w.log.ErrorWithID(ctx, "[WebSocketClientCallbacks: OnUserFullTranscript] Security violation: Session ID mismatch", map[string]any{
			"session_id": req.SessionID,
		})
		w.server.Disconnect(ctx, w.client)
		return
	}

	if req.SegmentID != w.client.currentSegmentID {
		w.log.ErrorWithID(ctx, "[WebSocketClientCallbacks: OnUserFullTranscript] Security violation: Segment ID mismatch", map[string]any{
			"session_id": req.SessionID,
			"segment_id": req.SegmentID,
		})
		w.server.Disconnect(ctx, w.client)
		return
	}

	w.logic.sendMessageTypeUserFullTranscript(ctx, w.client, req)
}

func (w *webSocketClientCallbacks) OnAIResponse(ctx context.Context, req MsgAIResponse) {
	w.log.InfoWithID(ctx, "[WebSocketClientCallbacks: OnAIResponse] Agent sent AI response", map[string]any{
		"session_id": req.SessionID,
		"message":    req.Message,
	})

	if req.SessionID != w.client.sessionID {
		w.log.ErrorWithID(ctx, "[WebSocketClientCallbacks: OnAIResponse] Security violation: Session ID mismatch", map[string]any{
			"session_id": req.SessionID,
		})
		w.server.Disconnect(ctx, w.client)
		return
	}

	w.logic.sendMessageTypeAIResponse(ctx, w.client, req)
}
