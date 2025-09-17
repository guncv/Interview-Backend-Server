package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
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
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/aws"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/database"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
	"gitlab.com/interview-simulation/interview-backend-server/internal/middleware"
	mockAws "gitlab.com/interview-simulation/interview-backend-server/internal/mocks/aws"
	mockDatabase "gitlab.com/interview-simulation/interview-backend-server/internal/mocks/database"
	mockMiddleware "gitlab.com/interview-simulation/interview-backend-server/internal/mocks/middleware"
	"gitlab.com/interview-simulation/interview-backend-server/internal/mocks/queue"
	mockRepositories "gitlab.com/interview-simulation/interview-backend-server/internal/mocks/repositories"
	mockServices "gitlab.com/interview-simulation/interview-backend-server/internal/mocks/services"
	mockUtils "gitlab.com/interview-simulation/interview-backend-server/internal/mocks/utils"
	"gitlab.com/interview-simulation/interview-backend-server/internal/repositories"
	utilsPkg "gitlab.com/interview-simulation/interview-backend-server/internal/utils"
)

func TestInterviewSessionService_CreateInterviewSessionWithNewResume(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()
	userID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	invalidUserID := "invalid-user-id"

	testCases := []struct {
		name   string
		input  *entities.CreateInterviewSessionWithNewResumeRequest
		setup  func() (*mockServices.MockResumeService, *mockMiddleware.MockAuthContext, *mockRepositories.MockResumeReposity, *mockUtils.MockGenerator, *mockRepositories.MockInterviewSessionRepository, *mockUtils.MockJwtToken, *config.Config, *mockAws.MockS3Storage, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient)
		verify func(t *testing.T, gotResp *entities.CreateInterviewSessionWithNewResumeResponse, gotErr error)
	}{
		{
			name: "Success - Create interview session with new resume",
			input: &entities.CreateInterviewSessionWithNewResumeRequest{
				File: &multipart.FileHeader{
					Filename: "resume.pdf",
					Size:     1024,
					Header:   map[string][]string{"Content-Type": {"application/pdf"}},
				},
				Position:  "Software Engineer",
				IsConsent: true,
			},
			setup: func() (*mockServices.MockResumeService, *mockMiddleware.MockAuthContext, *mockRepositories.MockResumeReposity, *mockUtils.MockGenerator, *mockRepositories.MockInterviewSessionRepository, *mockUtils.MockJwtToken, *config.Config, *mockAws.MockS3Storage, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeService := new(mockServices.MockResumeService)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockResumeRepo := new(mockRepositories.MockResumeReposity)
				mockGenerator := new(mockUtils.MockGenerator)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockJwtMaker := new(mockUtils.MockJwtToken)
				mockS3Storage := new(mockAws.MockS3Storage)
				mockPublisher := new(queue.MockRedisTaskPublisher)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				// Mock auth context
				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							ID:     userID,
							UserID: userID.String(),
							Role:   "user",
						},
					}, nil)

				// Mock generator for UUIDs
				sessionID := uuid.New()
				resumeID := uuid.New()
				tokenKey := uuid.New()

				mockGenerator.EXPECT().GenerateUUID(ctx).Return(sessionID).Times(1)
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(resumeID).Times(1)
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(tokenKey).Times(1)

				// Mock resume repository for JSON generation
				mockResumeRepo.EXPECT().
					ExtractResumeJsonForRAG(ctx, mock.AnythingOfType("*repositories.ExtractResumeJsonForRAGReq")).
					Return(nil)

				mockResumeRepo.EXPECT().
					ListAllResumesFileNameByUserID(ctx, userID).
					Return([]string{"resume.pdf"}, nil)

				mockResumeRepo.EXPECT().
					CheckIsDefaultResumeExistsByUserID(ctx, userID).
					Return(false, nil)

				// Mock S3 storage
				mockS3Storage.EXPECT().
					UploadFile(ctx, mock.AnythingOfType("*multipart.FileHeader"), constants.S3ResumeKey, userID.String()).
					Return("s3-key-123", nil)

				// Mock interview session repository
				mockInterviewSessionRepo.EXPECT().
					CreateInterviewSessionWithNewResumeTx(ctx, mock.AnythingOfType("*repositories.CreateInterviewSessionTxReq")).
					Return(nil)

				// Mock Redis client
				mockRedisClient.EXPECT().
					Set(ctx, mock.AnythingOfType("database.RedisPayload")).
					Return(nil)

				config := &config.Config{
					InterviewSessionConfig: config.InterviewSessionConfig{
						InterviewSessionTokenTTL: 24 * time.Hour,
					},
				}

				return mockResumeService, mockAuthContext, mockResumeRepo, mockGenerator, mockInterviewSessionRepo, mockJwtMaker, config, mockS3Storage, mockPublisher, mockRedisClient
			},
			verify: func(t *testing.T, gotResp *entities.CreateInterviewSessionWithNewResumeResponse, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, gotResp)
				assert.NotEmpty(t, gotResp.SessionToken)
			},
		},
		{
			name: "Success - Create interview session with converted resume",
			input: &entities.CreateInterviewSessionWithNewResumeRequest{
				File: &multipart.FileHeader{
					Filename: "resume.pdf",
					Size:     1024,
					Header:   map[string][]string{"Content-Type": {"application/pdf"}},
				},
				Position:  "Software Engineer",
				IsConsent: true,
			},
			setup: func() (*mockServices.MockResumeService, *mockMiddleware.MockAuthContext, *mockRepositories.MockResumeReposity, *mockUtils.MockGenerator, *mockRepositories.MockInterviewSessionRepository, *mockUtils.MockJwtToken, *config.Config, *mockAws.MockS3Storage, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeService := new(mockServices.MockResumeService)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockResumeRepo := new(mockRepositories.MockResumeReposity)
				mockGenerator := new(mockUtils.MockGenerator)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockJwtMaker := new(mockUtils.MockJwtToken)
				mockS3Storage := new(mockAws.MockS3Storage)
				mockPublisher := new(queue.MockRedisTaskPublisher)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				// Mock auth context
				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							ID:     userID,
							UserID: userID.String(),
							Role:   "user",
						},
					}, nil)

				// Mock generator for UUIDs
				sessionID := uuid.New()
				resumeID := uuid.New()
				tokenKey := uuid.New()

				mockGenerator.EXPECT().GenerateUUID(ctx).Return(sessionID).Times(1)
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(resumeID).Times(1)

				mockGenerator.EXPECT().GenerateUUID(ctx).Return(tokenKey).Times(1)

				// Mock resume repository for JSON generation
				mockResumeRepo.EXPECT().
					ExtractResumeJsonForRAG(ctx, mock.AnythingOfType("*repositories.ExtractResumeJsonForRAGReq")).
					Return(nil)

				mockResumeRepo.EXPECT().
					CheckIsDefaultResumeExistsByUserID(ctx, userID).
					Return(false, nil)

				mockResumeRepo.EXPECT().
					ListAllResumesFileNameByUserID(ctx, userID).
					Return([]string{"resume.pdf"}, nil)

				// Mock S3 storage
				mockS3Storage.EXPECT().
					UploadFile(ctx, mock.AnythingOfType("*multipart.FileHeader"), constants.S3ResumeKey, userID.String()).
					Return("s3-key-123", nil)

				// Mock interview session repository
				mockInterviewSessionRepo.EXPECT().
					CreateInterviewSessionWithNewResumeTx(ctx, mock.AnythingOfType("*repositories.CreateInterviewSessionTxReq")).
					Return(nil)

				// Mock Redis client
				mockRedisClient.EXPECT().
					Set(ctx, mock.AnythingOfType("database.RedisPayload")).
					Return(nil)

				config := &config.Config{
					InterviewSessionConfig: config.InterviewSessionConfig{
						InterviewSessionTokenTTL: 24 * time.Hour,
					},
				}

				return mockResumeService, mockAuthContext, mockResumeRepo, mockGenerator, mockInterviewSessionRepo, mockJwtMaker, config, mockS3Storage, mockPublisher, mockRedisClient
			},
			verify: func(t *testing.T, gotResp *entities.CreateInterviewSessionWithNewResumeResponse, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, gotResp)
				assert.NotEmpty(t, gotResp.SessionToken)
			},
		},
		{
			name: "Error - Auth context failure",
			input: &entities.CreateInterviewSessionWithNewResumeRequest{
				File: &multipart.FileHeader{
					Filename: "resume.pdf",
					Size:     1024,
					Header:   map[string][]string{"Content-Type": {"application/pdf"}},
				},
				Position:  "Software Engineer",
				IsConsent: true,
			},
			setup: func() (*mockServices.MockResumeService, *mockMiddleware.MockAuthContext, *mockRepositories.MockResumeReposity, *mockUtils.MockGenerator, *mockRepositories.MockInterviewSessionRepository, *mockUtils.MockJwtToken, *config.Config, *mockAws.MockS3Storage, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeService := new(mockServices.MockResumeService)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockResumeRepo := new(mockRepositories.MockResumeReposity)
				mockGenerator := new(mockUtils.MockGenerator)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockJwtMaker := new(mockUtils.MockJwtToken)
				mockS3Storage := new(mockAws.MockS3Storage)
				mockPublisher := new(queue.MockRedisTaskPublisher)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(nil, errors.New("auth failed"))

				config := &config.Config{}

				return mockResumeService, mockAuthContext, mockResumeRepo, mockGenerator, mockInterviewSessionRepo, mockJwtMaker, config, mockS3Storage, mockPublisher, mockRedisClient
			},
			verify: func(t *testing.T, gotResp *entities.CreateInterviewSessionWithNewResumeResponse, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
				assert.Equal(t, "auth failed", gotErr.Error())
			},
		},
		{
			name: "Error - Get resume JSON failure",
			input: &entities.CreateInterviewSessionWithNewResumeRequest{
				File: &multipart.FileHeader{
					Filename: "resume.pdf",
					Size:     1024,
					Header:   map[string][]string{"Content-Type": {"application/pdf"}},
				},
				Position:  "Software Engineer",
				IsConsent: true,
			},
			setup: func() (*mockServices.MockResumeService, *mockMiddleware.MockAuthContext, *mockRepositories.MockResumeReposity, *mockUtils.MockGenerator, *mockRepositories.MockInterviewSessionRepository, *mockUtils.MockJwtToken, *config.Config, *mockAws.MockS3Storage, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeService := new(mockServices.MockResumeService)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockResumeRepo := new(mockRepositories.MockResumeReposity)
				mockGenerator := new(mockUtils.MockGenerator)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockJwtMaker := new(mockUtils.MockJwtToken)
				mockS3Storage := new(mockAws.MockS3Storage)
				mockPublisher := new(queue.MockRedisTaskPublisher)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							ID:     userID,
							UserID: userID.String(),
							Role:   "user",
						},
					}, nil)

				// Mock generator for UUID (called before JSON generation)
				resumeID := uuid.New()
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(resumeID).Times(1)

				sessionID := uuid.New()
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(sessionID).Times(1)

				mockResumeRepo.EXPECT().
					ExtractResumeJsonForRAG(ctx, mock.AnythingOfType("*repositories.ExtractResumeJsonForRAGReq")).
					Return(errors.New("JSON generation failed"))

				config := &config.Config{}

				return mockResumeService, mockAuthContext, mockResumeRepo, mockGenerator, mockInterviewSessionRepo, mockJwtMaker, config, mockS3Storage, mockPublisher, mockRedisClient
			},
			verify: func(t *testing.T, gotResp *entities.CreateInterviewSessionWithNewResumeResponse, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
				assert.Contains(t, gotErr.Error(), "JSON generation failed")
			},
		},
		{
			name: "Error_Invalid user ID",
			input: &entities.CreateInterviewSessionWithNewResumeRequest{
				File: &multipart.FileHeader{
					Filename: "resume.pdf",
					Size:     1024,
					Header:   map[string][]string{"Content-Type": {"application/pdf"}},
				},
				Position:  "Software Engineer",
				IsConsent: true,
			},
			setup: func() (*mockServices.MockResumeService, *mockMiddleware.MockAuthContext, *mockRepositories.MockResumeReposity, *mockUtils.MockGenerator, *mockRepositories.MockInterviewSessionRepository, *mockUtils.MockJwtToken, *config.Config, *mockAws.MockS3Storage, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeService := new(mockServices.MockResumeService)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockResumeRepo := new(mockRepositories.MockResumeReposity)
				mockGenerator := new(mockUtils.MockGenerator)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockJwtMaker := new(mockUtils.MockJwtToken)
				mockS3Storage := new(mockAws.MockS3Storage)
				mockPublisher := new(queue.MockRedisTaskPublisher)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				// Mock auth context
				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							ID:     userID,
							UserID: invalidUserID,
							Role:   "user",
						},
					}, nil)
				resumeID := uuid.New()
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(resumeID).Times(1)

				// Mock generator for UUIDs
				sessionID := uuid.New()
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(sessionID).Times(1)

				// Mock resume repository for JSON generation
				mockResumeRepo.EXPECT().
					ExtractResumeJsonForRAG(ctx, mock.AnythingOfType("*repositories.ExtractResumeJsonForRAGReq")).
					Return(nil)

				config := &config.Config{
					InterviewSessionConfig: config.InterviewSessionConfig{
						InterviewSessionTokenTTL: 24 * time.Hour,
					},
				}

				return mockResumeService, mockAuthContext, mockResumeRepo, mockGenerator, mockInterviewSessionRepo, mockJwtMaker, config, mockS3Storage, mockPublisher, mockRedisClient
			},
			verify: func(t *testing.T, gotResp *entities.CreateInterviewSessionWithNewResumeResponse, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
				assert.Contains(t, gotErr.Error(), "The UUID is invalid. Please try again.")
				assert.Contains(t, gotErr.Error(), "[ONX0107]")
			},
		},
		{
			name: "Error - S3 upload failure",
			input: &entities.CreateInterviewSessionWithNewResumeRequest{
				File: &multipart.FileHeader{
					Filename: "resume.pdf",
					Size:     1024,
					Header:   map[string][]string{"Content-Type": {"application/pdf"}},
				},
				Position:  "Software Engineer",
				IsConsent: true,
			},
			setup: func() (*mockServices.MockResumeService, *mockMiddleware.MockAuthContext, *mockRepositories.MockResumeReposity, *mockUtils.MockGenerator, *mockRepositories.MockInterviewSessionRepository, *mockUtils.MockJwtToken, *config.Config, *mockAws.MockS3Storage, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeService := new(mockServices.MockResumeService)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockResumeRepo := new(mockRepositories.MockResumeReposity)
				mockGenerator := new(mockUtils.MockGenerator)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockJwtMaker := new(mockUtils.MockJwtToken)
				mockS3Storage := new(mockAws.MockS3Storage)
				mockPublisher := new(queue.MockRedisTaskPublisher)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							ID:     userID,
							UserID: userID.String(),
							Role:   "user",
						},
					}, nil)

				// Mock generator for UUID (called before JSON generation)
				resumeID := uuid.New()
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(resumeID).Times(1)

				sessionID := uuid.New()
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(sessionID).Times(1)

				mockResumeRepo.EXPECT().
					ExtractResumeJsonForRAG(ctx, mock.AnythingOfType("*repositories.ExtractResumeJsonForRAGReq")).
					Return(nil)

				mockResumeRepo.EXPECT().
					ListAllResumesFileNameByUserID(ctx, userID).
					Return(nil, errors.New("list all resumes file name by user ID failed"))

				config := &config.Config{}

				return mockResumeService, mockAuthContext, mockResumeRepo, mockGenerator, mockInterviewSessionRepo, mockJwtMaker, config, mockS3Storage, mockPublisher, mockRedisClient
			},
			verify: func(t *testing.T, gotResp *entities.CreateInterviewSessionWithNewResumeResponse, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
				assert.Contains(t, gotErr.Error(), "list all resumes file name by user ID failed")
			},
		},
		{
			name: "Error - S3 upload failure",
			input: &entities.CreateInterviewSessionWithNewResumeRequest{
				File: &multipart.FileHeader{
					Filename: "resume.pdf",
					Size:     1024,
					Header:   map[string][]string{"Content-Type": {"application/pdf"}},
				},
				Position:  "Software Engineer",
				IsConsent: true,
			},
			setup: func() (*mockServices.MockResumeService, *mockMiddleware.MockAuthContext, *mockRepositories.MockResumeReposity, *mockUtils.MockGenerator, *mockRepositories.MockInterviewSessionRepository, *mockUtils.MockJwtToken, *config.Config, *mockAws.MockS3Storage, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeService := new(mockServices.MockResumeService)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockResumeRepo := new(mockRepositories.MockResumeReposity)
				mockGenerator := new(mockUtils.MockGenerator)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockJwtMaker := new(mockUtils.MockJwtToken)
				mockS3Storage := new(mockAws.MockS3Storage)
				mockPublisher := new(queue.MockRedisTaskPublisher)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							ID:     userID,
							UserID: userID.String(),
							Role:   "user",
						},
					}, nil)

				// Mock generator for UUID (called before JSON generation)
				resumeID := uuid.New()
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(resumeID).Times(1)

				sessionID := uuid.New()
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(sessionID).Times(1)

				mockResumeRepo.EXPECT().
					ExtractResumeJsonForRAG(ctx, mock.AnythingOfType("*repositories.ExtractResumeJsonForRAGReq")).
					Return(nil)

				mockResumeRepo.EXPECT().
					ListAllResumesFileNameByUserID(ctx, userID).
					Return([]string{"resume.pdf"}, nil)

				mockS3Storage.EXPECT().
					UploadFile(ctx, mock.AnythingOfType("*multipart.FileHeader"), constants.S3ResumeKey, userID.String()).
					Return("", errors.New("S3 upload failed"))

				config := &config.Config{}

				return mockResumeService, mockAuthContext, mockResumeRepo, mockGenerator, mockInterviewSessionRepo, mockJwtMaker, config, mockS3Storage, mockPublisher, mockRedisClient
			},
			verify: func(t *testing.T, gotResp *entities.CreateInterviewSessionWithNewResumeResponse, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
				assert.Contains(t, gotErr.Error(), "S3 upload failed")
			},
		},
		{
			name: "Error - Default resume check failure",
			input: &entities.CreateInterviewSessionWithNewResumeRequest{
				File: &multipart.FileHeader{
					Filename: "resume.pdf",
					Size:     1024,
					Header:   map[string][]string{"Content-Type": {"application/pdf"}},
				},
				Position:  "Software Engineer",
				IsConsent: true,
			},
			setup: func() (*mockServices.MockResumeService, *mockMiddleware.MockAuthContext, *mockRepositories.MockResumeReposity, *mockUtils.MockGenerator, *mockRepositories.MockInterviewSessionRepository, *mockUtils.MockJwtToken, *config.Config, *mockAws.MockS3Storage, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeService := new(mockServices.MockResumeService)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockResumeRepo := new(mockRepositories.MockResumeReposity)
				mockGenerator := new(mockUtils.MockGenerator)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockJwtMaker := new(mockUtils.MockJwtToken)
				mockS3Storage := new(mockAws.MockS3Storage)
				mockPublisher := new(queue.MockRedisTaskPublisher)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							ID:     userID,
							UserID: userID.String(),
							Role:   "user",
						},
					}, nil)

				// Mock generator for UUID (called before JSON generation)
				resumeID := uuid.New()
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(resumeID).Times(1)

				sessionID := uuid.New()
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(sessionID).Times(1)

				mockResumeRepo.EXPECT().
					ExtractResumeJsonForRAG(ctx, mock.AnythingOfType("*repositories.ExtractResumeJsonForRAGReq")).
					Return(nil)

				mockResumeRepo.EXPECT().
					ListAllResumesFileNameByUserID(ctx, userID).
					Return([]string{"resume.pdf"}, nil)

				mockS3Storage.EXPECT().
					UploadFile(ctx, mock.AnythingOfType("*multipart.FileHeader"), constants.S3ResumeKey, userID.String()).
					Return("s3-key-123", nil)

				mockResumeRepo.EXPECT().
					CheckIsDefaultResumeExistsByUserID(ctx, userID).
					Return(false, errors.New("default resume check failed"))

				config := &config.Config{}

				return mockResumeService, mockAuthContext, mockResumeRepo, mockGenerator, mockInterviewSessionRepo, mockJwtMaker, config, mockS3Storage, mockPublisher, mockRedisClient
			},
			verify: func(t *testing.T, gotResp *entities.CreateInterviewSessionWithNewResumeResponse, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
				assert.Contains(t, gotErr.Error(), "default resume check failed")
			},
		},
		{
			name: "Error - Transaction failure",
			input: &entities.CreateInterviewSessionWithNewResumeRequest{
				File: &multipart.FileHeader{
					Filename: "resume.pdf",
					Size:     1024,
					Header:   map[string][]string{"Content-Type": {"application/pdf"}},
				},
				Position:  "Software Engineer",
				IsConsent: true,
			},
			setup: func() (*mockServices.MockResumeService, *mockMiddleware.MockAuthContext, *mockRepositories.MockResumeReposity, *mockUtils.MockGenerator, *mockRepositories.MockInterviewSessionRepository, *mockUtils.MockJwtToken, *config.Config, *mockAws.MockS3Storage, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeService := new(mockServices.MockResumeService)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockResumeRepo := new(mockRepositories.MockResumeReposity)
				mockGenerator := new(mockUtils.MockGenerator)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockJwtMaker := new(mockUtils.MockJwtToken)
				mockS3Storage := new(mockAws.MockS3Storage)
				mockPublisher := new(queue.MockRedisTaskPublisher)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							ID:     userID,
							UserID: userID.String(),
							Role:   "user",
						},
					}, nil)

				// Mock generator for UUID (called before JSON generation)
				resumeID := uuid.New()
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(resumeID).Times(1)

				sessionID := uuid.New()
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(sessionID).Times(1)

				mockResumeRepo.EXPECT().
					ExtractResumeJsonForRAG(ctx, mock.AnythingOfType("*repositories.ExtractResumeJsonForRAGReq")).
					Return(nil)

				mockResumeRepo.EXPECT().
					ListAllResumesFileNameByUserID(ctx, userID).
					Return([]string{"resume.pdf"}, nil)

				mockS3Storage.EXPECT().
					UploadFile(ctx, mock.AnythingOfType("*multipart.FileHeader"), constants.S3ResumeKey, userID.String()).
					Return("s3-key-123", nil)

				mockResumeRepo.EXPECT().
					CheckIsDefaultResumeExistsByUserID(ctx, userID).
					Return(false, nil)

				mockInterviewSessionRepo.EXPECT().
					CreateInterviewSessionWithNewResumeTx(ctx, mock.AnythingOfType("*repositories.CreateInterviewSessionTxReq")).
					Return(errors.New("transaction failed"))

				// Mock publisher for cleanup
				mockPublisher.EXPECT().
					PublishTaskDeleteFile(ctx, mock.AnythingOfType("*aws.DeleteFilePayload")).
					Return(nil)

				config := &config.Config{}

				return mockResumeService, mockAuthContext, mockResumeRepo, mockGenerator, mockInterviewSessionRepo, mockJwtMaker, config, mockS3Storage, mockPublisher, mockRedisClient
			},
			verify: func(t *testing.T, gotResp *entities.CreateInterviewSessionWithNewResumeResponse, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
				assert.Contains(t, gotErr.Error(), "transaction failed")
			},
		},
		{
			name: "Error - Transaction failure and Delete File Error",
			input: &entities.CreateInterviewSessionWithNewResumeRequest{
				File: &multipart.FileHeader{
					Filename: "resume.pdf",
					Size:     1024,
					Header:   map[string][]string{"Content-Type": {"application/pdf"}},
				},
				Position:  "Software Engineer",
				IsConsent: true,
			},
			setup: func() (*mockServices.MockResumeService, *mockMiddleware.MockAuthContext, *mockRepositories.MockResumeReposity, *mockUtils.MockGenerator, *mockRepositories.MockInterviewSessionRepository, *mockUtils.MockJwtToken, *config.Config, *mockAws.MockS3Storage, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeService := new(mockServices.MockResumeService)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockResumeRepo := new(mockRepositories.MockResumeReposity)
				mockGenerator := new(mockUtils.MockGenerator)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockJwtMaker := new(mockUtils.MockJwtToken)
				mockS3Storage := new(mockAws.MockS3Storage)
				mockPublisher := new(queue.MockRedisTaskPublisher)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							ID:     userID,
							UserID: userID.String(),
							Role:   "user",
						},
					}, nil)

				// Mock generator for UUID (called before JSON generation)
				resumeID := uuid.New()
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(resumeID).Times(1)

				sessionID := uuid.New()
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(sessionID).Times(1)

				mockResumeRepo.EXPECT().
					ExtractResumeJsonForRAG(ctx, mock.AnythingOfType("*repositories.ExtractResumeJsonForRAGReq")).
					Return(nil)

				mockResumeRepo.EXPECT().
					ListAllResumesFileNameByUserID(ctx, userID).
					Return([]string{"resume.pdf"}, nil)

				mockS3Storage.EXPECT().
					UploadFile(ctx, mock.AnythingOfType("*multipart.FileHeader"), constants.S3ResumeKey, userID.String()).
					Return("s3-key-123", nil)

				mockResumeRepo.EXPECT().
					CheckIsDefaultResumeExistsByUserID(ctx, userID).
					Return(false, nil)

				mockInterviewSessionRepo.EXPECT().
					CreateInterviewSessionWithNewResumeTx(ctx, mock.AnythingOfType("*repositories.CreateInterviewSessionTxReq")).
					Return(errors.New("transaction failed"))

				mockPublisher.EXPECT().
					PublishTaskDeleteFile(ctx, mock.AnythingOfType("*aws.DeleteFilePayload")).
					Return(errors.New("delete file error"))

				config := &config.Config{}

				return mockResumeService, mockAuthContext, mockResumeRepo, mockGenerator, mockInterviewSessionRepo, mockJwtMaker, config, mockS3Storage, mockPublisher, mockRedisClient
			},
			verify: func(t *testing.T, gotResp *entities.CreateInterviewSessionWithNewResumeResponse, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
				assert.Contains(t, gotErr.Error(), "transaction failed")
			},
		},
		{
			name: "Error - Redis set failure",
			input: &entities.CreateInterviewSessionWithNewResumeRequest{
				File: &multipart.FileHeader{
					Filename: "resume.pdf",
					Size:     1024,
					Header:   map[string][]string{"Content-Type": {"application/pdf"}},
				},
				Position:  "Software Engineer",
				IsConsent: true,
			},
			setup: func() (*mockServices.MockResumeService, *mockMiddleware.MockAuthContext, *mockRepositories.MockResumeReposity, *mockUtils.MockGenerator, *mockRepositories.MockInterviewSessionRepository, *mockUtils.MockJwtToken, *config.Config, *mockAws.MockS3Storage, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeService := new(mockServices.MockResumeService)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockResumeRepo := new(mockRepositories.MockResumeReposity)
				mockGenerator := new(mockUtils.MockGenerator)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockJwtMaker := new(mockUtils.MockJwtToken)
				mockS3Storage := new(mockAws.MockS3Storage)
				mockPublisher := new(queue.MockRedisTaskPublisher)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							ID:     userID,
							UserID: userID.String(),
							Role:   "user",
						},
					}, nil)

				// Mock generator for UUID (called before JSON generation)
				resumeID := uuid.New()
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(resumeID).Times(1)

				sessionID := uuid.New()
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(sessionID).Times(1)

				mockResumeRepo.EXPECT().
					ExtractResumeJsonForRAG(ctx, mock.AnythingOfType("*repositories.ExtractResumeJsonForRAGReq")).
					Return(nil)

				mockResumeRepo.EXPECT().
					ListAllResumesFileNameByUserID(ctx, userID).
					Return([]string{"resume.pdf"}, nil)

				mockS3Storage.EXPECT().
					UploadFile(ctx, mock.AnythingOfType("*multipart.FileHeader"), constants.S3ResumeKey, userID.String()).
					Return("s3-key-123", nil)

				mockResumeRepo.EXPECT().
					CheckIsDefaultResumeExistsByUserID(ctx, userID).
					Return(false, nil)

				tokenKey := uuid.New()
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(tokenKey).Times(1)

				mockInterviewSessionRepo.EXPECT().
					CreateInterviewSessionWithNewResumeTx(ctx, mock.AnythingOfType("*repositories.CreateInterviewSessionTxReq")).
					Return(nil)

				mockRedisClient.EXPECT().
					Set(ctx, mock.AnythingOfType("database.RedisPayload")).
					Return(errors.New("Redis set failed"))

				config := &config.Config{
					InterviewSessionConfig: config.InterviewSessionConfig{
						InterviewSessionTokenTTL: 24 * time.Hour,
					},
				}

				return mockResumeService, mockAuthContext, mockResumeRepo, mockGenerator, mockInterviewSessionRepo, mockJwtMaker, config, mockS3Storage, mockPublisher, mockRedisClient
			},
			verify: func(t *testing.T, gotResp *entities.CreateInterviewSessionWithNewResumeResponse, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
				assert.Contains(t, gotErr.Error(), "Redis set failed")
			},
		},
		{
			name: "Error - Resume processing failure",
			input: &entities.CreateInterviewSessionWithNewResumeRequest{
				File: &multipart.FileHeader{
					Filename: "resume.pdf",
					Size:     1024,
					Header:   map[string][]string{"Content-Type": {"application/pdf"}},
				},
				Position:  "Software Engineer",
				IsConsent: true,
			},
			setup: func() (*mockServices.MockResumeService, *mockMiddleware.MockAuthContext, *mockRepositories.MockResumeReposity, *mockUtils.MockGenerator, *mockRepositories.MockInterviewSessionRepository, *mockUtils.MockJwtToken, *config.Config, *mockAws.MockS3Storage, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeService := new(mockServices.MockResumeService)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockResumeRepo := new(mockRepositories.MockResumeReposity)
				mockGenerator := new(mockUtils.MockGenerator)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockJwtMaker := new(mockUtils.MockJwtToken)
				mockS3Storage := new(mockAws.MockS3Storage)
				mockPublisher := new(queue.MockRedisTaskPublisher)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							ID:     userID,
							UserID: userID.String(),
							Role:   "user",
						},
					}, nil)

				resumeID := uuid.New()
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(resumeID).Times(1)

				sessionID := uuid.New()
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(sessionID).Times(1)

				// Mock resume repository to return error
				mockResumeRepo.EXPECT().
					ExtractResumeJsonForRAG(ctx, mock.AnythingOfType("*repositories.ExtractResumeJsonForRAGReq")).
					Return(errors.New("resume processing failed"))

				config := &config.Config{}

				return mockResumeService, mockAuthContext, mockResumeRepo, mockGenerator, mockInterviewSessionRepo, mockJwtMaker, config, mockS3Storage, mockPublisher, mockRedisClient
			},
			verify: func(t *testing.T, gotResp *entities.CreateInterviewSessionWithNewResumeResponse, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
				assert.Contains(t, gotErr.Error(), "resume processing failed")
			},
		},
		{
			name: "Error - Redis connection failure",
			input: &entities.CreateInterviewSessionWithNewResumeRequest{
				File: &multipart.FileHeader{
					Filename: "resume.pdf",
					Size:     1024,
					Header:   map[string][]string{"Content-Type": {"application/pdf"}},
				},
				Position:  "Software Engineer",
				IsConsent: true,
			},
			setup: func() (*mockServices.MockResumeService, *mockMiddleware.MockAuthContext, *mockRepositories.MockResumeReposity, *mockUtils.MockGenerator, *mockRepositories.MockInterviewSessionRepository, *mockUtils.MockJwtToken, *config.Config, *mockAws.MockS3Storage, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeService := new(mockServices.MockResumeService)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockResumeRepo := new(mockRepositories.MockResumeReposity)
				mockGenerator := new(mockUtils.MockGenerator)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockJwtMaker := new(mockUtils.MockJwtToken)
				mockS3Storage := new(mockAws.MockS3Storage)
				mockPublisher := new(queue.MockRedisTaskPublisher)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							ID:     userID,
							UserID: userID.String(),
							Role:   "user",
						},
					}, nil)

				resumeID := uuid.New()
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(resumeID).Times(1)

				sessionID := uuid.New()
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(sessionID).Times(1)

				mockResumeRepo.EXPECT().
					ExtractResumeJsonForRAG(ctx, mock.AnythingOfType("*repositories.ExtractResumeJsonForRAGReq")).
					Return(nil)

				mockResumeRepo.EXPECT().
					ListAllResumesFileNameByUserID(ctx, userID).
					Return([]string{"resume.pdf"}, nil)

				mockS3Storage.EXPECT().
					UploadFile(ctx, mock.AnythingOfType("*multipart.FileHeader"), constants.S3ResumeKey, userID.String()).
					Return("s3-key-123", nil)

				mockResumeRepo.EXPECT().
					CheckIsDefaultResumeExistsByUserID(ctx, userID).
					Return(false, nil)

				mockInterviewSessionRepo.EXPECT().
					CreateInterviewSessionWithNewResumeTx(ctx, mock.AnythingOfType("*repositories.CreateInterviewSessionTxReq")).
					Return(nil)

				// Mock generator for token key
				tokenKey := uuid.New()
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(tokenKey).Times(1)

				// Mock Redis client to return error
				mockRedisClient.EXPECT().
					Set(ctx, mock.AnythingOfType("database.RedisPayload")).
					Return(errors.New("Redis connection failed"))

				config := &config.Config{
					InterviewSessionConfig: config.InterviewSessionConfig{
						InterviewSessionTokenTTL: 24 * time.Hour,
					},
				}

				return mockResumeService, mockAuthContext, mockResumeRepo, mockGenerator, mockInterviewSessionRepo, mockJwtMaker, config, mockS3Storage, mockPublisher, mockRedisClient
			},
			verify: func(t *testing.T, gotResp *entities.CreateInterviewSessionWithNewResumeResponse, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
				assert.Contains(t, gotErr.Error(), "Redis connection failed")
			},
		},
		{
			name: "Error - File conversion failure",
			input: &entities.CreateInterviewSessionWithNewResumeRequest{
				File: &multipart.FileHeader{
					Filename: "resume.pdf",
					Size:     1024,
					Header:   map[string][]string{"Content-Type": {"application/pdf"}},
				},
				Position:  "Software Engineer",
				IsConsent: true,
			},
			setup: func() (*mockServices.MockResumeService, *mockMiddleware.MockAuthContext, *mockRepositories.MockResumeReposity, *mockUtils.MockGenerator, *mockRepositories.MockInterviewSessionRepository, *mockUtils.MockJwtToken, *config.Config, *mockAws.MockS3Storage, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeService := new(mockServices.MockResumeService)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockResumeRepo := new(mockRepositories.MockResumeReposity)
				mockGenerator := new(mockUtils.MockGenerator)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockJwtMaker := new(mockUtils.MockJwtToken)
				mockS3Storage := new(mockAws.MockS3Storage)
				mockPublisher := new(queue.MockRedisTaskPublisher)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							ID:     userID,
							UserID: userID.String(),
							Role:   "user",
						},
					}, nil)

				// Mock generator for UUID (called before JSON generation)
				resumeID := uuid.New()
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(resumeID).Times(1)

				sessionID := uuid.New()
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(sessionID).Times(1)

				mockResumeRepo.EXPECT().
					ExtractResumeJsonForRAG(ctx, mock.AnythingOfType("*repositories.ExtractResumeJsonForRAGReq")).
					Return(errors.New("invalid file: file is nil or corrupted"))

				config := &config.Config{}

				return mockResumeService, mockAuthContext, mockResumeRepo, mockGenerator, mockInterviewSessionRepo, mockJwtMaker, config, mockS3Storage, mockPublisher, mockRedisClient
			},
			verify: func(t *testing.T, gotResp *entities.CreateInterviewSessionWithNewResumeResponse, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
				assert.Contains(t, gotErr.Error(), "invalid file: file is nil or corrupted")
			},
		},
		{
			name: "Success - File conversion succeeds",
			input: &entities.CreateInterviewSessionWithNewResumeRequest{
				File: &multipart.FileHeader{
					Filename: "resume.pdf",
					Size:     1024,
					Header:   map[string][]string{"Content-Type": {"application/pdf"}},
				},
				Position:  "Software Engineer",
				IsConsent: true,
			},
			setup: func() (*mockServices.MockResumeService, *mockMiddleware.MockAuthContext, *mockRepositories.MockResumeReposity, *mockUtils.MockGenerator, *mockRepositories.MockInterviewSessionRepository, *mockUtils.MockJwtToken, *config.Config, *mockAws.MockS3Storage, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeService := new(mockServices.MockResumeService)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockResumeRepo := new(mockRepositories.MockResumeReposity)
				mockGenerator := new(mockUtils.MockGenerator)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockJwtMaker := new(mockUtils.MockJwtToken)
				mockS3Storage := new(mockAws.MockS3Storage)
				mockPublisher := new(queue.MockRedisTaskPublisher)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				// Mock auth context
				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							ID:     userID,
							UserID: userID.String(),
							Role:   "user",
						},
					}, nil)

				// Mock generator for UUIDs
				sessionID := uuid.New()
				resumeID := uuid.New()
				tokenKey := uuid.New()

				mockGenerator.EXPECT().GenerateUUID(ctx).Return(sessionID).Times(1)
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(resumeID).Times(1)
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(tokenKey).Times(1)

				// Mock resume repository for JSON generation with successful file conversion
				mockResumeRepo.EXPECT().
					ExtractResumeJsonForRAG(ctx, mock.AnythingOfType("*repositories.ExtractResumeJsonForRAGReq")).
					Return(nil)

				mockResumeRepo.EXPECT().
					CheckIsDefaultResumeExistsByUserID(ctx, userID).
					Return(false, nil)

				mockResumeRepo.EXPECT().
					ListAllResumesFileNameByUserID(ctx, userID).
					Return([]string{"resume.pdf"}, nil)

				// Mock S3 storage
				mockS3Storage.EXPECT().
					UploadFile(ctx, mock.AnythingOfType("*multipart.FileHeader"), constants.S3ResumeKey, userID.String()).
					Return("s3-key-123", nil)

				// Mock interview session repository
				mockInterviewSessionRepo.EXPECT().
					CreateInterviewSessionWithNewResumeTx(ctx, mock.AnythingOfType("*repositories.CreateInterviewSessionTxReq")).
					Return(nil)

				// Mock Redis client
				mockRedisClient.EXPECT().
					Set(ctx, mock.AnythingOfType("database.RedisPayload")).
					Return(nil)

				config := &config.Config{
					InterviewSessionConfig: config.InterviewSessionConfig{
						InterviewSessionTokenTTL: 24 * time.Hour,
					},
				}

				return mockResumeService, mockAuthContext, mockResumeRepo, mockGenerator, mockInterviewSessionRepo, mockJwtMaker, config, mockS3Storage, mockPublisher, mockRedisClient
			},
			verify: func(t *testing.T, gotResp *entities.CreateInterviewSessionWithNewResumeResponse, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, gotResp)
				assert.NotEmpty(t, gotResp.SessionToken)
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			mockResumeService, mockAuthContext, mockResumeRepo, mockGenerator, mockInterviewSessionRepo, mockJwtMaker, config, mockS3Storage, mockPublisher, mockRedisClient := tC.setup()
			defer func() {
				if mockResumeService != nil {
					mockResumeService.AssertExpectations(t)
				}
				if mockAuthContext != nil {
					mockAuthContext.AssertExpectations(t)
				}
				if mockResumeRepo != nil {
					mockResumeRepo.AssertExpectations(t)
				}
				if mockGenerator != nil {
					mockGenerator.AssertExpectations(t)
				}
				if mockInterviewSessionRepo != nil {
					mockInterviewSessionRepo.AssertExpectations(t)
				}
				if mockJwtMaker != nil {
					mockJwtMaker.AssertExpectations(t)
				}
				if mockS3Storage != nil {
					mockS3Storage.AssertExpectations(t)
				}
				if mockPublisher != nil {
					mockPublisher.AssertExpectations(t)
				}
				if mockRedisClient != nil {
					mockRedisClient.AssertExpectations(t)
				}
			}()

			svc := NewInterviewSessionService(
				lgr,
				mockAuthContext,
				mockResumeRepo,
				mockGenerator,
				mockInterviewSessionRepo,
				mockJwtMaker,
				config,
				mockS3Storage,
				mockPublisher,
				mockRedisClient,
				nil,
				nil,
				nil,
			)

			gotResp, gotErr := svc.CreateInterviewSessionWithNewResume(ctx, tC.input)

			tC.verify(t, gotResp, gotErr)
		})
	}
}

