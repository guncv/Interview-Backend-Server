package repositories

import (
	"context"
	"errors"

	db "gitlab.com/interview-simulation/interview-backend-server/internal/db/sqlc"
	app_error "gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	log "gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
)

type InterviewSessionRepository interface {
	CreateInterviewSession(ctx context.Context, req *db.CreateInterviewSessionParams) error
	StartInterviewSession(ctx context.Context, req *db.StartInterviewSessionParams) error
}

type interviewSessionRepository struct {
	log *log.Logger
	db  db.Store
}

func NewInterviewSessionRepository(
	log *log.Logger,
	db db.Store,
) InterviewSessionRepository {
	return &interviewSessionRepository{
		log: log,
		db:  db,
	}
}

func (r *interviewSessionRepository) CreateInterviewSession(ctx context.Context, req *db.CreateInterviewSessionParams) error {
	r.log.InfoWithID(ctx, "[Repository: CreateInterviewSession] Called")

	if err := r.db.CreateInterviewSession(ctx, *req); err != nil {
		r.log.ErrorWithID(ctx, "[Repository: CreateInterviewSession] Error creating interview session", err)
		return app_error.HandleDatabaseError(err)
	}

	return nil
}

func (r *interviewSessionRepository) StartInterviewSession(ctx context.Context, req *db.StartInterviewSessionParams) error {
	r.log.InfoWithID(ctx, "[Repository: StartInterviewSession] Called")

	rowAffected, err := r.db.StartInterviewSession(ctx, *req)
	if err != nil {
		r.log.ErrorWithID(ctx, "[Repository: StartInterviewSession] Error starting interview session", err)
		return app_error.HandleDatabaseError(err)
	}

	if rowAffected == 0 {
		err := errors.New("interview session not found")
		r.log.ErrorWithID(ctx, "[Repository: StartInterviewSession] Interview session not found", err)
		return app_error.New(err, app_error.ErrCodeSessionNotFound)
	}

	return nil
}
