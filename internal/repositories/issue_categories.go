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
	CreateAdminIssueCategory(ctx context.Context, req *db.CreateAdminIssueCategoryParams) error
	ListIssueCategories(ctx context.Context) ([]db.ListIssueCategoriesRow, error)
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

func (r *issueCategoriesRepository) CreateAdminIssueCategory(ctx context.Context, req *db.CreateAdminIssueCategoryParams) error {
	r.log.InfoWithID(ctx, "[Repository: CreateAdminIssueCategory] Called")

	err := r.db.CreateAdminIssueCategory(ctx, *req)
	if err != nil {
		r.log.ErrorWithID(ctx, "[Repository: CreateAdminIssueCategory] Error creating admin issue category", err)
		return app_error.HandleDatabaseError(err)
	}

	return nil
}

func (r *issueCategoriesRepository) ListIssueCategories(ctx context.Context) ([]db.ListIssueCategoriesRow, error) {
	r.log.InfoWithID(ctx, "[Repository: ListIssueCategories] Called")

	resp, err := r.db.ListIssueCategories(ctx)
	if err != nil {
		r.log.ErrorWithID(ctx, "[Repository: ListIssueCategories] Error listing issue categories", err)
		return nil, app_error.HandleDatabaseError(err)
	}

	return resp, nil
}
