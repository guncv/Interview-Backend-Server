package websocket

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	"gitlab.com/interview-simulation/interview-backend-server/internal/entities"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/database"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/queue/publisher"
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
	publisher               publisher.RedisTaskPublisher
}

func NewWebSocketServerLogic(
	log *log.Logger,
	disconnect func(ctx context.Context, client *Client),
	writeJSON func(ctx context.Context, client *Client, data any),
	clientManager *ClientManager,
	interviewSessionService services.InterviewSessionService,
	redisClient database.RedisClient,
	generator utils.Generator,
	publisher publisher.RedisTaskPublisher,
) *WebSocketServerLogic {

	return &WebSocketServerLogic{
		log:                     log,
		disconnect:              disconnect,
		writeJSON:               writeJSON,
		clientManager:           clientManager,
		interviewSessionService: interviewSessionService,
		redisClient:             redisClient,
		generator:               generator,
		publisher:               publisher,
	}
}

func (s *WebSocketServerLogic) sendStartSessionConversationMessage(ctx context.Context, client *Client) {
	s.log.InfoWithID(ctx, "[WebSocketServer: sendMessageTypeInterviewerResponse] Called")

	go func() {
		time.Sleep(2 * time.Second)

		openingMsg := MsgStartSessionConversation{
			Type:      constants.WebSocketMessageTypeStartSessionConversation,
			SessionID: client.SessionID,
		}

		if agentClient, exists := s.clientManager.GetClientBySessionID(ctx, client.SessionID); exists {
			if err := agentClient.StartSessionConversation(ctx, openingMsg); err != nil {
				s.log.ErrorWithID(ctx, "[WebSocketServer: sendMessageTypeInterviewerResponse] Error forwarding audio to AI agent", err)
				s.sendMessageTypeError(ctx, client, app_error.ErrCodeWebSocketInvalidMessage)
			}
		}
	}()
}

func (s *WebSocketServerLogic) handleUserAudioBinaryMessage(ctx context.Context, client *Client, payload []byte) {
	s.log.InfoWithID(ctx, "[WebSocketServer: handleUserAudioBinaryMessage] Called")

	if len(payload) < 4 {
		s.log.ErrorWithID(ctx, "[WebSocketServer: handleUserAudioBinaryMessage] Invalid frame: too short")
		return
	}

	headerLength := binary.BigEndian.Uint32(payload[:4])

	if int(headerLength)+4 > len(payload) {
		s.log.ErrorWithID(ctx, "[WebSocketServer: handleUserAudioBinaryMessage] Invalid frame: header length too large")
		return
	}

	headerBytes := payload[4 : 4+headerLength]

	var header MsgUserAudioChunk
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		s.log.ErrorWithID(ctx, "[WebSocketServer: handleUserAudioBinaryMessage] Invalid header JSON", err)
		s.sendMessageTypeError(ctx, client, app_error.ErrCodeWebSocketInvalidMessage)
		return
	}

	if header.SessionID != client.SessionID {
		s.log.ErrorWithID(ctx, "[WebSocketServer: handleUserAudioBinaryMessage] Security violation: Session ID mismatch")
		s.sendMessageTypeError(ctx, client, app_error.ErrCodeWebSocketInvalidSessionID)
		return
	}

	s.log.InfoWithID(ctx, "[WebSocketServer: handleUserAudioBinaryMessage] Segment ID", map[string]any{
		"segment_id": header.SegmentID,
	})
	segmentMapping, err := s.getSegmentMapping(ctx, header.SegmentID)
	if err != nil {
		s.log.ErrorWithID(ctx, "[WebSocketServer: handleUserAudioBinaryMessage] Error getting segment mapping", err)
		s.sendMessageTypeError(ctx, client, app_error.ErrCodeWebSocketInvalidSegmentID)
		return
	}

	header.SegmentID = segmentMapping
	if client.CurrentSegmentID != segmentMapping {
		s.log.ErrorWithID(ctx, "[WebSocketServer: handleUserAudioBinaryMessage] Security violation: Segment ID mismatch")
		s.sendMessageTypeError(ctx, client, app_error.ErrCodeWebSocketInvalidSegmentID)
		return
	}

	audioData := payload[4+headerLength:]

	if agentClient, exists := s.clientManager.GetClientBySessionID(ctx, client.SessionID); exists {
		if err := agentClient.SendUserAudio(ctx, header, audioData); err != nil {
			s.log.ErrorWithID(ctx, "[WebSocketServer: handleUserAudioBinaryMessage] Error forwarding audio to AI agent", err)
			s.sendMessageTypeError(ctx, client, app_error.ErrCodeWebSocketInvalidMessage)
		}
	}
}

