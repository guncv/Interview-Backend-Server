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

type SessionRepository interface {
	CreateSession(ctx context.Context, req *db.CreateSessionParams) error
	GetSessionByID(ctx context.Context, id uuid.UUID) (db.Sessions, error)
	RevokeSessionByID(ctx context.Context, id uuid.UUID) error
}

type sessionRepository struct {
	log *log.Logger
	db  db.Queries
}

func NewSessionRepository(l *log.Logger, db db.Queries) SessionRepository {
	return &sessionRepository{
		log: l,
		db:  db,
	}
}

func (r *sessionRepository) CreateSession(ctx context.Context, req *db.CreateSessionParams) error {
	r.log.InfoWithID(ctx, "[Repository: CreateSession] Called")

	if _, err := r.db.CreateSession(ctx, *req); err != nil {
		r.log.ErrorWithID(ctx, "[Repository: CreateSession] Error creating session", err)
		return app_error.HandleDatabaseError(err)
	}

	return nil
}

func (r *sessionRepository) GetSessionByID(ctx context.Context, id uuid.UUID) (db.Sessions, error) {
	r.log.InfoWithID(ctx, "[Repository: GetSession] Called")

	session, err := r.db.GetSessionByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.log.ErrorWithID(ctx, "[Repository: GetSession] Session not found", err)
			return db.Sessions{}, app_error.New(err, app_error.ErrCodeAuthSessionNotFound)
		}
		r.log.ErrorWithID(ctx, "[Repository: GetSession] Error getting session", err)
		return db.Sessions{}, app_error.HandleDatabaseError(err)
	}

	return session, nil
}

func (r *sessionRepository) RevokeSessionByID(ctx context.Context, id uuid.UUID) error {
	r.log.InfoWithID(ctx, "[Repository: RevokeSession] Called")

	if err := r.db.RevokeSessionByID(ctx, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.log.ErrorWithID(ctx, "[Repository: RevokeSession] Session not found", err)
			return app_error.New(err, app_error.ErrCodeAuthSessionNotFound)
		}
		r.log.ErrorWithID(ctx, "[Repository: RevokeSession] Error revoking session", err)
		return app_error.HandleDatabaseError(err)
	}

	return nil
}
