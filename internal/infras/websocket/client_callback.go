package websocket

import (
	"context"

	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
)

type WebSocketClientCallbacks interface {
	OnConnectionEstablished(ctx context.Context, sessionID string)
	OnDisconnect(ctx context.Context, sessionID string)
}

type webSocketClientCallbacks struct {
	server WebSocketServerInterface
	client *Client
	log    *log.Logger
}

func NewWebSocketClientCallbacks(s WebSocketServerInterface, c *Client, l *log.Logger) WebSocketClientCallbacks {

	return &webSocketClientCallbacks{
		server: s,
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
