package websocket

import (
	"context"
	"encoding/binary"
	"encoding/json"

	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
)

type WebSocketServerLogic struct {
	log           *log.Logger
	disconnect    func(ctx context.Context, client *Client)
	writeJSON     func(ctx context.Context, client *Client, data any)
	clientManager *ClientManager
}

func NewWebSocketServerLogic(
	log *log.Logger,
	disconnect func(ctx context.Context, client *Client),
	writeJSON func(ctx context.Context, client *Client, data any),
	clientManager *ClientManager,
) *WebSocketServerLogic {

	return &WebSocketServerLogic{
		log:           log,
		disconnect:    disconnect,
		writeJSON:     writeJSON,
		clientManager: clientManager,
	}
}

func (s *WebSocketServerLogic) handleAudioBinaryMessage(ctx context.Context, client *Client, payload []byte) {
	s.log.InfoWithID(ctx, "[WebSocketServer: handleAudioBinaryMessage] Called")

	if len(payload) < 4 {
		s.log.ErrorWithID(ctx, "Invalid frame: too short")
		return
	}

	headerLength := binary.BigEndian.Uint32(payload[:4])

	if int(headerLength)+4 > len(payload) {
		s.log.ErrorWithID(ctx, "Invalid frame: header length too large")
		return
	}

	headerBytes := payload[4 : 4+headerLength]

	var header MsgAudioChunk
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		s.log.ErrorWithID(ctx, "Invalid header JSON", err)
		s.sendMessageTypeError(ctx, client, app_error.ErrCodeWebSocketInvalidMessage)
		return
	}

	if header.SessionID != client.sessionID {
		s.log.ErrorWithID(ctx, "Security violation: Session ID mismatch")
		s.sendMessageTypeError(ctx, client, app_error.ErrCodeWebSocketInvalidMessage)
		return
	}

	if header.SegmentID != client.currentSegmentID {
		s.log.ErrorWithID(ctx, "Security violation: Segment ID mismatch")
		s.sendMessageTypeError(ctx, client, app_error.ErrCodeWebSocketInvalidMessage)
		return
	}

	audioData := payload[4+headerLength:]

	if agentClient, exists := s.clientManager.GetClientBySessionID(ctx, client.sessionID); exists {
		if err := agentClient.SendAudio(ctx, header, audioData); err != nil {
			s.log.ErrorWithID(ctx, "[WebSocketServer: handleAudioBinaryMessage] Error forwarding audio to AI agent", err)
			s.sendMessageTypeError(ctx, client, app_error.ErrCodeWebSocketInvalidMessage)
		}
	}
}

func (s *WebSocketServerLogic) sendMessageTypeSegmentStart(ctx context.Context, client *Client, payload []byte) {
	s.log.InfoWithID(ctx, "[WebSocketServer: sendMessageTypeSegmentStart] Called")

	var m MsgSegmentStart
	if json.Unmarshal(payload, &m) != nil || m.SegmentID == "" {
		s.log.ErrorWithID(ctx, "Invalid segment start message")
		s.sendMessageTypeError(ctx, client, app_error.ErrCodeWebSocketInvalidSegmentStart)
		return
	}

	if client.sessionID != m.SessionID {
		s.log.ErrorWithID(ctx, "Security violation: Session ID mismatch")
		s.sendMessageTypeError(ctx, client, app_error.ErrCodeWebSocketInvalidMessage)
		return
	}

	client.currentSegmentID = m.SegmentID

	if agentClient, exists := s.clientManager.GetClientBySessionID(ctx, client.sessionID); exists {
		if err := agentClient.SegmentStart(ctx, m); err != nil {
			s.log.ErrorWithID(ctx, "[WebSocketServer: sendMessageTypeSegmentStart] Error forwarding segment start to AI agent", err)
			s.sendMessageTypeError(ctx, client, app_error.ErrCodeWebSocketInvalidMessage)
		}
	}

}

