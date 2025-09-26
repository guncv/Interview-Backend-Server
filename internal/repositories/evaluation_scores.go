package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"gitlab.com/interview-simulation/interview-backend-server/internal/config"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	db "gitlab.com/interview-simulation/interview-backend-server/internal/db/sqlc"
	app_error "gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/http"
	log "gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
)

type EvaluationScoresRepository interface {
	CreateEvaluationWithCriteriaScoreAndImproveSentenceTx(ctx context.Context, req *CreateEvaluationAndScoreTxReq) error
	GetAllEvaluationsBySessionID(ctx context.Context, sessionID uuid.UUID) ([]db.GetAllEvaluationsBySessionIDRow, error)
	GetEvaluationOverallSummary(ctx context.Context, req *CreateEvaluationOverallSummaryTxReq) (*CreateEvaluationOverallSummaryTxResp, error)
	InterviewFeedbackAndScore(ctx context.Context, req *InterviewFeedbackAndScoreReq) (*InterviewFeedbackAndScoreResp, error)
	IsLastUserStateTurnScored(ctx context.Context, dbReq *db.IsLastUserStateTurnScoredParams) (bool, error)
	GetEvaluationSummaryJsonBySessionAndState(ctx context.Context, dbReq *db.GetEvaluationSummaryJsonBySessionAndStateParams) (json.RawMessage, error)
	CalculateEachCriteriaCommentBySessionAndState(ctx context.Context, req *PreProcessedCriteriaReq) (*PostProcessedCriteriaResp, error)
	CreatePhraseEvaluationAndCriteriaScoreWithIsScoredState(ctx context.Context, req *CreatePhraseEvaluationAndCriteriaScoreWithIsScoredStateReqTx) error
	GetPhraseEvaluationsWithCriteriaBySessionID(ctx context.Context, sessionID uuid.UUID) ([]db.GetPhraseEvaluationsWithCriteriaBySessionIDRow, error)
}

type evaluationScoresRepository struct {
	log        *log.Logger
	db         db.Store
	cfg        *config.Config
	httpClient http.HTTPClient
}

func NewEvaluationScoresRepository(
	l *log.Logger,
	db db.Store,
	cfg *config.Config,
	httpClient http.HTTPClient,
) EvaluationScoresRepository {
	return &evaluationScoresRepository{
		log:        l,
		db:         db,
		cfg:        cfg,
		httpClient: httpClient,
	}
}

func (r *evaluationScoresRepository) CreateEvaluationWithCriteriaScoreAndImproveSentenceTx(ctx context.Context, req *CreateEvaluationAndScoreTxReq) error {
	r.log.InfoWithID(ctx, "[Repository: CreateEvaluationScore] Called")

	if err := r.db.ExecTx(ctx, func(q *db.Queries) error {

		evaluationReq := db.CreateEvaluationParams{
			ID:              req.EvaluationID,
			SessionID:       req.SessionID,
			TurnID:          req.TurnID,
			RubricID:        req.RubricID,
			EvaluatorUserID: req.UserID,
			CurrentState:    req.CurrentState,
			OverallScore:    req.OverallScore,
			SummaryMd:       req.SummaryMd,
			CreatedAt:       req.CreatedAt,
			UpdatedAt:       sql.NullTime{Time: req.UpdatedAt, Valid: true},
		}

		if err := q.CreateEvaluation(ctx, evaluationReq); err != nil {
			r.log.ErrorWithID(ctx, "[Repository: CreateEvaluationScore] Error creating evaluation", err)
			return app_error.HandleDatabaseError(err)
		}

		improveSentenceReq := db.CreateUserTurnImprovementParams{
			ID:                req.ImproveSentenceID,
			InterviewTurnID:   req.TurnID,
			CorrectedSentence: req.ImproveSentence,
			ModelVersion:      req.LLmModel,
			CreatedAt:         req.CreatedAt,
		}

		if err := q.CreateUserTurnImprovement(ctx, improveSentenceReq); err != nil {
			r.log.ErrorWithID(ctx, "[Repository: CreateEvaluationScore] Error creating improve sentence", err)
			return app_error.HandleDatabaseError(err)
		}

		for _, criterion := range req.Criteria {
			scoreReq := db.CreateEvaluationScoreParams{
				ID:            criterion.ID,
				EvaluationID:  req.EvaluationID,
				CriterionID:   criterion.CriterionID,
				CriterionName: criterion.CriterionName,
				Score:         int32(criterion.Score),
				CommentMd:     criterion.CommentMd,
				CreatedAt:     req.CreatedAt,
				UpdatedAt:     sql.NullTime{Time: req.UpdatedAt, Valid: true},
			}

			if err := q.CreateEvaluationScore(ctx, scoreReq); err != nil {
				r.log.ErrorWithID(ctx, "[Repository: CreateEvaluationScore] Error creating evaluation score", err)
				return app_error.HandleDatabaseError(err)
			}
		}

		return nil
	}); err != nil {
		r.log.ErrorWithID(ctx, "[Repository: CreateEvaluationScore] Error creating evaluation score", err)
		return err
	}

	return nil
}