func TestInterviewSessionService_CreateInterviewSessionWithExistingResume(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()
	userID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	invalidResumeID := "invalid-resume-id"
	invalidUserID := "invalid-user-id"
	resumeID := uuid.New()

	testCases := []struct {
		name   string
		input  *entities.CreateInterviewSessionWithExistingResumeReq
		setup  func() (*mockServices.MockResumeService, *mockMiddleware.MockAuthContext, *mockRepositories.MockResumeReposity, *mockUtils.MockGenerator, *mockRepositories.MockInterviewSessionRepository, *mockUtils.MockJwtToken, *config.Config, *mockAws.MockS3Storage, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient)
		verify func(t *testing.T, gotResp *entities.CreateInterviewSessionWithExistingResumeResp, gotErr error)
	}{
		{
			name: "Success - Create interview session with existing resume",
			input: &entities.CreateInterviewSessionWithExistingResumeReq{
				ResumeID:  resumeID.String(),
				Position:  "Software Engineer",
				IsConsent: true,
			},
			setup: func() (*mockServices.MockResumeService, *mockMiddleware.MockAuthContext, *mockRepositories.MockResumeReposity, *mockUtils.MockGenerator, *mockRepositories.MockInterviewSessionRepository, *mockUtils.MockJwtToken, *config.Config, *mockAws.MockS3Storage, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeService := new(mockServices.MockResumeService)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockResumeRepo := new(mockRepositories.MockResumeReposity)
				mockGenerator := new(mockUtils.MockGenerator)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockJwtMaker := new(mockUtils.MockJwtToken)
				mockS3Storage := new(mockAws.MockS3Storage)
				mockPublisher := new(queue.MockRedisTaskPublisher)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				// Mock auth context
				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							ID:     userID,
							UserID: userID.String(),
							Role:   "user",
						},
					}, nil)

				// Mock resume repository
				mockResumeRepo.EXPECT().
					GetResumeByID(ctx, resumeID).
					Return(&db.Resumes{
						ID:         resumeID,
						UserID:     userID,
						FileName:   "resume.pdf",
						StorageKey: "s3-key-123",
						MimeType:   "application/pdf",
						ByteSize:   1024,
						IsDefault:  false,
						CreatedAt:  time.Now(),
						UpdatedAt:  time.Now(),
					}, nil)

				// Mock S3 storage
				mockS3Storage.EXPECT().
					DownloadFile(ctx, "s3-key-123").
					Return(&aws.CustomFileHeader{
						Filename:    "resume.pdf",
						Size:        1024,
						Header:      map[string][]string{"Content-Type": {"application/pdf"}},
						FileContent: []byte("resume content"),
					}, nil)

				// Mock generator for UUIDs
				sessionID := uuid.New()
				tokenKey := uuid.New()

				mockGenerator.EXPECT().GenerateUUID(ctx).Return(sessionID).Times(1)
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(tokenKey).Times(1)

				// Mock resume repository for JSON generation
				mockResumeRepo.EXPECT().
					ExtractResumeJsonForRAG(ctx, mock.AnythingOfType("*repositories.ExtractResumeJsonForRAGReq")).
					Return(nil)

				// Mock interview session repository
				mockInterviewSessionRepo.EXPECT().
					CreateInterviewSession(ctx, mock.AnythingOfType("*db.CreateInterviewSessionParams")).
					Return(nil)

				// Mock Redis client
				mockRedisClient.EXPECT().
					Set(ctx, mock.AnythingOfType("database.RedisPayload")).
					Return(nil)

				config := &config.Config{
					InterviewSessionConfig: config.InterviewSessionConfig{
						InterviewSessionTokenTTL: 24 * time.Hour,
					},
				}

				return mockResumeService, mockAuthContext, mockResumeRepo, mockGenerator, mockInterviewSessionRepo, mockJwtMaker, config, mockS3Storage, mockPublisher, mockRedisClient
			},
			verify: func(t *testing.T, gotResp *entities.CreateInterviewSessionWithExistingResumeResp, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, gotResp)
				assert.NotEmpty(t, gotResp.SessionToken)
			},
		},
		{
			name: "Error - Auth context failure",
			input: &entities.CreateInterviewSessionWithExistingResumeReq{
				ResumeID:  resumeID.String(),
				Position:  "Software Engineer",
				IsConsent: true,
			},
			setup: func() (*mockServices.MockResumeService, *mockMiddleware.MockAuthContext, *mockRepositories.MockResumeReposity, *mockUtils.MockGenerator, *mockRepositories.MockInterviewSessionRepository, *mockUtils.MockJwtToken, *config.Config, *mockAws.MockS3Storage, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeService := new(mockServices.MockResumeService)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockResumeRepo := new(mockRepositories.MockResumeReposity)
				mockGenerator := new(mockUtils.MockGenerator)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockJwtMaker := new(mockUtils.MockJwtToken)
				mockS3Storage := new(mockAws.MockS3Storage)
				mockPublisher := new(queue.MockRedisTaskPublisher)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(nil, errors.New("auth failed"))

				config := &config.Config{}

				return mockResumeService, mockAuthContext, mockResumeRepo, mockGenerator, mockInterviewSessionRepo, mockJwtMaker, config, mockS3Storage, mockPublisher, mockRedisClient
			},
			verify: func(t *testing.T, gotResp *entities.CreateInterviewSessionWithExistingResumeResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
				assert.Equal(t, "auth failed", gotErr.Error())
			},
		},
		{
			name: "Error_Invalid Resume ID",
			input: &entities.CreateInterviewSessionWithExistingResumeReq{
				ResumeID:  invalidResumeID,
				Position:  "Software Engineer",
				IsConsent: true,
			},
			setup: func() (*mockServices.MockResumeService, *mockMiddleware.MockAuthContext, *mockRepositories.MockResumeReposity, *mockUtils.MockGenerator, *mockRepositories.MockInterviewSessionRepository, *mockUtils.MockJwtToken, *config.Config, *mockAws.MockS3Storage, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeService := new(mockServices.MockResumeService)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockResumeRepo := new(mockRepositories.MockResumeReposity)
				mockGenerator := new(mockUtils.MockGenerator)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockJwtMaker := new(mockUtils.MockJwtToken)
				mockS3Storage := new(mockAws.MockS3Storage)
				mockPublisher := new(queue.MockRedisTaskPublisher)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				// Mock auth context
				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							ID:     userID,
							UserID: userID.String(),
							Role:   "user",
						},
					}, nil)

				config := &config.Config{
					InterviewSessionConfig: config.InterviewSessionConfig{
						InterviewSessionTokenTTL: 24 * time.Hour,
					},
				}

				return mockResumeService, mockAuthContext, mockResumeRepo, mockGenerator, mockInterviewSessionRepo, mockJwtMaker, config, mockS3Storage, mockPublisher, mockRedisClient
			},
			verify: func(t *testing.T, gotResp *entities.CreateInterviewSessionWithExistingResumeResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "The UUID is invalid. Please try again.")
				assert.Contains(t, gotErr.Error(), "[ONX0107]")
			},
		},
		{
			name: "Error - Get resume by ID failure",
			input: &entities.CreateInterviewSessionWithExistingResumeReq{
				ResumeID:  resumeID.String(),
				Position:  "Software Engineer",
				IsConsent: true,
			},
			setup: func() (*mockServices.MockResumeService, *mockMiddleware.MockAuthContext, *mockRepositories.MockResumeReposity, *mockUtils.MockGenerator, *mockRepositories.MockInterviewSessionRepository, *mockUtils.MockJwtToken, *config.Config, *mockAws.MockS3Storage, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeService := new(mockServices.MockResumeService)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockResumeRepo := new(mockRepositories.MockResumeReposity)
				mockGenerator := new(mockUtils.MockGenerator)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockJwtMaker := new(mockUtils.MockJwtToken)
				mockS3Storage := new(mockAws.MockS3Storage)
				mockPublisher := new(queue.MockRedisTaskPublisher)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							ID:     userID,
							UserID: userID.String(),
							Role:   "user",
						},
					}, nil)

				mockResumeRepo.EXPECT().
					GetResumeByID(ctx, resumeID).
					Return(nil, errors.New("resume not found"))

				config := &config.Config{}

				return mockResumeService, mockAuthContext, mockResumeRepo, mockGenerator, mockInterviewSessionRepo, mockJwtMaker, config, mockS3Storage, mockPublisher, mockRedisClient
			},
			verify: func(t *testing.T, gotResp *entities.CreateInterviewSessionWithExistingResumeResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
				assert.Contains(t, gotErr.Error(), "resume not found")
			},
		},
		{
			name: "Error - S3 download failure",
			input: &entities.CreateInterviewSessionWithExistingResumeReq{
				ResumeID:  resumeID.String(),
				Position:  "Software Engineer",
				IsConsent: true,
			},
			setup: func() (*mockServices.MockResumeService, *mockMiddleware.MockAuthContext, *mockRepositories.MockResumeReposity, *mockUtils.MockGenerator, *mockRepositories.MockInterviewSessionRepository, *mockUtils.MockJwtToken, *config.Config, *mockAws.MockS3Storage, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeService := new(mockServices.MockResumeService)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockResumeRepo := new(mockRepositories.MockResumeReposity)
				mockGenerator := new(mockUtils.MockGenerator)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockJwtMaker := new(mockUtils.MockJwtToken)
				mockS3Storage := new(mockAws.MockS3Storage)
				mockPublisher := new(queue.MockRedisTaskPublisher)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							ID:     userID,
							UserID: userID.String(),
							Role:   "user",
						},
					}, nil)

				mockResumeRepo.EXPECT().
					GetResumeByID(ctx, resumeID).
					Return(&db.Resumes{
						ID:         resumeID,
						UserID:     userID,
						FileName:   "resume.pdf",
						StorageKey: "s3-key-123",
						MimeType:   "application/pdf",
						ByteSize:   1024,
						IsDefault:  false,
						CreatedAt:  time.Now(),
						UpdatedAt:  time.Now(),
					}, nil)

				mockS3Storage.EXPECT().
					DownloadFile(ctx, "s3-key-123").
					Return(nil, errors.New("S3 download failed"))

				config := &config.Config{}

				return mockResumeService, mockAuthContext, mockResumeRepo, mockGenerator, mockInterviewSessionRepo, mockJwtMaker, config, mockS3Storage, mockPublisher, mockRedisClient
			},
			verify: func(t *testing.T, gotResp *entities.CreateInterviewSessionWithExistingResumeResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
				assert.Contains(t, gotErr.Error(), "S3 download failed")
			},
		},
		{
			name: "Error - Get resume JSON failure",
			input: &entities.CreateInterviewSessionWithExistingResumeReq{
				ResumeID:  resumeID.String(),
				Position:  "Software Engineer",
				IsConsent: true,
			},
			setup: func() (*mockServices.MockResumeService, *mockMiddleware.MockAuthContext, *mockRepositories.MockResumeReposity, *mockUtils.MockGenerator, *mockRepositories.MockInterviewSessionRepository, *mockUtils.MockJwtToken, *config.Config, *mockAws.MockS3Storage, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeService := new(mockServices.MockResumeService)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockResumeRepo := new(mockRepositories.MockResumeReposity)
				mockGenerator := new(mockUtils.MockGenerator)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockJwtMaker := new(mockUtils.MockJwtToken)
				mockS3Storage := new(mockAws.MockS3Storage)
				mockPublisher := new(queue.MockRedisTaskPublisher)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							ID:     userID,
							UserID: userID.String(),
							Role:   "user",
						},
					}, nil)

				mockResumeRepo.EXPECT().
					GetResumeByID(ctx, resumeID).
					Return(&db.Resumes{
						ID:         resumeID,
						UserID:     userID,
						FileName:   "resume.pdf",
						StorageKey: "s3-key-123",
						MimeType:   "application/pdf",
						ByteSize:   1024,
						IsDefault:  false,
						CreatedAt:  time.Now(),
						UpdatedAt:  time.Now(),
					}, nil)

				mockS3Storage.EXPECT().
					DownloadFile(ctx, "s3-key-123").
					Return(&aws.CustomFileHeader{
						Filename:    "resume.pdf",
						Size:        1024,
						Header:      map[string][]string{"Content-Type": {"application/pdf"}},
						FileContent: []byte("resume content"),
					}, nil)

				// Mock generator for UUID (called before JSON generation)
				sessionID := uuid.New()
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(sessionID).Times(1)

				mockResumeRepo.EXPECT().
					ExtractResumeJsonForRAG(ctx, mock.AnythingOfType("*repositories.ExtractResumeJsonForRAGReq")).
					Return(errors.New("JSON generation failed"))

				config := &config.Config{}

				return mockResumeService, mockAuthContext, mockResumeRepo, mockGenerator, mockInterviewSessionRepo, mockJwtMaker, config, mockS3Storage, mockPublisher, mockRedisClient
			},
			verify: func(t *testing.T, gotResp *entities.CreateInterviewSessionWithExistingResumeResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
				assert.Contains(t, gotErr.Error(), "JSON generation failed")
			},
		},
		{
			name: "Error - Prompt JSON marshalling failure",
			input: &entities.CreateInterviewSessionWithExistingResumeReq{
				ResumeID:  resumeID.String(),
				Position:  "Software Engineer",
				IsConsent: true,
			},
			setup: func() (*mockServices.MockResumeService, *mockMiddleware.MockAuthContext, *mockRepositories.MockResumeReposity, *mockUtils.MockGenerator, *mockRepositories.MockInterviewSessionRepository, *mockUtils.MockJwtToken, *config.Config, *mockAws.MockS3Storage, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeService := new(mockServices.MockResumeService)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockResumeRepo := new(mockRepositories.MockResumeReposity)
				mockGenerator := new(mockUtils.MockGenerator)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockJwtMaker := new(mockUtils.MockJwtToken)
				mockS3Storage := new(mockAws.MockS3Storage)
				mockPublisher := new(queue.MockRedisTaskPublisher)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							ID:     userID,
							UserID: userID.String(),
							Role:   "user",
						},
					}, nil)

				mockResumeRepo.EXPECT().
					GetResumeByID(ctx, resumeID).
					Return(&db.Resumes{
						ID:         resumeID,
						UserID:     userID,
						FileName:   "resume.pdf",
						StorageKey: "s3-key-123",
						MimeType:   "application/pdf",
						ByteSize:   1024,
						IsDefault:  false,
						CreatedAt:  time.Now(),
						UpdatedAt:  time.Now(),
					}, nil)

				mockS3Storage.EXPECT().
					DownloadFile(ctx, "s3-key-123").
					Return(&aws.CustomFileHeader{
						Filename:    "resume.pdf",
						Size:        1024,
						Header:      map[string][]string{"Content-Type": {"application/pdf"}},
						FileContent: []byte("resume content"),
					}, nil)

				// Mock generator for UUID (called before JSON generation)
				sessionID := uuid.New()
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(sessionID).Times(1)

				mockResumeRepo.EXPECT().
					ExtractResumeJsonForRAG(ctx, mock.AnythingOfType("*repositories.ExtractResumeJsonForRAGReq")).
					Return(nil)

				// Mock interview session repository to return error
				mockInterviewSessionRepo.EXPECT().
					CreateInterviewSession(ctx, mock.AnythingOfType("*db.CreateInterviewSessionParams")).
					Return(errors.New("database transaction failed"))

				config := &config.Config{}

				return mockResumeService, mockAuthContext, mockResumeRepo, mockGenerator, mockInterviewSessionRepo, mockJwtMaker, config, mockS3Storage, mockPublisher, mockRedisClient
			},
			verify: func(t *testing.T, gotResp *entities.CreateInterviewSessionWithExistingResumeResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
				assert.Contains(t, gotErr.Error(), "database transaction failed")
			},
		},
		{
			name: "Error_Invalid user ID",
			input: &entities.CreateInterviewSessionWithExistingResumeReq{
				ResumeID:  resumeID.String(),
				Position:  "Software Engineer",
				IsConsent: true,
			},
			setup: func() (*mockServices.MockResumeService, *mockMiddleware.MockAuthContext, *mockRepositories.MockResumeReposity, *mockUtils.MockGenerator, *mockRepositories.MockInterviewSessionRepository, *mockUtils.MockJwtToken, *config.Config, *mockAws.MockS3Storage, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeService := new(mockServices.MockResumeService)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockResumeRepo := new(mockRepositories.MockResumeReposity)
				mockGenerator := new(mockUtils.MockGenerator)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockJwtMaker := new(mockUtils.MockJwtToken)
				mockS3Storage := new(mockAws.MockS3Storage)
				mockPublisher := new(queue.MockRedisTaskPublisher)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				// Mock auth context
				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							ID:     userID,
							UserID: invalidUserID,
							Role:   "user",
						},
					}, nil)

				// Mock resume repository
				mockResumeRepo.EXPECT().
					GetResumeByID(ctx, resumeID).
					Return(&db.Resumes{
						ID:         resumeID,
						UserID:     userID,
						FileName:   "resume.pdf",
						StorageKey: "s3-key-123",
						MimeType:   "application/pdf",
						ByteSize:   1024,
						IsDefault:  false,
						CreatedAt:  time.Now(),
						UpdatedAt:  time.Now(),
					}, nil)

				// Mock S3 storage
				mockS3Storage.EXPECT().
					DownloadFile(ctx, "s3-key-123").
					Return(&aws.CustomFileHeader{
						Filename:    "resume.pdf",
						Size:        1024,
						Header:      map[string][]string{"Content-Type": {"application/pdf"}},
						FileContent: []byte("resume content"),
					}, nil)

				sessionID := uuid.New()
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(sessionID).Times(1)

				mockResumeRepo.EXPECT().
					ExtractResumeJsonForRAG(ctx, mock.AnythingOfType("*repositories.ExtractResumeJsonForRAGReq")).
					Return(nil)

				config := &config.Config{
					InterviewSessionConfig: config.InterviewSessionConfig{
						InterviewSessionTokenTTL: 24 * time.Hour,
					},
				}

				return mockResumeService, mockAuthContext, mockResumeRepo, mockGenerator, mockInterviewSessionRepo, mockJwtMaker, config, mockS3Storage, mockPublisher, mockRedisClient
			},
			verify: func(t *testing.T, gotResp *entities.CreateInterviewSessionWithExistingResumeResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "The UUID is invalid. Please try again.")
				assert.Contains(t, gotErr.Error(), "[ONX0107]")
			},
		},
		{
			name: "Error - Transaction failure",
			input: &entities.CreateInterviewSessionWithExistingResumeReq{
				ResumeID:  resumeID.String(),
				Position:  "Software Engineer",
				IsConsent: true,
			},
			setup: func() (*mockServices.MockResumeService, *mockMiddleware.MockAuthContext, *mockRepositories.MockResumeReposity, *mockUtils.MockGenerator, *mockRepositories.MockInterviewSessionRepository, *mockUtils.MockJwtToken, *config.Config, *mockAws.MockS3Storage, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeService := new(mockServices.MockResumeService)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockResumeRepo := new(mockRepositories.MockResumeReposity)
				mockGenerator := new(mockUtils.MockGenerator)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockJwtMaker := new(mockUtils.MockJwtToken)
				mockS3Storage := new(mockAws.MockS3Storage)
				mockPublisher := new(queue.MockRedisTaskPublisher)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							ID:     userID,
							UserID: userID.String(),
							Role:   "user",
						},
					}, nil)

				mockResumeRepo.EXPECT().
					GetResumeByID(ctx, resumeID).
					Return(&db.Resumes{
						ID:         resumeID,
						UserID:     userID,
						FileName:   "resume.pdf",
						StorageKey: "s3-key-123",
						MimeType:   "application/pdf",
						ByteSize:   1024,
						IsDefault:  false,
						CreatedAt:  time.Now(),
						UpdatedAt:  time.Now(),
					}, nil)

				mockS3Storage.EXPECT().
					DownloadFile(ctx, "s3-key-123").
					Return(&aws.CustomFileHeader{
						Filename:    "resume.pdf",
						Size:        1024,
						Header:      map[string][]string{"Content-Type": {"application/pdf"}},
						FileContent: []byte("resume content"),
					}, nil)

				// Mock generator for UUID (called before JSON generation)
				sessionID := uuid.New()
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(sessionID).Times(1)

				mockResumeRepo.EXPECT().
					ExtractResumeJsonForRAG(ctx, mock.AnythingOfType("*repositories.ExtractResumeJsonForRAGReq")).
					Return(nil)

				mockInterviewSessionRepo.EXPECT().
					CreateInterviewSession(ctx, mock.AnythingOfType("*db.CreateInterviewSessionParams")).
					Return(errors.New("transaction failed"))

				config := &config.Config{}

				return mockResumeService, mockAuthContext, mockResumeRepo, mockGenerator, mockInterviewSessionRepo, mockJwtMaker, config, mockS3Storage, mockPublisher, mockRedisClient
			},
			verify: func(t *testing.T, gotResp *entities.CreateInterviewSessionWithExistingResumeResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
				assert.Contains(t, gotErr.Error(), "transaction failed")
			},
		},
		{
			name: "Error - Redis connection failure",
			input: &entities.CreateInterviewSessionWithExistingResumeReq{
				ResumeID:  resumeID.String(),
				Position:  "Software Engineer",
				IsConsent: true,
			},
			setup: func() (*mockServices.MockResumeService, *mockMiddleware.MockAuthContext, *mockRepositories.MockResumeReposity, *mockUtils.MockGenerator, *mockRepositories.MockInterviewSessionRepository, *mockUtils.MockJwtToken, *config.Config, *mockAws.MockS3Storage, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeService := new(mockServices.MockResumeService)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockResumeRepo := new(mockRepositories.MockResumeReposity)
				mockGenerator := new(mockUtils.MockGenerator)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockJwtMaker := new(mockUtils.MockJwtToken)
				mockS3Storage := new(mockAws.MockS3Storage)
				mockPublisher := new(queue.MockRedisTaskPublisher)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							ID:     userID,
							UserID: userID.String(),
							Role:   "user",
						},
					}, nil)

				mockResumeRepo.EXPECT().
					GetResumeByID(ctx, resumeID).
					Return(&db.Resumes{
						ID:         resumeID,
						UserID:     userID,
						FileName:   "resume.pdf",
						StorageKey: "s3-key-123",
						MimeType:   "application/pdf",
						ByteSize:   1024,
						IsDefault:  false,
						CreatedAt:  time.Now(),
						UpdatedAt:  time.Now(),
					}, nil)

				mockS3Storage.EXPECT().
					DownloadFile(ctx, "s3-key-123").
					Return(&aws.CustomFileHeader{
						Filename:    "resume.pdf",
						Size:        1024,
						Header:      map[string][]string{"Content-Type": {"application/pdf"}},
						FileContent: []byte("resume content"),
					}, nil)

				// Mock generator for UUID (called before JSON generation)
				sessionID := uuid.New()
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(sessionID).Times(1)

				mockResumeRepo.EXPECT().
					ExtractResumeJsonForRAG(ctx, mock.AnythingOfType("*repositories.ExtractResumeJsonForRAGReq")).
					Return(nil)

				// Mock generator for remaining UUIDs

				mockInterviewSessionRepo.EXPECT().
					CreateInterviewSession(ctx, mock.AnythingOfType("*db.CreateInterviewSessionParams")).
					Return(nil)

				// Mock generator for token key
				tokenKey := uuid.New()
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(tokenKey).Times(1)

				// Mock Redis client to return error
				mockRedisClient.EXPECT().
					Set(ctx, mock.AnythingOfType("database.RedisPayload")).
					Return(errors.New("Redis connection failed"))

				config := &config.Config{
					InterviewSessionConfig: config.InterviewSessionConfig{
						InterviewSessionTokenTTL: 24 * time.Hour,
					},
				}

				return mockResumeService, mockAuthContext, mockResumeRepo, mockGenerator, mockInterviewSessionRepo, mockJwtMaker, config, mockS3Storage, mockPublisher, mockRedisClient
			},
			verify: func(t *testing.T, gotResp *entities.CreateInterviewSessionWithExistingResumeResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
				assert.Contains(t, gotErr.Error(), "Redis connection failed")
			},
		},
		{
			name: "Error - Redis set failure",
			input: &entities.CreateInterviewSessionWithExistingResumeReq{
				ResumeID:  resumeID.String(),
				Position:  "Software Engineer",
				IsConsent: true,
			},
			setup: func() (*mockServices.MockResumeService, *mockMiddleware.MockAuthContext, *mockRepositories.MockResumeReposity, *mockUtils.MockGenerator, *mockRepositories.MockInterviewSessionRepository, *mockUtils.MockJwtToken, *config.Config, *mockAws.MockS3Storage, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeService := new(mockServices.MockResumeService)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockResumeRepo := new(mockRepositories.MockResumeReposity)
				mockGenerator := new(mockUtils.MockGenerator)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockJwtMaker := new(mockUtils.MockJwtToken)
				mockS3Storage := new(mockAws.MockS3Storage)
				mockPublisher := new(queue.MockRedisTaskPublisher)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							ID:     userID,
							UserID: userID.String(),
							Role:   "user",
						},
					}, nil)

				mockResumeRepo.EXPECT().
					GetResumeByID(ctx, resumeID).
					Return(&db.Resumes{
						ID:         resumeID,
						UserID:     userID,
						FileName:   "resume.pdf",
						StorageKey: "s3-key-123",
						MimeType:   "application/pdf",
						ByteSize:   1024,
						IsDefault:  false,
						CreatedAt:  time.Now(),
						UpdatedAt:  time.Now(),
					}, nil)

				mockS3Storage.EXPECT().
					DownloadFile(ctx, "s3-key-123").
					Return(&aws.CustomFileHeader{
						Filename:    "resume.pdf",
						Size:        1024,
						Header:      map[string][]string{"Content-Type": {"application/pdf"}},
						FileContent: []byte("resume content"),
					}, nil)

				// Mock generator for UUID (called before JSON generation)
				sessionID := uuid.New()
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(sessionID).Times(1)

				mockResumeRepo.EXPECT().
					ExtractResumeJsonForRAG(ctx, mock.AnythingOfType("*repositories.ExtractResumeJsonForRAGReq")).
					Return(nil)

				// Mock generator for remaining UUIDs
				tokenKey := uuid.New()

				mockGenerator.EXPECT().GenerateUUID(ctx).Return(tokenKey).Times(1)

				mockInterviewSessionRepo.EXPECT().
					CreateInterviewSession(ctx, mock.AnythingOfType("*db.CreateInterviewSessionParams")).
					Return(nil)

				mockRedisClient.EXPECT().
					Set(ctx, mock.AnythingOfType("database.RedisPayload")).
					Return(errors.New("Redis set failed"))

				config := &config.Config{
					InterviewSessionConfig: config.InterviewSessionConfig{
						InterviewSessionTokenTTL: 24 * time.Hour,
					},
				}

				return mockResumeService, mockAuthContext, mockResumeRepo, mockGenerator, mockInterviewSessionRepo, mockJwtMaker, config, mockS3Storage, mockPublisher, mockRedisClient
			},
			verify: func(t *testing.T, gotResp *entities.CreateInterviewSessionWithExistingResumeResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
				assert.Contains(t, gotErr.Error(), "Redis set failed")
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			mockResumeService, mockAuthContext, mockResumeRepo, mockGenerator, mockInterviewSessionRepo, mockJwtMaker, config, mockS3Storage, mockPublisher, mockRedisClient := tC.setup()
			defer func() {
				if mockResumeService != nil {
					mockResumeService.AssertExpectations(t)
				}
				if mockAuthContext != nil {
					mockAuthContext.AssertExpectations(t)
				}
				if mockResumeRepo != nil {
					mockResumeRepo.AssertExpectations(t)
				}
				if mockGenerator != nil {
					mockGenerator.AssertExpectations(t)
				}
				if mockInterviewSessionRepo != nil {
					mockInterviewSessionRepo.AssertExpectations(t)
				}
				if mockJwtMaker != nil {
					mockJwtMaker.AssertExpectations(t)
				}
				if mockS3Storage != nil {
					mockS3Storage.AssertExpectations(t)
				}
				if mockPublisher != nil {
					mockPublisher.AssertExpectations(t)
				}
				if mockRedisClient != nil {
					mockRedisClient.AssertExpectations(t)
				}
			}()

			svc := NewInterviewSessionService(
				lgr,
				mockAuthContext,
				mockResumeRepo,
				mockGenerator,
				mockInterviewSessionRepo,
				mockJwtMaker,
				config,
				mockS3Storage,
				mockPublisher,
				mockRedisClient,
				nil,
				nil,
				nil,
			)

			gotResp, gotErr := svc.CreateInterviewSessionWithExistingResume(ctx, tC.input)

			tC.verify(t, gotResp, gotErr)
		})
	}
}

