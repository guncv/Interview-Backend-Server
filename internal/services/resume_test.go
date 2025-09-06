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
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	db "gitlab.com/interview-simulation/interview-backend-server/internal/db/sqlc"
	"gitlab.com/interview-simulation/interview-backend-server/internal/entities"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
	"gitlab.com/interview-simulation/interview-backend-server/internal/middleware"
	mockS3 "gitlab.com/interview-simulation/interview-backend-server/internal/mocks/aws"
	mockDatabase "gitlab.com/interview-simulation/interview-backend-server/internal/mocks/database"
	mockMiddleware "gitlab.com/interview-simulation/interview-backend-server/internal/mocks/middleware"
	"gitlab.com/interview-simulation/interview-backend-server/internal/mocks/queue"
	mockResume "gitlab.com/interview-simulation/interview-backend-server/internal/mocks/repositories"
	utilsPkg "gitlab.com/interview-simulation/interview-backend-server/internal/utils"
)

func TestResumeService_ListResume(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()
	userID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	testCases := []struct {
		name   string
		input  *entities.ListResumeRequest
		setup  func() (*mockResume.MockResumeReposity, *mockMiddleware.MockAuthContext, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient)
		verify func(t *testing.T, gotResp *entities.ListResumeResponse, gotErr error)
	}{
		{
			name:  "Success - Redis hit for default resume",
			input: &entities.ListResumeRequest{},
			setup: func() (*mockResume.MockResumeReposity, *mockMiddleware.MockAuthContext, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeRepository := new(mockResume.MockResumeReposity)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockRedisTaskPublisher := new(queue.MockRedisTaskPublisher)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							ID:     userID,
							UserID: userID.String(),
						},
					}, nil)

				defaultResume := db.Resumes{
					ID: uuid.New(),
				}

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s:%s", constants.RedisPrefixDefaultResume, userID.String())).
					Return(`{"id":"`+defaultResume.ID.String()+`","file_name":"default.pdf","mime_type":"application/pdf","byte_size":1024,"is_default":true,"created_at":"2023-01-01T00:00:00Z","updated_at":"2023-01-01T00:00:00Z"}`, nil)

				mockResumeRepository.EXPECT().
					ListResumeByUserIDFirstPage(ctx, userID).
					Return([]db.Resumes{}, nil)

				return mockResumeRepository, mockAuthContext, mockRedisTaskPublisher, mockRedisClient
			},
			verify: func(t *testing.T, gotResp *entities.ListResumeResponse, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, gotResp)
				assert.Equal(t, 1, gotResp.Count)
				assert.Equal(t, "default.pdf", gotResp.ResumeContent.DefaultResume.FileName)
				assert.Equal(t, 0, len(gotResp.ResumeContent.Resumes))
			},
		},
		{
			name:  "Error_Invalid user ID",
			input: &entities.ListResumeRequest{},
			setup: func() (*mockResume.MockResumeReposity, *mockMiddleware.MockAuthContext, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeRepository := new(mockResume.MockResumeReposity)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockRedisTaskPublisher := new(queue.MockRedisTaskPublisher)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				invalidUserID := "invalid-user-id"

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							ID:     userID,
							UserID: invalidUserID,
						},
					}, nil)

				return mockResumeRepository, mockAuthContext, mockRedisTaskPublisher, mockRedisClient
			},
			verify: func(t *testing.T, gotResp *entities.ListResumeResponse, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "The UUID is invalid. Please try again.")
				assert.Contains(t, gotErr.Error(), "[ONX0107]")
			},
		},
		{
			name:  "Error - Redis hit but fetch list Resume First Page failed",
			input: &entities.ListResumeRequest{},
			setup: func() (*mockResume.MockResumeReposity, *mockMiddleware.MockAuthContext, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeRepository := new(mockResume.MockResumeReposity)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockRedisTaskPublisher := new(queue.MockRedisTaskPublisher)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							ID:     userID,
							UserID: userID.String(),
						},
					}, nil)

				defaultResume := db.Resumes{
					ID: uuid.New(),
				}

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s:%s", constants.RedisPrefixDefaultResume, userID.String())).
					Return(`{"id":"`+defaultResume.ID.String()+`","file_name":"default.pdf","mime_type":"application/pdf","byte_size":1024,"is_default":true,"created_at":"2023-01-01T00:00:00Z","updated_at":"2023-01-01T00:00:00Z"}`, nil)

				mockResumeRepository.EXPECT().
					ListResumeByUserIDFirstPage(ctx, userID).
					Return([]db.Resumes{}, errors.New("database error"))

				return mockResumeRepository, mockAuthContext, mockRedisTaskPublisher, mockRedisClient
			},
			verify: func(t *testing.T, gotResp *entities.ListResumeResponse, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
				assert.Equal(t, "database error", gotErr.Error())
			},
		},
		{
			name: "Error - Redis hit but fetch list Resume Paginated failed",
			input: &entities.ListResumeRequest{
				UpdatedAt: func() *time.Time {
					t := time.Now()
					return &t
				}(),
			},
			setup: func() (*mockResume.MockResumeReposity, *mockMiddleware.MockAuthContext, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeRepository := new(mockResume.MockResumeReposity)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockRedisTaskPublisher := new(queue.MockRedisTaskPublisher)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							ID:     userID,
							UserID: userID.String(),
						},
					}, nil)

				defaultResume := db.Resumes{
					ID: uuid.New(),
				}

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s:%s", constants.RedisPrefixDefaultResume, userID.String())).
					Return(`{"id":"`+defaultResume.ID.String()+`","file_name":"default.pdf","mime_type":"application/pdf","byte_size":1024,"is_default":true,"created_at":"2023-01-01T00:00:00Z","updated_at":"2023-01-01T00:00:00Z"}`, nil)

				mockResumeRepository.EXPECT().
					ListResumeByUserIDPaginated(ctx, mock.AnythingOfType("*db.ListResumeByUserIDPaginatedParams")).
					Return([]db.Resumes{}, errors.New("database error"))

				return mockResumeRepository, mockAuthContext, mockRedisTaskPublisher, mockRedisClient
			},
			verify: func(t *testing.T, gotResp *entities.ListResumeResponse, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
				assert.Equal(t, "database error", gotErr.Error())
			},
		},
		{
			name:  "Success - Redis miss, fetch both concurrently",
			input: &entities.ListResumeRequest{},
			setup: func() (*mockResume.MockResumeReposity, *mockMiddleware.MockAuthContext, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeRepository := new(mockResume.MockResumeReposity)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockRedisTaskPublisher := new(queue.MockRedisTaskPublisher)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							ID:     userID,
							UserID: userID.String(),
						},
					}, nil)

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s:%s", constants.RedisPrefixDefaultResume, userID.String())).
					Return("", redis.Nil)

				defaultResume := db.Resumes{
					ID:        uuid.New(),
					UserID:    userID,
					FileName:  "default.pdf",
					MimeType:  "application/pdf",
					ByteSize:  1024,
					IsDefault: true,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}

				mockResumeRepository.EXPECT().
					ListResumeByUserIDFirstPage(ctx, userID).
					Return([]db.Resumes{}, nil)

				mockResumeRepository.EXPECT().
					GetDefaultResumeByUserID(ctx, userID).
					Return(defaultResume, nil)

				mockRedisClient.EXPECT().
					Set(ctx, mock.AnythingOfType("database.RedisPayload")).
					Return(nil)

				return mockResumeRepository, mockAuthContext, mockRedisTaskPublisher, mockRedisClient
			},
			verify: func(t *testing.T, gotResp *entities.ListResumeResponse, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, gotResp)
				assert.Equal(t, 1, gotResp.Count)
				assert.Equal(t, "default.pdf", gotResp.ResumeContent.DefaultResume.FileName)
				assert.Equal(t, 0, len(gotResp.ResumeContent.Resumes))
			},
		},
		{
			name:  "Success - Redis miss, concurrent operations complete successfully",
			input: &entities.ListResumeRequest{},
			setup: func() (*mockResume.MockResumeReposity, *mockMiddleware.MockAuthContext, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeRepository := new(mockResume.MockResumeReposity)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockRedisTaskPublisher := new(queue.MockRedisTaskPublisher)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							ID:     userID,
							UserID: userID.String(),
						},
					}, nil)

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s:%s", constants.RedisPrefixDefaultResume, userID.String())).
					Return("", redis.Nil)

				mockResumeRepository.EXPECT().
					ListResumeByUserIDFirstPage(ctx, userID).
					RunAndReturn(func(ctx context.Context, userID uuid.UUID) ([]db.Resumes, error) {
						time.Sleep(10 * time.Millisecond)
						return []db.Resumes{}, nil
					})

				mockResumeRepository.EXPECT().
					GetDefaultResumeByUserID(ctx, userID).
					RunAndReturn(func(ctx context.Context, userID uuid.UUID) (db.Resumes, error) {
						time.Sleep(50 * time.Millisecond)
						return db.Resumes{
							ID:        uuid.New(),
							UserID:    userID,
							FileName:  "default.pdf",
							MimeType:  "application/pdf",
							ByteSize:  1024,
							IsDefault: true,
							CreatedAt: time.Now(),
							UpdatedAt: time.Now(),
						}, nil
					})

				mockRedisClient.EXPECT().
					Set(ctx, mock.AnythingOfType("database.RedisPayload")).
					Return(nil)

				return mockResumeRepository, mockAuthContext, mockRedisTaskPublisher, mockRedisClient
			},
			verify: func(t *testing.T, gotResp *entities.ListResumeResponse, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, gotResp)
				assert.Equal(t, "default.pdf", gotResp.ResumeContent.DefaultResume.FileName)
				assert.Equal(t, 0, len(gotResp.ResumeContent.Resumes))
			},
		},
		{
			name:  "Error - Context canceled during concurrent operations",
			input: &entities.ListResumeRequest{},
			setup: func() (*mockResume.MockResumeReposity, *mockMiddleware.MockAuthContext, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeRepository := new(mockResume.MockResumeReposity)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockRedisTaskPublisher := new(queue.MockRedisTaskPublisher)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockAuthContext.EXPECT().
					GetAuthContext(mock.AnythingOfType("*context.timerCtx")).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							ID:     userID,
							UserID: userID.String(),
						},
					}, nil)

				mockRedisClient.EXPECT().
					Get(mock.AnythingOfType("*context.timerCtx"), fmt.Sprintf("%s:%s", constants.RedisPrefixDefaultResume, userID.String())).
					Return("", redis.Nil)

				mockResumeRepository.EXPECT().
					ListResumeByUserIDFirstPage(mock.AnythingOfType("*context.timerCtx"), userID).
					RunAndReturn(func(ctx context.Context, userID uuid.UUID) ([]db.Resumes, error) {
						time.Sleep(10 * time.Millisecond)
						return []db.Resumes{}, nil
					})

				mockResumeRepository.EXPECT().
					GetDefaultResumeByUserID(mock.AnythingOfType("*context.timerCtx"), userID).
					RunAndReturn(func(ctx context.Context, userID uuid.UUID) (db.Resumes, error) {
						time.Sleep(100 * time.Millisecond)
						return db.Resumes{
							ID:        uuid.New(),
							UserID:    userID,
							FileName:  "default.pdf",
							MimeType:  "application/pdf",
							ByteSize:  1024,
							IsDefault: true,
							CreatedAt: time.Now(),
							UpdatedAt: time.Now(),
						}, nil
					})

				return mockResumeRepository, mockAuthContext, mockRedisTaskPublisher, mockRedisClient
			},
			verify: func(t *testing.T, gotResp *entities.ListResumeResponse, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
				assert.Contains(t, gotErr.Error(), "context deadline exceeded")
			},
		},
		{
			name:  "Error - Auth context failure",
			input: &entities.ListResumeRequest{},
			setup: func() (*mockResume.MockResumeReposity, *mockMiddleware.MockAuthContext, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeRepository := new(mockResume.MockResumeReposity)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockRedisTaskPublisher := new(queue.MockRedisTaskPublisher)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(nil, errors.New("auth failed"))

				return mockResumeRepository, mockAuthContext, mockRedisTaskPublisher, mockRedisClient
			},
			verify: func(t *testing.T, gotResp *entities.ListResumeResponse, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
				assert.Equal(t, "auth failed", gotErr.Error())
			},
		},
		{
			name:  "Error - Redis error",
			input: &entities.ListResumeRequest{},
			setup: func() (*mockResume.MockResumeReposity, *mockMiddleware.MockAuthContext, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeRepository := new(mockResume.MockResumeReposity)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockRedisTaskPublisher := new(queue.MockRedisTaskPublisher)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							ID:     userID,
							UserID: userID.String(),
						},
					}, nil)

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s:%s", constants.RedisPrefixDefaultResume, userID.String())).
					Return("", errors.New("redis error"))

				return mockResumeRepository, mockAuthContext, mockRedisTaskPublisher, mockRedisClient
			},
			verify: func(t *testing.T, gotResp *entities.ListResumeResponse, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
				assert.Equal(t, "redis error", gotErr.Error())
			},
		},
		{
			name:  "Error - Redis unmarshal failure (fallback to FetchBoth)",
			input: &entities.ListResumeRequest{},
			setup: func() (*mockResume.MockResumeReposity, *mockMiddleware.MockAuthContext, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeRepository := new(mockResume.MockResumeReposity)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockRedisTaskPublisher := new(queue.MockRedisTaskPublisher)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							ID:     userID,
							UserID: userID.String(),
						},
					}, nil)

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s:%s", constants.RedisPrefixDefaultResume, userID.String())).
					Return("invalid json", nil)

				defaultResume := db.Resumes{
					ID:        uuid.New(),
					UserID:    userID,
					FileName:  "default.pdf",
					MimeType:  "application/pdf",
					ByteSize:  1024,
					IsDefault: true,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}

				mockResumeRepository.EXPECT().
					ListResumeByUserIDFirstPage(ctx, userID).
					Return([]db.Resumes{}, nil)

				mockResumeRepository.EXPECT().
					GetDefaultResumeByUserID(ctx, userID).
					Return(defaultResume, nil)

				mockRedisClient.EXPECT().
					Set(ctx, mock.AnythingOfType("database.RedisPayload")).
					Return(nil)

				return mockResumeRepository, mockAuthContext, mockRedisTaskPublisher, mockRedisClient
			},
			verify: func(t *testing.T, gotResp *entities.ListResumeResponse, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, gotResp)
				assert.Equal(t, "default.pdf", gotResp.ResumeContent.DefaultResume.FileName)
				assert.Equal(t, 0, len(gotResp.ResumeContent.Resumes))
			},
		},
		{
			name:  "Error - Resume list fetch failure",
			input: &entities.ListResumeRequest{},
			setup: func() (*mockResume.MockResumeReposity, *mockMiddleware.MockAuthContext, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeRepository := new(mockResume.MockResumeReposity)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockRedisTaskPublisher := new(queue.MockRedisTaskPublisher)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							ID:     userID,
							UserID: userID.String(),
						},
					}, nil)

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s:%s", constants.RedisPrefixDefaultResume, userID.String())).
					Return("", redis.Nil)

				mockResumeRepository.EXPECT().
					ListResumeByUserIDFirstPage(ctx, userID).
					Return(nil, errors.New("database error"))

				mockResumeRepository.EXPECT().
					GetDefaultResumeByUserID(ctx, userID).
					Return(db.Resumes{}, nil)

				return mockResumeRepository, mockAuthContext, mockRedisTaskPublisher, mockRedisClient
			},
			verify: func(t *testing.T, gotResp *entities.ListResumeResponse, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
				assert.Equal(t, "database error", gotErr.Error())
			},
		},
		{
			name:  "Error_DefaultResumeFetchFailure_CausesServiceFailure",
			input: &entities.ListResumeRequest{},
			setup: func() (*mockResume.MockResumeReposity, *mockMiddleware.MockAuthContext, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeRepository := new(mockResume.MockResumeReposity)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockRedisTaskPublisher := new(queue.MockRedisTaskPublisher)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							ID:     userID,
							UserID: userID.String(),
						},
					}, nil)

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s:%s", constants.RedisPrefixDefaultResume, userID.String())).
					Return("", redis.Nil)

				mockResumeRepository.EXPECT().
					ListResumeByUserIDFirstPage(ctx, userID).
					Return([]db.Resumes{}, nil)

				mockResumeRepository.EXPECT().
					GetDefaultResumeByUserID(ctx, userID).
					Return(db.Resumes{}, errors.New("default resume not found"))

				return mockResumeRepository, mockAuthContext, mockRedisTaskPublisher, mockRedisClient
			},
			verify: func(t *testing.T, gotResp *entities.ListResumeResponse, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
				assert.Equal(t, "default resume not found", gotErr.Error())
			},
		},
		{
			name:  "Error - Redis marshalling failure",
			input: &entities.ListResumeRequest{},
			setup: func() (*mockResume.MockResumeReposity, *mockMiddleware.MockAuthContext, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeRepository := new(mockResume.MockResumeReposity)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockRedisTaskPublisher := new(queue.MockRedisTaskPublisher)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							ID:     userID,
							UserID: userID.String(),
						},
					}, nil)

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s:%s", constants.RedisPrefixDefaultResume, userID.String())).
					Return("", redis.Nil)

				defaultResume := db.Resumes{
					ID:        uuid.New(),
					UserID:    userID,
					FileName:  "default.pdf",
					MimeType:  "application/pdf",
					ByteSize:  1024,
					IsDefault: true,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}

				mockResumeRepository.EXPECT().
					ListResumeByUserIDFirstPage(ctx, userID).
					Return([]db.Resumes{}, nil)

				mockResumeRepository.EXPECT().
					GetDefaultResumeByUserID(ctx, userID).
					Return(defaultResume, nil)

				// Redis Set won't be called due to marshalling failure
				// But the service should still return success
				// Actually, the service still tries to call Redis.Set, so we need to mock it
				mockRedisClient.EXPECT().
					Set(ctx, mock.AnythingOfType("database.RedisPayload")).
					Return(nil)

				return mockResumeRepository, mockAuthContext, mockRedisTaskPublisher, mockRedisClient
			},
			verify: func(t *testing.T, gotResp *entities.ListResumeResponse, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, gotResp)
				assert.Equal(t, "default.pdf", gotResp.ResumeContent.DefaultResume.FileName)
				assert.Equal(t, 0, len(gotResp.ResumeContent.Resumes))
			},
		},
		{
			name:  "Error - Redis caching failure",
			input: &entities.ListResumeRequest{},
			setup: func() (*mockResume.MockResumeReposity, *mockMiddleware.MockAuthContext, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeRepository := new(mockResume.MockResumeReposity)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockRedisTaskPublisher := new(queue.MockRedisTaskPublisher)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							ID:     userID,
							UserID: userID.String(),
						},
					}, nil)

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s:%s", constants.RedisPrefixDefaultResume, userID.String())).
					Return("", redis.Nil)

				defaultResume := db.Resumes{
					ID:        uuid.New(),
					UserID:    userID,
					FileName:  "default.pdf",
					MimeType:  "application/pdf",
					ByteSize:  1024,
					IsDefault: true,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}

				mockResumeRepository.EXPECT().
					ListResumeByUserIDFirstPage(ctx, userID).
					Return([]db.Resumes{}, nil)

				mockResumeRepository.EXPECT().
					GetDefaultResumeByUserID(ctx, userID).
					Return(defaultResume, nil)

				mockRedisClient.EXPECT().
					Set(ctx, mock.AnythingOfType("database.RedisPayload")).
					Return(errors.New("redis set failed"))

				return mockResumeRepository, mockAuthContext, mockRedisTaskPublisher, mockRedisClient
			},
			verify: func(t *testing.T, gotResp *entities.ListResumeResponse, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, gotResp)
				assert.Equal(t, "default.pdf", gotResp.ResumeContent.DefaultResume.FileName)
				assert.Equal(t, 0, len(gotResp.ResumeContent.Resumes))
			},
		},
		{
			name: "Success - With pagination",
			input: &entities.ListResumeRequest{
				UpdatedAt: &time.Time{},
			},
			setup: func() (*mockResume.MockResumeReposity, *mockMiddleware.MockAuthContext, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeRepository := new(mockResume.MockResumeReposity)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockRedisTaskPublisher := new(queue.MockRedisTaskPublisher)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							ID:     userID,
							UserID: userID.String(),
						},
					}, nil)

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s:%s", constants.RedisPrefixDefaultResume, userID.String())).
					Return("", redis.Nil)

				defaultResume := db.Resumes{
					ID:        uuid.New(),
					UserID:    userID,
					FileName:  "default.pdf",
					MimeType:  "application/pdf",
					ByteSize:  1024,
					IsDefault: true,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}

				resumeResponse := []db.Resumes{
					{
						ID:        uuid.New(),
						UserID:    userID,
						FileName:  "resume1.pdf",
						MimeType:  "application/pdf",
						ByteSize:  1024,
						IsDefault: false,
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
				}

				mockResumeRepository.EXPECT().
					ListResumeByUserIDPaginated(ctx, mock.AnythingOfType("*db.ListResumeByUserIDPaginatedParams")).
					Return(resumeResponse, nil)

				mockResumeRepository.EXPECT().
					GetDefaultResumeByUserID(ctx, userID).
					Return(defaultResume, nil)

				mockRedisClient.EXPECT().
					Set(ctx, mock.AnythingOfType("database.RedisPayload")).
					Return(nil)

				return mockResumeRepository, mockAuthContext, mockRedisTaskPublisher, mockRedisClient
			},
			verify: func(t *testing.T, gotResp *entities.ListResumeResponse, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, gotResp)
				assert.Equal(t, "default.pdf", gotResp.ResumeContent.DefaultResume.FileName)
				assert.Equal(t, 1, len(gotResp.ResumeContent.Resumes))
			},
		},
		{
			name:  "Success - LastUpdatedAt logic: Only default resume exists",
			input: &entities.ListResumeRequest{},
			setup: func() (*mockResume.MockResumeReposity, *mockMiddleware.MockAuthContext, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeRepository := new(mockResume.MockResumeReposity)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockRedisTaskPublisher := new(queue.MockRedisTaskPublisher)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							ID:     userID,
							UserID: userID.String(),
						},
					}, nil)

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s:%s", constants.RedisPrefixDefaultResume, userID.String())).
					Return("", redis.Nil)

				defaultResume := db.Resumes{
					ID:        uuid.New(),
					UserID:    userID,
					FileName:  "default.pdf",
					MimeType:  "application/pdf",
					ByteSize:  1024,
					IsDefault: true,
					CreatedAt: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
					UpdatedAt: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
				}

				mockResumeRepository.EXPECT().
					ListResumeByUserIDFirstPage(ctx, userID).
					Return([]db.Resumes{}, nil)

				mockResumeRepository.EXPECT().
					GetDefaultResumeByUserID(ctx, userID).
					Return(defaultResume, nil)

				mockRedisClient.EXPECT().
					Set(ctx, mock.AnythingOfType("database.RedisPayload")).
					Return(nil)

				return mockResumeRepository, mockAuthContext, mockRedisTaskPublisher, mockRedisClient
			},
			verify: func(t *testing.T, gotResp *entities.ListResumeResponse, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, gotResp)
				assert.Equal(t, 1, gotResp.Count)
				assert.NotNil(t, gotResp.LastUpdatedAt)
				assert.Equal(t, "2024-01-15T10:30:00Z", *gotResp.LastUpdatedAt)
			},
		},
		{
			name:  "Success - LastUpdatedAt logic: Only regular resumes exist (no default)",
			input: &entities.ListResumeRequest{},
			setup: func() (*mockResume.MockResumeReposity, *mockMiddleware.MockAuthContext, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeRepository := new(mockResume.MockResumeReposity)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockRedisTaskPublisher := new(queue.MockRedisTaskPublisher)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							ID:     userID,
							UserID: userID.String(),
						},
					}, nil)

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s:%s", constants.RedisPrefixDefaultResume, userID.String())).
					Return("", redis.Nil)

				// No default resume
				emptyDefaultResume := db.Resumes{}

				resumeList := []db.Resumes{
					{
						ID:        uuid.New(),
						UserID:    userID,
						FileName:  "resume1.pdf",
						MimeType:  "application/pdf",
						ByteSize:  1024,
						IsDefault: false,
						CreatedAt: time.Date(2024, 1, 15, 8, 0, 0, 0, time.UTC),
						UpdatedAt: time.Date(2024, 1, 15, 8, 0, 0, 0, time.UTC),
					},
					{
						ID:        uuid.New(),
						UserID:    userID,
						FileName:  "resume2.pdf",
						MimeType:  "application/pdf",
						ByteSize:  2048,
						IsDefault: false,
						CreatedAt: time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC),
						UpdatedAt: time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC),
					},
				}

				mockResumeRepository.EXPECT().
					ListResumeByUserIDFirstPage(ctx, userID).
					Return(resumeList, nil)

				mockResumeRepository.EXPECT().
					GetDefaultResumeByUserID(ctx, userID).
					Return(emptyDefaultResume, sql.ErrNoRows)

				return mockResumeRepository, mockAuthContext, mockRedisTaskPublisher, mockRedisClient
			},
			verify: func(t *testing.T, gotResp *entities.ListResumeResponse, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, gotResp)
				assert.Equal(t, 2, gotResp.Count)
				assert.NotNil(t, gotResp.LastUpdatedAt)
				// Should be the most recent regular resume (resume2 at 9:00 AM)
				assert.Equal(t, "2024-01-15T09:00:00Z", *gotResp.LastUpdatedAt)
			},
		},
		{
			name:  "Success - LastUpdatedAt logic: Both default and regular resumes exist, default is more recent",
			input: &entities.ListResumeRequest{},
			setup: func() (*mockResume.MockResumeReposity, *mockMiddleware.MockAuthContext, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeRepository := new(mockResume.MockResumeReposity)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockRedisTaskPublisher := new(queue.MockRedisTaskPublisher)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							ID:     userID,
							UserID: userID.String(),
						},
					}, nil)

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s:%s", constants.RedisPrefixDefaultResume, userID.String())).
					Return("", redis.Nil)

				defaultResume := db.Resumes{
					ID:        uuid.New(),
					UserID:    userID,
					FileName:  "default.pdf",
					MimeType:  "application/pdf",
					ByteSize:  1024,
					IsDefault: true,
					CreatedAt: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
					UpdatedAt: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC), // Most recent
				}

				resumeList := []db.Resumes{
					{
						ID:        uuid.New(),
						UserID:    userID,
						FileName:  "resume1.pdf",
						MimeType:  "application/pdf",
						ByteSize:  1024,
						IsDefault: false,
						CreatedAt: time.Date(2024, 1, 15, 8, 0, 0, 0, time.UTC),
						UpdatedAt: time.Date(2024, 1, 15, 8, 0, 0, 0, time.UTC),
					},
					{
						ID:        uuid.New(),
						UserID:    userID,
						FileName:  "resume2.pdf",
						MimeType:  "application/pdf",
						ByteSize:  2048,
						IsDefault: false,
						CreatedAt: time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC),
						UpdatedAt: time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC),
					},
				}

				mockResumeRepository.EXPECT().
					ListResumeByUserIDFirstPage(ctx, userID).
					Return(resumeList, nil)

				mockResumeRepository.EXPECT().
					GetDefaultResumeByUserID(ctx, userID).
					Return(defaultResume, nil)

				mockRedisClient.EXPECT().
					Set(ctx, mock.AnythingOfType("database.RedisPayload")).
					Return(nil)

				return mockResumeRepository, mockAuthContext, mockRedisTaskPublisher, mockRedisClient
			},
			verify: func(t *testing.T, gotResp *entities.ListResumeResponse, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, gotResp)
				assert.Equal(t, 3, gotResp.Count)
				assert.NotNil(t, gotResp.LastUpdatedAt)
				// Should be the default resume since it's more recent (10:30 AM)
				assert.Equal(t, "2024-01-15T10:30:00Z", *gotResp.LastUpdatedAt)
			},
		},
		{
			name:  "Success - LastUpdatedAt logic: Both default and regular resumes exist, regular resume is more recent",
			input: &entities.ListResumeRequest{},
			setup: func() (*mockResume.MockResumeReposity, *mockMiddleware.MockAuthContext, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeRepository := new(mockResume.MockResumeReposity)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockRedisTaskPublisher := new(queue.MockRedisTaskPublisher)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							ID:     userID,
							UserID: userID.String(),
						},
					}, nil)

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s:%s", constants.RedisPrefixDefaultResume, userID.String())).
					Return("", redis.Nil)

				defaultResume := db.Resumes{
					ID:        uuid.New(),
					UserID:    userID,
					FileName:  "default.pdf",
					MimeType:  "application/pdf",
					ByteSize:  1024,
					IsDefault: true,
					CreatedAt: time.Date(2024, 1, 15, 8, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2024, 1, 15, 8, 0, 0, 0, time.UTC),
				}

				resumeList := []db.Resumes{
					{
						ID:        uuid.New(),
						UserID:    userID,
						FileName:  "resume1.pdf",
						MimeType:  "application/pdf",
						ByteSize:  1024,
						IsDefault: false,
						CreatedAt: time.Date(2024, 1, 15, 7, 0, 0, 0, time.UTC),
						UpdatedAt: time.Date(2024, 1, 15, 7, 0, 0, 0, time.UTC),
					},
					{
						ID:        uuid.New(),
						UserID:    userID,
						FileName:  "resume2.pdf",
						MimeType:  "application/pdf",
						ByteSize:  2048,
						IsDefault: false,
						CreatedAt: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC), // Most recent
						UpdatedAt: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC), // Most recent
					},
				}

				mockResumeRepository.EXPECT().
					ListResumeByUserIDFirstPage(ctx, userID).
					Return(resumeList, nil)

				mockResumeRepository.EXPECT().
					GetDefaultResumeByUserID(ctx, userID).
					Return(defaultResume, nil)

				mockRedisClient.EXPECT().
					Set(ctx, mock.AnythingOfType("database.RedisPayload")).
					Return(nil)

				return mockResumeRepository, mockAuthContext, mockRedisTaskPublisher, mockRedisClient
			},
			verify: func(t *testing.T, gotResp *entities.ListResumeResponse, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, gotResp)
				assert.Equal(t, 3, gotResp.Count)
				assert.NotNil(t, gotResp.LastUpdatedAt)
				// Should be the regular resume since it's more recent (10:30 AM)
				assert.Equal(t, "2024-01-15T10:30:00Z", *gotResp.LastUpdatedAt)
			},
		},
		{
			name:  "Success - LastUpdatedAt logic: No resumes exist at all",
			input: &entities.ListResumeRequest{},
			setup: func() (*mockResume.MockResumeReposity, *mockMiddleware.MockAuthContext, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeRepository := new(mockResume.MockResumeReposity)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockRedisTaskPublisher := new(queue.MockRedisTaskPublisher)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							ID:     userID,
							UserID: userID.String(),
						},
					}, nil)

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s:%s", constants.RedisPrefixDefaultResume, userID.String())).
					Return("", redis.Nil)

				// No default resume
				emptyDefaultResume := db.Resumes{}

				// No regular resumes
				emptyResumeList := []db.Resumes{}

				mockResumeRepository.EXPECT().
					ListResumeByUserIDFirstPage(ctx, userID).
					Return(emptyResumeList, nil)

				mockResumeRepository.EXPECT().
					GetDefaultResumeByUserID(ctx, userID).
					Return(emptyDefaultResume, sql.ErrNoRows)

				return mockResumeRepository, mockAuthContext, mockRedisTaskPublisher, mockRedisClient
			},
			verify: func(t *testing.T, gotResp *entities.ListResumeResponse, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, gotResp)
				assert.Equal(t, 0, gotResp.Count)
				assert.Nil(t, gotResp.LastUpdatedAt) // Should be nil when no resumes exist
				assert.Nil(t, gotResp.ResumeContent) // Should be nil when no resumes exist
			},
		},
		{
			name:  "Success - LastUpdatedAt logic: Multiple regular resumes, find the most recent",
			input: &entities.ListResumeRequest{},
			setup: func() (*mockResume.MockResumeReposity, *mockMiddleware.MockAuthContext, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeRepository := new(mockResume.MockResumeReposity)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockRedisTaskPublisher := new(queue.MockRedisTaskPublisher)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							ID:     userID,
							UserID: userID.String(),
						},
					}, nil)

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s:%s", constants.RedisPrefixDefaultResume, userID.String())).
					Return("", redis.Nil)

				// No default resume
				emptyDefaultResume := db.Resumes{}

				resumeList := []db.Resumes{
					{
						ID:        uuid.New(),
						UserID:    userID,
						FileName:  "resume1.pdf",
						MimeType:  "application/pdf",
						ByteSize:  1024,
						IsDefault: false,
						CreatedAt: time.Date(2024, 1, 15, 6, 0, 0, 0, time.UTC),
						UpdatedAt: time.Date(2024, 1, 15, 6, 0, 0, 0, time.UTC),
					},
					{
						ID:        uuid.New(),
						UserID:    userID,
						FileName:  "resume2.pdf",
						MimeType:  "application/pdf",
						ByteSize:  2048,
						IsDefault: false,
						CreatedAt: time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC),
						UpdatedAt: time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC),
					},
					{
						ID:        uuid.New(),
						UserID:    userID,
						FileName:  "resume3.pdf",
						MimeType:  "application/pdf",
						ByteSize:  3072,
						IsDefault: false,
						CreatedAt: time.Date(2024, 1, 15, 11, 0, 0, 0, time.UTC), // Most recent
						UpdatedAt: time.Date(2024, 1, 15, 11, 0, 0, 0, time.UTC), // Most recent
					},
					{
						ID:        uuid.New(),
						UserID:    userID,
						FileName:  "resume4.pdf",
						MimeType:  "application/pdf",
						ByteSize:  4096,
						IsDefault: false,
						CreatedAt: time.Date(2024, 1, 15, 7, 0, 0, 0, time.UTC),
						UpdatedAt: time.Date(2024, 1, 15, 7, 0, 0, 0, time.UTC),
					},
				}

				mockResumeRepository.EXPECT().
					ListResumeByUserIDFirstPage(ctx, userID).
					Return(resumeList, nil)

				mockResumeRepository.EXPECT().
					GetDefaultResumeByUserID(ctx, userID).
					Return(emptyDefaultResume, sql.ErrNoRows)

				return mockResumeRepository, mockAuthContext, mockRedisTaskPublisher, mockRedisClient
			},
			verify: func(t *testing.T, gotResp *entities.ListResumeResponse, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, gotResp)
				assert.Equal(t, 4, gotResp.Count)
				assert.NotNil(t, gotResp.LastUpdatedAt)
				// Should be the most recent regular resume (resume3 at 11:00 AM)
				assert.Equal(t, "2024-01-15T11:00:00Z", *gotResp.LastUpdatedAt)
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			mockResumeRepository, mockAuthContext, mockRedisTaskPublisher, mockRedisClient := tC.setup()
			defer func() {
				if mockResumeRepository != nil {
					mockResumeRepository.AssertExpectations(t)
				}
				if mockAuthContext != nil {
					mockAuthContext.AssertExpectations(t)
				}
				if mockRedisTaskPublisher != nil {
					mockRedisTaskPublisher.AssertExpectations(t)
				}
				if mockRedisClient != nil {
					mockRedisClient.AssertExpectations(t)
				}
			}()

			svc := NewResumeService(lgr, mockResumeRepository, mockAuthContext, nil, nil, mockRedisTaskPublisher, mockRedisClient, nil)

			var testCtx context.Context
			var cancel context.CancelFunc
			if tC.name == "Error - Context canceled during concurrent operations" {
				testCtx, cancel = context.WithTimeout(ctx, 50*time.Millisecond)
				defer cancel()
			} else {
				testCtx = ctx
			}

			gotResp, gotErr := svc.ListResume(testCtx, tC.input)

			tC.verify(t, gotResp, gotErr)
		})
	}
}

