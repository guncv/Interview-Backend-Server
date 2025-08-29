package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"mime/multipart"
	"time"

	"github.com/google/uuid"
	"github.com/sqlc-dev/pqtype"
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
	UpdateInterviewSessionStatus(ctx context.Context, req *entities.UpdateInterviewSessionStatusReq) error
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
	jobRequirementRepo   repositories.JobRequirementRepository
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
	jobRequirementRepo repositories.JobRequirementRepository,
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
		jobRequirementRepo:   jobRequirementRepo,
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

	getSummaryJsonReq := &repositories.GetResumeJsonWithSummaryDataReq{
		SessionID:       s.generator.GenerateUUID(ctx),
		Position:        req.Position,
		Company:         req.Company,
		WorkType:        req.WorkType,
		JobRequirements: req.JobRequirements,
		InterviewType:   req.InterviewType,
		Language:        req.Language,
		ResumeFile:      customFileHeader,
	}

	summaryJson, err := s.resumeRepo.GetResumeJsonWithSummaryData(ctx, getSummaryJsonReq)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: CreateInterviewSessionWithNewResume] Error getting resume json with summary data", err)
		return nil, err
	}

	key, err := s.s3Storage.UploadFile(ctx, req.File, constants.S3ResumeKey, authCtx.Payload.UserID)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: CreateInterviewSessionWithNewResume] Error uploading resume file", err)
		return nil, err
	}

	isDefaultResume, err := s.resumeRepo.CheckIsDefaultResumeExistsByUserID(ctx, uuid.MustParse(authCtx.Payload.UserID))
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: CreateInterviewSessionWithNewResume] Error checking if default resume exists", err)
		return nil, err
	}

	promptJsonBytes, err := json.Marshal(summaryJson.ParsedJson)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: CreateInterviewSessionWithNewResume] Error marshalling prompt json", err)
		return nil, err
	}

	createResumeAndJobRequirementReq := &repositories.CreateInterviewSessionTxReq{
		ResumeID:   s.generator.GenerateUUID(ctx),
		UserID:     uuid.MustParse(authCtx.Payload.UserID),
		FileName:   req.File.Filename,
		StorageKey: key,
		MimeType:   req.File.Header.Get("Content-Type"),
		ByteSize:   int32(req.File.Size),
		IsDefault:  !isDefaultResume,

		JobRequirementID: s.generator.GenerateUUID(ctx),
		Position:         req.Position,
		CompanyName:      req.Company,
		WorkType:         req.WorkType,
		JobRequirements:  req.JobRequirements,
		InterviewType:    req.InterviewType,
		Language:         req.Language,

		SessionID:  s.generator.GenerateUUID(ctx),
		PromptJson: pqtype.NullRawMessage{RawMessage: promptJsonBytes, Valid: true},
		Status:     constants.StatusPending,
		Modality:   constants.ModalityVoiceChat,
		IsConsent:  req.IsConsent,
		CreatedAt:  sql.NullTime{Time: time.Now(), Valid: true},
		UpdatedAt:  sql.NullTime{Time: time.Now(), Valid: true},
	}

	if err := s.interviewSessionRepo.CreateInterviewSessionWithNewResumeTx(ctx, createResumeAndJobRequirementReq); err != nil {
		s.log.ErrorWithID(ctx, "[Service: CreateInterviewSessionWithNewResume] Error creating interview session with new resume", err)

		deleteFilePayload := &aws.DeleteFilePayload{
			Key: createResumeAndJobRequirementReq.StorageKey,
		}

		if err := s.publisher.PublishTaskDeleteFile(ctx, deleteFilePayload); err != nil {
			s.log.ErrorWithID(ctx, "[Service: CreateInterviewSessionWithNewResume] Error deleting resume file", err)
		}
		return nil, err
	}

	tokenReq := map[string]any{
		"session_id": createResumeAndJobRequirementReq.SessionID,
		"user_id":    authCtx.Payload.UserID,
		"role":       authCtx.Payload.Role,
		"language":   constants.LanguageMapping[req.Language],
	}

	tokenReqJSON, err := json.Marshal(tokenReq)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: CreateInterviewSessionWithNewResume] Error marshaling token request to JSON", err)
		return nil, err
	}

	redisPayload := database.RedisPayload{
		Key:   s.generator.GenerateUUID(ctx).String(),
		Value: string(tokenReqJSON),
		TTL:   s.config.InterviewSessionConfig.InterviewSessionTokenTTL,
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

	resume, err := s.resumeRepo.GetResumeByID(ctx, uuid.MustParse(req.ResumeID))
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: CreateInterviewSessionWithExistingResume] Error getting resume", err)
		return nil, err
	}

	resumeFile, err := s.s3Storage.DownloadFile(ctx, resume.StorageKey)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: CreateInterviewSessionWithExistingResume] Error downloading resume file", err)
		return nil, err
	}

	getSummaryJsonReq := &repositories.GetResumeJsonWithSummaryDataReq{
		SessionID:       s.generator.GenerateUUID(ctx),
		Position:        req.Position,
		Company:         req.Company,
		WorkType:        req.WorkType,
		JobRequirements: req.JobRequirements,
		InterviewType:   req.InterviewType,
		Language:        req.Language,
		ResumeFile:      resumeFile,
	}

	sj, err := s.resumeRepo.GetResumeJsonWithSummaryData(ctx, getSummaryJsonReq)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: CreateInterviewSessionWithExistingResume] Error getting resume json with summary data", err)
		return nil, err
	}

	promptJsonBytes, err := json.Marshal(sj.ParsedJson)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: CreateInterviewSessionWithExistingResume] Error marshalling prompt json", err)
		return nil, err
	}

	createInterviewSessionWithExistingResumeReq := &repositories.CreateInterviewSessionWithExistingResumeTxReq{
		ResumeID: uuid.MustParse(req.ResumeID),
		UserID:   uuid.MustParse(authCtx.Payload.UserID),

		JobRequirementID: s.generator.GenerateUUID(ctx),
		Position:         req.Position,
		CompanyName:      req.Company,
		WorkType:         req.WorkType,
		JobRequirements:  req.JobRequirements,
		InterviewType:    req.InterviewType,
		Language:         req.Language,
		CreatedAt:        sql.NullTime{Time: time.Now(), Valid: true},
		UpdatedAt:        sql.NullTime{Time: time.Now(), Valid: true},

		SessionID:  s.generator.GenerateUUID(ctx),
		PromptJson: pqtype.NullRawMessage{RawMessage: promptJsonBytes, Valid: true},
		Status:     constants.StatusPending,
		Modality:   constants.ModalityVoiceChat,
		IsConsent:  req.IsConsent,
	}

	if err := s.interviewSessionRepo.CreateInterviewSessionWithExistingResumeTx(ctx, createInterviewSessionWithExistingResumeReq); err != nil {
		s.log.ErrorWithID(ctx, "[Service: CreateInterviewSessionWithExistingResume] Error creating interview session with existing resume", err)
		return nil, err
	}

	tokenReq := map[string]any{
		"session_id": createInterviewSessionWithExistingResumeReq.SessionID,
		"user_id":    authCtx.Payload.UserID,
		"language":   constants.LanguageMapping[req.Language],
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
		TTL:   s.config.InterviewSessionConfig.InterviewSessionTokenTTL,
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

func (s *interviewSessionService) UpdateInterviewSessionStatus(ctx context.Context, req *entities.UpdateInterviewSessionStatusReq) error {
	s.log.InfoWithID(ctx, "[Service: UpdateInterviewSessionStatus] Called")

	dbReq := &db.UpdateInterviewSessionStatusParams{
		ID:      uuid.MustParse(req.SessionID),
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

	exists, err := s.interviewSessionRepo.CheckInterviewSessionExists(ctx, uuid.MustParse(sessionPayload.SessionID))
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
		Language:  sessionPayload.Language,
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