func TestInterviewSessionService_UpdateInterviewSessionStatus(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()
	sessionID := uuid.New()
	invalidSessionID := "invalid-session-id"

	testCases := []struct {
		name   string
		input  *entities.UpdateInterviewSessionStatusReq
		setup  func() (*mockServices.MockResumeService, *mockMiddleware.MockAuthContext, *mockRepositories.MockResumeReposity, *mockUtils.MockGenerator, *mockRepositories.MockInterviewSessionRepository, *mockUtils.MockJwtToken, *config.Config, *mockAws.MockS3Storage, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient)
		verify func(t *testing.T, gotErr error)
	}{
		{
			name: "Success",
			input: &entities.UpdateInterviewSessionStatusReq{
				SessionID: sessionID.String(),
			},
			setup: func() (*mockServices.MockResumeService, *mockMiddleware.MockAuthContext, *mockRepositories.MockResumeReposity, *mockUtils.MockGenerator, *mockRepositories.MockInterviewSessionRepository, *mockUtils.MockJwtToken, *config.Config, *mockAws.MockS3Storage, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeService := new(mockServices.MockResumeService)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockResumeRepo := new(mockRepositories.MockResumeReposity)
				mockGenerator := new(mockUtils.MockGenerator)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockJwtMaker := new(mockUtils.MockJwtToken)
				mockS3Storage := new(mockAws.MockS3Storage)
				mockPublisher := new(queue.MockRedisTaskPublisher)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockInterviewSessionRepo.EXPECT().
					UpdateInterviewSessionStatus(ctx, mock.AnythingOfType("*db.UpdateInterviewSessionStatusParams")).
					Return(nil)

				config := &config.Config{}

				return mockResumeService, mockAuthContext, mockResumeRepo, mockGenerator, mockInterviewSessionRepo, mockJwtMaker, config, mockS3Storage, mockPublisher, mockRedisClient
			},
			verify: func(t *testing.T, gotErr error) {
				assert.NoError(t, gotErr)
			},
		},
		{
			name: "Error_Invalid session ID",
			input: &entities.UpdateInterviewSessionStatusReq{
				SessionID: invalidSessionID,
			},
			setup: func() (*mockServices.MockResumeService, *mockMiddleware.MockAuthContext, *mockRepositories.MockResumeReposity, *mockUtils.MockGenerator, *mockRepositories.MockInterviewSessionRepository, *mockUtils.MockJwtToken, *config.Config, *mockAws.MockS3Storage, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeService := new(mockServices.MockResumeService)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockResumeRepo := new(mockRepositories.MockResumeReposity)
				mockGenerator := new(mockUtils.MockGenerator)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockJwtMaker := new(mockUtils.MockJwtToken)
				mockS3Storage := new(mockAws.MockS3Storage)
				mockPublisher := new(queue.MockRedisTaskPublisher)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				config := &config.Config{}

				return mockResumeService, mockAuthContext, mockResumeRepo, mockGenerator, mockInterviewSessionRepo, mockJwtMaker, config, mockS3Storage, mockPublisher, mockRedisClient
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "The UUID is invalid. Please try again.")
				assert.Contains(t, gotErr.Error(), "[ONX0107]")
			},
		},
		{
			name: "Error - Update interview session status failure",
			input: &entities.UpdateInterviewSessionStatusReq{
				SessionID: sessionID.String(),
			},
			setup: func() (*mockServices.MockResumeService, *mockMiddleware.MockAuthContext, *mockRepositories.MockResumeReposity, *mockUtils.MockGenerator, *mockRepositories.MockInterviewSessionRepository, *mockUtils.MockJwtToken, *config.Config, *mockAws.MockS3Storage, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeService := new(mockServices.MockResumeService)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockResumeRepo := new(mockRepositories.MockResumeReposity)
				mockGenerator := new(mockUtils.MockGenerator)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockJwtMaker := new(mockUtils.MockJwtToken)
				mockS3Storage := new(mockAws.MockS3Storage)
				mockPublisher := new(queue.MockRedisTaskPublisher)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockInterviewSessionRepo.EXPECT().
					UpdateInterviewSessionStatus(ctx, mock.AnythingOfType("*db.UpdateInterviewSessionStatusParams")).
					Return(errors.New("database error"))

				config := &config.Config{}

				return mockResumeService, mockAuthContext, mockResumeRepo, mockGenerator, mockInterviewSessionRepo, mockJwtMaker, config, mockS3Storage, mockPublisher, mockRedisClient
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Equal(t, "database error", gotErr.Error())
			},
		},
		{
			name: "Error_Invalid session ID format",
			input: &entities.UpdateInterviewSessionStatusReq{
				SessionID: "invalid-uuid",
			},
			setup: func() (*mockServices.MockResumeService, *mockMiddleware.MockAuthContext, *mockRepositories.MockResumeReposity, *mockUtils.MockGenerator, *mockRepositories.MockInterviewSessionRepository, *mockUtils.MockJwtToken, *config.Config, *mockAws.MockS3Storage, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeService := new(mockServices.MockResumeService)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockResumeRepo := new(mockRepositories.MockResumeReposity)
				mockGenerator := new(mockUtils.MockGenerator)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockJwtMaker := new(mockUtils.MockJwtToken)
				mockS3Storage := new(mockAws.MockS3Storage)
				mockPublisher := new(queue.MockRedisTaskPublisher)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				config := &config.Config{}

				return mockResumeService, mockAuthContext, mockResumeRepo, mockGenerator, mockInterviewSessionRepo, mockJwtMaker, config, mockS3Storage, mockPublisher, mockRedisClient
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "The UUID is invalid. Please try again.")
				assert.Contains(t, gotErr.Error(), "[ONX0107]")
			},
		},
		{
			name: "Error - Empty session ID",
			input: &entities.UpdateInterviewSessionStatusReq{
				SessionID: "",
			},
			setup: func() (*mockServices.MockResumeService, *mockMiddleware.MockAuthContext, *mockRepositories.MockResumeReposity, *mockUtils.MockGenerator, *mockRepositories.MockInterviewSessionRepository, *mockUtils.MockJwtToken, *config.Config, *mockAws.MockS3Storage, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeService := new(mockServices.MockResumeService)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockResumeRepo := new(mockRepositories.MockResumeReposity)
				mockGenerator := new(mockUtils.MockGenerator)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockJwtMaker := new(mockUtils.MockJwtToken)
				mockS3Storage := new(mockAws.MockS3Storage)
				mockPublisher := new(queue.MockRedisTaskPublisher)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				config := &config.Config{}

				return mockResumeService, mockAuthContext, mockResumeRepo, mockGenerator, mockInterviewSessionRepo, mockJwtMaker, config, mockS3Storage, mockPublisher, mockRedisClient
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "The UUID is invalid. Please try again.")
				assert.Contains(t, gotErr.Error(), "[ONX0107]")
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			mockResumeService, mockAuthContext, mockResumeRepo, mockGenerator, mockInterviewSessionRepo, mockJwtMaker, config, mockS3Storage, mockPublisher, mockRedisClient := tC.setup()
			defer func() {
				if mockResumeService != nil {
					mockResumeService.AssertExpectations(t)
				}
				if mockAuthContext != nil {
					mockAuthContext.AssertExpectations(t)
				}
				if mockResumeRepo != nil {
					mockResumeRepo.AssertExpectations(t)
				}
				if mockGenerator != nil {
					mockGenerator.AssertExpectations(t)
				}
				if mockInterviewSessionRepo != nil {
					mockInterviewSessionRepo.AssertExpectations(t)
				}
				if mockJwtMaker != nil {
					mockJwtMaker.AssertExpectations(t)
				}
				if mockS3Storage != nil {
					mockS3Storage.AssertExpectations(t)
				}
				if mockPublisher != nil {
					mockPublisher.AssertExpectations(t)
				}
				if mockRedisClient != nil {
					mockRedisClient.AssertExpectations(t)
				}
			}()

			svc := NewInterviewSessionService(
				lgr,
				mockAuthContext,
				mockResumeRepo,
				mockGenerator,
				mockInterviewSessionRepo,
				mockJwtMaker,
				config,
				mockS3Storage,
				mockPublisher,
				mockRedisClient,
				nil,
				nil,
				nil,
			)

			gotErr := svc.UpdateInterviewSessionStatus(ctx, tC.input)

			tC.verify(t, gotErr)
		})
	}
}

