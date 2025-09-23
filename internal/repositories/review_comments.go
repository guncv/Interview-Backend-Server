package repositories

import (
	"context"
	"errors"

	db "gitlab.com/interview-simulation/interview-backend-server/internal/db/sqlc"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
)

type ReviewCommentRepository interface {
	CreateReviewComment(ctx context.Context, req *db.CreateReviewCommentParams) error
}

type reviewCommentRepository struct {
	log *log.Logger
	db  db.Store
}

func NewReviewCommentRepository(l *log.Logger, db db.Store) ReviewCommentRepository {
	return &reviewCommentRepository{
		log: l,
		db:  db,
	}
}

func (r *reviewCommentRepository) CreateReviewComment(ctx context.Context, req *db.CreateReviewCommentParams) error {
	r.log.InfoWithID(ctx, "[Repository: CreateReviewComment] Called")

	rowAffected, err := r.db.CreateReviewComment(ctx, *req)
	if err != nil {
		r.log.ErrorWithID(ctx, "[Repository: CreateReviewComment] Error creating review comment", err)
		return app_error.HandleDatabaseError(err)
	}

	if rowAffected == 0 {
		err := errors.New("session not found")
		r.log.ErrorWithID(ctx, "[Repository: CreateReviewComment] Review comment not found", err)
		return app_error.New(err, app_error.ErrCodeSessionNotFoundOrDeleted)
	}

	return nil
}
