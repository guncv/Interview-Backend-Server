package websocket

import (
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

func (m *ClientManager) Set(sessionID string, client WebSocketClient) {
	m.log.Info("[ClientManager] Set", "sessionID", sessionID)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.clients[sessionID] = client
}

func (m *ClientManager) Get(sessionID string) (WebSocketClient, bool) {
	m.log.Info("[ClientManager] Get", "sessionID", sessionID)
	m.mu.RLock()
	defer m.mu.RUnlock()
	client, ok := m.clients[sessionID]
	return client, ok
}

func (m *ClientManager) Delete(sessionID string) {
	m.log.Info("[ClientManager] Delete", "sessionID", sessionID)
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.clients, sessionID)
}

func (m *ClientManager) CloseAll() {
	m.log.Info("[ClientManager] CloseAll")
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, client := range m.clients {
		_ = client.Close()
	}
	m.clients = make(map[string]WebSocketClient)
}