func TestInterviewSessionService_IsSessionValid(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()

	testCases := []struct {
		name   string
		input  *entities.IsSessionValidReq
		setup  func() (*mockDatabase.MockRedisClient, *mockRepositories.MockInterviewSessionRepository)
		verify func(t *testing.T, gotResp *entities.IsSessionValidResp, gotErr error)
	}{
		{
			name: "Success",
			input: &entities.IsSessionValidReq{
				SessionToken: "valid_session_token",
				UserID:       "user_id",
			},
			setup: func() (*mockDatabase.MockRedisClient, *mockRepositories.MockInterviewSessionRepository) {
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)

				// Use a valid UUID for session_id
				validSessionID := "550e8400-e29b-41d4-a716-446655440000"

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewSessionToken, "valid_session_token")).
					Return("{\"user_id\":\"user_id\",\"session_id\":\""+validSessionID+"\"}", nil)

				mockInterviewSessionRepo.EXPECT().
					CheckInterviewSessionExists(ctx, mock.Anything).
					Return(true, nil)

				return mockRedisClient, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotResp *entities.IsSessionValidResp, gotErr error) {
				assert.NoError(t, gotErr)
				assert.Equal(t, "user_id", gotResp.UserID)
				assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", gotResp.SessionID)
			},
		},
		{
			name: "Error - Redis error",
			input: &entities.IsSessionValidReq{
				SessionToken: "valid_session_token",
				UserID:       "user_id",
			},
			setup: func() (*mockDatabase.MockRedisClient, *mockRepositories.MockInterviewSessionRepository) {
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewSessionToken, "valid_session_token")).
					Return("", errors.New("redis error"))

				return mockRedisClient, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotResp *entities.IsSessionValidResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
			},
		},
		{
			name: "Error - Unmarshal error",
			input: &entities.IsSessionValidReq{
				SessionToken: "valid_session_token",
				UserID:       "user_id",
			},
			setup: func() (*mockDatabase.MockRedisClient, *mockRepositories.MockInterviewSessionRepository) {
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewSessionToken, "valid_session_token")).
					Return("invalid_json", nil)

				return mockRedisClient, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotResp *entities.IsSessionValidResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
			},
		},
		{
			name: "Error - User ID mismatch",
			input: &entities.IsSessionValidReq{
				SessionToken: "valid_session_token",
				UserID:       "user_id",
			},
			setup: func() (*mockDatabase.MockRedisClient, *mockRepositories.MockInterviewSessionRepository) {
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)

				// Use a valid UUID for session_id
				validSessionID := "550e8400-e29b-41d4-a716-446655440000"

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewSessionToken, "valid_session_token")).
					Return("{\"user_id\":\"user_id_mismatch\",\"session_id\":\""+validSessionID+"\"}", nil)

				return mockRedisClient, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotResp *entities.IsSessionValidResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
			},
		},
		{
			name: "Error_Invalid session ID",
			input: &entities.IsSessionValidReq{
				SessionToken: "valid_session_token",
				UserID:       "user_id",
			},
			setup: func() (*mockDatabase.MockRedisClient, *mockRepositories.MockInterviewSessionRepository) {
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)

				// Use a valid UUID for session_id
				inValidSessionID := "invalid-session-id"

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewSessionToken, "valid_session_token")).
					Return("{\"user_id\":\"user_id\",\"session_id\":\""+inValidSessionID+"\"}", nil)

				return mockRedisClient, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotResp *entities.IsSessionValidResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "The UUID is invalid. Please try again.")
				assert.Contains(t, gotErr.Error(), "[ONX0107]")
			},
		},
		{
			name: "Error - Interview session error",
			input: &entities.IsSessionValidReq{
				SessionToken: "valid_session_token",
				UserID:       "user_id",
			},
			setup: func() (*mockDatabase.MockRedisClient, *mockRepositories.MockInterviewSessionRepository) {
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)

				// Use a valid UUID for session_id
				validSessionID := "550e8400-e29b-41d4-a716-446655440000"

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewSessionToken, "valid_session_token")).
					Return("{\"user_id\":\"user_id\",\"session_id\":\""+validSessionID+"\"}", nil)

				mockInterviewSessionRepo.EXPECT().
					CheckInterviewSessionExists(ctx, mock.Anything).
					Return(false, errors.New("database error"))

				return mockRedisClient, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotResp *entities.IsSessionValidResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
			},
		},
		{
			name: "Error - Interview session not found",
			input: &entities.IsSessionValidReq{
				SessionToken: "valid_session_token",
				UserID:       "user_id",
			},
			setup: func() (*mockDatabase.MockRedisClient, *mockRepositories.MockInterviewSessionRepository) {
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)

				// Use a valid UUID for session_id
				validSessionID := "550e8400-e29b-41d4-a716-446655440000"

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewSessionToken, "valid_session_token")).
					Return("{\"user_id\":\"user_id\",\"session_id\":\""+validSessionID+"\"}", nil)

				mockInterviewSessionRepo.EXPECT().
					CheckInterviewSessionExists(ctx, mock.Anything).
					Return(false, nil)

				return mockRedisClient, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotResp *entities.IsSessionValidResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			mockRedisClient, mockInterviewSessionRepo := tC.setup()
			defer func() {
				if mockRedisClient != nil {
					mockRedisClient.AssertExpectations(t)
				}
				if mockInterviewSessionRepo != nil {
					mockInterviewSessionRepo.AssertExpectations(t)
				}
			}()

			svc := NewInterviewSessionService(
				lgr,
				nil,
				nil,
				nil,
				mockInterviewSessionRepo,
				nil,
				nil,
				nil,
				nil,
				mockRedisClient,
				nil,
				nil,
				nil,
			)

			gotResp, gotErr := svc.IsSessionValid(ctx, tC.input)

			tC.verify(t, gotResp, gotErr)
		})
	}
}

