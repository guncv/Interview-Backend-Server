package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"mime/multipart"
	"strconv"
	"time"

	"github.com/google/uuid"
	"gitlab.com/interview-simulation/interview-backend-server/internal/config"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	db "gitlab.com/interview-simulation/interview-backend-server/internal/db/sqlc"
	"gitlab.com/interview-simulation/interview-backend-server/internal/entities"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/aws"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/database"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/queue/publisher"
	"gitlab.com/interview-simulation/interview-backend-server/internal/middleware"
	"gitlab.com/interview-simulation/interview-backend-server/internal/repositories"
	"gitlab.com/interview-simulation/interview-backend-server/internal/utils"
)

type InterviewSessionService interface {
	CreateInterviewSessionWithNewResume(ctx context.Context, req *entities.CreateInterviewSessionWithNewResumeRequest) (*entities.CreateInterviewSessionWithNewResumeResponse, error)
	CreateInterviewSessionWithExistingResume(ctx context.Context, req *entities.CreateInterviewSessionWithExistingResumeReq) (*entities.CreateInterviewSessionWithExistingResumeResp, error)
	CreateUserSessionTurnBySessionID(ctx context.Context, req *entities.CreateUserSessionTurnBySessionIDReq) error
	GetSessionStartedAtAndEndedAt(ctx context.Context, sessionID string) (map[string]string, error)
	CreateInterviewerSessionTurnBySessionID(ctx context.Context, req *entities.CreateInterviewerSessionTurnBySessionIDReq) error
	UpdateInterviewSessionStatus(ctx context.Context, req *entities.UpdateInterviewSessionStatusReq) error
	SetSessionStartTime(ctx context.Context, req *entities.SetSessionStartTimeReq) error
	SetSessionEndTime(ctx context.Context, req *entities.SetSessionEndTimeReq) error
	IsSessionValid(ctx context.Context, req *entities.IsSessionValidReq) (*entities.IsSessionValidResp, error)
	GetInterviewerLastMessage(ctx context.Context, req *entities.GetInterviewerLastMessageReq) (*entities.GetInterviewerLastMessageResp, error)
	GetChatHistoryBySessionToken(ctx context.Context, req *entities.GetChatHistoryBySessionTokenReq) (*entities.GetChatHistoryBySessionTokenResp, error)
	CheckExistsAndInitStartedAtInterviewSession(ctx context.Context, sessionId string) (*entities.CheckExistsAndInitStartedAtInterviewSessionResp, error)
	StartConversationBySessionID(ctx context.Context, sessionId string) error
	EndInterviewSession(ctx context.Context, req *entities.EndInterviewSessionReq) error
	ListInterviewSessionsByUserIDWithCursor(ctx context.Context, req *entities.ListInterviewSessionsByUserIDWithCursorReq) (*entities.ListInterviewSessionsByUserIDResp, error)
	ListInterviewSessionsByUserIDWithJumpPagination(ctx context.Context, req *entities.ListInterviewSessionsByUserIDWithJumpPaginationReq) (*entities.ListInterviewSessionsByUserIDResp, error)
	DeleteUserInterviewSessionByID(ctx context.Context, sessionIDReq string) error
	GetInterviewSessionInformationByID(ctx context.Context, sessionIDReq string) (*entities.GetInterviewSessionInformationResp, error)
	GetChatHistoryBySessionIDWithEvaluation(ctx context.Context, req *entities.GetChatHistoryBySessionIDWithEvaluationReq) (*entities.GetChatHistoryBySessionIDWithEvaluationResp, error)
	InitialFirstCurrentStateSession(ctx context.Context, req *entities.InitialFirstCurrentStateSessionReq) (*entities.InitialFirstCurrentStateSessionResp, error)
	UpdateCurrentStateSession(ctx context.Context, req *entities.UpdateCurrentStateSessionReq) (*entities.UpdateCurrentStateSessionResp, error)
}

type interviewSessionService struct {
	log                  *log.Logger
	authContext          middleware.AuthContext
	resumeRepo           repositories.ResumeReposity
	generator            utils.Generator
	interviewSessionRepo repositories.InterviewSessionRepository
	jwtMaker             utils.JwtToken
	config               *config.Config
	s3Storage            aws.S3Storage
	publisher            publisher.RedisTaskPublisher
	redisClient          database.RedisClient
	evaluationService    EvaluationService
	evaluationScoresRepo repositories.EvaluationScoresRepository
	interviewTurnsRepo   repositories.InterviewTurnsRepository
	interviewStateRepo   repositories.InterviewStateRepository
}

func NewInterviewSessionService(
	log *log.Logger,
	authContext middleware.AuthContext,
	resumeRepo repositories.ResumeReposity,
	generator utils.Generator,
	interviewSessionRepo repositories.InterviewSessionRepository,
	jwtMaker utils.JwtToken,
	config *config.Config,
	s3Storage aws.S3Storage,
	publisher publisher.RedisTaskPublisher,
	redisClient database.RedisClient,
	evaluationService EvaluationService,
	evaluationScoresRepo repositories.EvaluationScoresRepository,
	interviewTurnsRepo repositories.InterviewTurnsRepository,
	interviewStateRepo repositories.InterviewStateRepository,
) InterviewSessionService {
	return &interviewSessionService{
		log:                  log,
		authContext:          authContext,
		resumeRepo:           resumeRepo,
		generator:            generator,
		interviewSessionRepo: interviewSessionRepo,
		jwtMaker:             jwtMaker,
		config:               config,
		s3Storage:            s3Storage,
		publisher:            publisher,
		redisClient:          redisClient,
		evaluationService:    evaluationService,
		evaluationScoresRepo: evaluationScoresRepo,
		interviewTurnsRepo:   interviewTurnsRepo,
		interviewStateRepo:   interviewStateRepo,
	}
}

