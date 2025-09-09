package websocket

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"

	"github.com/redis/go-redis/v9"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	"gitlab.com/interview-simulation/interview-backend-server/internal/entities"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/database"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
	"gitlab.com/interview-simulation/interview-backend-server/internal/services"
	"gitlab.com/interview-simulation/interview-backend-server/internal/utils"
)

type WebSocketServerLogic struct {
	log                     *log.Logger
	disconnect              func(ctx context.Context, client *Client)
	writeJSON               func(ctx context.Context, client *Client, data any)
	clientManager           *ClientManager
	interviewSessionService services.InterviewSessionService
	redisClient             database.RedisClient
	generator               utils.Generator
}

func NewWebSocketServerLogic(
	log *log.Logger,
	disconnect func(ctx context.Context, client *Client),
	writeJSON func(ctx context.Context, client *Client, data any),
	clientManager *ClientManager,
	interviewSessionService services.InterviewSessionService,
	redisClient database.RedisClient,
	generator utils.Generator,
) *WebSocketServerLogic {

	return &WebSocketServerLogic{
		log:                     log,
		disconnect:              disconnect,
		writeJSON:               writeJSON,
		clientManager:           clientManager,
		interviewSessionService: interviewSessionService,
		redisClient:             redisClient,
		generator:               generator,
	}
}

