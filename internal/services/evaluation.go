package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	db "gitlab.com/interview-simulation/interview-backend-server/internal/db/sqlc"
	"gitlab.com/interview-simulation/interview-backend-server/internal/entities"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/database"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
	"gitlab.com/interview-simulation/interview-backend-server/internal/repositories"
	"gitlab.com/interview-simulation/interview-backend-server/internal/utils"
)

type EvaluationService interface {
	GetRubricWithCriteriaByName(ctx context.Context, rubricName string) (*entities.GetRubricWithCriteriaByNameResp, error)
	ListAllRubricsAndCriteria(ctx context.Context) (*entities.ListAllRubricsAndCriteriaResp, error)
	CalculateTurnScore(ctx context.Context, req *entities.CalculateTurnScoreReq) error
}

type evaluationService struct {
	log                   *log.Logger
	redisClient           database.RedisClient
	evaluationRubricsRepo repositories.EvaluationRubricsRepository
	evaluationScoresRepo  repositories.EvaluationScoresRepository
	interviewTurnsRepo    repositories.InterviewTurnsRepository
	generator             utils.Generator
}

func NewEvaluationService(
	log *log.Logger,
	redisClient database.RedisClient,
	evaluationRubricsRepo repositories.EvaluationRubricsRepository,
	evaluationScoresRepo repositories.EvaluationScoresRepository,
	interviewTurnsRepo repositories.InterviewTurnsRepository,
	generator utils.Generator,
) EvaluationService {
	return &evaluationService{
		log:                   log,
		evaluationRubricsRepo: evaluationRubricsRepo,
		redisClient:           redisClient,
		evaluationScoresRepo:  evaluationScoresRepo,
		interviewTurnsRepo:    interviewTurnsRepo,
		generator:             generator,
	}
}

func (s *evaluationService) GetRubricWithCriteriaByName(ctx context.Context, rubricName string) (*entities.GetRubricWithCriteriaByNameResp, error) {
	s.log.InfoWithID(ctx, "[Service: GetRubricWithCriteriaByName] Called")

	var resp entities.GetRubricWithCriteriaByNameResp
	redisKey := fmt.Sprintf("%s:%s", constants.RedisPrefixEvaluationRubric, rubricName)

	redisValue, err := s.redisClient.Get(ctx, redisKey)
	if err != nil {
		s.log.WarnWithID(ctx, "[Service: GetRubricWithCriteriaByName] Redis miss or error, falling back to DB", err)

		criteriaNameMapping := constants.StateToCriteriaStateMap[rubricName]
		req := &db.GetRubricWithCriteriaByNameParams{
			Name:         criteriaNameMapping,
			VersionLabel: constants.CurrentCriteriaVersion,
		}

		dbRows, err := s.evaluationRubricsRepo.GetRubricWithCriteriaByName(ctx, req)
		if err != nil {
			s.log.ErrorWithID(ctx, "[Service: GetRubricWithCriteriaByName] DB error after Redis fail", err)
			return nil, err
		}

		if len(dbRows) == 0 {
			s.log.ErrorWithID(ctx, "[Service: GetRubricWithCriteriaByName] No rubric/criteria found in DB")
			return nil, app_error.New(errors.New("no rubric/criteria found"), app_error.ErrCodeEvaluationRubricCriteriaNotFound)
		}

		criteria := make([]entities.CritetiaRow, 0, len(dbRows))
		for _, row := range dbRows {
			criteria = append(criteria, entities.CritetiaRow{
				CriterionID:            row.CriterionID.String(),
				CriterionCode:          row.CriterionCode,
				CriterionName:          row.CriterionName,
				CriterionDescriptionMd: row.CriterionDescriptionMd.String,
				CriterionWeight:        row.CriterionWeight,
				CriterionMaxScore:      row.CriterionMaxScore,
			})
		}

		resp = entities.GetRubricWithCriteriaByNameResp{
			RubricID:            dbRows[0].RubricID.String(),
			RubricName:          dbRows[0].RubricName,
			RubricDescriptionMd: dbRows[0].RubricDescriptionMd.String,
			RubricVersionLabel:  dbRows[0].RubricVersionLabel,
			Criteria:            criteria,
		}

		go func() {
			cacheCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			if jsonBytes, err := json.Marshal(resp); err == nil {
				_ = s.redisClient.Set(cacheCtx, database.RedisPayload{
					Key:   redisKey,
					Value: jsonBytes,
					TTL:   constants.RedisTTLEvaluationRubric,
				})
			}
		}()

		return &resp, nil
	}

	s.log.InfoWithID(ctx, "[Service: GetRubricWithCriteriaByName] Cache hit")
	if err := json.Unmarshal([]byte(redisValue), &resp); err != nil {
		s.log.ErrorWithID(ctx, "[Service: GetRubricWithCriteriaByName] Failed to unmarshal Redis value", err)
		return nil, app_error.New(err, app_error.ErrCodeGeneralUnmarshalFailed)
	}

	return &resp, nil
}