func (s *interviewSessionService) CreateInterviewSessionWithNewResume(
	ctx context.Context,
	req *entities.CreateInterviewSessionWithNewResumeRequest,
) (*entities.CreateInterviewSessionWithNewResumeResponse, error) {
	s.log.InfoWithID(ctx, "[Service: CreateInterviewSessionWithNewResume] Called")

	authCtx, err := s.authContext.GetAuthContext(ctx)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: CreateInterviewSessionWithNewResume] Error getting auth context", err)
		return nil, err
	}

	customFileHeader := s.convertToCustomFileHeader(req.File)
	sessionID := s.generator.GenerateUUID(ctx)
	resumeID := s.generator.GenerateUUID(ctx)

	extractResumeJsonForRAGReq := &repositories.ExtractResumeJsonForRAGReq{
		SessionID:  sessionID.String(),
		UserID:     authCtx.Payload.UserID,
		ResumeID:   resumeID.String(),
		ResumeFile: customFileHeader,
	}

	err = s.resumeRepo.ExtractResumeJsonForRAG(ctx, extractResumeJsonForRAGReq)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: CreateInterviewSessionWithNewResume] Error extracting resume json for RAG", err)
		return nil, err
	}

	userID, err := uuid.Parse(authCtx.Payload.UserID)
	if err != nil {
		err := app_error.New(err, app_error.ErrCodeGeneralInvalidUUID)
		s.log.ErrorWithID(ctx, "[Service: CreateInterviewSessionWithNewResume] Error parsing user ID", err)
		return nil, err
	}

	allUserResumes, err := s.resumeRepo.ListAllResumesFileNameByUserID(ctx, userID)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: CreateInterviewSessionWithNewResume] Error getting all user resumes", err)
		return nil, err
	}

	uniqueFilename := utils.GenerateUniqueFilename(req.File.Filename, allUserResumes)

	key, err := s.s3Storage.UploadFile(ctx, req.File, constants.S3ResumeKey, authCtx.Payload.UserID)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: CreateInterviewSessionWithNewResume] Error uploading resume file", err)
		return nil, err
	}

	isDefaultResume, err := s.resumeRepo.CheckIsDefaultResumeExistsByUserID(ctx, userID)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: CreateInterviewSessionWithNewResume] Error checking if default resume exists", err)
		return nil, err
	}

	createResumeAndJobRequirementReq := &repositories.CreateInterviewSessionTxReq{
		ResumeID:   resumeID,
		UserID:     userID,
		FileName:   uniqueFilename,
		StorageKey: key,
		MimeType:   req.File.Header.Get("Content-Type"),
		ByteSize:   int32(req.File.Size),
		IsDefault:  !isDefaultResume,

		SessionID: sessionID,
		Position:  req.Position,
		Status:    constants.StatusPending,
		Modality:  constants.ModalityVoiceChat,
		IsConsent: req.IsConsent,
	}

	if err := s.interviewSessionRepo.CreateInterviewSessionWithNewResumeTx(ctx, createResumeAndJobRequirementReq); err != nil {
		s.log.ErrorWithID(ctx, "[Service: CreateInterviewSessionWithNewResume] Error creating interview session with new resume", err)

		deleteFilePayload := &aws.DeleteFilePayload{
			Key: createResumeAndJobRequirementReq.StorageKey,
		}

		if err := s.publisher.PublishTaskDeleteFile(ctx, deleteFilePayload); err != nil {
			s.log.WarnWithID(ctx, "[Service: CreateInterviewSessionWithNewResume] Error deleting resume file", err)
		}
		return nil, err
	}

	tokenReq := map[string]any{
		"session_id": sessionID,
		"user_id":    authCtx.Payload.UserID,
		"resume_id":  resumeID.String(),
		"role":       authCtx.Payload.Role,
	}

	tokenReqJSON, err := json.Marshal(tokenReq)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: CreateInterviewSessionWithNewResume] Error marshaling token request to JSON", err)
		return nil, err
	}

	sessionToken := s.generator.GenerateUUID(ctx).String()
	redisKey := fmt.Sprintf("%s%s", constants.RedisPrefixInterviewSessionToken, sessionToken)

	redisPayload := database.RedisPayload{
		Key:   redisKey,
		Value: string(tokenReqJSON),
		TTL:   s.config.InterviewSessionConfig.InterviewSessionDuration,
	}

	err = s.redisClient.Set(ctx, redisPayload)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: CreateInterviewSessionWithNewResume] Error creating interview session token", err)
		return nil, err
	}

	resp := &entities.CreateInterviewSessionWithNewResumeResponse{
		SessionToken: sessionToken,
	}

	return resp, nil
}

func (s *interviewSessionService) CreateInterviewSessionWithExistingResume(
	ctx context.Context,
	req *entities.CreateInterviewSessionWithExistingResumeReq,
) (*entities.CreateInterviewSessionWithExistingResumeResp, error) {
	s.log.InfoWithID(ctx, "[Service: CreateInterviewSessionWithExistingResume] Called")

	authCtx, err := s.authContext.GetAuthContext(ctx)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: CreateInterviewSessionWithExistingResume] Error getting auth context", err)
		return nil, err
	}

	resumeID, err := uuid.Parse(req.ResumeID)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: CreateInterviewSessionWithExistingResume] Invalid resume ID", err)
		return nil, app_error.New(err, app_error.ErrCodeGeneralInvalidUUID)
	}

	resume, err := s.resumeRepo.GetResumeByID(ctx, resumeID)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: CreateInterviewSessionWithExistingResume] Error getting resume", err)
		return nil, err
	}

	resumeFile, err := s.s3Storage.DownloadFile(ctx, resume.StorageKey)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: CreateInterviewSessionWithExistingResume] Error downloading resume file", err)
		return nil, err
	}

	sessionID := s.generator.GenerateUUID(ctx)
	extractResumeJsonForRAGReq := &repositories.ExtractResumeJsonForRAGReq{
		SessionID:  sessionID.String(),
		UserID:     authCtx.Payload.UserID,
		ResumeID:   resumeID.String(),
		ResumeFile: resumeFile,
	}

	err = s.resumeRepo.ExtractResumeJsonForRAG(ctx, extractResumeJsonForRAGReq)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: CreateInterviewSessionWithExistingResume] Error extracting resume json for RAG", err)
		return nil, err
	}

	userID, err := uuid.Parse(authCtx.Payload.UserID)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: CreateInterviewSessionWithExistingResume] Invalid user ID", err)
		return nil, app_error.New(err, app_error.ErrCodeGeneralInvalidUUID)
	}

	createInterviewSessionWithExistingResumeReq := &db.CreateInterviewSessionParams{
		ID:             sessionID,
		ResumeID:       resumeID,
		UserID:         userID,
		ResumeFileName: resume.FileName,
		Position:       req.Position,
		Status:         constants.StatusPending,
		Modality:       constants.ModalityVoiceChat,
		IsConsent:      req.IsConsent,
	}

	if err := s.interviewSessionRepo.CreateInterviewSession(ctx, createInterviewSessionWithExistingResumeReq); err != nil {
		s.log.ErrorWithID(ctx, "[Service: CreateInterviewSessionWithExistingResume] Error creating interview session with existing resume", err)
		return nil, err
	}

	tokenReq := map[string]any{
		"session_id": sessionID,
		"user_id":    authCtx.Payload.UserID,
		"resume_id":  resumeID.String(),
		"role":       authCtx.Payload.Role,
	}

	tokenReqJSON, err := json.Marshal(tokenReq)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: CreateInterviewSessionWithExistingResume] Error marshaling token request to JSON", err)
		return nil, err
	}

	sessionToken := s.generator.GenerateUUID(ctx).String()
	redisKey := fmt.Sprintf("%s%s", constants.RedisPrefixInterviewSessionToken, sessionToken)
	redisPayload := database.RedisPayload{
		Key:   redisKey,
		Value: string(tokenReqJSON),
		TTL:   s.config.InterviewSessionConfig.InterviewSessionDuration,
	}

	if err := s.redisClient.Set(ctx, redisPayload); err != nil {
		s.log.ErrorWithID(ctx, "[Service: CreateInterviewSessionWithExistingResume] Error creating interview session token", err)
		return nil, err
	}

	resp := &entities.CreateInterviewSessionWithExistingResumeResp{
		SessionToken: sessionToken,
	}

	return resp, nil
}

