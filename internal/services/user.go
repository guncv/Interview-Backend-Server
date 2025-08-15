package services

import (
	"context"

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
	userRoleRepository   repositories.UserRoleRepository
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
	ur repositories.UserRoleRepository,
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
		userRoleRepository:   ur,
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
