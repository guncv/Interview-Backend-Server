package websocket

import (
	"context"
	"sync"

	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
)

type ClientManager struct {
	mu      sync.RWMutex
	clients map[string]WebSocketClient
	log     *log.Logger
}

func NewClientManager(log *log.Logger) *ClientManager {

	return &ClientManager{
		clients: make(map[string]WebSocketClient),
		log:     log,
	}
}

func (m *ClientManager) SetClientBySessionID(ctx context.Context, sessionID string, client WebSocketClient) {
	m.log.InfoWithID(ctx, "[ClientManager: Called] Set", "sessionID", sessionID)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.clients[sessionID] = client
}

func (m *ClientManager) GetClientBySessionID(ctx context.Context, sessionID string) (WebSocketClient, bool) {
	m.log.InfoWithID(ctx, "[ClientManager: Called] Get", "sessionID", sessionID)
	m.mu.RLock()
	defer m.mu.RUnlock()
	client, ok := m.clients[sessionID]
	return client, ok
}

func (m *ClientManager) DeleteClientBySessionID(ctx context.Context, sessionID string) {
	m.log.InfoWithID(ctx, "[ClientManager: Called] Delete", "sessionID", sessionID)
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.clients, sessionID)
}

func (m *ClientManager) CloseAllClients(ctx context.Context) {
	m.log.InfoWithID(ctx, "[ClientManager: Called] CloseAll")
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, client := range m.clients {
		_ = client.Close()
	}
	m.clients = make(map[string]WebSocketClient)
}