func (r *evaluationScoresRepository) GetAllEvaluationsBySessionID(ctx context.Context, sessionID uuid.UUID) ([]db.GetAllEvaluationsBySessionIDRow, error) {
	r.log.InfoWithID(ctx, "[Repository: GetAllEvaluationsBySessionID] Called")

	resp, err := r.db.GetAllEvaluationsBySessionID(ctx, sessionID)
	if err != nil {
		r.log.ErrorWithID(ctx, "[Repository: GetAllEvaluationsBySessionID] Error getting all evaluations by session ID", err)
		return nil, app_error.HandleDatabaseError(err)
	}

	return resp, nil
}

func (r *evaluationScoresRepository) GetEvaluationOverallSummary(ctx context.Context, req *CreateEvaluationOverallSummaryTxReq) (*CreateEvaluationOverallSummaryTxResp, error) {
	r.log.InfoWithID(ctx, "[Repository: GetEvaluationOverallSummary] Called")

	endpoint := r.cfg.InterviewSessionConfig.InterviewAgentURL + constants.PathEvaluationOverallSummaryAgent

	config := http.HTTPClientConfig{
		BaseURL:   endpoint,
		Method:    "POST",
		LogPrefix: "GetEvaluationOverallSummary",
	}

	var result CreateEvaluationOverallSummaryTxResp
	_, err := r.httpClient.MakeJSONRequest(ctx, config, req, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (r *evaluationScoresRepository) InterviewFeedbackAndScore(ctx context.Context, req *InterviewFeedbackAndScoreReq) (*InterviewFeedbackAndScoreResp, error) {
	r.log.InfoWithID(ctx, "[Repository: InterviewFeedbackAndScore] Called")

	endpoint := r.cfg.InterviewSessionConfig.InterviewAgentURL + constants.PathFeedbackAndScoreAgent

	config := http.HTTPClientConfig{
		BaseURL:   endpoint,
		Method:    "POST",
		LogPrefix: "InterviewFeedbackAndScore",
	}

	var result InterviewFeedbackAndScoreResp
	_, err := r.httpClient.MakeJSONRequest(ctx, config, req, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (r *evaluationScoresRepository) IsLastUserStateTurnScored(ctx context.Context, dbReq *db.IsLastUserStateTurnScoredParams) (bool, error) {
	r.log.InfoWithID(ctx, "[Repository: IsLastUserStateTurnScored] Called")

	resp, err := r.db.IsLastUserStateTurnScored(ctx, *dbReq)
	if err != nil {
		if err == sql.ErrNoRows {
			r.log.ErrorWithID(ctx, "[Repository: IsLastUserStateTurnScored] Last user state turn not found", err)
			return false, app_error.New(err, app_error.ErrCodeInterviewTurnsNotFound)
		}
		r.log.ErrorWithID(ctx, "[Repository: IsLastUserStateTurnScored] Error getting last user state turn scored", err)
		return false, app_error.HandleDatabaseError(err)
	}

	return resp.Bool, nil
}

func (r *evaluationScoresRepository) GetEvaluationSummaryJsonBySessionAndState(ctx context.Context, dbReq *db.GetEvaluationSummaryJsonBySessionAndStateParams) (json.RawMessage, error) {
	r.log.InfoWithID(ctx, "[Repository: GetEvaluationSummaryJsonBySessionAndState] Called")

	resp, err := r.db.GetEvaluationSummaryJsonBySessionAndState(ctx, *dbReq)
	if err != nil {
		if err == sql.ErrNoRows {
			r.log.ErrorWithID(ctx, "[Repository: GetEvaluationSummaryJsonBySessionAndState] Evaluation summary json by session and state not found", err)
			return nil, err
		}
		r.log.ErrorWithID(ctx, "[Repository: GetEvaluationSummaryJsonBySessionAndState] Error getting evaluation summary json by session and state", err)
		return nil, app_error.HandleDatabaseError(err)
	}

	return resp, nil
}

func (r *evaluationScoresRepository) CalculateEachCriteriaCommentBySessionAndState(ctx context.Context, req *PreProcessedCriteriaReq) (*PostProcessedCriteriaResp, error) {
	r.log.InfoWithID(ctx, "[Repository: CalculateEachCriteriaCommentBySessionAndState] Called")

	endpoint := r.cfg.InterviewSessionConfig.InterviewAgentURL + constants.PathCalculateEachCriteriaCommentBySessionAgent

	config := http.HTTPClientConfig{
		BaseURL:   endpoint,
		Method:    "POST",
		LogPrefix: "CalculateEachCriteriaCommentBySessionAndState",
	}

	var result PostProcessedCriteriaResp
	_, err := r.httpClient.MakeJSONRequest(ctx, config, req, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (r *evaluationScoresRepository) CreatePhraseEvaluationAndCriteriaScoreWithIsScoredState(ctx context.Context, req *CreatePhraseEvaluationAndCriteriaScoreWithIsScoredStateReqTx) error {
	r.log.InfoWithID(ctx, "[Repository: CreatePhraseEvaluationAndCriteriaScoreWithIsScoredState] Called")

	if err := r.db.ExecTx(ctx, func(q *db.Queries) error {

		dbCreatePhraseEvaluationParams := db.CreatePhraseEvaluationParams{
			ID:           req.EvaluationID,
			SessionID:    req.SessionID,
			StateID:      req.StateID,
			StateName:    req.StateName,
			OverallScore: req.OverallScore,
		}

		if err := q.CreatePhraseEvaluation(ctx, dbCreatePhraseEvaluationParams); err != nil {
			r.log.ErrorWithID(ctx, "[Repository: CreatePhraseEvaluationAndCriteriaScoreWithIsScoredState] Error creating phrase evaluation", err)
			return app_error.HandleDatabaseError(err)
		}

		dbCreateAllPhraseRubricScoresParams := db.CreateAllPhraseRubricScoresParams{
			Column1: req.Criteria.ID,
			Column2: req.Criteria.EvaluationID,
			Column3: req.Criteria.CriteriaID,
			Column4: req.Criteria.CriteriaName,
			Column5: req.Criteria.CriteriaAvgScore,
			Column6: req.Criteria.CriteriaComment,
		}

		if err := q.CreateAllPhraseRubricScores(ctx, dbCreateAllPhraseRubricScoresParams); err != nil {
			r.log.ErrorWithID(ctx, "[Repository: CreatePhraseEvaluationAndCriteriaScoreWithIsScoredState] Error creating phrase rubric scores", err)
			return app_error.HandleDatabaseError(err)
		}

		dbUpdateIsEvaluatedInterviewStateParams := db.UpdateIsEvaluatedInterviewStateByIDParams{
			ID:          req.StateID,
			IsEvaluated: sql.NullBool{Bool: true, Valid: true},
		}

		rowsAffected, err := q.UpdateIsEvaluatedInterviewStateByID(ctx, dbUpdateIsEvaluatedInterviewStateParams)
		if err != nil {
			r.log.ErrorWithID(ctx, "[Repository: CreatePhraseEvaluationAndCriteriaScoreWithIsScoredState] Error updating is evaluated interview state", err)
			return app_error.HandleDatabaseError(err)
		}

		if rowsAffected == 0 {
			r.log.ErrorWithID(ctx, "[Repository: CreatePhraseEvaluationAndCriteriaScoreWithIsScoredState] Error updating is evaluated interview state", errors.New("rows affected is 0"))
			return app_error.New(errors.New("interview state not found"), app_error.ErrCodeInterviewStateNotFound)
		}

		return nil
	}); err != nil {
		r.log.ErrorWithID(ctx, "[Repository: CreatePhraseEvaluationAndCriteriaScoreWithIsScoredState] Error creating phrase evaluation and criteria score", err)
		return err
	}

	return nil
}

func (r *evaluationScoresRepository) GetPhraseEvaluationsWithCriteriaBySessionID(ctx context.Context, sessionID uuid.UUID) ([]db.GetPhraseEvaluationsWithCriteriaBySessionIDRow, error) {
	r.log.InfoWithID(ctx, "[Repository: GetPhraseEvaluationsWithCriteriaBySessionID] Called")

	resp, err := r.db.GetPhraseEvaluationsWithCriteriaBySessionID(ctx, sessionID)
	if err != nil {
		r.log.ErrorWithID(ctx, "[Repository: GetPhraseEvaluationsWithCriteriaBySessionID] Error getting phrase evaluations with criteria by session ID", err)
		return nil, app_error.HandleDatabaseError(err)
	}

	r.log.InfoWithID(ctx, "[Repository: GetPhraseEvaluationsWithCriteriaBySessionID] Response: ", resp)

	return resp, nil
}
