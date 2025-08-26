package websocket

import (
	"sync"
)

type ClientManager struct {
	mu      sync.RWMutex
	clients map[string]WebSocketClient
}

func NewClientManager() *ClientManager {
	return &ClientManager{
		clients: make(map[string]WebSocketClient),
	}
}

func (m *ClientManager) AddClient(sessionID string, client WebSocketClient) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.clients[sessionID] = client
}

func (m *ClientManager) GetClient(sessionID string) (WebSocketClient, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	client, ok := m.clients[sessionID]
	return client, ok
}

func (m *ClientManager) HasClient(sessionID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.clients[sessionID]
	return ok
}

func (m *ClientManager) RemoveClient(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if client, ok := m.clients[sessionID]; ok {
		client.Close()
		delete(m.clients, sessionID)
	}
}

func (m *ClientManager) CloseAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for sessionID, client := range m.clients {
		client.Close()
		delete(m.clients, sessionID)
	}
}
