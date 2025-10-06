package repositories

import (
	"context"

	db "gitlab.com/interview-simulation/interview-backend-server/internal/db/sqlc"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
)

type UserRepository interface {
	HealthCheck(ctx context.Context) (string, error)
}

type userRepository struct {
	log *log.Logger
	db  db.Store
}

func NewUserRepository(l *log.Logger, db db.Store) UserRepository {
	return &userRepository{
		log: l,
		db:  db,
	}
}

func (r *userRepository) HealthCheck(ctx context.Context) (string, error) {
	return "Status OK", nil
}