func (s *WebSocketServerLogic) handleAudioBinaryMessage(ctx context.Context, client *Client, payload []byte) {
	s.log.InfoWithID(ctx, "[WebSocketServer: handleAudioBinaryMessage] Called")

	if len(payload) < 4 {
		s.log.ErrorWithID(ctx, "[WebSocketServer: handleAudioBinaryMessage] Invalid frame: too short")
		return
	}

	headerLength := binary.BigEndian.Uint32(payload[:4])

	if int(headerLength)+4 > len(payload) {
		s.log.ErrorWithID(ctx, "[WebSocketServer: handleAudioBinaryMessage] Invalid frame: header length too large")
		return
	}

	headerBytes := payload[4 : 4+headerLength]

	var header MsgAudioChunk
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		s.log.ErrorWithID(ctx, "[WebSocketServer: handleAudioBinaryMessage] Invalid header JSON", err)
		s.sendMessageTypeError(ctx, client, app_error.ErrCodeWebSocketInvalidMessage)
		return
	}

	if header.SessionID != client.sessionID {
		s.log.ErrorWithID(ctx, "[WebSocketServer: handleAudioBinaryMessage] Security violation: Session ID mismatch")
		s.sendMessageTypeError(ctx, client, app_error.ErrCodeWebSocketInvalidSessionID)
		return
	}

	s.log.InfoWithID(ctx, "[WebSocketServer: handleAudioBinaryMessage] Segment ID", map[string]any{
		"segment_id": header.SegmentID,
	})
	segmentMapping, err := s.getSegmentMapping(ctx, header.SegmentID)
	if err != nil {
		s.log.ErrorWithID(ctx, "[WebSocketServer: handleAudioBinaryMessage] Error getting segment mapping", err)
		s.sendMessageTypeError(ctx, client, app_error.ErrCodeWebSocketInvalidSegmentID)
		return
	}

	header.SegmentID = segmentMapping
	if client.currentSegmentID != segmentMapping {
		s.log.ErrorWithID(ctx, "[WebSocketServer: handleAudioBinaryMessage] Security violation: Segment ID mismatch")
		s.sendMessageTypeError(ctx, client, app_error.ErrCodeWebSocketInvalidSegmentID)
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
		s.log.ErrorWithID(ctx, "[WebSocketServer: sendMessageTypeSegmentStart] Invalid segment start message")
		s.sendMessageTypeError(ctx, client, app_error.ErrCodeWebSocketInvalidSegmentStart)
		return
	}

	if client.sessionID != m.SessionID {
		s.log.ErrorWithID(ctx, "[WebSocketServer: sendMessageTypeSegmentStart] Security violation: Session ID mismatch")
		s.sendMessageTypeError(ctx, client, app_error.ErrCodeWebSocketInvalidSessionID)
		return
	}

	segmentMapping := s.generator.GenerateUUID(ctx).String()

	client.currentSegmentID = segmentMapping
	if err := s.createSegmentMapping(ctx, m.SegmentID, segmentMapping); err != nil {
		s.log.ErrorWithID(ctx, "[WebSocketServer: sendMessageTypeSegmentStart] Error creating segment mapping", err)
		s.sendMessageTypeError(ctx, client, app_error.ErrCodeWebSocketInvalidSegmentID)
		return
	}

	m.SegmentID = segmentMapping
	setStartTimeReq := &entities.SetSessionStartTimeReq{
		SessionID: client.sessionID,
		StartedAt: m.StartedAt,
	}

	if err := s.interviewSessionService.SetSessionStartTime(ctx, setStartTimeReq); err != nil {
		s.log.ErrorWithID(ctx, "[WebSocketServer: sendMessageTypeSegmentStart] Error setting session start time", err)
		s.sendMessageTypeError(ctx, client, app_error.ErrCodeWebSocketInvalidMessage)
		return
	}

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
		s.log.ErrorWithID(ctx, "[WebSocketServer: sendMessageTypeSegmentEnd] Invalid segment end message")
		s.sendMessageTypeError(ctx, client, app_error.ErrCodeWebSocketInvalidSegmentEnd)
		return
	}

	if client.sessionID != m.SessionID {
		s.log.ErrorWithID(ctx, "[WebSocketServer: sendMessageTypeSegmentEnd] Security violation: Session ID mismatch")
		s.sendMessageTypeError(ctx, client, app_error.ErrCodeWebSocketInvalidSessionID)
		return
	}

	segmentMapping, err := s.getSegmentMapping(ctx, m.SegmentID)
	if err != nil {
		s.log.ErrorWithID(ctx, "[WebSocketServer: sendMessageTypeSegmentEnd] Error getting segment mapping", err)
		s.sendMessageTypeError(ctx, client, app_error.ErrCodeWebSocketInvalidSegmentID)
		return
	}

	m.SegmentID = segmentMapping
	if client.currentSegmentID != segmentMapping {
		s.log.ErrorWithID(ctx, "[WebSocketServer: sendMessageTypeSegmentEnd] Security violation: Segment ID mismatch")
		s.sendMessageTypeError(ctx, client, app_error.ErrCodeWebSocketInvalidSegmentID)
		return
	}

	setEndTimeReq := &entities.SetSessionEndTimeReq{
		SessionID: client.sessionID,
		EndedAt:   m.EndedAt,
	}

	if err := s.interviewSessionService.SetSessionEndTime(ctx, setEndTimeReq); err != nil {
		s.log.ErrorWithID(ctx, "[WebSocketServer: sendMessageTypeSegmentEnd] Error setting session end time", err)
		s.sendMessageTypeError(ctx, client, app_error.ErrCodeWebSocketInvalidMessage)
		return
	}

	if agentClient, exists := s.clientManager.GetClientBySessionID(ctx, client.sessionID); exists {
		if err := agentClient.SegmentEnd(ctx, m); err != nil {
			s.log.ErrorWithID(ctx, "[WebSocketServer: sendMessageTypeSegmentEnd] Error forwarding segment end to AI agent", err)
			s.sendMessageTypeError(ctx, client, app_error.ErrCodeWebSocketInvalidMessage)
		}
	}

	redisKey := fmt.Sprintf("%s%s", constants.RedisPrefixInterviewLastMessage, client.sessionID)
	lastMessage, err := s.redisClient.Get(context.Background(), redisKey)
	if err != nil {
		if err == redis.Nil {
			s.log.WarnWithID(ctx, "[WebSocketServer: sendMessageTypeSegmentEnd] Last message not found", err)
		} else {
			s.log.ErrorWithID(ctx, "[WebSocketServer: sendMessageTypeSegmentEnd] Error getting last message", err)
			s.sendMessageTypeError(ctx, client, app_error.ErrCodeGeneralRedisGetFailed)
			return
		}
	} else {
		s.log.InfoWithID(ctx, "[WebSocketServer: sendMessageTypeSegmentEnd] Last message retrieved", map[string]any{
			"last_message": lastMessage,
		})
	}
}

