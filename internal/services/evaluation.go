package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
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
	CalculateEvaluationInOldState(ctx context.Context, req *entities.CalculateEvaluationInOldStateReq) error
	GetPhraseEvaluationsWithCriteriaBySessionID(ctx context.Context, sessionID string) (*entities.GetPhraseEvaluationsWithCriteriaResp, error)
	FinalizeSessionPhraseEvaluation(ctx context.Context, sessionID string) error
	FinalizeSessionFailed(ctx context.Context, sessionID string)
	UpdateFinalizeStatusSessionEvaluation(ctx context.Context, sessionID string, finalizeStatus db.FinalizeStatusEnum) error
}

type evaluationService struct {
	log                   *log.Logger
	redisClient           database.RedisClient
	evaluationRubricsRepo repositories.EvaluationRubricsRepository
	evaluationScoresRepo  repositories.EvaluationScoresRepository
	interviewTurnsRepo    repositories.InterviewTurnsRepository
	interviewSessionsRepo repositories.InterviewSessionRepository
	interviewStatesRepo   repositories.InterviewStateRepository
	generator             utils.Generator
}

func NewEvaluationService(
	log *log.Logger,
	redisClient database.RedisClient,
	evaluationRubricsRepo repositories.EvaluationRubricsRepository,
	evaluationScoresRepo repositories.EvaluationScoresRepository,
	interviewTurnsRepo repositories.InterviewTurnsRepository,
	interviewSessionsRepo repositories.InterviewSessionRepository,
	interviewStatesRepo repositories.InterviewStateRepository,
	generator utils.Generator,
) EvaluationService {
	return &evaluationService{
		log:                   log,
		evaluationRubricsRepo: evaluationRubricsRepo,
		redisClient:           redisClient,
		evaluationScoresRepo:  evaluationScoresRepo,
		interviewTurnsRepo:    interviewTurnsRepo,
		interviewSessionsRepo: interviewSessionsRepo,
		interviewStatesRepo:   interviewStatesRepo,
		generator:             generator,
	}
}

func (s *evaluationService) GetRubricWithCriteriaByName(ctx context.Context, rubricName string) (*entities.GetRubricWithCriteriaByNameResp, error) {

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

	if err := json.Unmarshal([]byte(redisValue), &resp); err != nil {
		s.log.ErrorWithID(ctx, "[Service: GetRubricWithCriteriaByName] Failed to unmarshal Redis value", err)
		return nil, app_error.New(err, app_error.ErrCodeGeneralUnmarshalFailed)
	}

	return &resp, nil
}

