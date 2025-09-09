package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"strconv"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gitlab.com/interview-simulation/interview-backend-server/internal/config"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	db "gitlab.com/interview-simulation/interview-backend-server/internal/db/sqlc"
	"gitlab.com/interview-simulation/interview-backend-server/internal/entities"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/aws"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/database"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/queue"
	"gitlab.com/interview-simulation/interview-backend-server/internal/middleware"
	"gitlab.com/interview-simulation/interview-backend-server/internal/repositories"
	"gitlab.com/interview-simulation/interview-backend-server/internal/utils"
)

type InterviewSessionService interface {
	CreateInterviewSessionWithNewResume(ctx context.Context, req *entities.CreateInterviewSessionWithNewResumeRequest) (*entities.CreateInterviewSessionWithNewResumeResponse, error)
	CreateInterviewSessionWithExistingResume(ctx context.Context, req *entities.CreateInterviewSessionWithExistingResumeReq) (*entities.CreateInterviewSessionWithExistingResumeResp, error)
	CreateSessionTurnBySessionID(ctx context.Context, req *entities.CreateSessionTurnBySessionIDReq) error
	UpdateInterviewSessionStatus(ctx context.Context, req *entities.UpdateInterviewSessionStatusReq) error
	SetSessionStartTime(ctx context.Context, req *entities.SetSessionStartTimeReq) error
	SetSessionEndTime(ctx context.Context, req *entities.SetSessionEndTimeReq) error
	IsSessionValid(ctx context.Context, req *entities.IsSessionValidReq) (*entities.IsSessionValidResp, error)
}

type interviewSessionService struct {
	log                  *log.Logger
	authContext          middleware.AuthContext
	resumeService        ResumeService
	resumeRepo           repositories.ResumeReposity
	generator            utils.Generator
	interviewSessionRepo repositories.InterviewSessionRepository
	jwtMaker             utils.JwtToken
	config               *config.Config
	s3Storage            aws.S3Storage
	publisher            queue.RedisTaskPublisher
	redisClient          database.RedisClient
}

func NewInterviewSessionService(
	log *log.Logger,
	authContext middleware.AuthContext,
	resumeService ResumeService,
	resumeRepo repositories.ResumeReposity,
	generator utils.Generator,
	interviewSessionRepo repositories.InterviewSessionRepository,
	jwtMaker utils.JwtToken,
	config *config.Config,
	s3Storage aws.S3Storage,
	publisher queue.RedisTaskPublisher,
	redisClient database.RedisClient,
) InterviewSessionService {
	return &interviewSessionService{
		log:                  log,
		authContext:          authContext,
		resumeService:        resumeService,
		resumeRepo:           resumeRepo,
		generator:            generator,
		interviewSessionRepo: interviewSessionRepo,
		jwtMaker:             jwtMaker,
		config:               config,
		s3Storage:            s3Storage,
		publisher:            publisher,
		redisClient:          redisClient,
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
		FileName:   req.File.Filename,
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

	redisPayload := database.RedisPayload{
		Key:   s.generator.GenerateUUID(ctx).String(),
		Value: string(tokenReqJSON),
		TTL:   s.config.InterviewSessionConfig.InterviewSessionDuration,
	}

	err = s.redisClient.Set(ctx, redisPayload)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: CreateInterviewSessionWithNewResume] Error creating interview session token", err)
		return nil, err
	}

	resp := &entities.CreateInterviewSessionWithNewResumeResponse{
		SessionToken: redisPayload.Key,
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
		ID:        sessionID,
		ResumeID:  resumeID,
		UserID:    userID,
		Position:  req.Position,
		Status:    constants.StatusPending,
		Modality:  constants.ModalityVoiceChat,
		IsConsent: req.IsConsent,
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

	redisPayload := database.RedisPayload{
		Key:   s.generator.GenerateUUID(ctx).String(),
		Value: string(tokenReqJSON),
		TTL:   s.config.InterviewSessionConfig.InterviewSessionDuration,
	}

	if err := s.redisClient.Set(ctx, redisPayload); err != nil {
		s.log.ErrorWithID(ctx, "[Service: CreateInterviewSessionWithExistingResume] Error creating interview session token", err)
		return nil, err
	}

	resp := &entities.CreateInterviewSessionWithExistingResumeResp{
		SessionToken: redisPayload.Key,
	}

	return resp, nil
}

