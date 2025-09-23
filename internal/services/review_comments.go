package services

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	db "gitlab.com/interview-simulation/interview-backend-server/internal/db/sqlc"
	"gitlab.com/interview-simulation/interview-backend-server/internal/entities"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
	"gitlab.com/interview-simulation/interview-backend-server/internal/middleware"
	"gitlab.com/interview-simulation/interview-backend-server/internal/repositories"
	"gitlab.com/interview-simulation/interview-backend-server/internal/utils"
)

type ReviewCommentService interface {
	CreateReviewComment(ctx context.Context, req *entities.CreateReviewCommentReq) error
}

type reviewCommentService struct {
	log               *log.Logger
	reviewCommentRepo repositories.ReviewCommentRepository
	authContext       middleware.AuthContext
	generator         utils.Generator
}

func NewReviewCommentService(
	l *log.Logger,
	reviewCommentRepo repositories.ReviewCommentRepository,
	authContext middleware.AuthContext,
	generator utils.Generator,
) ReviewCommentService {
	return &reviewCommentService{
		log:               l,
		reviewCommentRepo: reviewCommentRepo,
		authContext:       authContext,
		generator:         generator,
	}
}

func (s *reviewCommentService) CreateReviewComment(ctx context.Context, req *entities.CreateReviewCommentReq) error {
	s.log.InfoWithID(ctx, "[Service: CreateReviewComment] Called")

	sessionID, err := uuid.Parse(req.SessionID)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: CreateReviewComment] Error parsing session ID", err)
		return app_error.New(err, app_error.ErrCodeGeneralInvalidUUID)
	}

	authCtx, err := s.authContext.GetAuthContext(ctx)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: CreateReviewComment] Error getting auth context", err)
		return err
	}

	userID, err := uuid.Parse(authCtx.Payload.UserID)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: CreateReviewComment] Error parsing user ID", err)
		return app_error.New(err, app_error.ErrCodeGeneralInvalidUUID)
	}

	dbReq := &db.CreateReviewCommentParams{
		ID:           s.generator.GenerateUUID(ctx),
		SessionID:    sessionID,
		AuthorType:   string(authCtx.Payload.Role),
		AuthorUserID: uuid.NullUUID{UUID: userID, Valid: true},
		Rating:       sql.NullInt16{Int16: int16(req.Rating), Valid: true},
		CreatedAt:    sql.NullTime{Time: time.Now(), Valid: true},
		UpdatedAt:    sql.NullTime{Time: time.Now(), Valid: true},
	}

	if req.Comment != nil {
		dbReq.Description = sql.NullString{String: *req.Comment, Valid: true}
	} else {
		dbReq.Description = sql.NullString{Valid: false}
	}

	if err := s.reviewCommentRepo.CreateReviewComment(ctx, dbReq); err != nil {
		s.log.ErrorWithID(ctx, "[Service: CreateReviewComment] Error creating review comment", err)
		return err
	}

	return nil
}