func (s *interviewSessionService) CreateInterviewerSessionTurnBySessionID(ctx context.Context, req *entities.CreateInterviewerSessionTurnBySessionIDReq) error {
	s.log.InfoWithID(ctx, "[Service: CreateInterviewerSessionTurnBySessionID] Called")

	redisKey := fmt.Sprintf("%s%s", constants.RedisPrefixInterviewMaxTurnNo, req.SessionID)

	segmentID, err := uuid.Parse(req.TurnID)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: CreateInterviewerSessionTurnBySessionID] Invalid segment ID", err)
		return app_error.New(err, app_error.ErrCodeGeneralInvalidUUID)
	}

	sessionID, err := uuid.Parse(req.SessionID)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: CreateInterviewerSessionTurnBySessionID] Invalid session ID", err)
		return app_error.New(err, app_error.ErrCodeGeneralInvalidUUID)
	}

	maxTurnNo, err := s.increaseMaxTurnNo(ctx, sessionID, redisKey)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: CreateUserSessionTurnBySessionID] Error getting max turn no", err)
		return err
	}

	dbReq := &db.CreateInterviewTurnParams{
		ID:             segmentID,
		SessionID:      sessionID,
		TurnNo:         maxTurnNo,
		Actor:          constants.ActorInterviewer,
		CurrentState:   req.CurrentState,
		TranscriptText: req.Transcript,
		StartAt:        req.StartedAt,
		EndAt:          req.EndedAt,
	}

	if err := s.interviewTurnsRepo.CreateSessionTurnBySessionID(context.Background(), dbReq); err != nil {
		s.log.ErrorWithID(ctx, "[Service: CreateInterviewerSessionTurnBySessionID] Error creating session turn by session ID", err)
		return err
	}

	redisKey = fmt.Sprintf("%s%s:%s", constants.RedisPrefixInterviewIsScoreSessionState, req.SessionID, req.CurrentState)
	redisPayload := database.RedisPayload{
		Key:   redisKey,
		Value: false,
		TTL:   constants.RedisTTLInterviewIsScoreSessionState,
	}

	_ = s.redisClient.Set(ctx, redisPayload)

	return nil
}

func (s *interviewSessionService) CreateUserSessionTurnBySessionID(ctx context.Context, req *entities.CreateUserSessionTurnBySessionIDReq) error {
	s.log.InfoWithID(ctx, "[Service: CreateUserSessionTurnBySessionID] Called")

	redisKey := fmt.Sprintf("%s%s", constants.RedisPrefixInterviewMaxTurnNo, req.SessionID)

	segmentID, err := uuid.Parse(req.TurnID)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: CreateUserSessionTurnBySessionID] Invalid segment ID", err)
		return app_error.New(err, app_error.ErrCodeGeneralInvalidUUID)
	}

	sessionID, err := uuid.Parse(req.SessionID)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: CreateUserSessionTurnBySessionID] Invalid session ID", err)
		return app_error.New(err, app_error.ErrCodeGeneralInvalidUUID)
	}

	maxTurnNo, err := s.increaseMaxTurnNo(ctx, sessionID, redisKey)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: CreateUserSessionTurnBySessionID] Error getting max turn no", err)
		return err
	}

	timeDuration, err := s.GetSessionStartedAtAndEndedAt(ctx, req.SessionID)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: CreateUserSessionTurnBySessionID] Error getting interview session start end time", err)
		return err
	}

	dbReq := &db.CreateInterviewTurnParams{
		ID:             segmentID,
		SessionID:      sessionID,
		TurnNo:         maxTurnNo,
		Actor:          constants.ActorUser,
		CurrentState:   req.CurrentState,
		TranscriptText: req.Transcript,
		StartAt:        timeDuration["started_at"],
		EndAt:          timeDuration["ended_at"],
	}

	if err := s.interviewTurnsRepo.CreateSessionTurnBySessionID(context.Background(), dbReq); err != nil {
		s.log.ErrorWithID(ctx, "[Service: CreateUserSessionTurnBySessionID] Error creating session turn by session ID", err)
		return err
	}

	return nil
}

func (s *interviewSessionService) GetSessionStartedAtAndEndedAt(ctx context.Context, sessionID string) (map[string]string, error) {
	s.log.InfoWithID(ctx, "[Service: GetSessionStartedAtAndEndedAt] Called")

	redisKey := fmt.Sprintf("%s%s", constants.RedisPrefixInterviewStartEndTime, sessionID)

	timeDuration, err := s.redisClient.HGetAll(context.Background(), redisKey)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: GetSessionStartedAtAndEndedAt] Error getting interview session start end time", err)
		return nil, err
	} else if len(timeDuration) == 0 {
		s.log.ErrorWithID(ctx, "[Service: GetSessionStartedAtAndEndedAt] Interview session start end time not found")
		return nil, app_error.New(constants.ErrInterviewSessionStartEndTimeNotFound, app_error.ErrCodeInterviewSessionStartEndTimeNotFound)
	} else if timeDuration["started_at"] == "" {
		s.log.ErrorWithID(ctx, "[Service: GetSessionStartedAtAndEndedAt] Interview session start time not found")
		return nil, app_error.New(constants.ErrInterviewSessionStartTimeNotFound, app_error.ErrCodeInterviewSessionStartTimeNotFound)
	} else if timeDuration["ended_at"] == "" {
		s.log.ErrorWithID(ctx, "[Service: GetSessionStartedAtAndEndedAt] Interview session end time not found")
		return nil, app_error.New(constants.ErrInterviewSessionEndTimeNotFound, app_error.ErrCodeInterviewSessionEndTimeNotFound)
	}

	return timeDuration, nil
}

func (s *interviewSessionService) increaseMaxTurnNo(ctx context.Context, sessionID uuid.UUID, redisKey string) (int64, error) {
	s.log.InfoWithID(ctx, "[Service: IncreaseMaxTurnNo] Called")

	maxTurnNo, err := s.redisClient.Increment(ctx, redisKey)

	if err != nil {
		dbMaxTurnNo, err := s.interviewTurnsRepo.GetMaxTurnNoBySessionID(ctx, sessionID)
		if err != nil {
			s.log.ErrorWithID(ctx, "[Service: IncreaseMaxTurnNo] Failed to get max turn from DB", err)
			return 0, err
		}

		maxTurnNo = dbMaxTurnNo + 1

		redisPayload := database.RedisPayload{
			Key:   redisKey,
			Value: maxTurnNo,
			TTL:   constants.RedisTTLInterviewTurn,
		}
		if err := s.redisClient.Set(ctx, redisPayload); err != nil {
			s.log.WarnWithID(ctx, "[Service: IncreaseMaxTurnNo] Failed to cache to Redis", err)
		}

	}

	return maxTurnNo, nil
}

func (s *interviewSessionService) SetSessionStartTime(ctx context.Context, req *entities.SetSessionStartTimeReq) error {
	s.log.InfoWithID(ctx, "[Service: SetSessionStartTime] Called")
	context := context.Background()
	redisKey := fmt.Sprintf("%s%s", constants.RedisPrefixInterviewStartEndTime, req.SessionID)

	if err := s.redisClient.HSet(context, redisKey, "started_at", req.StartedAt); err != nil {
		s.log.ErrorWithID(ctx, "[Service: SetSessionStartTime] Error creating interview session start time redis key", err)
		return err
	}

	return nil
}

func (s *interviewSessionService) SetSessionEndTime(ctx context.Context, req *entities.SetSessionEndTimeReq) error {
	s.log.InfoWithID(ctx, "[Service: SetSessionEndTime] Called")
	context := context.Background()
	redisKey := fmt.Sprintf("%s%s", constants.RedisPrefixInterviewStartEndTime, req.SessionID)

	if err := s.redisClient.HSet(context, redisKey, "ended_at", req.EndedAt); err != nil {
		s.log.ErrorWithID(ctx, "[Service: SetSessionEndTime] Error creating interview session end time redis key", err)
		return err
	}

	return nil
}