func (s *WebSocketServerLogic) sendMessageTypeUserPartialTranscript(ctx context.Context, client *Client, req MsgUserPartialTranscript) {
	s.log.InfoWithID(ctx, "[WebSocketServer: sendMessageTypeUserPartialTranscript] Called")

	if client.sessionID != req.SessionID {
		s.log.ErrorWithID(ctx, "[WebSocketServer: sendMessageTypeUserPartialTranscript] Security violation: Session ID mismatch")
		s.sendMessageTypeError(ctx, client, app_error.ErrCodeWebSocketInvalidMessage)
		return
	}

	if client.currentSegmentID != req.SegmentID {
		s.log.ErrorWithID(ctx, "[WebSocketServer: sendMessageTypeUserPartialTranscript] Security violation: Segment ID mismatch")
		s.sendMessageTypeError(ctx, client, app_error.ErrCodeWebSocketInvalidSegmentID)
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
		s.log.ErrorWithID(ctx, "[WebSocketServer: sendMessageTypeUserFullTranscript] Security violation: Session ID mismatch")
		s.sendMessageTypeError(ctx, client, app_error.ErrCodeWebSocketInvalidSessionID)
		return
	}

	if client.currentSegmentID != req.SegmentID {
		s.log.ErrorWithID(ctx, "[WebSocketServer: sendMessageTypeUserFullTranscript] Security violation: Segment ID mismatch")
		s.sendMessageTypeError(ctx, client, app_error.ErrCodeWebSocketInvalidSegmentID)
		return
	}

	createSessionTurnReq := &entities.CreateSessionTurnBySessionIDReq{
		TurnID:     req.SegmentID,
		SessionID:  req.SessionID,
		Actor:      req.Author,
		Transcript: req.Transcript,
	}

	if err := s.interviewSessionService.CreateSessionTurnBySessionID(ctx, createSessionTurnReq); err != nil {
		s.log.ErrorWithID(ctx, "[WebSocketServer: sendMessageTypeUserFullTranscript] Error creating session turn", err)
		s.sendMessageTypeError(ctx, client, app_error.ErrCodeWebSocketInvalidMessage)
		return
	}
}

func (s *WebSocketServerLogic) sendMessageTypeInterviewerResp(ctx context.Context, client *Client, req MsgAIResponse) {
	s.log.InfoWithID(ctx, "[WebSocketServer: sendMessageTypeInterviewerResp] Called")

	if client.sessionID != req.SessionID {
		s.log.ErrorWithID(ctx, "[WebSocketServer: sendMessageTypeInterviewerResp] Security violation: Session ID mismatch")
		s.sendMessageTypeError(ctx, client, app_error.ErrCodeWebSocketInvalidSessionID)
		return
	}

	turnID := s.generator.GenerateUUID(ctx).String()
	createSessionTurnReq := &entities.CreateSessionTurnBySessionIDReq{
		TurnID:     turnID,
		SessionID:  req.SessionID,
		Actor:      req.Author,
		Transcript: req.Message,
	}

	if err := s.interviewSessionService.CreateSessionTurnBySessionID(ctx, createSessionTurnReq); err != nil {
		s.log.ErrorWithID(ctx, "[WebSocketServer: sendMessageTypeInterviewerResp] Error creating session turn", err)
		s.sendMessageTypeError(ctx, client, app_error.ErrCodeWebSocketInvalidMessage)
		return
	}

	redisKey := fmt.Sprintf("%s%s", constants.RedisPrefixInterviewLastMessage, req.SessionID)
	redisPayload := database.RedisPayload{
		Key:   redisKey,
		Value: req.Message,
		TTL:   constants.RedisTTLInterviewLastMessage,
	}
	if err := s.redisClient.Set(context.Background(), redisPayload); err != nil {
		s.log.ErrorWithID(ctx, "[WebSocketServer: sendMessageTypeInterviewerResp] Error creating last message", err)
		s.sendMessageTypeError(ctx, client, app_error.ErrCodeGeneralRedisSetFailed)
		return
	}
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

func (s *WebSocketServerLogic) createSegmentMapping(ctx context.Context, fromSegmentID, toSegmentID string) error {
	s.log.InfoWithID(ctx, "[WebSocketServer: createSegmentMapping] Called")

	key := fmt.Sprintf("%s%s", constants.RedisPrefixInterviewSegmentMapping, fromSegmentID)

	if err := s.redisClient.Set(context.Background(), database.RedisPayload{
		Key:   key,
		Value: toSegmentID,
		TTL:   constants.RedisTTLInterviewSegmentMapping,
	}); err != nil {
		s.log.ErrorWithID(ctx, "[WebSocketServer: createSegmentMapping] Error creating segment mapping", err)
		return err
	}

	return nil
}

func (s *WebSocketServerLogic) getSegmentMapping(ctx context.Context, fromSegmentID string) (string, error) {
	s.log.InfoWithID(ctx, "[WebSocketServer: getSegmentMapping] Called")

	key := fmt.Sprintf("%s%s", constants.RedisPrefixInterviewSegmentMapping, fromSegmentID)

	value, err := s.redisClient.Get(context.Background(), key)
	if err != nil {
		s.log.ErrorWithID(ctx, "[WebSocketServer: getSegmentMapping] Error getting segment mapping", err)
		return "", err
	}

	return value, nil
}
