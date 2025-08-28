package websocket

func NewWebSocketCallbacks() *WebSocketCallbacks {
	return &WebSocketCallbacks{
		OnConnectionEstablished: func(sessionID string) {},
		OnDisconnect:            func(sessionID string) {},
	}
}

func (w *WebSocketCallbacks) WithConnectionEstablished(callback func(sessionID string)) *WebSocketCallbacks {
	w.OnConnectionEstablished = callback
	return w
}

func (w *WebSocketCallbacks) WithDisconnect(callback func(sessionID string)) *WebSocketCallbacks {
	w.OnDisconnect = callback
	return w
}