func (s *interviewSessionService) UpdateInterviewSessionStatus(ctx context.Context, req *entities.UpdateInterviewSessionStatusReq) error {
	s.log.InfoWithID(ctx, "[Service: UpdateInterviewSessionStatus] Called")

	sessionID, err := uuid.Parse(req.SessionID)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: UpdateInterviewSessionStatus] Invalid session ID", err)
		return app_error.New(err, app_error.ErrCodeGeneralInvalidUUID)
	}

	dbReq := &db.UpdateInterviewSessionStatusParams{
		ID:      sessionID,
		Column2: req.Status,
	}

	if err := s.interviewSessionRepo.UpdateInterviewSessionStatus(ctx, dbReq); err != nil {
		s.log.ErrorWithID(ctx, "[Service: UpdateInterviewSessionStatus] Error updating interview session status", err)
		return err
	}

	return nil
}

func (s *interviewSessionService) IsSessionValid(ctx context.Context, req *entities.IsSessionValidReq) (*entities.IsSessionValidResp, error) {
	s.log.InfoWithID(ctx, "[Service: IsSessionValid] Called")

	redisKey := fmt.Sprintf("%s%s", constants.RedisPrefixInterviewSessionToken, req.SessionToken)

	redisSessionToken, err := s.redisClient.Get(ctx, redisKey)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: IsSessionValid] Error getting redis session token", err)
		return nil, app_error.New(err, app_error.ErrCodeSessionNotFound)
	}

	var sessionPayload entities.RedisSessionToken
	if err := json.Unmarshal([]byte(redisSessionToken), &sessionPayload); err != nil {
		s.log.ErrorWithID(ctx, "[Service: IsSessionValid] Error unmarshalling redis session token", err)
		return nil, app_error.New(err, app_error.ErrCodeSessionInvalidToken)
	}

	if sessionPayload.UserID != req.UserID {
		s.log.ErrorWithID(ctx, "[Service: IsSessionValid] User ID mismatch")
		return nil, app_error.New(constants.ErrInvalidToken, app_error.ErrCodeSessionInvalidToken)
	}

	sessionID, err := uuid.Parse(sessionPayload.SessionID)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: IsSessionValid] Invalid session ID", err)
		return nil, app_error.New(err, app_error.ErrCodeGeneralInvalidUUID)
	}

	exists, err := s.interviewSessionRepo.CheckInterviewSessionExists(ctx, sessionID)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: IsSessionValid] Error checking interview session exists", err)
		return nil, err
	}
	if !exists {
		s.log.ErrorWithID(ctx, "[Service: IsSessionValid] Interview session not found")
		return nil, app_error.New(constants.ErrInvalidToken, app_error.ErrCodeSessionNotFound)
	}

	resp := &entities.IsSessionValidResp{
		UserID:    sessionPayload.UserID,
		SessionID: sessionPayload.SessionID,
		ResumeID:  sessionPayload.ResumeID,
	}

	return resp, nil
}

func (s *interviewSessionService) GetInterviewerLastMessage(ctx context.Context, req *entities.GetInterviewerLastMessageReq) (*entities.GetInterviewerLastMessageResp, error) {
	s.log.InfoWithID(ctx, "[Service: GetInterviewerLastMessage] Called")

	redisKey := fmt.Sprintf("%s%s", constants.RedisPrefixInterviewLastMessage, req.SessionID)
	lastMessage, err := s.redisClient.Get(context.Background(), redisKey)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: GetInterviewerLastMessage] Last message not found")

		sessionID, err := uuid.Parse(req.SessionID)
		if err != nil {
			s.log.ErrorWithID(ctx, "[Service: GetInterviewerLastMessage] Invalid session ID", err)
			return nil, app_error.New(err, app_error.ErrCodeGeneralInvalidUUID)
		}

		lastMessage, err := s.interviewTurnsRepo.GetInterviewerLastMessage(context.Background(), sessionID)
		if err != nil {
			s.log.ErrorWithID(ctx, "[Service: GetInterviewerLastMessage] Error getting last message", err)
			return nil, err
		}

		resp := &entities.GetInterviewerLastMessageResp{
			Message:      lastMessage.TranscriptText,
			CurrentState: lastMessage.CurrentState,
		}

		return resp, nil
	}

	var resp entities.GetInterviewerLastMessageResp
	if err := json.Unmarshal([]byte(lastMessage), &resp); err != nil {
		s.log.ErrorWithID(ctx, "[Service: GetInterviewerLastMessage] Error unmarshalling last message", err)
		return nil, app_error.New(err, app_error.ErrCodeSessionInvalidToken)
	}

	return &resp, nil
}

func (s *interviewSessionService) GetChatHistoryBySessionToken(ctx context.Context, req *entities.GetChatHistoryBySessionTokenReq) (*entities.GetChatHistoryBySessionTokenResp, error) {
	s.log.InfoWithID(ctx, "[Service: GetChatHistoryBySessionToken] Called")

	authContext, err := s.authContext.GetAuthContext(ctx)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: GetChatHistoryBySessionToken] Error getting auth context", err)
		return nil, err
	}

	sessionValidReq := &entities.IsSessionValidReq{
		SessionToken: req.SessionToken,
		UserID:       authContext.Payload.UserID,
	}

	sessionPayload, err := s.IsSessionValid(ctx, sessionValidReq)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: GetChatHistoryBySessionToken] Error checking session valid", err)
		return nil, err
	}

	chatHistory, err := s.interviewTurnsRepo.GetChatHistoryBySessionID(ctx, uuid.MustParse(sessionPayload.SessionID))
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: GetChatHistoryBySessionToken] Error getting chat history", err)
		return nil, err
	}

	chatHistoryResp := make([]entities.ChatHistory, len(chatHistory))
	for i, chat := range chatHistory {
		chatHistoryResp[i] = entities.ChatHistory{
			ID:             chat.ID,
			TurnNo:         chat.TurnNo,
			Actor:          chat.Actor,
			TranscriptText: chat.TranscriptText,
			StartAt:        chat.StartAt,
			EndAt:          chat.EndAt,
			CreatedAt:      utils.FormatToBangkokFullTimeFormat(chat.CreatedAt),
		}
	}

	resp := &entities.GetChatHistoryBySessionTokenResp{
		ChatHistory: chatHistoryResp,
	}

	return resp, nil
}

func (s *interviewSessionService) CheckExistsAndInitStartedAtInterviewSession(ctx context.Context, sessionId string) (*entities.CheckExistsAndInitStartedAtInterviewSessionResp, error) {
	s.log.InfoWithID(ctx, "[Service: GetStartedAtInterviewSession] Called")

	sessionID, err := uuid.Parse(sessionId)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: GetStartedAtInterviewSession] Invalid session ID", err)
		return nil, app_error.New(err, app_error.ErrCodeGeneralInvalidUUID)
	}

	dbResp, err := s.interviewSessionRepo.GetStartedAndIsStartedConversationSession(ctx, sessionID)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: GetStartedAtInterviewSession] Error getting started at interview session", err)
		return nil, err
	}

	var currentState string
	if dbResp.CurrentState.Valid {
		currentState = dbResp.CurrentState.String
	} else {
		currentState = ""
	}

	var currentStateID string
	if dbResp.CurrentStateID.Valid {
		currentStateID = dbResp.CurrentStateID.UUID.String()
	} else {
		currentStateID = ""
	}

	if dbResp.StartedAt.Valid {
		return &entities.CheckExistsAndInitStartedAtInterviewSessionResp{
			StartedAt:             utils.FormatToUTCString(dbResp.StartedAt.Time),
			IsStartedConversation: dbResp.IsStartedConversation.Bool,
			CurrentState:          currentState,
			CurrentStateID:        currentStateID,
		}, nil
	} else {
		currStartedAt := time.Now()

		updateReq := &db.UpdateStartedAtInterviewSessionParams{
			ID:        sessionID,
			StartedAt: sql.NullTime{Time: currStartedAt, Valid: true},
		}

		if err := s.interviewSessionRepo.UpdateStartedAtInterviewSession(ctx, updateReq); err != nil {
			s.log.ErrorWithID(ctx, "[Service: GetStartedAtInterviewSession] Error updating started at interview session", err)
			return nil, err
		}

		return &entities.CheckExistsAndInitStartedAtInterviewSessionResp{
			StartedAt:             utils.FormatToUTCString(currStartedAt),
			IsStartedConversation: dbResp.IsStartedConversation.Bool,
			CurrentState:          currentState,
			CurrentStateID:        currentStateID,
		}, nil
	}
}

