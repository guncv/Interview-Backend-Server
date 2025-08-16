package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"gitlab.com/interview-simulation/interview-backend-server/internal/config"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	db "gitlab.com/interview-simulation/interview-backend-server/internal/db/sqlc"
	"gitlab.com/interview-simulation/interview-backend-server/internal/entities"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/database"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/email"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/queue"
	"gitlab.com/interview-simulation/interview-backend-server/internal/middleware"
	"gitlab.com/interview-simulation/interview-backend-server/internal/repositories"
	"gitlab.com/interview-simulation/interview-backend-server/internal/utils"
)

type UserService interface {
	HealthCheck(ctx context.Context) (entities.HealthCheckResponse, error)
	SignUpUser(ctx context.Context, req *entities.SignUpUserRequest) (*entities.SignUpUserResponse, error)
	SendVerifyEmail(ctx context.Context, req *entities.VerifyEmailRequest) error
	ResetVerifyEmailCode(ctx context.Context, req *entities.ResetVerifyEmailCodeRequest) (*entities.ResetVerifyEmailCodeResponse, error)
	SignInUserByEmailAndPassword(ctx context.Context, req *entities.SignInUserByEmailAndPasswordRequest) (*entities.SignInUserByEmailAndPasswordResponse, error)
	ForgotPassword(ctx context.Context, req *entities.ForgotPasswordRequest) error
	ResetUserPassword(ctx context.Context, req *entities.ResetUserPasswordRequest) error
	RefreshToken(ctx context.Context, req *entities.RefreshTokenRequest) (*entities.RefreshTokenResponse, error)
	SignOut(ctx context.Context) error
}

type userService struct {
	log                *log.Logger
	userRepo           repositories.UserRepository
	sessionRepo        repositories.SessionRepository
	jwtToken           utils.JwtToken
	config             *config.Config
	db                 db.Store
	authContext        middleware.AuthContext
	resetTokenRepo     repositories.ResetTokenRepository
	redisClient        database.RedisClient
	redisTaskPublisher queue.RedisTaskPublisher
	password           utils.Password
	generator          utils.Generator
}

func NewUserService(l *log.Logger,
	r repositories.UserRepository,
	s repositories.SessionRepository,
	jt utils.JwtToken,
	db db.Store,
	config *config.Config,
	authContext middleware.AuthContext,
	resetTokenRepository repositories.ResetTokenRepository,
	redisClient database.RedisClient,
	redisTaskPublisher queue.RedisTaskPublisher,
	password utils.Password,
	generator utils.Generator,
) UserService {
	return &userService{
		log:                l,
		userRepo:           r,
		sessionRepo:        s,
		jwtToken:           jt,
		db:                 db,
		config:             config,
		authContext:        authContext,
		resetTokenRepo:     resetTokenRepository,
		redisClient:        redisClient,
		redisTaskPublisher: redisTaskPublisher,
		password:           password,
		generator:          generator,
	}
}

func (s *userService) HealthCheck(ctx context.Context) (entities.HealthCheckResponse, error) {
	s.log.InfoWithID(ctx, "[Service: HealthCheck] Called")

	res, err := s.userRepo.HealthCheck(ctx)
	if err != nil {
		return entities.HealthCheckResponse{}, app_error.New(err, app_error.ErrCodeGeneralServerUnavailable)
	}

	result := entities.HealthCheckResponse{
		Status: res,
	}

	return result, nil
}