func TestInterviewSessionService_CreateUserSessionTurnBySessionID(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()
	correctSessionID := uuid.New().String()
	correctTurnID := uuid.New().String()
	invalidID := "invalid-id"
	currentState := "Greeting"

	testCases := []struct {
		name   string
		input  *entities.CreateUserSessionTurnBySessionIDReq
		setup  func() (*mockUtils.MockGenerator, *mockDatabase.MockRedisClient, *mockRepositories.MockInterviewSessionRepository, *mockRepositories.MockInterviewTurnsRepository)
		verify func(t *testing.T, gotErr error)
	}{
		{
			name: "Success - TurnRedisTriggered",
			input: &entities.CreateUserSessionTurnBySessionIDReq{
				SessionID:    correctSessionID,
				TurnID:       correctTurnID,
				Transcript:   "transcript",
				CurrentState: currentState,
			},
			setup: func() (*mockUtils.MockGenerator, *mockDatabase.MockRedisClient, *mockRepositories.MockInterviewSessionRepository, *mockRepositories.MockInterviewTurnsRepository) {
				mockGenerator := new(mockUtils.MockGenerator)
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockInterviewTurnsRepo := new(mockRepositories.MockInterviewTurnsRepository)

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewMaxTurnNo, correctSessionID)).
					Return("1", nil)

				mockRedisClient.EXPECT().
					HGetAll(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewStartEndTime, correctSessionID)).
					Return(map[string]string{
						"started_at": "0.01",
						"ended_at":   "2.34",
					}, nil)

				mockInterviewTurnsRepo.EXPECT().
					CreateSessionTurnBySessionID(ctx, mock.MatchedBy(func(req *db.CreateInterviewTurnParams) bool {
						return req.SessionID.String() == correctSessionID &&
							req.ID.String() == correctTurnID &&
							req.TurnNo == 2 &&
							req.CurrentState == currentState &&
							req.Actor == "user" &&
							req.TranscriptText == "transcript" &&
							req.StartAt == "0.01" &&
							req.EndAt == "2.34"
					})).
					Return(nil)

				mockRedisClient.EXPECT().
					Set(ctx, database.RedisPayload{
						Key:   fmt.Sprintf("%s%s", constants.RedisPrefixInterviewMaxTurnNo, correctSessionID),
						Value: int64(2),
						TTL:   constants.RedisTTLInterviewTurn,
					}).
					Return(nil)

				return mockGenerator, mockRedisClient, mockInterviewSessionRepo, mockInterviewTurnsRepo
			},
			verify: func(t *testing.T, gotErr error) {
				assert.NoError(t, gotErr)
			},
		},
		{
			name: "Error_Invalid session ID",
			input: &entities.CreateUserSessionTurnBySessionIDReq{
				SessionID:  correctSessionID,
				TurnID:     invalidID,
				Transcript: "transcript",
			},
			setup: func() (*mockUtils.MockGenerator, *mockDatabase.MockRedisClient, *mockRepositories.MockInterviewSessionRepository, *mockRepositories.MockInterviewTurnsRepository) {

				return nil, nil, nil, nil
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "The UUID is invalid. Please try again.")
				assert.Contains(t, gotErr.Error(), "[ONX0107]")
			},
		},

		{
			name: "Error_Invalid turn ID",
			input: &entities.CreateUserSessionTurnBySessionIDReq{
				SessionID:    invalidID,
				TurnID:       correctTurnID,
				Transcript:   "transcript",
				CurrentState: currentState,
			},
			setup: func() (*mockUtils.MockGenerator, *mockDatabase.MockRedisClient, *mockRepositories.MockInterviewSessionRepository, *mockRepositories.MockInterviewTurnsRepository) {

				return nil, nil, nil, nil
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "The UUID is invalid. Please try again.")
				assert.Contains(t, gotErr.Error(), "[ONX0107]")
			},
		},
		{
			name: "Success - TurnRedisNotTriggered",
			input: &entities.CreateUserSessionTurnBySessionIDReq{
				SessionID:    correctSessionID,
				TurnID:       correctTurnID,
				Transcript:   "transcript",
				CurrentState: currentState,
			},
			setup: func() (*mockUtils.MockGenerator, *mockDatabase.MockRedisClient, *mockRepositories.MockInterviewSessionRepository, *mockRepositories.MockInterviewTurnsRepository) {
				mockGenerator := new(mockUtils.MockGenerator)
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockInterviewTurnsRepo := new(mockRepositories.MockInterviewTurnsRepository)

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewMaxTurnNo, correctSessionID)).
					Return("", redis.Nil)

				mockInterviewTurnsRepo.EXPECT().
					GetMaxTurnNoBySessionID(ctx, uuid.MustParse(correctSessionID)).
					Return(int64(2), nil)

				mockRedisClient.EXPECT().
					HGetAll(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewStartEndTime, correctSessionID)).
					Return(map[string]string{
						"started_at": "0.01",
						"ended_at":   "2.34",
					}, nil)

				mockInterviewTurnsRepo.EXPECT().
					CreateSessionTurnBySessionID(ctx, mock.MatchedBy(func(req *db.CreateInterviewTurnParams) bool {
						return req.SessionID.String() == correctSessionID &&
							req.ID.String() == correctTurnID &&
							req.TurnNo == 3 &&
							req.Actor == "user" &&
							req.TranscriptText == "transcript" &&
							req.StartAt == "0.01" &&
							req.EndAt == "2.34"
					})).
					Return(nil)

				mockRedisClient.EXPECT().
					Set(ctx, database.RedisPayload{
						Key:   fmt.Sprintf("%s%s", constants.RedisPrefixInterviewMaxTurnNo, correctSessionID),
						Value: int64(3),
						TTL:   constants.RedisTTLInterviewTurn,
					}).
					Return(nil)

				return mockGenerator, mockRedisClient, mockInterviewSessionRepo, mockInterviewTurnsRepo
			},
			verify: func(t *testing.T, gotErr error) {
				assert.NoError(t, gotErr)
			},
		},
		{
			name: "Success - With Warning Set Redis Error",
			input: &entities.CreateUserSessionTurnBySessionIDReq{
				SessionID:    correctSessionID,
				TurnID:       correctTurnID,
				Transcript:   "transcript",
				CurrentState: currentState,
			},
			setup: func() (*mockUtils.MockGenerator, *mockDatabase.MockRedisClient, *mockRepositories.MockInterviewSessionRepository, *mockRepositories.MockInterviewTurnsRepository) {
				mockGenerator := new(mockUtils.MockGenerator)
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockInterviewTurnsRepo := new(mockRepositories.MockInterviewTurnsRepository)

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewMaxTurnNo, correctSessionID)).
					Return("1", nil)

				mockRedisClient.EXPECT().
					HGetAll(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewStartEndTime, correctSessionID)).
					Return(map[string]string{
						"started_at": "0.01",
						"ended_at":   "2.34",
					}, nil)

				mockInterviewTurnsRepo.EXPECT().
					CreateSessionTurnBySessionID(ctx, mock.MatchedBy(func(req *db.CreateInterviewTurnParams) bool {
						return req.SessionID.String() == correctSessionID &&
							req.ID.String() == correctTurnID &&
							req.TurnNo == 2 &&
							req.Actor == "user" &&
							req.TranscriptText == "transcript" &&
							req.StartAt == "0.01" &&
							req.EndAt == "2.34"
					})).
					Return(nil)

				mockRedisClient.EXPECT().
					Set(ctx, database.RedisPayload{
						Key:   fmt.Sprintf("%s%s", constants.RedisPrefixInterviewMaxTurnNo, correctSessionID),
						Value: int64(2),
						TTL:   constants.RedisTTLInterviewTurn,
					}).
					Return(errors.New("redis error"))

				return mockGenerator, mockRedisClient, mockInterviewSessionRepo, mockInterviewTurnsRepo
			},
			verify: func(t *testing.T, gotErr error) {
				assert.NoError(t, gotErr)
			},
		},
		{
			name: "Error - Redis Get Turn error",
			input: &entities.CreateUserSessionTurnBySessionIDReq{
				SessionID:    correctSessionID,
				TurnID:       correctTurnID,
				Transcript:   "transcript",
				CurrentState: currentState,
			},
			setup: func() (*mockUtils.MockGenerator, *mockDatabase.MockRedisClient, *mockRepositories.MockInterviewSessionRepository, *mockRepositories.MockInterviewTurnsRepository) {
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewMaxTurnNo, correctSessionID)).
					Return("", errors.New("redis error"))

				return nil, mockRedisClient, nil, nil
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
			},
		},
		{
			name: "Error - GetMaxTurnRepo Error",
			input: &entities.CreateUserSessionTurnBySessionIDReq{
				SessionID:    correctSessionID,
				TurnID:       correctTurnID,
				Transcript:   "transcript",
				CurrentState: currentState,
			},
			setup: func() (*mockUtils.MockGenerator, *mockDatabase.MockRedisClient, *mockRepositories.MockInterviewSessionRepository, *mockRepositories.MockInterviewTurnsRepository) {
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockInterviewTurnsRepo := new(mockRepositories.MockInterviewTurnsRepository)

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewMaxTurnNo, correctSessionID)).
					Return("", redis.Nil)

				mockInterviewTurnsRepo.EXPECT().
					GetMaxTurnNoBySessionID(ctx, uuid.MustParse(correctSessionID)).
					Return(0, errors.New("database error"))

				return nil, mockRedisClient, mockInterviewSessionRepo, mockInterviewTurnsRepo
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
			},
		},
		{
			name: "Error - ParseIntTurnNo Error",
			input: &entities.CreateUserSessionTurnBySessionIDReq{
				SessionID:    correctSessionID,
				TurnID:       correctTurnID,
				Transcript:   "transcript",
				CurrentState: currentState,
			},
			setup: func() (*mockUtils.MockGenerator, *mockDatabase.MockRedisClient, *mockRepositories.MockInterviewSessionRepository, *mockRepositories.MockInterviewTurnsRepository) {
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewMaxTurnNo, correctSessionID)).
					Return("invalid_turn_no", nil)

				return nil, mockRedisClient, nil, nil
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
			},
		},
		{
			name: "Error - GetStartEndTime Redis Error",
			input: &entities.CreateUserSessionTurnBySessionIDReq{
				SessionID:    correctSessionID,
				TurnID:       correctTurnID,
				Transcript:   "transcript",
				CurrentState: currentState,
			},
			setup: func() (*mockUtils.MockGenerator, *mockDatabase.MockRedisClient, *mockRepositories.MockInterviewSessionRepository, *mockRepositories.MockInterviewTurnsRepository) {
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewMaxTurnNo, correctSessionID)).
					Return("0", nil)

				mockRedisClient.EXPECT().
					HGetAll(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewStartEndTime, correctSessionID)).
					Return(nil, errors.New("redis error"))

				return nil, mockRedisClient, nil, nil
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
			},
		},
		{
			name: "Error - GetStartEndTime Redis Nil",
			input: &entities.CreateUserSessionTurnBySessionIDReq{
				SessionID:    correctSessionID,
				TurnID:       correctTurnID,
				Transcript:   "transcript",
				CurrentState: currentState,
			},
			setup: func() (*mockUtils.MockGenerator, *mockDatabase.MockRedisClient, *mockRepositories.MockInterviewSessionRepository, *mockRepositories.MockInterviewTurnsRepository) {
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewMaxTurnNo, correctSessionID)).
					Return("0", nil)

				mockRedisClient.EXPECT().
					HGetAll(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewStartEndTime, correctSessionID)).
					Return(nil, redis.Nil)

				return nil, mockRedisClient, nil, nil
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
			},
		},
		{
			name: "Error - GetStartEndTime Not Found Any Field",
			input: &entities.CreateUserSessionTurnBySessionIDReq{
				SessionID:    correctSessionID,
				TurnID:       correctTurnID,
				Transcript:   "transcript",
				CurrentState: currentState,
			},
			setup: func() (*mockUtils.MockGenerator, *mockDatabase.MockRedisClient, *mockRepositories.MockInterviewSessionRepository, *mockRepositories.MockInterviewTurnsRepository) {
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewMaxTurnNo, correctSessionID)).
					Return("0", nil)

				mockRedisClient.EXPECT().
					HGetAll(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewStartEndTime, correctSessionID)).
					Return(map[string]string{}, nil)

				return nil, mockRedisClient, nil, nil
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), constants.ErrInterviewSessionStartEndTimeNotFound.Error())
			},
		},
		{
			name: "Error - GetStartEndTime StartedAt Not Found Error",
			input: &entities.CreateUserSessionTurnBySessionIDReq{
				SessionID:    correctSessionID,
				TurnID:       correctTurnID,
				Transcript:   "transcript",
				CurrentState: currentState,
			},
			setup: func() (*mockUtils.MockGenerator, *mockDatabase.MockRedisClient, *mockRepositories.MockInterviewSessionRepository, *mockRepositories.MockInterviewTurnsRepository) {
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewMaxTurnNo, correctSessionID)).
					Return("0", nil)

				mockRedisClient.EXPECT().
					HGetAll(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewStartEndTime, correctSessionID)).
					Return(map[string]string{
						"ended_at": "2.34",
					}, nil)

				return nil, mockRedisClient, nil, nil
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), constants.ErrInterviewSessionStartTimeNotFound.Error())
			},
		},
		{
			name: "Error - GetStartEndTime EndedAt Not Found Error",
			input: &entities.CreateUserSessionTurnBySessionIDReq{
				SessionID:  correctSessionID,
				TurnID:     correctTurnID,
				Transcript: "transcript",
			},
			setup: func() (*mockUtils.MockGenerator, *mockDatabase.MockRedisClient, *mockRepositories.MockInterviewSessionRepository, *mockRepositories.MockInterviewTurnsRepository) {
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewMaxTurnNo, correctSessionID)).
					Return("0", nil)

				mockRedisClient.EXPECT().
					HGetAll(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewStartEndTime, correctSessionID)).
					Return(map[string]string{
						"started_at": "0.01",
					}, nil)

				return nil, mockRedisClient, nil, nil
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), constants.ErrInterviewSessionEndTimeNotFound.Error())
			},
		},
		{
			name: "Error - CreateSessionTurnBySessionID Error",
			input: &entities.CreateUserSessionTurnBySessionIDReq{
				SessionID:    correctSessionID,
				TurnID:       correctTurnID,
				Transcript:   "transcript",
				CurrentState: currentState,
			},
			setup: func() (*mockUtils.MockGenerator, *mockDatabase.MockRedisClient, *mockRepositories.MockInterviewSessionRepository, *mockRepositories.MockInterviewTurnsRepository) {
				mockGenerator := new(mockUtils.MockGenerator)
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockInterviewTurnsRepo := new(mockRepositories.MockInterviewTurnsRepository)

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewMaxTurnNo, correctSessionID)).
					Return("0", nil)

				mockRedisClient.EXPECT().
					HGetAll(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewStartEndTime, correctSessionID)).
					Return(map[string]string{
						"started_at": "0.01",
						"ended_at":   "2.34",
					}, nil)

				mockInterviewTurnsRepo.EXPECT().
					CreateSessionTurnBySessionID(ctx, mock.MatchedBy(func(req *db.CreateInterviewTurnParams) bool {
						return req.SessionID.String() == correctSessionID &&
							req.ID.String() == correctTurnID &&
							req.TurnNo == 1 &&
							req.Actor == "user" &&
							req.TranscriptText == "transcript" &&
							req.StartAt == "0.01" &&
							req.EndAt == "2.34"
					})).
					Return(errors.New("database error"))

				return mockGenerator, mockRedisClient, mockInterviewSessionRepo, mockInterviewTurnsRepo
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			mockGenerator, mockRedisClient, mockInterviewSessionRepo, mockInterviewTurnsRepo := tC.setup()
			defer func() {
				if mockGenerator != nil {
					mockGenerator.AssertExpectations(t)
				}
				if mockRedisClient != nil {
					mockRedisClient.AssertExpectations(t)
				}
				if mockInterviewSessionRepo != nil {
					mockInterviewSessionRepo.AssertExpectations(t)
				}
				if mockInterviewTurnsRepo != nil {
					mockInterviewTurnsRepo.AssertExpectations(t)
				}
			}()

			svc := NewInterviewSessionService(
				lgr,
				nil,
				nil,
				mockGenerator,
				mockInterviewSessionRepo,
				nil,
				nil,
				nil,
				nil,
				mockRedisClient,
				nil,
				nil,
				mockInterviewTurnsRepo,
			)

			gotErr := svc.CreateUserSessionTurnBySessionID(ctx, tC.input)

			tC.verify(t, gotErr)
		})
	}
}