func (s *evaluationService) ListAllRubricsAndCriteria(ctx context.Context) (*entities.ListAllRubricsAndCriteriaResp, error) {

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

	redisKey := constants.RedisPrefixAllRubricsAndCriteria

	dbRows, err := s.evaluationRubricsRepo.ListAllRubricsAndCriteria(ctx, constants.CurrentCriteriaVersion)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: fetchAllRubricsAndCriteriaFromDB] Error getting all rubrics and criteria", err)
		return nil, err
	}

	if len(dbRows) == 0 {

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
		OverallScore: utils.FormatFloatToTwoDecimals(result.OverallScore),
		SummaryMd:    result.OverallFeedback,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Criteria:     criteria,

		ImproveSentenceID: s.generator.GenerateUUID(ctx),
		ImproveSentence:   result.ImproveSentence,
		LLmModel:          result.LLmModel,
	}

	if err := s.evaluationScoresRepo.CreateEvaluationWithCriteriaScoreAndImproveSentenceTx(ctx, createEvaluationAndScoreTxReq); err != nil {
		s.log.ErrorWithID(ctx, "[Service: CalculateTurnScore] Error creating evaluation and score", err)
		return err
	}

	if err := s.interviewTurnsRepo.FlagIsScoreEvaluated(ctx, userTurnID); err != nil {
		s.log.ErrorWithID(ctx, "[Service: CalculateTurnScore] Error flagging is score evaluated", err)
		return err
	}

	redisKey := fmt.Sprintf("%s%s:%s", constants.RedisPrefixInterviewLastTurnID, req.SessionID, req.CurrentState)
	const maxRedisRetries = 3
	const retryDelay = 300 * time.Millisecond

	var lastTurnID string
	for i := 0; i < maxRedisRetries; i++ {
		lastTurnID, err = s.redisClient.Get(ctx, redisKey)
		if err == nil {
			break
		}
		if errors.Is(err, redis.Nil) {
			s.log.WarnWithID(ctx, fmt.Sprintf("[Service: CalculateTurnScore] Redis key not found, retrying... (%d/%d)", i+1, maxRedisRetries))
			time.Sleep(retryDelay * time.Duration(i+1))
			continue
		}

		s.log.ErrorWithID(ctx, "[Service: CalculateTurnScore] Error getting last turn ID from redis", err)
		return err
	}

	if lastTurnID == "" {
		s.log.WarnWithID(ctx, "[Service: CalculateTurnScore] Last turn ID not found in Redis after retries, skipping evaluation trigger")
		return nil
	}

	if lastTurnID == req.UserTurnID {
		req := entities.CalculateEvaluationInOldStateReq{
			SessionID:        req.SessionID,
			CurrentState:     req.CurrentState,
			InterviewStateID: req.CurrentStateID,
		}
		_ = s.CalculateEvaluationInOldState(ctx, &req)
	} else {
		s.log.WarnWithID(ctx, "[Service: CalculateTurnScore] Last turn ID is not the same as user turn ID", req.UserTurnID, " ", lastTurnID)
	}

	if err := s.redisClient.Delete(ctx, redisKey); err != nil {
		s.log.ErrorWithID(ctx, "[Service: CalculateTurnScore] Error deleting last turn ID from redis", err)
		return err
	}

	return nil
}