func (s *userService) SignUpUser(ctx context.Context, req *entities.SignUpUserRequest) (*entities.SignUpUserResponse, error) {
	s.log.InfoWithID(ctx, "[Service: SignUpUser] Called")

	user, err := s.userRepo.CheckIsEmailExists(ctx, req.Email)
	if err != nil {
		if err == sql.ErrNoRows {
			hashedPassword, err := s.password.HashPassword(ctx, req.Password)
			if err != nil {
				s.log.ErrorWithID(ctx, "[Service: SignUpUser] Error hashing password", "error", err)
				return nil, err
			}

			createUserReq := &db.CreateUserParams{
				ID:           s.generator.GenerateUUID(ctx),
				Email:        req.Email,
				PasswordHash: hashedPassword,
				FullName:     req.FullName,
				Country:      req.Country,
				City:         req.City,
				Address:      req.Address,
				PostalCode:   req.PostalCode,
				Gender:       req.Gender,
				DateOfBirth:  req.DateOfBirth,
			}

			resp, err := s.userRepo.CreateUser(ctx, createUserReq)
			if err != nil {
				s.log.ErrorWithID(ctx, "[Service: SignUpUser] Error hashing password", "error", err)
				return nil, err
			}

			code := s.generator.GenerateRandomString(ctx, 6)

			verifyEmailReq := &entities.VerifyEmailTokenRequest{
				UserID:   resp.ID.String(),
				Email:    req.Email,
				Duration: s.config.AuthConfig.VerifyEmailTokenDuration,
			}

			verifyEmailToken, _, err := s.jwtToken.CreateVerifyEmailToken(ctx, verifyEmailReq)
			if err != nil {
				s.log.ErrorWithID(ctx, "[Service: SignUpUser] Error creating verify email token", "error", err)
				return nil, err
			}

			if err = s.redisClient.Set(ctx, database.RedisPayload{
				Key:   fmt.Sprintf("%s%s", constants.RedisPrefixVerifyEmail, verifyEmailToken),
				Value: code,
				TTL:   s.config.AuthConfig.VerifyEmailTokenDuration,
			}); err != nil {
				s.log.ErrorWithID(ctx, "[Service: SignUpUser] Error setting verify email token", "error", err)
				return nil, err
			}

			if err = s.redisTaskPublisher.PublishTaskSendVerifyEmail(ctx, &email.VerifyEmailPayload{
				EmailReceiver: req.Email,
				Code:          code,
			}); err != nil {
				s.log.ErrorWithID(ctx, "[Service: SignUpUser] Error publishing task send verify email", "error", err)
				return nil, err
			}

			result := entities.SignUpUserResponse{
				TokenId: verifyEmailToken,
			}

			return &result, nil
		}
		s.log.ErrorWithID(ctx, "[Service: SignUpUser] Error checking if email exists", "error", err)
		return nil, err
	}

	if user.IsEmailVerified.Bool {
		s.log.ErrorWithID(ctx, "[Service: SignUpUser] Error", "error", errors.New("this email already exists and verified"))
		return nil, app_error.New(errors.New("this email already exists"), app_error.ErrCodeAuthUserAlreadyExists)
	}

	newHashedPassword, err := s.password.HashPassword(ctx, req.Password)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: SignUpUser] Error hashing password", "error", err)
		return nil, err
	}

	updateUserReq := &db.UpdateUserParams{
		ID:           user.ID,
		Email:        req.Email,
		PasswordHash: newHashedPassword,
		FullName:     req.FullName,
		Country:      req.Country,
		City:         req.City,
		Address:      req.Address,
		PostalCode:   req.PostalCode,
		Gender:       req.Gender,
		DateOfBirth:  req.DateOfBirth,
	}

	resp, err := s.userRepo.UpdateUser(ctx, updateUserReq)
	if err != nil {
		if err == sql.ErrNoRows {
			s.log.ErrorWithID(ctx, "[Service: SignUpUser] Error updating user", "error", err)
			return nil, err
		}
		s.log.ErrorWithID(ctx, "[Service: SignUpUser] Error updating user", "error", err)
		return nil, err
	}

	code := s.generator.GenerateRandomString(ctx, 6)

	verifyEmailReq := &entities.VerifyEmailTokenRequest{
		UserID:   resp.ID.String(),
		Email:    req.Email,
		Duration: s.config.AuthConfig.VerifyEmailTokenDuration,
	}

	verifyEmailToken, _, err := s.jwtToken.CreateVerifyEmailToken(ctx, verifyEmailReq)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: SignUpUser] Error creating verify email token", "error", err)
		return nil, err
	}

	if err = s.redisClient.Set(ctx, database.RedisPayload{
		Key:   fmt.Sprintf("%s%s", constants.RedisPrefixVerifyEmail, verifyEmailToken),
		Value: code,
		TTL:   s.config.AuthConfig.VerifyEmailTokenDuration,
	}); err != nil {
		s.log.ErrorWithID(ctx, "[Service: SignUpUser] Error setting verify email token", "error", err)
		return nil, err
	}

	if err = s.redisClient.Set(ctx, database.RedisPayload{
		Key:   fmt.Sprintf("%s%s", constants.RedisAttemptPrefixVerifyEmail, verifyEmailToken),
		Value: "0",
		TTL:   s.config.AuthConfig.VerifyEmailTokenDuration,
	}); err != nil {
		s.log.ErrorWithID(ctx, "[Service: SignUpUser] Error setting attempt", "error", err)
		return nil, err
	}

	if err = s.redisTaskPublisher.PublishTaskSendVerifyEmail(ctx, &email.VerifyEmailPayload{
		EmailReceiver: req.Email,
		Code:          code,
	}); err != nil {
		s.log.ErrorWithID(ctx, "[Service: SignUpUser] Error publishing task send verify email", "error", err)
		return nil, err
	}

	result := entities.SignUpUserResponse{
		TokenId: verifyEmailToken,
	}

	return &result, nil
}