func TestResumeService_SwitchDefaultResume(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()
	userID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	resumeID := uuid.New()

	testCases := []struct {
		name   string
		input  *entities.SwitchDefaultResumeRequest
		setup  func() (*mockResume.MockResumeReposity, *mockMiddleware.MockAuthContext, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient)
		verify func(t *testing.T, gotErr error)
	}{
		{
			name: "Success - Redis hit",
			input: &entities.SwitchDefaultResumeRequest{
				ResumeID: resumeID.String(),
			},
			setup: func() (*mockResume.MockResumeReposity, *mockMiddleware.MockAuthContext, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeRepository := new(mockResume.MockResumeReposity)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockRedisTaskPublisher := new(queue.MockRedisTaskPublisher)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							ID:     userID,
							UserID: userID.String(),
						},
					}, nil)

				defaultResume := db.Resumes{
					ID: uuid.New(),
				}

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s:%s", constants.RedisPrefixDefaultResume, userID.String())).
					Return(`{"id":"`+defaultResume.ID.String()+`","file_name":"default.pdf","mime_type":"application/pdf","byte_size":1024,"is_default":true,"created_at":"2023-01-01T00:00:00Z","updated_at":"2023-01-01T00:00:00Z"}`, nil)

				mockResumeRepository.EXPECT().
					SwitchDefaultResume(ctx, defaultResume.ID, resumeID).
					Return(nil)

				mockRedisTaskPublisher.EXPECT().
					PublishTaskDeleteRedis(ctx, mock.AnythingOfType("*database.RedisDeletePayload")).
					Return(nil)

				return mockResumeRepository, mockAuthContext, mockRedisTaskPublisher, mockRedisClient
			},
			verify: func(t *testing.T, gotErr error) {
				assert.NoError(t, gotErr)
			},
		},
		{
			name: "Error - Redis Error",
			input: &entities.SwitchDefaultResumeRequest{
				ResumeID: resumeID.String(),
			},
			setup: func() (*mockResume.MockResumeReposity, *mockMiddleware.MockAuthContext, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeRepository := new(mockResume.MockResumeReposity)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockRedisTaskPublisher := new(queue.MockRedisTaskPublisher)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							ID:     userID,
							UserID: userID.String(),
						},
					}, nil)

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s:%s", constants.RedisPrefixDefaultResume, userID.String())).
					Return("", errors.New("redis error"))

				return mockResumeRepository, mockAuthContext, mockRedisTaskPublisher, mockRedisClient
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Equal(t, "redis error", gotErr.Error())
			},
		},
		{
			name: "Error_RedisHit_Invalid Resume ID",
			input: &entities.SwitchDefaultResumeRequest{
				ResumeID: "invalid-resume-id",
			},
			setup: func() (*mockResume.MockResumeReposity, *mockMiddleware.MockAuthContext, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeRepository := new(mockResume.MockResumeReposity)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockRedisTaskPublisher := new(queue.MockRedisTaskPublisher)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							ID:     userID,
							UserID: userID.String(),
						},
					}, nil)

				defaultResume := db.Resumes{
					ID: uuid.New(),
				}

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s:%s", constants.RedisPrefixDefaultResume, userID.String())).
					Return(`{"id":"`+defaultResume.ID.String()+`","file_name":"default.pdf","mime_type":"application/pdf","byte_size":1024,"is_default":true,"created_at":"2023-01-01T00:00:00Z","updated_at":"2023-01-01T00:00:00Z"}`, nil)

				return mockResumeRepository, mockAuthContext, mockRedisTaskPublisher, mockRedisClient
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "The UUID is invalid. Please try again.")
				assert.Contains(t, gotErr.Error(), "[ONX0107]")
			},
		},
		{
			name: "Success - Redis miss, fetch from DB",
			input: &entities.SwitchDefaultResumeRequest{
				ResumeID: resumeID.String(),
			},
			setup: func() (*mockResume.MockResumeReposity, *mockMiddleware.MockAuthContext, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeRepository := new(mockResume.MockResumeReposity)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockRedisTaskPublisher := new(queue.MockRedisTaskPublisher)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							ID:     userID,
							UserID: userID.String(),
						},
					}, nil)

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s:%s", constants.RedisPrefixDefaultResume, userID.String())).
					Return("", redis.Nil)

				defaultResume := db.Resumes{
					ID:        uuid.New(),
					UserID:    userID,
					FileName:  "default.pdf",
					MimeType:  "application/pdf",
					ByteSize:  1024,
					IsDefault: true,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}

				mockResumeRepository.EXPECT().
					GetDefaultResumeByUserID(ctx, userID).
					Return(defaultResume, nil)

				mockResumeRepository.EXPECT().
					SwitchDefaultResume(ctx, defaultResume.ID, resumeID).
					Return(nil)

				mockRedisTaskPublisher.EXPECT().
					PublishTaskDeleteRedis(ctx, mock.AnythingOfType("*database.RedisDeletePayload")).
					Return(nil)

				return mockResumeRepository, mockAuthContext, mockRedisTaskPublisher, mockRedisClient
			},
			verify: func(t *testing.T, gotErr error) {
				assert.NoError(t, gotErr)
			},
		},
		{
			name: "Error - Auth context failure",
			input: &entities.SwitchDefaultResumeRequest{
				ResumeID: resumeID.String(),
			},
			setup: func() (*mockResume.MockResumeReposity, *mockMiddleware.MockAuthContext, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeRepository := new(mockResume.MockResumeReposity)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockRedisTaskPublisher := new(queue.MockRedisTaskPublisher)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(nil, errors.New("auth failed"))

				return mockResumeRepository, mockAuthContext, mockRedisTaskPublisher, mockRedisClient
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Equal(t, "auth failed", gotErr.Error())
			},
		},
		{
			name: "Error_RedisMiss_Invalid user ID",
			input: &entities.SwitchDefaultResumeRequest{
				ResumeID: resumeID.String(),
			},
			setup: func() (*mockResume.MockResumeReposity, *mockMiddleware.MockAuthContext, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeRepository := new(mockResume.MockResumeReposity)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockRedisTaskPublisher := new(queue.MockRedisTaskPublisher)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				invalidUserID := "invalid-user-id"
				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							ID:     userID,
							UserID: invalidUserID,
						},
					}, nil)

				return mockResumeRepository, mockAuthContext, mockRedisTaskPublisher, mockRedisClient
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "The UUID is invalid. Please try again.")
				assert.Contains(t, gotErr.Error(), "[ONX0107]")
			},
		},
		{
			name: "Error - Get default resume failure",
			input: &entities.SwitchDefaultResumeRequest{
				ResumeID: resumeID.String(),
			},
			setup: func() (*mockResume.MockResumeReposity, *mockMiddleware.MockAuthContext, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeRepository := new(mockResume.MockResumeReposity)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockRedisTaskPublisher := new(queue.MockRedisTaskPublisher)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							ID:     userID,
							UserID: userID.String(),
						},
					}, nil)

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s:%s", constants.RedisPrefixDefaultResume, userID.String())).
					Return("", redis.Nil)

				mockResumeRepository.EXPECT().
					GetDefaultResumeByUserID(ctx, userID).
					Return(db.Resumes{}, errors.New("database error"))

				return mockResumeRepository, mockAuthContext, mockRedisTaskPublisher, mockRedisClient
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Equal(t, "database error", gotErr.Error())
			},
		},
		{
			name: "Error - Switch default resume failure",
			input: &entities.SwitchDefaultResumeRequest{
				ResumeID: resumeID.String(),
			},
			setup: func() (*mockResume.MockResumeReposity, *mockMiddleware.MockAuthContext, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeRepository := new(mockResume.MockResumeReposity)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockRedisTaskPublisher := new(queue.MockRedisTaskPublisher)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							ID:     userID,
							UserID: userID.String(),
						},
					}, nil)

				defaultResume := db.Resumes{
					ID: uuid.New(),
				}

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s:%s", constants.RedisPrefixDefaultResume, userID.String())).
					Return(`{"id":"`+defaultResume.ID.String()+`","file_name":"default.pdf","mime_type":"application/pdf","byte_size":1024,"is_default":true,"created_at":"2023-01-01T00:00:00Z","updated_at":"2023-01-01T00:00:00Z"}`, nil)

				mockResumeRepository.EXPECT().
					SwitchDefaultResume(ctx, defaultResume.ID, resumeID).
					Return(errors.New("switch failed"))

				return mockResumeRepository, mockAuthContext, mockRedisTaskPublisher, mockRedisClient
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Equal(t, "switch failed", gotErr.Error())
			},
		},
		{
			name: "Error - Redis unmarshal failure (fallback to database)",
			input: &entities.SwitchDefaultResumeRequest{
				ResumeID: resumeID.String(),
			},
			setup: func() (*mockResume.MockResumeReposity, *mockMiddleware.MockAuthContext, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeRepository := new(mockResume.MockResumeReposity)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockRedisTaskPublisher := new(queue.MockRedisTaskPublisher)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							ID:     userID,
							UserID: userID.String(),
						},
					}, nil)

				// Return invalid JSON that will fail to unmarshal
				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s:%s", constants.RedisPrefixDefaultResume, userID.String())).
					Return("invalid json", nil)

				// After unmarshal failure, it falls back to database
				defaultResume := db.Resumes{
					ID:        uuid.New(),
					UserID:    userID,
					FileName:  "default.pdf",
					MimeType:  "application/pdf",
					ByteSize:  1024,
					IsDefault: true,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}

				mockResumeRepository.EXPECT().
					GetDefaultResumeByUserID(ctx, userID).
					Return(defaultResume, nil)

				mockResumeRepository.EXPECT().
					SwitchDefaultResume(ctx, defaultResume.ID, resumeID).
					Return(nil)

				mockRedisTaskPublisher.EXPECT().
					PublishTaskDeleteRedis(ctx, mock.AnythingOfType("*database.RedisDeletePayload")).
					Return(nil)

				return mockResumeRepository, mockAuthContext, mockRedisTaskPublisher, mockRedisClient
			},
			verify: func(t *testing.T, gotErr error) {
				assert.NoError(t, gotErr)
			},
		},
		{
			name: "Error - Redis unmarshal failure with database error",
			input: &entities.SwitchDefaultResumeRequest{
				ResumeID: resumeID.String(),
			},
			setup: func() (*mockResume.MockResumeReposity, *mockMiddleware.MockAuthContext, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeRepository := new(mockResume.MockResumeReposity)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockRedisTaskPublisher := new(queue.MockRedisTaskPublisher)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							ID:     userID,
							UserID: userID.String(),
						},
					}, nil)

				// Return invalid JSON that will fail to unmarshal
				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s:%s", constants.RedisPrefixDefaultResume, userID.String())).
					Return("invalid json", nil)

				// After unmarshal failure, database fallback also fails
				mockResumeRepository.EXPECT().
					GetDefaultResumeByUserID(ctx, userID).
					Return(db.Resumes{}, errors.New("database error"))

				return mockResumeRepository, mockAuthContext, mockRedisTaskPublisher, mockRedisClient
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Equal(t, "database error", gotErr.Error())
			},
		},
		{
			name: "Error - Redis delete task publishing failure",
			input: &entities.SwitchDefaultResumeRequest{
				ResumeID: resumeID.String(),
			},
			setup: func() (*mockResume.MockResumeReposity, *mockMiddleware.MockAuthContext, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeRepository := new(mockResume.MockResumeReposity)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockRedisTaskPublisher := new(queue.MockRedisTaskPublisher)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							ID:     userID,
							UserID: userID.String(),
						},
					}, nil)

				defaultResume := db.Resumes{
					ID: uuid.New(),
				}

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s:%s", constants.RedisPrefixDefaultResume, userID.String())).
					Return(`{"id":"`+defaultResume.ID.String()+`","file_name":"default.pdf","mime_type":"application/pdf","byte_size":1024,"is_default":true,"created_at":"2023-01-01T00:00:00Z","updated_at":"2023-01-01T00:00:00Z"}`, nil)

				mockResumeRepository.EXPECT().
					SwitchDefaultResume(ctx, defaultResume.ID, resumeID).
					Return(nil)

				mockRedisTaskPublisher.EXPECT().
					PublishTaskDeleteRedis(ctx, mock.AnythingOfType("*database.RedisDeletePayload")).
					Return(errors.New("task publishing failed"))

				return mockResumeRepository, mockAuthContext, mockRedisTaskPublisher, mockRedisClient
			},
			verify: func(t *testing.T, gotErr error) {
				assert.NoError(t, gotErr)
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			mockResumeRepository, mockAuthContext, mockRedisTaskPublisher, mockRedisClient := tC.setup()
			defer func() {
				if mockResumeRepository != nil {
					mockResumeRepository.AssertExpectations(t)
				}
				if mockAuthContext != nil {
					mockAuthContext.AssertExpectations(t)
				}
				if mockRedisTaskPublisher != nil {
					mockRedisTaskPublisher.AssertExpectations(t)
				}
				if mockRedisClient != nil {
					mockRedisClient.AssertExpectations(t)
				}
			}()

			svc := NewResumeService(lgr, mockResumeRepository, mockAuthContext, nil, nil, mockRedisTaskPublisher, mockRedisClient, nil)
			gotErr := svc.SwitchDefaultResume(ctx, tC.input)

			tC.verify(t, gotErr)
		})
	}
}