func (s *evaluationService) CalculateEvaluationInOldState(ctx context.Context, req *entities.CalculateEvaluationInOldStateReq) error {

	sessionID, err := uuid.Parse(req.SessionID)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: CalculateEvaluationInOldState] Invalid session ID", err)
		return app_error.New(err, app_error.ErrCodeGeneralInvalidUUID)
	}

	interviewStateID, err := uuid.Parse(req.InterviewStateID)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: CalculateEvaluationInOldState] Invalid interview state ID", err)
		return app_error.New(err, app_error.ErrCodeGeneralInvalidUUID)
	}

	dbReq := &db.GetEvaluationSummaryJsonBySessionAndStateParams{
		SessionID:    sessionID,
		CurrentState: req.CurrentState,
	}

	resp, err := s.evaluationScoresRepo.GetEvaluationSummaryJsonBySessionAndState(ctx, dbReq)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.log.WarnWithID(ctx, "[Service: CalculateEvaluationInOldState] Evaluation summary json by session and state not found", err)
			return nil
		}
		s.log.ErrorWithID(ctx, "[Service: CalculateEvaluationInOldState] Error getting evaluation summary json by session and state", err)
		return err
	}

	var respOld entities.GetEvaluationSummaryJsonBySessionAndStateResp
	if err := json.Unmarshal(resp, &respOld); err != nil {
		s.log.ErrorWithID(ctx, "[Service: CalculateEvaluationInOldState] Error unmarshalling evaluation summary json", err)
		return app_error.New(err, app_error.ErrCodeGeneralUnmarshalFailed)
	}

	var preProcessedCriteria repositories.PreProcessedCriteriaReq
	for _, criteria := range respOld.Criteria {
		var comments []string
		var totalScore float64
		var scoreCount int

		for _, score := range criteria.CriteriaScore {
			if score.Comment != "" {
				comments = append(comments, score.Comment)
			}
			totalScore += float64(score.Score)
			scoreCount++
		}

		preProcessedCriteria.Criteria = append(preProcessedCriteria.Criteria, repositories.PreProcessedCriteria{
			CriteriaID:       criteria.CriteriaID,
			CriteriaName:     criteria.CriteriaName,
			CriteriaAvgScore: totalScore / float64(scoreCount),
			CriteriaComment:  comments,
		})
	}

	postProcessedCriteriaResp, err := s.evaluationScoresRepo.CalculateEachCriteriaCommentBySessionAndState(ctx, &preProcessedCriteria)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: CalculateEvaluationInOldState] Error calculating each criteria comment by session and state", err)
		return err
	}

	evaluationID := s.generator.GenerateUUID(ctx)

	var criteriaReqTx repositories.CreateCriteriaReqTx
	for _, criteria := range postProcessedCriteriaResp.Criteria {

		newId := s.generator.GenerateUUID(ctx)
		criteriaID, err := uuid.Parse(criteria.CriteriaID)
		if err != nil {
			s.log.ErrorWithID(ctx, "[Service: CalculateEvaluationInOldState] Invalid criteria ID", err)
			return app_error.New(err, app_error.ErrCodeGeneralInvalidUUID)
		}

		criteriaReqTx.ID = append(criteriaReqTx.ID, newId)
		criteriaReqTx.EvaluationID = append(criteriaReqTx.EvaluationID, evaluationID)
		criteriaReqTx.CriteriaID = append(criteriaReqTx.CriteriaID, criteriaID)
		criteriaReqTx.CriteriaName = append(criteriaReqTx.CriteriaName, criteria.CriteriaName)
		criteriaReqTx.CriteriaAvgScore = append(criteriaReqTx.CriteriaAvgScore, utils.RoundFloatToTwoDecimals(criteria.CriteriaAvgScore))
		criteriaReqTx.CriteriaComment = append(criteriaReqTx.CriteriaComment, criteria.CriteriaComment)
	}

	dbReqTx := &repositories.CreatePhraseEvaluationAndCriteriaScoreWithIsScoredStateReqTx{
		EvaluationID: evaluationID,
		SessionID:    sessionID,
		StateID:      interviewStateID,
		StateName:    req.CurrentState,
		OverallScore: utils.RoundFloatToTwoDecimals(respOld.EvaluationAvgScore),
		Criteria:     criteriaReqTx,
	}

	if err := s.evaluationScoresRepo.CreatePhraseEvaluationAndCriteriaScoreWithIsScoredState(ctx, dbReqTx); err != nil {
		s.log.ErrorWithID(ctx, "[Service: CalculateEvaluationInOldState] Error creating phrase evaluation and criteria score", err)
		return err
	}

	return nil
}

func (s *evaluationService) FinalizeSessionPhraseEvaluation(ctx context.Context, sessionIDReq string) error {

	sessionID, err := uuid.Parse(sessionIDReq)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: FinalizeSessionEvaluation] Invalid session ID", err)
		s.FinalizeSessionFailed(ctx, sessionIDReq)
		return app_error.New(err, app_error.ErrCodeGeneralInvalidUUID)
	}

	states, err := s.interviewStatesRepo.GetUnprocessedInterviewStatesBySessionID(ctx, sessionID)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: FinalizeSessionEvaluation] Error getting unprocessed interview states by session ID", err)
		s.FinalizeSessionFailed(ctx, sessionIDReq)
		return err
	}

	for _, state := range states {

		err := s.CalculateEvaluationInOldState(ctx, &entities.CalculateEvaluationInOldStateReq{
			SessionID:        sessionIDReq,
			CurrentState:     state.PhraseType,
			InterviewStateID: state.ID.String(),
		})
		if err != nil {
			s.log.ErrorWithID(ctx, "[Finalize] Failed evaluation", err)
			s.FinalizeSessionFailed(ctx, sessionIDReq)
			return err
		}
	}

	if err := s.UpdateFinalizeStatusSessionEvaluation(ctx, sessionIDReq, db.FinalizeStatusEnumFinalized); err != nil {
		s.log.ErrorWithID(ctx, "[Service: FinalizeSessionEvaluation] Error updating finalize status interview session by ID", err)
		s.FinalizeSessionFailed(ctx, sessionIDReq)
		return err
	}

	return nil
}