func (s *userService) SendVerifyEmail(ctx context.Context, req *entities.VerifyEmailRequest) error {
	s.log.InfoWithID(ctx, "[Service: VerifyEmail] Called")

	payload, err := s.jwtToken.VerifyVerifyEmailToken(ctx, req.Token)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: VerifyEmail] Error verifying token", "error", err)
		return err
	}

	code, err := s.redisClient.Get(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixVerifyEmail, req.Token))
	if err != nil {
		if err == redis.Nil {
			s.log.ErrorWithID(ctx, "[Service: VerifyEmail] Error getting verify email token", "error", err)
			return app_error.New(errors.New("invalid verify email token"), app_error.ErrCodeAuthInvalidVerifyEmailToken)
		}
		s.log.ErrorWithID(ctx, "[Service: VerifyEmail] Error getting verify email token", "error", err)
		return err
	}

	if code != req.Code {
		newAttempts, err := s.redisClient.Increment(ctx, fmt.Sprintf("%s%s", constants.RedisAttemptPrefixVerifyEmail, req.Token))
		if err != nil {
			s.log.ErrorWithID(ctx, "[Service: VerifyEmail] INCR attempts failed", "error", err)
			return err
		}

		if newAttempts >= int64(constants.MaxAttemptVerifyEmail) {
			if err := s.redisClient.Delete(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixVerifyEmail, req.Token), fmt.Sprintf("%s%s", constants.RedisAttemptPrefixVerifyEmail, req.Token)); err != nil {
				s.log.WarnWithID(ctx, "[Service: VerifyEmail] Cleanup after max attempts failed", "error", err)
			}

			s.log.ErrorWithID(ctx, "[Service: VerifyEmail] Max attempts reached")
			return app_error.New(errors.New("max attempt reached"), app_error.ErrCodeAuthMaxAttemptVerifyEmail)
		}

		s.log.ErrorWithID(ctx, "[Service: VerifyEmail] Invalid code", "attempt", newAttempts)
		return app_error.New(errors.New("invalid verify email code"), app_error.ErrCodeAuthInvalidVerifyEmailCode)
	}

	user, err := s.userRepo.CheckIsUserExistsByID(ctx, payload.UserID)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: VerifyEmail] Error checking if email exists", "error", err)
		return err
	}

	if user.IsEmailVerified.Bool {
		s.log.ErrorWithID(ctx, "[Service: VerifyEmail] Error", "error", errors.New("this email already verified"))
		return app_error.New(errors.New("this email already verified"), app_error.ErrCodeAuthUserAlreadyExists)
	}

	if err = s.userRepo.VerifyEmail(ctx, user.ID.String()); err != nil {
		s.log.ErrorWithID(ctx, "[Service: VerifyEmail] Error verifying email", "error", err)
		return err
	}

	if err = s.redisClient.Delete(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixVerifyEmail, req.Token)); err != nil {
		s.log.WarnWithID(ctx, "[Service: VerifyEmail] Cannot delete verify email token", "error", err)
		return err
	}

	if err = s.redisClient.Delete(ctx, fmt.Sprintf("%s%s", constants.RedisAttemptPrefixVerifyEmail, req.Token)); err != nil {
		s.log.WarnWithID(ctx, "[Service: VerifyEmail] Cannot delete attempt", "error", err)
	}

	return nil
}

