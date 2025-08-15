package services

import (
	"context"
	"database/sql"
	"errors"

	"gitlab.com/interview-simulation/interview-backend-server/internal/config"
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
	VerifyEmail(ctx context.Context, req *entities.VerifyEmailRequest) error
}

type userService struct {
	log                  *log.Logger
	userRepository       repositories.UserRepository
	jwtToken             utils.JwtToken
	config               *config.Config
	db                   db.Store
	authContext          middleware.AuthContext
	resetTokenRepository repositories.ResetTokenRepository
	redisClient          database.RedisClient
	redisTaskPublisher   queue.RedisTaskPublisher
	password             utils.Password
	generator            utils.Generator
}

func NewUserService(l *log.Logger,
	r repositories.UserRepository,
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
		log:                  l,
		userRepository:       r,
		jwtToken:             jt,
		db:                   db,
		config:               config,
		authContext:          authContext,
		resetTokenRepository: resetTokenRepository,
		redisClient:          redisClient,
		redisTaskPublisher:   redisTaskPublisher,
		password:             password,
		generator:            generator,
	}
}

func (s *userService) HealthCheck(ctx context.Context) (entities.HealthCheckResponse, error) {
	s.log.InfoWithID(ctx, "[Service: HealthCheck] Called")

	res, err := s.userRepository.HealthCheck(ctx)
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

	user, err := s.userRepository.CheckIsEmailExists(ctx, req.Email)
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

			resp, err := s.userRepository.CreateUser(ctx, createUserReq)
			if err != nil {
				s.log.ErrorWithID(ctx, "[Service: SignUpUser] Error hashing password", "error", err)
				return nil, err
			}

			code := s.generator.GenerateRandomString(ctx, 6)

			verifyEmailReq := &entities.VerifyEmailTokenRequest{
				UserID:   resp.ID.String(),
				Code:     code,
				Duration: s.config.AuthConfig.VerifyEmailTokenDuration,
			}

			verifyEmailToken, _, err := s.jwtToken.CreateVerifyEmailToken(ctx, verifyEmailReq)
			if err != nil {
				s.log.ErrorWithID(ctx, "[Service: SignUpUser] Error creating verify email token", "error", err)
				return nil, err
			}

			if err = s.redisClient.Set(ctx, database.RedisPayload{
				Key:   verifyEmailToken,
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

	resp, err := s.userRepository.UpdateUser(ctx, updateUserReq)
	if err != nil {
		if err == sql.ErrNoRows {
			s.log.ErrorWithID(ctx, "[Service: SignUpUser] Error updating user", "error", err)
			return nil, err
		}
		s.log.ErrorWithID(ctx, "[Service: SignUpUser] Error updating user", "error", err)
		return nil, err
	}

	code := s.generator.GenerateRandomString(ctx, 6)

	s.log.InfoWithID(ctx, "[Service: SignUpUser] Code", code)

	verifyEmailReq := &entities.VerifyEmailTokenRequest{
		UserID:   resp.ID.String(),
		Code:     code,
		Duration: s.config.AuthConfig.VerifyEmailTokenDuration,
	}

	verifyEmailToken, _, err := s.jwtToken.CreateVerifyEmailToken(ctx, verifyEmailReq)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: SignUpUser] Error creating verify email token", "error", err)
		return nil, err
	}

	if err = s.redisClient.Set(ctx, database.RedisPayload{
		Key:   verifyEmailToken,
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

func (s *userService) VerifyEmail(ctx context.Context, req *entities.VerifyEmailRequest) error {
	s.log.InfoWithID(ctx, "[Service: VerifyEmail] Called")

	payload, err := s.jwtToken.VerifyVerifyEmailToken(ctx, req.Token)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: VerifyEmail] Error verifying token", "error", err)
		return err
	}

	code, err := s.redisClient.Get(ctx, req.Token)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: VerifyEmail] Error getting verify email token", "error", err)
		return err
	}

	if code != req.Code {
		s.log.ErrorWithID(ctx, "[Service: VerifyEmail] Error verifying token", "error", errors.New("invalid code"))
		return app_error.New(errors.New("invalid verify email code"), app_error.ErrCodeAuthInvalidRequest)
	}

	user, err := s.userRepository.CheckIsUserExistsByID(ctx, payload.UserID)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: VerifyEmail] Error checking if email exists", "error", err)
		return err
	}

	if user.IsEmailVerified.Bool {
		s.log.ErrorWithID(ctx, "[Service: VerifyEmail] Error", "error", errors.New("this email already verified"))
		return app_error.New(errors.New("this email already verified"), app_error.ErrCodeAuthUserAlreadyExists)
	}

	if err = s.userRepository.VerifyEmail(ctx, user.ID.String()); err != nil {
		s.log.ErrorWithID(ctx, "[Service: VerifyEmail] Error verifying email", "error", err)
		return err
	}

	return nil
}
