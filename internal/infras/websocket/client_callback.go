package websocket

func NewWebSocketCallbacks() *WebSocketCallbacks {
	return &WebSocketCallbacks{
		OnConnectionEstablished: func(sessionID string) {},
		OnPing:                  func(timestamp float64) {},
		OnASRPartial:            func(segmentID, text string, seq int, stability float64) {},
		OnASRFinal:              func(segmentID, text string, seq int) {},
		OnTTSStart:              func(segmentID, ttsID, encoding string) {},
		OnTTSChunk:              func(segmentID string, data []byte) {},
		OnTTSEnd:                func(segmentID, ttsID string) {},
		OnEvaluation:            func(segmentID string, score float64, comment string) {},
		OnError:                 func(code, msg string) {},
	}
}

func (w *WebSocketCallbacks) WithConnectionEstablished(callback func(sessionID string)) *WebSocketCallbacks {
	w.OnConnectionEstablished = callback
	return w
}

func (w *WebSocketCallbacks) WithPing(callback func(timestamp float64)) *WebSocketCallbacks {
	w.OnPing = callback
	return w
}

func (w *WebSocketCallbacks) WithASRPartial(callback func(segmentID, text string, seq int, stability float64)) *WebSocketCallbacks {
	w.OnASRPartial = callback
	return w
}

func (w *WebSocketCallbacks) WithASRFinal(callback func(segmentID, text string, seq int)) *WebSocketCallbacks {
	w.OnASRFinal = callback
	return w
}

func (w *WebSocketCallbacks) WithTTSStart(callback func(segmentID, ttsID, encoding string)) *WebSocketCallbacks {
	w.OnTTSStart = callback
	return w
}

func (w *WebSocketCallbacks) WithTTSChunk(callback func(segmentID string, data []byte)) *WebSocketCallbacks {
	w.OnTTSChunk = callback
	return w
}

func (w *WebSocketCallbacks) WithTTSEnd(callback func(segmentID, ttsID string)) *WebSocketCallbacks {
	w.OnTTSEnd = callback
	return w
}

func (w *WebSocketCallbacks) WithEvaluation(callback func(segmentID string, score float64, comment string)) *WebSocketCallbacks {
	w.OnEvaluation = callback
	return w
}

func (w *WebSocketCallbacks) WithError(callback func(code, msg string)) *WebSocketCallbacks {
	w.OnError = callback
	return w
}
