package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"strconv"
	"time"

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
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/queue/publisher"
	"gitlab.com/interview-simulation/interview-backend-server/internal/middleware"
	"gitlab.com/interview-simulation/interview-backend-server/internal/repositories"
	"gitlab.com/interview-simulation/interview-backend-server/internal/utils"
)

type InterviewSessionService interface {
	CreateInterviewSessionWithNewResume(ctx context.Context, req *entities.CreateInterviewSessionWithNewResumeRequest) (*entities.CreateInterviewSessionWithNewResumeResponse, error)
	CreateInterviewSessionWithExistingResume(ctx context.Context, req *entities.CreateInterviewSessionWithExistingResumeReq) (*entities.CreateInterviewSessionWithExistingResumeResp, error)
	CreateUserSessionTurnBySessionID(ctx context.Context, req *entities.CreateUserSessionTurnBySessionIDReq) error
	CreateInterviewerSessionTurnBySessionID(ctx context.Context, req *entities.CreateInterviewerSessionTurnBySessionIDReq) error
	UpdateInterviewSessionStatus(ctx context.Context, req *entities.UpdateInterviewSessionStatusReq) error
	SetSessionStartTime(ctx context.Context, req *entities.SetSessionStartTimeReq) error
	SetSessionEndTime(ctx context.Context, req *entities.SetSessionEndTimeReq) error
	IsSessionValid(ctx context.Context, req *entities.IsSessionValidReq) (*entities.IsSessionValidResp, error)
	GetInterviewerLastMessage(ctx context.Context, req *entities.GetInterviewerLastMessageReq) (*entities.GetInterviewerLastMessageResp, error)
	CalculateTurnScore(ctx context.Context, req *entities.CalculateTurnScoreReq) error
	GetChatHistoryBySessionToken(ctx context.Context, req *entities.GetChatHistoryBySessionTokenReq) (*entities.GetChatHistoryBySessionTokenResp, error)
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
	publisher            publisher.RedisTaskPublisher
	redisClient          database.RedisClient
	evaluationService    EvaluationService
	evaluationScoresRepo repositories.EvaluationScoresRepository
	interviewTurnsRepo   repositories.InterviewTurnsRepository
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
	publisher publisher.RedisTaskPublisher,
	redisClient database.RedisClient,
	evaluationService EvaluationService,
	evaluationScoresRepo repositories.EvaluationScoresRepository,
	interviewTurnsRepo repositories.InterviewTurnsRepository,
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
		evaluationService:    evaluationService,
		evaluationScoresRepo: evaluationScoresRepo,
		interviewTurnsRepo:   interviewTurnsRepo,
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

	s.cacheMaxTurnNo(context.Background(), req.SessionID, maxTurnNo)
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

	redisKey = fmt.Sprintf("%s%s", constants.RedisPrefixInterviewStartEndTime, req.SessionID)
	timeDuration, err := s.redisClient.HGetAll(context.Background(), redisKey)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: CreateUserSessionTurnBySessionID] Error getting interview session start end time", err)
		return err
	} else if len(timeDuration) == 0 {
		s.log.ErrorWithID(ctx, "[Service: CreateUserSessionTurnBySessionID] Interview session start end time not found")
		return app_error.New(constants.ErrInterviewSessionStartEndTimeNotFound, app_error.ErrCodeInterviewSessionStartEndTimeNotFound)
	} else if timeDuration["started_at"] == "" {
		s.log.ErrorWithID(ctx, "[Service: CreateUserSessionTurnBySessionID] Interview session start time not found")
		return app_error.New(constants.ErrInterviewSessionStartTimeNotFound, app_error.ErrCodeInterviewSessionStartTimeNotFound)
	} else if timeDuration["ended_at"] == "" {
		s.log.ErrorWithID(ctx, "[Service: CreateUserSessionTurnBySessionID] Interview session end time not found")
		return app_error.New(constants.ErrInterviewSessionEndTimeNotFound, app_error.ErrCodeInterviewSessionEndTimeNotFound)
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

	s.cacheMaxTurnNo(context.Background(), req.SessionID, maxTurnNo)
	return nil
}

func (s *interviewSessionService) increaseMaxTurnNo(ctx context.Context, sessionID uuid.UUID, redisKey string) (int64, error) {
	s.log.InfoWithID(ctx, "[Service: IncreaseMaxTurnNo] Called")

	var maxTurnNo int64
	maxTurnNoStr, err := s.redisClient.Get(context.Background(), redisKey)

	if err == redis.Nil {
		maxTurnNo, err = s.interviewTurnsRepo.GetMaxTurnNoBySessionID(context.Background(), sessionID)
		if err != nil {
			s.log.ErrorWithID(ctx, "[Service: IncreaseMaxTurnNo] Failed to get max turn from DB", err)
			return 0, err
		}
	} else if err != nil {
		s.log.ErrorWithID(ctx, "[Service: IncreaseMaxTurnNo] Redis error", err)
		return 0, err
	} else {
		maxTurnNo, err = strconv.ParseInt(maxTurnNoStr, 10, 64)
		if err != nil {
			s.log.ErrorWithID(ctx, "[Service: IncreaseMaxTurnNo] Failed to parse turn no from Redis", err)
			return 0, err
		}
	}

	maxTurnNo++
	return maxTurnNo, nil
}