func (s *userService) ResetVerifyEmailCode(ctx context.Context, req *entities.ResetVerifyEmailCodeRequest) (*entities.ResetVerifyEmailCodeResponse, error) {
	s.log.InfoWithID(ctx, "[Service: ResetVerifyEmailCode] Called")

	code := s.generator.GenerateRandomString(ctx, 6)

	if err := s.redisClient.Delete(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixVerifyEmail, req.Token)); err != nil {
		s.log.WarnWithID(ctx, "[Service: ResetVerifyEmailCode] Cannot delete verify email token", "error", err)
	}

	if err := s.redisClient.Delete(ctx, fmt.Sprintf("%s%s", constants.RedisAttemptPrefixVerifyEmail, req.Token)); err != nil {
		s.log.WarnWithID(ctx, "[Service: ResetVerifyEmailCode] Cannot delete attempt", "error", err)
	}

	newToken, payload, err := s.jwtToken.RenewVerifyEmailToken(ctx, req.Token)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: ResetVerifyEmailCode] Error verifying token", "error", err)
		return nil, err
	}

	if err := s.redisClient.Set(ctx, database.RedisPayload{
		Key:   fmt.Sprintf("%s%s", constants.RedisPrefixVerifyEmail, newToken),
		Value: code,
		TTL:   s.config.AuthConfig.VerifyEmailTokenDuration,
	}); err != nil {
		s.log.ErrorWithID(ctx, "[Service: ResetVerifyEmailCode] Error setting verify email token", "error", err)
		return nil, err
	}

	if err := s.redisClient.Set(ctx, database.RedisPayload{
		Key:   fmt.Sprintf("%s%s", constants.RedisAttemptPrefixVerifyEmail, newToken),
		Value: "0",
		TTL:   s.config.AuthConfig.VerifyEmailTokenDuration,
	}); err != nil {
		s.log.ErrorWithID(ctx, "[Service: ResetVerifyEmailCode] Error setting attempt", "error", err)
		return nil, err
	}

	if err := s.redisTaskPublisher.PublishTaskSendVerifyEmail(ctx, &email.VerifyEmailPayload{
		EmailReceiver: payload.Email,
		Code:          code,
	}); err != nil {
		s.log.ErrorWithID(ctx, "[Service: ResetVerifyEmailCode] Error publishing task send verify email", "error", err)
		return nil, err
	}

	result := entities.ResetVerifyEmailCodeResponse{
		TokenId: newToken,
	}

	return &result, nil
}

func (s *userService) SignInUserByEmailAndPassword(ctx context.Context, req *entities.SignInUserByEmailAndPasswordRequest) (*entities.SignInUserByEmailAndPasswordResponse, error) {
	s.log.InfoWithID(ctx, "[Service: SignIn] Called")

	user, err := s.userRepo.CheckIsEmailExists(ctx, req.Email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.log.ErrorWithID(ctx, "[Service: SignInUserByEmailAndPassword] Error checking if email exists", err)
			return nil, app_error.New(errors.New("this email or password is incorrect"), app_error.ErrCodeAuthInvalidPassword)
		}
		s.log.ErrorWithID(ctx, "[Service: SignInUserByEmailAndPassword] Error checking if email exists", err)
		return nil, err
	}

	if !user.IsEmailVerified.Bool {
		s.log.ErrorWithID(ctx, "[Service: SignInUserByEmailAndPassword] Error", "error", errors.New("this email is not verified"))
		return nil, app_error.New(errors.New("this email or password is incorrect"), app_error.ErrCodeAuthInvalidPassword)
	}

	if err = s.password.CheckPassword(ctx, req.Password, user.PasswordHash); err != nil {
		s.log.ErrorWithID(ctx, "[Service: SignInUserByEmailAndPassword] Password is incorrect", err)
		return nil, err
	}

	accessTokenRequest := &entities.TokenRequest{
		UserID:   user.ID.String(),
		Role:     constants.UserRoleUser,
		Duration: s.config.AuthConfig.AccessTokenDuration,
	}

	accessToken, _, err := s.jwtToken.CreateToken(ctx, accessTokenRequest)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: SignInUserByEmailAndPassword] Error creating access token", err)
		return nil, err
	}

	refreshTokenRequest := &entities.TokenRequest{
		UserID:   user.ID.String(),
		Role:     constants.UserRoleUser,
		Duration: s.config.AuthConfig.RefreshTokenDuration,
	}

	refreshToken, refreshPayload, err := s.jwtToken.CreateToken(ctx, refreshTokenRequest)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: SignInUserByEmailAndPassword] Error creating refresh token", err)
		return nil, err
	}

	refreshTokenHash := s.jwtToken.HashTokenSHA256(ctx, refreshToken)

	txModel := &repositories.SignInUserByEmailAndPasswordTxModel{
		Email:            req.Email,
		LastLoginAt:      time.Now(),
		UserAgent:        ctx.Value(constants.UserAgentKey).(string),
		IpAddress:        ctx.Value(constants.ClientIPKey).(string),
		SessionID:        refreshPayload.ID,
		UserID:           user.ID,
		UpdatedAt:        time.Now(),
		RefreshTokenHash: refreshTokenHash,
		LastActive:       time.Now().Add(s.config.AuthConfig.RefreshTokenDuration),
		ExpiresAt:        time.Now().Add(s.config.AuthConfig.RefreshTokenDuration),
	}

	if err = s.userRepo.SignInUserByEmailAndPasswordTx(ctx, txModel); err != nil {
		s.log.ErrorWithID(ctx, "[Service: SignInUserByEmailAndPassword] Error signing in user", err)
		return nil, err
	}

	resp := entities.SignInUserByEmailAndPasswordResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}

	return &resp, nil
}

