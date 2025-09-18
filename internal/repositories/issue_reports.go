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

type IssueReportsRepository interface {
	CreateUserIssueReport(ctx context.Context, req *db.CreateUserIssueReportParams) (*db.CreateUserIssueReportRow, error)
	ListUserIssueReports(ctx context.Context, userID uuid.UUID) ([]db.ListUserIssueReportsRow, error)
	UpdateUserIssueReportByID(ctx context.Context, req *db.UpdateUserIssueReportByIDParams) (*db.UpdateUserIssueReportByIDRow, error)
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

func (r *issueReportsRepository) CreateUserIssueReport(ctx context.Context, req *db.CreateUserIssueReportParams) (*db.CreateUserIssueReportRow, error) {
	r.log.InfoWithID(ctx, "[Repository: CreateUserIssueReport] Called")

	resp, err := r.db.CreateUserIssueReport(ctx, *req)
	if err != nil {
		r.log.ErrorWithID(ctx, "[Repository: CreateUserIssueReport] Error creating user issue report", err)
		return nil, app_error.HandleDatabaseError(err)
	}

	return &resp, nil
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

func (r *issueReportsRepository) UpdateUserIssueReportByID(ctx context.Context, req *db.UpdateUserIssueReportByIDParams) (*db.UpdateUserIssueReportByIDRow, error) {
	r.log.InfoWithID(ctx, "[Repository: UpdateUserIssueReport] Called")

	resp, err := r.db.UpdateUserIssueReportByID(ctx, *req)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.log.ErrorWithID(ctx, "[Repository: UpdateUserIssueReport] User issue report not found", err)
			return nil, app_error.New(err, app_error.ErrCodeIssueReportNotFound)
		}
		r.log.ErrorWithID(ctx, "[Repository: UpdateUserIssueReport] Error updating user issue report", err)
		return nil, app_error.HandleDatabaseError(err)
	}

	return &resp, nil
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
