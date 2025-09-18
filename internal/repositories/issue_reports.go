package repositories

import (
	"context"
	"errors"

	"github.com/google/uuid"
	db "gitlab.com/interview-simulation/interview-backend-server/internal/db/sqlc"
	app_error "gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	log "gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
)

type IssueReportsRepository interface {
	CreateUserIssueReport(ctx context.Context, req *db.CreateUserIssueReportParams) error
	ListUserIssueReports(ctx context.Context, userID uuid.UUID) ([]db.ListUserIssueReportsRow, error)
	UpdateUserIssueReportByID(ctx context.Context, req *db.UpdateUserIssueReportByIDParams) error
	GetUserIssueReportUserIDAndStatusByID(ctx context.Context, id uuid.UUID) (*db.GetUserIssueReportUserIDAndStatusByIDRow, error)
}

type issueReportsRepository struct {
	log *log.Logger
	db  db.Store
}

func NewIssueReportsRepository(l *log.Logger, db db.Store) IssueReportsRepository {
	return &issueReportsRepository{
		log: l,
		db:  db,
	}
}

func (r *issueReportsRepository) CreateUserIssueReport(ctx context.Context, req *db.CreateUserIssueReportParams) error {
	r.log.InfoWithID(ctx, "[Repository: CreateUserIssueReport] Called")

	if err := r.db.CreateUserIssueReport(ctx, *req); err != nil {
		r.log.ErrorWithID(ctx, "[Repository: CreateUserIssueReport] Error creating user issue report", err)
		return app_error.HandleDatabaseError(err)
	}

	return nil
}

func (r *issueReportsRepository) ListUserIssueReports(ctx context.Context, userID uuid.UUID) ([]db.ListUserIssueReportsRow, error) {
	r.log.InfoWithID(ctx, "[Repository: ListUserIssueReports] Called")

	resp, err := r.db.ListUserIssueReports(ctx, uuid.NullUUID{UUID: userID, Valid: true})
	if err != nil {
		r.log.ErrorWithID(ctx, "[Repository: ListUserIssueReports] Error listing user issue reports", err)
		return nil, app_error.HandleDatabaseError(err)
	}

	return resp, nil
}

func (r *issueReportsRepository) UpdateUserIssueReportByID(ctx context.Context, req *db.UpdateUserIssueReportByIDParams) error {
	r.log.InfoWithID(ctx, "[Repository: UpdateUserIssueReport] Called")

	rowAffected, err := r.db.UpdateUserIssueReportByID(ctx, *req)
	if err != nil {
		r.log.ErrorWithID(ctx, "[Repository: UpdateUserIssueReport] Error updating user issue report", err)
		return app_error.HandleDatabaseError(err)
	}

	if rowAffected == 0 {
		r.log.ErrorWithID(ctx, "[Repository: UpdateUserIssueReport] User issue report not found")
		return app_error.New(errors.New("user issue report not found"), app_error.ErrCodeIssueReportNotFound)
	}

	return nil
}

func (r *issueReportsRepository) GetUserIssueReportUserIDAndStatusByID(ctx context.Context, id uuid.UUID) (*db.GetUserIssueReportUserIDAndStatusByIDRow, error) {
	r.log.InfoWithID(ctx, "[Repository: GetUserIssueReportUserIDAndStatusByID] Called")

	resp, err := r.db.GetUserIssueReportUserIDAndStatusByID(ctx, id)
	if err != nil {
		r.log.ErrorWithID(ctx, "[Repository: GetUserIssueReportUserIDAndStatusByID] Error getting user issue report user ID and status", err)
		return nil, app_error.HandleDatabaseError(err)
	}

	return &resp, nil
}
