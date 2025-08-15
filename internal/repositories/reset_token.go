package repositories

import (
	db "gitlab.com/interview-simulation/interview-backend-server/internal/db/sqlc"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
)

type ResetTokenRepository interface {
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
