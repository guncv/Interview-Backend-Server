package repositories

import (
	"context"

	"github.com/google/uuid"
	db "gitlab.com/interview-simulation/interview-backend-server/internal/db/sqlc"
	app_error "gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	log "gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
)

type IssueCategoriesRepository interface {
	CheckIssueCategoryExists(ctx context.Context, id uuid.UUID) (bool, error)
}

type issueCategoriesRepository struct {
	log *log.Logger
	db  db.Store
}

func NewIssueCategoriesRepository(log *log.Logger, db db.Store) IssueCategoriesRepository {
	return &issueCategoriesRepository{log: log, db: db}
}

func (r *issueCategoriesRepository) CheckIssueCategoryExists(ctx context.Context, id uuid.UUID) (bool, error) {
	r.log.InfoWithID(ctx, "[Repository: CheckIssueCategoryExists] Called")

	resp, err := r.db.CheckIssueCategoryExists(ctx, id)
	if err != nil {
		r.log.ErrorWithID(ctx, "[Repository: CheckIssueCategoryExists] Error checking issue category exists", err)
		return false, app_error.HandleDatabaseError(err)
	}

	return resp, nil
}
