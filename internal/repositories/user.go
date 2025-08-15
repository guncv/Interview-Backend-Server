package repositories

import (
	"context"

	"github.com/google/uuid"

	db "gitlab.com/interview-simulation/interview-backend-server/internal/db/sqlc"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
)

type UserRepository interface {
	HealthCheck(ctx context.Context) (string, error)
	CheckIsEmailExists(ctx context.Context, email string) (*db.Users, error)
	CreateUser(ctx context.Context, req *db.CreateUserParams) (*db.Users, error)
	UpdateUser(ctx context.Context, req *db.UpdateUserParams) (*db.Users, error)
	VerifyEmail(ctx context.Context, userID string) error
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
	r.log.InfoWithID(ctx, "[Repository: HealthCheck] Called")
	return "Status OK", nil
}

func (r *userRepository) CreateUser(ctx context.Context, req *db.CreateUserParams) (*db.Users, error) {
	r.log.InfoWithID(ctx, "[Repository: CreateUser] Called")

	user, err := r.db.CreateUser(ctx, *req)
	if err != nil {
		return nil, app_error.New(err, app_error.ErrCodeGeneralServerUnavailable)
	}

	return &user, nil
}

func (r *userRepository) CheckIsEmailExists(ctx context.Context, email string) (*db.Users, error) {
	r.log.InfoWithID(ctx, "[Repository: CheckIsEmailExists] Called")

	user, err := r.db.CheckIsEmailExists(ctx, email)
	if err != nil {
		return nil, app_error.New(err, app_error.ErrCodeGeneralServerUnavailable)
	}

	return &user, nil
}

func (r *userRepository) UpdateUser(ctx context.Context, req *db.UpdateUserParams) (*db.Users, error) {
	r.log.InfoWithID(ctx, "[Repository: UpdateUser] Called")

	user, err := r.db.UpdateUser(ctx, *req)
	if err != nil {
		return nil, app_error.New(err, app_error.ErrCodeGeneralServerUnavailable)
	}

	return &user, nil
}

func (r *userRepository) VerifyEmail(ctx context.Context, userID string) error {
	r.log.InfoWithID(ctx, "[Repository: VerifyEmail] Called")

	_, err := r.db.VerifyEmail(ctx, uuid.MustParse(userID))
	if err != nil {
		return app_error.New(err, app_error.ErrCodeGeneralServerUnavailable)
	}

	return nil
}