func (s *userService) ForgotPassword(ctx context.Context, req *entities.ForgotPasswordRequest) error {
	s.log.InfoWithID(ctx, "[Service: ForgotPassword] Called")

	user, err := s.userRepo.CheckIsEmailExists(ctx, req.Email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.log.ErrorWithID(ctx, "[Service: ForgotPassword] Error checking if user exists", err)
			return app_error.New(errors.New("this email is not registered"), app_error.ErrCodeAuthUserNotFound)
		}
		s.log.ErrorWithID(ctx, "[Service: ForgotPassword] Error checking if user exists", err)
		return err
	}

	if !user.IsEmailVerified.Bool {
		s.log.ErrorWithID(ctx, "[Service: ForgotPassword] Error", "error", errors.New("this email is not verified"))
		return app_error.New(errors.New("this email is not verified"), app_error.ErrCodeAuthEmailNotVerified)
	}

	token := s.generator.GenerateUUID(ctx).String()
	hashedToken := s.jwtToken.HashTokenSHA256(ctx, token)

	resetTokenRequest := &db.CreateResetTokenParams{
		ID:        s.generator.GenerateUUID(ctx),
		UserID:    user.ID,
		TokenHash: hashedToken,
		ExpiresAt: time.Now().Add(s.config.AuthConfig.ResetPasswordTokenDuration),
		IpAddress: sql.NullString{String: ctx.Value(constants.ClientIPKey).(string), Valid: true},
		UserAgent: sql.NullString{String: ctx.Value(constants.UserAgentKey).(string), Valid: true},
	}

	if err := s.resetTokenRepo.CreateResetToken(ctx, resetTokenRequest); err != nil {
		s.log.ErrorWithID(ctx, "[Service: ForgotPassword] Error creating reset token", err)
		return err
	}

	redisPayload := database.RedisPayload{
		Key:   hashedToken,
		Value: user.ID.String(),
		TTL:   s.config.AuthConfig.ResetPasswordTokenDuration,
	}

	if err := s.redisClient.Set(ctx, redisPayload); err != nil {
		s.log.WarnWithID(ctx, "[Service: ForgotPassword] Error setting reset password token", err)
	}

	emailPayload := &email.ResetPasswordEmailPayload{
		EmailReceiver: req.Email,
		Token:         token,
	}

	opts := s.redisTaskPublisher.DefineTaskOptions(constants.TaskSendResetPasswordEmail)
	if err := s.redisTaskPublisher.PublishTaskSendResetPasswordEmail(ctx, emailPayload, opts...); err != nil {
		s.log.ErrorWithID(ctx, "[Service: ForgotPassword] Error sending reset password email", err)
		return err
	}

	s.log.InfoWithID(ctx, "[Service: ForgotPassword] forgot password successfully ", token)

	return nil
}