func (s *interviewSessionService) StartConversationBySessionID(ctx context.Context, sessionId string) error {
	s.log.InfoWithID(ctx, "[Service: StartConversationBySessionID] Called")

	dbReq := &db.UpdateIsStartedConversationSessionParams{
		ID:                    uuid.MustParse(sessionId),
		IsStartedConversation: sql.NullBool{Bool: true, Valid: true},
	}

	if err := s.interviewSessionRepo.UpdateIsStartedConversationSession(ctx, dbReq); err != nil {
		s.log.ErrorWithID(ctx, "[Service: StartConversationBySessionID] Error updating started conversation", err)
		return err
	}

	return nil
}

func (s *interviewSessionService) EndInterviewSession(ctx context.Context, req *entities.EndInterviewSessionReq) error {
	s.log.InfoWithID(ctx, "[Service: EndInterviewSession] Called")

	sessionID, err := uuid.Parse(req.SessionId)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: EndInterviewSession] Invalid session ID", err)
		return app_error.New(err, app_error.ErrCodeGeneralInvalidUUID)
	}

	evaluations, err := s.evaluationScoresRepo.GetAllEvaluationsBySessionID(ctx, sessionID)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: EndInterviewSession] Error getting all evaluations by session ID", err)
		return err
	}

	var overallScore float64
	var overallSummaryMd string

	evaluationCount := len(evaluations)
	if evaluationCount > 0 {
		createEvaluationOverallSummaryTxReq := &repositories.CreateEvaluationOverallSummaryTxReq{
			SummaryMd: []string{},
		}

		for _, evaluation := range evaluations {
			overallScoreFloat, err := strconv.ParseFloat(evaluation.OverallScore, 64)
			if err != nil {
				s.log.ErrorWithID(ctx, "[Service: EndInterviewSession] Error parsing overall score", err)
				return app_error.New(err, app_error.ErrCodeGeneralInvalidNumber)
			}
			overallScore += overallScoreFloat
			createEvaluationOverallSummaryTxReq.SummaryMd = append(createEvaluationOverallSummaryTxReq.SummaryMd, evaluation.SummaryMd)
		}

		overallScore = overallScore / float64(evaluationCount)

		overallSummary, err := s.evaluationScoresRepo.GetEvaluationOverallSummary(ctx, createEvaluationOverallSummaryTxReq)
		if err != nil {
			s.log.ErrorWithID(ctx, "[Service: EndInterviewSession] Error getting evaluation overall summary", err)
			return err
		}

		overallSummaryMd = overallSummary.OverallSummaryMd
	} else {
		s.log.ErrorWithID(ctx, "[Service: EndInterviewSession] No evaluations found")
		overallScore = 0.0
		overallSummaryMd = constants.BlankOverallSummaryMd
	}

	dbReq := &db.EndInterviewSessionParams{
		ID:           sessionID,
		Status:       req.Status,
		EndedAt:      sql.NullTime{Time: time.Now().UTC(), Valid: true},
		OverallScore: sql.NullFloat64{Float64: math.Round(overallScore*100) / 100, Valid: true},
		SummaryMd:    sql.NullString{String: overallSummaryMd, Valid: true},
	}

	if err := s.interviewSessionRepo.EndInterviewSession(ctx, dbReq); err != nil {
		s.log.ErrorWithID(ctx, "[Service: EndInterviewSession] Error ending interview session", err)
		return err
	}

	return nil
}

