package repositories

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"
	"gitlab.com/interview-simulation/interview-backend-server/internal/config"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	db "gitlab.com/interview-simulation/interview-backend-server/internal/db/sqlc"
	app_error "gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
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
}

type evaluationScoresRepository struct {
	log *log.Logger
	db  db.Store
	cfg *config.Config
}

func NewEvaluationScoresRepository(l *log.Logger, db db.Store, cfg *config.Config) EvaluationScoresRepository {
	return &evaluationScoresRepository{
		log: l,
		db:  db,
		cfg: cfg,
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

	jsonBody, err := json.Marshal(req)
	if err != nil {
		r.log.ErrorWithID(ctx, "[Repository: GetEvaluationOverallSummary] Failed to marshal request body", err)
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", endpoint, bytes.NewReader(jsonBody))
	if err != nil {
		r.log.ErrorWithID(ctx, "[Repository: GetEvaluationOverallSummary] Failed to create HTTP request", err)
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: constants.TimeoutHTTP}
	resp, err := client.Do(httpReq)
	if err != nil {
		r.log.ErrorWithID(ctx, "[Repository: GetEvaluationOverallSummary] HTTP request failed", err)
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		r.log.ErrorWithID(ctx, "[Repository: GetEvaluationOverallSummary] Failed to read response body", err)
		return nil, err
	}

	r.log.InfoWithID(ctx, fmt.Sprintf("[Repository: GetEvaluationOverallSummary] Response: %s | Body: %s", resp.Status, string(bodyBytes)))

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("interview agent returned status: %s | body: %s", resp.Status, string(bodyBytes))
		r.log.ErrorWithID(ctx, "[Repository: GetEvaluationOverallSummary] HTTP error response", err)
		return nil, err
	}

	var result CreateEvaluationOverallSummaryTxResp
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		r.log.ErrorWithID(ctx, "[Repository: GetEvaluationOverallSummary] Failed to unmarshal response body", err)
		return nil, err
	}

	return &result, nil
}

func (r *evaluationScoresRepository) InterviewFeedbackAndScore(ctx context.Context, req *InterviewFeedbackAndScoreReq) (*InterviewFeedbackAndScoreResp, error) {
	r.log.InfoWithID(ctx, "[Repository: InterviewFeedbackAndScore] Called")

	endpoint := r.cfg.InterviewSessionConfig.InterviewAgentURL + constants.PathFeedbackAndScoreAgent

	jsonBody, err := json.Marshal(req)
	if err != nil {
		r.log.ErrorWithID(ctx, "[Repository: InterviewFeedbackAndScore] Failed to marshal request body", err)
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", endpoint, bytes.NewReader(jsonBody))
	if err != nil {
		r.log.ErrorWithID(ctx, "[Repository: InterviewFeedbackAndScore] Failed to create HTTP request", err)
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: constants.TimeoutHTTP}
	resp, err := client.Do(httpReq)
	if err != nil {
		r.log.ErrorWithID(ctx, "[Repository: InterviewFeedbackAndScore] HTTP request failed", err)
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		r.log.ErrorWithID(ctx, "[Repository: InterviewFeedbackAndScore] Failed to read response body", err)
		return nil, err
	}

	r.log.InfoWithID(ctx, fmt.Sprintf("[Repository: InterviewFeedbackAndScore] Response: %s | Body: %s", resp.Status, string(bodyBytes)))

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("interview agent returned status: %s | body: %s", resp.Status, string(bodyBytes))
		r.log.ErrorWithID(ctx, "[Repository: InterviewFeedbackAndScore] HTTP error response", err)
		return nil, err
	}

	var result InterviewFeedbackAndScoreResp
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		r.log.ErrorWithID(ctx, "[Repository: InterviewFeedbackAndScore] Failed to unmarshal response body", err)
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
		r.log.ErrorWithID(ctx, "[Repository: GetEvaluationSummaryJsonBySessionAndState] Error getting evaluation summary json by session and state", err)
		return nil, app_error.HandleDatabaseError(err)
	}

	return resp, nil
}

func (r *evaluationScoresRepository) CalculateEachCriteriaCommentBySessionAndState(ctx context.Context, req *PreProcessedCriteriaReq) (*PostProcessedCriteriaResp, error) {
	r.log.InfoWithID(ctx, "[Repository: CalculateEachCriteriaCommentBySessionAndState] Called")

	endpoint := r.cfg.InterviewSessionConfig.InterviewAgentURL + constants.PathCalculateEachCriteriaCommentBySessionAgent

	jsonBody, err := json.Marshal(req)
	if err != nil {
		r.log.ErrorWithID(ctx, "[Repository: CalculateEachCriteriaCommentBySessionAndState] Failed to marshal request body", err)
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", endpoint, bytes.NewReader(jsonBody))
	if err != nil {
		r.log.ErrorWithID(ctx, "[Repository: CalculateEachCriteriaCommentBySessionAndState] Failed to create HTTP request", err)
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: constants.TimeoutHTTP}
	resp, err := client.Do(httpReq)
	if err != nil {
		r.log.ErrorWithID(ctx, "[Repository: CalculateEachCriteriaCommentBySessionAndState] HTTP request failed", err)
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		r.log.ErrorWithID(ctx, "[Repository: CalculateEachCriteriaCommentBySessionAndState] Failed to read response body", err)
		return nil, err
	}

	r.log.InfoWithID(ctx, fmt.Sprintf("[Repository: CalculateEachCriteriaCommentBySessionAndState] Response: %s | Body: %s", resp.Status, string(bodyBytes)))

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("interview agent returned status: %s | body: %s", resp.Status, string(bodyBytes))
		r.log.ErrorWithID(ctx, "[Repository: CalculateEachCriteriaCommentBySessionAndState] HTTP error response", err)
		return nil, err
	}

	var result *PostProcessedCriteriaResp
	if err := json.Unmarshal(bodyBytes, result); err != nil {
		r.log.ErrorWithID(ctx, "[Repository: CalculateEachCriteriaCommentBySessionAndState] Failed to unmarshal response body", err)
		return nil, err
	}

	return result, nil
}
