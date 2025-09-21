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

type UserRepository interface {
	HealthCheck(ctx context.Context) (string, error)
	CheckIsUserExistsByID(ctx context.Context, id uuid.UUID) (*db.Users, error)
	CheckIsEmailExists(ctx context.Context, email string) (*db.Users, error)
	CreateUser(ctx context.Context, req *db.CreateUserParams) (*db.Users, error)
	UpdateUser(ctx context.Context, req *db.UpdateUserParams) (*db.Users, error)
	VerifyEmail(ctx context.Context, userID uuid.UUID) error
	SignInUserByEmailAndPasswordTx(ctx context.Context, req *SignInUserByEmailAndPasswordTxModel) error
	ResetUserPasswordAndUpdateResetTokenTx(ctx context.Context, req *ResetUserPasswordTxModel) error
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

func (r *userRepository) CheckIsUserExistsByID(ctx context.Context, id uuid.UUID) (*db.Users, error) {

	user, err := r.db.CheckIsUserExistsByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, app_error.New(err, app_error.ErrCodeAuthUserNotFound)
		}
		r.log.ErrorWithID(ctx, "[Repository: CheckIsUserExistsByID] Error checking if user exists", err)
		return nil, app_error.New(err, app_error.ErrCodeGeneralServerUnavailable)
	}

	return &user, nil
}

func (r *userRepository) CheckIsEmailExists(ctx context.Context, email string) (*db.Users, error) {
	r.log.InfoWithID(ctx, "[Repository: CheckIsEmailExists] Called")

	user, err := r.db.CheckIsEmailExists(ctx, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.log.ErrorWithID(ctx, "[Repository: CheckIsEmailExists] User not found", err)
			return nil, app_error.New(err, app_error.ErrCodeAuthUserNotFound)
		}
		r.log.ErrorWithID(ctx, "[Repository: CheckIsEmailExists] Error checking if email exists", err)
		return nil, app_error.New(err, app_error.ErrCodeGeneralServerUnavailable)
	}

	return &user, nil
}

func (r *userRepository) UpdateUser(ctx context.Context, req *db.UpdateUserParams) (*db.Users, error) {
	r.log.InfoWithID(ctx, "[Repository: UpdateUser] Called")

	user, err := r.db.UpdateUser(ctx, *req)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.log.ErrorWithID(ctx, "[Repository: UpdateUser] User not found", err)
			return nil, app_error.New(err, app_error.ErrCodeAuthUserNotFound)
		}
		r.log.ErrorWithID(ctx, "[Repository: UpdateUser] Error updating user", err)
		return nil, app_error.New(err, app_error.ErrCodeGeneralServerUnavailable)
	}

	return &user, nil
}

func (r *userRepository) VerifyEmail(ctx context.Context, userID uuid.UUID) error {
	r.log.InfoWithID(ctx, "[Repository: VerifyEmail] Called")

	rowsAffected, err := r.db.VerifyEmail(ctx, userID)
	if err != nil {
		r.log.ErrorWithID(ctx, "[Repository: VerifyEmail] Error verifying email", err)
		return app_error.New(err, app_error.ErrCodeGeneralServerUnavailable)
	}

	if rowsAffected == 0 {
		r.log.ErrorWithID(ctx, "[Repository: VerifyEmail] User not found", errors.New("user not found"))
		return app_error.New(errors.New("user not found"), app_error.ErrCodeAuthUserNotFound)
	}

	return nil
}

