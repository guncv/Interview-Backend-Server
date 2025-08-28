package websocket

import (
	"context"
	"encoding/binary"
	"encoding/json"

	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
)

type WebSocketServerCallbacks struct {
	log           *log.Logger
	disconnect    func(ctx context.Context, client *Client)
	writeJSON     func(client *Client, data any) error
	clientManager *ClientManager
}

func NewWebSocketServerCallbacks(
	log *log.Logger,
	disconnect func(ctx context.Context, client *Client),
	writeJSON func(client *Client, data any) error,
	clientManager *ClientManager,
) *WebSocketServerCallbacks {

	return &WebSocketServerCallbacks{
		log:           log,
		disconnect:    disconnect,
		writeJSON:     writeJSON,
		clientManager: clientManager,
	}
}

func (s *WebSocketServerCallbacks) handleAudioBinaryMessage(ctx context.Context, client *Client, payload []byte) {
	s.log.InfoWithID(ctx, "[WebSocketServer: handleAudioBinaryMessage] Called")

	if len(payload) < 4 {
		s.log.ErrorWithID(ctx, "Invalid frame: too short")
		s.disconnect(ctx, client)
		return
	}

	headerLength := binary.BigEndian.Uint32(payload[:4])

	if int(headerLength)+4 > len(payload) {
		s.log.ErrorWithID(ctx, "Invalid frame: header length too large")
		s.disconnect(ctx, client)
		return
	}

	headerBytes := payload[4 : 4+headerLength]

	var header msgAudioChunk
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		s.log.ErrorWithID(ctx, "Invalid header JSON", err)
		s.disconnect(ctx, client)
		return
	}

	if header.SessionID != client.sessionID {
		s.log.ErrorWithID(ctx, "Security violation: Session ID mismatch", map[string]any{
			"expected_session_id": client.sessionID,
			"received_session_id": header.SessionID,
			"user_id":             client.userID,
		})
		_ = s.writeJSON(client, msgError{
			Type:    "error",
			Code:    string(app_error.ErrCodeWebSocketInvalidMessage),
			Message: "Session ID mismatch",
		})
		s.disconnect(ctx, client)
		return
	}

	if header.SegmentID != client.currentSegmentID {
		s.log.ErrorWithID(ctx, "Security violation: Segment ID mismatch", map[string]any{
			"expected_segment_id": client.currentSegmentID,
			"received_segment_id": header.SegmentID,
			"session_id":          client.sessionID,
			"user_id":             client.userID,
		})
		_ = s.writeJSON(client, msgError{
			Type:    "error",
			Code:    string(app_error.ErrCodeWebSocketInvalidMessage),
			Message: "Segment ID mismatch",
		})
		s.disconnect(ctx, client)
		return
	}

	audioData := payload[4+headerLength:]

	if agentClient, exists := s.clientManager.GetClientBySessionID(ctx, client.sessionID); exists {
		if err := agentClient.SendAudio(ctx, header.SegmentID, audioData); err != nil {
			s.log.ErrorWithID(ctx, "[WebSocketServer: handleAudioBinaryMessage] Error forwarding audio to AI agent", err)
		}
	}
}

func (s *WebSocketServerCallbacks) sendMessageTypeSegmentStart(ctx context.Context, client *Client, payload []byte) {
	s.log.InfoWithID(ctx, "[WebSocketServer: sendMessageTypeSegmentStart] Called")

	var m msgSegmentStart
	if json.Unmarshal(payload, &m) != nil || m.SegmentID == "" {
		_ = s.writeJSON(client, msgError{
			Type:    "error",
			Code:    string(app_error.ErrCodeWebSocketInvalidSegmentStart),
			Message: app_error.ErrCodeWebSocketInvalidSegmentStart.Message(),
		})
		s.disconnect(ctx, client)
		return
	}

	if client.sessionID != m.SessionID {
		s.log.ErrorWithID(ctx, "Security violation: Session ID mismatch", map[string]any{
			"expected_session_id": client.sessionID,
			"received_session_id": m.SessionID,
			"user_id":             client.userID,
		})
		s.disconnect(ctx, client)
		return
	}

	client.currentSegmentID = m.SegmentID

	if agentClient, exists := s.clientManager.GetClientBySessionID(ctx, client.sessionID); exists {
		if err := agentClient.SegmentStart(ctx, m.SegmentID, m.SampleRate, m.Encoding, m.Channels); err != nil {
			s.log.ErrorWithID(ctx, "[WebSocketServer: sendMessageTypeSegmentStart] Error forwarding segment start to AI agent", err)
		}
	}

}

func (s *WebSocketServerCallbacks) sendMessageTypeSegmentEnd(ctx context.Context, client *Client, payload []byte) {
	s.log.InfoWithID(ctx, "[WebSocketServer: sendMessageTypeSegmentEnd] Called")

	var m msgSegmentEnd
	if json.Unmarshal(payload, &m) != nil || m.SegmentID == "" {
		_ = s.writeJSON(client, msgError{
			Type:    "error",
			Code:    string(app_error.ErrCodeWebSocketInvalidSegmentEnd),
			Message: app_error.ErrCodeWebSocketInvalidSegmentEnd.Message(),
		})
		return
	}

	if client.sessionID != m.SessionID {
		s.log.ErrorWithID(ctx, "Security violation: Session ID mismatch", map[string]any{
			"expected_session_id": client.sessionID,
			"received_session_id": m.SessionID,
			"user_id":             client.userID,
		})
		return
	}

	if client.currentSegmentID != m.SegmentID {
		s.log.ErrorWithID(ctx, "Security violation: Segment ID mismatch", map[string]any{
			"expected_segment_id": client.currentSegmentID,
			"received_segment_id": m.SegmentID,
			"session_id":          client.sessionID,
			"user_id":             client.userID,
		})
		return
	}

	client.currentSegmentID = ""

	if agentClient, exists := s.clientManager.GetClientBySessionID(ctx, client.sessionID); exists {
		if err := agentClient.SegmentEnd(ctx, m.SegmentID); err != nil {
			s.log.ErrorWithID(ctx, "[WebSocketServer: sendMessageTypeSegmentEnd] Error forwarding segment end to AI agent", err)
		}
	}
}

func (s *WebSocketServerCallbacks) sendMessageTypeError(ctx context.Context, client *Client, payload []byte) {
	s.log.InfoWithID(ctx, "[WebSocketServer: sendMessageTypeError] Called")

	_ = s.writeJSON(client, msgError{
		Type:    "error",
		Code:    string(app_error.ErrCodeWebSocketInvalidMessage),
		Message: app_error.ErrCodeWebSocketInvalidMessage.Message(),
	})
	_ = s.writeJSON(client, map[string]any{"v": 1, "type": "echo", "content": json.RawMessage(payload)})
}
