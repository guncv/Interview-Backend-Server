package repositories

import (
	"context"

	db "gitlab.com/interview-simulation/interview-backend-server/internal/db/sqlc"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
)

type UserRepository interface {
	HealthCheck(ctx context.Context) (string, error)
	CreateUserWithProvider(ctx context.Context, req *db.CreateUserWithProviderParams) error
	CheckUserExistsByProviderID(ctx context.Context, req db.CheckUserExistsByProviderIDParams) (bool, error)
	GetUserByProviderID(ctx context.Context, req db.GetUserByProviderIDParams) (db.Users, error)
	UpdateUserLoginInfo(ctx context.Context, req db.UpdateUserLoginInfoParams) error
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

func (r *userRepository) CreateUserWithProvider(ctx context.Context, req *db.CreateUserWithProviderParams) error {

	err := r.db.CreateUserWithProvider(ctx, *req)
	if err != nil {
		r.log.ErrorWithID(ctx, "[Repository: CreateUserWithProvider] Error creating user with provider", err)
		return app_error.HandleDatabaseError(err)
	}

	return nil
}

func (r *userRepository) CheckUserExistsByProviderID(ctx context.Context, req db.CheckUserExistsByProviderIDParams) (bool, error) {
	exists, err := r.db.CheckUserExistsByProviderID(ctx, req)
	if err != nil {
		r.log.ErrorWithID(ctx, "[Repository: CheckUserExistsByProviderID] Error checking user existence", err)
		return false, app_error.HandleDatabaseError(err)
	}
	return exists, nil
}

func (r *userRepository) GetUserByProviderID(ctx context.Context, req db.GetUserByProviderIDParams) (db.Users, error) {
	user, err := r.db.GetUserByProviderID(ctx, req)
	if err != nil {
		r.log.ErrorWithID(ctx, "[Repository: GetUserByProviderID] Error getting user by provider ID", err)
		return db.Users{}, app_error.HandleDatabaseError(err)
	}
	return user, nil
}

func (r *userRepository) UpdateUserLoginInfo(ctx context.Context, req db.UpdateUserLoginInfoParams) error {
	err := r.db.UpdateUserLoginInfo(ctx, req)
	if err != nil {
		r.log.ErrorWithID(ctx, "[Repository: UpdateUserLoginInfo] Error updating user login info", err)
		return app_error.HandleDatabaseError(err)
	}
	return nil
}
