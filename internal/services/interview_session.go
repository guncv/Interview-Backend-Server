package services

import (
	"context"

	"github.com/google/uuid"
	"github.com/sqlc-dev/pqtype"
	"gitlab.com/interview-simulation/interview-backend-server/internal/config"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	db "gitlab.com/interview-simulation/interview-backend-server/internal/db/sqlc"
	"gitlab.com/interview-simulation/interview-backend-server/internal/entities"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
	"gitlab.com/interview-simulation/interview-backend-server/internal/middleware"
	"gitlab.com/interview-simulation/interview-backend-server/internal/repositories"
	"gitlab.com/interview-simulation/interview-backend-server/internal/utils"
)

type InterviewSessionService interface {
	CreateInterviewSessionWithNewResume(ctx context.Context, req *entities.CreateInterviewSessionWithNewResumeRequest) (*entities.CreateInterviewSessionWithNewResumeResponse, error)
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
	}
}

func (s *interviewSessionService) CreateInterviewSessionWithNewResume(ctx context.Context, req *entities.CreateInterviewSessionWithNewResumeRequest) (*entities.CreateInterviewSessionWithNewResumeResponse, error) {
	s.log.InfoWithID(ctx, "[Service: CreateInterviewSessionWithNewResume] Called")

	authCtx, err := s.authContext.GetAuthContext(ctx)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: CreateInterviewSessionWithNewResume] Error getting auth context", err)
		return nil, err
	}

	resumeErrChan := make(chan error, 1)
	resumeChan := make(chan *entities.CreateResumeAndJobRequirementResp, 1)
	summaryJsonChan := make(chan *repositories.GetResumeJsonWithSummaryDataResponse, 1)
	summaryErrChan := make(chan error, 1)

	go func() {
		createResumeReq := &entities.CreateResumeWithRequirementsRequest{
			File:            req.File,
			Position:        req.Position,
			Company:         req.Company,
			WorkType:        req.WorkType,
			JobRequirements: req.JobRequirements,
			InterviewType:   req.InterviewType,
			Language:        req.Language,
		}

		resp, err := s.resumeService.CreateResumeWithRequirements(ctx, createResumeReq)
		if err != nil {
			s.log.ErrorWithID(ctx, "[Service: CreateInterviewSessionWithNewResume] Error creating resume", err)
			resumeErrChan <- err
			return
		}

		resumeChan <- resp
		resumeErrChan <- nil
	}()

	go func() {
		getSummaryJsonReq := &repositories.GetResumeJsonWithSummaryDataReq{
			SessionID:       s.generator.GenerateUUID(ctx),
			Position:        req.Position,
			Company:         req.Company,
			WorkType:        req.WorkType,
			JobRequirements: req.JobRequirements,
			InterviewType:   req.InterviewType,
			Language:        req.Language,
			ResumeFile:      req.File,
		}

		summaryJson, err := s.resumeRepo.GetResumeJsonWithSummaryData(ctx, getSummaryJsonReq)
		if err != nil {
			s.log.ErrorWithID(ctx, "[Service: CreateInterviewSessionWithNewResume] Error getting resume json with summary data", err)
			summaryErrChan <- err
			return
		}
		summaryJsonChan <- summaryJson
		summaryErrChan <- nil
	}()

	resumeErr := <-resumeErrChan
	summaryErr := <-summaryErrChan

	if resumeErr != nil {
		return nil, resumeErr
	}
	if summaryErr != nil {
		return nil, summaryErr
	}

	summaryJson := <-summaryJsonChan
	resume := <-resumeChan

	createInterviewSessionReq := &db.CreateInterviewSessionParams{
		ID:            s.generator.GenerateUUID(ctx),
		UserID:        uuid.MustParse(authCtx.Payload.UserID),
		ResumeID:      uuid.MustParse(resume.ResumeID),
		RequirementID: uuid.MustParse(resume.JobRequirementID),
		PromptJson:    pqtype.NullRawMessage{RawMessage: []byte(summaryJson.ParsedJson), Valid: true},
		Status:        constants.StatusPending,
		Modality:      constants.ModalityVoiceChat,
		ConsentAt:     req.ConsentAt,
	}

	if err := s.interviewSessionRepo.CreateInterviewSession(ctx, createInterviewSessionReq); err != nil {
		s.log.ErrorWithID(ctx, "[Service: CreateInterviewSessionWithNewResume] Error creating interview session", err)
		return nil, err
	}

	createInterviewSessionTokenReq := &entities.CreateInterviewSessionTokenReq{
		SessionID: createInterviewSessionReq.ID,
		UserID:    authCtx.Payload.UserID,
		Duration:  s.config.InterviewSessionConfig.InterviewSessionTokenDuration,
	}

	token, _, err := s.jwtMaker.CreateInterviewSessionToken(ctx, createInterviewSessionTokenReq)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: CreateInterviewSessionWithNewResume] Error creating interview session token", err)
		return nil, err
	}

	resp := &entities.CreateInterviewSessionWithNewResumeResponse{
		SessionToken: token,
	}

	return resp, nil
}
