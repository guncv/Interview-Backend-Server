package repositories

import (
	db "gitlab.com/interview-simulation/interview-backend-server/internal/db/sqlc"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
)

type UserRoleRepository interface {
}

type userRoleRepository struct {
	log *log.Logger
	db  db.Store
}

func NewUserRoleRepository(l *log.Logger, db db.Store) UserRoleRepository {
	return &userRoleRepository{
		log: l,
		db:  db,
	}
}
