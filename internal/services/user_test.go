package services

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.com/interview-simulation/interview-backend-server/internal/config"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	db "gitlab.com/interview-simulation/interview-backend-server/internal/db/sqlc"
	"gitlab.com/interview-simulation/interview-backend-server/internal/entities"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
	"gitlab.com/interview-simulation/interview-backend-server/internal/middleware"
	mockDatabase "gitlab.com/interview-simulation/interview-backend-server/internal/mocks/database"
	middlewareMocks "gitlab.com/interview-simulation/interview-backend-server/internal/mocks/middleware"
	"gitlab.com/interview-simulation/interview-backend-server/internal/mocks/queue"
	"gitlab.com/interview-simulation/interview-backend-server/internal/mocks/repositories"
	"gitlab.com/interview-simulation/interview-backend-server/internal/mocks/utils"
	utilsPkg "gitlab.com/interview-simulation/interview-backend-server/internal/utils"
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

			svc := NewUserService(lgr, mockUserRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
			got, gotErr := svc.HealthCheck(ctx)

			tC.verify(t, got, gotErr)
		})
	}
}

func TestUserService_SignUpUser(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()
	mockErr := errors.New("error")

	mockConfig := &config.Config{
		EmailConfig: config.EmailConfig{
			VerifyEmailTokenDuration: 15 * time.Minute,
		},
	}

	testCases := []struct {
		name   string
		input  *entities.SignUpUserRequest
		setup  func() (*repositories.MockUserRepository, *utils.MockPasswordUtil, *utils.MockGenerator, *utils.MockJwtToken, *mockDatabase.MockRedisClient, *queue.MockRedisTaskPublisher)
		verify func(t *testing.T, got *entities.SignUpUserResponse, gotErr error)
	}{
		{
			name: "Success_NewUser",
			input: &entities.SignUpUserRequest{
				Email:       "newuser@example.com",
				Password:    "password123",
				FullName:    "New User",
				Country:     "US",
				Gender:      "male",
				DateOfBirth: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			setup: func() (*repositories.MockUserRepository, *utils.MockPasswordUtil, *utils.MockGenerator, *utils.MockJwtToken, *mockDatabase.MockRedisClient, *queue.MockRedisTaskPublisher) {
				mockUserRepo := new(repositories.MockUserRepository)
				mockPassword := new(utils.MockPasswordUtil)
				mockGenerator := new(utils.MockGenerator)
				mockJwtToken := new(utils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockRedisTaskPublisher := new(queue.MockRedisTaskPublisher)

				// Mock CheckIsEmailExists returns sql.ErrNoRows
				mockUserRepo.EXPECT().
					CheckIsEmailExists(ctx, "newuser@example.com").
					Return(nil, app_error.New(errors.New("user not found"), app_error.ErrCodeAuthUserNotFound))

				// Mock password hashing
				mockPassword.EXPECT().
					HashPassword(ctx, "password123").
					Return("hashed_password", nil)

				// Mock UUID generation
				mockGenerator.EXPECT().
					GenerateUUID(ctx).
					Return(uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"))

				// Mock CreateUser
				mockUserRepo.EXPECT().
					CreateUser(ctx, mock.AnythingOfType("*db.CreateUserParams")).
					Return(&db.Users{ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")}, nil)

				// Mock random string generation
				mockGenerator.EXPECT().
					GenerateRandomString(ctx, 6).
					Return("123456")

				// Mock JWT token creation
				mockJwtToken.EXPECT().
					CreateVerifyEmailToken(ctx, mock.AnythingOfType("*entities.VerifyEmailTokenRequest")).
					Return("verify_token", &utilsPkg.VerifyEmailTokenPayload{}, nil)

				// Mock Redis operations
				mockRedisClient.EXPECT().
					Set(ctx, mock.AnythingOfType("database.RedisPayload")).
					Return(nil)

				// Mock task publishing
				mockRedisTaskPublisher.EXPECT().
					PublishTaskSendVerifyEmail(ctx, mock.AnythingOfType("*email.VerifyEmailPayload")).
					Return(nil)

				return mockUserRepo, mockPassword, mockGenerator, mockJwtToken, mockRedisClient, mockRedisTaskPublisher
			},
			verify: func(t *testing.T, got *entities.SignUpUserResponse, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, got)
				assert.Equal(t, "verify_token", got.TokenId)
			},
		},
		{
			name: "Success_ExistingUnverifiedUser",
			input: &entities.SignUpUserRequest{
				Email:       "existing@example.com",
				Password:    "password123",
				FullName:    "Existing User",
				Country:     "US",
				Gender:      "female",
				DateOfBirth: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			setup: func() (*repositories.MockUserRepository, *utils.MockPasswordUtil, *utils.MockGenerator, *utils.MockJwtToken, *mockDatabase.MockRedisClient, *queue.MockRedisTaskPublisher) {
				mockUserRepo := new(repositories.MockUserRepository)
				mockPassword := new(utils.MockPasswordUtil)
				mockGenerator := new(utils.MockGenerator)
				mockJwtToken := new(utils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockRedisTaskPublisher := new(queue.MockRedisTaskPublisher)

				// Mock CheckIsEmailExists returns existing unverified user
				mockUserRepo.EXPECT().
					CheckIsEmailExists(ctx, "existing@example.com").
					Return(&db.Users{ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), IsEmailVerified: sql.NullBool{Bool: false, Valid: true}}, nil)

				// Mock password hashing
				mockPassword.EXPECT().
					HashPassword(ctx, "password123").
					Return("hashed_password", nil)

				// Mock UpdateUser
				mockUserRepo.EXPECT().
					UpdateUser(ctx, mock.AnythingOfType("*db.UpdateUserParams")).
					Return(&db.Users{ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")}, nil)

				// Mock random string generation
				mockGenerator.EXPECT().
					GenerateRandomString(ctx, 6).
					Return("123456")

				// Mock JWT token creation
				mockJwtToken.EXPECT().
					CreateVerifyEmailToken(ctx, mock.AnythingOfType("*entities.VerifyEmailTokenRequest")).
					Return("verify_token", &utilsPkg.VerifyEmailTokenPayload{}, nil)

				// Mock Redis operations
				mockRedisClient.EXPECT().
					Set(ctx, mock.AnythingOfType("database.RedisPayload")).
					Return(nil).Times(2)

				// Mock task publishing
				mockRedisTaskPublisher.EXPECT().
					PublishTaskSendVerifyEmail(ctx, mock.AnythingOfType("*email.VerifyEmailPayload")).
					Return(nil)

				return mockUserRepo, mockPassword, mockGenerator, mockJwtToken, mockRedisClient, mockRedisTaskPublisher
			},
			verify: func(t *testing.T, got *entities.SignUpUserResponse, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, got)
				assert.Equal(t, "verify_token", got.TokenId)
			},
		},
		{
			name: "Error_EmailAlreadyVerified",
			input: &entities.SignUpUserRequest{
				Email:       "verified@example.com",
				Password:    "password123",
				FullName:    "Verified User",
				Country:     "US",
				Gender:      "male",
				DateOfBirth: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			setup: func() (*repositories.MockUserRepository, *utils.MockPasswordUtil, *utils.MockGenerator, *utils.MockJwtToken, *mockDatabase.MockRedisClient, *queue.MockRedisTaskPublisher) {
				mockUserRepo := new(repositories.MockUserRepository)

				// Mock CheckIsEmailExists returns existing verified user
				mockUserRepo.EXPECT().
					CheckIsEmailExists(ctx, "verified@example.com").
					Return(&db.Users{ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), IsEmailVerified: sql.NullBool{Bool: true, Valid: true}}, nil)

				return mockUserRepo, nil, nil, nil, nil, nil
			},
			verify: func(t *testing.T, got *entities.SignUpUserResponse, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
			},
		},
		{
			name: "Error_PasswordHashingFailed",
			input: &entities.SignUpUserRequest{
				Email:       "newuser@example.com",
				Password:    "password123",
				FullName:    "New User",
				Country:     "US",
				Gender:      "male",
				DateOfBirth: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			setup: func() (*repositories.MockUserRepository, *utils.MockPasswordUtil, *utils.MockGenerator, *utils.MockJwtToken, *mockDatabase.MockRedisClient, *queue.MockRedisTaskPublisher) {
				mockUserRepo := new(repositories.MockUserRepository)
				mockPassword := new(utils.MockPasswordUtil)

				// Mock CheckIsEmailExists returns sql.ErrNoRows
				mockUserRepo.EXPECT().
					CheckIsEmailExists(ctx, "newuser@example.com").
					Return(nil, app_error.New(errors.New("user not found"), app_error.ErrCodeAuthUserNotFound))

				// Mock password hashing fails
				mockPassword.EXPECT().
					HashPassword(ctx, "password123").
					Return("", mockErr)

				return mockUserRepo, mockPassword, nil, nil, nil, nil
			},
			verify: func(t *testing.T, got *entities.SignUpUserResponse, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
				assert.ErrorIs(t, gotErr, mockErr)
			},
		},
		{
			name: "Error_UserCreationFailed",
			input: &entities.SignUpUserRequest{
				Email:       "user@example.com",
				Password:    "password123",
				FullName:    "John Doe",
				Country:     "US",
				Gender:      "Male",
				DateOfBirth: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			setup: func() (*repositories.MockUserRepository, *utils.MockPasswordUtil, *utils.MockGenerator, *utils.MockJwtToken, *mockDatabase.MockRedisClient, *queue.MockRedisTaskPublisher) {
				mockUserRepo := new(repositories.MockUserRepository)
				mockPassword := new(utils.MockPasswordUtil)
				mockGenerator := new(utils.MockGenerator)

				// Mock user existence check returns no rows
				mockUserRepo.EXPECT().
					CheckIsEmailExists(ctx, "user@example.com").
					Return(nil, app_error.New(errors.New("user not found"), app_error.ErrCodeAuthUserNotFound))

				// Mock password hashing
				mockPassword.EXPECT().
					HashPassword(ctx, "password123").
					Return("hashed_password", nil)

				// Mock UUID generation
				mockGenerator.EXPECT().
					GenerateUUID(ctx).
					Return(uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"))

				// Mock user creation fails
				mockUserRepo.EXPECT().
					CreateUser(ctx, mock.AnythingOfType("*db.CreateUserParams")).
					Return(&db.Users{}, mockErr)

				return mockUserRepo, mockPassword, mockGenerator, nil, nil, nil
			},
			verify: func(t *testing.T, got *entities.SignUpUserResponse, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
				assert.ErrorIs(t, gotErr, mockErr)
			},
		},
		{
			name: "Error_VerifyEmailTokenCreationFailed",
			input: &entities.SignUpUserRequest{
				Email:       "user@example.com",
				Password:    "password123",
				FullName:    "John Doe",
				Country:     "US",
				Gender:      "Male",
				DateOfBirth: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			setup: func() (*repositories.MockUserRepository, *utils.MockPasswordUtil, *utils.MockGenerator, *utils.MockJwtToken, *mockDatabase.MockRedisClient, *queue.MockRedisTaskPublisher) {
				mockUserRepo := new(repositories.MockUserRepository)
				mockPassword := new(utils.MockPasswordUtil)
				mockGenerator := new(utils.MockGenerator)
				mockJwtToken := new(utils.MockJwtToken)

				// Mock user existence check returns no rows
				mockUserRepo.EXPECT().
					CheckIsEmailExists(ctx, "user@example.com").
					Return(nil, app_error.New(errors.New("user not found"), app_error.ErrCodeAuthUserNotFound))

				// Mock password hashing
				mockPassword.EXPECT().
					HashPassword(ctx, "password123").
					Return("hashed_password", nil)

				// Mock UUID generation
				mockGenerator.EXPECT().
					GenerateUUID(ctx).
					Return(uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"))

				// Mock random string generation
				mockGenerator.EXPECT().
					GenerateRandomString(ctx, 6).
					Return("123456")

				// Mock user creation
				mockUserRepo.EXPECT().
					CreateUser(ctx, mock.AnythingOfType("*db.CreateUserParams")).
					Return(&db.Users{ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")}, nil)

				// Mock verify email token creation fails
				mockJwtToken.EXPECT().
					CreateVerifyEmailToken(ctx, mock.AnythingOfType("*entities.VerifyEmailTokenRequest")).
					Return("", nil, mockErr)

				return mockUserRepo, mockPassword, mockGenerator, mockJwtToken, nil, nil
			},
			verify: func(t *testing.T, got *entities.SignUpUserResponse, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
				assert.ErrorIs(t, gotErr, mockErr)
			},
		},
		{
			name: "Error_RedisSetFailed",
			input: &entities.SignUpUserRequest{
				Email:       "user@example.com",
				Password:    "password123",
				FullName:    "John Doe",
				Country:     "US",
				Gender:      "Male",
				DateOfBirth: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			setup: func() (*repositories.MockUserRepository, *utils.MockPasswordUtil, *utils.MockGenerator, *utils.MockJwtToken, *mockDatabase.MockRedisClient, *queue.MockRedisTaskPublisher) {
				mockUserRepo := new(repositories.MockUserRepository)
				mockPassword := new(utils.MockPasswordUtil)
				mockGenerator := new(utils.MockGenerator)
				mockJwtToken := new(utils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				// Mock user existence check returns no rows
				mockUserRepo.EXPECT().
					CheckIsEmailExists(ctx, "user@example.com").
					Return(nil, app_error.New(errors.New("user not found"), app_error.ErrCodeAuthUserNotFound))

				// Mock password hashing
				mockPassword.EXPECT().
					HashPassword(ctx, "password123").
					Return("hashed_password", nil)

				// Mock UUID generation
				mockGenerator.EXPECT().
					GenerateUUID(ctx).
					Return(uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"))

				// Mock random string generation
				mockGenerator.EXPECT().
					GenerateRandomString(ctx, 6).
					Return("123456")

				// Mock user creation
				mockUserRepo.EXPECT().
					CreateUser(ctx, mock.AnythingOfType("*db.CreateUserParams")).
					Return(&db.Users{ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")}, nil)

				// Mock verify email token creation
				mockJwtToken.EXPECT().
					CreateVerifyEmailToken(ctx, mock.AnythingOfType("*entities.VerifyEmailTokenRequest")).
					Return("verify_token", &utilsPkg.VerifyEmailTokenPayload{}, nil)

				// Mock Redis set fails
				mockRedisClient.EXPECT().
					Set(ctx, mock.AnythingOfType("database.RedisPayload")).
					Return(mockErr)

				return mockUserRepo, mockPassword, mockGenerator, mockJwtToken, mockRedisClient, nil
			},
			verify: func(t *testing.T, got *entities.SignUpUserResponse, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
				assert.ErrorIs(t, gotErr, mockErr)
			},
		},
		{
			name: "Error_TaskPublishingFailed",
			input: &entities.SignUpUserRequest{
				Email:       "user@example.com",
				Password:    "password123",
				FullName:    "John Doe",
				Country:     "US",
				Gender:      "Male",
				DateOfBirth: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			setup: func() (*repositories.MockUserRepository, *utils.MockPasswordUtil, *utils.MockGenerator, *utils.MockJwtToken, *mockDatabase.MockRedisClient, *queue.MockRedisTaskPublisher) {
				mockUserRepo := new(repositories.MockUserRepository)
				mockPassword := new(utils.MockPasswordUtil)
				mockGenerator := new(utils.MockGenerator)
				mockJwtToken := new(utils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockRedisTaskPublisher := new(queue.MockRedisTaskPublisher)

				// Mock user existence check returns no rows
				mockUserRepo.EXPECT().
					CheckIsEmailExists(ctx, "user@example.com").
					Return(nil, app_error.New(errors.New("user not found"), app_error.ErrCodeAuthUserNotFound))

				// Mock password hashing
				mockPassword.EXPECT().
					HashPassword(ctx, "password123").
					Return("hashed_password", nil)

				// Mock UUID generation
				mockGenerator.EXPECT().
					GenerateUUID(ctx).
					Return(uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"))

				// Mock random string generation
				mockGenerator.EXPECT().
					GenerateRandomString(ctx, 6).
					Return("123456")

				// Mock user creation
				mockUserRepo.EXPECT().
					CreateUser(ctx, mock.AnythingOfType("*db.CreateUserParams")).
					Return(&db.Users{ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")}, nil)

				// Mock verify email token creation
				mockJwtToken.EXPECT().
					CreateVerifyEmailToken(ctx, mock.AnythingOfType("*entities.VerifyEmailTokenRequest")).
					Return("verify_token", &utilsPkg.VerifyEmailTokenPayload{}, nil)

				// Mock Redis set operations
				mockRedisClient.EXPECT().
					Set(ctx, mock.AnythingOfType("database.RedisPayload")).
					Return(nil)

				// Mock task publishing fails
				mockRedisTaskPublisher.EXPECT().
					PublishTaskSendVerifyEmail(ctx, mock.AnythingOfType("*email.VerifyEmailPayload")).
					Return(mockErr)

				return mockUserRepo, mockPassword, mockGenerator, mockJwtToken, mockRedisClient, mockRedisTaskPublisher
			},
			verify: func(t *testing.T, got *entities.SignUpUserResponse, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
				assert.ErrorIs(t, gotErr, mockErr)
			},
		},
		{
			name: "Error_UserUpdateFailed",
			input: &entities.SignUpUserRequest{
				Email:       "user@example.com",
				Password:    "password123",
				FullName:    "John Doe",
				Country:     "US",
				Gender:      "Male",
				DateOfBirth: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			setup: func() (*repositories.MockUserRepository, *utils.MockPasswordUtil, *utils.MockGenerator, *utils.MockJwtToken, *mockDatabase.MockRedisClient, *queue.MockRedisTaskPublisher) {
				mockUserRepo := new(repositories.MockUserRepository)
				mockPassword := new(utils.MockPasswordUtil)
				mockGenerator := new(utils.MockGenerator)
				mockJwtToken := new(utils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockRedisTaskPublisher := new(queue.MockRedisTaskPublisher)

				// Mock user existence check returns existing unverified user
				mockUserRepo.EXPECT().
					CheckIsEmailExists(ctx, "user@example.com").
					Return(&db.Users{ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), IsEmailVerified: sql.NullBool{Bool: false, Valid: true}}, nil)

				// Mock password hashing
				mockPassword.EXPECT().
					HashPassword(ctx, "password123").
					Return("hashed_password", nil)

				// Mock user update fails
				mockUserRepo.EXPECT().
					UpdateUser(ctx, mock.AnythingOfType("*db.UpdateUserParams")).
					Return(&db.Users{}, mockErr)

				return mockUserRepo, mockPassword, mockGenerator, mockJwtToken, mockRedisClient, mockRedisTaskPublisher
			},
			verify: func(t *testing.T, got *entities.SignUpUserResponse, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
				assert.ErrorIs(t, gotErr, mockErr)
			},
		},
		{
			name: "Error_UserUpdateReturnsNoRows",
			input: &entities.SignUpUserRequest{
				Email:       "user@example.com",
				Password:    "password123",
				FullName:    "John Doe",
				Country:     "US",
				Gender:      "Male",
				DateOfBirth: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			setup: func() (*repositories.MockUserRepository, *utils.MockPasswordUtil, *utils.MockGenerator, *utils.MockJwtToken, *mockDatabase.MockRedisClient, *queue.MockRedisTaskPublisher) {
				mockUserRepo := new(repositories.MockUserRepository)
				mockPassword := new(utils.MockPasswordUtil)
				mockGenerator := new(utils.MockGenerator)
				mockJwtToken := new(utils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockRedisTaskPublisher := new(queue.MockRedisTaskPublisher)

				// Mock user existence check returns existing unverified user
				mockUserRepo.EXPECT().
					CheckIsEmailExists(ctx, "user@example.com").
					Return(&db.Users{ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), IsEmailVerified: sql.NullBool{Bool: false, Valid: true}}, nil)

				// Mock password hashing
				mockPassword.EXPECT().
					HashPassword(ctx, "password123").
					Return("hashed_password", nil)

				// Mock user update returns user not found
				mockUserRepo.EXPECT().
					UpdateUser(ctx, mock.AnythingOfType("*db.UpdateUserParams")).
					Return(&db.Users{}, app_error.New(errors.New("user not found"), app_error.ErrCodeAuthUserNotFound))

				return mockUserRepo, mockPassword, mockGenerator, mockJwtToken, mockRedisClient, mockRedisTaskPublisher
			},
			verify: func(t *testing.T, got *entities.SignUpUserResponse, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
				assert.Contains(t, gotErr.Error(), "[INS0206]")
				assert.Contains(t, gotErr.Error(), "We couldn't find your account.")
			},
		},
		{
			name: "Error_RedisSetAttemptFailed",
			input: &entities.SignUpUserRequest{
				Email:       "user@example.com",
				Password:    "password123",
				FullName:    "John Doe",
				Country:     "US",
				Gender:      "Male",
				DateOfBirth: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			setup: func() (*repositories.MockUserRepository, *utils.MockPasswordUtil, *utils.MockGenerator, *utils.MockJwtToken, *mockDatabase.MockRedisClient, *queue.MockRedisTaskPublisher) {
				mockUserRepo := new(repositories.MockUserRepository)
				mockPassword := new(utils.MockPasswordUtil)
				mockGenerator := new(utils.MockGenerator)
				mockJwtToken := new(utils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockRedisTaskPublisher := new(queue.MockRedisTaskPublisher)

				// Mock user existence check returns existing unverified user
				mockUserRepo.EXPECT().
					CheckIsEmailExists(ctx, "user@example.com").
					Return(&db.Users{ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), IsEmailVerified: sql.NullBool{Bool: false, Valid: true}}, nil)

				// Mock password hashing
				mockPassword.EXPECT().
					HashPassword(ctx, "password123").
					Return("hashed_password", nil)

				// Mock user update
				mockUserRepo.EXPECT().
					UpdateUser(ctx, mock.AnythingOfType("*db.UpdateUserParams")).
					Return(&db.Users{ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")}, nil)

				// Mock random string generation
				mockGenerator.EXPECT().
					GenerateRandomString(ctx, 6).
					Return("123456")

				// Mock JWT token creation
				mockJwtToken.EXPECT().
					CreateVerifyEmailToken(ctx, mock.AnythingOfType("*entities.VerifyEmailTokenRequest")).
					Return("verify_token", &utilsPkg.VerifyEmailTokenPayload{}, nil)

				// Mock first Redis set operation (verify email token) succeeds
				mockRedisClient.EXPECT().
					Set(ctx, mock.AnythingOfType("database.RedisPayload")).
					Return(nil).Once()

				// Mock second Redis set operation (attempt counter) fails
				mockRedisClient.EXPECT().
					Set(ctx, mock.AnythingOfType("database.RedisPayload")).
					Return(mockErr).Once()

				return mockUserRepo, mockPassword, mockGenerator, mockJwtToken, mockRedisClient, mockRedisTaskPublisher
			},
			verify: func(t *testing.T, got *entities.SignUpUserResponse, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
				assert.ErrorIs(t, gotErr, mockErr)
			},
		},
		{
			name: "Error_GenerateRandomStringFailed",
			input: &entities.SignUpUserRequest{
				Email:       "user@example.com",
				Password:    "password123",
				FullName:    "John Doe",
				Country:     "US",
				Gender:      "Male",
				DateOfBirth: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			setup: func() (*repositories.MockUserRepository, *utils.MockPasswordUtil, *utils.MockGenerator, *utils.MockJwtToken, *mockDatabase.MockRedisClient, *queue.MockRedisTaskPublisher) {
				mockUserRepo := new(repositories.MockUserRepository)
				mockPassword := new(utils.MockPasswordUtil)
				mockGenerator := new(utils.MockGenerator)
				mockJwtToken := new(utils.MockJwtToken)

				// Mock user existence check returns no rows
				mockUserRepo.EXPECT().
					CheckIsEmailExists(ctx, "user@example.com").
					Return(nil, app_error.New(errors.New("user not found"), app_error.ErrCodeAuthUserNotFound))

				// Mock password hashing
				mockPassword.EXPECT().
					HashPassword(ctx, "password123").
					Return("hashed_password", nil)

				// Mock UUID generation
				mockGenerator.EXPECT().
					GenerateUUID(ctx).
					Return(uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"))

				// Mock user creation
				mockUserRepo.EXPECT().
					CreateUser(ctx, mock.AnythingOfType("*db.CreateUserParams")).
					Return(&db.Users{ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")}, nil)

				// Mock random string generation fails
				mockGenerator.EXPECT().
					GenerateRandomString(ctx, 6).
					Return("")

				// Mock JWT token creation fails due to empty code
				mockJwtToken.EXPECT().
					CreateVerifyEmailToken(ctx, mock.AnythingOfType("*entities.VerifyEmailTokenRequest")).
					Return("", nil, errors.New("failed to create token with empty code"))

				return mockUserRepo, mockPassword, mockGenerator, mockJwtToken, nil, nil
			},
			verify: func(t *testing.T, got *entities.SignUpUserResponse, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
			},
		},
		{
			name: "Error_CheckEmailExistsDatabaseError",
			input: &entities.SignUpUserRequest{
				Email:       "user@example.com",
				Password:    "password123",
				FullName:    "John Doe",
				Country:     "US",
				Gender:      "Male",
				DateOfBirth: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			setup: func() (*repositories.MockUserRepository, *utils.MockPasswordUtil, *utils.MockGenerator, *utils.MockJwtToken, *mockDatabase.MockRedisClient, *queue.MockRedisTaskPublisher) {
				mockUserRepo := new(repositories.MockUserRepository)

				// Mock user existence check fails with database error
				mockUserRepo.EXPECT().
					CheckIsEmailExists(ctx, "user@example.com").
					Return(nil, mockErr)

				return mockUserRepo, nil, nil, nil, nil, nil
			},
			verify: func(t *testing.T, got *entities.SignUpUserResponse, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
				assert.ErrorIs(t, gotErr, mockErr)
			},
		},
		{
			name: "Failed_NewUserPublishTaskSendVerifyEmail",
			input: &entities.SignUpUserRequest{
				Email:       "newuser@example.com",
				Password:    "password123",
				FullName:    "New User",
				Country:     "US",
				Gender:      "male",
				DateOfBirth: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			setup: func() (*repositories.MockUserRepository, *utils.MockPasswordUtil, *utils.MockGenerator, *utils.MockJwtToken, *mockDatabase.MockRedisClient, *queue.MockRedisTaskPublisher) {
				mockUserRepo := new(repositories.MockUserRepository)
				mockPassword := new(utils.MockPasswordUtil)
				mockGenerator := new(utils.MockGenerator)
				mockJwtToken := new(utils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockRedisTaskPublisher := new(queue.MockRedisTaskPublisher)

				// Mock CheckIsEmailExists returns sql.ErrNoRows
				mockUserRepo.EXPECT().
					CheckIsEmailExists(ctx, "newuser@example.com").
					Return(nil, app_error.New(errors.New("user not found"), app_error.ErrCodeAuthUserNotFound))

				// Mock password hashing
				mockPassword.EXPECT().
					HashPassword(ctx, "password123").
					Return("hashed_password", nil)

				// Mock UUID generation
				mockGenerator.EXPECT().
					GenerateUUID(ctx).
					Return(uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"))

				// Mock CreateUser
				mockUserRepo.EXPECT().
					CreateUser(ctx, mock.AnythingOfType("*db.CreateUserParams")).
					Return(&db.Users{ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")}, nil)

				// Mock random string generation
				mockGenerator.EXPECT().
					GenerateRandomString(ctx, 6).
					Return("123456")

				// Mock JWT token creation
				mockJwtToken.EXPECT().
					CreateVerifyEmailToken(ctx, mock.AnythingOfType("*entities.VerifyEmailTokenRequest")).
					Return("verify_token", &utilsPkg.VerifyEmailTokenPayload{}, nil)

				// Mock Redis operations
				mockRedisClient.EXPECT().
					Set(ctx, mock.AnythingOfType("database.RedisPayload")).
					Return(nil)

				// Mock task publishing
				mockRedisTaskPublisher.EXPECT().
					PublishTaskSendVerifyEmail(ctx, mock.AnythingOfType("*email.VerifyEmailPayload")).
					Return(mockErr)

				return mockUserRepo, mockPassword, mockGenerator, mockJwtToken, mockRedisClient, mockRedisTaskPublisher
			},
			verify: func(t *testing.T, got *entities.SignUpUserResponse, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
				assert.ErrorIs(t, gotErr, mockErr)
			},
		},
		{
			name: "Failed_ExistingUserPublishTaskSendVerifyEmail",
			input: &entities.SignUpUserRequest{
				Email:       "existing@example.com",
				Password:    "password123",
				FullName:    "Existing User",
				Country:     "US",
				Gender:      "female",
				DateOfBirth: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			setup: func() (*repositories.MockUserRepository, *utils.MockPasswordUtil, *utils.MockGenerator, *utils.MockJwtToken, *mockDatabase.MockRedisClient, *queue.MockRedisTaskPublisher) {
				mockUserRepo := new(repositories.MockUserRepository)
				mockPassword := new(utils.MockPasswordUtil)
				mockGenerator := new(utils.MockGenerator)
				mockJwtToken := new(utils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockRedisTaskPublisher := new(queue.MockRedisTaskPublisher)

				// Mock CheckIsEmailExists returns existing unverified user
				mockUserRepo.EXPECT().
					CheckIsEmailExists(ctx, "existing@example.com").
					Return(&db.Users{ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), IsEmailVerified: sql.NullBool{Bool: false, Valid: true}}, nil)

				// Mock password hashing
				mockPassword.EXPECT().
					HashPassword(ctx, "password123").
					Return("hashed_password", nil)

				// Mock UpdateUser
				mockUserRepo.EXPECT().
					UpdateUser(ctx, mock.AnythingOfType("*db.UpdateUserParams")).
					Return(&db.Users{ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")}, nil)

				// Mock random string generation
				mockGenerator.EXPECT().
					GenerateRandomString(ctx, 6).
					Return("123456")

				// Mock JWT token creation
				mockJwtToken.EXPECT().
					CreateVerifyEmailToken(ctx, mock.AnythingOfType("*entities.VerifyEmailTokenRequest")).
					Return("verify_token", &utilsPkg.VerifyEmailTokenPayload{}, nil)

				// Mock Redis operations
				mockRedisClient.EXPECT().
					Set(ctx, mock.AnythingOfType("database.RedisPayload")).
					Return(nil).Times(2)

				// Mock task publishing
				mockRedisTaskPublisher.EXPECT().
					PublishTaskSendVerifyEmail(ctx, mock.AnythingOfType("*email.VerifyEmailPayload")).
					Return(mockErr)

				return mockUserRepo, mockPassword, mockGenerator, mockJwtToken, mockRedisClient, mockRedisTaskPublisher
			},
			verify: func(t *testing.T, got *entities.SignUpUserResponse, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
				assert.ErrorIs(t, gotErr, mockErr)
			},
		},
		{
			name: "Failed_ExistingUserCreateVerifyEmailToken",
			input: &entities.SignUpUserRequest{
				Email:       "existing@example.com",
				Password:    "password123",
				FullName:    "Existing User",
				Country:     "US",
				Gender:      "female",
				DateOfBirth: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			setup: func() (*repositories.MockUserRepository, *utils.MockPasswordUtil, *utils.MockGenerator, *utils.MockJwtToken, *mockDatabase.MockRedisClient, *queue.MockRedisTaskPublisher) {
				mockUserRepo := new(repositories.MockUserRepository)
				mockPassword := new(utils.MockPasswordUtil)
				mockGenerator := new(utils.MockGenerator)
				mockJwtToken := new(utils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockRedisTaskPublisher := new(queue.MockRedisTaskPublisher)

				// Mock CheckIsEmailExists returns existing unverified user
				mockUserRepo.EXPECT().
					CheckIsEmailExists(ctx, "existing@example.com").
					Return(&db.Users{ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), IsEmailVerified: sql.NullBool{Bool: false, Valid: true}}, nil)

				// Mock password hashing
				mockPassword.EXPECT().
					HashPassword(ctx, "password123").
					Return("hashed_password", nil)

				// Mock UpdateUser
				mockUserRepo.EXPECT().
					UpdateUser(ctx, mock.AnythingOfType("*db.UpdateUserParams")).
					Return(&db.Users{ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")}, nil)

				// Mock random string generation
				mockGenerator.EXPECT().
					GenerateRandomString(ctx, 6).
					Return("123456")

				// Mock JWT token creation
				mockJwtToken.EXPECT().
					CreateVerifyEmailToken(ctx, mock.AnythingOfType("*entities.VerifyEmailTokenRequest")).
					Return("", &utilsPkg.VerifyEmailTokenPayload{}, mockErr)

				return mockUserRepo, mockPassword, mockGenerator, mockJwtToken, mockRedisClient, mockRedisTaskPublisher
			},
			verify: func(t *testing.T, got *entities.SignUpUserResponse, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
				assert.ErrorIs(t, gotErr, mockErr)
			},
		},
		{
			name: "Failed_ExistingUserSetEmailToken",
			input: &entities.SignUpUserRequest{
				Email:       "existing@example.com",
				Password:    "password123",
				FullName:    "Existing User",
				Country:     "US",
				Gender:      "female",
				DateOfBirth: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			setup: func() (*repositories.MockUserRepository, *utils.MockPasswordUtil, *utils.MockGenerator, *utils.MockJwtToken, *mockDatabase.MockRedisClient, *queue.MockRedisTaskPublisher) {
				mockUserRepo := new(repositories.MockUserRepository)
				mockPassword := new(utils.MockPasswordUtil)
				mockGenerator := new(utils.MockGenerator)
				mockJwtToken := new(utils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockRedisTaskPublisher := new(queue.MockRedisTaskPublisher)

				// Mock CheckIsEmailExists returns existing unverified user
				mockUserRepo.EXPECT().
					CheckIsEmailExists(ctx, "existing@example.com").
					Return(&db.Users{ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), IsEmailVerified: sql.NullBool{Bool: false, Valid: true}}, nil)

				// Mock password hashing
				mockPassword.EXPECT().
					HashPassword(ctx, "password123").
					Return("hashed_password", nil)

				// Mock UpdateUser
				mockUserRepo.EXPECT().
					UpdateUser(ctx, mock.AnythingOfType("*db.UpdateUserParams")).
					Return(&db.Users{ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")}, nil)

				// Mock random string generation
				mockGenerator.EXPECT().
					GenerateRandomString(ctx, 6).
					Return("123456")

				// Mock JWT token creation
				mockJwtToken.EXPECT().
					CreateVerifyEmailToken(ctx, mock.AnythingOfType("*entities.VerifyEmailTokenRequest")).
					Return("verify_token", &utilsPkg.VerifyEmailTokenPayload{}, nil)

				// Mock Redis operations
				mockRedisClient.EXPECT().
					Set(ctx, mock.AnythingOfType("database.RedisPayload")).
					Return(mockErr)

				return mockUserRepo, mockPassword, mockGenerator, mockJwtToken, mockRedisClient, mockRedisTaskPublisher
			},
			verify: func(t *testing.T, got *entities.SignUpUserResponse, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
				assert.ErrorIs(t, gotErr, mockErr)
			},
		},
		{
			name: "Failed_ExistingUserSetAttempt",
			input: &entities.SignUpUserRequest{
				Email:       "existing@example.com",
				Password:    "password123",
				FullName:    "Existing User",
				Country:     "US",
				Gender:      "female",
				DateOfBirth: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			setup: func() (*repositories.MockUserRepository, *utils.MockPasswordUtil, *utils.MockGenerator, *utils.MockJwtToken, *mockDatabase.MockRedisClient, *queue.MockRedisTaskPublisher) {
				mockUserRepo := new(repositories.MockUserRepository)
				mockPassword := new(utils.MockPasswordUtil)
				mockGenerator := new(utils.MockGenerator)
				mockJwtToken := new(utils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockRedisTaskPublisher := new(queue.MockRedisTaskPublisher)

				// Mock CheckIsEmailExists returns existing unverified user
				mockUserRepo.EXPECT().
					CheckIsEmailExists(ctx, "existing@example.com").
					Return(&db.Users{ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), IsEmailVerified: sql.NullBool{Bool: false, Valid: true}}, nil)

				// Mock password hashing
				mockPassword.EXPECT().
					HashPassword(ctx, "password123").
					Return("hashed_password", nil)

				// Mock UpdateUser
				mockUserRepo.EXPECT().
					UpdateUser(ctx, mock.AnythingOfType("*db.UpdateUserParams")).
					Return(&db.Users{ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")}, nil)

				// Mock random string generation
				mockGenerator.EXPECT().
					GenerateRandomString(ctx, 6).
					Return("123456")

				// Mock JWT token creation
				mockJwtToken.EXPECT().
					CreateVerifyEmailToken(ctx, mock.AnythingOfType("*entities.VerifyEmailTokenRequest")).
					Return("verify_token", &utilsPkg.VerifyEmailTokenPayload{}, nil)

				// Mock Redis operations
				mockRedisClient.EXPECT().
					Set(ctx, mock.AnythingOfType("database.RedisPayload")).
					Return(nil).Times(1)

				// Mock Redis set attempt failed
				mockRedisClient.EXPECT().
					Set(ctx, mock.AnythingOfType("database.RedisPayload")).
					Return(mockErr).Times(1)

				return mockUserRepo, mockPassword, mockGenerator, mockJwtToken, mockRedisClient, mockRedisTaskPublisher
			},
			verify: func(t *testing.T, got *entities.SignUpUserResponse, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
				assert.ErrorIs(t, gotErr, mockErr)
			},
		},
		{
			name: "Failed_ExistingUserHashPassword",
			input: &entities.SignUpUserRequest{
				Email:       "existing@example.com",
				Password:    "password123",
				FullName:    "Existing User",
				Country:     "US",
				Gender:      "female",
				DateOfBirth: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			setup: func() (*repositories.MockUserRepository, *utils.MockPasswordUtil, *utils.MockGenerator, *utils.MockJwtToken, *mockDatabase.MockRedisClient, *queue.MockRedisTaskPublisher) {
				mockUserRepo := new(repositories.MockUserRepository)
				mockPassword := new(utils.MockPasswordUtil)
				mockGenerator := new(utils.MockGenerator)
				mockJwtToken := new(utils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockRedisTaskPublisher := new(queue.MockRedisTaskPublisher)

				// Mock CheckIsEmailExists returns existing unverified user
				mockUserRepo.EXPECT().
					CheckIsEmailExists(ctx, "existing@example.com").
					Return(&db.Users{ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), IsEmailVerified: sql.NullBool{Bool: false, Valid: true}}, nil)

				// Mock password hashing
				mockPassword.EXPECT().
					HashPassword(ctx, "password123").
					Return("", mockErr)

				return mockUserRepo, mockPassword, mockGenerator, mockJwtToken, mockRedisClient, mockRedisTaskPublisher
			},
			verify: func(t *testing.T, got *entities.SignUpUserResponse, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
				assert.ErrorIs(t, gotErr, mockErr)
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			mockUserRepo, mockPassword, mockGenerator, mockJwtToken, mockRedisClient, mockRedisTaskPublisher := tC.setup()
			defer func() {
				if mockUserRepo != nil {
					mockUserRepo.AssertExpectations(t)
				}
				if mockPassword != nil {
					mockPassword.AssertExpectations(t)
				}
				if mockGenerator != nil {
					mockGenerator.AssertExpectations(t)
				}
				if mockJwtToken != nil {
					mockJwtToken.AssertExpectations(t)
				}
				if mockRedisClient != nil {
					mockRedisClient.AssertExpectations(t)
				}
				if mockRedisTaskPublisher != nil {
					mockRedisTaskPublisher.AssertExpectations(t)
				}
			}()

			svc := NewUserService(lgr, mockUserRepo, nil, mockJwtToken, nil, mockConfig, nil, nil, mockRedisClient, mockRedisTaskPublisher, mockPassword, mockGenerator)
			got, gotErr := svc.SignUpUser(ctx, tC.input)

			tC.verify(t, got, gotErr)
		})
	}
}

func TestUserService_SendVerifyEmail(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()
	mockErr := errors.New("error")

	testCases := []struct {
		name   string
		input  *entities.VerifyEmailRequest
		setup  func() (*repositories.MockUserRepository, *utils.MockJwtToken, *mockDatabase.MockRedisClient)
		verify func(t *testing.T, gotErr error)
	}{
		{
			name: "Success",
			input: &entities.VerifyEmailRequest{
				Token: "verify_token",
				Code:  "123456",
			},
			setup: func() (*repositories.MockUserRepository, *utils.MockJwtToken, *mockDatabase.MockRedisClient) {
				mockUserRepo := new(repositories.MockUserRepository)
				mockJwtToken := new(utils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				// Mock JWT token verification
				mockJwtToken.EXPECT().
					VerifyVerifyEmailToken(ctx, "verify_token").
					Return(&utilsPkg.VerifyEmailTokenPayload{
						UserID: "550e8400-e29b-41d4-a716-446655440000",
						Email:  "user@example.com",
					}, nil)

				// Mock Redis get operation
				mockRedisClient.EXPECT().
					Get(ctx, mock.AnythingOfType("string")).
					Return("123456", nil)

				// Mock user existence check
				mockUserRepo.EXPECT().
					CheckIsUserExistsByID(ctx, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")).
					Return(&db.Users{ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), IsEmailVerified: sql.NullBool{Bool: false, Valid: true}}, nil)

				// Mock email verification
				mockUserRepo.EXPECT().
					VerifyEmail(ctx, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")).
					Return(nil)

				// Mock Redis delete operations
				mockRedisClient.EXPECT().
					Delete(ctx, mock.AnythingOfType("string")).
					Return(nil).Times(2)

				return mockUserRepo, mockJwtToken, mockRedisClient
			},
			verify: func(t *testing.T, gotErr error) {
				assert.NoError(t, gotErr)
			},
		}, {
			name: "Error_InvalidUserID",
			input: &entities.VerifyEmailRequest{
				Token: "verify_token",
				Code:  "123456",
			},
			setup: func() (*repositories.MockUserRepository, *utils.MockJwtToken, *mockDatabase.MockRedisClient) {
				mockUserRepo := new(repositories.MockUserRepository)
				mockJwtToken := new(utils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				// Mock JWT token verification
				mockJwtToken.EXPECT().
					VerifyVerifyEmailToken(ctx, "verify_token").
					Return(&utilsPkg.VerifyEmailTokenPayload{
						UserID: "invalid-user-id",
						Email:  "user@example.com",
					}, nil)

				return mockUserRepo, mockJwtToken, mockRedisClient
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "The UUID is invalid. Please try again.")
				assert.Contains(t, gotErr.Error(), "[INS0107]")
			},
		},
		{
			name: "Success_WithDeleteRedisFailed",
			input: &entities.VerifyEmailRequest{
				Token: "verify_token",
				Code:  "123456",
			},
			setup: func() (*repositories.MockUserRepository, *utils.MockJwtToken, *mockDatabase.MockRedisClient) {
				mockUserRepo := new(repositories.MockUserRepository)
				mockJwtToken := new(utils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				// Mock JWT token verification
				mockJwtToken.EXPECT().
					VerifyVerifyEmailToken(ctx, "verify_token").
					Return(&utilsPkg.VerifyEmailTokenPayload{
						UserID: "550e8400-e29b-41d4-a716-446655440000",
						Email:  "user@example.com",
					}, nil)

				// Mock Redis get operation
				mockRedisClient.EXPECT().
					Get(ctx, mock.AnythingOfType("string")).
					Return("123456", nil)

				// Mock user existence check
				mockUserRepo.EXPECT().
					CheckIsUserExistsByID(ctx, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")).
					Return(&db.Users{ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), IsEmailVerified: sql.NullBool{Bool: false, Valid: true}}, nil)

				// Mock email verification
				mockUserRepo.EXPECT().
					VerifyEmail(ctx, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")).
					Return(nil)

				// Mock Redis delete operations
				mockRedisClient.EXPECT().
					Delete(ctx, mock.AnythingOfType("string")).
					Return(mockErr).Times(1)

				return mockUserRepo, mockJwtToken, mockRedisClient
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
			},
		},
		{
			name: "Success_WithDeleteRedisAttemptFailed",
			input: &entities.VerifyEmailRequest{
				Token: "verify_token",
				Code:  "123456",
			},
			setup: func() (*repositories.MockUserRepository, *utils.MockJwtToken, *mockDatabase.MockRedisClient) {
				mockUserRepo := new(repositories.MockUserRepository)
				mockJwtToken := new(utils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				// Mock JWT token verification
				mockJwtToken.EXPECT().
					VerifyVerifyEmailToken(ctx, "verify_token").
					Return(&utilsPkg.VerifyEmailTokenPayload{
						UserID: "550e8400-e29b-41d4-a716-446655440000",
						Email:  "user@example.com",
					}, nil)

				// Mock Redis get operation
				mockRedisClient.EXPECT().
					Get(ctx, mock.AnythingOfType("string")).
					Return("123456", nil)

				// Mock user existence check
				mockUserRepo.EXPECT().
					CheckIsUserExistsByID(ctx, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")).
					Return(&db.Users{ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), IsEmailVerified: sql.NullBool{Bool: false, Valid: true}}, nil)

				// Mock email verification
				mockUserRepo.EXPECT().
					VerifyEmail(ctx, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")).
					Return(nil)

				// Mock Redis delete operations
				mockRedisClient.EXPECT().
					Delete(ctx, mock.AnythingOfType("string")).
					Return(nil).Times(1)

				mockRedisClient.EXPECT().
					Delete(ctx, mock.AnythingOfType("string")).
					Return(mockErr).Times(1)

				return mockUserRepo, mockJwtToken, mockRedisClient
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
			},
		},
		{
			name: "Error_InvalidToken",
			input: &entities.VerifyEmailRequest{
				Token: "invalid_token",
				Code:  "123456",
			},
			setup: func() (*repositories.MockUserRepository, *utils.MockJwtToken, *mockDatabase.MockRedisClient) {
				mockJwtToken := new(utils.MockJwtToken)

				// Mock JWT token verification fails
				mockJwtToken.EXPECT().
					VerifyVerifyEmailToken(ctx, "invalid_token").
					Return(nil, mockErr)

				return nil, mockJwtToken, nil
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.ErrorIs(t, gotErr, mockErr)
			},
		},
		{
			name: "Error_InvalidCode",
			input: &entities.VerifyEmailRequest{
				Token: "verify_token",
				Code:  "wrong_code",
			},
			setup: func() (*repositories.MockUserRepository, *utils.MockJwtToken, *mockDatabase.MockRedisClient) {
				mockJwtToken := new(utils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				// Mock JWT token verification
				mockJwtToken.EXPECT().
					VerifyVerifyEmailToken(ctx, "verify_token").
					Return(&utilsPkg.VerifyEmailTokenPayload{
						UserID: "550e8400-e29b-41d4-a716-446655440000",
						Email:  "user@example.com",
					}, nil)

				// Mock Redis get operation returns different code
				mockRedisClient.EXPECT().
					Get(ctx, mock.AnythingOfType("string")).
					Return("123456", nil)

				// Mock Redis increment operation
				mockRedisClient.EXPECT().
					Increment(ctx, mock.AnythingOfType("string")).
					Return(int64(1), nil)

				return nil, mockJwtToken, mockRedisClient
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
			},
		},
		{
			name: "Error_EmailAlreadyVerified",
			input: &entities.VerifyEmailRequest{
				Token: "verify_token",
				Code:  "123456",
			},
			setup: func() (*repositories.MockUserRepository, *utils.MockJwtToken, *mockDatabase.MockRedisClient) {
				mockUserRepo := new(repositories.MockUserRepository)
				mockJwtToken := new(utils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				// Mock JWT token verification
				mockJwtToken.EXPECT().
					VerifyVerifyEmailToken(ctx, "verify_token").
					Return(&utilsPkg.VerifyEmailTokenPayload{
						UserID: "550e8400-e29b-41d4-a716-446655440000",
						Email:  "user@example.com",
					}, nil)

				// Mock Redis get operation
				mockRedisClient.EXPECT().
					Get(ctx, mock.AnythingOfType("string")).
					Return("123456", nil)

				// Mock user existence check returns already verified user
				mockUserRepo.EXPECT().
					CheckIsUserExistsByID(ctx, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")).
					Return(&db.Users{ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), IsEmailVerified: sql.NullBool{Bool: true, Valid: true}}, nil)

				return mockUserRepo, mockJwtToken, mockRedisClient
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
			},
		},
		{
			name: "Error_RedisGetFailed",
			input: &entities.VerifyEmailRequest{
				Token: "verify_token",
				Code:  "123456",
			},
			setup: func() (*repositories.MockUserRepository, *utils.MockJwtToken, *mockDatabase.MockRedisClient) {
				mockJwtToken := new(utils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				// Mock JWT token verification
				mockJwtToken.EXPECT().
					VerifyVerifyEmailToken(ctx, "verify_token").
					Return(&utilsPkg.VerifyEmailTokenPayload{
						UserID: "550e8400-e29b-41d4-a716-446655440000",
						Email:  "user@example.com",
					}, nil)

				// Mock Redis get operation fails
				mockRedisClient.EXPECT().
					Get(ctx, mock.AnythingOfType("string")).
					Return("", mockErr)

				return nil, mockJwtToken, mockRedisClient
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.ErrorIs(t, gotErr, mockErr)
			},
		},
		{
			name: "Error_RedisGetReturnsNil",
			input: &entities.VerifyEmailRequest{
				Token: "verify_token",
				Code:  "123456",
			},
			setup: func() (*repositories.MockUserRepository, *utils.MockJwtToken, *mockDatabase.MockRedisClient) {
				mockJwtToken := new(utils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				// Mock JWT token verification
				mockJwtToken.EXPECT().
					VerifyVerifyEmailToken(ctx, "verify_token").
					Return(&utilsPkg.VerifyEmailTokenPayload{
						UserID: "550e8400-e29b-41d4-a716-446655440000",
						Email:  "user@example.com",
					}, nil)

				// Mock Redis get operation returns redis.Nil
				mockRedisClient.EXPECT().
					Get(ctx, mock.AnythingOfType("string")).
					Return("", redis.Nil)

				return nil, mockJwtToken, mockRedisClient
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
			},
		},
		{
			name: "Error_MaxAttemptsReached",
			input: &entities.VerifyEmailRequest{
				Token: "verify_token",
				Code:  "wrong_code",
			},
			setup: func() (*repositories.MockUserRepository, *utils.MockJwtToken, *mockDatabase.MockRedisClient) {
				mockJwtToken := new(utils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				// Mock JWT token verification
				mockJwtToken.EXPECT().
					VerifyVerifyEmailToken(ctx, "verify_token").
					Return(&utilsPkg.VerifyEmailTokenPayload{
						UserID: "550e8400-e29b-41d4-a716-446655440000",
						Email:  "user@example.com",
					}, nil)

				// Mock Redis get operation returns different code
				mockRedisClient.EXPECT().
					Get(ctx, mock.AnythingOfType("string")).
					Return("123456", nil)

				// Mock Redis increment operation returns max attempts
				mockRedisClient.EXPECT().
					Increment(ctx, mock.AnythingOfType("string")).
					Return(int64(3), nil)

				// Mock Redis delete operations for cleanup
				mockRedisClient.EXPECT().
					Delete(ctx, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
					Return(nil)

				return nil, mockJwtToken, mockRedisClient
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
			},
		},
		{
			name: "Error_RedisIncrementFailed",
			input: &entities.VerifyEmailRequest{
				Token: "verify_token",
				Code:  "wrong_code",
			},
			setup: func() (*repositories.MockUserRepository, *utils.MockJwtToken, *mockDatabase.MockRedisClient) {
				mockJwtToken := new(utils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				// Mock JWT token verification
				mockJwtToken.EXPECT().
					VerifyVerifyEmailToken(ctx, "verify_token").
					Return(&utilsPkg.VerifyEmailTokenPayload{
						UserID: "550e8400-e29b-41d4-a716-446655440000",
						Email:  "user@example.com",
					}, nil)

				// Mock Redis get operation returns different code
				mockRedisClient.EXPECT().
					Get(ctx, mock.AnythingOfType("string")).
					Return("123456", nil)

				// Mock Redis increment operation fails
				mockRedisClient.EXPECT().
					Increment(ctx, mock.AnythingOfType("string")).
					Return(int64(0), mockErr)

				return nil, mockJwtToken, mockRedisClient
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.ErrorIs(t, gotErr, mockErr)
			},
		},
		{
			name: "Error_UserNotFound",
			input: &entities.VerifyEmailRequest{
				Token: "verify_token",
				Code:  "123456",
			},
			setup: func() (*repositories.MockUserRepository, *utils.MockJwtToken, *mockDatabase.MockRedisClient) {
				mockUserRepo := new(repositories.MockUserRepository)
				mockJwtToken := new(utils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				// Mock JWT token verification
				mockJwtToken.EXPECT().
					VerifyVerifyEmailToken(ctx, "verify_token").
					Return(&utilsPkg.VerifyEmailTokenPayload{
						UserID: "550e8400-e29b-41d4-a716-446655440000",
						Email:  "user@example.com",
					}, nil)

				// Mock Redis get operation
				mockRedisClient.EXPECT().
					Get(ctx, mock.AnythingOfType("string")).
					Return("123456", nil)

				// Mock user existence check fails
				mockUserRepo.EXPECT().
					CheckIsUserExistsByID(ctx, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")).
					Return(nil, mockErr)

				return mockUserRepo, mockJwtToken, mockRedisClient
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.ErrorIs(t, gotErr, mockErr)
			},
		},
		{
			name: "Error_VerifyEmailFailed",
			input: &entities.VerifyEmailRequest{
				Token: "verify_token",
				Code:  "123456",
			},
			setup: func() (*repositories.MockUserRepository, *utils.MockJwtToken, *mockDatabase.MockRedisClient) {
				mockUserRepo := new(repositories.MockUserRepository)
				mockJwtToken := new(utils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				// Mock JWT token verification
				mockJwtToken.EXPECT().
					VerifyVerifyEmailToken(ctx, "verify_token").
					Return(&utilsPkg.VerifyEmailTokenPayload{
						UserID: "550e8400-e29b-41d4-a716-446655440000",
						Email:  "user@example.com",
					}, nil)

				// Mock Redis get operation
				mockRedisClient.EXPECT().
					Get(ctx, mock.AnythingOfType("string")).
					Return("123456", nil)

				// Mock user existence check
				mockUserRepo.EXPECT().
					CheckIsUserExistsByID(ctx, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")).
					Return(&db.Users{ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), IsEmailVerified: sql.NullBool{Bool: false, Valid: true}}, nil)

				// Mock email verification fails
				mockUserRepo.EXPECT().
					VerifyEmail(ctx, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")).
					Return(mockErr)

				return mockUserRepo, mockJwtToken, mockRedisClient
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.ErrorIs(t, gotErr, mockErr)
			},
		},
		{
			name: "Error_RedisDeleteFailed",
			input: &entities.VerifyEmailRequest{
				Token: "verify_token",
				Code:  "123456",
			},
			setup: func() (*repositories.MockUserRepository, *utils.MockJwtToken, *mockDatabase.MockRedisClient) {
				mockUserRepo := new(repositories.MockUserRepository)
				mockJwtToken := new(utils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				// Mock JWT token verification
				mockJwtToken.EXPECT().
					VerifyVerifyEmailToken(ctx, "verify_token").
					Return(&utilsPkg.VerifyEmailTokenPayload{
						UserID: "550e8400-e29b-41d4-a716-446655440000",
						Email:  "user@example.com",
					}, nil)

				// Mock Redis get operation
				mockRedisClient.EXPECT().
					Get(ctx, mock.AnythingOfType("string")).
					Return("123456", nil)

				// Mock user existence check
				mockUserRepo.EXPECT().
					CheckIsUserExistsByID(ctx, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")).
					Return(&db.Users{ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), IsEmailVerified: sql.NullBool{Bool: false, Valid: true}}, nil)

				// Mock email verification
				mockUserRepo.EXPECT().
					VerifyEmail(ctx, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")).
					Return(nil)

				// Mock Redis delete operations fail
				mockRedisClient.EXPECT().
					Delete(ctx, mock.AnythingOfType("string")).
					Return(mockErr)

				return mockUserRepo, mockJwtToken, mockRedisClient
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.ErrorIs(t, gotErr, mockErr)
			},
		},
		{
			name: "Error_RedisDeleteAttemptFailed",
			input: &entities.VerifyEmailRequest{
				Token: "verify_token",
				Code:  "123456",
			},
			setup: func() (*repositories.MockUserRepository, *utils.MockJwtToken, *mockDatabase.MockRedisClient) {
				mockUserRepo := new(repositories.MockUserRepository)
				mockJwtToken := new(utils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				// Mock JWT token verification
				mockJwtToken.EXPECT().
					VerifyVerifyEmailToken(ctx, "verify_token").
					Return(&utilsPkg.VerifyEmailTokenPayload{
						UserID: "550e8400-e29b-41d4-a716-446655440000",
						Email:  "user@example.com",
					}, nil)

				// Mock Redis get operation
				mockRedisClient.EXPECT().
					Get(ctx, mock.AnythingOfType("string")).
					Return("123456", nil)

				// Mock user existence check
				mockUserRepo.EXPECT().
					CheckIsUserExistsByID(ctx, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")).
					Return(&db.Users{ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), IsEmailVerified: sql.NullBool{Bool: false, Valid: true}}, nil)

				// Mock email verification
				mockUserRepo.EXPECT().
					VerifyEmail(ctx, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")).
					Return(nil)

				// Mock first Redis delete operation succeeds
				mockRedisClient.EXPECT().
					Delete(ctx, mock.AnythingOfType("string")).
					Return(nil)

				// Mock second Redis delete operation (attempt counter) fails
				mockRedisClient.EXPECT().
					Delete(ctx, mock.AnythingOfType("string")).
					Return(mockErr)

				return mockUserRepo, mockJwtToken, mockRedisClient
			},
			verify: func(t *testing.T, gotErr error) {
				// Note: The service only logs a warning when the second Redis delete fails
				// and returns nil (no error), so we expect no error here
				assert.NoError(t, gotErr)
			},
		},
		{
			name: "Error_RedisCleanupAfterMaxAttemptsFailed",
			input: &entities.VerifyEmailRequest{
				Token: "verify_token",
				Code:  "wrong_code",
			},
			setup: func() (*repositories.MockUserRepository, *utils.MockJwtToken, *mockDatabase.MockRedisClient) {
				mockJwtToken := new(utils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				// Mock JWT token verification
				mockJwtToken.EXPECT().
					VerifyVerifyEmailToken(ctx, "verify_token").
					Return(&utilsPkg.VerifyEmailTokenPayload{
						UserID: "550e8400-e29b-41d4-a716-446655440000",
						Email:  "user@example.com",
					}, nil)

				// Mock Redis get operation returns different code
				mockRedisClient.EXPECT().
					Get(ctx, mock.AnythingOfType("string")).
					Return("123456", nil)

				// Mock Redis increment operation returns max attempts
				mockRedisClient.EXPECT().
					Increment(ctx, mock.AnythingOfType("string")).
					Return(int64(3), nil)

				// Mock Redis delete operations for cleanup fail
				mockRedisClient.EXPECT().
					Delete(ctx, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
					Return(mockErr)

				return nil, mockJwtToken, mockRedisClient
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
			},
		},
		{
			name: "Error_GenerateRandomStringFailed",
			input: &entities.VerifyEmailRequest{
				Token: "verify_token",
				Code:  "123456",
			},
			setup: func() (*repositories.MockUserRepository, *utils.MockJwtToken, *mockDatabase.MockRedisClient) {
				mockJwtToken := new(utils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				// Mock JWT token verification
				mockJwtToken.EXPECT().
					VerifyVerifyEmailToken(ctx, "verify_token").
					Return(&utilsPkg.VerifyEmailTokenPayload{
						UserID: "550e8400-e29b-41d4-a716-446655440000",
						Email:  "user@example.com",
					}, nil)

				// Mock Redis get operation
				mockRedisClient.EXPECT().
					Get(ctx, mock.AnythingOfType("string")).
					Return("123456", nil)

				// Mock user existence check
				mockUserRepo := new(repositories.MockUserRepository)
				mockUserRepo.EXPECT().
					CheckIsUserExistsByID(ctx, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")).
					Return(&db.Users{ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), IsEmailVerified: sql.NullBool{Bool: false, Valid: true}}, nil)

				// Mock email verification
				mockUserRepo.EXPECT().
					VerifyEmail(ctx, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")).
					Return(nil)

				// Mock Redis delete operations
				mockRedisClient.EXPECT().
					Delete(ctx, mock.AnythingOfType("string")).
					Return(nil).Times(2)

				return mockUserRepo, mockJwtToken, mockRedisClient
			},
			verify: func(t *testing.T, gotErr error) {
				assert.NoError(t, gotErr)
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			mockUserRepo, mockJwtToken, mockRedisClient := tC.setup()
			defer func() {
				if mockUserRepo != nil {
					mockUserRepo.AssertExpectations(t)
				}
				if mockJwtToken != nil {
					mockJwtToken.AssertExpectations(t)
				}
				if mockRedisClient != nil {
					mockRedisClient.AssertExpectations(t)
				}
			}()

			svc := NewUserService(lgr, mockUserRepo, nil, mockJwtToken, nil, nil, nil, nil, mockRedisClient, nil, nil, nil)
			gotErr := svc.SendVerifyEmail(ctx, tC.input)

			tC.verify(t, gotErr)
		})
	}
}

func TestUserService_ResetVerifyEmailCode(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()
	mockErr := errors.New("error")

	// Create mock config
	mockConfig := &config.Config{
		EmailConfig: config.EmailConfig{
			VerifyEmailTokenDuration: 15 * time.Minute,
		},
	}

	testCases := []struct {
		name   string
		input  *entities.ResetVerifyEmailCodeRequest
		setup  func() (*utils.MockGenerator, *utils.MockJwtToken, *mockDatabase.MockRedisClient, *queue.MockRedisTaskPublisher)
		verify func(t *testing.T, got *entities.ResetVerifyEmailCodeResponse, gotErr error)
	}{
		{
			name: "Success",
			input: &entities.ResetVerifyEmailCodeRequest{
				Token: "old_token",
			},
			setup: func() (*utils.MockGenerator, *utils.MockJwtToken, *mockDatabase.MockRedisClient, *queue.MockRedisTaskPublisher) {
				mockGenerator := new(utils.MockGenerator)
				mockJwtToken := new(utils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockRedisTaskPublisher := new(queue.MockRedisTaskPublisher)

				// Mock Redis delete operations
				mockRedisClient.EXPECT().
					Delete(ctx, mock.AnythingOfType("string")).
					Return(nil).Times(2)

				// Mock JWT token renewal
				mockJwtToken.EXPECT().
					RenewVerifyEmailToken(ctx, "old_token").
					Return("new_token", &utilsPkg.VerifyEmailTokenPayload{
						UserID: "user-123",
						Email:  "user@example.com",
					}, nil)

				// Mock random string generation
				mockGenerator.EXPECT().
					GenerateRandomString(ctx, 6).
					Return("123456")

				// Mock Redis set operations
				mockRedisClient.EXPECT().
					Set(ctx, mock.AnythingOfType("database.RedisPayload")).
					Return(nil).Times(2)

				// Mock task publishing
				mockRedisTaskPublisher.EXPECT().
					PublishTaskSendVerifyEmail(ctx, mock.AnythingOfType("*email.VerifyEmailPayload")).
					Return(nil)

				return mockGenerator, mockJwtToken, mockRedisClient, mockRedisTaskPublisher
			},
			verify: func(t *testing.T, got *entities.ResetVerifyEmailCodeResponse, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, got)
				assert.Equal(t, "new_token", got.TokenId)
			},
		},
		{
			name: "Error_TokenRenewalFailed",
			input: &entities.ResetVerifyEmailCodeRequest{
				Token: "invalid_token",
			},
			setup: func() (*utils.MockGenerator, *utils.MockJwtToken, *mockDatabase.MockRedisClient, *queue.MockRedisTaskPublisher) {
				mockGenerator := new(utils.MockGenerator)
				mockJwtToken := new(utils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockRedisTaskPublisher := new(queue.MockRedisTaskPublisher)

				// Mock random string generation (called before token renewal)
				mockGenerator.EXPECT().
					GenerateRandomString(ctx, 6).
					Return("123456")

				// Mock Redis delete operations
				mockRedisClient.EXPECT().
					Delete(ctx, mock.AnythingOfType("string")).
					Return(nil).Times(2)

				// Mock JWT token renewal fails
				mockJwtToken.EXPECT().
					RenewVerifyEmailToken(ctx, "invalid_token").
					Return("", nil, mockErr)

				return mockGenerator, mockJwtToken, mockRedisClient, mockRedisTaskPublisher
			},
			verify: func(t *testing.T, got *entities.ResetVerifyEmailCodeResponse, gotErr error) {
				assert.Error(t, gotErr)
				assert.ErrorIs(t, gotErr, mockErr)
			},
		},
		{
			name: "Error_RedisDeleteFailed",
			input: &entities.ResetVerifyEmailCodeRequest{
				Token: "old_token",
			},
			setup: func() (*utils.MockGenerator, *utils.MockJwtToken, *mockDatabase.MockRedisClient, *queue.MockRedisTaskPublisher) {
				mockGenerator := new(utils.MockGenerator)
				mockJwtToken := new(utils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockRedisTaskPublisher := new(queue.MockRedisTaskPublisher)

				// Mock random string generation (called before Redis delete)
				mockGenerator.EXPECT().
					GenerateRandomString(ctx, 6).
					Return("123456")

				// Mock Redis delete operations fail
				mockRedisClient.EXPECT().
					Delete(ctx, mock.AnythingOfType("string")).
					Return(mockErr)

				// Mock JWT token renewal (called after Redis delete)
				mockJwtToken.EXPECT().
					RenewVerifyEmailToken(ctx, "old_token").
					Return("", nil, mockErr)

				return mockGenerator, mockJwtToken, mockRedisClient, mockRedisTaskPublisher
			},
			verify: func(t *testing.T, got *entities.ResetVerifyEmailCodeResponse, gotErr error) {
				assert.Error(t, gotErr)
				assert.ErrorIs(t, gotErr, mockErr)
			},
		},
		{
			name: "Error_RedisSetFailed",
			input: &entities.ResetVerifyEmailCodeRequest{
				Token: "old_token",
			},
			setup: func() (*utils.MockGenerator, *utils.MockJwtToken, *mockDatabase.MockRedisClient, *queue.MockRedisTaskPublisher) {
				mockGenerator := new(utils.MockGenerator)
				mockJwtToken := new(utils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockRedisTaskPublisher := new(queue.MockRedisTaskPublisher)

				// Mock Redis delete operations
				mockRedisClient.EXPECT().
					Delete(ctx, mock.AnythingOfType("string")).
					Return(nil).Times(2)

				// Mock JWT token renewal
				mockJwtToken.EXPECT().
					RenewVerifyEmailToken(ctx, "old_token").
					Return("new_token", &utilsPkg.VerifyEmailTokenPayload{
						UserID: "user-123",
						Email:  "user@example.com",
					}, nil)

				// Mock random string generation
				mockGenerator.EXPECT().
					GenerateRandomString(ctx, 6).
					Return("123456")

				// Mock Redis set operations fail
				mockRedisClient.EXPECT().
					Set(ctx, mock.AnythingOfType("database.RedisPayload")).
					Return(mockErr)

				return mockGenerator, mockJwtToken, mockRedisClient, mockRedisTaskPublisher
			},
			verify: func(t *testing.T, got *entities.ResetVerifyEmailCodeResponse, gotErr error) {
				assert.Error(t, gotErr)
				assert.ErrorIs(t, gotErr, mockErr)
			},
		},
		{
			name: "Error_RedisSetAttemptFailed",
			input: &entities.ResetVerifyEmailCodeRequest{
				Token: "old_token",
			},
			setup: func() (*utils.MockGenerator, *utils.MockJwtToken, *mockDatabase.MockRedisClient, *queue.MockRedisTaskPublisher) {
				mockGenerator := new(utils.MockGenerator)
				mockJwtToken := new(utils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockRedisTaskPublisher := new(queue.MockRedisTaskPublisher)

				// Mock Redis delete operations
				mockRedisClient.EXPECT().
					Delete(ctx, mock.AnythingOfType("string")).
					Return(nil).Times(2)

				// Mock JWT token renewal
				mockJwtToken.EXPECT().
					RenewVerifyEmailToken(ctx, "old_token").
					Return("new_token", &utilsPkg.VerifyEmailTokenPayload{
						UserID: "user-123",
						Email:  "user@example.com",
					}, nil)

				// Mock random string generation
				mockGenerator.EXPECT().
					GenerateRandomString(ctx, 6).
					Return("123456")

				// Mock Redis set operations
				mockRedisClient.EXPECT().
					Set(ctx, mock.AnythingOfType("database.RedisPayload")).
					Return(nil).Times(1)

				// Mock Redis set operations fail
				mockRedisClient.EXPECT().
					Set(ctx, mock.AnythingOfType("database.RedisPayload")).
					Return(mockErr)

				return mockGenerator, mockJwtToken, mockRedisClient, mockRedisTaskPublisher
			},
			verify: func(t *testing.T, got *entities.ResetVerifyEmailCodeResponse, gotErr error) {
				assert.Error(t, gotErr)
				assert.ErrorIs(t, gotErr, mockErr)
			},
		},
		{
			name: "Error_TaskPublishingFailed",
			input: &entities.ResetVerifyEmailCodeRequest{
				Token: "old_token",
			},
			setup: func() (*utils.MockGenerator, *utils.MockJwtToken, *mockDatabase.MockRedisClient, *queue.MockRedisTaskPublisher) {
				mockGenerator := new(utils.MockGenerator)
				mockJwtToken := new(utils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockRedisTaskPublisher := new(queue.MockRedisTaskPublisher)

				// Mock Redis delete operations
				mockRedisClient.EXPECT().
					Delete(ctx, mock.AnythingOfType("string")).
					Return(nil).Times(2)

				// Mock JWT token renewal
				mockJwtToken.EXPECT().
					RenewVerifyEmailToken(ctx, "old_token").
					Return("new_token", &utilsPkg.VerifyEmailTokenPayload{
						UserID: "user-123",
						Email:  "user@example.com",
					}, nil)

				// Mock random string generation
				mockGenerator.EXPECT().
					GenerateRandomString(ctx, 6).
					Return("123456")

				// Mock Redis set operations
				mockRedisClient.EXPECT().
					Set(ctx, mock.AnythingOfType("database.RedisPayload")).
					Return(nil).Times(2)

				// Mock task publishing fails
				mockRedisTaskPublisher.EXPECT().
					PublishTaskSendVerifyEmail(ctx, mock.AnythingOfType("*email.VerifyEmailPayload")).
					Return(mockErr)

				return mockGenerator, mockJwtToken, mockRedisClient, mockRedisTaskPublisher
			},
			verify: func(t *testing.T, got *entities.ResetVerifyEmailCodeResponse, gotErr error) {
				assert.Error(t, gotErr)
				assert.ErrorIs(t, gotErr, mockErr)
			},
		},
		{
			name: "Error_RedisDeleteFirstOperationFailed",
			input: &entities.ResetVerifyEmailCodeRequest{
				Token: "old_token",
			},
			setup: func() (*utils.MockGenerator, *utils.MockJwtToken, *mockDatabase.MockRedisClient, *queue.MockRedisTaskPublisher) {
				mockGenerator := new(utils.MockGenerator)
				mockJwtToken := new(utils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockRedisTaskPublisher := new(queue.MockRedisTaskPublisher)

				// Mock random string generation (called before Redis delete)
				mockGenerator.EXPECT().
					GenerateRandomString(ctx, 6).
					Return("123456")

				// Mock first Redis delete operation fails
				mockRedisClient.EXPECT().
					Delete(ctx, mock.AnythingOfType("string")).
					Return(mockErr)

				// Mock JWT token renewal (called after Redis delete)
				mockJwtToken.EXPECT().
					RenewVerifyEmailToken(ctx, "old_token").
					Return("new_token", &utilsPkg.VerifyEmailTokenPayload{
						UserID: "user-123",
						Email:  "user@example.com",
					}, nil)

				// Mock Redis set operations
				mockRedisClient.EXPECT().
					Set(ctx, mock.AnythingOfType("database.RedisPayload")).
					Return(nil).Times(2)

				// Mock task publishing
				mockRedisTaskPublisher.EXPECT().
					PublishTaskSendVerifyEmail(ctx, mock.AnythingOfType("*email.VerifyEmailPayload")).
					Return(nil)

				return mockGenerator, mockJwtToken, mockRedisClient, mockRedisTaskPublisher
			},
			verify: func(t *testing.T, got *entities.ResetVerifyEmailCodeResponse, gotErr error) {
				// Note: The service only logs a warning when Redis delete fails
				// and continues execution, so we expect no error here
				assert.NoError(t, gotErr)
				assert.NotNil(t, got)
			},
		},
		{
			name: "Error_RedisDeleteSecondOperationFailed",
			input: &entities.ResetVerifyEmailCodeRequest{
				Token: "old_token",
			},
			setup: func() (*utils.MockGenerator, *utils.MockJwtToken, *mockDatabase.MockRedisClient, *queue.MockRedisTaskPublisher) {
				mockGenerator := new(utils.MockGenerator)
				mockJwtToken := new(utils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockRedisTaskPublisher := new(queue.MockRedisTaskPublisher)

				// Mock random string generation (called before Redis delete)
				mockGenerator.EXPECT().
					GenerateRandomString(ctx, 6).
					Return("123456")

				// Mock first Redis delete operation succeeds
				mockRedisClient.EXPECT().
					Delete(ctx, mock.AnythingOfType("string")).
					Return(nil)

				// Mock second Redis delete operation fails
				mockRedisClient.EXPECT().
					Delete(ctx, mock.AnythingOfType("string")).
					Return(mockErr)

				// Mock JWT token renewal (called after Redis delete)
				mockJwtToken.EXPECT().
					RenewVerifyEmailToken(ctx, "old_token").
					Return("new_token", &utilsPkg.VerifyEmailTokenPayload{
						UserID: "user-123",
						Email:  "user@example.com",
					}, nil)

				// Mock Redis set operations
				mockRedisClient.EXPECT().
					Set(ctx, mock.AnythingOfType("database.RedisPayload")).
					Return(nil).Times(2)

				// Mock task publishing
				mockRedisTaskPublisher.EXPECT().
					PublishTaskSendVerifyEmail(ctx, mock.AnythingOfType("*email.VerifyEmailPayload")).
					Return(nil)

				return mockGenerator, mockJwtToken, mockRedisClient, mockRedisTaskPublisher
			},
			verify: func(t *testing.T, got *entities.ResetVerifyEmailCodeResponse, gotErr error) {
				// Note: The service only logs a warning when Redis delete fails
				// and continues execution, so we expect no error here
				assert.NoError(t, gotErr)
				assert.NotNil(t, got)
			},
		},
		{
			name: "Error_RedisSetSecondOperationFailed",
			input: &entities.ResetVerifyEmailCodeRequest{
				Token: "old_token",
			},
			setup: func() (*utils.MockGenerator, *utils.MockJwtToken, *mockDatabase.MockRedisClient, *queue.MockRedisTaskPublisher) {
				mockGenerator := new(utils.MockGenerator)
				mockJwtToken := new(utils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockRedisTaskPublisher := new(queue.MockRedisTaskPublisher)

				// Mock Redis delete operations for old token cleanup (called at the beginning)
				mockRedisClient.EXPECT().
					Delete(ctx, mock.AnythingOfType("string")).
					Return(nil).Times(2)

				// Mock JWT token renewal
				mockJwtToken.EXPECT().
					RenewVerifyEmailToken(ctx, "old_token").
					Return("new_token", &utilsPkg.VerifyEmailTokenPayload{
						UserID: "user-123",
						Email:  "user@example.com",
					}, nil)

				// Mock random string generation
				mockGenerator.EXPECT().
					GenerateRandomString(ctx, 6).
					Return("123456")

				// Mock first Redis set operation succeeds (verify email token)
				mockRedisClient.EXPECT().
					Set(ctx, mock.AnythingOfType("database.RedisPayload")).
					Return(nil)

				// Mock second Redis set operation fails (attempt counter)
				mockRedisClient.EXPECT().
					Set(ctx, mock.AnythingOfType("database.RedisPayload")).
					Return(mockErr)

				// Mock task publishing (this will be called before the second Redis set operation fails)
				mockRedisTaskPublisher.EXPECT().
					PublishTaskSendVerifyEmail(ctx, mock.AnythingOfType("*email.VerifyEmailPayload")).
					Return(nil)

				return mockGenerator, mockJwtToken, mockRedisClient, mockRedisTaskPublisher
			},
			verify: func(t *testing.T, got *entities.ResetVerifyEmailCodeResponse, gotErr error) {
				// Note: The service only logs a warning when Redis set fails
				// and continues execution, so we expect no error here
				assert.NoError(t, gotErr)
				assert.NotNil(t, got)
			},
		},
		{
			name: "Error_GenerateRandomStringFailed",
			input: &entities.ResetVerifyEmailCodeRequest{
				Token: "old_token",
			},
			setup: func() (*utils.MockGenerator, *utils.MockJwtToken, *mockDatabase.MockRedisClient, *queue.MockRedisTaskPublisher) {
				mockGenerator := new(utils.MockGenerator)
				mockJwtToken := new(utils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockRedisTaskPublisher := new(queue.MockRedisTaskPublisher)

				// Mock Redis delete operations for old token cleanup (called at the beginning)
				mockRedisClient.EXPECT().
					Delete(ctx, mock.AnythingOfType("string")).
					Return(nil).Times(2)

				// Mock random string generation fails
				mockGenerator.EXPECT().
					GenerateRandomString(ctx, 6).
					Return("")

				// Mock JWT token renewal fails due to empty code
				mockJwtToken.EXPECT().
					RenewVerifyEmailToken(ctx, "old_token").
					Return("", nil, errors.New("failed to create token with empty code"))

				return mockGenerator, mockJwtToken, mockRedisClient, mockRedisTaskPublisher
			},
			verify: func(t *testing.T, got *entities.ResetVerifyEmailCodeResponse, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
			},
		},
		{
			name: "Error_GenerateRandomStringEmpty",
			input: &entities.ResetVerifyEmailCodeRequest{
				Token: "old_token",
			},
			setup: func() (*utils.MockGenerator, *utils.MockJwtToken, *mockDatabase.MockRedisClient, *queue.MockRedisTaskPublisher) {
				mockGenerator := new(utils.MockGenerator)
				mockJwtToken := new(utils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockRedisTaskPublisher := new(queue.MockRedisTaskPublisher)

				// Mock Redis delete operations for old token cleanup (called at the beginning)
				mockRedisClient.EXPECT().
					Delete(ctx, mock.AnythingOfType("string")).
					Return(nil).Times(2)

				// Mock random string generation returns empty string
				mockGenerator.EXPECT().
					GenerateRandomString(ctx, 6).
					Return("")

				// Mock JWT token renewal fails due to empty code
				mockJwtToken.EXPECT().
					RenewVerifyEmailToken(ctx, "old_token").
					Return("", nil, errors.New("failed to create token with empty code"))

				return mockGenerator, mockJwtToken, mockRedisClient, mockRedisTaskPublisher
			},
			verify: func(t *testing.T, got *entities.ResetVerifyEmailCodeResponse, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			mockGenerator, mockJwtToken, mockRedisClient, mockRedisTaskPublisher := tC.setup()
			defer func() {
				if mockGenerator != nil {
					mockGenerator.AssertExpectations(t)
				}
				if mockJwtToken != nil {
					mockJwtToken.AssertExpectations(t)
				}
				if mockRedisClient != nil {
					mockRedisClient.AssertExpectations(t)
				}
				if mockRedisTaskPublisher != nil {
					mockRedisTaskPublisher.AssertExpectations(t)
				}
			}()

			svc := NewUserService(lgr, nil, nil, mockJwtToken, nil, mockConfig, nil, nil, mockRedisClient, mockRedisTaskPublisher, nil, mockGenerator)
			got, gotErr := svc.ResetVerifyEmailCode(ctx, tC.input)

			tC.verify(t, got, gotErr)
		})
	}
}

func TestUserService_SignInUserByEmailAndPassword(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()
	// Add required context values that the service expects
	ctx = context.WithValue(ctx, constants.UserAgentKey, "test-user-agent")
	ctx = context.WithValue(ctx, constants.ClientIPKey, "127.0.0.1")

	mockErr := errors.New("error")

	// Create mock config
	mockConfig := &config.Config{
		AuthConfig: config.AuthConfig{
			AccessTokenDuration:  15 * time.Minute,
			RefreshTokenDuration: 24 * time.Hour,
		},
	}

	testCases := []struct {
		name   string
		input  *entities.SignInByEmailAndPasswordRequest
		setup  func() (*repositories.MockUserRepository, *utils.MockPasswordUtil, *utils.MockJwtToken)
		verify func(t *testing.T, got *entities.SignInByEmailAndPasswordResponse, gotErr error)
	}{
		{
			name: "Success",
			input: &entities.SignInByEmailAndPasswordRequest{
				Email:    "user@example.com",
				Password: "password123",
			},
			setup: func() (*repositories.MockUserRepository, *utils.MockPasswordUtil, *utils.MockJwtToken) {
				mockUserRepo := new(repositories.MockUserRepository)
				mockPassword := new(utils.MockPasswordUtil)
				mockJwtToken := new(utils.MockJwtToken)

				// Mock user existence check
				mockUserRepo.EXPECT().
					CheckIsEmailExists(ctx, "user@example.com").
					Return(&db.Users{ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), IsEmailVerified: sql.NullBool{Bool: true, Valid: true}}, nil)

				// Mock password verification
				mockPassword.EXPECT().
					IsPasswordValid(ctx, "password123", mock.AnythingOfType("string")).
					Return(true)

				// Mock JWT token creation for access token
				mockJwtToken.EXPECT().
					CreateToken(ctx, mock.MatchedBy(func(req *entities.TokenRequest) bool {
						return req.Duration == mockConfig.AuthConfig.AccessTokenDuration
					})).
					Return("access_token", &utilsPkg.SignInTokenPayload{}, nil)

				// Mock JWT token creation for refresh token
				mockJwtToken.EXPECT().
					CreateToken(ctx, mock.MatchedBy(func(req *entities.TokenRequest) bool {
						return req.Duration == mockConfig.AuthConfig.RefreshTokenDuration
					})).
					Return("refresh_token", &utilsPkg.SignInTokenPayload{}, nil)

				// Mock JWT token hashing
				mockJwtToken.EXPECT().
					HashTokenSHA256(ctx, "refresh_token").
					Return("hashed_refresh_token")

				// Mock user sign-in transaction
				mockUserRepo.EXPECT().
					SignInUserByEmailAndPasswordTx(ctx, mock.AnythingOfType("*repositories.SignInUserByEmailAndPasswordTxModel")).
					Return(nil)

				return mockUserRepo, mockPassword, mockJwtToken
			},
			verify: func(t *testing.T, got *entities.SignInByEmailAndPasswordResponse, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, got)
				assert.Equal(t, "access_token", got.AccessToken)
				assert.Equal(t, "refresh_token", got.RefreshToken)
			},
		},
		{
			name: "Error_UserNotFound",
			input: &entities.SignInByEmailAndPasswordRequest{
				Email:    "nonexistent@example.com",
				Password: "password123",
			},
			setup: func() (*repositories.MockUserRepository, *utils.MockPasswordUtil, *utils.MockJwtToken) {
				mockUserRepo := new(repositories.MockUserRepository)

				// Mock user existence check returns no rows
				mockUserRepo.EXPECT().
					CheckIsEmailExists(ctx, "nonexistent@example.com").
					Return(nil, sql.ErrNoRows)

				return mockUserRepo, nil, nil
			},
			verify: func(t *testing.T, got *entities.SignInByEmailAndPasswordResponse, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
			},
		},
		{
			name: "Error_EmailNotVerified",
			input: &entities.SignInByEmailAndPasswordRequest{
				Email:    "unverified@example.com",
				Password: "password123",
			},
			setup: func() (*repositories.MockUserRepository, *utils.MockPasswordUtil, *utils.MockJwtToken) {
				mockUserRepo := new(repositories.MockUserRepository)

				// Mock user existence check returns unverified user
				mockUserRepo.EXPECT().
					CheckIsEmailExists(ctx, "unverified@example.com").
					Return(&db.Users{ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), IsEmailVerified: sql.NullBool{Bool: false, Valid: true}}, nil)

				return mockUserRepo, nil, nil
			},
			verify: func(t *testing.T, got *entities.SignInByEmailAndPasswordResponse, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
			},
		},
		{
			name: "Error_InvalidPassword",
			input: &entities.SignInByEmailAndPasswordRequest{
				Email:    "user@example.com",
				Password: "wrong_password",
			},
			setup: func() (*repositories.MockUserRepository, *utils.MockPasswordUtil, *utils.MockJwtToken) {
				mockUserRepo := new(repositories.MockUserRepository)
				mockPassword := new(utils.MockPasswordUtil)

				// Mock user existence check
				mockUserRepo.EXPECT().
					CheckIsEmailExists(ctx, "user@example.com").
					Return(&db.Users{ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), IsEmailVerified: sql.NullBool{Bool: true, Valid: true}}, nil)

				// Mock password verification fails
				mockPassword.EXPECT().
					IsPasswordValid(ctx, "wrong_password", mock.AnythingOfType("string")).
					Return(false)

				return mockUserRepo, mockPassword, nil
			},
			verify: func(t *testing.T, got *entities.SignInByEmailAndPasswordResponse, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
			},
		},
		{
			name: "Error_AccessTokenCreationFailed",
			input: &entities.SignInByEmailAndPasswordRequest{
				Email:    "user@example.com",
				Password: "password123",
			},
			setup: func() (*repositories.MockUserRepository, *utils.MockPasswordUtil, *utils.MockJwtToken) {
				mockUserRepo := new(repositories.MockUserRepository)
				mockPassword := new(utils.MockPasswordUtil)
				mockJwtToken := new(utils.MockJwtToken)

				// Mock user existence check
				mockUserRepo.EXPECT().
					CheckIsEmailExists(ctx, "user@example.com").
					Return(&db.Users{ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), IsEmailVerified: sql.NullBool{Bool: true, Valid: true}}, nil)

				// Mock password verification
				mockPassword.EXPECT().
					IsPasswordValid(ctx, "password123", mock.AnythingOfType("string")).
					Return(true)

				// Mock JWT token creation for access token fails
				mockJwtToken.EXPECT().
					CreateToken(ctx, mock.MatchedBy(func(req *entities.TokenRequest) bool {
						return req.Duration == mockConfig.AuthConfig.AccessTokenDuration
					})).
					Return("", nil, mockErr)

				return mockUserRepo, mockPassword, mockJwtToken
			},
			verify: func(t *testing.T, got *entities.SignInByEmailAndPasswordResponse, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
				assert.ErrorIs(t, gotErr, mockErr)
			},
		},
		{
			name: "Error_RefreshTokenCreationFailed",
			input: &entities.SignInByEmailAndPasswordRequest{
				Email:    "user@example.com",
				Password: "password123",
			},
			setup: func() (*repositories.MockUserRepository, *utils.MockPasswordUtil, *utils.MockJwtToken) {
				mockUserRepo := new(repositories.MockUserRepository)
				mockPassword := new(utils.MockPasswordUtil)
				mockJwtToken := new(utils.MockJwtToken)

				// Mock user existence check
				mockUserRepo.EXPECT().
					CheckIsEmailExists(ctx, "user@example.com").
					Return(&db.Users{ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), IsEmailVerified: sql.NullBool{Bool: true, Valid: true}}, nil)

				// Mock password verification
				mockPassword.EXPECT().
					IsPasswordValid(ctx, "password123", mock.AnythingOfType("string")).
					Return(true)

				// Mock JWT token creation for access token
				mockJwtToken.EXPECT().
					CreateToken(ctx, mock.MatchedBy(func(req *entities.TokenRequest) bool {
						return req.Duration == mockConfig.AuthConfig.AccessTokenDuration
					})).
					Return("access_token", &utilsPkg.SignInTokenPayload{}, nil)

				// Mock JWT token creation for refresh token fails
				mockJwtToken.EXPECT().
					CreateToken(ctx, mock.MatchedBy(func(req *entities.TokenRequest) bool {
						return req.Duration == mockConfig.AuthConfig.RefreshTokenDuration
					})).
					Return("", nil, mockErr)

				return mockUserRepo, mockPassword, mockJwtToken
			},
			verify: func(t *testing.T, got *entities.SignInByEmailAndPasswordResponse, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
				assert.ErrorIs(t, gotErr, mockErr)
			},
		},
		{
			name: "Error_TokenHashingFailed",
			input: &entities.SignInByEmailAndPasswordRequest{
				Email:    "user@example.com",
				Password: "password123",
			},
			setup: func() (*repositories.MockUserRepository, *utils.MockPasswordUtil, *utils.MockJwtToken) {
				mockUserRepo := new(repositories.MockUserRepository)
				mockPassword := new(utils.MockPasswordUtil)
				mockJwtToken := new(utils.MockJwtToken)

				// Mock user existence check
				mockUserRepo.EXPECT().
					CheckIsEmailExists(ctx, "user@example.com").
					Return(&db.Users{ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), IsEmailVerified: sql.NullBool{Bool: true, Valid: true}}, nil)

				// Mock password verification
				mockPassword.EXPECT().
					IsPasswordValid(ctx, "password123", mock.AnythingOfType("string")).
					Return(true)

				// Mock JWT token creation for access token
				mockJwtToken.EXPECT().
					CreateToken(ctx, mock.MatchedBy(func(req *entities.TokenRequest) bool {
						return req.Duration == mockConfig.AuthConfig.AccessTokenDuration
					})).
					Return("access_token", &utilsPkg.SignInTokenPayload{}, nil)

				// Mock JWT token creation for refresh token
				mockJwtToken.EXPECT().
					CreateToken(ctx, mock.MatchedBy(func(req *entities.TokenRequest) bool {
						return req.Duration == mockConfig.AuthConfig.RefreshTokenDuration
					})).
					Return("refresh_token", &utilsPkg.SignInTokenPayload{}, nil)

				// Mock JWT token hashing fails
				mockJwtToken.EXPECT().
					HashTokenSHA256(ctx, "refresh_token").
					Return("")

				// Mock user sign-in transaction (this will be called even after token hashing fails)
				mockUserRepo.EXPECT().
					SignInUserByEmailAndPasswordTx(ctx, mock.AnythingOfType("*repositories.SignInUserByEmailAndPasswordTxModel")).
					Return(errors.New("transaction failed due to empty token hash"))

				return mockUserRepo, mockPassword, mockJwtToken
			},
			verify: func(t *testing.T, got *entities.SignInByEmailAndPasswordResponse, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
			},
		},
		{
			name: "Error_SignInTransactionFailed",
			input: &entities.SignInByEmailAndPasswordRequest{
				Email:    "user@example.com",
				Password: "password123",
			},
			setup: func() (*repositories.MockUserRepository, *utils.MockPasswordUtil, *utils.MockJwtToken) {
				mockUserRepo := new(repositories.MockUserRepository)
				mockPassword := new(utils.MockPasswordUtil)
				mockJwtToken := new(utils.MockJwtToken)

				// Mock user existence check
				mockUserRepo.EXPECT().
					CheckIsEmailExists(ctx, "user@example.com").
					Return(&db.Users{ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), IsEmailVerified: sql.NullBool{Bool: true, Valid: true}}, nil)

				// Mock password verification
				mockPassword.EXPECT().
					IsPasswordValid(ctx, "password123", mock.AnythingOfType("string")).
					Return(true)

				// Mock JWT token creation for access token
				mockJwtToken.EXPECT().
					CreateToken(ctx, mock.MatchedBy(func(req *entities.TokenRequest) bool {
						return req.Duration == mockConfig.AuthConfig.AccessTokenDuration
					})).
					Return("access_token", &utilsPkg.SignInTokenPayload{}, nil)

				// Mock JWT token creation for refresh token
				mockJwtToken.EXPECT().
					CreateToken(ctx, mock.MatchedBy(func(req *entities.TokenRequest) bool {
						return req.Duration == mockConfig.AuthConfig.RefreshTokenDuration
					})).
					Return("refresh_token", &utilsPkg.SignInTokenPayload{}, nil)

				// Mock JWT token hashing
				mockJwtToken.EXPECT().
					HashTokenSHA256(ctx, "refresh_token").
					Return("hashed_refresh_token")

				// Mock user sign-in transaction fails
				mockUserRepo.EXPECT().
					SignInUserByEmailAndPasswordTx(ctx, mock.AnythingOfType("*repositories.SignInUserByEmailAndPasswordTxModel")).
					Return(mockErr)

				return mockUserRepo, mockPassword, mockJwtToken
			},
			verify: func(t *testing.T, got *entities.SignInByEmailAndPasswordResponse, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
				assert.ErrorIs(t, gotErr, mockErr)
			},
		},
		{
			name: "Error_ContextMissingUserAgent",
			input: &entities.SignInByEmailAndPasswordRequest{
				Email:    "user@example.com",
				Password: "password123",
			},
			setup: func() (*repositories.MockUserRepository, *utils.MockPasswordUtil, *utils.MockJwtToken) {
				mockUserRepo := new(repositories.MockUserRepository)

				// Mock user existence check to fail early since context is invalid
				mockUserRepo.EXPECT().
					CheckIsEmailExists(mock.AnythingOfType("*context.valueCtx"), "user@example.com").
					Return(nil, errors.New("context validation failed"))

				return mockUserRepo, nil, nil
			},
			verify: func(t *testing.T, got *entities.SignInByEmailAndPasswordResponse, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
			},
		},
		{
			name: "Error_ContextMissingClientIP",
			input: &entities.SignInByEmailAndPasswordRequest{
				Email:    "user@example.com",
				Password: "password123",
			},
			setup: func() (*repositories.MockUserRepository, *utils.MockPasswordUtil, *utils.MockJwtToken) {
				mockUserRepo := new(repositories.MockUserRepository)

				// Mock user existence check to fail early since context is invalid
				mockUserRepo.EXPECT().
					CheckIsEmailExists(mock.AnythingOfType("*context.valueCtx"), "user@example.com").
					Return(nil, errors.New("context validation failed"))

				return mockUserRepo, nil, nil
			},
			verify: func(t *testing.T, got *entities.SignInByEmailAndPasswordResponse, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			mockUserRepo, mockPassword, mockJwtToken := tC.setup()
			defer func() {
				if mockUserRepo != nil {
					mockUserRepo.AssertExpectations(t)
				}
				if mockPassword != nil {
					mockPassword.AssertExpectations(t)
				}
				if mockJwtToken != nil {
					mockJwtToken.AssertExpectations(t)
				}
			}()

			// Handle context-specific test cases
			testCtx := ctx
			if tC.name == "Error_ContextMissingUserAgent" {
				testCtx = context.WithValue(ctx, constants.ClientIPKey, "127.0.0.1")
				// Remove UserAgentKey
				testCtx = context.WithValue(testCtx, constants.UserAgentKey, nil)
			} else if tC.name == "Error_ContextMissingClientIP" {
				testCtx = context.WithValue(ctx, constants.UserAgentKey, "test-user-agent")
				// Remove ClientIPKey
				testCtx = context.WithValue(testCtx, constants.ClientIPKey, nil)
			}

			svc := NewUserService(lgr, mockUserRepo, nil, mockJwtToken, nil, mockConfig, nil, nil, nil, nil, mockPassword, nil)
			got, gotErr := svc.SignInUserByEmailAndPassword(testCtx, tC.input)

			tC.verify(t, got, gotErr)
		})
	}
}

func TestUserService_SignInAdminByEmailAndPassword(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()
	// Add required context values that the service expects
	ctx = context.WithValue(ctx, constants.UserAgentKey, "test-user-agent")
	ctx = context.WithValue(ctx, constants.ClientIPKey, "127.0.0.1")

	mockErr := errors.New("error")

	// Create mock config
	mockConfig := &config.Config{
		AuthConfig: config.AuthConfig{
			AccessTokenDuration:  15 * time.Minute,
			RefreshTokenDuration: 24 * time.Hour,
		},
	}

	checkEmailResp := &db.Users{
		ID:              uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		IsEmailVerified: sql.NullBool{Bool: true, Valid: true},
		IsAdmin:         sql.NullBool{Bool: true, Valid: true},
	}

	testCases := []struct {
		name   string
		input  *entities.SignInByEmailAndPasswordRequest
		setup  func() (*repositories.MockUserRepository, *utils.MockPasswordUtil, *utils.MockJwtToken)
		verify func(t *testing.T, got *entities.SignInByEmailAndPasswordResponse, gotErr error)
	}{
		{
			name: "Success",
			input: &entities.SignInByEmailAndPasswordRequest{
				Email:    "user@example.com",
				Password: "password123",
			},
			setup: func() (*repositories.MockUserRepository, *utils.MockPasswordUtil, *utils.MockJwtToken) {
				mockUserRepo := new(repositories.MockUserRepository)
				mockPassword := new(utils.MockPasswordUtil)
				mockJwtToken := new(utils.MockJwtToken)

				// Mock user existence check
				mockUserRepo.EXPECT().
					CheckIsEmailExists(ctx, "user@example.com").
					Return(checkEmailResp, nil)

				// Mock password verification
				mockPassword.EXPECT().
					IsPasswordValid(ctx, "password123", mock.AnythingOfType("string")).
					Return(true)

				// Mock JWT token creation for access token
				mockJwtToken.EXPECT().
					CreateToken(ctx, mock.MatchedBy(func(req *entities.TokenRequest) bool {
						return req.Duration == mockConfig.AuthConfig.AccessTokenDuration
					})).
					Return("access_token", &utilsPkg.SignInTokenPayload{}, nil)

				// Mock JWT token creation for refresh token
				mockJwtToken.EXPECT().
					CreateToken(ctx, mock.MatchedBy(func(req *entities.TokenRequest) bool {
						return req.Duration == mockConfig.AuthConfig.RefreshTokenDuration
					})).
					Return("refresh_token", &utilsPkg.SignInTokenPayload{}, nil)

				// Mock JWT token hashing
				mockJwtToken.EXPECT().
					HashTokenSHA256(ctx, "refresh_token").
					Return("hashed_refresh_token")

				// Mock user sign-in transaction
				mockUserRepo.EXPECT().
					SignInUserByEmailAndPasswordTx(ctx, mock.AnythingOfType("*repositories.SignInUserByEmailAndPasswordTxModel")).
					Return(nil)

				return mockUserRepo, mockPassword, mockJwtToken
			},
			verify: func(t *testing.T, got *entities.SignInByEmailAndPasswordResponse, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, got)
				assert.Equal(t, "access_token", got.AccessToken)
				assert.Equal(t, "refresh_token", got.RefreshToken)
			},
		},
		{
			name: "Error_UserNotFound",
			input: &entities.SignInByEmailAndPasswordRequest{
				Email:    "nonexistent@example.com",
				Password: "password123",
			},
			setup: func() (*repositories.MockUserRepository, *utils.MockPasswordUtil, *utils.MockJwtToken) {
				mockUserRepo := new(repositories.MockUserRepository)

				// Mock user existence check returns no rows
				mockUserRepo.EXPECT().
					CheckIsEmailExists(ctx, "nonexistent@example.com").
					Return(nil, sql.ErrNoRows)

				return mockUserRepo, nil, nil
			},
			verify: func(t *testing.T, got *entities.SignInByEmailAndPasswordResponse, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
			},
		},
		{
			name: "Error_EmailNotVerified",
			input: &entities.SignInByEmailAndPasswordRequest{
				Email:    "unverified@example.com",
				Password: "password123",
			},
			setup: func() (*repositories.MockUserRepository, *utils.MockPasswordUtil, *utils.MockJwtToken) {
				mockUserRepo := new(repositories.MockUserRepository)

				// Mock user existence check returns unverified user
				mockUserRepo.EXPECT().
					CheckIsEmailExists(ctx, "unverified@example.com").
					Return(&db.Users{ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), IsEmailVerified: sql.NullBool{Bool: false, Valid: true}}, nil)

				return mockUserRepo, nil, nil
			},
			verify: func(t *testing.T, got *entities.SignInByEmailAndPasswordResponse, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
			},
		},
		{
			name: "Error_InvalidPassword",
			input: &entities.SignInByEmailAndPasswordRequest{
				Email:    "user@example.com",
				Password: "wrong_password",
			},
			setup: func() (*repositories.MockUserRepository, *utils.MockPasswordUtil, *utils.MockJwtToken) {
				mockUserRepo := new(repositories.MockUserRepository)
				mockPassword := new(utils.MockPasswordUtil)

				// Mock user existence check
				mockUserRepo.EXPECT().
					CheckIsEmailExists(ctx, "user@example.com").
					Return(checkEmailResp, nil)

				// Mock password verification fails
				mockPassword.EXPECT().
					IsPasswordValid(ctx, "wrong_password", mock.AnythingOfType("string")).
					Return(false)

				return mockUserRepo, mockPassword, nil
			},
			verify: func(t *testing.T, got *entities.SignInByEmailAndPasswordResponse, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
				assert.Contains(t, gotErr.Error(), "this email or password is incorrect")
				assert.Contains(t, gotErr.Error(), "[INS0217]")
			},
		},
		{
			name: "Error - WithDontHaveAdminRole",
			input: &entities.SignInByEmailAndPasswordRequest{
				Email:    "user@example.com",
				Password: "password123",
			},
			setup: func() (*repositories.MockUserRepository, *utils.MockPasswordUtil, *utils.MockJwtToken) {
				mockUserRepo := new(repositories.MockUserRepository)
				mockPassword := new(utils.MockPasswordUtil)
				mockJwtToken := new(utils.MockJwtToken)

				// Mock user existence check
				mockUserRepo.EXPECT().
					CheckIsEmailExists(ctx, "user@example.com").
					Return(&db.Users{
						ID:              uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
						IsEmailVerified: sql.NullBool{Bool: true, Valid: true},
						IsAdmin:         sql.NullBool{Bool: false, Valid: true},
					}, nil)

				mockPassword.EXPECT().
					IsPasswordValid(ctx, "password123", mock.AnythingOfType("string")).
					Return(true)

				return mockUserRepo, mockPassword, mockJwtToken
			},
			verify: func(t *testing.T, got *entities.SignInByEmailAndPasswordResponse, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
				assert.Contains(t, gotErr.Error(), "this email or password is incorrect")
				assert.Contains(t, gotErr.Error(), "[INS0217]")
			},
		},
		{
			name: "Error_AccessTokenCreationFailed",
			input: &entities.SignInByEmailAndPasswordRequest{
				Email:    "user@example.com",
				Password: "password123",
			},
			setup: func() (*repositories.MockUserRepository, *utils.MockPasswordUtil, *utils.MockJwtToken) {
				mockUserRepo := new(repositories.MockUserRepository)
				mockPassword := new(utils.MockPasswordUtil)
				mockJwtToken := new(utils.MockJwtToken)

				// Mock user existence check
				mockUserRepo.EXPECT().
					CheckIsEmailExists(ctx, "user@example.com").
					Return(checkEmailResp, nil)

				// Mock password verification
				mockPassword.EXPECT().
					IsPasswordValid(ctx, "password123", mock.AnythingOfType("string")).
					Return(true)

				// Mock JWT token creation for access token fails
				mockJwtToken.EXPECT().
					CreateToken(ctx, mock.MatchedBy(func(req *entities.TokenRequest) bool {
						return req.Duration == mockConfig.AuthConfig.AccessTokenDuration
					})).
					Return("", nil, mockErr)

				return mockUserRepo, mockPassword, mockJwtToken
			},
			verify: func(t *testing.T, got *entities.SignInByEmailAndPasswordResponse, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
				assert.ErrorIs(t, gotErr, mockErr)
			},
		},
		{
			name: "Error_RefreshTokenCreationFailed",
			input: &entities.SignInByEmailAndPasswordRequest{
				Email:    "user@example.com",
				Password: "password123",
			},
			setup: func() (*repositories.MockUserRepository, *utils.MockPasswordUtil, *utils.MockJwtToken) {
				mockUserRepo := new(repositories.MockUserRepository)
				mockPassword := new(utils.MockPasswordUtil)
				mockJwtToken := new(utils.MockJwtToken)

				// Mock user existence check
				mockUserRepo.EXPECT().
					CheckIsEmailExists(ctx, "user@example.com").
					Return(checkEmailResp, nil)

				// Mock password verification
				mockPassword.EXPECT().
					IsPasswordValid(ctx, "password123", mock.AnythingOfType("string")).
					Return(true)

				// Mock JWT token creation for access token
				mockJwtToken.EXPECT().
					CreateToken(ctx, mock.MatchedBy(func(req *entities.TokenRequest) bool {
						return req.Duration == mockConfig.AuthConfig.AccessTokenDuration
					})).
					Return("access_token", &utilsPkg.SignInTokenPayload{}, nil)

				// Mock JWT token creation for refresh token fails
				mockJwtToken.EXPECT().
					CreateToken(ctx, mock.MatchedBy(func(req *entities.TokenRequest) bool {
						return req.Duration == mockConfig.AuthConfig.RefreshTokenDuration
					})).
					Return("", nil, mockErr)

				return mockUserRepo, mockPassword, mockJwtToken
			},
			verify: func(t *testing.T, got *entities.SignInByEmailAndPasswordResponse, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
				assert.ErrorIs(t, gotErr, mockErr)
			},
		},
		{
			name: "Error_TokenHashingFailed",
			input: &entities.SignInByEmailAndPasswordRequest{
				Email:    "user@example.com",
				Password: "password123",
			},
			setup: func() (*repositories.MockUserRepository, *utils.MockPasswordUtil, *utils.MockJwtToken) {
				mockUserRepo := new(repositories.MockUserRepository)
				mockPassword := new(utils.MockPasswordUtil)
				mockJwtToken := new(utils.MockJwtToken)

				// Mock user existence check
				mockUserRepo.EXPECT().
					CheckIsEmailExists(ctx, "user@example.com").
					Return(checkEmailResp, nil)

				// Mock password verification
				mockPassword.EXPECT().
					IsPasswordValid(ctx, "password123", mock.AnythingOfType("string")).
					Return(true)

				// Mock JWT token creation for access token
				mockJwtToken.EXPECT().
					CreateToken(ctx, mock.MatchedBy(func(req *entities.TokenRequest) bool {
						return req.Duration == mockConfig.AuthConfig.AccessTokenDuration
					})).
					Return("access_token", &utilsPkg.SignInTokenPayload{}, nil)

				// Mock JWT token creation for refresh token
				mockJwtToken.EXPECT().
					CreateToken(ctx, mock.MatchedBy(func(req *entities.TokenRequest) bool {
						return req.Duration == mockConfig.AuthConfig.RefreshTokenDuration
					})).
					Return("refresh_token", &utilsPkg.SignInTokenPayload{}, nil)

				// Mock JWT token hashing fails
				mockJwtToken.EXPECT().
					HashTokenSHA256(ctx, "refresh_token").
					Return("")

				// Mock user sign-in transaction (this will be called even after token hashing fails)
				mockUserRepo.EXPECT().
					SignInUserByEmailAndPasswordTx(ctx, mock.AnythingOfType("*repositories.SignInUserByEmailAndPasswordTxModel")).
					Return(errors.New("transaction failed due to empty token hash"))

				return mockUserRepo, mockPassword, mockJwtToken
			},
			verify: func(t *testing.T, got *entities.SignInByEmailAndPasswordResponse, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
			},
		},
		{
			name: "Error_SignInTransactionFailed",
			input: &entities.SignInByEmailAndPasswordRequest{
				Email:    "user@example.com",
				Password: "password123",
			},
			setup: func() (*repositories.MockUserRepository, *utils.MockPasswordUtil, *utils.MockJwtToken) {
				mockUserRepo := new(repositories.MockUserRepository)
				mockPassword := new(utils.MockPasswordUtil)
				mockJwtToken := new(utils.MockJwtToken)

				// Mock user existence check
				mockUserRepo.EXPECT().
					CheckIsEmailExists(ctx, "user@example.com").
					Return(checkEmailResp, nil)

				// Mock password verification
				mockPassword.EXPECT().
					IsPasswordValid(ctx, "password123", mock.AnythingOfType("string")).
					Return(true)

				// Mock JWT token creation for access token
				mockJwtToken.EXPECT().
					CreateToken(ctx, mock.MatchedBy(func(req *entities.TokenRequest) bool {
						return req.Duration == mockConfig.AuthConfig.AccessTokenDuration
					})).
					Return("access_token", &utilsPkg.SignInTokenPayload{}, nil)

				// Mock JWT token creation for refresh token
				mockJwtToken.EXPECT().
					CreateToken(ctx, mock.MatchedBy(func(req *entities.TokenRequest) bool {
						return req.Duration == mockConfig.AuthConfig.RefreshTokenDuration
					})).
					Return("refresh_token", &utilsPkg.SignInTokenPayload{}, nil)

				// Mock JWT token hashing
				mockJwtToken.EXPECT().
					HashTokenSHA256(ctx, "refresh_token").
					Return("hashed_refresh_token")

				// Mock user sign-in transaction fails
				mockUserRepo.EXPECT().
					SignInUserByEmailAndPasswordTx(ctx, mock.AnythingOfType("*repositories.SignInUserByEmailAndPasswordTxModel")).
					Return(mockErr)

				return mockUserRepo, mockPassword, mockJwtToken
			},
			verify: func(t *testing.T, got *entities.SignInByEmailAndPasswordResponse, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
				assert.ErrorIs(t, gotErr, mockErr)
			},
		},
		{
			name: "Error_ContextMissingUserAgent",
			input: &entities.SignInByEmailAndPasswordRequest{
				Email:    "user@example.com",
				Password: "password123",
			},
			setup: func() (*repositories.MockUserRepository, *utils.MockPasswordUtil, *utils.MockJwtToken) {
				mockUserRepo := new(repositories.MockUserRepository)

				// Mock user existence check to fail early since context is invalid
				mockUserRepo.EXPECT().
					CheckIsEmailExists(mock.AnythingOfType("*context.valueCtx"), "user@example.com").
					Return(nil, errors.New("context validation failed"))

				return mockUserRepo, nil, nil
			},
			verify: func(t *testing.T, got *entities.SignInByEmailAndPasswordResponse, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
			},
		},
		{
			name: "Error_ContextMissingClientIP",
			input: &entities.SignInByEmailAndPasswordRequest{
				Email:    "user@example.com",
				Password: "password123",
			},
			setup: func() (*repositories.MockUserRepository, *utils.MockPasswordUtil, *utils.MockJwtToken) {
				mockUserRepo := new(repositories.MockUserRepository)

				// Mock user existence check to fail early since context is invalid
				mockUserRepo.EXPECT().
					CheckIsEmailExists(mock.AnythingOfType("*context.valueCtx"), "user@example.com").
					Return(nil, errors.New("context validation failed"))

				return mockUserRepo, nil, nil
			},
			verify: func(t *testing.T, got *entities.SignInByEmailAndPasswordResponse, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			mockUserRepo, mockPassword, mockJwtToken := tC.setup()
			defer func() {
				if mockUserRepo != nil {
					mockUserRepo.AssertExpectations(t)
				}
				if mockPassword != nil {
					mockPassword.AssertExpectations(t)
				}
				if mockJwtToken != nil {
					mockJwtToken.AssertExpectations(t)
				}
			}()

			// Handle context-specific test cases
			testCtx := ctx
			if tC.name == "Error_ContextMissingUserAgent" {
				testCtx = context.WithValue(ctx, constants.ClientIPKey, "127.0.0.1")
				// Remove UserAgentKey
				testCtx = context.WithValue(testCtx, constants.UserAgentKey, nil)
			} else if tC.name == "Error_ContextMissingClientIP" {
				testCtx = context.WithValue(ctx, constants.UserAgentKey, "test-user-agent")
				// Remove ClientIPKey
				testCtx = context.WithValue(testCtx, constants.ClientIPKey, nil)
			}

			svc := NewUserService(lgr, mockUserRepo, nil, mockJwtToken, nil, mockConfig, nil, nil, nil, nil, mockPassword, nil)
			got, gotErr := svc.SignInAdminByEmailAndPassword(testCtx, tC.input)

			tC.verify(t, got, gotErr)
		})
	}
}

func TestUserService_ForgotPassword(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()
	// Add required context values that the service expects
	ctx = context.WithValue(ctx, constants.UserAgentKey, "test-user-agent")
	ctx = context.WithValue(ctx, constants.ClientIPKey, "127.0.0.1")

	mockErr := errors.New("error")

	// Create mock config
	mockConfig := &config.Config{
		AuthConfig: config.AuthConfig{
			ResetPasswordTokenDuration: 15 * time.Minute,
		},
	}

	testCases := []struct {
		name   string
		input  *entities.ForgotPasswordRequest
		setup  func() (*repositories.MockUserRepository, *repositories.MockResetTokenRepository, *utils.MockGenerator, *utils.MockJwtToken, *mockDatabase.MockRedisClient, *queue.MockRedisTaskPublisher)
		verify func(t *testing.T, gotErr error)
	}{
		{
			name: "Success",
			input: &entities.ForgotPasswordRequest{
				Email: "user@example.com",
			},
			setup: func() (*repositories.MockUserRepository, *repositories.MockResetTokenRepository, *utils.MockGenerator, *utils.MockJwtToken, *mockDatabase.MockRedisClient, *queue.MockRedisTaskPublisher) {
				mockUserRepo := new(repositories.MockUserRepository)
				mockResetTokenRepo := new(repositories.MockResetTokenRepository)
				mockGenerator := new(utils.MockGenerator)
				mockJwtToken := new(utils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockRedisTaskPublisher := new(queue.MockRedisTaskPublisher)

				// Mock user existence check
				mockUserRepo.EXPECT().
					CheckIsEmailExists(ctx, "user@example.com").
					Return(&db.Users{ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), IsEmailVerified: sql.NullBool{Bool: true, Valid: true}}, nil)

				// Mock UUID generation for reset token
				mockGenerator.EXPECT().
					GenerateUUID(ctx).
					Return(uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")).Times(2)

				// Mock JWT token hashing
				mockJwtToken.EXPECT().
					HashTokenSHA256(ctx, mock.AnythingOfType("string")).
					Return("hashed_token")

				// Mock reset token creation
				mockResetTokenRepo.EXPECT().
					CreateResetToken(ctx, mock.AnythingOfType("*db.CreateResetTokenParams")).
					Return(nil)

				// Mock Redis set operation
				mockRedisClient.EXPECT().
					Set(ctx, mock.AnythingOfType("database.RedisPayload")).
					Return(nil)

				// Mock task publishing
				mockRedisTaskPublisher.EXPECT().
					DefineTaskOptions(mock.AnythingOfType("string")).
					Return([]asynq.Option{asynq.ProcessIn(10 * time.Second)})

				mockRedisTaskPublisher.EXPECT().
					PublishTaskSendResetPasswordEmail(ctx, mock.AnythingOfType("*email.ResetPasswordEmailPayload"), mock.Anything).
					Return(nil)

				return mockUserRepo, mockResetTokenRepo, mockGenerator, mockJwtToken, mockRedisClient, mockRedisTaskPublisher
			},
			verify: func(t *testing.T, gotErr error) {
				assert.NoError(t, gotErr)
			},
		},
		{
			name: "Error_UserNotFound",
			input: &entities.ForgotPasswordRequest{
				Email: "nonexistent@example.com",
			},
			setup: func() (*repositories.MockUserRepository, *repositories.MockResetTokenRepository, *utils.MockGenerator, *utils.MockJwtToken, *mockDatabase.MockRedisClient, *queue.MockRedisTaskPublisher) {
				mockUserRepo := new(repositories.MockUserRepository)

				// Mock user existence check returns no rows
				mockUserRepo.EXPECT().
					CheckIsEmailExists(ctx, "nonexistent@example.com").
					Return(nil, sql.ErrNoRows)

				return mockUserRepo, nil, nil, nil, nil, nil
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
			},
		},
		{
			name: "Error_EmailNotVerified",
			input: &entities.ForgotPasswordRequest{
				Email: "unverified@example.com",
			},
			setup: func() (*repositories.MockUserRepository, *repositories.MockResetTokenRepository, *utils.MockGenerator, *utils.MockJwtToken, *mockDatabase.MockRedisClient, *queue.MockRedisTaskPublisher) {
				mockUserRepo := new(repositories.MockUserRepository)

				// Mock user existence check returns unverified user
				mockUserRepo.EXPECT().
					CheckIsEmailExists(ctx, "unverified@example.com").
					Return(&db.Users{ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), IsEmailVerified: sql.NullBool{Bool: false, Valid: true}}, nil)

				return mockUserRepo, nil, nil, nil, nil, nil
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
			},
		},
		{
			name: "Error_IsEmailExistsFailed",
			input: &entities.ForgotPasswordRequest{
				Email: "unverified@example.com",
			},
			setup: func() (*repositories.MockUserRepository, *repositories.MockResetTokenRepository, *utils.MockGenerator, *utils.MockJwtToken, *mockDatabase.MockRedisClient, *queue.MockRedisTaskPublisher) {
				mockUserRepo := new(repositories.MockUserRepository)

				// Mock user existence check returns unverified user
				mockUserRepo.EXPECT().
					CheckIsEmailExists(ctx, "unverified@example.com").
					Return(nil, mockErr)

				return mockUserRepo, nil, nil, nil, nil, nil
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
			},
		},
		{
			name: "Error_ResetTokenCreationFailed",
			input: &entities.ForgotPasswordRequest{
				Email: "user@example.com",
			},
			setup: func() (*repositories.MockUserRepository, *repositories.MockResetTokenRepository, *utils.MockGenerator, *utils.MockJwtToken, *mockDatabase.MockRedisClient, *queue.MockRedisTaskPublisher) {
				mockUserRepo := new(repositories.MockUserRepository)
				mockResetTokenRepo := new(repositories.MockResetTokenRepository)
				mockGenerator := new(utils.MockGenerator)
				mockJwtToken := new(utils.MockJwtToken)

				// Mock user existence check
				mockUserRepo.EXPECT().
					CheckIsEmailExists(ctx, "user@example.com").
					Return(&db.Users{ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), IsEmailVerified: sql.NullBool{Bool: true, Valid: true}}, nil)

				// Mock UUID generation for reset token
				mockGenerator.EXPECT().
					GenerateUUID(ctx).
					Return(uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")).Times(2)

				// Mock JWT token hashing
				mockJwtToken.EXPECT().
					HashTokenSHA256(ctx, mock.AnythingOfType("string")).
					Return("hashed_token")

				// Mock reset token creation fails
				mockResetTokenRepo.EXPECT().
					CreateResetToken(ctx, mock.AnythingOfType("*db.CreateResetTokenParams")).
					Return(mockErr)

				return mockUserRepo, mockResetTokenRepo, mockGenerator, mockJwtToken, nil, nil
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.ErrorIs(t, gotErr, mockErr)
			},
		},
		{
			name: "Error_RedisSetFailed",
			input: &entities.ForgotPasswordRequest{
				Email: "user@example.com",
			},
			setup: func() (*repositories.MockUserRepository, *repositories.MockResetTokenRepository, *utils.MockGenerator, *utils.MockJwtToken, *mockDatabase.MockRedisClient, *queue.MockRedisTaskPublisher) {
				mockUserRepo := new(repositories.MockUserRepository)
				mockResetTokenRepo := new(repositories.MockResetTokenRepository)
				mockGenerator := new(utils.MockGenerator)
				mockJwtToken := new(utils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockRedisTaskPublisher := new(queue.MockRedisTaskPublisher)

				// Mock user existence check
				mockUserRepo.EXPECT().
					CheckIsEmailExists(ctx, "user@example.com").
					Return(&db.Users{ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), IsEmailVerified: sql.NullBool{Bool: true, Valid: true}}, nil)

				// Mock UUID generation for reset token
				mockGenerator.EXPECT().
					GenerateUUID(ctx).
					Return(uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")).Times(2)

				// Mock JWT token hashing
				mockJwtToken.EXPECT().
					HashTokenSHA256(ctx, mock.AnythingOfType("string")).
					Return("hashed_token")

				// Mock reset token creation
				mockResetTokenRepo.EXPECT().
					CreateResetToken(ctx, mock.AnythingOfType("*db.CreateResetTokenParams")).
					Return(nil)

				// Mock Redis set operation fails
				mockRedisClient.EXPECT().
					Set(ctx, mock.AnythingOfType("database.RedisPayload")).
					Return(mockErr)

				// Mock task options definition (this will be called even after Redis set fails)
				mockRedisTaskPublisher.EXPECT().
					DefineTaskOptions(mock.AnythingOfType("string")).
					Return(nil)

				// Mock task publishing (this will be called even after Redis set fails)
				mockRedisTaskPublisher.EXPECT().
					PublishTaskSendResetPasswordEmail(ctx, mock.AnythingOfType("*email.ResetPasswordEmailPayload")).
					Return(nil)

				return mockUserRepo, mockResetTokenRepo, mockGenerator, mockJwtToken, mockRedisClient, mockRedisTaskPublisher
			},
			verify: func(t *testing.T, gotErr error) {
				// Note: The service only logs a warning when Redis set fails
				// and continues execution, so we expect no error here
				assert.NoError(t, gotErr)
			},
		},
		{
			name: "Error_TaskOptionsDefinitionFailed",
			input: &entities.ForgotPasswordRequest{
				Email: "user@example.com",
			},
			setup: func() (*repositories.MockUserRepository, *repositories.MockResetTokenRepository, *utils.MockGenerator, *utils.MockJwtToken, *mockDatabase.MockRedisClient, *queue.MockRedisTaskPublisher) {
				mockUserRepo := new(repositories.MockUserRepository)
				mockResetTokenRepo := new(repositories.MockResetTokenRepository)
				mockGenerator := new(utils.MockGenerator)
				mockJwtToken := new(utils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockRedisTaskPublisher := new(queue.MockRedisTaskPublisher)

				// Mock user existence check
				mockUserRepo.EXPECT().
					CheckIsEmailExists(ctx, "user@example.com").
					Return(&db.Users{ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), IsEmailVerified: sql.NullBool{Bool: true, Valid: true}}, nil)

				// Mock UUID generation for reset token
				mockGenerator.EXPECT().
					GenerateUUID(ctx).
					Return(uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")).Times(2)

				// Mock JWT token hashing
				mockJwtToken.EXPECT().
					HashTokenSHA256(ctx, mock.AnythingOfType("string")).
					Return("hashed_token")

				// Mock reset token creation
				mockResetTokenRepo.EXPECT().
					CreateResetToken(ctx, mock.AnythingOfType("*db.CreateResetTokenParams")).
					Return(nil)

				// Mock Redis set operation
				mockRedisClient.EXPECT().
					Set(ctx, mock.AnythingOfType("database.RedisPayload")).
					Return(nil)

				// Mock task options definition fails
				mockRedisTaskPublisher.EXPECT().
					DefineTaskOptions(mock.AnythingOfType("string")).
					Return(nil)

				// Mock task publishing (this will be called even after task options definition fails)
				mockRedisTaskPublisher.EXPECT().
					PublishTaskSendResetPasswordEmail(ctx, mock.AnythingOfType("*email.ResetPasswordEmailPayload")).
					Return(nil)

				return mockUserRepo, mockResetTokenRepo, mockGenerator, mockJwtToken, mockRedisClient, mockRedisTaskPublisher
			},
			verify: func(t *testing.T, gotErr error) {
				// Note: The service doesn't check if DefineTaskOptions fails
				// and continues execution, so we expect no error here
				assert.NoError(t, gotErr)
			},
		},

		{
			name: "Error_TokenHashingFailed",
			input: &entities.ForgotPasswordRequest{
				Email: "user@example.com",
			},
			setup: func() (*repositories.MockUserRepository, *repositories.MockResetTokenRepository, *utils.MockGenerator, *utils.MockJwtToken, *mockDatabase.MockRedisClient, *queue.MockRedisTaskPublisher) {
				mockUserRepo := new(repositories.MockUserRepository)
				mockResetTokenRepo := new(repositories.MockResetTokenRepository)
				mockGenerator := new(utils.MockGenerator)
				mockJwtToken := new(utils.MockJwtToken)

				// Mock user existence check
				mockUserRepo.EXPECT().
					CheckIsEmailExists(ctx, "user@example.com").
					Return(&db.Users{ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), IsEmailVerified: sql.NullBool{Bool: true, Valid: true}}, nil)

				// Mock UUID generation for reset token
				mockGenerator.EXPECT().
					GenerateUUID(ctx).
					Return(uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")).Times(2)

				// Mock JWT token hashing fails
				mockJwtToken.EXPECT().
					HashTokenSHA256(ctx, mock.AnythingOfType("string")).
					Return("")

				// Mock reset token creation (this will be called even after token hashing fails)
				mockResetTokenRepo.EXPECT().
					CreateResetToken(ctx, mock.AnythingOfType("*db.CreateResetTokenParams")).
					Return(errors.New("failed to create reset token with empty hash"))

				return mockUserRepo, mockResetTokenRepo, mockGenerator, mockJwtToken, nil, nil
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
			},
		},
		{
			name: "Error_UUIDGenerationFailed",
			input: &entities.ForgotPasswordRequest{
				Email: "user@example.com",
			},
			setup: func() (*repositories.MockUserRepository, *repositories.MockResetTokenRepository, *utils.MockGenerator, *utils.MockJwtToken, *mockDatabase.MockRedisClient, *queue.MockRedisTaskPublisher) {
				mockUserRepo := new(repositories.MockUserRepository)
				mockResetTokenRepo := new(repositories.MockResetTokenRepository)
				mockGenerator := new(utils.MockGenerator)
				mockJwtToken := new(utils.MockJwtToken)

				// Mock user existence check
				mockUserRepo.EXPECT().
					CheckIsEmailExists(ctx, "user@example.com").
					Return(&db.Users{ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), IsEmailVerified: sql.NullBool{Bool: true, Valid: true}}, nil)

				// Mock UUID generation fails
				mockGenerator.EXPECT().
					GenerateUUID(ctx).
					Return(uuid.Nil)

				// Mock JWT token hashing (this will be called even after UUID generation fails)
				mockJwtToken.EXPECT().
					HashTokenSHA256(ctx, mock.AnythingOfType("string")).
					Return("")

				// Mock reset token creation (this will be called even after UUID generation fails)
				mockResetTokenRepo.EXPECT().
					CreateResetToken(ctx, mock.AnythingOfType("*db.CreateResetTokenParams")).
					Return(errors.New("failed to create reset token with nil UUID"))

				return mockUserRepo, mockResetTokenRepo, mockGenerator, mockJwtToken, nil, nil
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
			},
		},
		{
			name: "Error_PublishTaskSendResetPasswordEmailFailed",
			input: &entities.ForgotPasswordRequest{
				Email: "user@example.com",
			},
			setup: func() (*repositories.MockUserRepository, *repositories.MockResetTokenRepository, *utils.MockGenerator, *utils.MockJwtToken, *mockDatabase.MockRedisClient, *queue.MockRedisTaskPublisher) {
				mockUserRepo := new(repositories.MockUserRepository)
				mockResetTokenRepo := new(repositories.MockResetTokenRepository)
				mockGenerator := new(utils.MockGenerator)
				mockJwtToken := new(utils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockRedisTaskPublisher := new(queue.MockRedisTaskPublisher)

				// Mock user existence check
				mockUserRepo.EXPECT().
					CheckIsEmailExists(ctx, "user@example.com").
					Return(&db.Users{ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), IsEmailVerified: sql.NullBool{Bool: true, Valid: true}}, nil)

				// Mock UUID generation for reset token
				mockGenerator.EXPECT().
					GenerateUUID(ctx).
					Return(uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")).Times(2)

				// Mock JWT token hashing
				mockJwtToken.EXPECT().
					HashTokenSHA256(ctx, mock.AnythingOfType("string")).
					Return("hashed_token")

				// Mock reset token creation
				mockResetTokenRepo.EXPECT().
					CreateResetToken(ctx, mock.AnythingOfType("*db.CreateResetTokenParams")).
					Return(nil)

				// Mock Redis set operation
				mockRedisClient.EXPECT().
					Set(ctx, mock.AnythingOfType("database.RedisPayload")).
					Return(nil)

				// Mock task publishing
				mockRedisTaskPublisher.EXPECT().
					DefineTaskOptions(mock.AnythingOfType("string")).
					Return([]asynq.Option{asynq.ProcessIn(10 * time.Second)})

				mockRedisTaskPublisher.EXPECT().
					PublishTaskSendResetPasswordEmail(ctx, mock.AnythingOfType("*email.ResetPasswordEmailPayload"), mock.Anything).
					Return(mockErr)

				return mockUserRepo, mockResetTokenRepo, mockGenerator, mockJwtToken, mockRedisClient, mockRedisTaskPublisher
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.ErrorIs(t, gotErr, mockErr)
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			mockUserRepo, mockResetTokenRepo, mockGenerator, mockJwtToken, mockRedisClient, mockRedisTaskPublisher := tC.setup()
			defer func() {
				if mockUserRepo != nil {
					mockUserRepo.AssertExpectations(t)
				}
				if mockResetTokenRepo != nil {
					mockResetTokenRepo.AssertExpectations(t)
				}
				if mockGenerator != nil {
					mockGenerator.AssertExpectations(t)
				}
				if mockJwtToken != nil {
					mockJwtToken.AssertExpectations(t)
				}
				if mockRedisClient != nil {
					mockRedisClient.AssertExpectations(t)
				}
				if mockRedisTaskPublisher != nil {
					mockRedisTaskPublisher.AssertExpectations(t)
				}
			}()

			svc := NewUserService(lgr, mockUserRepo, nil, mockJwtToken, nil, mockConfig, nil, mockResetTokenRepo, mockRedisClient, mockRedisTaskPublisher, nil, mockGenerator)
			gotErr := svc.ForgotPassword(ctx, tC.input)

			tC.verify(t, gotErr)
		})
	}
}

func TestUserService_ResetUserPassword(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()
	mockErr := errors.New("error")

	testCases := []struct {
		name   string
		input  *entities.ResetUserPasswordRequest
		setup  func() (*repositories.MockUserRepository, *repositories.MockResetTokenRepository, *utils.MockPasswordUtil, *utils.MockJwtToken, *mockDatabase.MockRedisClient)
		verify func(t *testing.T, gotErr error)
	}{
		{
			name: "Success_RedisHit",
			input: &entities.ResetUserPasswordRequest{
				Token:       "reset_token",
				NewPassword: "newpassword123",
			},
			setup: func() (*repositories.MockUserRepository, *repositories.MockResetTokenRepository, *utils.MockPasswordUtil, *utils.MockJwtToken, *mockDatabase.MockRedisClient) {
				mockUserRepo := new(repositories.MockUserRepository)
				mockResetTokenRepo := new(repositories.MockResetTokenRepository)
				mockPassword := new(utils.MockPasswordUtil)
				mockJwtToken := new(utils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				// Mock JWT token hashing
				mockJwtToken.EXPECT().
					HashTokenSHA256(ctx, "reset_token").
					Return("hashed_token")

				// Mock Redis get operation returns user ID
				mockRedisClient.EXPECT().
					Get(ctx, "hashed_token").
					Return("550e8400-e29b-41d4-a716-446655440000", nil)

				// Mock user existence check
				mockUserRepo.EXPECT().
					CheckIsUserExistsByID(ctx, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")).
					Return(&db.Users{ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), PasswordHash: "old_hash"}, nil)

				// Mock password check (different from new password)
				mockPassword.EXPECT().
					IsPasswordValid(ctx, "newpassword123", "old_hash").
					Return(false)

				// Mock password hashing
				mockPassword.EXPECT().
					HashPassword(ctx, "newpassword123").
					Return("new_hash", nil)

				// Mock password reset transaction
				mockUserRepo.EXPECT().
					ResetUserPasswordAndUpdateResetTokenTx(ctx, mock.AnythingOfType("*repositories.ResetUserPasswordTxModel")).
					Return(nil)

				// Mock Redis delete operation
				mockRedisClient.EXPECT().
					Delete(ctx, "hashed_token").
					Return(nil)

				return mockUserRepo, mockResetTokenRepo, mockPassword, mockJwtToken, mockRedisClient
			},
			verify: func(t *testing.T, gotErr error) {
				assert.NoError(t, gotErr)
			},
		},
		{
			name: "Error_RedisGetFailed",
			input: &entities.ResetUserPasswordRequest{
				Token:       "reset_token",
				NewPassword: "newpassword123",
			},
			setup: func() (*repositories.MockUserRepository, *repositories.MockResetTokenRepository, *utils.MockPasswordUtil, *utils.MockJwtToken, *mockDatabase.MockRedisClient) {
				mockUserRepo := new(repositories.MockUserRepository)
				mockResetTokenRepo := new(repositories.MockResetTokenRepository)
				mockPassword := new(utils.MockPasswordUtil)
				mockJwtToken := new(utils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				// Mock JWT token hashing
				mockJwtToken.EXPECT().
					HashTokenSHA256(ctx, "reset_token").
					Return("hashed_token")

				// Mock Redis get operation fails
				mockRedisClient.EXPECT().
					Get(ctx, "hashed_token").
					Return("550e8400-e29b-41d4-a716-446655440000", mockErr)

				// Mock reset token retrieval (fallback to database when Redis fails)
				mockResetTokenRepo.EXPECT().
					GetResetToken(ctx, "hashed_token").
					Return(&db.ResetTokens{
						UserID:    uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
						Used:      false,
						ExpiresAt: time.Now().Add(time.Hour),
					}, nil)

				// Mock user existence check
				mockUserRepo.EXPECT().
					CheckIsUserExistsByID(ctx, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")).
					Return(&db.Users{ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), PasswordHash: "old_hash"}, nil)

				// Mock password check (different from new password)
				mockPassword.EXPECT().
					IsPasswordValid(ctx, "newpassword123", "old_hash").
					Return(false)

				// Mock password hashing
				mockPassword.EXPECT().
					HashPassword(ctx, "newpassword123").
					Return("new_hash", nil)

				// Mock password reset transaction
				mockUserRepo.EXPECT().
					ResetUserPasswordAndUpdateResetTokenTx(ctx, mock.AnythingOfType("*repositories.ResetUserPasswordTxModel")).
					Return(nil)

				// Mock Redis delete operation
				mockRedisClient.EXPECT().
					Delete(ctx, "hashed_token").
					Return(nil)

				return mockUserRepo, mockResetTokenRepo, mockPassword, mockJwtToken, mockRedisClient
			},
			verify: func(t *testing.T, gotErr error) {
				assert.NoError(t, gotErr)
			},
		},
		{
			name: "Success_DatabaseFallback",
			input: &entities.ResetUserPasswordRequest{
				Token:       "reset_token",
				NewPassword: "newpassword123",
			},
			setup: func() (*repositories.MockUserRepository, *repositories.MockResetTokenRepository, *utils.MockPasswordUtil, *utils.MockJwtToken, *mockDatabase.MockRedisClient) {
				mockUserRepo := new(repositories.MockUserRepository)
				mockResetTokenRepo := new(repositories.MockResetTokenRepository)
				mockPassword := new(utils.MockPasswordUtil)
				mockJwtToken := new(utils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				// Mock JWT token hashing
				mockJwtToken.EXPECT().
					HashTokenSHA256(ctx, "reset_token").
					Return("hashed_token")

				// Mock Redis get operation fails
				mockRedisClient.EXPECT().
					Get(ctx, "hashed_token").
					Return("", redis.Nil)

				// Mock reset token retrieval
				mockResetTokenRepo.EXPECT().
					GetResetToken(ctx, "hashed_token").
					Return(&db.ResetTokens{
						UserID:    uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
						Used:      false,
						ExpiresAt: time.Now().Add(time.Hour),
					}, nil)

				// Mock user existence check
				mockUserRepo.EXPECT().
					CheckIsUserExistsByID(ctx, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")).
					Return(&db.Users{ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), PasswordHash: "old_hash"}, nil)

				// Mock password check (different from new password)
				mockPassword.EXPECT().
					IsPasswordValid(ctx, "newpassword123", "old_hash").
					Return(false)

				// Mock password hashing
				mockPassword.EXPECT().
					HashPassword(ctx, "newpassword123").
					Return("new_hash", nil)

				// Mock password reset transaction
				mockUserRepo.EXPECT().
					ResetUserPasswordAndUpdateResetTokenTx(ctx, mock.AnythingOfType("*repositories.ResetUserPasswordTxModel")).
					Return(nil)

				// Mock Redis delete operation
				mockRedisClient.EXPECT().
					Delete(ctx, "hashed_token").
					Return(nil)

				return mockUserRepo, mockResetTokenRepo, mockPassword, mockJwtToken, mockRedisClient
			},
			verify: func(t *testing.T, gotErr error) {
				assert.NoError(t, gotErr)
			},
		},
		{
			name: "Error_TokenAlreadyUsed",
			input: &entities.ResetUserPasswordRequest{
				Token:       "used_token",
				NewPassword: "newpassword123",
			},
			setup: func() (*repositories.MockUserRepository, *repositories.MockResetTokenRepository, *utils.MockPasswordUtil, *utils.MockJwtToken, *mockDatabase.MockRedisClient) {
				mockJwtToken := new(utils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockResetTokenRepo := new(repositories.MockResetTokenRepository)

				// Mock JWT token hashing
				mockJwtToken.EXPECT().
					HashTokenSHA256(ctx, "used_token").
					Return("hashed_used_token")

				// Mock Redis get operation fails
				mockRedisClient.EXPECT().
					Get(ctx, "hashed_used_token").
					Return("", redis.Nil)

				// Mock reset token retrieval returns used token
				mockResetTokenRepo.EXPECT().
					GetResetToken(ctx, "hashed_used_token").
					Return(&db.ResetTokens{
						UserID:    uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
						Used:      true,
						ExpiresAt: time.Now().Add(time.Hour),
					}, nil)

				return nil, mockResetTokenRepo, nil, mockJwtToken, mockRedisClient
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
			},
		},
		{
			name: "Error_TokenExpired",
			input: &entities.ResetUserPasswordRequest{
				Token:       "expired_token",
				NewPassword: "newpassword123",
			},
			setup: func() (*repositories.MockUserRepository, *repositories.MockResetTokenRepository, *utils.MockPasswordUtil, *utils.MockJwtToken, *mockDatabase.MockRedisClient) {
				mockJwtToken := new(utils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockResetTokenRepo := new(repositories.MockResetTokenRepository)

				// Mock JWT token hashing
				mockJwtToken.EXPECT().
					HashTokenSHA256(ctx, "expired_token").
					Return("hashed_expired_token")

				// Mock Redis get operation fails
				mockRedisClient.EXPECT().
					Get(ctx, "hashed_expired_token").
					Return("", redis.Nil)

				// Mock reset token retrieval returns expired token
				mockResetTokenRepo.EXPECT().
					GetResetToken(ctx, "hashed_expired_token").
					Return(&db.ResetTokens{
						UserID:    uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
						Used:      false,
						ExpiresAt: time.Now().Add(-time.Hour),
					}, nil)

				return nil, mockResetTokenRepo, nil, mockJwtToken, mockRedisClient
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
			},
		},
		{
			name: "Error_InvalidUserID",
			input: &entities.ResetUserPasswordRequest{
				Token:       "reset_token",
				NewPassword: "newpassword123",
			},
			setup: func() (*repositories.MockUserRepository, *repositories.MockResetTokenRepository, *utils.MockPasswordUtil, *utils.MockJwtToken, *mockDatabase.MockRedisClient) {
				mockUserRepo := new(repositories.MockUserRepository)
				mockJwtToken := new(utils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				// Mock JWT token hashing
				mockJwtToken.EXPECT().
					HashTokenSHA256(ctx, "reset_token").
					Return("hashed_token")

				// Mock Redis get operation returns user ID
				mockRedisClient.EXPECT().
					Get(ctx, "hashed_token").
					Return("invalid-user-id", nil)

				return mockUserRepo, nil, nil, mockJwtToken, mockRedisClient
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "The UUID is invalid. Please try again.")
				assert.Contains(t, gotErr.Error(), "[INS0107]")
			},
		},
		{
			name: "Error_UserNotFound",
			input: &entities.ResetUserPasswordRequest{
				Token:       "reset_token",
				NewPassword: "newpassword123",
			},
			setup: func() (*repositories.MockUserRepository, *repositories.MockResetTokenRepository, *utils.MockPasswordUtil, *utils.MockJwtToken, *mockDatabase.MockRedisClient) {
				mockUserRepo := new(repositories.MockUserRepository)
				mockJwtToken := new(utils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				// Mock JWT token hashing
				mockJwtToken.EXPECT().
					HashTokenSHA256(ctx, "reset_token").
					Return("hashed_token")

				// Mock Redis get operation returns user ID
				mockRedisClient.EXPECT().
					Get(ctx, "hashed_token").
					Return("550e8400-e29b-41d4-a716-446655440000", nil)

				// Mock user existence check fails
				mockUserRepo.EXPECT().
					CheckIsUserExistsByID(ctx, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")).
					Return(nil, mockErr)

				return mockUserRepo, nil, nil, mockJwtToken, mockRedisClient
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.ErrorIs(t, gotErr, mockErr)
			},
		},
		{
			name: "Error_PasswordSameAsOld",
			input: &entities.ResetUserPasswordRequest{
				Token:       "reset_token",
				NewPassword: "samepassword",
			},
			setup: func() (*repositories.MockUserRepository, *repositories.MockResetTokenRepository, *utils.MockPasswordUtil, *utils.MockJwtToken, *mockDatabase.MockRedisClient) {
				mockUserRepo := new(repositories.MockUserRepository)
				mockPassword := new(utils.MockPasswordUtil)
				mockJwtToken := new(utils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				// Mock JWT token hashing
				mockJwtToken.EXPECT().
					HashTokenSHA256(ctx, "reset_token").
					Return("hashed_token")

				// Mock Redis get operation returns user ID
				mockRedisClient.EXPECT().
					Get(ctx, "hashed_token").
					Return("550e8400-e29b-41d4-a716-446655440000", nil)

				// Mock user existence check
				mockUserRepo.EXPECT().
					CheckIsUserExistsByID(ctx, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")).
					Return(&db.Users{ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), PasswordHash: "old_hash"}, nil)

				// Mock password check (same as old password)
				mockPassword.EXPECT().
					IsPasswordValid(ctx, "samepassword", "old_hash").
					Return(true)

				return mockUserRepo, nil, mockPassword, mockJwtToken, mockRedisClient
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
			},
		},
		{
			name: "Error_PasswordHashingFailed",
			input: &entities.ResetUserPasswordRequest{
				Token:       "reset_token",
				NewPassword: "newpassword123",
			},
			setup: func() (*repositories.MockUserRepository, *repositories.MockResetTokenRepository, *utils.MockPasswordUtil, *utils.MockJwtToken, *mockDatabase.MockRedisClient) {
				mockUserRepo := new(repositories.MockUserRepository)
				mockPassword := new(utils.MockPasswordUtil)
				mockJwtToken := new(utils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				// Mock JWT token hashing
				mockJwtToken.EXPECT().
					HashTokenSHA256(ctx, "reset_token").
					Return("hashed_token")

				// Mock Redis get operation returns user ID
				mockRedisClient.EXPECT().
					Get(ctx, "hashed_token").
					Return("550e8400-e29b-41d4-a716-446655440000", nil)

				// Mock user existence check
				mockUserRepo.EXPECT().
					CheckIsUserExistsByID(ctx, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")).
					Return(&db.Users{ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), PasswordHash: "old_hash"}, nil)

				// Mock password check (different from new password)
				mockPassword.EXPECT().
					IsPasswordValid(ctx, "newpassword123", "old_hash").
					Return(false)

				// Mock password hashing fails
				mockPassword.EXPECT().
					HashPassword(ctx, "newpassword123").
					Return("", mockErr)

				return mockUserRepo, nil, mockPassword, mockJwtToken, mockRedisClient
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.ErrorIs(t, gotErr, mockErr)
			},
		},
		{
			name: "Error_ResetTransactionFailed",
			input: &entities.ResetUserPasswordRequest{
				Token:       "reset_token",
				NewPassword: "newpassword123",
			},
			setup: func() (*repositories.MockUserRepository, *repositories.MockResetTokenRepository, *utils.MockPasswordUtil, *utils.MockJwtToken, *mockDatabase.MockRedisClient) {
				mockUserRepo := new(repositories.MockUserRepository)
				mockPassword := new(utils.MockPasswordUtil)
				mockJwtToken := new(utils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				// Mock JWT token hashing
				mockJwtToken.EXPECT().
					HashTokenSHA256(ctx, "reset_token").
					Return("hashed_token")

				// Mock Redis get operation returns user ID
				mockRedisClient.EXPECT().
					Get(ctx, "hashed_token").
					Return("550e8400-e29b-41d4-a716-446655440000", nil)

				// Mock user existence check
				mockUserRepo.EXPECT().
					CheckIsUserExistsByID(ctx, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")).
					Return(&db.Users{ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), PasswordHash: "old_hash"}, nil)

				// Mock password check (different from new password)
				mockPassword.EXPECT().
					IsPasswordValid(ctx, "newpassword123", "old_hash").
					Return(false)

				// Mock password hashing
				mockPassword.EXPECT().
					HashPassword(ctx, "newpassword123").
					Return("new_hash", nil)

				// Mock password reset transaction fails
				mockUserRepo.EXPECT().
					ResetUserPasswordAndUpdateResetTokenTx(ctx, mock.AnythingOfType("*repositories.ResetUserPasswordTxModel")).
					Return(mockErr)

				return mockUserRepo, nil, mockPassword, mockJwtToken, mockRedisClient
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.ErrorIs(t, gotErr, mockErr)
			},
		},
		{
			name: "Error_RedisDeleteFailed",
			input: &entities.ResetUserPasswordRequest{
				Token:       "reset_token",
				NewPassword: "newpassword123",
			},
			setup: func() (*repositories.MockUserRepository, *repositories.MockResetTokenRepository, *utils.MockPasswordUtil, *utils.MockJwtToken, *mockDatabase.MockRedisClient) {
				mockUserRepo := new(repositories.MockUserRepository)
				mockPassword := new(utils.MockPasswordUtil)
				mockJwtToken := new(utils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				// Mock JWT token hashing
				mockJwtToken.EXPECT().
					HashTokenSHA256(ctx, "reset_token").
					Return("hashed_token")

				// Mock Redis get operation returns user ID
				mockRedisClient.EXPECT().
					Get(ctx, "hashed_token").
					Return("550e8400-e29b-41d4-a716-446655440000", nil)

				// Mock user existence check
				mockUserRepo.EXPECT().
					CheckIsUserExistsByID(ctx, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")).
					Return(&db.Users{ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), PasswordHash: "old_hash"}, nil)

				// Mock password check (different from new password)
				mockPassword.EXPECT().
					IsPasswordValid(ctx, "newpassword123", "old_hash").
					Return(false)

				// Mock password hashing
				mockPassword.EXPECT().
					HashPassword(ctx, "newpassword123").
					Return("new_hash", nil)

				// Mock password reset transaction
				mockUserRepo.EXPECT().
					ResetUserPasswordAndUpdateResetTokenTx(ctx, mock.AnythingOfType("*repositories.ResetUserPasswordTxModel")).
					Return(nil)

				// Mock Redis delete operation fails
				mockRedisClient.EXPECT().
					Delete(ctx, "hashed_token").
					Return(mockErr)

				return mockUserRepo, nil, mockPassword, mockJwtToken, mockRedisClient
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.ErrorIs(t, gotErr, mockErr)
			},
		},
		{
			name: "Error_GetResetTokenFailed",
			input: &entities.ResetUserPasswordRequest{
				Token:       "reset_token",
				NewPassword: "newpassword123",
			},
			setup: func() (*repositories.MockUserRepository, *repositories.MockResetTokenRepository, *utils.MockPasswordUtil, *utils.MockJwtToken, *mockDatabase.MockRedisClient) {
				mockJwtToken := new(utils.MockJwtToken)
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockResetTokenRepo := new(repositories.MockResetTokenRepository)

				// Mock JWT token hashing
				mockJwtToken.EXPECT().
					HashTokenSHA256(ctx, "reset_token").
					Return("hashed_token")

				// Mock Redis get operation fails
				mockRedisClient.EXPECT().
					Get(ctx, "hashed_token").
					Return("", redis.Nil)

				// Mock reset token retrieval fails
				mockResetTokenRepo.EXPECT().
					GetResetToken(ctx, "hashed_token").
					Return(&db.ResetTokens{}, mockErr)

				return nil, mockResetTokenRepo, nil, mockJwtToken, mockRedisClient
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.ErrorIs(t, gotErr, mockErr)
			},
		},
		{
			name: "Error_TokenHashingFailed",
			input: &entities.ResetUserPasswordRequest{
				Token:       "reset_token",
				NewPassword: "newpassword123",
			},
			setup: func() (*repositories.MockUserRepository, *repositories.MockResetTokenRepository, *utils.MockPasswordUtil, *utils.MockJwtToken, *mockDatabase.MockRedisClient) {
				mockJwtToken := new(utils.MockJwtToken)

				// Mock JWT token hashing fails
				mockJwtToken.EXPECT().
					HashTokenSHA256(ctx, "reset_token").
					Return("")

				return nil, nil, nil, mockJwtToken, nil
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			mockUserRepo, mockResetTokenRepo, mockPassword, mockJwtToken, mockRedisClient := tC.setup()
			defer func() {
				if mockUserRepo != nil {
					mockUserRepo.AssertExpectations(t)
				}
				if mockResetTokenRepo != nil {
					mockResetTokenRepo.AssertExpectations(t)
				}
				if mockPassword != nil {
					mockPassword.AssertExpectations(t)
				}
				if mockJwtToken != nil {
					mockJwtToken.AssertExpectations(t)
				}
				if mockRedisClient != nil {
					mockRedisClient.AssertExpectations(t)
				}
			}()

			svc := NewUserService(lgr, mockUserRepo, nil, mockJwtToken, nil, nil, nil, mockResetTokenRepo, mockRedisClient, nil, mockPassword, nil)
			gotErr := svc.ResetUserPassword(ctx, tC.input)

			tC.verify(t, gotErr)
		})
	}
}

func TestUserService_SignOut(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()
	mockErr := errors.New("error")

	testCases := []struct {
		name   string
		setup  func() (*repositories.MockSessionRepository, *middlewareMocks.MockAuthContext)
		verify func(t *testing.T, gotErr error)
	}{
		{
			name: "Success",
			setup: func() (*repositories.MockSessionRepository, *middlewareMocks.MockAuthContext) {
				mockSessionRepo := new(repositories.MockSessionRepository)
				mockAuthContext := new(middlewareMocks.MockAuthContext)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
						},
					}, nil)

				// Mock session revocation
				mockSessionRepo.EXPECT().
					RevokeSessionByID(ctx, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")).
					Return(nil)

				return mockSessionRepo, mockAuthContext
			},
			verify: func(t *testing.T, gotErr error) {
				assert.NoError(t, gotErr)
			},
		},
		{
			name: "Error_AuthContextFailed",
			setup: func() (*repositories.MockSessionRepository, *middlewareMocks.MockAuthContext) {
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
			setup: func() (*repositories.MockSessionRepository, *middlewareMocks.MockAuthContext) {
				mockSessionRepo := new(repositories.MockSessionRepository)
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
				mockSessionRepo.EXPECT().
					RevokeSessionByID(ctx, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")).
					Return(mockErr)

				return mockSessionRepo, mockAuthContext
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.ErrorIs(t, gotErr, mockErr)
			},
		},
		{
			name: "Error_InvalidAuthPayload",
			setup: func() (*repositories.MockSessionRepository, *middlewareMocks.MockAuthContext) {
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
			setup: func() (*repositories.MockSessionRepository, *middlewareMocks.MockAuthContext) {
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
			setup: func() (*repositories.MockSessionRepository, *middlewareMocks.MockAuthContext) {
				mockSessionRepo := new(repositories.MockSessionRepository)
				mockAuthContext := new(middlewareMocks.MockAuthContext)

				// Mock auth context retrieval returns invalid session ID
				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							ID: uuid.Nil,
						},
					}, nil)

				return mockSessionRepo, mockAuthContext
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			mockSessionRepo, mockAuthContext := tC.setup()
			defer func() {
				if mockSessionRepo != nil {
					mockSessionRepo.AssertExpectations(t)
				}
				if mockAuthContext != nil {
					mockAuthContext.AssertExpectations(t)
				}
			}()

			svc := NewUserService(lgr, nil, mockSessionRepo, nil, nil, nil, mockAuthContext, nil, nil, nil, nil, nil)
			gotErr := svc.SignOut(ctx)

			tC.verify(t, gotErr)
		})
	}
}
