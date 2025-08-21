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
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/aws"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/database"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/queue"
	"gitlab.com/interview-simulation/interview-backend-server/internal/middleware"
	"gitlab.com/interview-simulation/interview-backend-server/internal/repositories"
	"gitlab.com/interview-simulation/interview-backend-server/internal/utils"
	"golang.org/x/sync/errgroup"
)

type InterviewSessionService interface {
	CreateInterviewSessionWithNewResume(ctx context.Context, req *entities.CreateInterviewSessionWithNewResumeRequest) (*entities.CreateInterviewSessionWithNewResumeResponse, error)
	CreateInterviewSessionWithExistingResume(ctx context.Context, req *entities.CreateInterviewSessionWithExistingResumeReq) (*entities.CreateInterviewSessionWithExistingResumeResp, error)
	StartInterviewSession(ctx context.Context, req *entities.StartInterviewSessionReq) error
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

	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	g, gctx := errgroup.WithContext(ctx)

	var (
		createdResp *entities.CreateResumeAndJobRequirementResp
		summaryJson *repositories.GetResumeJsonWithSummaryDataResponse
	)

	g.Go(func() error {
		key, err := s.s3Storage.UploadFile(gctx, req.File, constants.S3ResumeKey, authCtx.Payload.UserID)
		if err != nil {
			s.log.ErrorWithID(gctx, "[Service: CreateInterviewSessionWithNewResume] Error uploading resume file", err)
			return err
		}

		isDefaultResume, err := s.resumeRepo.CheckIsDefaultResumeExistsByUserID(gctx, uuid.MustParse(authCtx.Payload.UserID))
		if err != nil {
			s.log.ErrorWithID(gctx, "[Service: CreateInterviewSessionWithNewResume] Error checking if default resume exists", err)
			return err
		}

		createResumeAndJobRequirementReq := &repositories.CreateResumeAndJobRequirementReq{
			ResumeID:   s.generator.GenerateUUID(gctx),
			UserID:     uuid.MustParse(authCtx.Payload.UserID),
			FileName:   req.File.Filename,
			StorageKey: key,
			MimeType:   req.File.Header.Get("Content-Type"),
			ByteSize:   int32(req.File.Size),
			IsDefault:  !isDefaultResume,

			JobRequirementID: s.generator.GenerateUUID(gctx),
			Position:         req.Position,
			CompanyName:      req.Company,
			WorkType:         req.WorkType,
			JobRequirements:  req.JobRequirements,
			InterviewType:    req.InterviewType,
			Language:         req.Language,
		}

		if err := s.resumeRepo.CreateResumeAndJobRequirement(gctx, createResumeAndJobRequirementReq); err != nil {
			s.log.ErrorWithID(gctx, "[Service: CreateInterviewSessionWithNewResume] Error creating resume and job requirement", err)
			return err
		}
		createdResp = &entities.CreateResumeAndJobRequirementResp{
			ResumeID:         createResumeAndJobRequirementReq.ResumeID.String(),
			JobRequirementID: createResumeAndJobRequirementReq.JobRequirementID.String(),
		}
		return nil
	})

	g.Go(func() error {
		customFileHeader := s.convertToCustomFileHeader(req.File)

		getSummaryJsonReq := &repositories.GetResumeJsonWithSummaryDataReq{
			SessionID:       s.generator.GenerateUUID(gctx),
			Position:        req.Position,
			Company:         req.Company,
			WorkType:        req.WorkType,
			JobRequirements: req.JobRequirements,
			InterviewType:   req.InterviewType,
			Language:        req.Language,
			ResumeFile:      customFileHeader,
		}
		sj, err := s.resumeRepo.GetResumeJsonWithSummaryData(gctx, getSummaryJsonReq)
		if err != nil {
			s.log.ErrorWithID(gctx, "[Service: CreateInterviewSessionWithNewResume] Error getting resume json with summary data", err)
			return err
		}
		summaryJson = sj
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	promptJsonBytes, err := json.Marshal(summaryJson.ParsedJson)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: CreateInterviewSessionWithNewResume] Error marshalling prompt json", err)
		return nil, err
	}

	params := &db.CreateInterviewSessionParams{
		ID:            s.generator.GenerateUUID(ctx),
		UserID:        uuid.MustParse(authCtx.Payload.UserID),
		ResumeID:      uuid.MustParse(createdResp.ResumeID),
		RequirementID: uuid.MustParse(createdResp.JobRequirementID),
		PromptJson:    pqtype.NullRawMessage{RawMessage: promptJsonBytes, Valid: true},
		Status:        constants.StatusPending,
		Modality:      constants.ModalityVoiceChat,
		ConsentAt:     req.ConsentAt,
	}

	if err := s.interviewSessionRepo.CreateInterviewSession(ctx, params); err != nil {
		s.log.ErrorWithID(ctx, "[Service: CreateInterviewSessionWithNewResume] Error creating interview session", err)
		return nil, err
	}

	tokenReq := map[string]any{
		"session_id": params.ID,
		"user_id":    authCtx.Payload.UserID,
		"role":       authCtx.Payload.Role,
	}

	redisPayload := database.RedisPayload{
		Key:   s.generator.GenerateUUID(ctx).String(),
		Value: tokenReq,
		TTL:   s.config.InterviewSessionConfig.InterviewSessionTokenDuration,
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

	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	g, gctx := errgroup.WithContext(ctx)

	var (
		requirementID uuid.UUID
		summaryJson   *repositories.GetResumeJsonWithSummaryDataResponse
	)

	g.Go(func() error {
		id := s.generator.GenerateUUID(gctx)
		createJobRequirementReq := &db.CreateJobRequirementParams{
			ID:              id,
			UserID:          uuid.MustParse(authCtx.Payload.UserID),
			Position:        req.Position,
			CompanyName:     req.Company,
			WorkType:        req.WorkType,
			JobRequirements: req.JobRequirements,
			InterviewType:   req.InterviewType,
			Language:        req.Language,
		}
		if err := s.jobRequirementRepo.CreateJobRequirement(gctx, createJobRequirementReq); err != nil {
			s.log.ErrorWithID(gctx, "[Service: CreateInterviewSessionWithExistingResume] Error creating job requirement", err)
			return err
		}
		requirementID = id
		return nil
	})

	g.Go(func() error {
		resumeFile, err := s.s3Storage.DownloadFile(gctx, resume.StorageKey)
		if err != nil {
			s.log.ErrorWithID(gctx, "[Service: CreateInterviewSessionWithExistingResume] Error downloading resume file", err)
			return err
		}

		getSummaryJsonReq := &repositories.GetResumeJsonWithSummaryDataReq{
			SessionID:       s.generator.GenerateUUID(gctx),
			Position:        req.Position,
			Company:         req.Company,
			WorkType:        req.WorkType,
			JobRequirements: req.JobRequirements,
			InterviewType:   req.InterviewType,
			Language:        req.Language,
			ResumeFile:      resumeFile,
		}

		sj, err := s.resumeRepo.GetResumeJsonWithSummaryData(gctx, getSummaryJsonReq)
		if err != nil {
			s.log.ErrorWithID(gctx, "[Service: CreateInterviewSessionWithExistingResume] Error getting resume json with summary data", err)
			return err
		}
		summaryJson = sj
		return nil
	})

	if err := g.Wait(); err != nil {
		if requirementID != uuid.Nil {
			delPayload := &entities.DeleteJobRequirementPayload{
				JobRequirementID: requirementID,
			}

			if err := s.publisher.PublishTaskDeleteJobRequirement(ctx, delPayload); err != nil {
				s.log.WarnWithID(ctx, "[Service: CreateInterviewSessionWithExistingResume] Compensation delete job requirement failed", err)
			}
		}
		return nil, err
	}

	promptJsonBytes, err := json.Marshal(summaryJson.ParsedJson)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: CreateInterviewSessionWithExistingResume] Error marshalling prompt json", err)
		return nil, err
	}

	createInterviewSessionParams := &db.CreateInterviewSessionParams{
		ID:            s.generator.GenerateUUID(ctx),
		UserID:        uuid.MustParse(authCtx.Payload.UserID),
		ResumeID:      uuid.MustParse(req.ResumeID),
		RequirementID: requirementID,
		PromptJson:    pqtype.NullRawMessage{RawMessage: promptJsonBytes, Valid: true},
		Status:        constants.StatusPending,
		Modality:      constants.ModalityVoiceChat,
		ConsentAt:     req.ConsentAt,
	}

	if err := s.interviewSessionRepo.CreateInterviewSession(ctx, createInterviewSessionParams); err != nil {
		s.log.ErrorWithID(ctx, "[Service: CreateInterviewSessionWithExistingResume] Error creating interview session", err)
		return nil, err
	}

	tokenReq := map[string]any{
		"session_id": createInterviewSessionParams.ID,
		"user_id":    authCtx.Payload.UserID,
		"role":       authCtx.Payload.Role,
	}

	redisPayload := database.RedisPayload{
		Key:   s.generator.GenerateUUID(ctx).String(),
		Value: tokenReq,
		TTL:   s.config.InterviewSessionConfig.InterviewSessionTokenDuration,
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

func (s *interviewSessionService) StartInterviewSession(ctx context.Context, req *entities.StartInterviewSessionReq) error {
	s.log.InfoWithID(ctx, "[Service: StartInterviewSession] Called")

	dbReq := &db.StartInterviewSessionParams{
		ID:        uuid.MustParse(req.SessionID),
		Status:    constants.StatusOnGoing,
		StartedAt: sql.NullTime{Time: time.Now(), Valid: true},
	}

	if err := s.interviewSessionRepo.StartInterviewSession(ctx, dbReq); err != nil {
		s.log.ErrorWithID(ctx, "[Service: StartInterviewSession] Error starting interview session", err)
		return err
	}

	return nil
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