func TestInterviewSessionService_CreateInterviewerSessionTurnBySessionID(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()
	correctSessionID := uuid.New().String()
	correctTurnID := uuid.New().String()
	invalidID := "invalid-id"
	correctStartedAt := "0.01"
	correctEndedAt := "2.34"
	currentState := "Greeting"

	testCases := []struct {
		name   string
		input  *entities.CreateInterviewerSessionTurnBySessionIDReq
		setup  func() (*mockUtils.MockGenerator, *mockDatabase.MockRedisClient, *mockRepositories.MockInterviewSessionRepository, *mockRepositories.MockInterviewTurnsRepository)
		verify func(t *testing.T, gotErr error)
	}{
		{
			name: "Success - TurnRedisTriggered",
			input: &entities.CreateInterviewerSessionTurnBySessionIDReq{
				SessionID:    correctSessionID,
				TurnID:       correctTurnID,
				Transcript:   "transcript",
				StartedAt:    correctStartedAt,
				EndedAt:      correctEndedAt,
				CurrentState: currentState,
			},
			setup: func() (*mockUtils.MockGenerator, *mockDatabase.MockRedisClient, *mockRepositories.MockInterviewSessionRepository, *mockRepositories.MockInterviewTurnsRepository) {
				mockGenerator := new(mockUtils.MockGenerator)
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockInterviewTurnsRepo := new(mockRepositories.MockInterviewTurnsRepository)

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewMaxTurnNo, correctSessionID)).
					Return("1", nil)

				mockInterviewTurnsRepo.EXPECT().
					CreateSessionTurnBySessionID(context.Background(), mock.MatchedBy(func(req *db.CreateInterviewTurnParams) bool {
						return req.SessionID.String() == correctSessionID &&
							req.ID.String() == correctTurnID &&
							req.TurnNo == 2 &&
							req.Actor == "interviewer" &&
							req.TranscriptText == "transcript" &&
							req.StartAt == correctStartedAt &&
							req.EndAt == correctEndedAt
					})).
					Return(nil)

				mockRedisClient.EXPECT().
					Set(ctx, database.RedisPayload{
						Key:   fmt.Sprintf("%s%s", constants.RedisPrefixInterviewMaxTurnNo, correctSessionID),
						Value: int64(2),
						TTL:   constants.RedisTTLInterviewTurn,
					}).
					Return(nil)

				return mockGenerator, mockRedisClient, mockInterviewSessionRepo, mockInterviewTurnsRepo
			},
			verify: func(t *testing.T, gotErr error) {
				assert.NoError(t, gotErr)
			},
		},
		{
			name: "Error_Invalid session ID",
			input: &entities.CreateInterviewerSessionTurnBySessionIDReq{
				SessionID:    correctSessionID,
				TurnID:       invalidID,
				Transcript:   "transcript",
				StartedAt:    correctStartedAt,
				EndedAt:      correctEndedAt,
				CurrentState: currentState,
			},
			setup: func() (*mockUtils.MockGenerator, *mockDatabase.MockRedisClient, *mockRepositories.MockInterviewSessionRepository, *mockRepositories.MockInterviewTurnsRepository) {

				return nil, nil, nil, nil
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "The UUID is invalid. Please try again.")
				assert.Contains(t, gotErr.Error(), "[ONX0107]")
			},
		},

		{
			name: "Error_Invalid turn ID",
			input: &entities.CreateInterviewerSessionTurnBySessionIDReq{
				SessionID:    invalidID,
				TurnID:       correctTurnID,
				Transcript:   "transcript",
				StartedAt:    correctStartedAt,
				EndedAt:      correctEndedAt,
				CurrentState: currentState,
			},
			setup: func() (*mockUtils.MockGenerator, *mockDatabase.MockRedisClient, *mockRepositories.MockInterviewSessionRepository, *mockRepositories.MockInterviewTurnsRepository) {

				return nil, nil, nil, nil
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "The UUID is invalid. Please try again.")
				assert.Contains(t, gotErr.Error(), "[ONX0107]")
			},
		},
		{
			name: "Success - TurnRedisNotTriggered",
			input: &entities.CreateInterviewerSessionTurnBySessionIDReq{
				SessionID:    correctSessionID,
				TurnID:       correctTurnID,
				Transcript:   "transcript",
				StartedAt:    correctStartedAt,
				EndedAt:      correctEndedAt,
				CurrentState: currentState,
			},
			setup: func() (*mockUtils.MockGenerator, *mockDatabase.MockRedisClient, *mockRepositories.MockInterviewSessionRepository, *mockRepositories.MockInterviewTurnsRepository) {
				mockGenerator := new(mockUtils.MockGenerator)
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockInterviewTurnsRepo := new(mockRepositories.MockInterviewTurnsRepository)

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewMaxTurnNo, correctSessionID)).
					Return("", redis.Nil)

				mockInterviewTurnsRepo.EXPECT().
					GetMaxTurnNoBySessionID(ctx, uuid.MustParse(correctSessionID)).
					Return(int64(2), nil)

				mockInterviewTurnsRepo.EXPECT().
					CreateSessionTurnBySessionID(context.Background(), mock.MatchedBy(func(req *db.CreateInterviewTurnParams) bool {
						return req.SessionID.String() == correctSessionID &&
							req.ID.String() == correctTurnID &&
							req.TurnNo == 3 &&
							req.Actor == "interviewer" &&
							req.TranscriptText == "transcript" &&
							req.StartAt == correctStartedAt &&
							req.EndAt == correctEndedAt
					})).
					Return(nil)

				mockRedisClient.EXPECT().
					Set(ctx, database.RedisPayload{
						Key:   fmt.Sprintf("%s%s", constants.RedisPrefixInterviewMaxTurnNo, correctSessionID),
						Value: int64(3),
						TTL:   constants.RedisTTLInterviewTurn,
					}).
					Return(nil)

				return mockGenerator, mockRedisClient, mockInterviewSessionRepo, mockInterviewTurnsRepo
			},
			verify: func(t *testing.T, gotErr error) {
				assert.NoError(t, gotErr)
			},
		},
		{
			name: "Success - With Warning Set Redis Error",
			input: &entities.CreateInterviewerSessionTurnBySessionIDReq{
				SessionID:    correctSessionID,
				TurnID:       correctTurnID,
				Transcript:   "transcript",
				StartedAt:    correctStartedAt,
				EndedAt:      correctEndedAt,
				CurrentState: currentState,
			},
			setup: func() (*mockUtils.MockGenerator, *mockDatabase.MockRedisClient, *mockRepositories.MockInterviewSessionRepository, *mockRepositories.MockInterviewTurnsRepository) {
				mockGenerator := new(mockUtils.MockGenerator)
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockInterviewTurnsRepo := new(mockRepositories.MockInterviewTurnsRepository)

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewMaxTurnNo, correctSessionID)).
					Return("1", nil)

				mockInterviewTurnsRepo.EXPECT().
					CreateSessionTurnBySessionID(context.Background(), mock.MatchedBy(func(req *db.CreateInterviewTurnParams) bool {
						return req.SessionID.String() == correctSessionID &&
							req.ID.String() == correctTurnID &&
							req.TurnNo == 2 &&
							req.Actor == "interviewer" &&
							req.TranscriptText == "transcript" &&
							req.StartAt == correctStartedAt &&
							req.EndAt == correctEndedAt
					})).
					Return(nil)

				mockRedisClient.EXPECT().
					Set(ctx, database.RedisPayload{
						Key:   fmt.Sprintf("%s%s", constants.RedisPrefixInterviewMaxTurnNo, correctSessionID),
						Value: int64(2),
						TTL:   constants.RedisTTLInterviewTurn,
					}).
					Return(errors.New("redis error"))

				return mockGenerator, mockRedisClient, mockInterviewSessionRepo, mockInterviewTurnsRepo
			},
			verify: func(t *testing.T, gotErr error) {
				assert.NoError(t, gotErr)
			},
		},
		{
			name: "Error - Redis Get Turn error",
			input: &entities.CreateInterviewerSessionTurnBySessionIDReq{
				SessionID:    correctSessionID,
				TurnID:       correctTurnID,
				Transcript:   "transcript",
				StartedAt:    correctStartedAt,
				EndedAt:      correctEndedAt,
				CurrentState: currentState,
			},
			setup: func() (*mockUtils.MockGenerator, *mockDatabase.MockRedisClient, *mockRepositories.MockInterviewSessionRepository, *mockRepositories.MockInterviewTurnsRepository) {
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewMaxTurnNo, correctSessionID)).
					Return("", errors.New("redis error"))

				return nil, mockRedisClient, nil, nil
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
			},
		},
		{
			name: "Error - GetMaxTurnRepo Error",
			input: &entities.CreateInterviewerSessionTurnBySessionIDReq{
				SessionID:    correctSessionID,
				TurnID:       correctTurnID,
				Transcript:   "transcript",
				StartedAt:    correctStartedAt,
				EndedAt:      correctEndedAt,
				CurrentState: currentState,
			},
			setup: func() (*mockUtils.MockGenerator, *mockDatabase.MockRedisClient, *mockRepositories.MockInterviewSessionRepository, *mockRepositories.MockInterviewTurnsRepository) {
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockInterviewTurnsRepo := new(mockRepositories.MockInterviewTurnsRepository)

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewMaxTurnNo, correctSessionID)).
					Return("", redis.Nil)

				mockInterviewTurnsRepo.EXPECT().
					GetMaxTurnNoBySessionID(ctx, uuid.MustParse(correctSessionID)).
					Return(0, errors.New("database error"))

				return nil, mockRedisClient, mockInterviewSessionRepo, mockInterviewTurnsRepo
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
			},
		},
		{
			name: "Error - ParseIntTurnNo Error",
			input: &entities.CreateInterviewerSessionTurnBySessionIDReq{
				SessionID:    correctSessionID,
				TurnID:       correctTurnID,
				Transcript:   "transcript",
				StartedAt:    correctStartedAt,
				EndedAt:      correctEndedAt,
				CurrentState: currentState,
			},
			setup: func() (*mockUtils.MockGenerator, *mockDatabase.MockRedisClient, *mockRepositories.MockInterviewSessionRepository, *mockRepositories.MockInterviewTurnsRepository) {
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewMaxTurnNo, correctSessionID)).
					Return("invalid_turn_no", nil)

				return nil, mockRedisClient, nil, nil
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
			},
		},
		{
			name: "Error - CreateSessionTurnBySessionID Error",
			input: &entities.CreateInterviewerSessionTurnBySessionIDReq{
				SessionID:    correctSessionID,
				TurnID:       correctTurnID,
				Transcript:   "transcript",
				StartedAt:    correctStartedAt,
				EndedAt:      correctEndedAt,
				CurrentState: currentState,
			},
			setup: func() (*mockUtils.MockGenerator, *mockDatabase.MockRedisClient, *mockRepositories.MockInterviewSessionRepository, *mockRepositories.MockInterviewTurnsRepository) {
				mockGenerator := new(mockUtils.MockGenerator)
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockInterviewTurnsRepo := new(mockRepositories.MockInterviewTurnsRepository)

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewMaxTurnNo, correctSessionID)).
					Return("0", nil)

				mockInterviewTurnsRepo.EXPECT().
					CreateSessionTurnBySessionID(context.Background(), mock.MatchedBy(func(req *db.CreateInterviewTurnParams) bool {
						return req.SessionID.String() == correctSessionID &&
							req.ID.String() == correctTurnID &&
							req.TurnNo == 1 &&
							req.Actor == "interviewer" &&
							req.TranscriptText == "transcript" &&
							req.StartAt == correctStartedAt &&
							req.EndAt == correctEndedAt
					})).
					Return(errors.New("database error"))

				return mockGenerator, mockRedisClient, mockInterviewSessionRepo, mockInterviewTurnsRepo
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			mockGenerator, mockRedisClient, mockInterviewSessionRepo, mockInterviewTurnsRepo := tC.setup()
			defer func() {
				if mockGenerator != nil {
					mockGenerator.AssertExpectations(t)
				}
				if mockRedisClient != nil {
					mockRedisClient.AssertExpectations(t)
				}
				if mockInterviewSessionRepo != nil {
					mockInterviewSessionRepo.AssertExpectations(t)
				}
				if mockInterviewTurnsRepo != nil {
					mockInterviewTurnsRepo.AssertExpectations(t)
				}
			}()

			svc := NewInterviewSessionService(
				lgr,
				nil,
				nil,
				mockGenerator,
				mockInterviewSessionRepo,
				nil,
				nil,
				nil,
				nil,
				mockRedisClient,
				nil,
				nil,
				mockInterviewTurnsRepo,
			)

			gotErr := svc.CreateInterviewerSessionTurnBySessionID(ctx, tC.input)

			tC.verify(t, gotErr)
		})
	}
}