func (s *interviewSessionService) ListInterviewSessionsByUserIDWithCursor(ctx context.Context, req *entities.ListInterviewSessionsByUserIDWithCursorReq) (*entities.ListInterviewSessionsByUserIDResp, error) {
	s.log.InfoWithID(ctx, "[Service: ListInterviewSessionsByUserID] Called")

	authContext, err := s.authContext.GetAuthContext(ctx)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: ListInterviewSessionsByUserID] Error getting auth context", err)
		return nil, err
	}

	userId, err := uuid.Parse(authContext.Payload.UserID)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: ListInterviewSessionsByUserID] Invalid user ID", err)
		return nil, app_error.New(err, app_error.ErrCodeGeneralInvalidUUID)
	}

	var limit int32 = 20
	if req.Limit != nil {
		limit = int32(*req.Limit)
	}

	paginationType := constants.PaginationCursorTypeNext
	if req.Type != nil {
		if *req.Type != constants.PaginationCursorTypeNext && *req.Type != constants.PaginationCursorTypePrev {
			s.log.ErrorWithID(ctx, "[Service: ListInterviewSessionsByUserID] Invalid pagination type", "pagination_type", *req.Type)
			return nil, app_error.New(errors.New("invalid pagination type"), app_error.ErrCodeGeneralInvalidPaginationType)
		} else {
			paginationType = *req.Type
		}
	}

	countReq := &db.CountInterviewSessionsByUserIDParams{
		UserID: userId,
	}

	searchText := ""
	if req.SearchText != nil {
		searchText = *req.SearchText
		countReq.Column2 = *req.SearchText
	} else {
		s.log.InfoWithID(ctx, "[Service: ListInterviewSessionsByUserID] Search text is nil", "search_text", "NULL")
		countReq.Column2 = ""
	}

	totalCount, err := s.interviewSessionRepo.CountInterviewSessionsByUserID(ctx, countReq)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: ListInterviewSessionsByUserID] Error counting interview sessions by user ID", err)
		return nil, err
	}

	pageSize := int(limit)
	totalPages := int((totalCount + int64(pageSize) - 1) / int64(pageSize))

	if paginationType == constants.PaginationCursorTypePrev {
		limit += 1
	}

	var sessions []entities.InterviewSessionSummary
	if req.Cursor == nil {
		firstPageReq := &db.ListInterviewSessionsByUserIDFirstPageParams{
			UserID:  userId,
			Column2: searchText,
			Limit:   limit,
		}

		dbResp, err := s.interviewSessionRepo.ListInterviewSessionsByUserIDFirstPage(ctx, firstPageReq)
		if err != nil {
			s.log.ErrorWithID(ctx, "[Service: ListInterviewSessionsByUserID] Error listing interview sessions by user ID first page", err)
			return nil, err
		}

		sessions = make([]entities.InterviewSessionSummary, 0, len(dbResp))
		for _, row := range dbResp {
			session := entities.InterviewSessionSummary{
				ID:               row.ID.String(),
				ResumeID:         row.ResumeID.String(),
				ResumeFileName:   row.ResumeFileName,
				Position:         row.Position,
				Status:           row.Status,
				CreatedAt:        utils.FormatNullableTimeToUTCString(row.CreatedAt),
				CreatedAtDisplay: utils.FormatNullableTimeToBangkokString(row.CreatedAt),
				OverallScore:     utils.GetNullableFloat64(row.OverallScore, 0.00),
			}

			if row.StartedAt.Valid && row.EndedAt.Valid {
				duration := row.EndedAt.Time.Sub(row.StartedAt.Time)
				totalMinutes := int(duration.Minutes())
				totalSeconds := int(duration.Seconds()) % 60
				session.TotalTime = fmt.Sprintf("%02d.%02d", totalMinutes, totalSeconds)
			} else {
				session.TotalTime = "00.00"
			}

			sessions = append(sessions, session)
		}
	} else {
		cursorID, err := uuid.Parse(req.Cursor.ID)
		if err != nil {
			s.log.ErrorWithID(ctx, "[Service: ListInterviewSessionsByUserID] Invalid cursor ID", err)
			return nil, app_error.New(err, app_error.ErrCodeGeneralInvalidUUID)
		}

		cursorCreatedAt, err := time.Parse(time.RFC3339, req.Cursor.CreatedAt)
		if err != nil {
			s.log.ErrorWithID(ctx, "[Service: ListInterviewSessionsByUserID] Invalid cursor created at", err)
			return nil, app_error.New(err, app_error.ErrCodeGeneralInvalidTime)
		}

		cursorReq := &db.ListInterviewSessionsByUserIDWithCursorParams{
			UserID:    userId,
			Column2:   searchText,
			CreatedAt: sql.NullTime{Time: cursorCreatedAt, Valid: true},
			Column4:   paginationType,
			ID:        cursorID,
			Limit:     limit,
		}

		dbResp, err := s.interviewSessionRepo.ListInterviewSessionsByUserIDWithCursor(ctx, cursorReq)
		if err != nil {
			s.log.ErrorWithID(ctx, "[Service: ListInterviewSessionsByUserID] Error listing interview sessions by user ID with cursor", err)
			return nil, err
		}

		sessions = make([]entities.InterviewSessionSummary, 0, len(dbResp))
		for _, row := range dbResp {
			if paginationType == constants.PaginationCursorTypePrev && row.ID == cursorID {
				continue
			}

			session := entities.InterviewSessionSummary{
				ID:               row.ID.String(),
				ResumeID:         row.ResumeID.String(),
				ResumeFileName:   row.ResumeFileName,
				Position:         row.Position,
				Status:           row.Status,
				CreatedAt:        utils.FormatNullableTimeToUTCString(row.CreatedAt),
				CreatedAtDisplay: utils.FormatNullableTimeToBangkokString(row.CreatedAt),
			}

			session.OverallScore = utils.GetNullableFloat64(row.OverallScore, 0.00)

			if row.StartedAt.Valid && row.EndedAt.Valid {
				duration := row.EndedAt.Time.Sub(row.StartedAt.Time)
				totalMinutes := int(duration.Minutes())
				totalSeconds := int(duration.Seconds()) % 60
				session.TotalTime = fmt.Sprintf("%02d.%02d", totalMinutes, totalSeconds)
			} else {
				session.TotalTime = "00.00"
			}

			sessions = append(sessions, session)
		}
	}

	if paginationType == constants.PaginationCursorTypePrev {
		for i, j := 0, len(sessions)-1; i < j; i, j = i+1, j-1 {
			sessions[i], sessions[j] = sessions[j], sessions[i]
		}
	}

	resp := entities.ListInterviewSessionsByUserIDResp{
		Sessions:   sessions,
		TotalPages: totalPages,
		PageSize:   pageSize,
	}

	if len(sessions) > 0 {
		lastSession := sessions[len(sessions)-1]
		resp.NextCursor = &entities.Cursor{
			ID:        lastSession.ID,
			CreatedAt: lastSession.CreatedAt,
		}

		if req.Cursor != nil {
			firstSession := sessions[0]
			resp.PrevCursor = &entities.Cursor{
				ID:        firstSession.ID,
				CreatedAt: firstSession.CreatedAt,
			}
		}
	}

	return &resp, nil
}

func (s *interviewSessionService) ListInterviewSessionsByUserIDWithJumpPagination(ctx context.Context, req *entities.ListInterviewSessionsByUserIDWithJumpPaginationReq) (*entities.ListInterviewSessionsByUserIDResp, error) {
	s.log.InfoWithID(ctx, "[Service: ListInterviewSessionsByUserID] Called")

	authContext, err := s.authContext.GetAuthContext(ctx)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: ListInterviewSessionsByUserID] Error getting auth context", err)
		return nil, err
	}

	userId, err := uuid.Parse(authContext.Payload.UserID)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: ListInterviewSessionsByUserID] Invalid user ID", err)
		return nil, app_error.New(err, app_error.ErrCodeGeneralInvalidUUID)
	}

	var dbResp []db.ListInterviewSessionsByUserIDWithJumpPaginationRow
	var limit int32 = 20
	if req.Limit != nil {
		limit = int32(*req.Limit)
	}

	dbReq := &db.ListInterviewSessionsByUserIDWithJumpPaginationParams{
		UserID: userId,
		Limit:  limit,
	}

	if req.Offset != nil {
		dbReq.Column2 = *req.Offset
	} else {
		dbReq.Column2 = 1
	}

	countReq := &db.CountInterviewSessionsByUserIDParams{
		UserID: userId,
	}
	if req.SearchText != nil {
		countReq.Column2 = *req.SearchText
		dbReq.Column4 = *req.SearchText
	}

	totalCount, err := s.interviewSessionRepo.CountInterviewSessionsByUserID(ctx, countReq)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: ListInterviewSessionsByUserID] Error counting interview sessions by user ID", err)
		return nil, err
	}

	dbResp, err = s.interviewSessionRepo.ListInterviewSessionsByUserIDWithJumpPagination(ctx, dbReq)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: ListInterviewSessionsByUserID] Error listing interview sessions by user ID with search and jump pagination", err)
		return nil, err
	}

	sessions := make([]entities.InterviewSessionSummary, 0, len(dbResp))
	for _, row := range dbResp {
		session := entities.InterviewSessionSummary{
			ID:               row.ID.String(),
			ResumeID:         row.ResumeID.String(),
			ResumeFileName:   row.ResumeFileName,
			Position:         row.Position,
			Status:           row.Status,
			CreatedAt:        utils.FormatNullableTimeToUTCString(row.CreatedAt),
			CreatedAtDisplay: utils.FormatNullableTimeToBangkokString(row.CreatedAt),
		}

		session.OverallScore = utils.GetNullableFloat64(row.OverallScore, 0.00)

		if row.StartedAt.Valid && row.EndedAt.Valid {
			duration := row.EndedAt.Time.Sub(row.StartedAt.Time)
			totalMinutes := int(duration.Minutes())
			totalSeconds := int(duration.Seconds()) % 60
			session.TotalTime = fmt.Sprintf("%02d.%02d", totalMinutes, totalSeconds)
		} else {
			session.TotalTime = "00.00"
		}

		sessions = append(sessions, session)
	}

	pageSize := int(limit)
	totalPages := int((totalCount + int64(pageSize) - 1) / int64(pageSize))

	resp := entities.ListInterviewSessionsByUserIDResp{
		Sessions:   sessions,
		TotalPages: totalPages,
		PageSize:   pageSize,
	}

	if len(sessions) > 0 {
		lastSession := sessions[len(sessions)-1]
		resp.NextCursor = &entities.Cursor{
			ID:        lastSession.ID,
			CreatedAt: lastSession.CreatedAt,
		}

		if req.Offset != nil {
			firstSession := sessions[0]
			resp.PrevCursor = &entities.Cursor{
				ID:        firstSession.ID,
				CreatedAt: firstSession.CreatedAt,
			}
		}
	}

	return &resp, nil
}

