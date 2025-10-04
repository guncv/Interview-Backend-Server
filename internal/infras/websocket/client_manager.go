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
	m.mu.Lock()
	defer m.mu.Unlock()
	m.clients[sessionID] = client
}

func (m *ClientManager) GetClientBySessionID(ctx context.Context, sessionID string) (WebSocketClient, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	client, ok := m.clients[sessionID]
	return client, ok
}

func (m *ClientManager) DeleteClientBySessionID(ctx context.Context, sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.clients, sessionID)
}

func (m *ClientManager) CloseAllClients(ctx context.Context) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, client := range m.clients {
		_ = client.Close(ctx)
	}
	m.clients = make(map[string]WebSocketClient)
}
