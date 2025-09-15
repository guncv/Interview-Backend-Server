package repositories

import (
	"context"
	"database/sql"

	db "gitlab.com/interview-simulation/interview-backend-server/internal/db/sqlc"
	app_error "gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	log "gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
)

type EvaluationScoresRepository interface {
	CreateEvaluationWithCriteriaScoreAndImproveSentenceTx(ctx context.Context, req *CreateEvaluationAndScoreTxReq) error
}

type evaluationScoresRepository struct {
	log *log.Logger
	db  db.Store
}

func NewEvaluationScoresRepository(l *log.Logger, db db.Store) EvaluationScoresRepository {
	return &evaluationScoresRepository{
		log: l,
		db:  db,
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
				ID:           criterion.ID,
				EvaluationID: req.EvaluationID,
				CriterionID:  criterion.CriterionID,
				Score:        int32(criterion.Score),
				CommentMd:    criterion.CommentMd,
				CreatedAt:    req.CreatedAt,
				UpdatedAt:    sql.NullTime{Time: req.UpdatedAt, Valid: true},
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