func (s *interviewSessionService) DeleteUserInterviewSessionByID(ctx context.Context, sessionIDReq string) error {
	s.log.InfoWithID(ctx, "[Service: DeleteUserInterviewSessionByID] Called")

	sessionID, err := uuid.Parse(sessionIDReq)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: DeleteUserInterviewSessionByID] Invalid session ID", err)
		return app_error.New(err, app_error.ErrCodeGeneralInvalidUUID)
	}

	authContext, err := s.authContext.GetAuthContext(ctx)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: DeleteUserInterviewSessionByID] Error getting auth context", err)
		return err
	}

	userID, err := uuid.Parse(authContext.Payload.UserID)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: DeleteUserInterviewSessionByID] Invalid user ID", err)
		return app_error.New(err, app_error.ErrCodeGeneralInvalidUUID)
	}

	dbReq := &db.DeleteUserInterviewSessionByIDParams{
		ID:     sessionID,
		UserID: userID,
	}

	if err := s.interviewSessionRepo.DeleteUserInterviewSessionByID(ctx, dbReq); err != nil {
		s.log.ErrorWithID(ctx, "[Service: DeleteUserInterviewSessionByID] Error deleting interview session", err)
		return err
	}

	return nil
}

func (s *interviewSessionService) GetInterviewSessionInformationByID(ctx context.Context, sessionIDReq string) (*entities.GetInterviewSessionInformationResp, error) {
	s.log.InfoWithID(ctx, "[Service: GetInterviewSessionInformationByID] Called")

	sessionID, err := uuid.Parse(sessionIDReq)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: GetInterviewSessionInformationByID] Invalid session ID", err)
		return nil, app_error.New(err, app_error.ErrCodeGeneralInvalidUUID)
	}

	authContext, err := s.authContext.GetAuthContext(ctx)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: GetInterviewSessionInformationByID] Error getting auth context", err)
		return nil, err
	}

	userID, err := uuid.Parse(authContext.Payload.UserID)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: GetInterviewSessionInformationByID] Invalid user ID", err)
		return nil, app_error.New(err, app_error.ErrCodeGeneralInvalidUUID)
	}

	dbResp, err := s.interviewSessionRepo.GetInterviewSessionInformationByID(ctx, sessionID)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: GetInterviewSessionInformationByID] Error getting interview session information", err)
		return nil, err
	}

	if dbResp.UserID != userID {
		s.log.ErrorWithID(ctx, "[Service: GetInterviewSessionInformationByID] User ID does not match", err)
		return nil, app_error.New(constants.ErrPermissionDenied, app_error.ErrCodeSessionUserNotMatch)
	}

	overallScore := utils.GetNullableFloat64(dbResp.OverallScore, 0.00)

	if !utils.ValidateScoreRange(overallScore) {
		s.log.ErrorWithID(ctx, "[Service: GetInterviewSessionInformationByID] Invalid overall score", err)
		return nil, app_error.New(constants.ErrInterviewSessionInvalidOverallScore, app_error.ErrCodeSessionInvalidOverallScore)
	}

	if !utils.ValidateStatus(dbResp.Status) {
		s.log.ErrorWithID(ctx, "[Service: GetInterviewSessionInformationByID] Invalid status", err)
		return nil, app_error.New(constants.ErrInterviewSessionInvalidStatus, app_error.ErrCodeSessionInvalidStatus)
	}

	overallScorePercent, overallScoreColor := utils.FormatScoreWithColor(overallScore)
	statusColor := utils.GetStatusColor(dbResp.Status)
	statusDisplayName := utils.GetStatusDisplayName(dbResp.Status)

	resp := entities.GetInterviewSessionInformationResp{
		ResumeID:            dbResp.ResumeID,
		ResumeFileName:      dbResp.ResumeFileName,
		Position:            dbResp.Position,
		Status:              dbResp.Status,
		StatusDisplayName:   statusDisplayName,
		StatusColor:         statusColor,
		StartedAt:           utils.FormatNullableTimeToBangkokString(dbResp.StartedAt),
		EndedAt:             utils.FormatNullableTimeToBangkokString(dbResp.EndedAt),
		OverallScore:        overallScore,
		OverallScorePercent: overallScorePercent,
		OverallScoreColor:   overallScoreColor,
		SummaryMd:           utils.GetNullableString(dbResp.SummaryMd, constants.BlankOverallSummaryMd),
		CreatedAt:           utils.FormatNullableTimeToBangkokString(dbResp.CreatedAt),
		CreatedAtFullName:   utils.FormatNullableTimeToBangkokStringFullTimeFormat(dbResp.CreatedAt),
	}

	return &resp, nil
}