func (s *userService) ResetUserPassword(ctx context.Context, req *entities.ResetUserPasswordRequest) error {
	s.log.InfoWithID(ctx, "[Service: ResetUserPassword] Called")

	hashedToken := s.jwtToken.HashTokenSHA256(ctx, req.Token)

	userID, err := s.redisClient.Get(ctx, hashedToken)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			s.log.WarnWithID(ctx, "[Service: ResetUserPassword] Redis Miss, fallback to DB")
		} else {
			s.log.ErrorWithID(ctx, "[Service: ResetUserPassword] Unexpected Redis failure", err)
		}

		resetToken, err := s.resetTokenRepo.GetResetToken(ctx, hashedToken)
		if err != nil {
			s.log.ErrorWithID(ctx, "[Service: ResetUserPassword] Error checking if reset token exists", err)
			return err
		}

		if resetToken.Used {
			s.log.ErrorWithID(ctx, "[Service: ResetUserPassword] Reset token already used", errors.New("reset token already used"))
			return app_error.New(errors.New("reset token already used"), app_error.ErrCodeAuthResetTokenUsed)
		}

		if resetToken.ExpiresAt.Before(time.Now()) {
			s.log.ErrorWithID(ctx, "[Service: ResetUserPassword] Reset token expired", errors.New("reset token expired"))
			return app_error.New(errors.New("reset token expired"), app_error.ErrCodeAuthResetTokenExpired)
		}

		userID = resetToken.UserID.String()
	}

	user, err := s.userRepo.CheckIsUserExistsByID(ctx, userID)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: ResetUserPassword] Error checking if user exists", err)
		return err
	}

	if err = s.password.CheckPassword(ctx, req.NewPassword, user.PasswordHash); err == nil {
		error := errors.New("password is the same as the old password")
		s.log.ErrorWithID(ctx, "[Service: ResetUserPassword] Password is the same as the old password", app_error.New(error, app_error.ErrCodeAuthPasswordSameAsOld))
		return app_error.New(error, app_error.ErrCodeAuthPasswordSameAsOld)
	}

	newHashedPassword, err := s.password.HashPassword(ctx, req.NewPassword)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: ResetUserPassword] Error hashing password", err)
		return err
	}

	resetUserReq := &repositories.ResetUserPasswordTxModel{
		UserID:       userID,
		PasswordHash: newHashedPassword,
		ResetToken:   hashedToken,
		UpdatedAt:    time.Now(),
	}

	if err := s.userRepo.ResetUserPasswordAndUpdateResetTokenTx(ctx, resetUserReq); err != nil {
		s.log.ErrorWithID(ctx, "[Service: ResetUserPassword] Error resetting user password", err)
		return err
	}

	if err := s.redisClient.Delete(ctx, hashedToken); err != nil {
		s.log.WarnWithID(ctx, "[Service: ResetUserPassword] Error deleting reset token", err)
	}

	return nil
}

func (s *userService) RefreshToken(ctx context.Context, req *entities.RefreshTokenRequest) (*entities.RefreshTokenResponse, error) {
	s.log.InfoWithID(ctx, "[Service: RefreshToken] Called")

	refreshPayload, err := s.jwtToken.VerifyToken(ctx, req.RefreshToken)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: RefreshToken] Error verifying token", err)
		return nil, err
	}

	session, err := s.sessionRepo.GetSessionByID(ctx, refreshPayload.ID)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: RefreshToken] Error getting session", err)
		return nil, err
	}

	if session.IsRevoked.Bool {
		s.log.ErrorWithID(ctx, "[Service: RefreshToken] Session is revoked", errors.New("session is revoked"))
		return nil, app_error.New(errors.New("session is revoked"), app_error.ErrCodeAuthInvalidToken)
	}

	tokenRequest := &entities.TokenRequest{
		UserID:   session.UserID.String(),
		Role:     refreshPayload.Role,
		Duration: s.config.AuthConfig.AccessTokenDuration,
	}

	accessToken, _, err := s.jwtToken.CreateToken(ctx, tokenRequest)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: RefreshToken] Error creating access token", err)
		return nil, err
	}

	return &entities.RefreshTokenResponse{
		AccessToken: accessToken,
	}, nil
}

func (s *userService) SignOut(ctx context.Context) error {
	s.log.InfoWithID(ctx, "[Service: SignOut] Called")

	authCtx, err := s.authContext.GetAuthContext(ctx)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: SignOut] Error getting auth context", err)
		return app_error.New(err, app_error.ErrCodeAuthInvalidHeader)
	}

	if err := s.sessionRepo.RevokeSessionByID(ctx, authCtx.Payload.ID); err != nil {
		s.log.ErrorWithID(ctx, "[Service: SignOut] Error revoking session", err)
		return err
	}

	return nil
}