func TestInterviewSessionService_CalculateTurnScore(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()
	correctSessionID := "550e8400-e29b-41d4-a716-446655440000"
	correctTurnID := "550e8400-e29b-41d4-a716-446655440000"
	correctUserID := "550e8400-e29b-41d4-a716-446655440000"
	correctRubricID := "550e8400-e29b-41d4-a716-446655440001"
	correctCriterionID := "550e8400-e29b-41d4-a716-446655440002"
	currentState := "Greeting"

	validReq := &entities.CalculateTurnScoreReq{
		SessionID:          correctSessionID,
		UserTurnID:         correctTurnID,
		UserID:             correctUserID,
		UserMessage:        "user message",
		InterviewerMessage: "interviewer message",
		CurrentState:       currentState,
	}

	invalidSessionID := "invalid-session-id"
	invalidTurnID := "invalid-turn-id"
	invalidUserID := "invalid-user-id"
	invalidRubricID := "invalid-rubric-id"

	testCases := []struct {
		name   string
		input  *entities.CalculateTurnScoreReq
		setup  func() (*mockUtils.MockGenerator, *mockServices.MockEvaluationService, *mockRepositories.MockEvaluationScoresRepository, *mockRepositories.MockInterviewSessionRepository)
		verify func(t *testing.T, gotErr error)
	}{
		{
			name:  "Success",
			input: validReq,
			setup: func() (*mockUtils.MockGenerator, *mockServices.MockEvaluationService, *mockRepositories.MockEvaluationScoresRepository, *mockRepositories.MockInterviewSessionRepository) {
				mockGenerator := new(mockUtils.MockGenerator)
				mockEvaluationService := new(mockServices.MockEvaluationService)
				mockEvaluationScoresRepo := new(mockRepositories.MockEvaluationScoresRepository)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)

				evaluationID := uuid.New()
				mockGenerator.EXPECT().
					GenerateUUID(ctx).
					Return(evaluationID)

				mockEvaluationService.EXPECT().
					GetRubricWithCriteriaByName(ctx, validReq.CurrentState).
					Return(&entities.GetRubricWithCriteriaByNameResp{
						RubricID:            correctRubricID,
						RubricName:          "test",
						RubricDescriptionMd: "test",
						Criteria:            []entities.CritetiaRow{},
					}, nil)

				mockInterviewSessionRepo.EXPECT().
					InterviewFeedbackAndScore(ctx, mock.MatchedBy(func(req *repositories.InterviewFeedbackAndScoreReq) bool {
						return req.UserMessage == validReq.UserMessage &&
							req.InterviewerMessage == validReq.InterviewerMessage &&
							req.RubricName == "test" &&
							req.RubricDescriptionMd == "test" &&
							len(req.Criteria) == 0
					})).
					Return(&repositories.InterviewFeedbackAndScoreResp{
						OverallScore:    0,
						OverallFeedback: "test",
						CriteriaScores:  []repositories.CriteriaScore{},
					}, nil)

				mockEvaluationScoresRepo.EXPECT().
					CreateEvaluationWithCriteriaScoreAndImproveSentenceTx(ctx, mock.MatchedBy(func(req *repositories.CreateEvaluationAndScoreTxReq) bool {
						return req.EvaluationID == evaluationID &&
							req.SessionID.String() == correctSessionID &&
							req.TurnID.String() == correctTurnID &&
							req.UserID.String() == correctUserID &&
							req.RubricID.String() == correctRubricID &&
							req.CurrentState == validReq.CurrentState &&
							req.OverallScore == "0.000000" &&
							req.SummaryMd == "test" &&
							len(req.Criteria) == 0
					})).
					Return(nil)

				return mockGenerator, mockEvaluationService, mockEvaluationScoresRepo, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotErr error) {
				assert.NoError(t, gotErr)
			},
		},
		{
			name:  "Success - WithHavingCriteriaList",
			input: validReq,
			setup: func() (*mockUtils.MockGenerator, *mockServices.MockEvaluationService, *mockRepositories.MockEvaluationScoresRepository, *mockRepositories.MockInterviewSessionRepository) {
				mockGenerator := new(mockUtils.MockGenerator)
				mockEvaluationService := new(mockServices.MockEvaluationService)
				mockEvaluationScoresRepo := new(mockRepositories.MockEvaluationScoresRepository)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)

				evaluationID := uuid.New()
				mockGenerator.EXPECT().
					GenerateUUID(ctx).
					Return(evaluationID)

				mockEvaluationService.EXPECT().
					GetRubricWithCriteriaByName(ctx, validReq.CurrentState).
					Return(&entities.GetRubricWithCriteriaByNameResp{
						RubricID:            correctRubricID,
						RubricName:          "test",
						RubricDescriptionMd: "test",
						Criteria: []entities.CritetiaRow{
							{
								CriterionID:            correctCriterionID,
								CriterionCode:          "test",
								CriterionName:          "test",
								CriterionDescriptionMd: "test",
								CriterionWeight:        "test",
								CriterionMaxScore:      "test",
							},
						},
					}, nil)

				mockInterviewSessionRepo.EXPECT().
					InterviewFeedbackAndScore(ctx, mock.MatchedBy(func(req *repositories.InterviewFeedbackAndScoreReq) bool {
						return req.UserMessage == validReq.UserMessage &&
							req.InterviewerMessage == validReq.InterviewerMessage &&
							req.RubricName == "test" &&
							req.RubricDescriptionMd == "test" &&
							len(req.Criteria) == 1
					})).
					Return(&repositories.InterviewFeedbackAndScoreResp{
						OverallScore:    0,
						OverallFeedback: "test",
						CriteriaScores: []repositories.CriteriaScore{
							{
								CriterionID:       correctCriterionID,
								CriterionCode:     "test",
								CriterionName:     "test",
								CriterionScore:    0,
								CriterionFeedback: "test",
							},
						},
					}, nil)

				mockEvaluationScoresRepo.EXPECT().
					CreateEvaluationWithCriteriaScoreAndImproveSentenceTx(ctx, mock.MatchedBy(func(req *repositories.CreateEvaluationAndScoreTxReq) bool {
						return req.EvaluationID == evaluationID &&
							req.SessionID.String() == correctSessionID &&
							req.TurnID.String() == correctTurnID &&
							req.UserID.String() == correctUserID &&
							req.RubricID.String() == correctRubricID &&
							req.CurrentState == validReq.CurrentState &&
							req.OverallScore == "0.000000" &&
							req.SummaryMd == "test" &&
							len(req.Criteria) == 1
					})).
					Return(nil)

				return mockGenerator, mockEvaluationService, mockEvaluationScoresRepo, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotErr error) {
				assert.NoError(t, gotErr)
			},
		},
		{
			name:  "Error - WithGetRubricWithCriteriaByNameError",
			input: validReq,
			setup: func() (*mockUtils.MockGenerator, *mockServices.MockEvaluationService, *mockRepositories.MockEvaluationScoresRepository, *mockRepositories.MockInterviewSessionRepository) {
				mockEvaluationService := new(mockServices.MockEvaluationService)

				mockEvaluationService.EXPECT().
					GetRubricWithCriteriaByName(ctx, validReq.CurrentState).
					Return(nil, errors.New("database error"))

				return nil, mockEvaluationService, nil, nil
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
			},
		},
		{
			name:  "Error - WithInterviewFeedbackAndScoreError",
			input: validReq,
			setup: func() (*mockUtils.MockGenerator, *mockServices.MockEvaluationService, *mockRepositories.MockEvaluationScoresRepository, *mockRepositories.MockInterviewSessionRepository) {
				mockEvaluationService := new(mockServices.MockEvaluationService)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)

				mockEvaluationService.EXPECT().
					GetRubricWithCriteriaByName(ctx, validReq.CurrentState).
					Return(&entities.GetRubricWithCriteriaByNameResp{
						RubricID:            correctRubricID,
						RubricName:          "test",
						RubricDescriptionMd: "test",
						Criteria:            []entities.CritetiaRow{},
					}, nil)

				mockInterviewSessionRepo.EXPECT().
					InterviewFeedbackAndScore(ctx, mock.MatchedBy(func(req *repositories.InterviewFeedbackAndScoreReq) bool {
						return req.UserMessage == validReq.UserMessage &&
							req.InterviewerMessage == validReq.InterviewerMessage &&
							req.RubricName == "test" &&
							req.RubricDescriptionMd == "test" &&
							len(req.Criteria) == 0
					})).
					Return(nil, errors.New("database error"))

				return nil, mockEvaluationService, nil, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
			},
		},
		{
			name: "Error - WithParseSessionIDError",
			input: &entities.CalculateTurnScoreReq{
				SessionID:          invalidSessionID,
				UserTurnID:         validReq.UserTurnID,
				UserID:             validReq.UserID,
				UserMessage:        validReq.UserMessage,
				InterviewerMessage: validReq.InterviewerMessage,
				CurrentState:       validReq.CurrentState,
			},
			setup: func() (*mockUtils.MockGenerator, *mockServices.MockEvaluationService, *mockRepositories.MockEvaluationScoresRepository, *mockRepositories.MockInterviewSessionRepository) {
				mockGenerator := new(mockUtils.MockGenerator)
				mockEvaluationService := new(mockServices.MockEvaluationService)
				mockEvaluationScoresRepo := new(mockRepositories.MockEvaluationScoresRepository)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)

				mockEvaluationService.EXPECT().
					GetRubricWithCriteriaByName(ctx, validReq.CurrentState).
					Return(&entities.GetRubricWithCriteriaByNameResp{
						RubricID:            correctRubricID,
						RubricName:          "test",
						RubricDescriptionMd: "test",
						Criteria:            []entities.CritetiaRow{},
					}, nil)

				mockInterviewSessionRepo.EXPECT().
					InterviewFeedbackAndScore(ctx, mock.MatchedBy(func(req *repositories.InterviewFeedbackAndScoreReq) bool {
						return req.UserMessage == validReq.UserMessage &&
							req.InterviewerMessage == validReq.InterviewerMessage &&
							req.RubricName == "test" &&
							req.RubricDescriptionMd == "test" &&
							len(req.Criteria) == 0
					})).
					Return(&repositories.InterviewFeedbackAndScoreResp{
						OverallScore:    0,
						OverallFeedback: "test",
						CriteriaScores:  []repositories.CriteriaScore{},
					}, nil)

				evaluationID := uuid.New()
				mockGenerator.EXPECT().
					GenerateUUID(ctx).
					Return(evaluationID)

				return mockGenerator, mockEvaluationService, mockEvaluationScoresRepo, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "The UUID is invalid. Please try again.")
				assert.Contains(t, gotErr.Error(), "[ONX0107]")
			},
		},
		{
			name: "Error - WithParseUserTurnIDError",
			input: &entities.CalculateTurnScoreReq{
				SessionID:          validReq.SessionID,
				UserTurnID:         invalidTurnID,
				UserID:             validReq.UserID,
				UserMessage:        validReq.UserMessage,
				InterviewerMessage: validReq.InterviewerMessage,
				CurrentState:       validReq.CurrentState,
			},
			setup: func() (*mockUtils.MockGenerator, *mockServices.MockEvaluationService, *mockRepositories.MockEvaluationScoresRepository, *mockRepositories.MockInterviewSessionRepository) {
				mockGenerator := new(mockUtils.MockGenerator)
				mockEvaluationService := new(mockServices.MockEvaluationService)
				mockEvaluationScoresRepo := new(mockRepositories.MockEvaluationScoresRepository)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)

				mockEvaluationService.EXPECT().
					GetRubricWithCriteriaByName(ctx, validReq.CurrentState).
					Return(&entities.GetRubricWithCriteriaByNameResp{
						RubricID:            correctRubricID,
						RubricName:          "test",
						RubricDescriptionMd: "test",
						Criteria:            []entities.CritetiaRow{},
					}, nil)

				mockInterviewSessionRepo.EXPECT().
					InterviewFeedbackAndScore(ctx, mock.MatchedBy(func(req *repositories.InterviewFeedbackAndScoreReq) bool {
						return req.UserMessage == validReq.UserMessage &&
							req.InterviewerMessage == validReq.InterviewerMessage &&
							req.RubricName == "test" &&
							req.RubricDescriptionMd == "test" &&
							len(req.Criteria) == 0
					})).
					Return(&repositories.InterviewFeedbackAndScoreResp{
						OverallScore:    0,
						OverallFeedback: "test",
						CriteriaScores:  []repositories.CriteriaScore{},
					}, nil)

				evaluationID := uuid.New()
				mockGenerator.EXPECT().
					GenerateUUID(ctx).
					Return(evaluationID)

				return mockGenerator, mockEvaluationService, mockEvaluationScoresRepo, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "The UUID is invalid. Please try again.")
				assert.Contains(t, gotErr.Error(), "[ONX0107]")
			},
		},
		{
			name: "Error - WithParseUserIDError",
			input: &entities.CalculateTurnScoreReq{
				SessionID:          validReq.SessionID,
				UserTurnID:         validReq.UserTurnID,
				UserID:             invalidUserID,
				UserMessage:        validReq.UserMessage,
				InterviewerMessage: validReq.InterviewerMessage,
				CurrentState:       validReq.CurrentState,
			},
			setup: func() (*mockUtils.MockGenerator, *mockServices.MockEvaluationService, *mockRepositories.MockEvaluationScoresRepository, *mockRepositories.MockInterviewSessionRepository) {
				mockGenerator := new(mockUtils.MockGenerator)
				mockEvaluationService := new(mockServices.MockEvaluationService)
				mockEvaluationScoresRepo := new(mockRepositories.MockEvaluationScoresRepository)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)

				mockEvaluationService.EXPECT().
					GetRubricWithCriteriaByName(ctx, validReq.CurrentState).
					Return(&entities.GetRubricWithCriteriaByNameResp{
						RubricID:            correctRubricID,
						RubricName:          "test",
						RubricDescriptionMd: "test",
						Criteria:            []entities.CritetiaRow{},
					}, nil)

				mockInterviewSessionRepo.EXPECT().
					InterviewFeedbackAndScore(ctx, mock.MatchedBy(func(req *repositories.InterviewFeedbackAndScoreReq) bool {
						return req.UserMessage == validReq.UserMessage &&
							req.InterviewerMessage == validReq.InterviewerMessage &&
							req.RubricName == "test" &&
							req.RubricDescriptionMd == "test" &&
							len(req.Criteria) == 0
					})).
					Return(&repositories.InterviewFeedbackAndScoreResp{
						OverallScore:    0,
						OverallFeedback: "test",
						CriteriaScores:  []repositories.CriteriaScore{},
					}, nil)

				evaluationID := uuid.New()
				mockGenerator.EXPECT().
					GenerateUUID(ctx).
					Return(evaluationID)

				return mockGenerator, mockEvaluationService, mockEvaluationScoresRepo, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "The UUID is invalid. Please try again.")
				assert.Contains(t, gotErr.Error(), "[ONX0107]")
			},
		},
		{
			name: "Error - WithParseRubricIDError",
			input: &entities.CalculateTurnScoreReq{
				SessionID:          validReq.SessionID,
				UserTurnID:         validReq.UserTurnID,
				UserID:             validReq.UserID,
				UserMessage:        validReq.UserMessage,
				InterviewerMessage: validReq.InterviewerMessage,
				CurrentState:       validReq.CurrentState,
			},
			setup: func() (*mockUtils.MockGenerator, *mockServices.MockEvaluationService, *mockRepositories.MockEvaluationScoresRepository, *mockRepositories.MockInterviewSessionRepository) {
				mockGenerator := new(mockUtils.MockGenerator)
				mockEvaluationService := new(mockServices.MockEvaluationService)
				mockEvaluationScoresRepo := new(mockRepositories.MockEvaluationScoresRepository)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)

				mockEvaluationService.EXPECT().
					GetRubricWithCriteriaByName(ctx, validReq.CurrentState).
					Return(&entities.GetRubricWithCriteriaByNameResp{
						RubricID:            invalidRubricID,
						RubricName:          "test",
						RubricDescriptionMd: "test",
						Criteria:            []entities.CritetiaRow{},
					}, nil)

				mockInterviewSessionRepo.EXPECT().
					InterviewFeedbackAndScore(ctx, mock.MatchedBy(func(req *repositories.InterviewFeedbackAndScoreReq) bool {
						return req.UserMessage == validReq.UserMessage &&
							req.InterviewerMessage == validReq.InterviewerMessage &&
							req.RubricName == "test" &&
							req.RubricDescriptionMd == "test" &&
							len(req.Criteria) == 0
					})).
					Return(&repositories.InterviewFeedbackAndScoreResp{
						OverallScore:    0,
						OverallFeedback: "test",
						CriteriaScores:  []repositories.CriteriaScore{},
					}, nil)

				evaluationID := uuid.New()
				mockGenerator.EXPECT().
					GenerateUUID(ctx).
					Return(evaluationID)

				return mockGenerator, mockEvaluationService, mockEvaluationScoresRepo, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "The UUID is invalid. Please try again.")
				assert.Contains(t, gotErr.Error(), "[ONX0107]")
			},
		},
		{
			name:  "Error - WithParseCriterionIDError",
			input: validReq,
			setup: func() (*mockUtils.MockGenerator, *mockServices.MockEvaluationService, *mockRepositories.MockEvaluationScoresRepository, *mockRepositories.MockInterviewSessionRepository) {
				mockGenerator := new(mockUtils.MockGenerator)
				mockEvaluationService := new(mockServices.MockEvaluationService)
				mockEvaluationScoresRepo := new(mockRepositories.MockEvaluationScoresRepository)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)

				evaluationID := uuid.New()
				mockGenerator.EXPECT().
					GenerateUUID(ctx).
					Return(evaluationID)

				mockEvaluationService.EXPECT().
					GetRubricWithCriteriaByName(ctx, validReq.CurrentState).
					Return(&entities.GetRubricWithCriteriaByNameResp{
						RubricID:            correctRubricID,
						RubricName:          "test",
						RubricDescriptionMd: "test",
						Criteria: []entities.CritetiaRow{
							{
								CriterionID:            "invalid-criterion-id",
								CriterionCode:          "test",
								CriterionName:          "test",
								CriterionDescriptionMd: "test",
								CriterionWeight:        "test",
								CriterionMaxScore:      "test",
							},
						},
					}, nil)

				mockInterviewSessionRepo.EXPECT().
					InterviewFeedbackAndScore(ctx, mock.MatchedBy(func(req *repositories.InterviewFeedbackAndScoreReq) bool {
						return req.UserMessage == validReq.UserMessage &&
							req.InterviewerMessage == validReq.InterviewerMessage &&
							req.RubricName == "test" &&
							req.RubricDescriptionMd == "test" &&
							len(req.Criteria) == 1
					})).
					Return(&repositories.InterviewFeedbackAndScoreResp{
						OverallScore:    0,
						OverallFeedback: "test",
						CriteriaScores: []repositories.CriteriaScore{
							{
								CriterionID:       "invalid-criterion-id",
								CriterionCode:     "test",
								CriterionName:     "test",
								CriterionScore:    0,
								CriterionFeedback: "test",
							},
						},
					}, nil)

				return mockGenerator, mockEvaluationService, mockEvaluationScoresRepo, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "The UUID is invalid. Please try again.")
				assert.Contains(t, gotErr.Error(), "[ONX0107]")
			},
		},
		{
			name:  "Error - WithCreateEvaluationWithCriteriaScoreTxError",
			input: validReq,
			setup: func() (*mockUtils.MockGenerator, *mockServices.MockEvaluationService, *mockRepositories.MockEvaluationScoresRepository, *mockRepositories.MockInterviewSessionRepository) {
				mockGenerator := new(mockUtils.MockGenerator)
				mockEvaluationService := new(mockServices.MockEvaluationService)
				mockEvaluationScoresRepo := new(mockRepositories.MockEvaluationScoresRepository)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)

				evaluationID := uuid.New()
				mockGenerator.EXPECT().
					GenerateUUID(ctx).
					Return(evaluationID)

				mockEvaluationService.EXPECT().
					GetRubricWithCriteriaByName(ctx, validReq.CurrentState).
					Return(&entities.GetRubricWithCriteriaByNameResp{
						RubricID:            correctRubricID,
						RubricName:          "test",
						RubricDescriptionMd: "test",
						Criteria:            []entities.CritetiaRow{},
					}, nil)

				mockInterviewSessionRepo.EXPECT().
					InterviewFeedbackAndScore(ctx, mock.MatchedBy(func(req *repositories.InterviewFeedbackAndScoreReq) bool {
						return req.UserMessage == validReq.UserMessage &&
							req.InterviewerMessage == validReq.InterviewerMessage &&
							req.RubricName == "test" &&
							req.RubricDescriptionMd == "test" &&
							len(req.Criteria) == 0
					})).
					Return(&repositories.InterviewFeedbackAndScoreResp{
						OverallScore:    0,
						OverallFeedback: "test",
						CriteriaScores:  []repositories.CriteriaScore{},
					}, nil)

				mockEvaluationScoresRepo.EXPECT().
					CreateEvaluationWithCriteriaScoreAndImproveSentenceTx(ctx, mock.MatchedBy(func(req *repositories.CreateEvaluationAndScoreTxReq) bool {
						return req.EvaluationID == evaluationID &&
							req.SessionID.String() == correctSessionID &&
							req.TurnID.String() == correctTurnID &&
							req.UserID.String() == correctUserID &&
							req.RubricID.String() == correctRubricID &&
							req.CurrentState == validReq.CurrentState &&
							req.OverallScore == "0.000000" &&
							req.SummaryMd == "test" &&
							len(req.Criteria) == 0
					})).
					Return(errors.New("database error")).Times(3)

				return mockGenerator, mockEvaluationService, mockEvaluationScoresRepo, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "database error")
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			mockGenerator, mockEvaluationService, mockEvaluationScoresRepo, mockInterviewSessionRepo := tC.setup()
			defer func() {
				if mockGenerator != nil {
					mockGenerator.AssertExpectations(t)
				}
				if mockEvaluationService != nil {
					mockEvaluationService.AssertExpectations(t)
				}
				if mockEvaluationScoresRepo != nil {
					mockEvaluationScoresRepo.AssertExpectations(t)
				}
				if mockInterviewSessionRepo != nil {
					mockInterviewSessionRepo.AssertExpectations(t)
				}
			}()

			svc := NewInterviewSessionService(
				lgr,
				nil,
				nil,
				mockGenerator,
				mockInterviewSessionRepo,
				nil,
				nil,
				nil,
				nil,
				nil,
				mockEvaluationService,
				mockEvaluationScoresRepo,
				nil,
			)

			gotErr := svc.CalculateTurnScore(ctx, tC.input)

			tC.verify(t, gotErr)
		})
	}
}

