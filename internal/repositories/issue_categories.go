package repositories

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	db "gitlab.com/interview-simulation/interview-backend-server/internal/db/sqlc"
	app_error "gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	log "gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
)

type IssueCategoriesRepository interface {
	GetIssueCategoryIfExists(ctx context.Context, id uuid.UUID) (*db.GetIssueCategoryIfExistsRow, error)
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

func (r *issueCategoriesRepository) GetIssueCategoryIfExists(ctx context.Context, id uuid.UUID) (*db.GetIssueCategoryIfExistsRow, error) {
	resp, err := r.db.GetIssueCategoryIfExists(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.log.ErrorWithID(ctx, "[Repository: GetIssueCategoryIfExists] Issue category not found", err)
			return nil, app_error.New(err, app_error.ErrCodeIssueCategoryNotFound)
		}
		r.log.ErrorWithID(ctx, "[Repository: GetIssueCategoryIfExists] Error getting issue category if exists", err)
		return nil, app_error.HandleDatabaseError(err)
	}

	return &resp, nil
}

func (r *issueCategoriesRepository) CreateAdminIssueCategory(ctx context.Context, req *db.CreateAdminIssueCategoryParams) error {
	err := r.db.CreateAdminIssueCategory(ctx, *req)
	if err != nil {
		r.log.ErrorWithID(ctx, "[Repository: CreateAdminIssueCategory] Error creating admin issue category", err)
		return app_error.HandleDatabaseError(err)
	}

	return nil
}

func (r *issueCategoriesRepository) ListIssueCategories(ctx context.Context) ([]db.ListIssueCategoriesRow, error) {
	resp, err := r.db.ListIssueCategories(ctx)
	if err != nil {
		r.log.ErrorWithID(ctx, "[Repository: ListIssueCategories] Error listing issue categories", err)
		return nil, app_error.HandleDatabaseError(err)
	}

	return resp, nil
}