func (s *evaluationService) ListAllRubricsAndCriteria(ctx context.Context) (*entities.ListAllRubricsAndCriteriaResp, error) {
	s.log.InfoWithID(ctx, "[Service: ListAllRubricsAndCriteria] Called")

	var resp entities.ListAllRubricsAndCriteriaResp
	redisKey := constants.RedisPrefixAllRubricsAndCriteria

	redisValue, err := s.redisClient.Get(ctx, redisKey)
	if err != nil {
		s.log.WarnWithID(ctx, "[Service: ListAllRubricsAndCriteria] Redis miss or error, falling back to DB", err)

		resp, err := s.fetchAllRubricsAndCriteriaFromDB(ctx)
		if err != nil {
			s.log.ErrorWithID(ctx, "[Service: ListAllRubricsAndCriteria] Error getting all rubrics and criteria", err)
			return nil, err
		}
		return resp, nil
	}

	if err := json.Unmarshal([]byte(redisValue), &resp); err != nil {
		s.log.WarnWithID(ctx, "[Service: ListAllRubricsAndCriteria] Failed to unmarshal Redis value, falling back to DB", err)

		resp, err := s.fetchAllRubricsAndCriteriaFromDB(ctx)
		if err != nil {
			s.log.ErrorWithID(ctx, "[Service: ListAllRubricsAndCriteria] Error getting all rubrics and criteria", err)
			return nil, err
		}
		return resp, nil
	}

	return &resp, nil
}

func (s *evaluationService) fetchAllRubricsAndCriteriaFromDB(ctx context.Context) (*entities.ListAllRubricsAndCriteriaResp, error) {
	s.log.InfoWithID(ctx, "[Service: fetchAllRubricsAndCriteriaFromDB] Fetching all rubrics and criteria from DB")

	redisKey := constants.RedisPrefixAllRubricsAndCriteria

	dbRows, err := s.evaluationRubricsRepo.ListAllRubricsAndCriteria(ctx, constants.CurrentCriteriaVersion)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: fetchAllRubricsAndCriteriaFromDB] Error getting all rubrics and criteria", err)
		return nil, err
	}

	if len(dbRows) == 0 {
		s.log.InfoWithID(ctx, "[Service: fetchAllRubricsAndCriteriaFromDB] No rubrics found")

		resp := entities.ListAllRubricsAndCriteriaResp{
			Rubrics: []entities.RubricAndCriteriaRow{},
		}

		return &resp, nil
	}

	rubricMap := make(map[string]*entities.RubricAndCriteriaRow)

	for _, row := range dbRows {
		rubricID := row.RubricID.String()

		if _, exists := rubricMap[rubricID]; !exists {
			rubricMap[rubricID] = &entities.RubricAndCriteriaRow{
				ID:            row.RubricID.String(),
				Name:          row.RubricName,
				DescriptionMd: row.RubricDescriptionMd.String,
				Criteria:      []entities.CriterionRow{},
			}
		}

		percentage := utils.ConvertWeightToPercentage(row.CriterionWeight)
		color := utils.GetPercentageColor(percentage)

		criterion := entities.CriterionRow{
			ID:            row.CriterionID.String(),
			Name:          row.CriterionName,
			DescriptionMd: row.CriterionDescriptionMd.String,
			Percentage:    percentage,
			Color:         color,
		}

		rubricMap[rubricID].Criteria = append(rubricMap[rubricID].Criteria, criterion)
	}

	result := make([]entities.RubricAndCriteriaRow, 0, len(rubricMap))
	for _, rubricAndCriteria := range rubricMap {
		result = append(result, *rubricAndCriteria)
	}

	resp := entities.ListAllRubricsAndCriteriaResp{
		Rubrics: result,
	}

	go func() {
		cacheCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		if jsonBytes, err := json.Marshal(resp); err == nil {
			_ = s.redisClient.Set(cacheCtx, database.RedisPayload{
				Key:   redisKey,
				Value: jsonBytes,
				TTL:   constants.RedisTTLEvaluationRubric,
			})
		}
	}()

	return &resp, nil
}

func (s *evaluationService) CalculateTurnScore(ctx context.Context, req *entities.CalculateTurnScoreReq) error {
	s.log.InfoWithID(ctx, "[Service: CalculateTurnScore] Called")

	rubric, err := s.GetRubricWithCriteriaByName(ctx, req.CurrentState)
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

	result, err := s.evaluationScoresRepo.InterviewFeedbackAndScore(ctx, interviewFeedbackAndScoreReq)
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
			ID:            s.generator.GenerateUUID(ctx),
			CriterionID:   criterionID,
			CriterionName: criterion.CriterionName,
			Score:         criterion.CriterionScore,
			CommentMd:     criterion.CriterionFeedback,
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

	if lastErr != nil {
		s.log.ErrorWithID(ctx, "[Service: CalculateTurnScore] Error creating evaluation and score", lastErr)
		return lastErr
	}

	if err := s.interviewTurnsRepo.FlagIsScoreEvaluated(ctx, userTurnID); err != nil {
		s.log.ErrorWithID(ctx, "[Service: CalculateTurnScore] Error flagging is score evaluated", err)
		return err
	}

	redisKey := fmt.Sprintf("%s%s:%s", constants.RedisPrefixInterviewIsScoreSessionState, req.SessionID, req.CurrentState)
	redisPayload := database.RedisPayload{
		Key:   redisKey,
		Value: true,
		TTL:   constants.RedisTTLInterviewIsScoreSessionState,
	}
	_ = s.redisClient.Set(ctx, redisPayload)

	return nil
}