func (s *evaluationService) FinalizeSessionFailed(ctx context.Context, sessionIDReq string) {

	if err := s.UpdateFinalizeStatusSessionEvaluation(ctx, sessionIDReq, db.FinalizeStatusEnumFailed); err != nil {
		s.log.ErrorWithID(ctx, "[Service: FinalizeSessionFailed] Error updating finalize status interview session by ID", err)
		return
	}
}

func (s *evaluationService) UpdateFinalizeStatusSessionEvaluation(ctx context.Context, sessionIDReq string, finalizeStatus db.FinalizeStatusEnum) error {

	sessionID, err := uuid.Parse(sessionIDReq)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: UpdateFinalizeStatusSessionEvaluation] Invalid session ID", err)
		return app_error.New(err, app_error.ErrCodeGeneralInvalidUUID)
	}

	dbReq := &db.UpdateFinalizeStatusInterviewSessionByIDParams{
		ID: sessionID,
		FinalizeStatus: db.NullFinalizeStatusEnum{
			FinalizeStatusEnum: finalizeStatus,
			Valid:              true,
		},
	}

	if err := s.interviewSessionsRepo.UpdateFinalizeStatusInterviewSessionByID(ctx, dbReq); err != nil {
		s.log.ErrorWithID(ctx, "[Service: UpdateFinalizeStatusSessionEvaluation] Error updating finalize status interview session by ID", err)
		return err
	}

	return nil
}

func (s *evaluationService) GetPhraseEvaluationsWithCriteriaBySessionID(ctx context.Context, sessionIdReq string) (*entities.GetPhraseEvaluationsWithCriteriaResp, error) {

	sessionID, err := uuid.Parse(sessionIdReq)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: GetPhraseEvaluationsWithCriteriaBySessionID] Invalid session ID", err)
		return nil, app_error.New(err, app_error.ErrCodeGeneralInvalidUUID)
	}

	dbResp, err := s.evaluationScoresRepo.GetPhraseEvaluationsWithCriteriaBySessionID(ctx, sessionID)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: GetPhraseEvaluationsWithCriteriaBySessionID] Error getting phrase evaluations with criteria by session ID", err)
		return nil, err
	}

	var respEntities = make([]entities.PhraseEvaluations, 0, len(dbResp))
	for _, row := range dbResp {
		var respEntity entities.PhraseEvaluations
		respEntity.StateID = row.StateID.String()
		respEntity.StateName = row.StateName
		respEntity.OverallScore = utils.RoundFloatToTwoDecimals(row.OverallScore)
		respEntity.OverallColor = utils.GetScoreColor(row.OverallScore)
		respEntity.MaxScore = 5

		if err := json.Unmarshal(row.Criteria.([]byte), &respEntity.Criteria); err != nil {
			s.log.ErrorWithID(ctx, "[Service: GetPhraseEvaluationsWithCriteriaBySessionID] Error unmarshalling phrase evaluations with criteria", err)
			return nil, app_error.New(err, app_error.ErrCodeGeneralUnmarshalFailed)
		}

		for i := range respEntity.Criteria {
			respEntity.Criteria[i].MaxScore = 5
			respEntity.Criteria[i].CriteriaColor = utils.GetScoreColor(respEntity.Criteria[i].CriteriaScore)
		}

		respEntities = append(respEntities, respEntity)
	}

	resp := &entities.GetPhraseEvaluationsWithCriteriaResp{
		PhraseEvaluations: respEntities,
	}

	return resp, nil
}