func (r *userRepository) SignInUserByEmailAndPasswordTx(ctx context.Context, req *SignInUserByEmailAndPasswordTxModel) error {
	r.log.InfoWithID(ctx, "[Repository: SignInUserByEmailAndPasswordTx] Called")

	err := r.db.ExecTx(ctx, func(q *db.Queries) error {
		userReq := db.SignInUserByEmailAndPasswordParams{
			Email:              req.Email,
			LastLoginAt:        sql.NullTime{Time: req.LastLoginAt, Valid: true},
			LastLoginIp:        sql.NullString{String: req.IpAddress, Valid: true},
			LastLoginUserAgent: sql.NullString{String: req.UserAgent, Valid: true},
			UpdatedAt:          sql.NullTime{Time: req.UpdatedAt, Valid: true},
		}

		rowAffected, err := q.SignInUserByEmailAndPassword(ctx, userReq)
		if err != nil {
			r.log.ErrorWithID(ctx, "[Repository: SignInUserByEmailAndPasswordTx] Error signing in user", err)
			return app_error.HandleDatabaseError(err)
		}

		if rowAffected == 0 {
			err := errors.New("user not found")
			r.log.ErrorWithID(ctx, "[Repository: SignInUserByEmailAndPasswordTx] User not found", err)
			return app_error.New(err, app_error.ErrCodeAuthUserNotFound)
		}

		sessionReq := db.CreateAuthSessionParams{
			ID:               req.SessionID,
			UserID:           req.UserID,
			RefreshTokenHash: req.RefreshTokenHash,
			UserAgent:        req.UserAgent,
			IpAddress:        req.IpAddress,
			LastActive:       sql.NullTime{Time: req.LastActive, Valid: true},
			ExpiresAt:        sql.NullTime{Time: req.ExpiresAt, Valid: true},
		}

		err = q.CreateAuthSession(ctx, sessionReq)
		if err != nil {
			r.log.ErrorWithID(ctx, "[Repository: SignInUserByEmailAndPasswordTx] Error creating session", err)
			return app_error.HandleDatabaseError(err)
		}

		return nil
	})

	if err != nil {
		r.log.ErrorWithID(ctx, "[Repository: SignInUserByEmailAndPasswordTx] Transaction failed", err)
		return err
	}

	return nil
}

func (r *userRepository) ResetUserPasswordAndUpdateResetTokenTx(ctx context.Context, req *ResetUserPasswordTxModel) error {
	r.log.InfoWithID(ctx, "[Repository: ResetUserPasswordAndUpdateResetTokenTx] Called")

	err := r.db.ExecTx(ctx, func(q *db.Queries) error {

		rowAffected, err := q.ResetUserPassword(ctx, db.ResetUserPasswordParams{
			ID:           req.UserID,
			PasswordHash: req.PasswordHash,
			UpdatedAt:    sql.NullTime{Time: req.UpdatedAt, Valid: true},
		})
		if err != nil {
			r.log.ErrorWithID(ctx, "[Repository: ResetUserPasswordAndUpdateResetTokenTx] Error resetting user password", err)
			return app_error.HandleDatabaseError(err)
		}

		if rowAffected == 0 {
			err := errors.New("user not found")
			r.log.ErrorWithID(ctx, "[Repository: ResetUserPasswordAndUpdateResetTokenTx] User not found", err)
			return app_error.New(err, app_error.ErrCodeAuthUserNotFound)
		}

		resetTokenReq := db.UpdateResetTokenUsedParams{
			TokenHash: req.ResetToken,
			UsedAt:    sql.NullTime{Time: req.UpdatedAt, Valid: true},
		}

		rowAffected, err = q.UpdateResetTokenUsed(ctx, resetTokenReq)
		if err != nil {
			r.log.ErrorWithID(ctx, "[Repository: ResetUserPasswordAndUpdateResetTokenTx] Error updating reset token used", err)
			return app_error.HandleDatabaseError(err)
		}

		if rowAffected == 0 {
			r.log.ErrorWithID(ctx, "[Repository: ResetUserPasswordAndUpdateResetTokenTx] Reset token not found", err)
			return app_error.New(errors.New("reset token not found"), app_error.ErrCodeAuthResetTokenNotFound)
		}

		return nil
	})
	if err != nil {
		r.log.ErrorWithID(ctx, "[Repository: ResetUserPasswordAndUpdateResetTokenTx] Transaction failed", err)
		return app_error.HandleDatabaseError(err)
	}

	return nil
}
