package repositories

import (
	"context"
	"database/sql"
	"errors"

	db "gitlab.com/interview-simulation/interview-backend-server/internal/db/sqlc"
	app_error "gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
)

type ResetTokenRepository interface {
	CreateResetToken(ctx context.Context, req *db.CreateResetTokenParams) error
	GetResetToken(ctx context.Context, token string) (db.ResetTokens, error)
	UpdateResetTokenUsed(ctx context.Context, token string) error
}

type resetTokenRepository struct {
	log *log.Logger
	db  db.Store
}

func NewResetTokenRepository(l *log.Logger, db db.Store) ResetTokenRepository {
	return &resetTokenRepository{
		log: l,
		db:  db,
	}
}

func (r *resetTokenRepository) CreateResetToken(ctx context.Context, req *db.CreateResetTokenParams) error {
	r.log.InfoWithID(ctx, "[Repository: CreateResetToken] Called")

	if err := r.db.CreateResetToken(ctx, *req); err != nil {
		r.log.ErrorWithID(ctx, "[Repository: CreateResetToken] Error creating reset token", err)
		return app_error.HandleDatabaseError(err)
	}

	return nil
}

func (r *resetTokenRepository) GetResetToken(ctx context.Context, token string) (db.ResetTokens, error) {
	r.log.InfoWithID(ctx, "[Repository: GetResetToken] Called")

	resetToken, err := r.db.GetResetToken(ctx, token)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.log.ErrorWithID(ctx, "[Repository: GetResetToken] Reset token not found", err)
			return db.ResetTokens{}, app_error.New(err, app_error.ErrCodeAuthResetTokenNotFound)
		}
		r.log.ErrorWithID(ctx, "[Repository: GetResetToken] Error getting reset token", err)
		return db.ResetTokens{}, app_error.HandleDatabaseError(err)
	}

	return resetToken, nil
}

func (r *resetTokenRepository) UpdateResetTokenUsed(ctx context.Context, token string) error {
	r.log.InfoWithID(ctx, "[Repository: UpdateResetTokenUsed] Called")

	if err := r.db.UpdateResetTokenUsed(ctx, token); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.log.ErrorWithID(ctx, "[Repository: UpdateResetTokenUsed] Reset token not found", err)
			return app_error.New(err, app_error.ErrCodeAuthResetTokenNotFound)
		}
		r.log.ErrorWithID(ctx, "[Repository: UpdateResetTokenUsed] Error updating reset token", err)
		return app_error.HandleDatabaseError(err)
	}

	return nil
}