func (s *WebSocketServerLogic) SendMessageTypeSegmentStart(ctx context.Context, client *Client, payload []byte) {
	s.log.InfoWithID(ctx, "[WebSocketServer: sendMessageTypeSegmentStart] Called")

	var m MsgSegmentStart
	if json.Unmarshal(payload, &m) != nil || m.SegmentID == "" {
		s.log.ErrorWithID(ctx, "[WebSocketServer: sendMessageTypeSegmentStart] Invalid segment start message")
		s.sendMessageTypeError(ctx, client, app_error.ErrCodeWebSocketInvalidSegmentStart)
		return
	}

	if client.SessionID != m.SessionID {
		s.log.ErrorWithID(ctx, "[WebSocketServer: sendMessageTypeSegmentStart] Security violation: Session ID mismatch")
		s.sendMessageTypeError(ctx, client, app_error.ErrCodeWebSocketInvalidSessionID)
		return
	}

	segmentMapping := s.generator.GenerateUUID(ctx).String()

	client.PreviousSegmentID = client.CurrentSegmentID
	client.PreviousSegmentExpiredAt = time.Now().Add(constants.WebSocketPreviousSegmentExpiredDuration)

	client.CurrentSegmentID = segmentMapping
	if err := s.createSegmentMapping(ctx, m.SegmentID, segmentMapping); err != nil {
		s.log.ErrorWithID(ctx, "[WebSocketServer: sendMessageTypeSegmentStart] Error creating segment mapping", err)
		s.sendMessageTypeError(ctx, client, app_error.ErrCodeWebSocketInvalidSegmentID)
		return
	}

	startTime := time.Since(client.StartSessionTime)
	m.SegmentID = segmentMapping
	setStartTimeReq := &entities.SetSessionStartTimeReq{
		SessionID: client.SessionID,
		StartedAt: utils.FormatSecondsToMMSS(startTime.Seconds()),
	}

	if err := s.interviewSessionService.SetSessionStartTime(ctx, setStartTimeReq); err != nil {
		s.log.ErrorWithID(ctx, "[WebSocketServer: sendMessageTypeSegmentStart] Error setting session start time", err)
		s.sendMessageTypeError(ctx, client, app_error.ErrCodeWebSocketInvalidMessage)
		return
	}

	if agentClient, exists := s.clientManager.GetClientBySessionID(ctx, client.SessionID); exists {
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

	if client.SessionID != m.SessionID {
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
	if client.CurrentSegmentID != segmentMapping {
		s.log.ErrorWithID(ctx, "[WebSocketServer: sendMessageTypeSegmentEnd] Security violation: Segment ID mismatch")
		s.sendMessageTypeError(ctx, client, app_error.ErrCodeWebSocketInvalidSegmentID)
		return
	}

	endedTime := time.Since(client.StartSessionTime)
	setEndTimeReq := &entities.SetSessionEndTimeReq{
		SessionID: client.SessionID,
		EndedAt:   utils.FormatSecondsToMMSS(endedTime.Seconds()),
	}

	if err := s.interviewSessionService.SetSessionEndTime(ctx, setEndTimeReq); err != nil {
		s.log.ErrorWithID(ctx, "[WebSocketServer: sendMessageTypeSegmentEnd] Error setting session end time", err)
		s.sendMessageTypeError(ctx, client, app_error.ErrCodeWebSocketInvalidMessage)
		return
	}

	if agentClient, exists := s.clientManager.GetClientBySessionID(ctx, client.SessionID); exists {
		if err := agentClient.SegmentEnd(ctx, m); err != nil {
			s.log.ErrorWithID(ctx, "[WebSocketServer: sendMessageTypeSegmentEnd] Error forwarding segment end to AI agent", err)
			s.sendMessageTypeError(ctx, client, app_error.ErrCodeWebSocketInvalidMessage)
		}
	}

	redisKey := fmt.Sprintf("%s%s", constants.RedisPrefixInterviewLastMessage, client.SessionID)
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

	if client.SessionID != req.SessionID {
		s.log.ErrorWithID(ctx, "[WebSocketServer: sendMessageTypeUserPartialTranscript] Security violation: Session ID mismatch")
		s.sendMessageTypeError(ctx, client, app_error.ErrCodeWebSocketInvalidMessage)
		return
	}

	if client.CurrentSegmentID != req.SegmentID {
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

	if client.SessionID != req.SessionID {
		s.log.ErrorWithID(ctx, "[WebSocketServer: sendMessageTypeUserFullTranscript] Security violation: Session ID mismatch")
		s.sendMessageTypeError(ctx, client, app_error.ErrCodeWebSocketInvalidSessionID)
		return
	}

	if client.CurrentSegmentID != req.SegmentID {
		s.log.ErrorWithID(ctx, "[WebSocketServer: sendMessageTypeUserFullTranscript] Security violation: Segment ID mismatch")
		s.sendMessageTypeError(ctx, client, app_error.ErrCodeWebSocketInvalidSegmentID)
		return
	}

	getInterviewLastMessageReq := &entities.GetInterviewerLastMessageReq{
		SessionID: req.SessionID,
	}

	lastMessage, err := s.interviewSessionService.GetInterviewerLastMessage(ctx, getInterviewLastMessageReq)
	if err != nil {
		s.log.ErrorWithID(ctx, "[WebSocketServer: sendMessageTypeUserFullTranscript] Error getting last message", err)
		s.sendMessageTypeError(ctx, client, app_error.ErrCodeGeneralRedisGetFailed)
		return
	}

	createSessionTurnReq := &entities.CreateUserSessionTurnBySessionIDReq{
		TurnID:       req.SegmentID,
		SessionID:    req.SessionID,
		CurrentState: lastMessage.CurrentState,
		Transcript:   req.Transcript,
	}

	if err := s.interviewSessionService.CreateUserSessionTurnBySessionID(ctx, createSessionTurnReq); err != nil {
		s.log.ErrorWithID(ctx, "[WebSocketServer: sendMessageTypeUserFullTranscript] Error creating session turn", err)
		s.sendMessageTypeError(ctx, client, app_error.ErrCodeWebSocketInvalidMessage)
		return
	}

	calculateTurnScoreReq := &entities.CalculateTurnScoreReq{
		SessionID:          client.SessionID,
		UserTurnID:         req.SegmentID,
		UserID:             client.userID,
		UserMessage:        req.Transcript,
		InterviewerMessage: lastMessage.Message,
		CurrentState:       lastMessage.CurrentState,
	}

	if err := s.publisher.PublishTaskCalculateTurnScore(context.Background(), calculateTurnScoreReq); err != nil {
		s.log.ErrorWithID(ctx, "[WebSocketServer: sendMessageTypeUserFullTranscript] Error publishing task calculate turn score", err)
		s.sendMessageTypeError(ctx, client, app_error.ErrCodeWebSocketInvalidMessage)
		return
	}
}

func (s *WebSocketServerLogic) sendMessageTypeInterviewerResp(ctx context.Context, client *Client, req MsgInterviewerResp) {
	s.log.InfoWithID(ctx, "[WebSocketServer: sendMessageTypeInterviewerResp] Called: ", req)

	if client.SessionID != req.SessionID {
		s.log.ErrorWithID(ctx, "[WebSocketServer: sendMessageTypeInterviewerResp] Security violation: Session ID mismatch")
		s.sendMessageTypeError(ctx, client, app_error.ErrCodeWebSocketInvalidSessionID)
		return
	}

	turnID := s.generator.GenerateUUID(ctx).String()
	startedAt, err := utils.ParseAndFormatDurationSince(req.StartedAt, client.StartSessionTime)
	if err != nil {
		s.log.ErrorWithID(ctx, "[WebSocketServer: sendMessageTypeInterviewerResp] Error parsing started at", err)
		s.sendMessageTypeError(ctx, client, app_error.ErrCodeWebSocketInvalidMessage)
		return
	}

	endedAt, err := utils.ParseAndFormatDurationSince(req.EndedAt, client.StartSessionTime)
	if err != nil {
		s.log.ErrorWithID(ctx, "[WebSocketServer: sendMessageTypeInterviewerResp] Error parsing ended at", err)
		s.sendMessageTypeError(ctx, client, app_error.ErrCodeWebSocketInvalidMessage)
		return
	}

	createSessionTurnReq := &entities.CreateInterviewerSessionTurnBySessionIDReq{
		TurnID:       turnID,
		SessionID:    req.SessionID,
		Transcript:   req.Message,
		StartedAt:    startedAt,
		EndedAt:      endedAt,
		CurrentState: req.CurrentState,
	}

	if err := s.interviewSessionService.CreateInterviewerSessionTurnBySessionID(ctx, createSessionTurnReq); err != nil {
		s.log.ErrorWithID(ctx, "[WebSocketServer: sendMessageTypeInterviewerResp] Error creating session turn", err)
		s.sendMessageTypeError(ctx, client, app_error.ErrCodeWebSocketInvalidMessage)
		return
	}

	redisKey := fmt.Sprintf("%s%s", constants.RedisPrefixInterviewLastMessage, req.SessionID)

	lastMessagePayload := entities.RedisLastMessagePayload{
		Message:      req.Message,
		CurrentState: req.CurrentState,
	}

	payloadBytes, err := json.Marshal(lastMessagePayload)
	if err != nil {
		s.log.ErrorWithID(ctx, "[WebSocketServer: sendMessageTypeInterviewerResp] Error marshaling last message payload", err)
		s.sendMessageTypeError(ctx, client, app_error.ErrCodeGeneralRedisSetFailed)
		return
	}

	redisPayload := database.RedisPayload{
		Key:   redisKey,
		Value: string(payloadBytes),
		TTL:   constants.RedisTTLInterviewLastMessage,
	}
	if err := s.redisClient.Set(context.Background(), redisPayload); err != nil {
		s.log.ErrorWithID(ctx, "[WebSocketServer: sendMessageTypeInterviewerResp] Error creating last message", err)
		s.sendMessageTypeError(ctx, client, app_error.ErrCodeGeneralRedisSetFailed)
		return
	} else {
		s.log.InfoWithID(ctx, "[WebSocketServer: sendMessageTypeInterviewerResp] Last message created")
	}

	s.writeJSON(ctx, client, map[string]interface{}{
		"type":          req.Type,
		"session_id":    req.SessionID,
		"message":       req.Message,
		"started_at":    req.StartedAt,
		"ended_at":      req.EndedAt,
		"current_state": req.CurrentState,
	})
}

func (s *WebSocketServerLogic) sendMessageTypeInterviewerAudioChunk(
	ctx context.Context,
	client *Client,
	req MsgInterviewerAudioChunk,
	audioData []byte,
) {
	s.log.InfoWithID(ctx, "[WebSocketServer: sendMessageTypeInterviewerAudioChunk] Called")

	if client.SessionID != req.SessionID {
		s.log.ErrorWithID(ctx, "[WebSocketServer: sendMessageTypeInterviewerAudioChunk] Security violation: Session ID mismatch")
		s.sendMessageTypeError(ctx, client, app_error.ErrCodeWebSocketInvalidSessionID)
		return
	}

	header := map[string]interface{}{
		"type":       req.Type,
		"session_id": req.SessionID,
	}

	headerBytes, err := json.Marshal(header)
	if err != nil {
		s.log.ErrorWithID(ctx, "[WebSocketServer: sendMessageTypeInterviewerAudioChunk] Error marshaling header", err)
		return
	}

	headerLength := uint32(len(headerBytes))

	payload := make([]byte, 4+len(headerBytes)+len(audioData))
	binary.BigEndian.PutUint32(payload[:4], headerLength)
	copy(payload[4:4+headerLength], headerBytes)
	copy(payload[4+headerLength:], audioData)

	if err := client.conn.WriteMessage(websocket.BinaryMessage, payload); err != nil {
		s.log.ErrorWithID(ctx, "[WebSocketServer: sendMessageTypeInterviewerAudioChunk] Error sending binary audio", err)
		s.sendMessageTypeError(ctx, client, app_error.ErrCodeWebSocketInvalidMessage)
		return
	}

	s.log.InfoWithID(ctx, "[WebSocketServer: sendMessageTypeInterviewerAudioChunk] Audio chunk sent", map[string]any{
		"session_id": req.SessionID,
		"audio_size": len(audioData),
	})
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