func TestInterviewSessionService_GetInterviewerLastMessage(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()
	currentState := "Greeting"
	message := "message"
	correctSessionID := "550e8400-e29b-41d4-a716-446655440000"

	validResp := &entities.GetInterviewerLastMessageResp{
		Message:      message,
		CurrentState: currentState,
	}

	validRespBytes, err := json.Marshal(validResp)
	if err != nil {
		t.Fatalf("Error marshaling validResp: %v", err)
	}

	testCases := []struct {
		name   string
		input  *entities.GetInterviewerLastMessageReq
		setup  func() (*mockDatabase.MockRedisClient, *mockRepositories.MockInterviewTurnsRepository)
		verify func(t *testing.T, gotErr error, gotResp *entities.GetInterviewerLastMessageResp)
	}{
		{
			name: "Success - WithRedisHit",
			input: &entities.GetInterviewerLastMessageReq{
				SessionID: correctSessionID,
			},
			setup: func() (*mockDatabase.MockRedisClient, *mockRepositories.MockInterviewTurnsRepository) {
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewLastMessage, correctSessionID)).
					Return(string(validRespBytes), nil)

				return mockRedisClient, nil
			},
			verify: func(t *testing.T, gotErr error, gotResp *entities.GetInterviewerLastMessageResp) {
				assert.NoError(t, gotErr)
				assert.Equal(t, validResp, gotResp)
			},
		},
		{
			name: "Success - WithRedisMiss",
			input: &entities.GetInterviewerLastMessageReq{
				SessionID: correctSessionID,
			},
			setup: func() (*mockDatabase.MockRedisClient, *mockRepositories.MockInterviewTurnsRepository) {
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockInterviewTurnsRepo := new(mockRepositories.MockInterviewTurnsRepository)

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewLastMessage, correctSessionID)).
					Return("", errors.New("redis error"))

				mockInterviewTurnsRepo.EXPECT().
					GetInterviewerLastMessage(ctx, uuid.MustParse(correctSessionID)).
					Return(&db.GetInterviewerLastMessageRow{
						TranscriptText: message,
						CurrentState:   currentState,
					}, nil)

				return mockRedisClient, mockInterviewTurnsRepo
			},
			verify: func(t *testing.T, gotErr error, gotResp *entities.GetInterviewerLastMessageResp) {
				assert.NoError(t, gotErr)
				assert.Equal(t, validResp, gotResp)
			},
		},
		{
			name: "Error - WithInvalidSessionID",
			input: &entities.GetInterviewerLastMessageReq{
				SessionID: "invalid-session-id",
			},
			setup: func() (*mockDatabase.MockRedisClient, *mockRepositories.MockInterviewTurnsRepository) {
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockInterviewTurnsRepo := new(mockRepositories.MockInterviewTurnsRepository)

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewLastMessage, "invalid-session-id")).
					Return("", errors.New("redis error"))

				return mockRedisClient, mockInterviewTurnsRepo
			},
			verify: func(t *testing.T, gotErr error, gotResp *entities.GetInterviewerLastMessageResp) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "The UUID is invalid. Please try again.")
				assert.Contains(t, gotErr.Error(), "[ONX0107]")
				assert.Nil(t, gotResp)
			},
		},
		{
			name: "Success - WithInterviewTurnsRepoError",
			input: &entities.GetInterviewerLastMessageReq{
				SessionID: correctSessionID,
			},
			setup: func() (*mockDatabase.MockRedisClient, *mockRepositories.MockInterviewTurnsRepository) {
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockInterviewTurnsRepo := new(mockRepositories.MockInterviewTurnsRepository)

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewLastMessage, correctSessionID)).
					Return("", errors.New("redis error"))

				mockInterviewTurnsRepo.EXPECT().
					GetInterviewerLastMessage(context.Background(), uuid.MustParse(correctSessionID)).
					Return(nil, errors.New("interview turns repo error"))

				return mockRedisClient, mockInterviewTurnsRepo
			},
			verify: func(t *testing.T, gotErr error, gotResp *entities.GetInterviewerLastMessageResp) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
			},
		},
		{
			name: "Error - WithRedisHitButUnmarshalError",
			input: &entities.GetInterviewerLastMessageReq{
				SessionID: correctSessionID,
			},
			setup: func() (*mockDatabase.MockRedisClient, *mockRepositories.MockInterviewTurnsRepository) {
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewLastMessage, correctSessionID)).
					Return("invalid json", nil)

				return mockRedisClient, nil
			},
			verify: func(t *testing.T, gotErr error, gotResp *entities.GetInterviewerLastMessageResp) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			mockRedisClient, mockInterviewTurnsRepo := tC.setup()
			defer func() {
				if mockRedisClient != nil {
					mockRedisClient.AssertExpectations(t)
				}
				if mockInterviewTurnsRepo != nil {
					mockInterviewTurnsRepo.AssertExpectations(t)
				}
			}()

			svc := NewInterviewSessionService(
				lgr,
				nil,
				nil,
				nil,
				nil,
				nil,
				nil,
				nil,
				nil,
				mockRedisClient,
				nil,
				nil,
				mockInterviewTurnsRepo,
			)

			gotResp, gotErr := svc.GetInterviewerLastMessage(ctx, tC.input)

			tC.verify(t, gotErr, gotResp)
		})
	}
}

func TestInterviewSessionService_GetChatHistoryBySessionToken(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()
	message := "message"
	globalID := "550e8400-e29b-41d4-a716-446655440000"

	validResp := &entities.GetChatHistoryBySessionTokenResp{
		ChatHistory: []entities.ChatHistory{
			{
				ID:             uuid.MustParse(globalID),
				TurnNo:         1,
				Actor:          "interviewer",
				TranscriptText: message,
				StartAt:        "00.12",
				EndAt:          "00.14",
				CreatedAt:      "1 Jan 2021",
			},
		},
	}

	validNonResp := &entities.GetChatHistoryBySessionTokenResp{
		ChatHistory: []entities.ChatHistory{},
	}

	testCases := []struct {
		name   string
		input  *entities.GetChatHistoryBySessionTokenReq
		setup  func() (*mockMiddleware.MockAuthContext, *mockRepositories.MockInterviewTurnsRepository, *mockDatabase.MockRedisClient, *mockRepositories.MockInterviewSessionRepository)
		verify func(t *testing.T, gotErr error, gotResp *entities.GetChatHistoryBySessionTokenResp)
	}{
		{
			name: "Success - WithNonChatHistory",
			input: &entities.GetChatHistoryBySessionTokenReq{
				SessionToken: globalID,
			},
			setup: func() (*mockMiddleware.MockAuthContext, *mockRepositories.MockInterviewTurnsRepository, *mockDatabase.MockRedisClient, *mockRepositories.MockInterviewSessionRepository) {
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockInterviewTurnsRepo := new(mockRepositories.MockInterviewTurnsRepository)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							UserID: globalID,
						},
					}, nil)

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewSessionToken, globalID)).
					Return("{\"user_id\":\""+globalID+"\",\"session_id\":\""+globalID+"\"}", nil)

				mockInterviewSessionRepo.EXPECT().
					CheckInterviewSessionExists(ctx, uuid.MustParse(globalID)).
					Return(true, nil)

				mockInterviewTurnsRepo.EXPECT().
					GetChatHistoryBySessionID(ctx, uuid.MustParse(globalID)).
					Return([]db.GetChatHistoryBySessionIDRow{}, nil)

				return mockAuthContext, mockInterviewTurnsRepo, mockRedisClient, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotErr error, gotResp *entities.GetChatHistoryBySessionTokenResp) {
				assert.NoError(t, gotErr)
				assert.Equal(t, validNonResp, gotResp)
			},
		},
		{
			name: "Success - WithChatHistory",
			input: &entities.GetChatHistoryBySessionTokenReq{
				SessionToken: globalID,
			},
			setup: func() (*mockMiddleware.MockAuthContext, *mockRepositories.MockInterviewTurnsRepository, *mockDatabase.MockRedisClient, *mockRepositories.MockInterviewSessionRepository) {
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockInterviewTurnsRepo := new(mockRepositories.MockInterviewTurnsRepository)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							UserID: globalID,
						},
					}, nil)

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewSessionToken, globalID)).
					Return("{\"user_id\":\""+globalID+"\",\"session_id\":\""+globalID+"\"}", nil)

				mockInterviewSessionRepo.EXPECT().
					CheckInterviewSessionExists(ctx, uuid.MustParse(globalID)).
					Return(true, nil)

				mockInterviewTurnsRepo.EXPECT().
					GetChatHistoryBySessionID(ctx, uuid.MustParse(globalID)).
					Return([]db.GetChatHistoryBySessionIDRow{
						{
							ID:             uuid.MustParse(globalID),
							TurnNo:         1,
							Actor:          "interviewer",
							TranscriptText: message,
							StartAt:        "00.12",
							EndAt:          "00.14",
							CreatedAt:      time.Now(),
						},
					}, nil)

				return mockAuthContext, mockInterviewTurnsRepo, mockRedisClient, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotErr error, gotResp *entities.GetChatHistoryBySessionTokenResp) {
				assert.NoError(t, gotErr)
				assert.Equal(t, gotResp.ChatHistory[0].ID, validResp.ChatHistory[0].ID)
				assert.Equal(t, gotResp.ChatHistory[0].TurnNo, validResp.ChatHistory[0].TurnNo)
				assert.Equal(t, gotResp.ChatHistory[0].Actor, validResp.ChatHistory[0].Actor)
				assert.Equal(t, gotResp.ChatHistory[0].TranscriptText, validResp.ChatHistory[0].TranscriptText)
				assert.Equal(t, gotResp.ChatHistory[0].StartAt, validResp.ChatHistory[0].StartAt)
				assert.Equal(t, gotResp.ChatHistory[0].EndAt, validResp.ChatHistory[0].EndAt)
			},
		},
		{
			name: "Error - WithGetAuthContextError",
			input: &entities.GetChatHistoryBySessionTokenReq{
				SessionToken: globalID,
			},
			setup: func() (*mockMiddleware.MockAuthContext, *mockRepositories.MockInterviewTurnsRepository, *mockDatabase.MockRedisClient, *mockRepositories.MockInterviewSessionRepository) {
				mockAuthContext := new(mockMiddleware.MockAuthContext)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(nil, errors.New("get auth context error"))

				return mockAuthContext, nil, nil, nil
			},
			verify: func(t *testing.T, gotErr error, gotResp *entities.GetChatHistoryBySessionTokenResp) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
			},
		},
		{
			name: "Error - WithCheckIsSessionValidError",
			input: &entities.GetChatHistoryBySessionTokenReq{
				SessionToken: globalID,
			},
			setup: func() (*mockMiddleware.MockAuthContext, *mockRepositories.MockInterviewTurnsRepository, *mockDatabase.MockRedisClient, *mockRepositories.MockInterviewSessionRepository) {
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockInterviewTurnsRepo := new(mockRepositories.MockInterviewTurnsRepository)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							UserID: globalID,
						},
					}, nil)

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewSessionToken, globalID)).
					Return("{\"user_id\":\""+globalID+"\",\"session_id\":\""+globalID+"\"}", nil)

				mockInterviewSessionRepo.EXPECT().
					CheckInterviewSessionExists(ctx, uuid.MustParse(globalID)).
					Return(false, errors.New("check is session valid error"))

				return mockAuthContext, mockInterviewTurnsRepo, mockRedisClient, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotErr error, gotResp *entities.GetChatHistoryBySessionTokenResp) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
			},
		},
		{
			name: "Error - WithInvalidGetChatHistoryBySessionID",
			input: &entities.GetChatHistoryBySessionTokenReq{
				SessionToken: globalID,
			},
			setup: func() (*mockMiddleware.MockAuthContext, *mockRepositories.MockInterviewTurnsRepository, *mockDatabase.MockRedisClient, *mockRepositories.MockInterviewSessionRepository) {
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockInterviewTurnsRepo := new(mockRepositories.MockInterviewTurnsRepository)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							UserID: globalID,
						},
					}, nil)

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewSessionToken, globalID)).
					Return("{\"user_id\":\""+globalID+"\",\"session_id\":\""+globalID+"\"}", nil)

				mockInterviewSessionRepo.EXPECT().
					CheckInterviewSessionExists(ctx, uuid.MustParse(globalID)).
					Return(true, nil)

				mockInterviewTurnsRepo.EXPECT().
					GetChatHistoryBySessionID(ctx, uuid.MustParse(globalID)).
					Return(nil, errors.New("get chat history by session id error"))

				return mockAuthContext, mockInterviewTurnsRepo, mockRedisClient, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotErr error, gotResp *entities.GetChatHistoryBySessionTokenResp) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			mockAuthContext, mockInterviewTurnsRepo, mockRedisClient, mockInterviewSessionRepo := tC.setup()
			defer func() {
				if mockAuthContext != nil {
					mockAuthContext.AssertExpectations(t)
				}
				if mockInterviewTurnsRepo != nil {
					mockInterviewTurnsRepo.AssertExpectations(t)
				}
				if mockRedisClient != nil {
					mockRedisClient.AssertExpectations(t)
				}
				if mockInterviewSessionRepo != nil {
					mockInterviewSessionRepo.AssertExpectations(t)
				}
			}()

			svc := NewInterviewSessionService(
				lgr,
				mockAuthContext,
				nil,
				nil,
				mockInterviewSessionRepo,
				nil,
				nil,
				nil,
				nil,
				mockRedisClient,
				nil,
				nil,
				mockInterviewTurnsRepo,
			)

			gotResp, gotErr := svc.GetChatHistoryBySessionToken(ctx, tC.input)

			tC.verify(t, gotErr, gotResp)
		})
	}
}