func (s *interviewSessionService) cacheMaxTurnNo(ctx context.Context, sessionID string, maxTurnNo int64) {
	s.log.InfoWithID(ctx, "[Service: cacheMaxTurnNo] Called")

	redisPayload := database.RedisPayload{
		Key:   fmt.Sprintf("%s%s", constants.RedisPrefixInterviewMaxTurnNo, sessionID),
		Value: maxTurnNo,
		TTL:   constants.RedisTTLInterviewTurn,
	}

	if err := s.redisClient.Set(ctx, redisPayload); err != nil {
		s.log.WarnWithID(ctx, "[Service: cacheMaxTurnNo] Failed to cache", err)
	}
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

func (s *interviewSessionService) CalculateTurnScore(ctx context.Context, req *entities.CalculateTurnScoreReq) error {
	s.log.InfoWithID(ctx, "[Service: CalculateTurnScore] Called")

	rubric, err := s.evaluationService.GetRubricWithCriteriaByName(ctx, req.CurrentState)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: CalculateTurnScore] Error getting rubric with criteria by name", err)
		return err
	}

	interviewFeedbackAndScoreReq := &repositories.InterviewFeedbackAndScoreReq{
		UserMessage:         req.UserMessage,
		InterviewerMessage:  req.InterviewerMessage,
		RubricName:          rubric.RubricName,
		RubricDescriptionMd: rubric.RubricDescriptionMd,
		Criteria:            rubric.Criteria,
	}

	result, err := s.interviewSessionRepo.InterviewFeedbackAndScore(ctx, interviewFeedbackAndScoreReq)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: CalculateTurnScore] Error interviewing feedback and score", err)
		return err
	}

	evaluationID := s.generator.GenerateUUID(ctx)

	sessionID, err := uuid.Parse(req.SessionID)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: CalculateTurnScore] Invalid session ID", err)
		return app_error.New(err, app_error.ErrCodeGeneralInvalidUUID)
	}

	userTurnID, err := uuid.Parse(req.UserTurnID)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: CalculateTurnScore] Invalid user turn ID", err)
		return app_error.New(err, app_error.ErrCodeGeneralInvalidUUID)
	}

	rubricID, err := uuid.Parse(rubric.RubricID)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: CalculateTurnScore] Invalid rubric ID", err)
		return app_error.New(err, app_error.ErrCodeGeneralInvalidUUID)
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: CalculateTurnScore] Invalid user ID", err)
		return app_error.New(err, app_error.ErrCodeGeneralInvalidUUID)
	}

	criteria := make([]repositories.CreateScoreTxReq, len(result.CriteriaScores))
	for i, criterion := range result.CriteriaScores {
		criterionID, err := uuid.Parse(criterion.CriterionID)
		if err != nil {
			s.log.ErrorWithID(ctx, "[Service: CalculateTurnScore] Invalid criterion ID", err)
			return app_error.New(err, app_error.ErrCodeGeneralInvalidUUID)
		}

		criteria[i] = repositories.CreateScoreTxReq{
			ID:          s.generator.GenerateUUID(ctx),
			CriterionID: criterionID,
			Score:       criterion.CriterionScore,
			CommentMd:   criterion.CriterionFeedback,
		}
	}

	createEvaluationAndScoreTxReq := &repositories.CreateEvaluationAndScoreTxReq{
		EvaluationID: evaluationID,
		SessionID:    sessionID,
		TurnID:       userTurnID,
		RubricID:     rubricID,
		UserID:       userID,
		CurrentState: req.CurrentState,
		OverallScore: fmt.Sprintf("%f", result.OverallScore),
		SummaryMd:    result.OverallFeedback,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Criteria:     criteria,

		ImproveSentenceID: s.generator.GenerateUUID(ctx),
		ImproveSentence:   result.ImproveSentence,
		LLmModel:          result.LLmModel,
	}

	var lastErr error
	for attempt := 1; attempt <= constants.MaxRetryDbEvaluationTx; attempt++ {
		err := s.evaluationScoresRepo.CreateEvaluationWithCriteriaScoreAndImproveSentenceTx(ctx, createEvaluationAndScoreTxReq)
		if err == nil {
			return nil
		}

		s.log.WarnWithID(ctx, fmt.Sprintf("[Service: CalculateTurnScore] Attempt %d/%d failed: %v", attempt, constants.MaxRetryDbEvaluationTx, err))
		lastErr = err

		time.Sleep(constants.RetryDelayDbEvaluationTx * time.Duration(attempt))
	}

	return lastErr
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
			CreatedAt:      utils.FormatToBangkokTime(chat.CreatedAt),
		}
	}

	resp := &entities.GetChatHistoryBySessionTokenResp{
		ChatHistory: chatHistoryResp,
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
