package repositories

import (
	"context"

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

	if err := r.db.CreateReviewComment(ctx, *req); err != nil {
		r.log.ErrorWithID(ctx, "[Repository: CreateReviewComment] Error creating review comment", err)
		return app_error.HandleDatabaseError(err)
	}

	return nil
}