func TestResumeService_GetResumeByID(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()
	resumeID := uuid.New()

	testCases := []struct {
		name   string
		input  *entities.GetResumeByIDRequest
		setup  func() (*mockResume.MockResumeReposity, *mockS3.MockS3Storage)
		verify func(t *testing.T, gotResp *entities.GetResumeByIDResponse, gotErr error)
	}{
		{
			name: "Success",
			input: &entities.GetResumeByIDRequest{
				ResumeID: resumeID.String(),
			},
			setup: func() (*mockResume.MockResumeReposity, *mockS3.MockS3Storage) {
				mockResumeRepository := new(mockResume.MockResumeReposity)
				mockS3Storage := new(mockS3.MockS3Storage)

				resume := db.Resumes{
					ID:         resumeID,
					UserID:     uuid.New(),
					FileName:   "resume.pdf",
					StorageKey: "s3-key-123",
					MimeType:   "application/pdf",
					ByteSize:   1024,
					IsDefault:  true,
					CreatedAt:  time.Now(),
					UpdatedAt:  time.Now(),
				}

				mockResumeRepository.EXPECT().
					GetResumeByID(ctx, resumeID).
					Return(&resume, nil)

				mockS3Storage.EXPECT().
					GeneratePresignedURL(ctx, resume.StorageKey, constants.S3PresignedURLTTL).
					Return("https://s3.amazonaws.com/presigned-url", nil)

				return mockResumeRepository, mockS3Storage
			},
			verify: func(t *testing.T, gotResp *entities.GetResumeByIDResponse, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, gotResp)
				assert.Equal(t, resumeID.String(), gotResp.ID)
				assert.Equal(t, "resume.pdf", gotResp.FileName)
				assert.Equal(t, "https://s3.amazonaws.com/presigned-url", gotResp.FileUrl)
			},
		},
		{
			name: "Error - Resume not found",
			input: &entities.GetResumeByIDRequest{
				ResumeID: resumeID.String(),
			},
			setup: func() (*mockResume.MockResumeReposity, *mockS3.MockS3Storage) {
				mockResumeRepository := new(mockResume.MockResumeReposity)
				mockS3Storage := new(mockS3.MockS3Storage)

				mockResumeRepository.EXPECT().
					GetResumeByID(ctx, resumeID).
					Return(&db.Resumes{}, errors.New("resume not found"))

				return mockResumeRepository, mockS3Storage
			},
			verify: func(t *testing.T, gotResp *entities.GetResumeByIDResponse, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
				assert.Equal(t, "resume not found", gotErr.Error())
			},
		},
		{
			name: "Error - S3 presigned URL generation failure",
			input: &entities.GetResumeByIDRequest{
				ResumeID: resumeID.String(),
			},
			setup: func() (*mockResume.MockResumeReposity, *mockS3.MockS3Storage) {
				mockResumeRepository := new(mockResume.MockResumeReposity)
				mockS3Storage := new(mockS3.MockS3Storage)

				resume := db.Resumes{
					ID:         resumeID,
					UserID:     uuid.New(),
					FileName:   "resume.pdf",
					StorageKey: "s3-key-123",
					MimeType:   "application/pdf",
					ByteSize:   1024,
					IsDefault:  true,
					CreatedAt:  time.Now(),
					UpdatedAt:  time.Now(),
				}

				mockResumeRepository.EXPECT().
					GetResumeByID(ctx, resumeID).
					Return(&resume, nil)

				mockS3Storage.EXPECT().
					GeneratePresignedURL(ctx, resume.StorageKey, constants.S3PresignedURLTTL).
					Return("", errors.New("S3 error"))

				return mockResumeRepository, mockS3Storage
			},
			verify: func(t *testing.T, gotResp *entities.GetResumeByIDResponse, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
				assert.Equal(t, "S3 error", gotErr.Error())
			},
		},
		{
			name: "Error - Invalid UUID format",
			input: &entities.GetResumeByIDRequest{
				ResumeID: "invalid-uuid",
			},
			setup: func() (*mockResume.MockResumeReposity, *mockS3.MockS3Storage) {
				mockResumeRepository := new(mockResume.MockResumeReposity)
				mockS3Storage := new(mockS3.MockS3Storage)

				// No mocks needed as the service should fail before calling them

				return mockResumeRepository, mockS3Storage
			},
			verify: func(t *testing.T, gotResp *entities.GetResumeByIDResponse, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
				assert.Contains(t, gotErr.Error(), "invalid UUID")
			},
		},
		{
			name: "Error - Empty resume ID",
			input: &entities.GetResumeByIDRequest{
				ResumeID: "",
			},
			setup: func() (*mockResume.MockResumeReposity, *mockS3.MockS3Storage) {
				mockResumeRepository := new(mockResume.MockResumeReposity)
				mockS3Storage := new(mockS3.MockS3Storage)

				// No mocks needed as the service should fail before calling them

				return mockResumeRepository, mockS3Storage
			},
			verify: func(t *testing.T, gotResp *entities.GetResumeByIDResponse, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
				assert.Contains(t, gotErr.Error(), "invalid UUID")
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			mockResumeRepository, mockS3Storage := tC.setup()
			defer func() {
				if mockResumeRepository != nil {
					mockResumeRepository.AssertExpectations(t)
				}
				if mockS3Storage != nil {
					mockS3Storage.AssertExpectations(t)
				}
			}()

			svc := NewResumeService(lgr, mockResumeRepository, nil, mockS3Storage, nil, nil, nil, nil)
			gotResp, gotErr := svc.GetResumeByID(ctx, tC.input)

			tC.verify(t, gotResp, gotErr)
		})
	}
}
