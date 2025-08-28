package websocket

type WebSocketClientCallbacks struct {
	OnConnectionEstablished func(sessionID string)
	OnDisconnect            func(sessionID string)
}

func NewWebSocketClientCallbacks() *WebSocketClientCallbacks {
	return &WebSocketClientCallbacks{
		OnConnectionEstablished: func(sessionID string) {},
		OnDisconnect:            func(sessionID string) {},
	}
}

func (w *WebSocketClientCallbacks) WithConnectionEstablished(callback func(sessionID string)) *WebSocketClientCallbacks {
	w.OnConnectionEstablished = callback
	return w
}

func (w *WebSocketClientCallbacks) WithDisconnect(callback func(sessionID string)) *WebSocketClientCallbacks {
	w.OnDisconnect = callback
	return w
}