func (s *interviewSessionService) CreateSessionTurnBySessionID(ctx context.Context, req *entities.CreateSessionTurnBySessionIDReq) error {
	s.log.InfoWithID(ctx, "[Service: CreateSessionTurnBySessionID] Called")

	redisKey := fmt.Sprintf("%s%s", constants.RedisPrefixInterviewMaxTurnNo, req.SessionID)

	var maxTurnNo int64

	segmentID, err := uuid.Parse(req.TurnID)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: CreateSessionTurnBySessionID] Invalid segment ID", err)
		return app_error.New(err, app_error.ErrCodeGeneralInvalidUUID)
	}

	sessionID, err := uuid.Parse(req.SessionID)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: CreateSessionTurnBySessionID] Invalid session ID", err)
		return app_error.New(err, app_error.ErrCodeGeneralInvalidUUID)
	}

	maxTurnNoStr, err := s.redisClient.Get(context.Background(), redisKey)
	if err == redis.Nil {
		maxTurnNo, err = s.interviewSessionRepo.GetMaxTurnNoBySessionID(context.Background(), sessionID)
		if err != nil {
			s.log.ErrorWithID(ctx, "[Service: CreateSessionTurnBySessionID] Failed to get max turn from DB", err)
			return err
		}
	} else if err != nil {
		s.log.ErrorWithID(ctx, "[Service: CreateSessionTurnBySessionID] Redis error", err)
		return err
	} else {
		maxTurnNo, err = strconv.ParseInt(maxTurnNoStr, 10, 64)
		if err != nil {
			s.log.ErrorWithID(ctx, "[Service: CreateSessionTurnBySessionID] Failed to parse turn no from Redis", err)
			return err
		}
	}

	maxTurnNo++

	redisKey = fmt.Sprintf("%s%s", constants.RedisPrefixInterviewStartEndTime, req.SessionID)
	timeDuration, err := s.redisClient.HGetAll(context.Background(), redisKey)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: CreateSessionTurnBySessionID] Error getting interview session start end time", err)
		return err
	} else if len(timeDuration) == 0 {
		s.log.ErrorWithID(ctx, "[Service: CreateSessionTurnBySessionID] Interview session start end time not found")
		return app_error.New(constants.ErrInterviewSessionStartEndTimeNotFound, app_error.ErrCodeInterviewSessionStartEndTimeNotFound)
	} else if timeDuration["started_at"] == "" {
		s.log.ErrorWithID(ctx, "[Service: CreateSessionTurnBySessionID] Interview session start time not found")
		return app_error.New(constants.ErrInterviewSessionStartTimeNotFound, app_error.ErrCodeInterviewSessionStartTimeNotFound)
	} else if timeDuration["ended_at"] == "" {
		s.log.ErrorWithID(ctx, "[Service: CreateSessionTurnBySessionID] Interview session end time not found")
		return app_error.New(constants.ErrInterviewSessionEndTimeNotFound, app_error.ErrCodeInterviewSessionEndTimeNotFound)
	}

	dbReq := &db.CreateInterviewTurnParams{
		ID:             segmentID,
		SessionID:      sessionID,
		TurnNo:         maxTurnNo,
		Actor:          req.Actor,
		TranscriptText: sql.NullString{String: req.Transcript, Valid: true},
		StartAt:        timeDuration["started_at"],
		EndAt:          timeDuration["ended_at"],
	}

	if err := s.interviewSessionRepo.CreateSessionTurnBySessionID(context.Background(), dbReq); err != nil {
		s.log.ErrorWithID(ctx, "[Service: CreateSessionTurnBySessionID] Error creating session turn by session ID", err)
		return err
	}

	redisPayload := database.RedisPayload{
		Key:   fmt.Sprintf("%s%s", constants.RedisPrefixInterviewMaxTurnNo, req.SessionID),
		Value: maxTurnNo,
		TTL:   constants.RedisTTLInterviewTurn,
	}

	if err := s.redisClient.Set(context.Background(), redisPayload); err != nil {
		s.log.WarnWithID(ctx, "[Service: CreateSessionTurnBySessionID] Error creating interview turn redis key", err)
	}

	return nil
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

	redisSessionToken, err := s.redisClient.Get(ctx, req.SessionToken)
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