func (s *interviewSessionService) GetChatHistoryBySessionIDWithEvaluation(ctx context.Context, req *entities.GetChatHistoryBySessionIDWithEvaluationReq) (*entities.GetChatHistoryBySessionIDWithEvaluationResp, error) {
	s.log.InfoWithID(ctx, "[Service: GetChatHistoryBySessionIDWithEvaluation] Called")

	sessionID, err := uuid.Parse(req.SessionID)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: GetChatHistoryBySessionIDWithEvaluation] Invalid session ID", err)
		return nil, app_error.New(err, app_error.ErrCodeGeneralInvalidUUID)
	}

	var turnNo int32 = constants.TurnNoDefault
	if req.TurnNo != nil {
		turnNo = *req.TurnNo
	}

	dbReq := &db.GetChatHistoryBySessionIDWithEvaluationParams{
		SessionID: sessionID,
		TurnNo:    int64(turnNo),
		Limit:     constants.DefaultPageSize,
	}

	dbResp, err := s.interviewTurnsRepo.GetChatHistoryBySessionIDWithEvaluation(ctx, dbReq)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: GetChatHistoryBySessionIDWithEvaluation] Error getting chat history by session ID with evaluation", err)
		return nil, err
	}

	chatHistoryResp := make([]entities.ChatHistoryWithEvaluation, len(dbResp))
	for i, chat := range dbResp {
		var correctedSentence *string
		if chat.CorrectedSentence.Valid {
			correctedSentence = &chat.CorrectedSentence.String
		}

		var evaluation *entities.Evaluation
		if chat.Evaluation != nil {
			var evalMap map[string]interface{}
			if err := json.Unmarshal(chat.Evaluation.([]byte), &evalMap); err == nil {
				s.log.InfoWithID(ctx, "[Service: GetChatHistoryBySessionIDWithEvaluation] Evaluation: ", evalMap)

				var overallScoreFloat float64
				var err error

				if overallScoreStr, ok := evalMap["overall_score"].(string); ok {
					overallScoreFloat, err = strconv.ParseFloat(overallScoreStr, 64)
				} else if overallScoreNum, ok := evalMap["overall_score"].(float64); ok {
					overallScoreFloat = overallScoreNum
				} else {
					s.log.ErrorWithID(ctx, "[Service: GetInterviewSessionInformationByID] Invalid overall score format", nil)
					return nil, app_error.New(constants.ErrInterviewSessionInvalidOverallScore, app_error.ErrCodeSessionInvalidOverallScore)
				}
				if err != nil {
					s.log.ErrorWithID(ctx, "[Service: GetInterviewSessionInformationByID] Failed to parse overall score", err)
					return nil, app_error.New(constants.ErrInterviewSessionInvalidOverallScore, app_error.ErrCodeSessionInvalidOverallScore)
				}

				if !utils.ValidateScoreRange(overallScoreFloat) {
					s.log.ErrorWithID(ctx, "[Service: GetInterviewSessionInformationByID] Overall score out of valid range", nil)
					return nil, app_error.New(constants.ErrInterviewSessionInvalidOverallScore, app_error.ErrCodeSessionInvalidOverallScore)
				}

				overallColor := utils.GetScoreColor(overallScoreFloat)

				evaluation = &entities.Evaluation{
					OverallScore: fmt.Sprintf("%.2f", overallScoreFloat),
					OverallColor: overallColor,
					SummaryMd:    evalMap["summary_md"].(string),
				}

				if rawScores, ok := evalMap["scores"].([]interface{}); ok {
					criteriaScores := make([]entities.CriteriaScore, len(rawScores))
					for j, raw := range rawScores {
						scoreMap, ok := raw.(map[string]interface{})
						if !ok {
							s.log.ErrorWithID(ctx, "[Service: GetInterviewSessionInformationByID] Invalid score map format", nil)
							return nil, app_error.New(constants.ErrInterviewSessionInvalidOverallScore, app_error.ErrCodeSessionInvalidOverallScore)
						}

						criteriaScoreStr, ok := scoreMap["score"].(string)
						if !ok {
							if fScore, ok := scoreMap["score"].(float64); ok {
								criteriaScoreStr = fmt.Sprintf("%.2f", fScore)
							} else {
								s.log.ErrorWithID(ctx, "[Service: GetInterviewSessionInformationByID] Invalid criteria score format", nil)
								return nil, app_error.New(constants.ErrInterviewSessionInvalidOverallScore, app_error.ErrCodeSessionInvalidOverallScore)
							}
						}

						criteriaScoreFloat, err := strconv.ParseFloat(criteriaScoreStr, 64)
						if err != nil {
							s.log.ErrorWithID(ctx, "[Service: GetInterviewSessionInformationByID] Failed to parse criteria score", err)
							return nil, app_error.New(constants.ErrInterviewSessionInvalidOverallScore, app_error.ErrCodeSessionInvalidOverallScore)
						}

						if !utils.ValidateScoreRange(criteriaScoreFloat) {
							s.log.ErrorWithID(ctx, "[Service: GetInterviewSessionInformationByID] Criteria score out of valid range", nil)
							return nil, app_error.New(constants.ErrInterviewSessionInvalidOverallScore, app_error.ErrCodeSessionInvalidOverallScore)
						}

						criteriaScores[j] = entities.CriteriaScore{
							CriterionID:   scoreMap["criterion_id"].(string),
							CriterionName: scoreMap["criterion_name"].(string),
							Score:         criteriaScoreStr,
							ScoreColor:    utils.GetScoreColor(criteriaScoreFloat),
							CommentMd:     scoreMap["comment"].(string),
						}
					}

					evaluation.Scores = criteriaScores
				}
			}
		}

		chatHistoryResp[i] = entities.ChatHistoryWithEvaluation{
			ID:                chat.TurnID.String(),
			TurnNo:            chat.TurnNo,
			Actor:             chat.Actor,
			Content:           chat.TranscriptText,
			StartAt:           chat.StartAt,
			EndAt:             chat.EndAt,
			Evaluation:        evaluation,
			CorrectedSentence: correctedSentence,
			CurrentState:      chat.CurrentState,
			CurrentStateColor: constants.GetColorFromState(chat.CurrentState),
		}
	}

	resp := &entities.GetChatHistoryBySessionIDWithEvaluationResp{
		ChatHistory: chatHistoryResp,
	}

	if len(chatHistoryResp) > 0 {
		resp.CursorTurnNext = int32(chatHistoryResp[len(chatHistoryResp)-1].TurnNo)
	} else {
		resp.CursorTurnNext = 0
	}

	return resp, nil
}

func (s *interviewSessionService) convertToCustomFileHeader(fileHeader *multipart.FileHeader) *aws.CustomFileHeader {
	file, err := fileHeader.Open()
	if err != nil {
		return nil
	}
	defer file.Close()

	fileContent, err := io.ReadAll(file)
	if err != nil {
		return nil
	}

	return aws.NewCustomFileHeader(
		fileHeader.Filename,
		fileHeader.Size,
		fileHeader.Header,
		fileContent,
	)
}

func (s *interviewSessionService) InitialFirstCurrentStateSession(ctx context.Context, req *entities.InitialFirstCurrentStateSessionReq) (*entities.InitialFirstCurrentStateSessionResp, error) {
	s.log.InfoWithID(ctx, "[Service: InitialFirstCurrentStateSession] Called")

	sessionID, err := uuid.Parse(req.SessionID)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: InitialFirstCurrentStateSession] Invalid session ID", err)
		return nil, app_error.New(err, app_error.ErrCodeGeneralInvalidUUID)
	}

	interviewStateID := s.generator.GenerateUUID(ctx)

	dbReq := &repositories.CreateInterviewStateWithUpdateFlagSessionTxReq{
		ID:         interviewStateID,
		SessionID:  sessionID,
		PhraseType: req.CurrentState,
		StartedAt:  time.Now(),
	}

	err = s.interviewStateRepo.CreateInterviewStateWithUpdateFlagSessionTx(ctx, dbReq)

	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: InitialFirstCurrentStateSession] Error creating interview state", err)
		return nil, err
	}

	resp := entities.InitialFirstCurrentStateSessionResp{
		CurrentState:   req.CurrentState,
		CurrentStateID: interviewStateID.String(),
	}

	return &resp, nil
}

func (s *interviewSessionService) UpdateCurrentStateSession(ctx context.Context, req *entities.UpdateCurrentStateSessionReq) (*entities.UpdateCurrentStateSessionResp, error) {
	s.log.InfoWithID(ctx, "[Service: UpdateCurrentStateSession] Called")

	sessionID, err := uuid.Parse(req.SessionID)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: UpdateCurrentStateSession] Invalid session ID", err)
		return nil, app_error.New(err, app_error.ErrCodeGeneralInvalidUUID)
	}

	oldInterviewStateID, err := uuid.Parse(req.OldCurrentStateID)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: UpdateCurrentStateSession] Invalid old interview state ID", err)
		return nil, app_error.New(err, app_error.ErrCodeGeneralInvalidUUID)
	}

	currentTime := time.Now()
	newInterviewStateID := s.generator.GenerateUUID(ctx)

	dbReq := &repositories.EndOldInterviewStateAndCreateNewInterviewStateWithUpdateFlagSessionTxReq{
		ID:      oldInterviewStateID,
		EndedAt: currentTime,

		NewID:      newInterviewStateID,
		SessionID:  sessionID,
		PhraseType: req.CurrentState,
		StartedAt:  currentTime,
	}

	s.log.InfoWithID(ctx, "[Service: UpdateCurrentStateSession] Ending old interview state and creating new interview state Req", dbReq)

	if err = s.interviewStateRepo.EndOldInterviewStateAndCreateNewInterviewStateWithUpdateFlagSessionTx(ctx, dbReq); err != nil {
		s.log.ErrorWithID(ctx, "[Service: UpdateCurrentStateSession] Error creating interview state", err)
		return nil, err
	}

	resp := entities.UpdateCurrentStateSessionResp{
		CurrentState:   req.CurrentState,
		CurrentStateID: newInterviewStateID.String(),
	}

	return &resp, nil
}
