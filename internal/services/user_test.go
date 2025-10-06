package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.com/interview-simulation/interview-backend-server/internal/config"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	db "gitlab.com/interview-simulation/interview-backend-server/internal/db/sqlc"
	"gitlab.com/interview-simulation/interview-backend-server/internal/entities"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/auth"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/database"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
	"gitlab.com/interview-simulation/interview-backend-server/internal/middleware"
	mockAuthInfras "gitlab.com/interview-simulation/interview-backend-server/internal/mocks/auth"
	mockDatabase "gitlab.com/interview-simulation/interview-backend-server/internal/mocks/database"
	middlewareMocks "gitlab.com/interview-simulation/interview-backend-server/internal/mocks/middleware"
	"gitlab.com/interview-simulation/interview-backend-server/internal/mocks/repositories"
	mockUtils "gitlab.com/interview-simulation/interview-backend-server/internal/mocks/utils"
	utilsPkg "gitlab.com/interview-simulation/interview-backend-server/internal/utils"
	"golang.org/x/oauth2"
)

func TestUserService_HealthCheck(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()
	mockErr := errors.New("error")

	okResponse := entities.HealthCheckResponse{
		Status: "ok",
	}
	testCases := []struct {
		name   string
		setup  func() *repositories.MockUserRepository
		verify func(t *testing.T, got entities.HealthCheckResponse, gotErr error)
	}{
		{
			name: "OK",
			setup: func() *repositories.MockUserRepository {
				mockUserRepo := new(repositories.MockUserRepository)
				mockUserRepo.EXPECT().
					HealthCheck(ctx).
					Return("ok", nil)

				return mockUserRepo
			},
			verify: func(t *testing.T, got entities.HealthCheckResponse, gotErr error) {
				assert.Equal(t, okResponse, got)
				assert.NoError(t, gotErr)
			},
		},
		{
			name: "Error",
			setup: func() *repositories.MockUserRepository {
				mockUserRepo := new(repositories.MockUserRepository)
				mockUserRepo.EXPECT().
					HealthCheck(ctx).
					Return("", mockErr)

				return mockUserRepo
			},
			verify: func(t *testing.T, got entities.HealthCheckResponse, gotErr error) {
				assert.Equal(t, entities.HealthCheckResponse{}, got)
				assert.Error(t, gotErr)
				assert.ErrorIs(t, gotErr, mockErr)
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			mockUserRepo := tC.setup()
			defer mockUserRepo.AssertExpectations(t)

			svc := NewUserService(lgr, mockUserRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
			got, gotErr := svc.HealthCheck(ctx)

			tC.verify(t, got, gotErr)
		})
	}
}

func TestUserService_HandleGoogleCallback(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()

	validCode := "valid_auth_code"
	validState := "valid_oauth_state"
	invalidCode := "invalid_auth_code"
	invalidState := "invalid_oauth_state"
	expiredState := "expired_oauth_state"

	validUserInfo := &auth.GoogleUserInfo{
		Email:         "test@example.com",
		VerifiedEmail: true,
		Name:          "Test User",
		GivenName:     "Test",
		FamilyName:    "User",
		Picture:       "https://example.com/picture.jpg",
		Locale:        "en-US",
		ID:            "google_user_123",
	}

	invalidUserInfo := &auth.GoogleUserInfo{
		Email:         "",
		VerifiedEmail: false,
		Name:          "Test User",
		ID:            "google_user_456",
	}

	testCases := []struct {
		name   string
		input  *entities.HandleGoogleCallbackReq
		setup  func() (*mockAuthInfras.MockGoogleClient, *mockUtils.MockGenerator, *repositories.MockUserRepository, *repositories.MockAuthSessionRepository, *mockUtils.MockJwtToken, *mockDatabase.MockRedisClient)
		verify func(t *testing.T, got *entities.HandleGoogleCallbackResp, gotErr error)
	}{
		{
			name: "Success_NewUser",
			input: &entities.HandleGoogleCallbackReq{
				Code:  validCode,
				State: validState,
			},
			setup: func() (*mockAuthInfras.MockGoogleClient, *mockUtils.MockGenerator, *repositories.MockUserRepository, *repositories.MockAuthSessionRepository, *mockUtils.MockJwtToken, *mockDatabase.MockRedisClient) {
				mockGoogleClient := new(mockAuthInfras.MockGoogleClient)
				mockGenerator := new(mockUtils.MockGenerator)
				mockUserRepo := new(repositories.MockUserRepository)
				mockAuthSessionRepo := new(repositories.MockAuthSessionRepository)
				mockJwtToken := new(mockUtils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockGoogleClient.EXPECT().
					ExchangeCodeForToken(ctx, validCode).
					Return(&oauth2.Token{AccessToken: "google_access_token"}, nil)

				redisKey := fmt.Sprintf("%s%s", constants.RedisPrefixOAuthState, validState)
				mockRedisClient.EXPECT().
					Get(ctx, redisKey).
					Return(constants.GoogleProvider, nil)

				mockGoogleClient.EXPECT().
					GetUserInfo(ctx, &oauth2.Token{AccessToken: "google_access_token"}).
					Return(validUserInfo, nil)

				mockUserRepo.EXPECT().
					CheckUserExistsByProviderID(ctx, db.CheckUserExistsByProviderIDParams{
						ProviderID: sql.NullString{String: validUserInfo.ID, Valid: true},
						Provider:   constants.GoogleProvider,
					}).
					Return(false, nil)

				userID := uuid.New()
				mockGenerator.EXPECT().
					GenerateUUID(ctx).
					Return(userID)

				mockUserRepo.EXPECT().
					CreateUserWithProvider(ctx, mock.MatchedBy(func(params *db.CreateUserWithProviderParams) bool {
						return params.ID == userID &&
							params.Email == validUserInfo.Email &&
							params.FullName == validUserInfo.Name &&
							params.Provider == constants.GoogleProvider &&
							params.ProviderID.String == validUserInfo.ID
					})).
					Return(nil)

				createdUser := db.Users{
					ID:         userID,
					Email:      validUserInfo.Email,
					FullName:   validUserInfo.Name,
					Provider:   constants.GoogleProvider,
					ProviderID: sql.NullString{String: validUserInfo.ID, Valid: true},
				}
				mockUserRepo.EXPECT().
					GetUserByProviderID(ctx, db.GetUserByProviderIDParams{
						ProviderID: sql.NullString{String: validUserInfo.ID, Valid: true},
						Provider:   constants.GoogleProvider,
					}).
					Return(createdUser, nil)

				mockJwtToken.EXPECT().
					CreateToken(ctx, mock.MatchedBy(func(req *entities.TokenRequest) bool {
						return req.UserID == userID.String() && req.Role == constants.UserRoleUser
					})).
					Return("access_token", &utilsPkg.SignInTokenPayload{ID: uuid.New(), ExpiredAt: time.Now().Add(time.Hour)}, nil).
					Times(1)

				mockJwtToken.EXPECT().
					CreateToken(ctx, mock.MatchedBy(func(req *entities.TokenRequest) bool {
						return req.UserID == userID.String() && req.Role == constants.UserRoleUser
					})).
					Return("refresh_token", &utilsPkg.SignInTokenPayload{ID: uuid.New(), ExpiredAt: time.Now().Add(time.Hour)}, nil).
					Times(1)

				mockJwtToken.EXPECT().
					HashTokenSHA256(ctx, "refresh_token").
					Return("hashed_refresh_token")

				mockAuthSessionRepo.EXPECT().
					CreateAuthSession(ctx, mock.MatchedBy(func(params *db.CreateAuthSessionParams) bool {
						return params.UserID == userID && params.RefreshTokenHash == "hashed_refresh_token"
					})).
					Return(nil)

				return mockGoogleClient, mockGenerator, mockUserRepo, mockAuthSessionRepo, mockJwtToken, mockRedisClient
			},
			verify: func(t *testing.T, got *entities.HandleGoogleCallbackResp, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, got)
				assert.Equal(t, "access_token", got.AccessToken)
				assert.Equal(t, "refresh_token", got.RefreshToken)
			},
		},
		{
			name: "Success_ExistingUser",
			input: &entities.HandleGoogleCallbackReq{
				Code:  validCode,
				State: validState,
			},
			setup: func() (*mockAuthInfras.MockGoogleClient, *mockUtils.MockGenerator, *repositories.MockUserRepository, *repositories.MockAuthSessionRepository, *mockUtils.MockJwtToken, *mockDatabase.MockRedisClient) {
				mockGoogleClient := new(mockAuthInfras.MockGoogleClient)
				mockGenerator := new(mockUtils.MockGenerator)
				mockUserRepo := new(repositories.MockUserRepository)
				mockAuthSessionRepo := new(repositories.MockAuthSessionRepository)
				mockJwtToken := new(mockUtils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockGoogleClient.EXPECT().
					ExchangeCodeForToken(ctx, validCode).
					Return(&oauth2.Token{AccessToken: "google_access_token"}, nil)

				redisKey := fmt.Sprintf("%s%s", constants.RedisPrefixOAuthState, validState)
				mockRedisClient.EXPECT().
					Get(ctx, redisKey).
					Return(constants.GoogleProvider, nil)

				mockGoogleClient.EXPECT().
					GetUserInfo(ctx, &oauth2.Token{AccessToken: "google_access_token"}).
					Return(validUserInfo, nil)

				mockUserRepo.EXPECT().
					CheckUserExistsByProviderID(ctx, db.CheckUserExistsByProviderIDParams{
						ProviderID: sql.NullString{String: validUserInfo.ID, Valid: true},
						Provider:   constants.GoogleProvider,
					}).
					Return(true, nil)

				existingUser := db.Users{
					ID:         uuid.New(),
					Email:      validUserInfo.Email,
					FullName:   validUserInfo.Name,
					Provider:   constants.GoogleProvider,
					ProviderID: sql.NullString{String: validUserInfo.ID, Valid: true},
				}
				mockUserRepo.EXPECT().
					GetUserByProviderID(ctx, db.GetUserByProviderIDParams{
						ProviderID: sql.NullString{String: validUserInfo.ID, Valid: true},
						Provider:   constants.GoogleProvider,
					}).
					Return(existingUser, nil)

				mockUserRepo.EXPECT().
					UpdateUserLoginInfo(ctx, mock.MatchedBy(func(params db.UpdateUserLoginInfoParams) bool {
						return params.ID == existingUser.ID
					})).
					Return(nil)

				mockJwtToken.EXPECT().
					CreateToken(ctx, mock.MatchedBy(func(req *entities.TokenRequest) bool {
						return req.UserID == existingUser.ID.String() && req.Role == constants.UserRoleUser
					})).
					Return("access_token", &utilsPkg.SignInTokenPayload{ID: uuid.New(), ExpiredAt: time.Now().Add(time.Hour)}, nil).
					Times(1)

				mockJwtToken.EXPECT().
					CreateToken(ctx, mock.MatchedBy(func(req *entities.TokenRequest) bool {
						return req.UserID == existingUser.ID.String() && req.Role == constants.UserRoleUser
					})).
					Return("refresh_token", &utilsPkg.SignInTokenPayload{ID: uuid.New(), ExpiredAt: time.Now().Add(time.Hour)}, nil).
					Times(1)

				mockJwtToken.EXPECT().
					HashTokenSHA256(ctx, "refresh_token").
					Return("hashed_refresh_token")

				mockAuthSessionRepo.EXPECT().
					CreateAuthSession(ctx, mock.MatchedBy(func(params *db.CreateAuthSessionParams) bool {
						return params.UserID == existingUser.ID
					})).
					Return(nil)

				return mockGoogleClient, mockGenerator, mockUserRepo, mockAuthSessionRepo, mockJwtToken, mockRedisClient
			},
			verify: func(t *testing.T, got *entities.HandleGoogleCallbackResp, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, got)
				assert.Equal(t, "access_token", got.AccessToken)
				assert.Equal(t, "refresh_token", got.RefreshToken)
			},
		},
		{
			name: "Error_InvalidCode",
			input: &entities.HandleGoogleCallbackReq{
				Code:  invalidCode,
				State: validState,
			},
			setup: func() (*mockAuthInfras.MockGoogleClient, *mockUtils.MockGenerator, *repositories.MockUserRepository, *repositories.MockAuthSessionRepository, *mockUtils.MockJwtToken, *mockDatabase.MockRedisClient) {
				mockGoogleClient := new(mockAuthInfras.MockGoogleClient)
				mockGenerator := new(mockUtils.MockGenerator)
				mockUserRepo := new(repositories.MockUserRepository)
				mockAuthSessionRepo := new(repositories.MockAuthSessionRepository)
				mockJwtToken := new(mockUtils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockGoogleClient.EXPECT().
					ExchangeCodeForToken(ctx, invalidCode).
					Return(nil, errors.New("invalid authorization code"))

				return mockGoogleClient, mockGenerator, mockUserRepo, mockAuthSessionRepo, mockJwtToken, mockRedisClient
			},
			verify: func(t *testing.T, got *entities.HandleGoogleCallbackResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
				assert.Contains(t, gotErr.Error(), "invalid authorization code")
			},
		},
		{
			name: "Error_InvalidState",
			input: &entities.HandleGoogleCallbackReq{
				Code:  validCode,
				State: invalidState,
			},
			setup: func() (*mockAuthInfras.MockGoogleClient, *mockUtils.MockGenerator, *repositories.MockUserRepository, *repositories.MockAuthSessionRepository, *mockUtils.MockJwtToken, *mockDatabase.MockRedisClient) {
				mockGoogleClient := new(mockAuthInfras.MockGoogleClient)
				mockGenerator := new(mockUtils.MockGenerator)
				mockUserRepo := new(repositories.MockUserRepository)
				mockAuthSessionRepo := new(repositories.MockAuthSessionRepository)
				mockJwtToken := new(mockUtils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockGoogleClient.EXPECT().
					ExchangeCodeForToken(ctx, validCode).
					Return(&oauth2.Token{AccessToken: "google_access_token"}, nil)

				redisKey := fmt.Sprintf("%s%s", constants.RedisPrefixOAuthState, invalidState)
				mockRedisClient.EXPECT().
					Get(ctx, redisKey).
					Return("invalid", nil)

				return mockGoogleClient, mockGenerator, mockUserRepo, mockAuthSessionRepo, mockJwtToken, mockRedisClient
			},
			verify: func(t *testing.T, got *entities.HandleGoogleCallbackResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
				assert.Contains(t, gotErr.Error(), "invalid state")
			},
		},
		{
			name: "Error_StateNotFound",
			input: &entities.HandleGoogleCallbackReq{
				Code:  validCode,
				State: expiredState,
			},
			setup: func() (*mockAuthInfras.MockGoogleClient, *mockUtils.MockGenerator, *repositories.MockUserRepository, *repositories.MockAuthSessionRepository, *mockUtils.MockJwtToken, *mockDatabase.MockRedisClient) {
				mockGoogleClient := new(mockAuthInfras.MockGoogleClient)
				mockGenerator := new(mockUtils.MockGenerator)
				mockUserRepo := new(repositories.MockUserRepository)
				mockAuthSessionRepo := new(repositories.MockAuthSessionRepository)
				mockJwtToken := new(mockUtils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockGoogleClient.EXPECT().
					ExchangeCodeForToken(ctx, validCode).
					Return(&oauth2.Token{AccessToken: "google_access_token"}, nil)

				redisKey := fmt.Sprintf("%s%s", constants.RedisPrefixOAuthState, expiredState)
				mockRedisClient.EXPECT().
					Get(ctx, redisKey).
					Return("", errors.New("key not found"))

				return mockGoogleClient, mockGenerator, mockUserRepo, mockAuthSessionRepo, mockJwtToken, mockRedisClient
			},
			verify: func(t *testing.T, got *entities.HandleGoogleCallbackResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
				assert.Contains(t, gotErr.Error(), "key not found")
			},
		},
		{
			name: "Error_InvalidUserInfo",
			input: &entities.HandleGoogleCallbackReq{
				Code:  validCode,
				State: validState,
			},
			setup: func() (*mockAuthInfras.MockGoogleClient, *mockUtils.MockGenerator, *repositories.MockUserRepository, *repositories.MockAuthSessionRepository, *mockUtils.MockJwtToken, *mockDatabase.MockRedisClient) {
				mockGoogleClient := new(mockAuthInfras.MockGoogleClient)
				mockGenerator := new(mockUtils.MockGenerator)
				mockUserRepo := new(repositories.MockUserRepository)
				mockAuthSessionRepo := new(repositories.MockAuthSessionRepository)
				mockJwtToken := new(mockUtils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockGoogleClient.EXPECT().
					ExchangeCodeForToken(ctx, validCode).
					Return(&oauth2.Token{AccessToken: "google_access_token"}, nil)

				redisKey := fmt.Sprintf("%s%s", constants.RedisPrefixOAuthState, validState)
				mockRedisClient.EXPECT().
					Get(ctx, redisKey).
					Return(constants.GoogleProvider, nil)

				mockGoogleClient.EXPECT().
					GetUserInfo(ctx, mock.MatchedBy(func(token *oauth2.Token) bool {
						return token.AccessToken == "google_access_token"
					})).
					Return(invalidUserInfo, nil)

				return mockGoogleClient, mockGenerator, mockUserRepo, mockAuthSessionRepo, mockJwtToken, mockRedisClient
			},
			verify: func(t *testing.T, got *entities.HandleGoogleCallbackResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
				assert.Contains(t, gotErr.Error(), "invalid user information from Google")
			},
		},
		{
			name: "Error_UserInfoRetrievalFailed",
			input: &entities.HandleGoogleCallbackReq{
				Code:  validCode,
				State: validState,
			},
			setup: func() (*mockAuthInfras.MockGoogleClient, *mockUtils.MockGenerator, *repositories.MockUserRepository, *repositories.MockAuthSessionRepository, *mockUtils.MockJwtToken, *mockDatabase.MockRedisClient) {
				mockGoogleClient := new(mockAuthInfras.MockGoogleClient)
				mockGenerator := new(mockUtils.MockGenerator)
				mockUserRepo := new(repositories.MockUserRepository)
				mockAuthSessionRepo := new(repositories.MockAuthSessionRepository)
				mockJwtToken := new(mockUtils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockGoogleClient.EXPECT().
					ExchangeCodeForToken(ctx, validCode).
					Return(&oauth2.Token{AccessToken: "google_access_token"}, nil)

				redisKey := fmt.Sprintf("%s%s", constants.RedisPrefixOAuthState, validState)
				mockRedisClient.EXPECT().
					Get(ctx, redisKey).
					Return(constants.GoogleProvider, nil)

				mockGoogleClient.EXPECT().
					GetUserInfo(ctx, &oauth2.Token{AccessToken: "google_access_token"}).
					Return(nil, errors.New("failed to get user info"))

				return mockGoogleClient, mockGenerator, mockUserRepo, mockAuthSessionRepo, mockJwtToken, mockRedisClient
			},
			verify: func(t *testing.T, got *entities.HandleGoogleCallbackResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
				assert.Contains(t, gotErr.Error(), "failed to get user info")
			},
		},
		{
			name: "Error_DatabaseLookupFailed",
			input: &entities.HandleGoogleCallbackReq{
				Code:  validCode,
				State: validState,
			},
			setup: func() (*mockAuthInfras.MockGoogleClient, *mockUtils.MockGenerator, *repositories.MockUserRepository, *repositories.MockAuthSessionRepository, *mockUtils.MockJwtToken, *mockDatabase.MockRedisClient) {
				mockGoogleClient := new(mockAuthInfras.MockGoogleClient)
				mockGenerator := new(mockUtils.MockGenerator)
				mockUserRepo := new(repositories.MockUserRepository)
				mockAuthSessionRepo := new(repositories.MockAuthSessionRepository)
				mockJwtToken := new(mockUtils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockGoogleClient.EXPECT().
					ExchangeCodeForToken(ctx, validCode).
					Return(&oauth2.Token{AccessToken: "google_access_token"}, nil)

				redisKey := fmt.Sprintf("%s%s", constants.RedisPrefixOAuthState, validState)
				mockRedisClient.EXPECT().
					Get(ctx, redisKey).
					Return(constants.GoogleProvider, nil)

				mockGoogleClient.EXPECT().
					GetUserInfo(ctx, &oauth2.Token{AccessToken: "google_access_token"}).
					Return(validUserInfo, nil)

				mockUserRepo.EXPECT().
					CheckUserExistsByProviderID(ctx, db.CheckUserExistsByProviderIDParams{
						ProviderID: sql.NullString{String: validUserInfo.ID, Valid: true},
						Provider:   constants.GoogleProvider,
					}).
					Return(false, errors.New("database connection failed"))

				return mockGoogleClient, mockGenerator, mockUserRepo, mockAuthSessionRepo, mockJwtToken, mockRedisClient
			},
			verify: func(t *testing.T, got *entities.HandleGoogleCallbackResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
				assert.Contains(t, gotErr.Error(), "database connection failed")
			},
		},
		{
			name: "Error_UserCreationFailed",
			input: &entities.HandleGoogleCallbackReq{
				Code:  validCode,
				State: validState,
			},
			setup: func() (*mockAuthInfras.MockGoogleClient, *mockUtils.MockGenerator, *repositories.MockUserRepository, *repositories.MockAuthSessionRepository, *mockUtils.MockJwtToken, *mockDatabase.MockRedisClient) {
				mockGoogleClient := new(mockAuthInfras.MockGoogleClient)
				mockGenerator := new(mockUtils.MockGenerator)
				mockUserRepo := new(repositories.MockUserRepository)
				mockAuthSessionRepo := new(repositories.MockAuthSessionRepository)
				mockJwtToken := new(mockUtils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockGoogleClient.EXPECT().
					ExchangeCodeForToken(ctx, validCode).
					Return(&oauth2.Token{AccessToken: "google_access_token"}, nil)

				redisKey := fmt.Sprintf("%s%s", constants.RedisPrefixOAuthState, validState)
				mockRedisClient.EXPECT().
					Get(ctx, redisKey).
					Return(constants.GoogleProvider, nil)

				mockGoogleClient.EXPECT().
					GetUserInfo(ctx, &oauth2.Token{AccessToken: "google_access_token"}).
					Return(validUserInfo, nil)

				mockUserRepo.EXPECT().
					CheckUserExistsByProviderID(ctx, db.CheckUserExistsByProviderIDParams{
						ProviderID: sql.NullString{String: validUserInfo.ID, Valid: true},
						Provider:   constants.GoogleProvider,
					}).
					Return(false, nil)

				userID := uuid.New()
				mockGenerator.EXPECT().
					GenerateUUID(ctx).
					Return(userID)

				mockUserRepo.EXPECT().
					CreateUserWithProvider(ctx, mock.MatchedBy(func(params *db.CreateUserWithProviderParams) bool {
						return params.ID == userID
					})).
					Return(errors.New("user creation failed"))

				return mockGoogleClient, mockGenerator, mockUserRepo, mockAuthSessionRepo, mockJwtToken, mockRedisClient
			},
			verify: func(t *testing.T, got *entities.HandleGoogleCallbackResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
				assert.Contains(t, gotErr.Error(), "user creation failed")
			},
		},
		{
			name: "Error_TokenCreationFailed",
			input: &entities.HandleGoogleCallbackReq{
				Code:  validCode,
				State: validState,
			},
			setup: func() (*mockAuthInfras.MockGoogleClient, *mockUtils.MockGenerator, *repositories.MockUserRepository, *repositories.MockAuthSessionRepository, *mockUtils.MockJwtToken, *mockDatabase.MockRedisClient) {
				mockGoogleClient := new(mockAuthInfras.MockGoogleClient)
				mockGenerator := new(mockUtils.MockGenerator)
				mockUserRepo := new(repositories.MockUserRepository)
				mockAuthSessionRepo := new(repositories.MockAuthSessionRepository)
				mockJwtToken := new(mockUtils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockGoogleClient.EXPECT().
					ExchangeCodeForToken(ctx, validCode).
					Return(&oauth2.Token{AccessToken: "google_access_token"}, nil)

				redisKey := fmt.Sprintf("%s%s", constants.RedisPrefixOAuthState, validState)
				mockRedisClient.EXPECT().
					Get(ctx, redisKey).
					Return(constants.GoogleProvider, nil)

				mockGoogleClient.EXPECT().
					GetUserInfo(ctx, &oauth2.Token{AccessToken: "google_access_token"}).
					Return(validUserInfo, nil)

				mockUserRepo.EXPECT().
					CheckUserExistsByProviderID(ctx, db.CheckUserExistsByProviderIDParams{
						ProviderID: sql.NullString{String: validUserInfo.ID, Valid: true},
						Provider:   constants.GoogleProvider,
					}).
					Return(true, nil)

				existingUser := db.Users{
					ID:         uuid.New(),
					Email:      validUserInfo.Email,
					FullName:   validUserInfo.Name,
					Provider:   constants.GoogleProvider,
					ProviderID: sql.NullString{String: validUserInfo.ID, Valid: true},
				}
				mockUserRepo.EXPECT().
					GetUserByProviderID(ctx, db.GetUserByProviderIDParams{
						ProviderID: sql.NullString{String: validUserInfo.ID, Valid: true},
						Provider:   constants.GoogleProvider,
					}).
					Return(existingUser, nil)

				mockUserRepo.EXPECT().
					UpdateUserLoginInfo(ctx, mock.MatchedBy(func(params db.UpdateUserLoginInfoParams) bool {
						return params.ID == existingUser.ID
					})).
					Return(nil)

				mockJwtToken.EXPECT().
					CreateToken(ctx, mock.MatchedBy(func(req *entities.TokenRequest) bool {
						return req.UserID == existingUser.ID.String()
					})).
					Return("", nil, errors.New("token creation failed"))

				return mockGoogleClient, mockGenerator, mockUserRepo, mockAuthSessionRepo, mockJwtToken, mockRedisClient
			},
			verify: func(t *testing.T, got *entities.HandleGoogleCallbackResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
				assert.Contains(t, gotErr.Error(), "token creation failed")
			},
		},
		{
			name: "Error_SessionCreationFailed",
			input: &entities.HandleGoogleCallbackReq{
				Code:  validCode,
				State: validState,
			},
			setup: func() (*mockAuthInfras.MockGoogleClient, *mockUtils.MockGenerator, *repositories.MockUserRepository, *repositories.MockAuthSessionRepository, *mockUtils.MockJwtToken, *mockDatabase.MockRedisClient) {
				mockGoogleClient := new(mockAuthInfras.MockGoogleClient)
				mockGenerator := new(mockUtils.MockGenerator)
				mockUserRepo := new(repositories.MockUserRepository)
				mockAuthSessionRepo := new(repositories.MockAuthSessionRepository)
				mockJwtToken := new(mockUtils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockGoogleClient.EXPECT().
					ExchangeCodeForToken(ctx, validCode).
					Return(&oauth2.Token{AccessToken: "google_access_token"}, nil)

				redisKey := fmt.Sprintf("%s%s", constants.RedisPrefixOAuthState, validState)
				mockRedisClient.EXPECT().
					Get(ctx, redisKey).
					Return(constants.GoogleProvider, nil)

				mockGoogleClient.EXPECT().
					GetUserInfo(ctx, &oauth2.Token{AccessToken: "google_access_token"}).
					Return(validUserInfo, nil)

				mockUserRepo.EXPECT().
					CheckUserExistsByProviderID(ctx, db.CheckUserExistsByProviderIDParams{
						ProviderID: sql.NullString{String: validUserInfo.ID, Valid: true},
						Provider:   constants.GoogleProvider,
					}).
					Return(true, nil)

				existingUser := db.Users{
					ID:         uuid.New(),
					Email:      validUserInfo.Email,
					FullName:   validUserInfo.Name,
					Provider:   constants.GoogleProvider,
					ProviderID: sql.NullString{String: validUserInfo.ID, Valid: true},
				}
				mockUserRepo.EXPECT().
					GetUserByProviderID(ctx, db.GetUserByProviderIDParams{
						ProviderID: sql.NullString{String: validUserInfo.ID, Valid: true},
						Provider:   constants.GoogleProvider,
					}).
					Return(existingUser, nil)

				mockUserRepo.EXPECT().
					UpdateUserLoginInfo(ctx, mock.MatchedBy(func(params db.UpdateUserLoginInfoParams) bool {
						return params.ID == existingUser.ID
					})).
					Return(nil)

				mockJwtToken.EXPECT().
					CreateToken(ctx, mock.MatchedBy(func(req *entities.TokenRequest) bool {
						return req.UserID == existingUser.ID.String()
					})).
					Return("access_token", &utilsPkg.SignInTokenPayload{ID: uuid.New(), ExpiredAt: time.Now().Add(time.Hour)}, nil).
					Times(2)

				mockJwtToken.EXPECT().
					HashTokenSHA256(ctx, "access_token").
					Return("hashed_refresh_token")

				mockAuthSessionRepo.EXPECT().
					CreateAuthSession(ctx, mock.MatchedBy(func(params *db.CreateAuthSessionParams) bool {
						return params.UserID == existingUser.ID
					})).
					Return(errors.New("session creation failed"))

				return mockGoogleClient, mockGenerator, mockUserRepo, mockAuthSessionRepo, mockJwtToken, mockRedisClient
			},
			verify: func(t *testing.T, got *entities.HandleGoogleCallbackResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
				assert.Contains(t, gotErr.Error(), "session creation failed")
			},
		},
		{
			name: "Error_CreatedUserRetrievalFailed",
			input: &entities.HandleGoogleCallbackReq{
				Code:  validCode,
				State: validState,
			},
			setup: func() (*mockAuthInfras.MockGoogleClient, *mockUtils.MockGenerator, *repositories.MockUserRepository, *repositories.MockAuthSessionRepository, *mockUtils.MockJwtToken, *mockDatabase.MockRedisClient) {
				mockGoogleClient := new(mockAuthInfras.MockGoogleClient)
				mockGenerator := new(mockUtils.MockGenerator)
				mockUserRepo := new(repositories.MockUserRepository)
				mockAuthSessionRepo := new(repositories.MockAuthSessionRepository)
				mockJwtToken := new(mockUtils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockGoogleClient.EXPECT().
					ExchangeCodeForToken(ctx, validCode).
					Return(&oauth2.Token{AccessToken: "google_access_token"}, nil)

				redisKey := fmt.Sprintf("%s%s", constants.RedisPrefixOAuthState, validState)
				mockRedisClient.EXPECT().
					Get(ctx, redisKey).
					Return(constants.GoogleProvider, nil)

				mockGoogleClient.EXPECT().
					GetUserInfo(ctx, &oauth2.Token{AccessToken: "google_access_token"}).
					Return(validUserInfo, nil)

				mockUserRepo.EXPECT().
					CheckUserExistsByProviderID(ctx, db.CheckUserExistsByProviderIDParams{
						ProviderID: sql.NullString{String: validUserInfo.ID, Valid: true},
						Provider:   constants.GoogleProvider,
					}).
					Return(false, nil)

				userID := uuid.New()
				mockGenerator.EXPECT().
					GenerateUUID(ctx).
					Return(userID)

				mockUserRepo.EXPECT().
					CreateUserWithProvider(ctx, mock.MatchedBy(func(params *db.CreateUserWithProviderParams) bool {
						return params.ID == userID
					})).
					Return(nil)

				mockUserRepo.EXPECT().
					GetUserByProviderID(ctx, db.GetUserByProviderIDParams{
						ProviderID: sql.NullString{String: validUserInfo.ID, Valid: true},
						Provider:   constants.GoogleProvider,
					}).
					Return(db.Users{}, errors.New("failed to retrieve created user"))

				return mockGoogleClient, mockGenerator, mockUserRepo, mockAuthSessionRepo, mockJwtToken, mockRedisClient
			},
			verify: func(t *testing.T, got *entities.HandleGoogleCallbackResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
				assert.Contains(t, gotErr.Error(), "failed to retrieve created user")
			},
		},
		{
			name: "Error_ExistingUserRetrievalFailed",
			input: &entities.HandleGoogleCallbackReq{
				Code:  validCode,
				State: validState,
			},
			setup: func() (*mockAuthInfras.MockGoogleClient, *mockUtils.MockGenerator, *repositories.MockUserRepository, *repositories.MockAuthSessionRepository, *mockUtils.MockJwtToken, *mockDatabase.MockRedisClient) {
				mockGoogleClient := new(mockAuthInfras.MockGoogleClient)
				mockGenerator := new(mockUtils.MockGenerator)
				mockUserRepo := new(repositories.MockUserRepository)
				mockAuthSessionRepo := new(repositories.MockAuthSessionRepository)
				mockJwtToken := new(mockUtils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockGoogleClient.EXPECT().
					ExchangeCodeForToken(ctx, validCode).
					Return(&oauth2.Token{AccessToken: "google_access_token"}, nil)

				redisKey := fmt.Sprintf("%s%s", constants.RedisPrefixOAuthState, validState)
				mockRedisClient.EXPECT().
					Get(ctx, redisKey).
					Return(constants.GoogleProvider, nil)

				mockGoogleClient.EXPECT().
					GetUserInfo(ctx, &oauth2.Token{AccessToken: "google_access_token"}).
					Return(validUserInfo, nil)

				mockUserRepo.EXPECT().
					CheckUserExistsByProviderID(ctx, db.CheckUserExistsByProviderIDParams{
						ProviderID: sql.NullString{String: validUserInfo.ID, Valid: true},
						Provider:   constants.GoogleProvider,
					}).
					Return(true, nil)

				mockUserRepo.EXPECT().
					GetUserByProviderID(ctx, db.GetUserByProviderIDParams{
						ProviderID: sql.NullString{String: validUserInfo.ID, Valid: true},
						Provider:   constants.GoogleProvider,
					}).
					Return(db.Users{}, errors.New("failed to fetch existing user"))

				return mockGoogleClient, mockGenerator, mockUserRepo, mockAuthSessionRepo, mockJwtToken, mockRedisClient
			},
			verify: func(t *testing.T, got *entities.HandleGoogleCallbackResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
				assert.Contains(t, gotErr.Error(), "failed to fetch existing user")
			},
		},
		{
			name: "Error WithExistingUserLoginInfoUpdateFailed",
			input: &entities.HandleGoogleCallbackReq{
				Code:  validCode,
				State: validState,
			},
			setup: func() (*mockAuthInfras.MockGoogleClient, *mockUtils.MockGenerator, *repositories.MockUserRepository, *repositories.MockAuthSessionRepository, *mockUtils.MockJwtToken, *mockDatabase.MockRedisClient) {
				mockGoogleClient := new(mockAuthInfras.MockGoogleClient)
				mockGenerator := new(mockUtils.MockGenerator)
				mockUserRepo := new(repositories.MockUserRepository)
				mockAuthSessionRepo := new(repositories.MockAuthSessionRepository)
				mockJwtToken := new(mockUtils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockGoogleClient.EXPECT().
					ExchangeCodeForToken(ctx, validCode).
					Return(&oauth2.Token{AccessToken: "google_access_token"}, nil)

				redisKey := fmt.Sprintf("%s%s", constants.RedisPrefixOAuthState, validState)
				mockRedisClient.EXPECT().
					Get(ctx, redisKey).
					Return(constants.GoogleProvider, nil)

				mockGoogleClient.EXPECT().
					GetUserInfo(ctx, &oauth2.Token{AccessToken: "google_access_token"}).
					Return(validUserInfo, nil)

				mockUserRepo.EXPECT().
					CheckUserExistsByProviderID(ctx, db.CheckUserExistsByProviderIDParams{
						ProviderID: sql.NullString{String: validUserInfo.ID, Valid: true},
						Provider:   constants.GoogleProvider,
					}).
					Return(true, nil)

				existingUser := db.Users{
					ID:         uuid.New(),
					Email:      validUserInfo.Email,
					FullName:   validUserInfo.Name,
					Provider:   constants.GoogleProvider,
					ProviderID: sql.NullString{String: validUserInfo.ID, Valid: true},
				}
				mockUserRepo.EXPECT().
					GetUserByProviderID(ctx, db.GetUserByProviderIDParams{
						ProviderID: sql.NullString{String: validUserInfo.ID, Valid: true},
						Provider:   constants.GoogleProvider,
					}).
					Return(existingUser, nil)

				mockUserRepo.EXPECT().
					UpdateUserLoginInfo(ctx, mock.MatchedBy(func(params db.UpdateUserLoginInfoParams) bool {
						return params.ID == existingUser.ID
					})).
					Return(errors.New("failed to update login info"))

				return mockGoogleClient, mockGenerator, mockUserRepo, mockAuthSessionRepo, mockJwtToken, mockRedisClient
			},
			verify: func(t *testing.T, got *entities.HandleGoogleCallbackResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
				assert.Contains(t, gotErr.Error(), "failed to update login info")
			},
		},
		{
			name: "Error_RefreshTokenCreationFailed",
			input: &entities.HandleGoogleCallbackReq{
				Code:  validCode,
				State: validState,
			},
			setup: func() (*mockAuthInfras.MockGoogleClient, *mockUtils.MockGenerator, *repositories.MockUserRepository, *repositories.MockAuthSessionRepository, *mockUtils.MockJwtToken, *mockDatabase.MockRedisClient) {
				mockGoogleClient := new(mockAuthInfras.MockGoogleClient)
				mockGenerator := new(mockUtils.MockGenerator)
				mockUserRepo := new(repositories.MockUserRepository)
				mockAuthSessionRepo := new(repositories.MockAuthSessionRepository)
				mockJwtToken := new(mockUtils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockGoogleClient.EXPECT().
					ExchangeCodeForToken(ctx, validCode).
					Return(&oauth2.Token{AccessToken: "google_access_token"}, nil)

				redisKey := fmt.Sprintf("%s%s", constants.RedisPrefixOAuthState, validState)
				mockRedisClient.EXPECT().
					Get(ctx, redisKey).
					Return(constants.GoogleProvider, nil)

				mockGoogleClient.EXPECT().
					GetUserInfo(ctx, &oauth2.Token{AccessToken: "google_access_token"}).
					Return(validUserInfo, nil)

				mockUserRepo.EXPECT().
					CheckUserExistsByProviderID(ctx, db.CheckUserExistsByProviderIDParams{
						ProviderID: sql.NullString{String: validUserInfo.ID, Valid: true},
						Provider:   constants.GoogleProvider,
					}).
					Return(true, nil)

				existingUser := db.Users{
					ID:         uuid.New(),
					Email:      validUserInfo.Email,
					FullName:   validUserInfo.Name,
					Provider:   constants.GoogleProvider,
					ProviderID: sql.NullString{String: validUserInfo.ID, Valid: true},
				}
				mockUserRepo.EXPECT().
					GetUserByProviderID(ctx, db.GetUserByProviderIDParams{
						ProviderID: sql.NullString{String: validUserInfo.ID, Valid: true},
						Provider:   constants.GoogleProvider,
					}).
					Return(existingUser, nil)

				mockUserRepo.EXPECT().
					UpdateUserLoginInfo(ctx, mock.MatchedBy(func(params db.UpdateUserLoginInfoParams) bool {
						return params.ID == existingUser.ID
					})).
					Return(nil)

				mockJwtToken.EXPECT().
					CreateToken(ctx, mock.MatchedBy(func(req *entities.TokenRequest) bool {
						return req.UserID == existingUser.ID.String() && req.Role == constants.UserRoleUser
					})).
					Return("access_token", &utilsPkg.SignInTokenPayload{ID: uuid.New(), ExpiredAt: time.Now().Add(time.Hour)}, nil).
					Times(1)

				mockJwtToken.EXPECT().
					CreateToken(ctx, mock.MatchedBy(func(req *entities.TokenRequest) bool {
						return req.UserID == existingUser.ID.String() && req.Role == constants.UserRoleUser
					})).
					Return("", nil, errors.New("refresh token creation failed"))

				return mockGoogleClient, mockGenerator, mockUserRepo, mockAuthSessionRepo, mockJwtToken, mockRedisClient
			},
			verify: func(t *testing.T, got *entities.HandleGoogleCallbackResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
				assert.Contains(t, gotErr.Error(), "refresh token creation failed")
			},
		},
		{
			name: "Error_RedisStateError",
			input: &entities.HandleGoogleCallbackReq{
				Code:  validCode,
				State: validState,
			},
			setup: func() (*mockAuthInfras.MockGoogleClient, *mockUtils.MockGenerator, *repositories.MockUserRepository, *repositories.MockAuthSessionRepository, *mockUtils.MockJwtToken, *mockDatabase.MockRedisClient) {
				mockGoogleClient := new(mockAuthInfras.MockGoogleClient)
				mockGenerator := new(mockUtils.MockGenerator)
				mockUserRepo := new(repositories.MockUserRepository)
				mockAuthSessionRepo := new(repositories.MockAuthSessionRepository)
				mockJwtToken := new(mockUtils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockGoogleClient.EXPECT().
					ExchangeCodeForToken(ctx, validCode).
					Return(&oauth2.Token{AccessToken: "google_access_token"}, nil)

				redisKey := fmt.Sprintf("%s%s", constants.RedisPrefixOAuthState, validState)
				mockRedisClient.EXPECT().
					Get(ctx, redisKey).
					Return("", errors.New("redis connection failed"))

				return mockGoogleClient, mockGenerator, mockUserRepo, mockAuthSessionRepo, mockJwtToken, mockRedisClient
			},
			verify: func(t *testing.T, got *entities.HandleGoogleCallbackResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
				assert.Contains(t, gotErr.Error(), "redis connection failed")
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			mockGoogleClient, mockGenerator, mockUserRepo, mockAuthSessionRepo, mockJwtToken, mockRedisClient := tC.setup()
			defer mockGoogleClient.AssertExpectations(t)
			defer mockGenerator.AssertExpectations(t)
			defer mockUserRepo.AssertExpectations(t)
			defer mockAuthSessionRepo.AssertExpectations(t)
			defer mockJwtToken.AssertExpectations(t)
			defer mockRedisClient.AssertExpectations(t)

			mockConfig := &config.Config{
				AuthConfig: config.AuthConfig{
					AccessTokenDuration:  time.Hour,
					RefreshTokenDuration: time.Hour * 24,
				},
			}

			svc := NewUserService(
				lgr,
				mockUserRepo,
				mockAuthSessionRepo,
				mockJwtToken,
				nil,
				mockConfig,
				nil,
				mockRedisClient,
				nil,
				nil,
				mockGenerator,
				mockGoogleClient,
				nil,
			)

			got, gotErr := svc.HandleGoogleCallback(ctx, tC.input)
			tC.verify(t, got, gotErr)
		})
	}
}

func TestUserService_GetGoogleAuthURL(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()
	mockErr := errors.New("error")

	okResponse := &entities.GoogleAuthURLResponse{
		AuthURL: "https://www.google.com/auth/url",
	}

	testCases := []struct {
		name   string
		setup  func() (*mockDatabase.MockRedisClient, *mockAuthInfras.MockGoogleClient, *mockUtils.MockGenerator)
		verify func(t *testing.T, got *entities.GoogleAuthURLResponse, gotErr error)
	}{
		{
			name: "Success",
			setup: func() (*mockDatabase.MockRedisClient, *mockAuthInfras.MockGoogleClient, *mockUtils.MockGenerator) {
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockGoogleClient := new(mockAuthInfras.MockGoogleClient)
				mockGenerator := new(mockUtils.MockGenerator)

				redisKey := fmt.Sprintf("%s%s", constants.RedisPrefixOAuthState, "oauth_state")
				redisPayload := database.RedisPayload{
					Key:   redisKey,
					Value: constants.GoogleProvider,
					TTL:   constants.RedisTTLOAuthState,
				}

				mockGenerator.EXPECT().
					GenerateCryptographicallySecureString(ctx, 32).
					Return("oauth_state")

				mockRedisClient.EXPECT().
					Set(ctx, redisPayload).
					Return(nil)

				mockGoogleClient.EXPECT().
					GetAuthURL("oauth_state").
					Return("https://www.google.com/auth/url")

				return mockRedisClient, mockGoogleClient, mockGenerator
			},
			verify: func(t *testing.T, got *entities.GoogleAuthURLResponse, gotErr error) {
				assert.Equal(t, okResponse, got)
				assert.NoError(t, gotErr)
			},
		},
		{
			name: "Error WithRedisSetFailed",
			setup: func() (*mockDatabase.MockRedisClient, *mockAuthInfras.MockGoogleClient, *mockUtils.MockGenerator) {
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockGoogleClient := new(mockAuthInfras.MockGoogleClient)
				mockGenerator := new(mockUtils.MockGenerator)

				redisKey := fmt.Sprintf("%s%s", constants.RedisPrefixOAuthState, "oauth_state")
				redisPayload := database.RedisPayload{
					Key:   redisKey,
					Value: constants.GoogleProvider,
					TTL:   constants.RedisTTLOAuthState,
				}

				mockGenerator.EXPECT().
					GenerateCryptographicallySecureString(ctx, 32).
					Return("oauth_state")

				mockRedisClient.EXPECT().
					Set(ctx, redisPayload).
					Return(mockErr)

				return mockRedisClient, mockGoogleClient, mockGenerator
			},
			verify: func(t *testing.T, got *entities.GoogleAuthURLResponse, gotErr error) {
				assert.Error(t, gotErr)
				assert.ErrorIs(t, gotErr, mockErr)
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			mockRedisClient, mockGoogleClient, mockGenerator := tC.setup()
			defer mockRedisClient.AssertExpectations(t)
			defer mockGoogleClient.AssertExpectations(t)
			defer mockGenerator.AssertExpectations(t)

			svc := NewUserService(lgr, nil, nil, nil, nil, nil, nil, mockRedisClient, nil, nil, mockGenerator, mockGoogleClient, nil)
			got, gotErr := svc.GetGoogleAuthURL(ctx)

			tC.verify(t, got, gotErr)
		})
	}
}

func TestUserService_SignOut(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()
	mockErr := errors.New("error")

	testCases := []struct {
		name   string
		setup  func() (*repositories.MockAuthSessionRepository, *middlewareMocks.MockAuthContext)
		verify func(t *testing.T, gotErr error)
	}{
		{
			name: "Success",
			setup: func() (*repositories.MockAuthSessionRepository, *middlewareMocks.MockAuthContext) {
				mockAuthSessionRepo := new(repositories.MockAuthSessionRepository)
				mockAuthContext := new(middlewareMocks.MockAuthContext)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
						},
					}, nil)

				// Mock session revocation
				mockAuthSessionRepo.EXPECT().
					RevokeAuthSessionByID(ctx, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")).
					Return(nil)

				return mockAuthSessionRepo, mockAuthContext
			},
			verify: func(t *testing.T, gotErr error) {
				assert.NoError(t, gotErr)
			},
		},
		{
			name: "Error_AuthContextFailed",
			setup: func() (*repositories.MockAuthSessionRepository, *middlewareMocks.MockAuthContext) {
				mockAuthContext := new(middlewareMocks.MockAuthContext)

				// Mock auth context retrieval fails
				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(nil, mockErr)

				return nil, mockAuthContext
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
			},
		},
		{
			name: "Error_SessionRevocationFailed",
			setup: func() (*repositories.MockAuthSessionRepository, *middlewareMocks.MockAuthContext) {
				mockAuthSessionRepo := new(repositories.MockAuthSessionRepository)
				mockAuthContext := new(middlewareMocks.MockAuthContext)

				// Mock auth context retrieval
				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
						},
					}, nil)

				// Mock session revocation fails
				mockAuthSessionRepo.EXPECT().
					RevokeAuthSessionByID(ctx, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")).
					Return(mockErr)

				return mockAuthSessionRepo, mockAuthContext
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.ErrorIs(t, gotErr, mockErr)
			},
		},
		{
			name: "Error_InvalidAuthPayload",
			setup: func() (*repositories.MockAuthSessionRepository, *middlewareMocks.MockAuthContext) {
				mockAuthContext := new(middlewareMocks.MockAuthContext)

				// Mock auth context retrieval returns invalid payload
				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: nil,
					}, nil)

				return nil, mockAuthContext
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
			},
		},
		{
			name: "Error_NilAuthContext",
			setup: func() (*repositories.MockAuthSessionRepository, *middlewareMocks.MockAuthContext) {
				mockAuthContext := new(middlewareMocks.MockAuthContext)

				// Mock auth context retrieval returns nil
				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(nil, nil)

				return nil, mockAuthContext
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
			},
		},
		{
			name: "Error_InvalidSessionID",
			setup: func() (*repositories.MockAuthSessionRepository, *middlewareMocks.MockAuthContext) {
				mockAuthSessionRepo := new(repositories.MockAuthSessionRepository)
				mockAuthContext := new(middlewareMocks.MockAuthContext)

				// Mock auth context retrieval returns invalid session ID
				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							ID: uuid.Nil,
						},
					}, nil)

				return mockAuthSessionRepo, mockAuthContext
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			mockAuthSessionRepo, mockAuthContext := tC.setup()
			defer func() {
				if mockAuthSessionRepo != nil {
					mockAuthSessionRepo.AssertExpectations(t)
				}
				if mockAuthContext != nil {
					mockAuthContext.AssertExpectations(t)
				}
			}()

			svc := NewUserService(lgr, nil, mockAuthSessionRepo, nil, nil, nil, mockAuthContext, nil, nil, nil, nil, nil, nil)
			gotErr := svc.SignOut(ctx)

			tC.verify(t, gotErr)
		})
	}
}

func TestUserService_HandleFacebookCallback(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()

	validCode := "valid_auth_code"
	validState := "valid_oauth_state"

	validUserInfo := &auth.FacebookUserInfo{
		Email:  "test@example.com",
		Name:   "Test User",
		ID:     "facebook_user_123",
		Locale: "en-US",
		Picture: struct {
			Data struct {
				URL string `json:"url"`
			} `json:"data"`
		}{
			Data: struct {
				URL string `json:"url"`
			}{
				URL: "https://example.com/picture.jpg",
			},
		},
	}

	testCases := []struct {
		name   string
		input  *entities.HandleFacebookCallbackReq
		setup  func() (*mockAuthInfras.MockFacebookClient, *mockUtils.MockGenerator, *repositories.MockUserRepository, *repositories.MockAuthSessionRepository, *mockUtils.MockJwtToken, *mockDatabase.MockRedisClient)
		verify func(t *testing.T, got *entities.HandleFacebookCallbackResp, gotErr error)
	}{
		{
			name: "Success_NewUser",
			input: &entities.HandleFacebookCallbackReq{
				Code:  validCode,
				State: validState,
			},
			setup: func() (*mockAuthInfras.MockFacebookClient, *mockUtils.MockGenerator, *repositories.MockUserRepository, *repositories.MockAuthSessionRepository, *mockUtils.MockJwtToken, *mockDatabase.MockRedisClient) {
				mockFacebookClient := new(mockAuthInfras.MockFacebookClient)
				mockGenerator := new(mockUtils.MockGenerator)
				mockUserRepo := new(repositories.MockUserRepository)
				mockAuthSessionRepo := new(repositories.MockAuthSessionRepository)
				mockJwtToken := new(mockUtils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockFacebookClient.EXPECT().
					ExchangeCodeForToken(ctx, validCode).
					Return(&oauth2.Token{AccessToken: "facebook_access_token"}, nil)

				redisKey := fmt.Sprintf("%s%s", constants.RedisPrefixOAuthState, validState)
				mockRedisClient.EXPECT().
					Get(ctx, redisKey).
					Return(constants.FacebookProvider, nil)

				mockFacebookClient.EXPECT().
					GetUserInfo(ctx, &oauth2.Token{AccessToken: "facebook_access_token"}).
					Return(validUserInfo, nil)

				mockUserRepo.EXPECT().
					CheckUserExistsByProviderID(ctx, db.CheckUserExistsByProviderIDParams{
						ProviderID: sql.NullString{String: validUserInfo.ID, Valid: true},
						Provider:   constants.FacebookProvider,
					}).
					Return(false, nil)

				userID := uuid.New()
				mockGenerator.EXPECT().
					GenerateUUID(ctx).
					Return(userID)

				mockUserRepo.EXPECT().
					CreateUserWithProvider(ctx, mock.MatchedBy(func(params *db.CreateUserWithProviderParams) bool {
						return params.ID == userID &&
							params.Email == validUserInfo.Email &&
							params.FullName == validUserInfo.Name &&
							params.Provider == constants.FacebookProvider &&
							params.ProviderID.String == validUserInfo.ID
					})).
					Return(nil)

				createdUser := db.Users{
					ID:         userID,
					Email:      validUserInfo.Email,
					FullName:   validUserInfo.Name,
					Provider:   constants.FacebookProvider,
					ProviderID: sql.NullString{String: validUserInfo.ID, Valid: true},
				}
				mockUserRepo.EXPECT().
					GetUserByProviderID(ctx, db.GetUserByProviderIDParams{
						ProviderID: sql.NullString{String: validUserInfo.ID, Valid: true},
						Provider:   constants.FacebookProvider,
					}).
					Return(createdUser, nil)

				mockJwtToken.EXPECT().
					CreateToken(ctx, mock.MatchedBy(func(req *entities.TokenRequest) bool {
						return req.UserID == userID.String() && req.Role == constants.UserRoleUser
					})).
					Return("access_token", &utilsPkg.SignInTokenPayload{ID: uuid.New(), ExpiredAt: time.Now().Add(time.Hour)}, nil).
					Times(1)

				mockJwtToken.EXPECT().
					CreateToken(ctx, mock.MatchedBy(func(req *entities.TokenRequest) bool {
						return req.UserID == userID.String() && req.Role == constants.UserRoleUser
					})).
					Return("refresh_token", &utilsPkg.SignInTokenPayload{ID: uuid.New(), ExpiredAt: time.Now().Add(time.Hour)}, nil).
					Times(1)

				mockJwtToken.EXPECT().
					HashTokenSHA256(ctx, "refresh_token").
					Return("hashed_refresh_token")

				mockAuthSessionRepo.EXPECT().
					CreateAuthSession(ctx, mock.MatchedBy(func(params *db.CreateAuthSessionParams) bool {
						return params.UserID == userID && params.RefreshTokenHash == "hashed_refresh_token"
					})).
					Return(nil)

				return mockFacebookClient, mockGenerator, mockUserRepo, mockAuthSessionRepo, mockJwtToken, mockRedisClient
			},
			verify: func(t *testing.T, got *entities.HandleFacebookCallbackResp, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, got)
				assert.Equal(t, "access_token", got.AccessToken)
				assert.Equal(t, "refresh_token", got.RefreshToken)
			},
		},
		{
			name: "Success_ExistingUser",
			input: &entities.HandleFacebookCallbackReq{
				Code:  validCode,
				State: validState,
			},
			setup: func() (*mockAuthInfras.MockFacebookClient, *mockUtils.MockGenerator, *repositories.MockUserRepository, *repositories.MockAuthSessionRepository, *mockUtils.MockJwtToken, *mockDatabase.MockRedisClient) {
				mockFacebookClient := new(mockAuthInfras.MockFacebookClient)
				mockGenerator := new(mockUtils.MockGenerator)
				mockUserRepo := new(repositories.MockUserRepository)
				mockAuthSessionRepo := new(repositories.MockAuthSessionRepository)
				mockJwtToken := new(mockUtils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockFacebookClient.EXPECT().
					ExchangeCodeForToken(ctx, validCode).
					Return(&oauth2.Token{AccessToken: "facebook_access_token"}, nil)

				redisKey := fmt.Sprintf("%s%s", constants.RedisPrefixOAuthState, validState)
				mockRedisClient.EXPECT().
					Get(ctx, redisKey).
					Return(constants.FacebookProvider, nil)

				mockFacebookClient.EXPECT().
					GetUserInfo(ctx, &oauth2.Token{AccessToken: "facebook_access_token"}).
					Return(validUserInfo, nil)

				mockUserRepo.EXPECT().
					CheckUserExistsByProviderID(ctx, db.CheckUserExistsByProviderIDParams{
						ProviderID: sql.NullString{String: validUserInfo.ID, Valid: true},
						Provider:   constants.FacebookProvider,
					}).
					Return(true, nil)

				existingUser := db.Users{
					ID:         uuid.New(),
					Email:      validUserInfo.Email,
					FullName:   validUserInfo.Name,
					Provider:   constants.FacebookProvider,
					ProviderID: sql.NullString{String: validUserInfo.ID, Valid: true},
				}
				mockUserRepo.EXPECT().
					GetUserByProviderID(ctx, db.GetUserByProviderIDParams{
						ProviderID: sql.NullString{String: validUserInfo.ID, Valid: true},
						Provider:   constants.FacebookProvider,
					}).
					Return(existingUser, nil)

				mockUserRepo.EXPECT().
					UpdateUserLoginInfo(ctx, mock.MatchedBy(func(params db.UpdateUserLoginInfoParams) bool {
						return params.ID == existingUser.ID
					})).
					Return(nil)

				mockJwtToken.EXPECT().
					CreateToken(ctx, mock.MatchedBy(func(req *entities.TokenRequest) bool {
						return req.UserID == existingUser.ID.String() && req.Role == constants.UserRoleUser
					})).
					Return("access_token", &utilsPkg.SignInTokenPayload{ID: uuid.New(), ExpiredAt: time.Now().Add(time.Hour)}, nil).
					Times(1)

				mockJwtToken.EXPECT().
					CreateToken(ctx, mock.MatchedBy(func(req *entities.TokenRequest) bool {
						return req.UserID == existingUser.ID.String() && req.Role == constants.UserRoleUser
					})).
					Return("refresh_token", &utilsPkg.SignInTokenPayload{ID: uuid.New(), ExpiredAt: time.Now().Add(time.Hour)}, nil).
					Times(1)

				mockJwtToken.EXPECT().
					HashTokenSHA256(ctx, "refresh_token").
					Return("hashed_refresh_token")

				mockAuthSessionRepo.EXPECT().
					CreateAuthSession(ctx, mock.MatchedBy(func(params *db.CreateAuthSessionParams) bool {
						return params.UserID == existingUser.ID && params.RefreshTokenHash == "hashed_refresh_token"
					})).
					Return(nil)

				return mockFacebookClient, mockGenerator, mockUserRepo, mockAuthSessionRepo, mockJwtToken, mockRedisClient
			},
			verify: func(t *testing.T, got *entities.HandleFacebookCallbackResp, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, got)
				assert.Equal(t, "access_token", got.AccessToken)
				assert.Equal(t, "refresh_token", got.RefreshToken)
			},
		},
		{
			name: "Error_InvalidCode",
			input: &entities.HandleFacebookCallbackReq{
				Code:  "",
				State: validState,
			},
			setup: func() (*mockAuthInfras.MockFacebookClient, *mockUtils.MockGenerator, *repositories.MockUserRepository, *repositories.MockAuthSessionRepository, *mockUtils.MockJwtToken, *mockDatabase.MockRedisClient) {
				mockFacebookClient := new(mockAuthInfras.MockFacebookClient)
				mockGenerator := new(mockUtils.MockGenerator)
				mockUserRepo := new(repositories.MockUserRepository)
				mockAuthSessionRepo := new(repositories.MockAuthSessionRepository)
				mockJwtToken := new(mockUtils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockFacebookClient.EXPECT().
					ExchangeCodeForToken(ctx, "").
					Return(nil, errors.New("invalid code"))

				return mockFacebookClient, mockGenerator, mockUserRepo, mockAuthSessionRepo, mockJwtToken, mockRedisClient
			},
			verify: func(t *testing.T, got *entities.HandleFacebookCallbackResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
			},
		},
		{
			name: "Error_InvalidState",
			input: &entities.HandleFacebookCallbackReq{
				Code:  validCode,
				State: "",
			},
			setup: func() (*mockAuthInfras.MockFacebookClient, *mockUtils.MockGenerator, *repositories.MockUserRepository, *repositories.MockAuthSessionRepository, *mockUtils.MockJwtToken, *mockDatabase.MockRedisClient) {
				mockFacebookClient := new(mockAuthInfras.MockFacebookClient)
				mockGenerator := new(mockUtils.MockGenerator)
				mockUserRepo := new(repositories.MockUserRepository)
				mockAuthSessionRepo := new(repositories.MockAuthSessionRepository)
				mockJwtToken := new(mockUtils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockFacebookClient.EXPECT().
					ExchangeCodeForToken(ctx, validCode).
					Return(&oauth2.Token{AccessToken: "facebook_access_token"}, nil)

				redisKey := fmt.Sprintf("%s%s", constants.RedisPrefixOAuthState, "")
				mockRedisClient.EXPECT().
					Get(ctx, redisKey).
					Return("", errors.New("invalid state"))

				return mockFacebookClient, mockGenerator, mockUserRepo, mockAuthSessionRepo, mockJwtToken, mockRedisClient
			},
			verify: func(t *testing.T, got *entities.HandleFacebookCallbackResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
			},
		},
		{
			name: "Error_StateNotFound",
			input: &entities.HandleFacebookCallbackReq{
				Code:  validCode,
				State: "invalid_state",
			},
			setup: func() (*mockAuthInfras.MockFacebookClient, *mockUtils.MockGenerator, *repositories.MockUserRepository, *repositories.MockAuthSessionRepository, *mockUtils.MockJwtToken, *mockDatabase.MockRedisClient) {
				mockFacebookClient := new(mockAuthInfras.MockFacebookClient)
				mockGenerator := new(mockUtils.MockGenerator)
				mockUserRepo := new(repositories.MockUserRepository)
				mockAuthSessionRepo := new(repositories.MockAuthSessionRepository)
				mockJwtToken := new(mockUtils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockFacebookClient.EXPECT().
					ExchangeCodeForToken(ctx, validCode).
					Return(&oauth2.Token{AccessToken: "facebook_access_token"}, nil)

				redisKey := fmt.Sprintf("%s%s", constants.RedisPrefixOAuthState, "invalid_state")
				mockRedisClient.EXPECT().
					Get(ctx, redisKey).
					Return("", redis.Nil)

				return mockFacebookClient, mockGenerator, mockUserRepo, mockAuthSessionRepo, mockJwtToken, mockRedisClient
			},
			verify: func(t *testing.T, got *entities.HandleFacebookCallbackResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
			},
		},
		{
			name: "Error_InvalidUserInfo",
			input: &entities.HandleFacebookCallbackReq{
				Code:  validCode,
				State: validState,
			},
			setup: func() (*mockAuthInfras.MockFacebookClient, *mockUtils.MockGenerator, *repositories.MockUserRepository, *repositories.MockAuthSessionRepository, *mockUtils.MockJwtToken, *mockDatabase.MockRedisClient) {
				mockFacebookClient := new(mockAuthInfras.MockFacebookClient)
				mockGenerator := new(mockUtils.MockGenerator)
				mockUserRepo := new(repositories.MockUserRepository)
				mockAuthSessionRepo := new(repositories.MockAuthSessionRepository)
				mockJwtToken := new(mockUtils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockFacebookClient.EXPECT().
					ExchangeCodeForToken(ctx, validCode).
					Return(&oauth2.Token{AccessToken: "facebook_access_token"}, nil)

				redisKey := fmt.Sprintf("%s%s", constants.RedisPrefixOAuthState, validState)
				mockRedisClient.EXPECT().
					Get(ctx, redisKey).
					Return(constants.FacebookProvider, nil)

				invalidUserInfo := &auth.FacebookUserInfo{
					Email: "",
					Name:  "",
					ID:    "",
				}
				mockFacebookClient.EXPECT().
					GetUserInfo(ctx, &oauth2.Token{AccessToken: "facebook_access_token"}).
					Return(invalidUserInfo, nil)

				return mockFacebookClient, mockGenerator, mockUserRepo, mockAuthSessionRepo, mockJwtToken, mockRedisClient
			},
			verify: func(t *testing.T, got *entities.HandleFacebookCallbackResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
			},
		},
		{
			name: "Error_UserInfoRetrievalFailed",
			input: &entities.HandleFacebookCallbackReq{
				Code:  validCode,
				State: validState,
			},
			setup: func() (*mockAuthInfras.MockFacebookClient, *mockUtils.MockGenerator, *repositories.MockUserRepository, *repositories.MockAuthSessionRepository, *mockUtils.MockJwtToken, *mockDatabase.MockRedisClient) {
				mockFacebookClient := new(mockAuthInfras.MockFacebookClient)
				mockGenerator := new(mockUtils.MockGenerator)
				mockUserRepo := new(repositories.MockUserRepository)
				mockAuthSessionRepo := new(repositories.MockAuthSessionRepository)
				mockJwtToken := new(mockUtils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockFacebookClient.EXPECT().
					ExchangeCodeForToken(ctx, validCode).
					Return(&oauth2.Token{AccessToken: "facebook_access_token"}, nil)

				redisKey := fmt.Sprintf("%s%s", constants.RedisPrefixOAuthState, validState)
				mockRedisClient.EXPECT().
					Get(ctx, redisKey).
					Return(constants.FacebookProvider, nil)

				mockFacebookClient.EXPECT().
					GetUserInfo(ctx, &oauth2.Token{AccessToken: "facebook_access_token"}).
					Return(nil, errors.New("facebook api error"))

				return mockFacebookClient, mockGenerator, mockUserRepo, mockAuthSessionRepo, mockJwtToken, mockRedisClient
			},
			verify: func(t *testing.T, got *entities.HandleFacebookCallbackResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
			},
		},
		{
			name: "Error_DatabaseLookupFailed",
			input: &entities.HandleFacebookCallbackReq{
				Code:  validCode,
				State: validState,
			},
			setup: func() (*mockAuthInfras.MockFacebookClient, *mockUtils.MockGenerator, *repositories.MockUserRepository, *repositories.MockAuthSessionRepository, *mockUtils.MockJwtToken, *mockDatabase.MockRedisClient) {
				mockFacebookClient := new(mockAuthInfras.MockFacebookClient)
				mockGenerator := new(mockUtils.MockGenerator)
				mockUserRepo := new(repositories.MockUserRepository)
				mockAuthSessionRepo := new(repositories.MockAuthSessionRepository)
				mockJwtToken := new(mockUtils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockFacebookClient.EXPECT().
					ExchangeCodeForToken(ctx, validCode).
					Return(&oauth2.Token{AccessToken: "facebook_access_token"}, nil)

				redisKey := fmt.Sprintf("%s%s", constants.RedisPrefixOAuthState, validState)
				mockRedisClient.EXPECT().
					Get(ctx, redisKey).
					Return(constants.FacebookProvider, nil)

				mockFacebookClient.EXPECT().
					GetUserInfo(ctx, &oauth2.Token{AccessToken: "facebook_access_token"}).
					Return(validUserInfo, nil)

				mockUserRepo.EXPECT().
					CheckUserExistsByProviderID(ctx, db.CheckUserExistsByProviderIDParams{
						ProviderID: sql.NullString{String: validUserInfo.ID, Valid: true},
						Provider:   constants.FacebookProvider,
					}).
					Return(false, errors.New("database error"))

				return mockFacebookClient, mockGenerator, mockUserRepo, mockAuthSessionRepo, mockJwtToken, mockRedisClient
			},
			verify: func(t *testing.T, got *entities.HandleFacebookCallbackResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
			},
		},
		{
			name: "Error_UserCreationFailed",
			input: &entities.HandleFacebookCallbackReq{
				Code:  validCode,
				State: validState,
			},
			setup: func() (*mockAuthInfras.MockFacebookClient, *mockUtils.MockGenerator, *repositories.MockUserRepository, *repositories.MockAuthSessionRepository, *mockUtils.MockJwtToken, *mockDatabase.MockRedisClient) {
				mockFacebookClient := new(mockAuthInfras.MockFacebookClient)
				mockGenerator := new(mockUtils.MockGenerator)
				mockUserRepo := new(repositories.MockUserRepository)
				mockAuthSessionRepo := new(repositories.MockAuthSessionRepository)
				mockJwtToken := new(mockUtils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockFacebookClient.EXPECT().
					ExchangeCodeForToken(ctx, validCode).
					Return(&oauth2.Token{AccessToken: "facebook_access_token"}, nil)

				redisKey := fmt.Sprintf("%s%s", constants.RedisPrefixOAuthState, validState)
				mockRedisClient.EXPECT().
					Get(ctx, redisKey).
					Return(constants.FacebookProvider, nil)

				mockFacebookClient.EXPECT().
					GetUserInfo(ctx, &oauth2.Token{AccessToken: "facebook_access_token"}).
					Return(validUserInfo, nil)

				mockUserRepo.EXPECT().
					CheckUserExistsByProviderID(ctx, db.CheckUserExistsByProviderIDParams{
						ProviderID: sql.NullString{String: validUserInfo.ID, Valid: true},
						Provider:   constants.FacebookProvider,
					}).
					Return(false, nil)

				userID := uuid.New()
				mockGenerator.EXPECT().
					GenerateUUID(ctx).
					Return(userID)

				mockUserRepo.EXPECT().
					CreateUserWithProvider(ctx, mock.MatchedBy(func(params *db.CreateUserWithProviderParams) bool {
						return params.ID == userID &&
							params.Email == validUserInfo.Email &&
							params.FullName == validUserInfo.Name &&
							params.Provider == constants.FacebookProvider &&
							params.ProviderID.String == validUserInfo.ID
					})).
					Return(errors.New("user creation failed"))

				return mockFacebookClient, mockGenerator, mockUserRepo, mockAuthSessionRepo, mockJwtToken, mockRedisClient
			},
			verify: func(t *testing.T, got *entities.HandleFacebookCallbackResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
			},
		},
		{
			name: "Error_TokenCreationFailed",
			input: &entities.HandleFacebookCallbackReq{
				Code:  validCode,
				State: validState,
			},
			setup: func() (*mockAuthInfras.MockFacebookClient, *mockUtils.MockGenerator, *repositories.MockUserRepository, *repositories.MockAuthSessionRepository, *mockUtils.MockJwtToken, *mockDatabase.MockRedisClient) {
				mockFacebookClient := new(mockAuthInfras.MockFacebookClient)
				mockGenerator := new(mockUtils.MockGenerator)
				mockUserRepo := new(repositories.MockUserRepository)
				mockAuthSessionRepo := new(repositories.MockAuthSessionRepository)
				mockJwtToken := new(mockUtils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockFacebookClient.EXPECT().
					ExchangeCodeForToken(ctx, validCode).
					Return(&oauth2.Token{AccessToken: "facebook_access_token"}, nil)

				redisKey := fmt.Sprintf("%s%s", constants.RedisPrefixOAuthState, validState)
				mockRedisClient.EXPECT().
					Get(ctx, redisKey).
					Return(constants.FacebookProvider, nil)

				mockFacebookClient.EXPECT().
					GetUserInfo(ctx, &oauth2.Token{AccessToken: "facebook_access_token"}).
					Return(validUserInfo, nil)

				mockUserRepo.EXPECT().
					CheckUserExistsByProviderID(ctx, db.CheckUserExistsByProviderIDParams{
						ProviderID: sql.NullString{String: validUserInfo.ID, Valid: true},
						Provider:   constants.FacebookProvider,
					}).
					Return(false, nil)

				userID := uuid.New()
				mockGenerator.EXPECT().
					GenerateUUID(ctx).
					Return(userID)

				mockUserRepo.EXPECT().
					CreateUserWithProvider(ctx, mock.MatchedBy(func(params *db.CreateUserWithProviderParams) bool {
						return params.ID == userID &&
							params.Email == validUserInfo.Email &&
							params.FullName == validUserInfo.Name &&
							params.Provider == constants.FacebookProvider &&
							params.ProviderID.String == validUserInfo.ID
					})).
					Return(nil)

				createdUser := db.Users{
					ID:         userID,
					Email:      validUserInfo.Email,
					FullName:   validUserInfo.Name,
					Provider:   constants.FacebookProvider,
					ProviderID: sql.NullString{String: validUserInfo.ID, Valid: true},
				}
				mockUserRepo.EXPECT().
					GetUserByProviderID(ctx, db.GetUserByProviderIDParams{
						ProviderID: sql.NullString{String: validUserInfo.ID, Valid: true},
						Provider:   constants.FacebookProvider,
					}).
					Return(createdUser, nil)

				mockJwtToken.EXPECT().
					CreateToken(ctx, mock.MatchedBy(func(req *entities.TokenRequest) bool {
						return req.UserID == userID.String() && req.Role == constants.UserRoleUser
					})).
					Return("", nil, errors.New("token creation failed"))

				return mockFacebookClient, mockGenerator, mockUserRepo, mockAuthSessionRepo, mockJwtToken, mockRedisClient
			},
			verify: func(t *testing.T, got *entities.HandleFacebookCallbackResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
			},
		},
		{
			name: "Error_SessionCreationFailed",
			input: &entities.HandleFacebookCallbackReq{
				Code:  validCode,
				State: validState,
			},
			setup: func() (*mockAuthInfras.MockFacebookClient, *mockUtils.MockGenerator, *repositories.MockUserRepository, *repositories.MockAuthSessionRepository, *mockUtils.MockJwtToken, *mockDatabase.MockRedisClient) {
				mockFacebookClient := new(mockAuthInfras.MockFacebookClient)
				mockGenerator := new(mockUtils.MockGenerator)
				mockUserRepo := new(repositories.MockUserRepository)
				mockAuthSessionRepo := new(repositories.MockAuthSessionRepository)
				mockJwtToken := new(mockUtils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockFacebookClient.EXPECT().
					ExchangeCodeForToken(ctx, validCode).
					Return(&oauth2.Token{AccessToken: "facebook_access_token"}, nil)

				redisKey := fmt.Sprintf("%s%s", constants.RedisPrefixOAuthState, validState)
				mockRedisClient.EXPECT().
					Get(ctx, redisKey).
					Return(constants.FacebookProvider, nil)

				mockFacebookClient.EXPECT().
					GetUserInfo(ctx, &oauth2.Token{AccessToken: "facebook_access_token"}).
					Return(validUserInfo, nil)

				mockUserRepo.EXPECT().
					CheckUserExistsByProviderID(ctx, db.CheckUserExistsByProviderIDParams{
						ProviderID: sql.NullString{String: validUserInfo.ID, Valid: true},
						Provider:   constants.FacebookProvider,
					}).
					Return(false, nil)

				userID := uuid.New()
				mockGenerator.EXPECT().
					GenerateUUID(ctx).
					Return(userID)

				mockUserRepo.EXPECT().
					CreateUserWithProvider(ctx, mock.MatchedBy(func(params *db.CreateUserWithProviderParams) bool {
						return params.ID == userID &&
							params.Email == validUserInfo.Email &&
							params.FullName == validUserInfo.Name &&
							params.Provider == constants.FacebookProvider &&
							params.ProviderID.String == validUserInfo.ID
					})).
					Return(nil)

				createdUser := db.Users{
					ID:         userID,
					Email:      validUserInfo.Email,
					FullName:   validUserInfo.Name,
					Provider:   constants.FacebookProvider,
					ProviderID: sql.NullString{String: validUserInfo.ID, Valid: true},
				}
				mockUserRepo.EXPECT().
					GetUserByProviderID(ctx, db.GetUserByProviderIDParams{
						ProviderID: sql.NullString{String: validUserInfo.ID, Valid: true},
						Provider:   constants.FacebookProvider,
					}).
					Return(createdUser, nil)

				mockJwtToken.EXPECT().
					CreateToken(ctx, mock.MatchedBy(func(req *entities.TokenRequest) bool {
						return req.UserID == userID.String() && req.Role == constants.UserRoleUser
					})).
					Return("access_token", &utilsPkg.SignInTokenPayload{ID: uuid.New(), ExpiredAt: time.Now().Add(time.Hour)}, nil).
					Times(1)

				mockJwtToken.EXPECT().
					CreateToken(ctx, mock.MatchedBy(func(req *entities.TokenRequest) bool {
						return req.UserID == userID.String() && req.Role == constants.UserRoleUser
					})).
					Return("refresh_token", &utilsPkg.SignInTokenPayload{ID: uuid.New(), ExpiredAt: time.Now().Add(time.Hour)}, nil).
					Times(1)

				mockJwtToken.EXPECT().
					HashTokenSHA256(ctx, "refresh_token").
					Return("hashed_refresh_token")

				mockAuthSessionRepo.EXPECT().
					CreateAuthSession(ctx, mock.MatchedBy(func(params *db.CreateAuthSessionParams) bool {
						return params.UserID == userID && params.RefreshTokenHash == "hashed_refresh_token"
					})).
					Return(errors.New("session creation failed"))

				return mockFacebookClient, mockGenerator, mockUserRepo, mockAuthSessionRepo, mockJwtToken, mockRedisClient
			},
			verify: func(t *testing.T, got *entities.HandleFacebookCallbackResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
			},
		},
		{
			name: "Error_RedisStateError",
			input: &entities.HandleFacebookCallbackReq{
				Code:  validCode,
				State: validState,
			},
			setup: func() (*mockAuthInfras.MockFacebookClient, *mockUtils.MockGenerator, *repositories.MockUserRepository, *repositories.MockAuthSessionRepository, *mockUtils.MockJwtToken, *mockDatabase.MockRedisClient) {
				mockFacebookClient := new(mockAuthInfras.MockFacebookClient)
				mockGenerator := new(mockUtils.MockGenerator)
				mockUserRepo := new(repositories.MockUserRepository)
				mockAuthSessionRepo := new(repositories.MockAuthSessionRepository)
				mockJwtToken := new(mockUtils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockFacebookClient.EXPECT().
					ExchangeCodeForToken(ctx, validCode).
					Return(&oauth2.Token{AccessToken: "facebook_access_token"}, nil)

				redisKey := fmt.Sprintf("%s%s", constants.RedisPrefixOAuthState, validState)
				mockRedisClient.EXPECT().
					Get(ctx, redisKey).
					Return("", errors.New("redis error"))

				return mockFacebookClient, mockGenerator, mockUserRepo, mockAuthSessionRepo, mockJwtToken, mockRedisClient
			},
			verify: func(t *testing.T, got *entities.HandleFacebookCallbackResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
			},
		},
		{
			name: "Error_InvalidStateValidation",
			input: &entities.HandleFacebookCallbackReq{
				Code:  validCode,
				State: validState,
			},
			setup: func() (*mockAuthInfras.MockFacebookClient, *mockUtils.MockGenerator, *repositories.MockUserRepository, *repositories.MockAuthSessionRepository, *mockUtils.MockJwtToken, *mockDatabase.MockRedisClient) {
				mockFacebookClient := new(mockAuthInfras.MockFacebookClient)
				mockGenerator := new(mockUtils.MockGenerator)
				mockUserRepo := new(repositories.MockUserRepository)
				mockAuthSessionRepo := new(repositories.MockAuthSessionRepository)
				mockJwtToken := new(mockUtils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockFacebookClient.EXPECT().
					ExchangeCodeForToken(ctx, validCode).
					Return(&oauth2.Token{AccessToken: "facebook_access_token"}, nil)

				redisKey := fmt.Sprintf("%s%s", constants.RedisPrefixOAuthState, validState)
				mockRedisClient.EXPECT().
					Get(ctx, redisKey).
					Return("different_state", nil)

				return mockFacebookClient, mockGenerator, mockUserRepo, mockAuthSessionRepo, mockJwtToken, mockRedisClient
			},
			verify: func(t *testing.T, got *entities.HandleFacebookCallbackResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
			},
		},
		{
			name: "Error_CreatedUserRetrievalFailed",
			input: &entities.HandleFacebookCallbackReq{
				Code:  validCode,
				State: validState,
			},
			setup: func() (*mockAuthInfras.MockFacebookClient, *mockUtils.MockGenerator, *repositories.MockUserRepository, *repositories.MockAuthSessionRepository, *mockUtils.MockJwtToken, *mockDatabase.MockRedisClient) {
				mockFacebookClient := new(mockAuthInfras.MockFacebookClient)
				mockGenerator := new(mockUtils.MockGenerator)
				mockUserRepo := new(repositories.MockUserRepository)
				mockAuthSessionRepo := new(repositories.MockAuthSessionRepository)
				mockJwtToken := new(mockUtils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockFacebookClient.EXPECT().
					ExchangeCodeForToken(ctx, validCode).
					Return(&oauth2.Token{AccessToken: "facebook_access_token"}, nil)

				redisKey := fmt.Sprintf("%s%s", constants.RedisPrefixOAuthState, validState)
				mockRedisClient.EXPECT().
					Get(ctx, redisKey).
					Return(constants.FacebookProvider, nil)

				mockFacebookClient.EXPECT().
					GetUserInfo(ctx, &oauth2.Token{AccessToken: "facebook_access_token"}).
					Return(validUserInfo, nil)

				mockUserRepo.EXPECT().
					CheckUserExistsByProviderID(ctx, db.CheckUserExistsByProviderIDParams{
						ProviderID: sql.NullString{String: validUserInfo.ID, Valid: true},
						Provider:   constants.FacebookProvider,
					}).
					Return(false, nil)

				userID := uuid.New()
				mockGenerator.EXPECT().
					GenerateUUID(ctx).
					Return(userID)

				mockUserRepo.EXPECT().
					CreateUserWithProvider(ctx, mock.MatchedBy(func(params *db.CreateUserWithProviderParams) bool {
						return params.ID == userID &&
							params.Email == validUserInfo.Email &&
							params.FullName == validUserInfo.Name &&
							params.Provider == constants.FacebookProvider &&
							params.ProviderID.String == validUserInfo.ID
					})).
					Return(nil)

				mockUserRepo.EXPECT().
					GetUserByProviderID(ctx, db.GetUserByProviderIDParams{
						ProviderID: sql.NullString{String: validUserInfo.ID, Valid: true},
						Provider:   constants.FacebookProvider,
					}).
					Return(db.Users{}, errors.New("user retrieval failed"))

				return mockFacebookClient, mockGenerator, mockUserRepo, mockAuthSessionRepo, mockJwtToken, mockRedisClient
			},
			verify: func(t *testing.T, got *entities.HandleFacebookCallbackResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
			},
		},
		{
			name: "Error_ExistingUserRetrievalFailed",
			input: &entities.HandleFacebookCallbackReq{
				Code:  validCode,
				State: validState,
			},
			setup: func() (*mockAuthInfras.MockFacebookClient, *mockUtils.MockGenerator, *repositories.MockUserRepository, *repositories.MockAuthSessionRepository, *mockUtils.MockJwtToken, *mockDatabase.MockRedisClient) {
				mockFacebookClient := new(mockAuthInfras.MockFacebookClient)
				mockGenerator := new(mockUtils.MockGenerator)
				mockUserRepo := new(repositories.MockUserRepository)
				mockAuthSessionRepo := new(repositories.MockAuthSessionRepository)
				mockJwtToken := new(mockUtils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockFacebookClient.EXPECT().
					ExchangeCodeForToken(ctx, validCode).
					Return(&oauth2.Token{AccessToken: "facebook_access_token"}, nil)

				redisKey := fmt.Sprintf("%s%s", constants.RedisPrefixOAuthState, validState)
				mockRedisClient.EXPECT().
					Get(ctx, redisKey).
					Return(constants.FacebookProvider, nil)

				mockFacebookClient.EXPECT().
					GetUserInfo(ctx, &oauth2.Token{AccessToken: "facebook_access_token"}).
					Return(validUserInfo, nil)

				mockUserRepo.EXPECT().
					CheckUserExistsByProviderID(ctx, db.CheckUserExistsByProviderIDParams{
						ProviderID: sql.NullString{String: validUserInfo.ID, Valid: true},
						Provider:   constants.FacebookProvider,
					}).
					Return(true, nil)

				mockUserRepo.EXPECT().
					GetUserByProviderID(ctx, db.GetUserByProviderIDParams{
						ProviderID: sql.NullString{String: validUserInfo.ID, Valid: true},
						Provider:   constants.FacebookProvider,
					}).
					Return(db.Users{}, errors.New("user retrieval failed"))

				return mockFacebookClient, mockGenerator, mockUserRepo, mockAuthSessionRepo, mockJwtToken, mockRedisClient
			},
			verify: func(t *testing.T, got *entities.HandleFacebookCallbackResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
			},
		},
		{
			name: "Error_UpdateLoginInfoFailed",
			input: &entities.HandleFacebookCallbackReq{
				Code:  validCode,
				State: validState,
			},
			setup: func() (*mockAuthInfras.MockFacebookClient, *mockUtils.MockGenerator, *repositories.MockUserRepository, *repositories.MockAuthSessionRepository, *mockUtils.MockJwtToken, *mockDatabase.MockRedisClient) {
				mockFacebookClient := new(mockAuthInfras.MockFacebookClient)
				mockGenerator := new(mockUtils.MockGenerator)
				mockUserRepo := new(repositories.MockUserRepository)
				mockAuthSessionRepo := new(repositories.MockAuthSessionRepository)
				mockJwtToken := new(mockUtils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockFacebookClient.EXPECT().
					ExchangeCodeForToken(ctx, validCode).
					Return(&oauth2.Token{AccessToken: "facebook_access_token"}, nil)

				redisKey := fmt.Sprintf("%s%s", constants.RedisPrefixOAuthState, validState)
				mockRedisClient.EXPECT().
					Get(ctx, redisKey).
					Return(constants.FacebookProvider, nil)

				mockFacebookClient.EXPECT().
					GetUserInfo(ctx, &oauth2.Token{AccessToken: "facebook_access_token"}).
					Return(validUserInfo, nil)

				mockUserRepo.EXPECT().
					CheckUserExistsByProviderID(ctx, db.CheckUserExistsByProviderIDParams{
						ProviderID: sql.NullString{String: validUserInfo.ID, Valid: true},
						Provider:   constants.FacebookProvider,
					}).
					Return(true, nil)

				existingUser := db.Users{
					ID:         uuid.New(),
					Email:      validUserInfo.Email,
					FullName:   validUserInfo.Name,
					Provider:   constants.FacebookProvider,
					ProviderID: sql.NullString{String: validUserInfo.ID, Valid: true},
				}
				mockUserRepo.EXPECT().
					GetUserByProviderID(ctx, db.GetUserByProviderIDParams{
						ProviderID: sql.NullString{String: validUserInfo.ID, Valid: true},
						Provider:   constants.FacebookProvider,
					}).
					Return(existingUser, nil)

				mockUserRepo.EXPECT().
					UpdateUserLoginInfo(ctx, mock.MatchedBy(func(params db.UpdateUserLoginInfoParams) bool {
						return params.ID == existingUser.ID
					})).
					Return(errors.New("update login info failed"))

				return mockFacebookClient, mockGenerator, mockUserRepo, mockAuthSessionRepo, mockJwtToken, mockRedisClient
			},
			verify: func(t *testing.T, got *entities.HandleFacebookCallbackResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			mockFacebookClient, mockGenerator, mockUserRepo, mockAuthSessionRepo, mockJwtToken, mockRedisClient := tC.setup()
			defer mockFacebookClient.AssertExpectations(t)
			defer mockGenerator.AssertExpectations(t)
			defer mockUserRepo.AssertExpectations(t)
			defer mockAuthSessionRepo.AssertExpectations(t)
			defer mockJwtToken.AssertExpectations(t)
			defer mockRedisClient.AssertExpectations(t)

			mockConfig := &config.Config{
				AuthConfig: config.AuthConfig{
					AccessTokenDuration:  time.Hour,
					RefreshTokenDuration: time.Hour * 24,
				},
			}

			svc := NewUserService(
				lgr,
				mockUserRepo,
				mockAuthSessionRepo,
				mockJwtToken,
				nil,
				mockConfig,
				nil,
				mockRedisClient,
				nil,
				nil,
				mockGenerator,
				nil,
				mockFacebookClient,
			)

			got, gotErr := svc.HandleFacebookCallback(ctx, tC.input)
			tC.verify(t, got, gotErr)
		})
	}
}

func TestUserService_GetFacebookAuthURL(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()
	mockErr := errors.New("error")

	okResponse := &entities.FacebookAuthURLResponse{
		AuthURL: "https://www.facebook.com/auth/url",
	}

	testCases := []struct {
		name   string
		setup  func() (*mockDatabase.MockRedisClient, *mockAuthInfras.MockFacebookClient, *mockUtils.MockGenerator)
		verify func(t *testing.T, got *entities.FacebookAuthURLResponse, gotErr error)
	}{
		{
			name: "Success",
			setup: func() (*mockDatabase.MockRedisClient, *mockAuthInfras.MockFacebookClient, *mockUtils.MockGenerator) {
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockFacebookClient := new(mockAuthInfras.MockFacebookClient)
				mockGenerator := new(mockUtils.MockGenerator)

				redisKey := fmt.Sprintf("%s%s", constants.RedisPrefixOAuthState, "oauth_state")
				redisPayload := database.RedisPayload{
					Key:   redisKey,
					Value: constants.FacebookProvider,
					TTL:   constants.RedisTTLOAuthState,
				}

				mockGenerator.EXPECT().
					GenerateCryptographicallySecureString(ctx, 32).
					Return("oauth_state")

				mockRedisClient.EXPECT().
					Set(ctx, redisPayload).
					Return(nil)

				mockFacebookClient.EXPECT().
					GetAuthURL("oauth_state").
					Return("https://www.facebook.com/auth/url")

				return mockRedisClient, mockFacebookClient, mockGenerator
			},
			verify: func(t *testing.T, got *entities.FacebookAuthURLResponse, gotErr error) {
				assert.Equal(t, okResponse, got)
				assert.NoError(t, gotErr)
			},
		},
		{
			name: "Error_RedisSetFailed",
			setup: func() (*mockDatabase.MockRedisClient, *mockAuthInfras.MockFacebookClient, *mockUtils.MockGenerator) {
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockFacebookClient := new(mockAuthInfras.MockFacebookClient)
				mockGenerator := new(mockUtils.MockGenerator)

				redisKey := fmt.Sprintf("%s%s", constants.RedisPrefixOAuthState, "oauth_state")
				redisPayload := database.RedisPayload{
					Key:   redisKey,
					Value: constants.FacebookProvider,
					TTL:   constants.RedisTTLOAuthState,
				}

				mockGenerator.EXPECT().
					GenerateCryptographicallySecureString(ctx, 32).
					Return("oauth_state")

				mockRedisClient.EXPECT().
					Set(ctx, redisPayload).
					Return(mockErr)

				return mockRedisClient, mockFacebookClient, mockGenerator
			},
			verify: func(t *testing.T, got *entities.FacebookAuthURLResponse, gotErr error) {
				assert.Error(t, gotErr)
				assert.ErrorIs(t, gotErr, mockErr)
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			mockRedisClient, mockFacebookClient, mockGenerator := tC.setup()
			defer mockRedisClient.AssertExpectations(t)
			defer mockFacebookClient.AssertExpectations(t)
			defer mockGenerator.AssertExpectations(t)

			svc := NewUserService(lgr, nil, nil, nil, nil, nil, nil, mockRedisClient, nil, nil, mockGenerator, nil, mockFacebookClient)
			got, gotErr := svc.GetFacebookAuthURL(ctx)

			tC.verify(t, got, gotErr)
		})
	}
}
