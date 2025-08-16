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

	refreshToken, _, err := s.jwtToken.CreateToken(ctx, refreshTokenRequest)
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
		SessionID:        s.generator.GenerateUUID(ctx),
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
