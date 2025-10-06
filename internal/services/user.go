package services

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"gitlab.com/interview-simulation/interview-backend-server/internal/config"
	db "gitlab.com/interview-simulation/interview-backend-server/internal/db/sqlc"
	"gitlab.com/interview-simulation/interview-backend-server/internal/entities"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/database"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/queue/publisher"
	"gitlab.com/interview-simulation/interview-backend-server/internal/middleware"
	"gitlab.com/interview-simulation/interview-backend-server/internal/repositories"
	"gitlab.com/interview-simulation/interview-backend-server/internal/utils"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type UserService interface {
	HealthCheck(ctx context.Context) (entities.HealthCheckResponse, error)
	SignOut(ctx context.Context) error
}

type userService struct {
	log                *log.Logger
	userRepo           repositories.UserRepository
	sessionRepo        repositories.AuthSessionRepository
	jwtToken           utils.JwtToken
	config             *config.Config
	db                 db.Store
	authContext        middleware.AuthContext
	redisClient        database.RedisClient
	redisTaskPublisher publisher.RedisTaskPublisher
	password           utils.PasswordUtil
	generator          utils.Generator
}

func NewUserService(l *log.Logger,
	r repositories.UserRepository,
	s repositories.AuthSessionRepository,
	jt utils.JwtToken,
	db db.Store,
	config *config.Config,
	authContext middleware.AuthContext,
	redisClient database.RedisClient,
	redisTaskPublisher publisher.RedisTaskPublisher,
	password utils.PasswordUtil,
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
		redisClient:        redisClient,
		redisTaskPublisher: redisTaskPublisher,
		password:           password,
		generator:          generator,
	}
}

func (s *userService) HealthCheck(ctx context.Context) (entities.HealthCheckResponse, error) {

	res, err := s.userRepo.HealthCheck(ctx)
	if err != nil {
		return entities.HealthCheckResponse{}, app_error.New(err, app_error.ErrCodeGeneralServerUnavailable)
	}

	result := entities.HealthCheckResponse{
		Status: res,
	}

	return result, nil
}

func (s *userService) HandleGoogleCallback(ctx context.Context, code string) (map[string]interface{}, error) {
	conf := &oauth2.Config{
		ClientID:     s.config.OAuthConfig.OAuthGoogleClientID,
		ClientSecret: s.config.OAuthConfig.OAuthGoogleClientSecret,
		RedirectURL:  s.config.OAuthConfig.OAuthGoogleRedirectURI,
		Scopes:       []string{"email", "profile"},
		Endpoint:     google.Endpoint,
	}

	token, err := conf.Exchange(ctx, code)
	if err != nil {
		return nil, errors.New("failed to exchange auth code: " + err.Error())
	}

	client := conf.Client(ctx, token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return nil, errors.New("failed to fetch user info: " + err.Error())
	}
	defer resp.Body.Close()

	var userInfo map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, err
	}

	// // Map user info
	// user := &repositories.User{
	// 	Email:      userInfo["email"].(string),
	// 	Name:       userInfo["name"].(string),
	// 	Picture:    userInfo["picture"].(string),
	// 	Provider:   "google",
	// 	ProviderID: userInfo["id"].(string),
	// }

	// // Create or update in DB
	// err = s.UserRepo.CreateOrUpdate(ctx, user)
	// if err != nil {
	// 	return nil, err
	// }

	return userInfo, nil
}

func (s *userService) SignOut(ctx context.Context) error {

	authCtx, err := s.authContext.GetAuthContext(ctx)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: SignOut] Error getting auth context", err)
		return app_error.New(err, app_error.ErrCodeAuthInvalidHeader)
	}

	if authCtx == nil || authCtx.Payload == nil || authCtx.Payload.ID == uuid.Nil {
		s.log.ErrorWithID(ctx, "[Service: SignOut] Invalid auth payload", errors.New("invalid auth payload"))
		return app_error.New(errors.New("invalid auth payload"), app_error.ErrCodeAuthInvalidToken)
	}

	if err := s.sessionRepo.RevokeAuthSessionByID(ctx, authCtx.Payload.ID); err != nil {
		s.log.ErrorWithID(ctx, "[Service: SignOut] Error revoking session", err)
		return err
	}

	return nil
}
