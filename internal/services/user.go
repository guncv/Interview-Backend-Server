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
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/queue"
	"gitlab.com/interview-simulation/interview-backend-server/internal/middleware"
	"gitlab.com/interview-simulation/interview-backend-server/internal/repositories"
	"gitlab.com/interview-simulation/interview-backend-server/internal/utils"
)

type UserService interface {
	HealthCheck(ctx context.Context) (entities.HealthCheckResponse, error)
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

			return &entities.SignUpUserResponse{
				TokenId: resp.ID.String(),
			}, nil
		}
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

	return &entities.SignUpUserResponse{
		TokenId: resp.ID.String(),
	}, nil
}
