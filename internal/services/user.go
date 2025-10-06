package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gitlab.com/interview-simulation/interview-backend-server/internal/config"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	db "gitlab.com/interview-simulation/interview-backend-server/internal/db/sqlc"
	"gitlab.com/interview-simulation/interview-backend-server/internal/entities"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/auth"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/database"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/queue/publisher"
	"gitlab.com/interview-simulation/interview-backend-server/internal/middleware"
	"gitlab.com/interview-simulation/interview-backend-server/internal/repositories"
	"gitlab.com/interview-simulation/interview-backend-server/internal/utils"
)

type UserService interface {
	HealthCheck(ctx context.Context) (entities.HealthCheckResponse, error)
	SignOut(ctx context.Context) error
	HandleGoogleCallback(ctx context.Context, req *entities.HandleGoogleCallbackReq) (*entities.HandleGoogleCallbackResp, error)
	GetGoogleAuthURL(ctx context.Context) (*entities.GoogleAuthURLResponse, error)
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
	googleClient       auth.GoogleClient
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
	googleClient auth.GoogleClient,
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
		googleClient:       googleClient,
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

func (s *userService) HandleGoogleCallback(ctx context.Context, req *entities.HandleGoogleCallbackReq) (*entities.HandleGoogleCallbackResp, error) {

	token, err := s.googleClient.ExchangeCodeForToken(ctx, req.Code)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Google OAuth] Failed to exchange code for token", err)
		return nil, app_error.New(err, app_error.ErrCodeAuthInvalidToken)
	}

	redisKey := fmt.Sprintf("%s%s", constants.RedisPrefixOAuthState, req.State)
	state, err := s.redisClient.Get(ctx, redisKey)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Google OAuth] Failed to get state", err)
		return nil, app_error.New(err, app_error.ErrCodeAuthInvalidToken)
	}

	if state != "valid" {
		s.log.ErrorWithID(ctx, "[Google OAuth] Invalid state", errors.New("invalid state"))
		return nil, app_error.New(errors.New("invalid state"), app_error.ErrCodeAuthInvalidToken)
	}

	userInfo, err := s.googleClient.GetUserInfo(ctx, token)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Google OAuth] Failed to get user info", err)
		return nil, app_error.New(err, app_error.ErrCodeAuthInvalidToken)
	}

	if userInfo.Email == "" || userInfo.ID == "" {
		s.log.ErrorWithID(ctx, "[Google OAuth] Invalid user info from Google")
		return nil, app_error.New(errors.New("invalid user information from Google"), app_error.ErrCodeAuthInvalidToken)
	}

	checkUserParams := db.CheckUserExistsByProviderIDParams{
		ProviderID: sql.NullString{String: userInfo.ID, Valid: true},
		Provider:   constants.GoogleProvider,
	}

	userExists, err := s.userRepo.CheckUserExistsByProviderID(ctx, checkUserParams)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Google OAuth] Database lookup failed", err)
		return nil, app_error.HandleDatabaseError(err)
	}

	clientIP := ""
	userAgent := ""
	if ip, ok := ctx.Value(constants.ClientIPKey).(string); ok {
		clientIP = ip
	}
	if ua, ok := ctx.Value(constants.UserAgentKey).(string); ok {
		userAgent = ua
	}

	var dbUser db.Users
	if !userExists {
		newUserId := s.generator.GenerateUUID(ctx)
		createUserParams := db.CreateUserWithProviderParams{
			ID:                 newUserId,
			Email:              userInfo.Email,
			FullName:           userInfo.Name,
			ProfileImage:       sql.NullString{String: userInfo.Picture, Valid: userInfo.Picture != ""},
			Provider:           constants.GoogleProvider,
			ProviderID:         sql.NullString{String: userInfo.ID, Valid: true},
			IsAdmin:            sql.NullBool{Bool: false, Valid: true},
			IsSuspended:        sql.NullBool{Bool: false, Valid: true},
			LastLoginAt:        sql.NullTime{Time: time.Now(), Valid: true},
			LastLoginIp:        sql.NullString{String: clientIP, Valid: clientIP != ""},
			LastLoginUserAgent: sql.NullString{String: userAgent, Valid: userAgent != ""},
			Locale:             sql.NullString{String: userInfo.Locale, Valid: userInfo.Locale != ""},
		}

		if err := s.userRepo.CreateUserWithProvider(ctx, &createUserParams); err != nil {
			s.log.ErrorWithID(ctx, "[Google OAuth] Failed to create user", err)
			return nil, app_error.HandleDatabaseError(err)
		}

		dbUser, err = s.userRepo.GetUserByProviderID(ctx, db.GetUserByProviderIDParams{
			ProviderID: sql.NullString{String: userInfo.ID, Valid: true},
			Provider:   constants.GoogleProvider,
		})
		if err != nil {
			s.log.ErrorWithID(ctx, "[Google OAuth] Failed to retrieve created user", err)
			return nil, app_error.HandleDatabaseError(err)
		}

	} else {
		getUserParams := db.GetUserByProviderIDParams{
			ProviderID: sql.NullString{String: userInfo.ID, Valid: true},
			Provider:   constants.GoogleProvider,
		}

		existingUser, err := s.userRepo.GetUserByProviderID(ctx, getUserParams)
		if err != nil {
			s.log.ErrorWithID(ctx, "[Google OAuth] Failed to fetch existing user", err)
			return nil, app_error.HandleDatabaseError(err)
		}

		dbUser = existingUser

		if err := s.userRepo.UpdateUserLoginInfo(ctx, db.UpdateUserLoginInfoParams{
			ID:                 dbUser.ID,
			LastLoginIp:        sql.NullString{String: clientIP, Valid: clientIP != ""},
			LastLoginUserAgent: sql.NullString{String: userAgent, Valid: userAgent != ""},
			Locale:             sql.NullString{String: userInfo.Locale, Valid: userInfo.Locale != ""},
		}); err != nil {
			s.log.WarnWithID(ctx, "[Google OAuth] Failed to update login info", err)
		}
	}

	accessToken, _, err := s.jwtToken.CreateToken(ctx, &entities.TokenRequest{
		UserID:   dbUser.ID.String(),
		Role:     constants.UserRoleUser,
		Duration: s.config.AuthConfig.AccessTokenDuration,
	})

	if err != nil {
		s.log.ErrorWithID(ctx, "[Google OAuth] Failed to create access token", err)
		return nil, fmt.Errorf("failed to create access token: %w", err)
	}

	refreshToken, refreshPayload, err := s.jwtToken.CreateToken(ctx, &entities.TokenRequest{
		UserID:   dbUser.ID.String(),
		Role:     constants.UserRoleUser,
		Duration: s.config.AuthConfig.RefreshTokenDuration,
	})
	if err != nil {
		s.log.ErrorWithID(ctx, "[Google OAuth] Failed to create refresh token", err)
		return nil, fmt.Errorf("failed to create refresh token: %w", err)
	}

	sessionID := refreshPayload.ID
	refreshTokenHash := s.jwtToken.HashTokenSHA256(ctx, refreshToken)

	createSessionParams := db.CreateAuthSessionParams{
		ID:               sessionID,
		UserID:           dbUser.ID,
		RefreshTokenHash: refreshTokenHash,
		UserAgent:        userAgent,
		IpAddress:        clientIP,
		LastActive:       sql.NullTime{Time: time.Now(), Valid: true},
		ExpiresAt:        sql.NullTime{Time: refreshPayload.ExpiredAt, Valid: true},
	}

	if err := s.sessionRepo.CreateAuthSession(ctx, &createSessionParams); err != nil {
		s.log.ErrorWithID(ctx, "[Google OAuth] Failed to create auth session", err)
		return nil, app_error.HandleDatabaseError(err)
	}

	result := &entities.HandleGoogleCallbackResp{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}

	return result, nil
}

func (s *userService) GetGoogleAuthURL(ctx context.Context) (*entities.GoogleAuthURLResponse, error) {
	state := s.generator.GenerateCryptographicallySecureString(ctx, 32)

	redisPayload := database.RedisPayload{
		Key:   fmt.Sprintf("%s%s", constants.RedisPrefixOAuthState, state),
		Value: "valid",
		TTL:   constants.RedisTTLOAuthState,
	}

	if err := s.redisClient.Set(ctx, redisPayload); err != nil {
		s.log.ErrorWithID(ctx, "[Google OAuth] Failed to store state", err)
		return nil, err
	}

	authURL := s.googleClient.GetAuthURL(state)
	resp := &entities.GoogleAuthURLResponse{
		AuthURL: authURL,
	}

	return resp, nil
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
