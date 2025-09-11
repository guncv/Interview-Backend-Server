package repositories

import (
	"context"

	db "gitlab.com/interview-simulation/interview-backend-server/internal/db/sqlc"
	app_error "gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	log "gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
)

type EvaluationRubricsRepository interface {
	GetRubricWithCriteriaByName(ctx context.Context, req *db.GetRubricWithCriteriaByNameParams) ([]db.GetRubricWithCriteriaByNameRow, error)
}

type evaluationRubricsRepository struct {
	log *log.Logger
	db  db.Store
}

func NewEvaluationRubricsRepository(l *log.Logger, db db.Store) EvaluationRubricsRepository {
	return &evaluationRubricsRepository{
		log: l,
		db:  db,
	}
}

func (r *evaluationRubricsRepository) GetRubricWithCriteriaByName(ctx context.Context, req *db.GetRubricWithCriteriaByNameParams) ([]db.GetRubricWithCriteriaByNameRow, error) {
	r.log.InfoWithID(ctx, "[Repository: GetRubricWithCriteriaByName] Called")

	resp, err := r.db.GetRubricWithCriteriaByName(ctx, *req)
	if err != nil {
		r.log.ErrorWithID(ctx, "[Repository: GetRubricWithCriteriaByName] Error getting rubric with criteria by name", err)
		return nil, app_error.HandleDatabaseError(err)
	}

	return resp, nil
}
