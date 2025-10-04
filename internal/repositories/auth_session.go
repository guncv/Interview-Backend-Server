package repositories

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	db "gitlab.com/interview-simulation/interview-backend-server/internal/db/sqlc"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
)

type AuthSessionRepository interface {
	CreateAuthSession(ctx context.Context, req *db.CreateAuthSessionParams) error
	GetAuthSessionByID(ctx context.Context, id uuid.UUID) (*db.AuthSessions, error)
	RevokeAuthSessionByID(ctx context.Context, id uuid.UUID) error
}

type authSessionRepository struct {
	log *log.Logger
	db  db.Store
}

func NewAuthSessionRepository(l *log.Logger, db db.Store) AuthSessionRepository {
	return &authSessionRepository{
		log: l,
		db:  db,
	}
}

func (r *authSessionRepository) CreateAuthSession(ctx context.Context, req *db.CreateAuthSessionParams) error {

	if err := r.db.CreateAuthSession(ctx, *req); err != nil {
		r.log.ErrorWithID(ctx, "[Repository: CreateAuthSession] Error creating session", err)
		return app_error.HandleDatabaseError(err)
	}

	return nil
}

func (r *authSessionRepository) GetAuthSessionByID(ctx context.Context, id uuid.UUID) (*db.AuthSessions, error) {

	session, err := r.db.GetAuthSessionByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.log.ErrorWithID(ctx, "[Repository: GetAuthSessionByID] Session not found", err)
			return nil, app_error.New(err, app_error.ErrCodeAuthSessionNotFound)
		}
		r.log.ErrorWithID(ctx, "[Repository: GetAuthSessionByID] Error getting session", err)
		return nil, app_error.HandleDatabaseError(err)
	}

	return &session, nil
}

func (r *authSessionRepository) RevokeAuthSessionByID(ctx context.Context, id uuid.UUID) error {

	if err := r.db.RevokeAuthSessionByID(ctx, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.log.ErrorWithID(ctx, "[Repository: RevokeAuthSessionByID] Session not found", err)
			return app_error.New(err, app_error.ErrCodeAuthSessionNotFound)
		}
		r.log.ErrorWithID(ctx, "[Repository: RevokeAuthSessionByID] Error revoking session", err)
		return app_error.HandleDatabaseError(err)
	}

	return nil
}