func (s *WebSocketServerLogic) sendMessageTypeSegmentEnd(ctx context.Context, client *Client, payload []byte) {
	s.log.InfoWithID(ctx, "[WebSocketServer: sendMessageTypeSegmentEnd] Called")

	var m MsgSegmentEnd
	if json.Unmarshal(payload, &m) != nil || m.SegmentID == "" {
		s.log.ErrorWithID(ctx, "Invalid segment end message")
		s.sendMessageTypeError(ctx, client, app_error.ErrCodeWebSocketInvalidSegmentEnd)
		return
	}

	if client.sessionID != m.SessionID {
		s.log.ErrorWithID(ctx, "[WebSocketServer: sendMessageTypeSegmentEnd] Security violation: Session ID mismatch")
		s.sendMessageTypeError(ctx, client, app_error.ErrCodeWebSocketInvalidMessage)
		return
	}

	if client.currentSegmentID != m.SegmentID {
		s.log.ErrorWithID(ctx, "[WebSocketServer: sendMessageTypeSegmentEnd] Security violation: Segment ID mismatch")
		s.sendMessageTypeError(ctx, client, app_error.ErrCodeWebSocketInvalidMessage)
		return
	}

	if agentClient, exists := s.clientManager.GetClientBySessionID(ctx, client.sessionID); exists {
		if err := agentClient.SegmentEnd(ctx, m); err != nil {
			s.log.ErrorWithID(ctx, "[WebSocketServer: sendMessageTypeSegmentEnd] Error forwarding segment end to AI agent", err)
			s.sendMessageTypeError(ctx, client, app_error.ErrCodeWebSocketInvalidMessage)
		}
	}
}

func (s *WebSocketServerLogic) sendMessageTypeUserPartialTranscript(ctx context.Context, client *Client, req MsgUserPartialTranscript) {
	s.log.InfoWithID(ctx, "[WebSocketServer: sendMessageTypeUserPartialTranscript] Called")

	if client.sessionID != req.SessionID {
		s.log.ErrorWithID(ctx, "Security violation: Session ID mismatch")
		s.sendMessageTypeError(ctx, client, app_error.ErrCodeWebSocketInvalidMessage)
		return
	}

	if client.currentSegmentID != req.SegmentID {
		s.log.ErrorWithID(ctx, "Security violation: Segment ID mismatch")
		s.sendMessageTypeError(ctx, client, app_error.ErrCodeWebSocketInvalidMessage)
		return
	}

	request := map[string]interface{}{
		"type":       req.Type,
		"author":     req.Author,
		"session_id": req.SessionID,
		"segment_id": req.SegmentID,
		"transcript": req.Transcript,
	}

	s.writeJSON(ctx, client, request)
}

func (s *WebSocketServerLogic) sendMessageTypeUserFullTranscript(ctx context.Context, client *Client, req MsgUserFullTranscript) {
	s.log.InfoWithID(ctx, "[WebSocketServer: sendMessageTypeUserFullTranscript] Called")

	if client.sessionID != req.SessionID {
		s.log.ErrorWithID(ctx, "Security violation: Session ID mismatch")
		s.sendMessageTypeError(ctx, client, app_error.ErrCodeWebSocketInvalidMessage)
		return
	}

	if client.currentSegmentID != req.SegmentID {
		s.log.ErrorWithID(ctx, "Security violation: Segment ID mismatch")
		s.sendMessageTypeError(ctx, client, app_error.ErrCodeWebSocketInvalidMessage)
		return
	}

	request := map[string]interface{}{
		"type":       req.Type,
		"author":     req.Author,
		"session_id": req.SessionID,
		"segment_id": req.SegmentID,
		"transcript": req.Transcript,
	}

	s.writeJSON(ctx, client, request)
}

func (s *WebSocketServerLogic) sendMessageTypeError(ctx context.Context, client *Client, errCode app_error.ErrorCode) {
	s.log.InfoWithID(ctx, "[WebSocketServer: sendMessageTypeError] Called")

	s.writeJSON(ctx, client, msgError{
		Type:    constants.WebSocketMessageTypeError,
		Code:    string(errCode),
		Message: errCode.Message(),
	})

	s.disconnect(ctx, client)
}
