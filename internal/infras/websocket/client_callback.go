package websocket

import (
	"context"

	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
)

type WebSocketClientCallbacks interface {
	OnConnectionEstablished(ctx context.Context, sessionID string)
	OnDisconnect(ctx context.Context, sessionID string)
	OnUserPartialTranscript(ctx context.Context, req MsgUserPartialTranscript)
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

	w.logic.sendMessageTypeUserPartialTranscript(ctx, w.client, req)
}
