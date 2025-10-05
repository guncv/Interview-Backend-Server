package services

import (
	"context"
	"database/sql"
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
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
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
	"gitlab.com/interview-simulation/interview-backend-server/internal/utils"
)

func TestInterviewSessionService_CreateInterviewSessionWithNewResume(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()
	userID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	invalidUserID := "invalid-user-id"
	validExtractResp := &repositories.ExtractResumeJsonForRAGResp{
		BiasPrompt: "bias_prompt",
	}

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
						Payload: &utils.SignInTokenPayload{
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
					Return(validExtractResp, nil)

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

				// Mock Redis Set call
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
						Payload: &utils.SignInTokenPayload{
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
					Return(validExtractResp, nil)

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
						Payload: &utils.SignInTokenPayload{
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
					Return(nil, errors.New("JSON generation failed"))

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
						Payload: &utils.SignInTokenPayload{
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
					Return(validExtractResp, nil)

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
				assert.Contains(t, gotErr.Error(), "[INS0107]")
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
						Payload: &utils.SignInTokenPayload{
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
					Return(validExtractResp, nil)

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
						Payload: &utils.SignInTokenPayload{
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
					Return(validExtractResp, nil)

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
						Payload: &utils.SignInTokenPayload{
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
					Return(validExtractResp, nil)

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
						Payload: &utils.SignInTokenPayload{
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
					Return(validExtractResp, nil)

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
						Payload: &utils.SignInTokenPayload{
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
					Return(validExtractResp, nil)

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
						Payload: &utils.SignInTokenPayload{
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
					Return(validExtractResp, nil)

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
						Payload: &utils.SignInTokenPayload{
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
					Return(nil, errors.New("resume processing failed"))

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
						Payload: &utils.SignInTokenPayload{
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
					Return(validExtractResp, nil)

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
						Payload: &utils.SignInTokenPayload{
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
					Return(nil, errors.New("invalid file: file is nil or corrupted"))

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
						Payload: &utils.SignInTokenPayload{
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
					Return(validExtractResp, nil)

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
	validExtractResp := &repositories.ExtractResumeJsonForRAGResp{
		BiasPrompt: "bias_prompt",
	}

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
						Payload: &utils.SignInTokenPayload{
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
					Return(validExtractResp, nil)

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
						Payload: &utils.SignInTokenPayload{
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
				assert.Contains(t, gotErr.Error(), "[INS0107]")
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
						Payload: &utils.SignInTokenPayload{
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
						Payload: &utils.SignInTokenPayload{
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
						Payload: &utils.SignInTokenPayload{
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
					Return(nil, errors.New("JSON generation failed"))

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
						Payload: &utils.SignInTokenPayload{
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
					Return(validExtractResp, nil)

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
						Payload: &utils.SignInTokenPayload{
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
					Return(validExtractResp, nil)

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
				assert.Contains(t, gotErr.Error(), "[INS0107]")
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
						Payload: &utils.SignInTokenPayload{
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
					Return(validExtractResp, nil)

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
						Payload: &utils.SignInTokenPayload{
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
					Return(validExtractResp, nil)

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
						Payload: &utils.SignInTokenPayload{
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
					Return(validExtractResp, nil)

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
				Status:    constants.StatusPending,
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
			name: "Success WithSetRedisError",
			input: &entities.UpdateInterviewSessionStatusReq{
				SessionID: sessionID.String(),
				Status:    constants.StatusPending,
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
			name: "Success WithSetAndDeleteRedisError",
			input: &entities.UpdateInterviewSessionStatusReq{
				SessionID: sessionID.String(),
				Status:    constants.StatusPending,
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
				Status:    constants.StatusPending,
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
				assert.Contains(t, gotErr.Error(), "[INS0107]")
			},
		},
		{
			name: "Error WithValidStatus",
			input: &entities.UpdateInterviewSessionStatusReq{
				SessionID: sessionID.String(),
				Status:    "Status",
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
				assert.Contains(t, gotErr.Error(), "The status is invalid. Please try again.")
				assert.Contains(t, gotErr.Error(), "[INS0421]")
			},
		},
		{
			name: "Error - Update interview session status failure",
			input: &entities.UpdateInterviewSessionStatusReq{
				SessionID: sessionID.String(),
				Status:    constants.StatusPending,
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
				Status:    constants.StatusPending,
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
				assert.Contains(t, gotErr.Error(), "[INS0107]")
			},
		},
		{
			name: "Error - Empty session ID",
			input: &entities.UpdateInterviewSessionStatusReq{
				SessionID: "",
				Status:    constants.StatusPending,
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
				assert.Contains(t, gotErr.Error(), "[INS0107]")
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
				assert.Contains(t, gotErr.Error(), "[INS0107]")
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
					Increment(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewMaxTurnNo, correctSessionID)).
					Return(2, nil)

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
				assert.Contains(t, gotErr.Error(), "[INS0107]")
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
				assert.Contains(t, gotErr.Error(), "[INS0107]")
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
					Increment(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewMaxTurnNo, correctSessionID)).
					Return(0, redis.Nil)

				mockInterviewTurnsRepo.EXPECT().
					GetMaxTurnNoBySessionID(ctx, uuid.MustParse(correctSessionID)).
					Return(int64(2), nil)

				mockRedisClient.EXPECT().
					Set(ctx, database.RedisPayload{
						Key:   fmt.Sprintf("%s%s", constants.RedisPrefixInterviewMaxTurnNo, correctSessionID),
						Value: int64(3),
						TTL:   constants.RedisTTLInterviewTurn,
					}).
					Return(nil)

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
					Increment(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewMaxTurnNo, correctSessionID)).
					Return(2, nil)

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

				return mockGenerator, mockRedisClient, mockInterviewSessionRepo, mockInterviewTurnsRepo
			},
			verify: func(t *testing.T, gotErr error) {
				assert.NoError(t, gotErr)
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
					Increment(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewMaxTurnNo, correctSessionID)).
					Return(0, redis.Nil)

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
					Increment(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewMaxTurnNo, correctSessionID)).
					Return(1, nil)

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
					Increment(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewMaxTurnNo, correctSessionID)).
					Return(1, nil)

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
					Increment(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewMaxTurnNo, correctSessionID)).
					Return(1, nil)

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
					Increment(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewMaxTurnNo, correctSessionID)).
					Return(1, nil)

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
					Increment(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewMaxTurnNo, correctSessionID)).
					Return(1, nil)

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
					Increment(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewMaxTurnNo, correctSessionID)).
					Return(1, nil)

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
				nil,
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
					Increment(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewMaxTurnNo, correctSessionID)).
					Return(2, nil)

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

				redisKey := fmt.Sprintf("%s%s:%s", constants.RedisPrefixInterviewIsScoreSessionState, correctSessionID, currentState)
				mockRedisClient.EXPECT().
					Set(ctx, database.RedisPayload{
						Key:   redisKey,
						Value: false,
						TTL:   constants.RedisTTLInterviewIsScoreSessionState,
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
				assert.Contains(t, gotErr.Error(), "[INS0107]")
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
				assert.Contains(t, gotErr.Error(), "[INS0107]")
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
					Increment(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewMaxTurnNo, correctSessionID)).
					Return(1, redis.Nil)

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

				redisKey := fmt.Sprintf("%s%s:%s", constants.RedisPrefixInterviewIsScoreSessionState, correctSessionID, currentState)
				mockRedisClient.EXPECT().
					Set(ctx, database.RedisPayload{
						Key:   redisKey,
						Value: false,
						TTL:   constants.RedisTTLInterviewIsScoreSessionState,
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
					Increment(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewMaxTurnNo, correctSessionID)).
					Return(2, nil)

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

				redisKey := fmt.Sprintf("%s%s:%s", constants.RedisPrefixInterviewIsScoreSessionState, correctSessionID, currentState)
				mockRedisClient.EXPECT().
					Set(ctx, database.RedisPayload{
						Key:   redisKey,
						Value: false,
						TTL:   constants.RedisTTLInterviewIsScoreSessionState,
					}).
					Return(nil)

				return mockGenerator, mockRedisClient, mockInterviewSessionRepo, mockInterviewTurnsRepo
			},
			verify: func(t *testing.T, gotErr error) {
				assert.NoError(t, gotErr)
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
					Increment(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewMaxTurnNo, correctSessionID)).
					Return(0, redis.Nil)

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
					Increment(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewMaxTurnNo, correctSessionID)).
					Return(1, nil)

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
				nil,
			)

			gotErr := svc.CreateInterviewerSessionTurnBySessionID(ctx, tC.input)

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
				assert.Contains(t, gotErr.Error(), "[INS0107]")
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
				nil,
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
		CursorTurnNext: 1,
	}

	validNonResp := &entities.GetChatHistoryBySessionTokenResp{
		ChatHistory:    []entities.ChatHistory{},
		CursorTurnNext: 0,
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
						Payload: &utils.SignInTokenPayload{
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
					GetChatHistoryBySessionID(ctx, mock.MatchedBy(func(req *db.GetChatHistoryBySessionIDParams) bool {
						return req.SessionID == uuid.MustParse(globalID) && req.Limit == constants.DefaultPageSize
					})).
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
						Payload: &utils.SignInTokenPayload{
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
					GetChatHistoryBySessionID(ctx, mock.MatchedBy(func(req *db.GetChatHistoryBySessionIDParams) bool {
						return req.SessionID == uuid.MustParse(globalID) && req.Limit == constants.DefaultPageSize
					})).
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
				assert.Equal(t, validResp.CursorTurnNext, gotResp.CursorTurnNext)
			},
		},
		{
			name: "Success - WithTurnNo",
			input: &entities.GetChatHistoryBySessionTokenReq{
				SessionToken: globalID,
				TurnNo:       func() *int32 { v := int32(5); return &v }(),
			},
			setup: func() (*mockMiddleware.MockAuthContext, *mockRepositories.MockInterviewTurnsRepository, *mockDatabase.MockRedisClient, *mockRepositories.MockInterviewSessionRepository) {
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockInterviewTurnsRepo := new(mockRepositories.MockInterviewTurnsRepository)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utils.SignInTokenPayload{
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
					GetChatHistoryBySessionIDWithCursor(ctx, mock.MatchedBy(func(req *db.GetChatHistoryBySessionIDWithCursorParams) bool {
						return req.SessionID == uuid.MustParse(globalID) && req.TurnNo == 5 && req.Limit == constants.DefaultPageSize
					})).
					Return([]db.GetChatHistoryBySessionIDWithCursorRow{
						{
							ID:             uuid.MustParse(globalID),
							TurnNo:         4,
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
				assert.Len(t, gotResp.ChatHistory, 1)
				assert.Equal(t, int64(4), gotResp.ChatHistory[0].TurnNo)
				assert.Equal(t, int32(4), gotResp.CursorTurnNext)
			},
		},
		{
			name: "Error - InvalidSessionIDParsing",
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
						Payload: &utils.SignInTokenPayload{
							UserID: globalID,
						},
					}, nil)

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewSessionToken, globalID)).
					Return("{\"user_id\":\""+globalID+"\",\"session_id\":\"invalid-uuid\"}", nil)

				return mockAuthContext, mockInterviewTurnsRepo, mockRedisClient, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotErr error, gotResp *entities.GetChatHistoryBySessionTokenResp) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
				assert.Contains(t, gotErr.Error(), "invalid UUID")
			},
		},
		{
			name: "Error - GetChatHistoryWithCursorError",
			input: &entities.GetChatHistoryBySessionTokenReq{
				SessionToken: globalID,
				TurnNo:       func() *int32 { v := int32(5); return &v }(),
			},
			setup: func() (*mockMiddleware.MockAuthContext, *mockRepositories.MockInterviewTurnsRepository, *mockDatabase.MockRedisClient, *mockRepositories.MockInterviewSessionRepository) {
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockInterviewTurnsRepo := new(mockRepositories.MockInterviewTurnsRepository)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utils.SignInTokenPayload{
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
					GetChatHistoryBySessionIDWithCursor(ctx, mock.Anything).
					Return(nil, errors.New("database cursor error"))

				return mockAuthContext, mockInterviewTurnsRepo, mockRedisClient, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotErr error, gotResp *entities.GetChatHistoryBySessionTokenResp) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
			},
		},
		{
			name: "Success - WithTurnNoEmptyResults",
			input: &entities.GetChatHistoryBySessionTokenReq{
				SessionToken: globalID,
				TurnNo:       func() *int32 { v := int32(5); return &v }(),
			},
			setup: func() (*mockMiddleware.MockAuthContext, *mockRepositories.MockInterviewTurnsRepository, *mockDatabase.MockRedisClient, *mockRepositories.MockInterviewSessionRepository) {
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockInterviewTurnsRepo := new(mockRepositories.MockInterviewTurnsRepository)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utils.SignInTokenPayload{
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
					GetChatHistoryBySessionIDWithCursor(ctx, mock.MatchedBy(func(req *db.GetChatHistoryBySessionIDWithCursorParams) bool {
						return req.SessionID == uuid.MustParse(globalID) && req.TurnNo == 5 && req.Limit == constants.DefaultPageSize
					})).
					Return([]db.GetChatHistoryBySessionIDWithCursorRow{}, nil)

				return mockAuthContext, mockInterviewTurnsRepo, mockRedisClient, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotErr error, gotResp *entities.GetChatHistoryBySessionTokenResp) {
				assert.NoError(t, gotErr)
				assert.Len(t, gotResp.ChatHistory, 0)
				assert.Equal(t, int32(0), gotResp.CursorTurnNext) // Should be 0 when no results
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
						Payload: &utils.SignInTokenPayload{
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
						Payload: &utils.SignInTokenPayload{
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
					GetChatHistoryBySessionID(ctx, mock.Anything).
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
				nil,
			)

			gotResp, gotErr := svc.GetChatHistoryBySessionToken(ctx, tC.input)

			tC.verify(t, gotErr, gotResp)
		})
	}
}

func TestInterviewSessionService_CheckExistsAndInitStartedAtInterviewSession(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()
	globalID := "550e8400-e29b-41d4-a716-446655440000"
	startAt := time.Date(2025, 9, 4, 18, 35, 49, 777972000, time.FixedZone("UTC+7", 7*3600))

	validResp := &entities.GetInterviewSessionStateResp{
		StartedAt:             utils.FormatToUTCString(startAt),
		IsStartedConversation: true,
	}

	testCases := []struct {
		name   string
		input  string
		setup  func() *mockRepositories.MockInterviewSessionRepository
		verify func(t *testing.T, gotResp *entities.GetInterviewSessionStateResp, gotErr error)
	}{
		{
			name:  "Success - WithStartedAtValid",
			input: globalID,
			setup: func() *mockRepositories.MockInterviewSessionRepository {
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)

				mockInterviewSessionRepo.EXPECT().
					GetSessionState(ctx, uuid.MustParse(globalID)).
					Return(&db.GetSessionStateRow{
						StartedAt:             sql.NullTime{Time: startAt, Valid: true},
						IsStartedConversation: sql.NullBool{Bool: true, Valid: true},
					}, nil)

				return mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotResp *entities.GetInterviewSessionStateResp, gotErr error) {
				assert.NoError(t, gotErr)
				assert.Equal(t, validResp, gotResp)
			},
		},
		{
			name:  "Success - WithStartedAtInvalid",
			input: globalID,
			setup: func() *mockRepositories.MockInterviewSessionRepository {
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)

				mockInterviewSessionRepo.EXPECT().
					GetSessionState(ctx, uuid.MustParse(globalID)).
					Return(&db.GetSessionStateRow{
						StartedAt:             sql.NullTime{Time: time.Time{}, Valid: false},
						IsStartedConversation: sql.NullBool{Bool: true, Valid: true},
					}, nil)

				mockInterviewSessionRepo.EXPECT().
					UpdateStartedAtInterviewSession(ctx, mock.MatchedBy(func(req *db.UpdateStartedAtInterviewSessionParams) bool {
						return req.ID == uuid.MustParse(globalID)
					})).
					Return(nil)

				return mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotResp *entities.GetInterviewSessionStateResp, gotErr error) {
				assert.NoError(t, gotErr)
				assert.Equal(t, true, gotResp.IsStartedConversation)
			},
		},
		{
			name:  "Error - WithGetStartedAtInterviewSessionError",
			input: "invalid_session_id",
			setup: func() *mockRepositories.MockInterviewSessionRepository {
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)

				return mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotResp *entities.GetInterviewSessionStateResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "The UUID is invalid. Please try again.")
				assert.Contains(t, gotErr.Error(), "[INS0107]")
				assert.Nil(t, gotResp)
			},
		},
		{
			name:  "Error - WithGetStartedAtInterviewSessionError",
			input: globalID,
			setup: func() *mockRepositories.MockInterviewSessionRepository {
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)

				mockInterviewSessionRepo.EXPECT().
					GetSessionState(ctx, uuid.MustParse(globalID)).
					Return(&db.GetSessionStateRow{
						StartedAt:             sql.NullTime{Time: time.Time{}, Valid: false},
						IsStartedConversation: sql.NullBool{Bool: true, Valid: true},
					}, errors.New("get started at interview session error"))

				return mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotResp *entities.GetInterviewSessionStateResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
			},
		},
		{
			name:  "Error - WithUpdateStartedAtInterviewSessionError",
			input: globalID,
			setup: func() *mockRepositories.MockInterviewSessionRepository {
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)

				mockInterviewSessionRepo.EXPECT().
					GetSessionState(ctx, uuid.MustParse(globalID)).
					Return(&db.GetSessionStateRow{
						StartedAt:             sql.NullTime{Time: time.Time{}, Valid: false},
						IsStartedConversation: sql.NullBool{Bool: true, Valid: true},
					}, nil)

				mockInterviewSessionRepo.EXPECT().
					UpdateStartedAtInterviewSession(ctx, mock.MatchedBy(func(req *db.UpdateStartedAtInterviewSessionParams) bool {
						return req.ID == uuid.MustParse(globalID)
					})).
					Return(errors.New("update started at interview session error"))

				return mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotResp *entities.GetInterviewSessionStateResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			mockInterviewSessionRepo := tC.setup()
			defer func() {
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
				nil,
				nil,
				nil,
				nil,
				nil,
			)

			gotResp, gotErr := svc.GetInterviewSessionState(ctx, tC.input)

			tC.verify(t, gotResp, gotErr)
		})
	}
}

func TestInterviewSessionService_EndInterviewSessionsByUserID(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()
	sessionID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	validScore := []db.GetAllEvaluationsBySessionIDRow{
		{
			OverallScore: "3.5",
			SummaryMd:    "summary1",
		},
		{
			OverallScore: "2.5",
			SummaryMd:    "summary2",
		},
		{
			OverallScore: "3",
			SummaryMd:    "summary3",
		},
	}

	invalidScore := []db.GetAllEvaluationsBySessionIDRow{
		{
			OverallScore: "invalid",
			SummaryMd:    "summary1",
		},
	}

	testCases := []struct {
		name   string
		input  *entities.EndInterviewSessionReq
		setup  func() (*mockDatabase.MockRedisClient, *mockServices.MockEvaluationService, *mockRepositories.MockEvaluationScoresRepository, *mockRepositories.MockInterviewSessionRepository)
		verify func(t *testing.T, gotErr error)
	}{
		{
			name: "Success",
			input: &entities.EndInterviewSessionReq{
				SessionId: sessionID.String(),
				Status:    constants.StatusCompleted,
			},
			setup: func() (*mockDatabase.MockRedisClient, *mockServices.MockEvaluationService, *mockRepositories.MockEvaluationScoresRepository, *mockRepositories.MockInterviewSessionRepository) {
				mockRedisClient := mockDatabase.NewMockRedisClient(t)
				mockEvaluationService := mockServices.NewMockEvaluationService(t)
				mockEvaluationScoresRepo := mockRepositories.NewMockEvaluationScoresRepository(t)
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				mockEvaluationScoresRepo.EXPECT().GetAllEvaluationsBySessionID(ctx, sessionID).
					Return(validScore, nil)

				getOverallSummary := &repositories.CreateEvaluationOverallSummaryTxReq{
					SummaryMd: []string{
						"summary1",
						"summary2",
						"summary3",
					},
				}

				mockEvaluationScoresRepo.EXPECT().GetEvaluationOverallSummary(ctx, getOverallSummary).
					Return(&repositories.CreateEvaluationOverallSummaryTxResp{
						OverallSummaryMd: "summary overall",
					}, nil)

				mockInterviewSessionRepo.EXPECT().EndInterviewSession(ctx, mock.MatchedBy(func(req *db.EndInterviewSessionParams) bool {
					return req.ID == sessionID && req.Status == constants.StatusCompleted && req.OverallScore.Valid && req.OverallScore.Float64 == 3.00 && req.SummaryMd.String == "summary overall"
				})).
					Return(nil)

				mockEvaluationService.EXPECT().FinalizeSessionPhraseEvaluation(ctx, sessionID.String()).
					Return(nil)

				return mockRedisClient, mockEvaluationService, mockEvaluationScoresRepo, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotErr error) {
				assert.NoError(t, gotErr)
			},
		},
		{
			name: "Success WithNoEvaluations",
			input: &entities.EndInterviewSessionReq{
				SessionId: sessionID.String(),
				Status:    constants.StatusCompleted,
			},
			setup: func() (*mockDatabase.MockRedisClient, *mockServices.MockEvaluationService, *mockRepositories.MockEvaluationScoresRepository, *mockRepositories.MockInterviewSessionRepository) {
				mockRedisClient := mockDatabase.NewMockRedisClient(t)
				mockEvaluationService := mockServices.NewMockEvaluationService(t)
				mockEvaluationScoresRepo := mockRepositories.NewMockEvaluationScoresRepository(t)
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				mockEvaluationScoresRepo.EXPECT().GetAllEvaluationsBySessionID(ctx, sessionID).
					Return([]db.GetAllEvaluationsBySessionIDRow{}, nil)

				mockInterviewSessionRepo.EXPECT().EndInterviewSession(ctx, mock.MatchedBy(func(req *db.EndInterviewSessionParams) bool {
					return req.ID == sessionID && req.Status == constants.StatusCompleted && req.OverallScore.Valid && req.OverallScore.Float64 == 0.00 && req.SummaryMd.String == constants.BlankOverallSummaryMd
				})).
					Return(nil)

				mockEvaluationService.EXPECT().FinalizeSessionPhraseEvaluation(ctx, sessionID.String()).
					Return(nil)

				return mockRedisClient, mockEvaluationService, mockEvaluationScoresRepo, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotErr error) {
				assert.NoError(t, gotErr)
			},
		},
		{
			name: "Error WithInvalidSessionID",
			input: &entities.EndInterviewSessionReq{
				SessionId: "invalid-session-id",
				Status:    constants.StatusCompleted,
			},
			setup: func() (*mockDatabase.MockRedisClient, *mockServices.MockEvaluationService, *mockRepositories.MockEvaluationScoresRepository, *mockRepositories.MockInterviewSessionRepository) {
				mockRedisClient := mockDatabase.NewMockRedisClient(t)
				mockEvaluationService := mockServices.NewMockEvaluationService(t)
				mockEvaluationScoresRepo := mockRepositories.NewMockEvaluationScoresRepository(t)
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				mockEvaluationService.EXPECT().FinalizeSessionFailed(ctx, "invalid-session-id")

				return mockRedisClient, mockEvaluationService, mockEvaluationScoresRepo, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Equal(t, app_error.ErrCodeGeneralInvalidUUID, gotErr.(*app_error.AppError).Code)
			},
		},
		{
			name: "Error WithGetInterviewSessionStatusByIDError",
			input: &entities.EndInterviewSessionReq{
				SessionId: sessionID.String(),
				Status:    constants.StatusCompleted,
			},
			setup: func() (*mockDatabase.MockRedisClient, *mockServices.MockEvaluationService, *mockRepositories.MockEvaluationScoresRepository, *mockRepositories.MockInterviewSessionRepository) {
				mockRedisClient := mockDatabase.NewMockRedisClient(t)
				mockEvaluationService := mockServices.NewMockEvaluationService(t)
				mockEvaluationScoresRepo := mockRepositories.NewMockEvaluationScoresRepository(t)
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				mockEvaluationScoresRepo.EXPECT().GetAllEvaluationsBySessionID(ctx, sessionID).
					Return([]db.GetAllEvaluationsBySessionIDRow{}, nil)

				mockInterviewSessionRepo.EXPECT().EndInterviewSession(ctx, mock.MatchedBy(func(req *db.EndInterviewSessionParams) bool {
					return req.ID.String() == sessionID.String()
				})).Return(errors.New("get interview session status by ID error"))

				mockEvaluationService.EXPECT().FinalizeSessionFailed(ctx, sessionID.String())

				return mockRedisClient, mockEvaluationService, mockEvaluationScoresRepo, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "get interview session status by ID error")
			},
		},
		{
			name: "Error WithGetAllEvaluationsBySessionIDError",
			input: &entities.EndInterviewSessionReq{
				SessionId: sessionID.String(),
				Status:    constants.StatusCompleted,
			},
			setup: func() (*mockDatabase.MockRedisClient, *mockServices.MockEvaluationService, *mockRepositories.MockEvaluationScoresRepository, *mockRepositories.MockInterviewSessionRepository) {
				mockRedisClient := mockDatabase.NewMockRedisClient(t)
				mockEvaluationService := mockServices.NewMockEvaluationService(t)
				mockEvaluationScoresRepo := mockRepositories.NewMockEvaluationScoresRepository(t)
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				mockEvaluationScoresRepo.EXPECT().GetAllEvaluationsBySessionID(ctx, sessionID).
					Return(nil, errors.New("get all evaluations by session ID error"))

				mockEvaluationService.EXPECT().FinalizeSessionFailed(ctx, sessionID.String()).
					Return()

				return mockRedisClient, mockEvaluationService, mockEvaluationScoresRepo, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "get all evaluations by session ID error")
			},
		},
		{
			name: "Error WithParseOverallScoreError",
			input: &entities.EndInterviewSessionReq{
				SessionId: sessionID.String(),
				Status:    constants.StatusCompleted,
			},
			setup: func() (*mockDatabase.MockRedisClient, *mockServices.MockEvaluationService, *mockRepositories.MockEvaluationScoresRepository, *mockRepositories.MockInterviewSessionRepository) {
				mockRedisClient := mockDatabase.NewMockRedisClient(t)
				mockEvaluationService := mockServices.NewMockEvaluationService(t)
				mockEvaluationScoresRepo := mockRepositories.NewMockEvaluationScoresRepository(t)
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				mockEvaluationScoresRepo.EXPECT().GetAllEvaluationsBySessionID(ctx, sessionID).
					Return(invalidScore, nil)

				mockEvaluationService.EXPECT().FinalizeSessionFailed(ctx, sessionID.String())

				return mockRedisClient, mockEvaluationService, mockEvaluationScoresRepo, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "The number is invalid. Please try again.")
				assert.Contains(t, gotErr.Error(), "[INS0114]")
			},
		},
		{
			name: "Error WithGetEvaluationOverallSummaryError",
			input: &entities.EndInterviewSessionReq{
				SessionId: sessionID.String(),
				Status:    constants.StatusCompleted,
			},
			setup: func() (*mockDatabase.MockRedisClient, *mockServices.MockEvaluationService, *mockRepositories.MockEvaluationScoresRepository, *mockRepositories.MockInterviewSessionRepository) {
				mockRedisClient := mockDatabase.NewMockRedisClient(t)
				mockEvaluationService := mockServices.NewMockEvaluationService(t)
				mockEvaluationScoresRepo := mockRepositories.NewMockEvaluationScoresRepository(t)
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				mockEvaluationScoresRepo.EXPECT().GetAllEvaluationsBySessionID(ctx, sessionID).
					Return(validScore, nil)

				getOverallSummary := &repositories.CreateEvaluationOverallSummaryTxReq{
					SummaryMd: []string{
						"summary1",
						"summary2",
						"summary3",
					},
				}

				mockEvaluationScoresRepo.EXPECT().GetEvaluationOverallSummary(ctx, getOverallSummary).
					Return(&repositories.CreateEvaluationOverallSummaryTxResp{
						OverallSummaryMd: "summary overall",
					}, errors.New("get evaluation overall summary error"))

				mockEvaluationService.EXPECT().FinalizeSessionFailed(ctx, sessionID.String())

				return mockRedisClient, mockEvaluationService, mockEvaluationScoresRepo, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "get evaluation overall summary error")
			},
		},
		{
			name: "Error WithEndInterviewSessionError",
			input: &entities.EndInterviewSessionReq{
				SessionId: sessionID.String(),
				Status:    constants.StatusCompleted,
			},
			setup: func() (*mockDatabase.MockRedisClient, *mockServices.MockEvaluationService, *mockRepositories.MockEvaluationScoresRepository, *mockRepositories.MockInterviewSessionRepository) {
				mockRedisClient := mockDatabase.NewMockRedisClient(t)
				mockEvaluationService := mockServices.NewMockEvaluationService(t)
				mockEvaluationScoresRepo := mockRepositories.NewMockEvaluationScoresRepository(t)
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				mockEvaluationScoresRepo.EXPECT().GetAllEvaluationsBySessionID(ctx, sessionID).
					Return(validScore, nil)

				getOverallSummary := &repositories.CreateEvaluationOverallSummaryTxReq{
					SummaryMd: []string{
						"summary1",
						"summary2",
						"summary3",
					},
				}

				mockEvaluationScoresRepo.EXPECT().GetEvaluationOverallSummary(ctx, getOverallSummary).
					Return(&repositories.CreateEvaluationOverallSummaryTxResp{
						OverallSummaryMd: "summary overall",
					}, nil)

				mockInterviewSessionRepo.EXPECT().EndInterviewSession(ctx, mock.MatchedBy(func(req *db.EndInterviewSessionParams) bool {
					return req.ID == sessionID && req.Status == constants.StatusCompleted && req.OverallScore.Valid && req.OverallScore.Float64 == 3.00 && req.SummaryMd.String == "summary overall"
				})).
					Return(errors.New("end interview session error"))

				mockEvaluationService.EXPECT().FinalizeSessionFailed(ctx, sessionID.String())

				return mockRedisClient, mockEvaluationService, mockEvaluationScoresRepo, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "end interview session error")
			},
		},
		{
			name: "Error WithFinalizeSessionPhraseEvaluationError",
			input: &entities.EndInterviewSessionReq{
				SessionId: sessionID.String(),
				Status:    constants.StatusCompleted,
			},
			setup: func() (*mockDatabase.MockRedisClient, *mockServices.MockEvaluationService, *mockRepositories.MockEvaluationScoresRepository, *mockRepositories.MockInterviewSessionRepository) {
				mockRedisClient := mockDatabase.NewMockRedisClient(t)
				mockEvaluationService := mockServices.NewMockEvaluationService(t)
				mockEvaluationScoresRepo := mockRepositories.NewMockEvaluationScoresRepository(t)
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				mockEvaluationScoresRepo.EXPECT().GetAllEvaluationsBySessionID(ctx, sessionID).
					Return(validScore, nil)

				getOverallSummary := &repositories.CreateEvaluationOverallSummaryTxReq{
					SummaryMd: []string{
						"summary1",
						"summary2",
						"summary3",
					},
				}

				mockEvaluationScoresRepo.EXPECT().GetEvaluationOverallSummary(ctx, getOverallSummary).
					Return(&repositories.CreateEvaluationOverallSummaryTxResp{
						OverallSummaryMd: "summary overall",
					}, nil)

				mockInterviewSessionRepo.EXPECT().EndInterviewSession(ctx, mock.MatchedBy(func(req *db.EndInterviewSessionParams) bool {
					return req.ID == sessionID && req.Status == constants.StatusCompleted && req.OverallScore.Valid && req.OverallScore.Float64 == 3.00 && req.SummaryMd.String == "summary overall"
				})).
					Return(nil)

				mockEvaluationService.EXPECT().FinalizeSessionPhraseEvaluation(ctx, sessionID.String()).
					Return(errors.New("finalize session phrase evaluation error"))

				mockEvaluationService.EXPECT().FinalizeSessionFailed(ctx, sessionID.String())

				return mockRedisClient, mockEvaluationService, mockEvaluationScoresRepo, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "finalize session phrase evaluation error")
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			mockRedisClient, mockEvaluationService, mockEvaluationScoresRepo, mockInterviewSessionRepo := tC.setup()

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
				mockEvaluationService,
				mockEvaluationScoresRepo,
				nil,
				nil,
			)

			gotErr := svc.EndInterviewSession(ctx, tC.input)

			tC.verify(t, gotErr)
		})
	}
}

func TestInterviewSessionService_ListInterviewSessionsByUserIDWithCursor(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()
	userID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	invalidUUID := "invalid-uuid"
	validTime := "2023-01-01T12:00:00Z"
	invalidTime := "invalid-time"
	invalidPaginationType := "invalid-pagination-type"

	validCursor := &entities.Cursor{
		ID:        userID.String(),
		CreatedAt: validTime,
	}

	invalidCursorID := &entities.Cursor{
		ID:        invalidUUID,
		CreatedAt: validTime,
	}

	invalidCursorTime := &entities.Cursor{
		ID:        userID.String(),
		CreatedAt: invalidTime,
	}

	testCases := []struct {
		name   string
		input  *entities.ListInterviewSessionsByUserIDWithCursorReq
		setup  func() (*mockMiddleware.MockAuthContext, *mockRepositories.MockInterviewSessionRepository)
		verify func(t *testing.T, gotResp *entities.ListInterviewSessionsByUserIDResp, gotErr error)
	}{
		{
			name:  "Success - ListInterviewSessionsWithDefaultParameters",
			input: &entities.ListInterviewSessionsByUserIDWithCursorReq{},
			setup: func() (*mockMiddleware.MockAuthContext, *mockRepositories.MockInterviewSessionRepository) {
				mockAuthContext := mockMiddleware.NewMockAuthContext(t)
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				mockAuthContext.EXPECT().GetAuthContext(ctx).Return(&middleware.AuthPayload{
					Payload: &utils.SignInTokenPayload{
						UserID: userID.String(),
					},
				}, nil)

				dbRows := []db.ListInterviewSessionsByUserIDFirstPageRow{
					{
						ID:             userID,
						ResumeID:       userID,
						ResumeFileName: "test-resume.pdf",
						Position:       "Software Engineer",
						Status:         "completed",
						CreatedAt:      sql.NullTime{Time: time.Now(), Valid: true},
						OverallScore:   sql.NullFloat64{Float64: 2, Valid: true},
						StartedAt:      sql.NullTime{Time: time.Now().Add(-30 * time.Minute), Valid: true},
						EndedAt:        sql.NullTime{Time: time.Now(), Valid: true},
					},
				}

				mockInterviewSessionRepo.EXPECT().ListInterviewSessionsByUserIDFirstPage(ctx, mock.MatchedBy(func(req *db.ListInterviewSessionsByUserIDFirstPageParams) bool {
					return req.UserID == userID && req.Limit == 20 && req.Column2 == ""
				})).
					Return(dbRows, nil)

				mockInterviewSessionRepo.EXPECT().CountInterviewSessionsByUserID(ctx, mock.MatchedBy(func(req *db.CountInterviewSessionsByUserIDParams) bool {
					return req.UserID == userID && req.Column2 == ""
				})).Return(int64(1), nil)

				return mockAuthContext, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotResp *entities.ListInterviewSessionsByUserIDResp, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, gotResp)
				assert.Len(t, gotResp.Sessions, 1)
				assert.Equal(t, userID.String(), gotResp.Sessions[0].ID)
				assert.Equal(t, "test-resume.pdf", gotResp.Sessions[0].ResumeFileName)
				assert.Equal(t, "Software Engineer", gotResp.Sessions[0].Position)
				assert.Equal(t, "completed", gotResp.Sessions[0].Status)
				assert.Equal(t, float64(2), gotResp.Sessions[0].OverallScore)
				assert.Equal(t, "30.00", gotResp.Sessions[0].TotalTime)
				assert.Equal(t, 1, gotResp.TotalPages)
				assert.Equal(t, 20, gotResp.PageSize)
				assert.NotNil(t, gotResp.NextCursor)
				assert.Equal(t, "#FFC107", gotResp.Sessions[0].OverallScoreColor)
				assert.Equal(t, "#28A745", gotResp.Sessions[0].StatusColor)
			},
		},
		{
			name: "Success - ListInterviewSessionsWithCustomLimit",
			input: &entities.ListInterviewSessionsByUserIDWithCursorReq{
				Limit: func() *int {
					limit := 10
					return &limit
				}(),
			},
			setup: func() (*mockMiddleware.MockAuthContext, *mockRepositories.MockInterviewSessionRepository) {
				mockAuthContext := mockMiddleware.NewMockAuthContext(t)
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				mockAuthContext.EXPECT().GetAuthContext(ctx).Return(&middleware.AuthPayload{
					Payload: &utils.SignInTokenPayload{
						UserID: userID.String(),
					},
				}, nil)
				mockInterviewSessionRepo.EXPECT().ListInterviewSessionsByUserIDFirstPage(ctx, mock.MatchedBy(func(req *db.ListInterviewSessionsByUserIDFirstPageParams) bool {
					return req.UserID == userID && req.Limit == 10 && req.Column2 == ""
				})).
					Return([]db.ListInterviewSessionsByUserIDFirstPageRow{}, nil)

				mockInterviewSessionRepo.EXPECT().CountInterviewSessionsByUserID(ctx, mock.MatchedBy(func(req *db.CountInterviewSessionsByUserIDParams) bool {
					return req.UserID == userID && req.Column2 == ""
				})).Return(int64(0), nil)

				return mockAuthContext, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotResp *entities.ListInterviewSessionsByUserIDResp, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, gotResp)
				assert.Len(t, gotResp.Sessions, 0)
				assert.Equal(t, 0, gotResp.TotalPages)
				assert.Equal(t, 10, gotResp.PageSize)
				assert.Nil(t, gotResp.NextCursor)
			},
		},
		{
			name: "Success - ListInterviewSessionsWithSearchText",
			input: &entities.ListInterviewSessionsByUserIDWithCursorReq{
				SearchText: func() *string { search := "engineer"; return &search }(),
			},
			setup: func() (*mockMiddleware.MockAuthContext, *mockRepositories.MockInterviewSessionRepository) {
				mockAuthContext := mockMiddleware.NewMockAuthContext(t)
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				mockAuthContext.EXPECT().GetAuthContext(ctx).Return(&middleware.AuthPayload{
					Payload: &utils.SignInTokenPayload{
						UserID: userID.String(),
					},
				}, nil)

				mockInterviewSessionRepo.EXPECT().ListInterviewSessionsByUserIDFirstPage(ctx, mock.MatchedBy(func(req *db.ListInterviewSessionsByUserIDFirstPageParams) bool {
					return req.UserID == userID && req.Limit == 20 && req.Column2 == "engineer"
				})).
					Return([]db.ListInterviewSessionsByUserIDFirstPageRow{}, nil)

				mockInterviewSessionRepo.EXPECT().CountInterviewSessionsByUserID(ctx, mock.MatchedBy(func(req *db.CountInterviewSessionsByUserIDParams) bool {
					return req.UserID == userID && req.Column2 == "engineer"
				})).Return(int64(0), nil)

				return mockAuthContext, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotResp *entities.ListInterviewSessionsByUserIDResp, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, gotResp)
				assert.Len(t, gotResp.Sessions, 0)
				assert.Equal(t, 0, gotResp.TotalPages)
				assert.Equal(t, 20, gotResp.PageSize)
			},
		},
		{
			name: "Success - ListInterviewSessionsWithCursor",
			input: &entities.ListInterviewSessionsByUserIDWithCursorReq{
				Cursor: validCursor,
			},
			setup: func() (*mockMiddleware.MockAuthContext, *mockRepositories.MockInterviewSessionRepository) {
				mockAuthContext := mockMiddleware.NewMockAuthContext(t)
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				mockAuthContext.EXPECT().GetAuthContext(ctx).Return(&middleware.AuthPayload{
					Payload: &utils.SignInTokenPayload{
						UserID: userID.String(),
					},
				}, nil)

				parsedTime, _ := time.Parse(time.RFC3339, validCursor.CreatedAt)
				mockInterviewSessionRepo.EXPECT().ListInterviewSessionsByUserIDWithCursor(ctx, mock.MatchedBy(func(req *db.ListInterviewSessionsByUserIDWithCursorParams) bool {
					return req.UserID == userID && req.Limit == 20 && req.Column2 == "" && req.CreatedAt.Time.Equal(parsedTime) && req.Column4 == "next" && req.ID == uuid.MustParse(validCursor.ID)
				})).
					Return([]db.ListInterviewSessionsByUserIDWithCursorRow{}, nil)

				mockInterviewSessionRepo.EXPECT().CountInterviewSessionsByUserID(ctx, mock.MatchedBy(func(req *db.CountInterviewSessionsByUserIDParams) bool {
					return req.UserID == userID && req.Column2 == ""
				})).Return(int64(0), nil)

				return mockAuthContext, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotResp *entities.ListInterviewSessionsByUserIDResp, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, gotResp)
				assert.Len(t, gotResp.Sessions, 0)
				assert.Equal(t, 0, gotResp.TotalPages)
				assert.Equal(t, 20, gotResp.PageSize)
			},
		},
		{
			name: "Success - ListInterviewWithCursorWithDefaultOverAllScoreAndTime",
			input: &entities.ListInterviewSessionsByUserIDWithCursorReq{
				Cursor: validCursor,
			},
			setup: func() (*mockMiddleware.MockAuthContext, *mockRepositories.MockInterviewSessionRepository) {
				mockAuthContext := mockMiddleware.NewMockAuthContext(t)
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				mockAuthContext.EXPECT().GetAuthContext(ctx).Return(&middleware.AuthPayload{
					Payload: &utils.SignInTokenPayload{
						UserID: userID.String(),
					},
				}, nil)

				parsedTime, _ := time.Parse(time.RFC3339, validCursor.CreatedAt)
				mockInterviewSessionRepo.EXPECT().ListInterviewSessionsByUserIDWithCursor(ctx, mock.MatchedBy(func(req *db.ListInterviewSessionsByUserIDWithCursorParams) bool {
					return req.UserID == userID && req.Limit == 20 && req.Column2 == "" && req.CreatedAt.Time.Equal(parsedTime) && req.Column4 == "next" && req.ID == uuid.MustParse(validCursor.ID)
				})).
					Return([]db.ListInterviewSessionsByUserIDWithCursorRow{
						{
							ID:             userID,
							ResumeID:       userID,
							ResumeFileName: "test-resume.pdf",
							Position:       "Software Engineer",
							Status:         "completed",
							CreatedAt:      sql.NullTime{Time: time.Now(), Valid: true},
							OverallScore:   sql.NullFloat64{Valid: false},
							StartedAt:      sql.NullTime{Valid: false},
							EndedAt:        sql.NullTime{Valid: false},
						},
					}, nil)

				mockInterviewSessionRepo.EXPECT().CountInterviewSessionsByUserID(ctx, mock.MatchedBy(func(req *db.CountInterviewSessionsByUserIDParams) bool {
					return req.UserID == userID && req.Column2 == ""
				})).Return(int64(1), nil)

				return mockAuthContext, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotResp *entities.ListInterviewSessionsByUserIDResp, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, gotResp)
				assert.Len(t, gotResp.Sessions, 1)
				assert.Equal(t, 1, gotResp.TotalPages)
				assert.Equal(t, 20, gotResp.PageSize)
			},
		},
		{
			name: "Success - ListInterviewSessionsWithAllParameters",
			input: &entities.ListInterviewSessionsByUserIDWithCursorReq{
				SearchText: func() *string { search := "engineer"; return &search }(),
				Cursor:     validCursor,
				Limit:      func() *int { limit := 15; return &limit }(),
			},
			setup: func() (*mockMiddleware.MockAuthContext, *mockRepositories.MockInterviewSessionRepository) {
				mockAuthContext := mockMiddleware.NewMockAuthContext(t)
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				mockAuthContext.EXPECT().GetAuthContext(ctx).Return(&middleware.AuthPayload{
					Payload: &utils.SignInTokenPayload{
						UserID: userID.String(),
					},
				}, nil)

				parsedTime, _ := time.Parse(time.RFC3339, validCursor.CreatedAt)
				mockInterviewSessionRepo.EXPECT().ListInterviewSessionsByUserIDWithCursor(ctx, mock.MatchedBy(func(req *db.ListInterviewSessionsByUserIDWithCursorParams) bool {
					return req.UserID == userID && req.Limit == 15 && req.Column2 == "engineer" && req.CreatedAt.Time.Equal(parsedTime) && req.Column4 == "next" && req.ID == uuid.MustParse(validCursor.ID)
				})).
					Return([]db.ListInterviewSessionsByUserIDWithCursorRow{}, nil)

				mockInterviewSessionRepo.EXPECT().CountInterviewSessionsByUserID(ctx, mock.MatchedBy(func(req *db.CountInterviewSessionsByUserIDParams) bool {
					return req.UserID == userID && req.Column2 == "engineer"
				})).Return(int64(0), nil)

				return mockAuthContext, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotResp *entities.ListInterviewSessionsByUserIDResp, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, gotResp)
				assert.Len(t, gotResp.Sessions, 0)
				assert.Equal(t, 0, gotResp.TotalPages)
				assert.Equal(t, 15, gotResp.PageSize)
			},
		},
		{
			name: "Success - ListInterviewSessionsWithPaginationHasMore",
			input: &entities.ListInterviewSessionsByUserIDWithCursorReq{
				Limit: func() *int { limit := 2; return &limit }(),
			},
			setup: func() (*mockMiddleware.MockAuthContext, *mockRepositories.MockInterviewSessionRepository) {
				mockAuthContext := mockMiddleware.NewMockAuthContext(t)
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				mockAuthContext.EXPECT().GetAuthContext(ctx).Return(&middleware.AuthPayload{
					Payload: &utils.SignInTokenPayload{
						UserID: userID.String(),
					},
				}, nil)

				dbRows := []db.ListInterviewSessionsByUserIDFirstPageRow{
					{
						ID:             userID,
						ResumeID:       userID,
						ResumeFileName: "test-resume1.pdf",
						Position:       "Software Engineer",
						Status:         "completed",
						CreatedAt:      sql.NullTime{Time: time.Now(), Valid: true},
						OverallScore:   sql.NullFloat64{Float64: 2, Valid: true},
						StartedAt:      sql.NullTime{Time: time.Now().Add(-30 * time.Minute), Valid: true},
						EndedAt:        sql.NullTime{Time: time.Now(), Valid: true},
					},
					{
						ID:             uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"),
						ResumeID:       userID,
						ResumeFileName: "test-resume2.pdf",
						Position:       "Senior Engineer",
						Status:         "completed",
						CreatedAt:      sql.NullTime{Time: time.Now().Add(-time.Hour), Valid: true},
						OverallScore:   sql.NullFloat64{Float64: 4, Valid: true},
						StartedAt:      sql.NullTime{Time: time.Now().Add(-90 * time.Minute), Valid: true},
						EndedAt:        sql.NullTime{Time: time.Now().Add(-60 * time.Minute), Valid: true},
					},
				}

				mockInterviewSessionRepo.EXPECT().ListInterviewSessionsByUserIDFirstPage(ctx, mock.MatchedBy(func(req *db.ListInterviewSessionsByUserIDFirstPageParams) bool {
					return req.UserID == userID && req.Limit == 2 && req.Column2 == ""
				})).Return(dbRows, nil)

				mockInterviewSessionRepo.EXPECT().CountInterviewSessionsByUserID(ctx, mock.MatchedBy(func(req *db.CountInterviewSessionsByUserIDParams) bool {
					return req.UserID == userID && req.Column2 == ""
				})).Return(int64(5), nil)

				return mockAuthContext, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotResp *entities.ListInterviewSessionsByUserIDResp, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, gotResp)
				assert.Len(t, gotResp.Sessions, 2)
				assert.Equal(t, 3, gotResp.TotalPages)
				assert.Equal(t, 2, gotResp.PageSize)
				assert.NotNil(t, gotResp.NextCursor)
				assert.Equal(t, "550e8400-e29b-41d4-a716-446655440001", gotResp.NextCursor.ID)
				assert.Equal(t, "#FFC107", gotResp.Sessions[0].OverallScoreColor)
				assert.Equal(t, "#28A745", gotResp.Sessions[0].StatusColor)
				assert.Equal(t, "#28A745", gotResp.Sessions[1].OverallScoreColor)
				assert.Equal(t, "#28A745", gotResp.Sessions[1].StatusColor)
			},
		},
		{
			name:  "Success - ListInterviewSessionsWithNullScoresAndTimes",
			input: &entities.ListInterviewSessionsByUserIDWithCursorReq{},
			setup: func() (*mockMiddleware.MockAuthContext, *mockRepositories.MockInterviewSessionRepository) {
				mockAuthContext := mockMiddleware.NewMockAuthContext(t)
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				mockAuthContext.EXPECT().GetAuthContext(ctx).Return(&middleware.AuthPayload{
					Payload: &utils.SignInTokenPayload{
						UserID: userID.String(),
					},
				}, nil)

				// Mock response with null scores and times
				dbRows := []db.ListInterviewSessionsByUserIDFirstPageRow{
					{
						ID:             userID,
						ResumeID:       userID,
						ResumeFileName: "test-resume.pdf",
						Position:       "Software Engineer",
						Status:         "pending",
						CreatedAt:      sql.NullTime{Time: time.Now(), Valid: true},
						OverallScore:   sql.NullFloat64{Valid: false},
						StartedAt:      sql.NullTime{Valid: false},
						EndedAt:        sql.NullTime{Valid: false},
					},
				}

				mockInterviewSessionRepo.EXPECT().ListInterviewSessionsByUserIDFirstPage(ctx, mock.MatchedBy(func(req *db.ListInterviewSessionsByUserIDFirstPageParams) bool {
					return req.UserID == userID && req.Limit == 20 && req.Column2 == ""
				})).Return(dbRows, nil)

				mockInterviewSessionRepo.EXPECT().CountInterviewSessionsByUserID(ctx, mock.MatchedBy(func(req *db.CountInterviewSessionsByUserIDParams) bool {
					return req.UserID == userID && req.Column2 == ""
				})).Return(int64(1), nil)

				return mockAuthContext, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotResp *entities.ListInterviewSessionsByUserIDResp, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, gotResp)
				assert.Len(t, gotResp.Sessions, 1)
				assert.Equal(t, 0.00, gotResp.Sessions[0].OverallScore)
				assert.Equal(t, "0.00", gotResp.Sessions[0].TotalTime)
				assert.Equal(t, 1, gotResp.TotalPages)
				assert.Equal(t, 20, gotResp.PageSize)
			},
		},
		{
			name:  "Error - Auth context failure",
			input: &entities.ListInterviewSessionsByUserIDWithCursorReq{},
			setup: func() (*mockMiddleware.MockAuthContext, *mockRepositories.MockInterviewSessionRepository) {
				mockAuthContext := mockMiddleware.NewMockAuthContext(t)
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				mockAuthContext.EXPECT().GetAuthContext(ctx).Return(nil, errors.New("auth context error"))

				return mockAuthContext, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotResp *entities.ListInterviewSessionsByUserIDResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
				assert.Equal(t, "auth context error", gotErr.Error())
			},
		},
		{
			name:  "Error - InvalidUserId",
			input: &entities.ListInterviewSessionsByUserIDWithCursorReq{},
			setup: func() (*mockMiddleware.MockAuthContext, *mockRepositories.MockInterviewSessionRepository) {
				mockAuthContext := mockMiddleware.NewMockAuthContext(t)
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				mockAuthContext.EXPECT().GetAuthContext(ctx).Return(&middleware.AuthPayload{
					Payload: &utils.SignInTokenPayload{
						UserID: "invalid-user-id",
					},
				}, nil)

				return mockAuthContext, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotResp *entities.ListInterviewSessionsByUserIDResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
				assert.Contains(t, gotErr.Error(), "The UUID is invalid. Please try again.")
				assert.Contains(t, gotErr.Error(), "[INS0107]")
			},
		},
		{
			name: "Error - InvalidPaginationType",
			input: &entities.ListInterviewSessionsByUserIDWithCursorReq{
				Type: &invalidPaginationType,
			},
			setup: func() (*mockMiddleware.MockAuthContext, *mockRepositories.MockInterviewSessionRepository) {
				mockAuthContext := mockMiddleware.NewMockAuthContext(t)
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				mockAuthContext.EXPECT().GetAuthContext(ctx).Return(&middleware.AuthPayload{
					Payload: &utils.SignInTokenPayload{
						UserID: userID.String(),
					},
				}, nil)

				return mockAuthContext, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotResp *entities.ListInterviewSessionsByUserIDResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
				assert.Equal(t, app_error.ErrCodeGeneralInvalidPaginationType, gotErr.(*app_error.AppError).Code)
				assert.Contains(t, gotErr.Error(), "[INS0116]")
				assert.Contains(t, gotErr.Error(), "The pagination type is invalid. Please try again.")
			},
		},
		{
			name: "Error - InvalidCursorID",
			input: &entities.ListInterviewSessionsByUserIDWithCursorReq{
				Cursor: invalidCursorID,
			},
			setup: func() (*mockMiddleware.MockAuthContext, *mockRepositories.MockInterviewSessionRepository) {
				mockAuthContext := mockMiddleware.NewMockAuthContext(t)
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				mockAuthContext.EXPECT().GetAuthContext(ctx).Return(&middleware.AuthPayload{
					Payload: &utils.SignInTokenPayload{
						UserID: userID.String(),
					},
				}, nil)

				mockInterviewSessionRepo.EXPECT().CountInterviewSessionsByUserID(ctx, mock.MatchedBy(func(req *db.CountInterviewSessionsByUserIDParams) bool {
					return req.UserID == userID && req.Column2 == ""
				})).Return(int64(0), nil)

				return mockAuthContext, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotResp *entities.ListInterviewSessionsByUserIDResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
				if appErr, ok := gotErr.(*app_error.AppError); ok {
					assert.Equal(t, app_error.ErrCodeGeneralInvalidUUID, appErr.Code)
				} else {
					t.Fatalf("Expected AppError, got %T", gotErr)
				}
			},
		},
		{
			name: "Error - InvalidCursorTime",
			input: &entities.ListInterviewSessionsByUserIDWithCursorReq{
				Cursor: invalidCursorTime,
			},
			setup: func() (*mockMiddleware.MockAuthContext, *mockRepositories.MockInterviewSessionRepository) {
				mockAuthContext := mockMiddleware.NewMockAuthContext(t)
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				mockAuthContext.EXPECT().GetAuthContext(ctx).Return(&middleware.AuthPayload{
					Payload: &utils.SignInTokenPayload{
						UserID: userID.String(),
					},
				}, nil)

				mockInterviewSessionRepo.EXPECT().CountInterviewSessionsByUserID(ctx, mock.MatchedBy(func(req *db.CountInterviewSessionsByUserIDParams) bool {
					return req.UserID == userID && req.Column2 == ""
				})).Return(int64(0), nil)

				return mockAuthContext, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotResp *entities.ListInterviewSessionsByUserIDResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
				if appErr, ok := gotErr.(*app_error.AppError); ok {
					assert.Equal(t, app_error.ErrCodeGeneralInvalidTime, appErr.Code)
				} else {
					t.Fatalf("Expected AppError, got %T", gotErr)
				}
			},
		},
		{
			name:  "Error - ListInterviewSessionsByUserIDFirstPageError",
			input: &entities.ListInterviewSessionsByUserIDWithCursorReq{},
			setup: func() (*mockMiddleware.MockAuthContext, *mockRepositories.MockInterviewSessionRepository) {
				mockAuthContext := mockMiddleware.NewMockAuthContext(t)
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				mockAuthContext.EXPECT().GetAuthContext(ctx).Return(&middleware.AuthPayload{
					Payload: &utils.SignInTokenPayload{
						UserID: userID.String(),
					},
				}, nil)

				mockInterviewSessionRepo.EXPECT().CountInterviewSessionsByUserID(ctx, mock.MatchedBy(func(req *db.CountInterviewSessionsByUserIDParams) bool {
					return req.UserID == userID && req.Column2 == ""
				})).Return(int64(0), nil)

				mockInterviewSessionRepo.EXPECT().ListInterviewSessionsByUserIDFirstPage(ctx, mock.MatchedBy(func(req *db.ListInterviewSessionsByUserIDFirstPageParams) bool {
					return req.UserID == userID && req.Limit == 20 && req.Column2 == ""
				})).Return(nil, errors.New("database error"))

				return mockAuthContext, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotResp *entities.ListInterviewSessionsByUserIDResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
				assert.Equal(t, "database error", gotErr.Error())
			},
		},
		{
			name: "Error - ListInterviewSessionsByUserIDWithCursorError",
			input: &entities.ListInterviewSessionsByUserIDWithCursorReq{
				Cursor: validCursor,
			},
			setup: func() (*mockMiddleware.MockAuthContext, *mockRepositories.MockInterviewSessionRepository) {
				mockAuthContext := mockMiddleware.NewMockAuthContext(t)
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				mockAuthContext.EXPECT().GetAuthContext(ctx).Return(&middleware.AuthPayload{
					Payload: &utils.SignInTokenPayload{
						UserID: userID.String(),
					},
				}, nil)

				parsedTime, _ := time.Parse(time.RFC3339, validCursor.CreatedAt)
				mockInterviewSessionRepo.EXPECT().ListInterviewSessionsByUserIDWithCursor(ctx, mock.MatchedBy(func(req *db.ListInterviewSessionsByUserIDWithCursorParams) bool {
					return req.UserID == userID && req.Limit == 20 && req.Column2 == "" && req.CreatedAt.Time.Equal(parsedTime) && req.Column4 == "next" && req.ID == uuid.MustParse(validCursor.ID)
				})).
					Return([]db.ListInterviewSessionsByUserIDWithCursorRow{}, errors.New("database error"))

				mockInterviewSessionRepo.EXPECT().CountInterviewSessionsByUserID(ctx, mock.MatchedBy(func(req *db.CountInterviewSessionsByUserIDParams) bool {
					return req.UserID == userID && req.Column2 == ""
				})).Return(int64(0), nil)

				return mockAuthContext, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotResp *entities.ListInterviewSessionsByUserIDResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
				assert.Equal(t, "database error", gotErr.Error())
			},
		},
		{
			name: "Success - AllNilFields",
			input: &entities.ListInterviewSessionsByUserIDWithCursorReq{
				SearchText: nil,
				Cursor:     nil,
				Limit:      nil,
			},
			setup: func() (*mockMiddleware.MockAuthContext, *mockRepositories.MockInterviewSessionRepository) {
				mockAuthContext := mockMiddleware.NewMockAuthContext(t)
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				mockAuthContext.EXPECT().GetAuthContext(ctx).Return(&middleware.AuthPayload{
					Payload: &utils.SignInTokenPayload{
						UserID: userID.String(),
					},
				}, nil)

				mockInterviewSessionRepo.EXPECT().CountInterviewSessionsByUserID(ctx, mock.MatchedBy(func(req *db.CountInterviewSessionsByUserIDParams) bool {
					return req.UserID == userID && req.Column2 == ""
				})).Return(int64(0), nil)

				mockInterviewSessionRepo.EXPECT().ListInterviewSessionsByUserIDFirstPage(ctx, mock.MatchedBy(func(req *db.ListInterviewSessionsByUserIDFirstPageParams) bool {
					return req.UserID == userID && req.Limit == 20 && req.Column2 == ""
				})).Return([]db.ListInterviewSessionsByUserIDFirstPageRow{}, nil)

				return mockAuthContext, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotResp *entities.ListInterviewSessionsByUserIDResp, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, gotResp)
				assert.Len(t, gotResp.Sessions, 0)
				assert.Equal(t, 0, gotResp.TotalPages)
				assert.Equal(t, 20, gotResp.PageSize)
				assert.Nil(t, gotResp.NextCursor)
			},
		},
		{
			name: "Success - ListInterviewSessionsWithPrevPagination",
			input: &entities.ListInterviewSessionsByUserIDWithCursorReq{
				Type:   func() *string { paginationType := "prev"; return &paginationType }(),
				Cursor: validCursor,
			},
			setup: func() (*mockMiddleware.MockAuthContext, *mockRepositories.MockInterviewSessionRepository) {
				mockAuthContext := mockMiddleware.NewMockAuthContext(t)
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				mockAuthContext.EXPECT().GetAuthContext(ctx).Return(&middleware.AuthPayload{
					Payload: &utils.SignInTokenPayload{
						UserID: userID.String(),
					},
				}, nil)

				dbRows := []db.ListInterviewSessionsByUserIDWithCursorRow{
					{
						ID:             uuid.MustParse("550e8400-e29b-41d4-a716-446655440002"),
						ResumeID:       userID,
						ResumeFileName: "test-resume3.pdf",
						Position:       "Senior Engineer",
						Status:         "completed",
						CreatedAt:      sql.NullTime{Time: time.Now().Add(-2 * time.Hour), Valid: true},
						OverallScore:   sql.NullFloat64{Float64: 90.0, Valid: true},
						StartedAt:      sql.NullTime{Time: time.Now().Add(-90 * time.Minute), Valid: true},
						EndedAt:        sql.NullTime{Time: time.Now().Add(-60 * time.Minute), Valid: true},
					},
					{
						ID:             uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"),
						ResumeID:       userID,
						ResumeFileName: "test-resume2.pdf",
						Position:       "Senior Engineer",
						Status:         "completed",
						CreatedAt:      sql.NullTime{Time: time.Now().Add(-time.Hour), Valid: true},
						OverallScore:   sql.NullFloat64{Float64: 90.0, Valid: true},
						StartedAt:      sql.NullTime{Time: time.Now().Add(-90 * time.Minute), Valid: true},
						EndedAt:        sql.NullTime{Time: time.Now().Add(-60 * time.Minute), Valid: true},
					},
					{
						ID:             userID,
						ResumeID:       userID,
						ResumeFileName: "test-resume1.pdf",
						Position:       "Software Engineer",
						Status:         "completed",
						CreatedAt:      sql.NullTime{Time: time.Now(), Valid: true},
						OverallScore:   sql.NullFloat64{Float64: 85.5, Valid: true},
						StartedAt:      sql.NullTime{Time: time.Now().Add(-30 * time.Minute), Valid: true},
						EndedAt:        sql.NullTime{Time: time.Now(), Valid: true},
					},
				}

				parsedTime, _ := time.Parse(time.RFC3339, validCursor.CreatedAt)
				mockInterviewSessionRepo.EXPECT().ListInterviewSessionsByUserIDWithCursor(ctx, mock.MatchedBy(func(req *db.ListInterviewSessionsByUserIDWithCursorParams) bool {
					return req.UserID == userID && req.Limit == 21 && req.Column2 == "" && req.CreatedAt.Time.Equal(parsedTime) && req.Column4 == "prev" && req.ID == uuid.MustParse(validCursor.ID)
				})).
					Return(dbRows, nil)

				mockInterviewSessionRepo.EXPECT().CountInterviewSessionsByUserID(ctx, mock.MatchedBy(func(req *db.CountInterviewSessionsByUserIDParams) bool {
					return req.UserID == userID && req.Column2 == ""
				})).Return(int64(2), nil)

				return mockAuthContext, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotResp *entities.ListInterviewSessionsByUserIDResp, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, gotResp)
				assert.Len(t, gotResp.Sessions, 2)
				assert.Equal(t, 1, gotResp.TotalPages)
				assert.Equal(t, 20, gotResp.PageSize)
				assert.Equal(t, "550e8400-e29b-41d4-a716-446655440002", gotResp.NextCursor.ID)
				assert.Equal(t, "550e8400-e29b-41d4-a716-446655440001", gotResp.PrevCursor.ID)
			},
		},
		{
			name: "Success - ListInterviewSessionsWithCustomPaginationType",
			input: &entities.ListInterviewSessionsByUserIDWithCursorReq{
				Type: func() *string { paginationType := "next"; return &paginationType }(),
			},
			setup: func() (*mockMiddleware.MockAuthContext, *mockRepositories.MockInterviewSessionRepository) {
				mockAuthContext := mockMiddleware.NewMockAuthContext(t)
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				mockAuthContext.EXPECT().GetAuthContext(ctx).Return(&middleware.AuthPayload{
					Payload: &utils.SignInTokenPayload{
						UserID: userID.String(),
					},
				}, nil)

				mockInterviewSessionRepo.EXPECT().ListInterviewSessionsByUserIDFirstPage(ctx, mock.MatchedBy(func(req *db.ListInterviewSessionsByUserIDFirstPageParams) bool {
					return req.UserID == userID && req.Limit == 20 && req.Column2 == ""
				})).
					Return([]db.ListInterviewSessionsByUserIDFirstPageRow{}, nil)

				mockInterviewSessionRepo.EXPECT().CountInterviewSessionsByUserID(ctx, mock.MatchedBy(func(req *db.CountInterviewSessionsByUserIDParams) bool {
					return req.UserID == userID && req.Column2 == ""
				})).Return(int64(0), nil)

				return mockAuthContext, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotResp *entities.ListInterviewSessionsByUserIDResp, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, gotResp)
				assert.Len(t, gotResp.Sessions, 0)
				assert.Equal(t, 0, gotResp.TotalPages)
				assert.Equal(t, 20, gotResp.PageSize)
			},
		},
		{
			name:  "Error - CountInterviewSessionsByUserIDFailure",
			input: &entities.ListInterviewSessionsByUserIDWithCursorReq{},
			setup: func() (*mockMiddleware.MockAuthContext, *mockRepositories.MockInterviewSessionRepository) {
				mockAuthContext := mockMiddleware.NewMockAuthContext(t)
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				mockAuthContext.EXPECT().GetAuthContext(ctx).Return(&middleware.AuthPayload{
					Payload: &utils.SignInTokenPayload{
						UserID: userID.String(),
					},
				}, nil)

				mockInterviewSessionRepo.EXPECT().CountInterviewSessionsByUserID(ctx, mock.MatchedBy(func(req *db.CountInterviewSessionsByUserIDParams) bool {
					return req.UserID == userID && req.Column2 == ""
				})).Return(int64(0), errors.New("count error"))

				return mockAuthContext, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotResp *entities.ListInterviewSessionsByUserIDResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
				assert.Equal(t, "count error", gotErr.Error())
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			mockAuthContext, mockInterviewSessionRepo := tC.setup()

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
				nil,
				nil,
				nil,
				nil,
				nil,
			)

			gotResp, gotErr := svc.ListInterviewSessionsByUserIDWithCursor(ctx, tC.input)
			tC.verify(t, gotResp, gotErr)
		})
	}
}

func TestInterviewSessionService_ListInterviewSessionsByUserIDWithJumpPagination(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()
	userID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	testCases := []struct {
		name   string
		input  *entities.ListInterviewSessionsByUserIDWithJumpPaginationReq
		setup  func() (*mockMiddleware.MockAuthContext, *mockRepositories.MockInterviewSessionRepository)
		verify func(t *testing.T, gotResp *entities.ListInterviewSessionsByUserIDResp, gotErr error)
	}{
		{
			name:  "Success - ListInterviewSessionsWithDefaultParameters",
			input: &entities.ListInterviewSessionsByUserIDWithJumpPaginationReq{},
			setup: func() (*mockMiddleware.MockAuthContext, *mockRepositories.MockInterviewSessionRepository) {
				mockAuthContext := mockMiddleware.NewMockAuthContext(t)
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				mockAuthContext.EXPECT().GetAuthContext(ctx).Return(&middleware.AuthPayload{
					Payload: &utils.SignInTokenPayload{
						UserID: userID.String(),
					},
				}, nil)

				dbRows := []db.ListInterviewSessionsByUserIDWithJumpPaginationRow{
					{
						ID:             userID,
						ResumeID:       userID,
						ResumeFileName: "test-resume.pdf",
						Position:       "Software Engineer",
						Status:         "completed",
						CreatedAt:      sql.NullTime{Time: time.Now(), Valid: true},
						OverallScore:   sql.NullFloat64{Float64: 85.5, Valid: true},
						StartedAt:      sql.NullTime{Time: time.Now().Add(-30 * time.Minute), Valid: true},
						EndedAt:        sql.NullTime{Time: time.Now(), Valid: true},
					},
				}

				mockInterviewSessionRepo.EXPECT().ListInterviewSessionsByUserIDWithJumpPagination(ctx, mock.MatchedBy(func(req *db.ListInterviewSessionsByUserIDWithJumpPaginationParams) bool {
					return req.UserID == userID && req.Limit == 20 && req.Column2 == 1 && req.Column4 == ""
				})).
					Return(dbRows, nil)

				mockInterviewSessionRepo.EXPECT().CountInterviewSessionsByUserID(ctx, mock.MatchedBy(func(req *db.CountInterviewSessionsByUserIDParams) bool {
					return req.UserID == userID && req.Column2 == ""
				})).Return(int64(1), nil)

				return mockAuthContext, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotResp *entities.ListInterviewSessionsByUserIDResp, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, gotResp)
				assert.Len(t, gotResp.Sessions, 1)
				assert.Equal(t, userID.String(), gotResp.Sessions[0].ID)
				assert.Equal(t, "test-resume.pdf", gotResp.Sessions[0].ResumeFileName)
				assert.Equal(t, "Software Engineer", gotResp.Sessions[0].Position)
				assert.Equal(t, "completed", gotResp.Sessions[0].Status)
				assert.Equal(t, 85.5, gotResp.Sessions[0].OverallScore)
				assert.Equal(t, "30.00", gotResp.Sessions[0].TotalTime)
				assert.Equal(t, 1, gotResp.TotalPages)
				assert.Equal(t, 20, gotResp.PageSize)
				assert.NotNil(t, gotResp.NextCursor)
				assert.Nil(t, gotResp.PrevCursor) // No offset means no prev cursor
			},
		},
		{
			name: "Success - WithCustomLimit",
			input: &entities.ListInterviewSessionsByUserIDWithJumpPaginationReq{
				Limit: func() *int { limit := 10; return &limit }(),
			},
			setup: func() (*mockMiddleware.MockAuthContext, *mockRepositories.MockInterviewSessionRepository) {
				mockAuthContext := mockMiddleware.NewMockAuthContext(t)
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				mockAuthContext.EXPECT().GetAuthContext(ctx).Return(&middleware.AuthPayload{
					Payload: &utils.SignInTokenPayload{
						UserID: userID.String(),
					},
				}, nil)

				dbRows := []db.ListInterviewSessionsByUserIDWithJumpPaginationRow{
					{
						ID:             userID,
						ResumeID:       userID,
						ResumeFileName: "test-resume.pdf",
						Position:       "Software Engineer",
						Status:         "completed",
						CreatedAt:      sql.NullTime{Time: time.Now(), Valid: true},
						OverallScore:   sql.NullFloat64{Float64: 85.5, Valid: true},
						StartedAt:      sql.NullTime{Time: time.Now().Add(-30 * time.Minute), Valid: true},
						EndedAt:        sql.NullTime{Time: time.Now(), Valid: true},
					},
				}

				mockInterviewSessionRepo.EXPECT().ListInterviewSessionsByUserIDWithJumpPagination(ctx, mock.MatchedBy(func(req *db.ListInterviewSessionsByUserIDWithJumpPaginationParams) bool {
					return req.UserID == userID && req.Limit == 10 && req.Column2 == 1 && req.Column4 == ""
				})).
					Return(dbRows, nil)

				mockInterviewSessionRepo.EXPECT().CountInterviewSessionsByUserID(ctx, mock.MatchedBy(func(req *db.CountInterviewSessionsByUserIDParams) bool {
					return req.UserID == userID && req.Column2 == ""
				})).Return(int64(1), nil)

				return mockAuthContext, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotResp *entities.ListInterviewSessionsByUserIDResp, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, gotResp)
				assert.Len(t, gotResp.Sessions, 1)
				assert.Equal(t, 1, gotResp.TotalPages)
				assert.Equal(t, 10, gotResp.PageSize)
				assert.NotNil(t, gotResp.NextCursor)
				assert.Nil(t, gotResp.PrevCursor)
			},
		},
		{
			name: "Success - WithOffset",
			input: &entities.ListInterviewSessionsByUserIDWithJumpPaginationReq{
				Offset: func() *int { offset := 2; return &offset }(),
				Limit:  func() *int { limit := 5; return &limit }(),
			},
			setup: func() (*mockMiddleware.MockAuthContext, *mockRepositories.MockInterviewSessionRepository) {
				mockAuthContext := mockMiddleware.NewMockAuthContext(t)
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				mockAuthContext.EXPECT().GetAuthContext(ctx).Return(&middleware.AuthPayload{
					Payload: &utils.SignInTokenPayload{
						UserID: userID.String(),
					},
				}, nil)

				dbRows := []db.ListInterviewSessionsByUserIDWithJumpPaginationRow{
					{
						ID:             userID,
						ResumeID:       userID,
						ResumeFileName: "test-resume.pdf",
						Position:       "Software Engineer",
						Status:         "completed",
						CreatedAt:      sql.NullTime{Time: time.Now(), Valid: true},
						OverallScore:   sql.NullFloat64{Float64: 85.5, Valid: true},
						StartedAt:      sql.NullTime{Time: time.Now().Add(-30 * time.Minute), Valid: true},
						EndedAt:        sql.NullTime{Time: time.Now(), Valid: true},
					},
				}

				mockInterviewSessionRepo.EXPECT().ListInterviewSessionsByUserIDWithJumpPagination(ctx, mock.MatchedBy(func(req *db.ListInterviewSessionsByUserIDWithJumpPaginationParams) bool {
					return req.UserID == userID && req.Limit == 5 && req.Column2 == 2 && req.Column4 == ""
				})).
					Return(dbRows, nil)

				mockInterviewSessionRepo.EXPECT().CountInterviewSessionsByUserID(ctx, mock.MatchedBy(func(req *db.CountInterviewSessionsByUserIDParams) bool {
					return req.UserID == userID && req.Column2 == ""
				})).Return(int64(10), nil)

				return mockAuthContext, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotResp *entities.ListInterviewSessionsByUserIDResp, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, gotResp)
				assert.Len(t, gotResp.Sessions, 1)
				assert.Equal(t, 2, gotResp.TotalPages)
				assert.Equal(t, 5, gotResp.PageSize)
				assert.NotNil(t, gotResp.NextCursor)
				assert.NotNil(t, gotResp.PrevCursor) // With offset, prev cursor should be present
			},
		},
		{
			name: "Success - WithSearchText",
			input: &entities.ListInterviewSessionsByUserIDWithJumpPaginationReq{
				SearchText: func() *string { search := "engineer"; return &search }(),
			},
			setup: func() (*mockMiddleware.MockAuthContext, *mockRepositories.MockInterviewSessionRepository) {
				mockAuthContext := mockMiddleware.NewMockAuthContext(t)
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				mockAuthContext.EXPECT().GetAuthContext(ctx).Return(&middleware.AuthPayload{
					Payload: &utils.SignInTokenPayload{
						UserID: userID.String(),
					},
				}, nil)

				dbRows := []db.ListInterviewSessionsByUserIDWithJumpPaginationRow{
					{
						ID:             userID,
						ResumeID:       userID,
						ResumeFileName: "test-resume.pdf",
						Position:       "Software Engineer",
						Status:         "completed",
						CreatedAt:      sql.NullTime{Time: time.Now(), Valid: true},
						OverallScore:   sql.NullFloat64{Float64: 85.5, Valid: true},
						StartedAt:      sql.NullTime{Time: time.Now().Add(-30 * time.Minute), Valid: true},
						EndedAt:        sql.NullTime{Time: time.Now(), Valid: true},
					},
				}

				mockInterviewSessionRepo.EXPECT().ListInterviewSessionsByUserIDWithJumpPagination(ctx, mock.MatchedBy(func(req *db.ListInterviewSessionsByUserIDWithJumpPaginationParams) bool {
					return req.UserID == userID && req.Limit == 20 && req.Column2 == 1 && req.Column4 == "engineer"
				})).
					Return(dbRows, nil)

				mockInterviewSessionRepo.EXPECT().CountInterviewSessionsByUserID(ctx, mock.MatchedBy(func(req *db.CountInterviewSessionsByUserIDParams) bool {
					return req.UserID == userID && req.Column2 == "engineer"
				})).Return(int64(1), nil)

				return mockAuthContext, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotResp *entities.ListInterviewSessionsByUserIDResp, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, gotResp)
				assert.Len(t, gotResp.Sessions, 1)
				assert.Equal(t, "Software Engineer", gotResp.Sessions[0].Position)
				assert.Equal(t, 1, gotResp.TotalPages)
				assert.Equal(t, 20, gotResp.PageSize)
			},
		},
		{
			name:  "Success - EmptyResults",
			input: &entities.ListInterviewSessionsByUserIDWithJumpPaginationReq{},
			setup: func() (*mockMiddleware.MockAuthContext, *mockRepositories.MockInterviewSessionRepository) {
				mockAuthContext := mockMiddleware.NewMockAuthContext(t)
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				mockAuthContext.EXPECT().GetAuthContext(ctx).Return(&middleware.AuthPayload{
					Payload: &utils.SignInTokenPayload{
						UserID: userID.String(),
					},
				}, nil)

				mockInterviewSessionRepo.EXPECT().ListInterviewSessionsByUserIDWithJumpPagination(ctx, mock.MatchedBy(func(req *db.ListInterviewSessionsByUserIDWithJumpPaginationParams) bool {
					return req.UserID == userID && req.Limit == 20 && req.Column2 == 1 && req.Column4 == ""
				})).
					Return([]db.ListInterviewSessionsByUserIDWithJumpPaginationRow{}, nil)

				mockInterviewSessionRepo.EXPECT().CountInterviewSessionsByUserID(ctx, mock.MatchedBy(func(req *db.CountInterviewSessionsByUserIDParams) bool {
					return req.UserID == userID && req.Column2 == ""
				})).Return(int64(0), nil)

				return mockAuthContext, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotResp *entities.ListInterviewSessionsByUserIDResp, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, gotResp)
				assert.Len(t, gotResp.Sessions, 0)
				assert.Equal(t, 0, gotResp.TotalPages)
				assert.Equal(t, 20, gotResp.PageSize)
				assert.Nil(t, gotResp.NextCursor)
				assert.Nil(t, gotResp.PrevCursor)
			},
		},
		{
			name:  "Error - AuthContextError",
			input: &entities.ListInterviewSessionsByUserIDWithJumpPaginationReq{},
			setup: func() (*mockMiddleware.MockAuthContext, *mockRepositories.MockInterviewSessionRepository) {
				mockAuthContext := mockMiddleware.NewMockAuthContext(t)
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				mockAuthContext.EXPECT().GetAuthContext(ctx).Return(nil, errors.New("auth context error"))

				return mockAuthContext, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotResp *entities.ListInterviewSessionsByUserIDResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
				assert.Contains(t, gotErr.Error(), "auth context error")
			},
		},
		{
			name:  "Error - InvalidUserID",
			input: &entities.ListInterviewSessionsByUserIDWithJumpPaginationReq{},
			setup: func() (*mockMiddleware.MockAuthContext, *mockRepositories.MockInterviewSessionRepository) {
				mockAuthContext := mockMiddleware.NewMockAuthContext(t)
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				mockAuthContext.EXPECT().GetAuthContext(ctx).Return(&middleware.AuthPayload{
					Payload: &utils.SignInTokenPayload{
						UserID: "invalid-uuid",
					},
				}, nil)

				return mockAuthContext, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotResp *entities.ListInterviewSessionsByUserIDResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
				assert.Contains(t, gotErr.Error(), "invalid UUID")
			},
		},
		{
			name:  "Error - CountInterviewSessionsError",
			input: &entities.ListInterviewSessionsByUserIDWithJumpPaginationReq{},
			setup: func() (*mockMiddleware.MockAuthContext, *mockRepositories.MockInterviewSessionRepository) {
				mockAuthContext := mockMiddleware.NewMockAuthContext(t)
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				mockAuthContext.EXPECT().GetAuthContext(ctx).Return(&middleware.AuthPayload{
					Payload: &utils.SignInTokenPayload{
						UserID: userID.String(),
					},
				}, nil)

				mockInterviewSessionRepo.EXPECT().CountInterviewSessionsByUserID(ctx, mock.MatchedBy(func(req *db.CountInterviewSessionsByUserIDParams) bool {
					return req.UserID == userID && req.Column2 == ""
				})).Return(int64(0), errors.New("database error"))

				return mockAuthContext, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotResp *entities.ListInterviewSessionsByUserIDResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
				assert.Contains(t, gotErr.Error(), "database error")
			},
		},
		{
			name:  "Error - ListInterviewSessionsError",
			input: &entities.ListInterviewSessionsByUserIDWithJumpPaginationReq{},
			setup: func() (*mockMiddleware.MockAuthContext, *mockRepositories.MockInterviewSessionRepository) {
				mockAuthContext := mockMiddleware.NewMockAuthContext(t)
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				mockAuthContext.EXPECT().GetAuthContext(ctx).Return(&middleware.AuthPayload{
					Payload: &utils.SignInTokenPayload{
						UserID: userID.String(),
					},
				}, nil)

				mockInterviewSessionRepo.EXPECT().CountInterviewSessionsByUserID(ctx, mock.MatchedBy(func(req *db.CountInterviewSessionsByUserIDParams) bool {
					return req.UserID == userID && req.Column2 == ""
				})).Return(int64(1), nil)

				mockInterviewSessionRepo.EXPECT().ListInterviewSessionsByUserIDWithJumpPagination(ctx, mock.MatchedBy(func(req *db.ListInterviewSessionsByUserIDWithJumpPaginationParams) bool {
					return req.UserID == userID && req.Limit == 20 && req.Column2 == 1 && req.Column4 == ""
				})).
					Return(nil, errors.New("database query error"))

				return mockAuthContext, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotResp *entities.ListInterviewSessionsByUserIDResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
				assert.Contains(t, gotErr.Error(), "database query error")
			},
		},
		{
			name:  "Success - DataTransformationWithNullValues",
			input: &entities.ListInterviewSessionsByUserIDWithJumpPaginationReq{},
			setup: func() (*mockMiddleware.MockAuthContext, *mockRepositories.MockInterviewSessionRepository) {
				mockAuthContext := mockMiddleware.NewMockAuthContext(t)
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				mockAuthContext.EXPECT().GetAuthContext(ctx).Return(&middleware.AuthPayload{
					Payload: &utils.SignInTokenPayload{
						UserID: userID.String(),
					},
				}, nil)

				dbRows := []db.ListInterviewSessionsByUserIDWithJumpPaginationRow{
					{
						ID:             userID,
						ResumeID:       userID,
						ResumeFileName: "test-resume.pdf",
						Position:       "Software Engineer",
						Status:         "completed",
						CreatedAt:      sql.NullTime{Time: time.Now(), Valid: true},
						OverallScore:   sql.NullFloat64{Valid: false}, // Null score
						StartedAt:      sql.NullTime{Valid: false},    // Null started_at
						EndedAt:        sql.NullTime{Valid: false},    // Null ended_at
					},
				}

				mockInterviewSessionRepo.EXPECT().ListInterviewSessionsByUserIDWithJumpPagination(ctx, mock.MatchedBy(func(req *db.ListInterviewSessionsByUserIDWithJumpPaginationParams) bool {
					return req.UserID == userID && req.Limit == 20 && req.Column2 == 1 && req.Column4 == ""
				})).
					Return(dbRows, nil)

				mockInterviewSessionRepo.EXPECT().CountInterviewSessionsByUserID(ctx, mock.MatchedBy(func(req *db.CountInterviewSessionsByUserIDParams) bool {
					return req.UserID == userID && req.Column2 == ""
				})).Return(int64(1), nil)

				return mockAuthContext, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotResp *entities.ListInterviewSessionsByUserIDResp, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, gotResp)
				assert.Len(t, gotResp.Sessions, 1)
				assert.Equal(t, 0.0, gotResp.Sessions[0].OverallScore) // Should be 0.0 for null score
				assert.Equal(t, "0.00", gotResp.Sessions[0].TotalTime) // Should be "0.00" for null times
				assert.Equal(t, 1, gotResp.TotalPages)
				assert.Equal(t, 20, gotResp.PageSize)
			},
		},
		{
			name:  "Success - DataTransformationWithPartialTimes",
			input: &entities.ListInterviewSessionsByUserIDWithJumpPaginationReq{},
			setup: func() (*mockMiddleware.MockAuthContext, *mockRepositories.MockInterviewSessionRepository) {
				mockAuthContext := mockMiddleware.NewMockAuthContext(t)
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				mockAuthContext.EXPECT().GetAuthContext(ctx).Return(&middleware.AuthPayload{
					Payload: &utils.SignInTokenPayload{
						UserID: userID.String(),
					},
				}, nil)

				dbRows := []db.ListInterviewSessionsByUserIDWithJumpPaginationRow{
					{
						ID:             userID,
						ResumeID:       userID,
						ResumeFileName: "test-resume.pdf",
						Position:       "Software Engineer",
						Status:         "completed",
						CreatedAt:      sql.NullTime{Time: time.Now(), Valid: true},
						OverallScore:   sql.NullFloat64{Float64: 75.0, Valid: true},
						StartedAt:      sql.NullTime{Time: time.Now().Add(-45 * time.Minute), Valid: true},
						EndedAt:        sql.NullTime{Valid: false}, // Only started_at is valid
					},
				}

				mockInterviewSessionRepo.EXPECT().ListInterviewSessionsByUserIDWithJumpPagination(ctx, mock.MatchedBy(func(req *db.ListInterviewSessionsByUserIDWithJumpPaginationParams) bool {
					return req.UserID == userID && req.Limit == 20 && req.Column2 == 1 && req.Column4 == ""
				})).
					Return(dbRows, nil)

				mockInterviewSessionRepo.EXPECT().CountInterviewSessionsByUserID(ctx, mock.MatchedBy(func(req *db.CountInterviewSessionsByUserIDParams) bool {
					return req.UserID == userID && req.Column2 == ""
				})).Return(int64(1), nil)

				return mockAuthContext, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotResp *entities.ListInterviewSessionsByUserIDResp, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, gotResp)
				assert.Len(t, gotResp.Sessions, 1)
				assert.Equal(t, 75.0, gotResp.Sessions[0].OverallScore)
				assert.Equal(t, "0.00", gotResp.Sessions[0].TotalTime) // Should be "0.00" when ended_at is null
				assert.Equal(t, 1, gotResp.TotalPages)
				assert.Equal(t, 20, gotResp.PageSize)
			},
		},
		{
			name: "Success - MultiplePages",
			input: &entities.ListInterviewSessionsByUserIDWithJumpPaginationReq{
				Limit: func() *int { limit := 5; return &limit }(),
			},
			setup: func() (*mockMiddleware.MockAuthContext, *mockRepositories.MockInterviewSessionRepository) {
				mockAuthContext := mockMiddleware.NewMockAuthContext(t)
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				mockAuthContext.EXPECT().GetAuthContext(ctx).Return(&middleware.AuthPayload{
					Payload: &utils.SignInTokenPayload{
						UserID: userID.String(),
					},
				}, nil)

				dbRows := []db.ListInterviewSessionsByUserIDWithJumpPaginationRow{
					{
						ID:             userID,
						ResumeID:       userID,
						ResumeFileName: "test-resume.pdf",
						Position:       "Software Engineer",
						Status:         "completed",
						CreatedAt:      sql.NullTime{Time: time.Now(), Valid: true},
						OverallScore:   sql.NullFloat64{Float64: 85.5, Valid: true},
						StartedAt:      sql.NullTime{Time: time.Now().Add(-30 * time.Minute), Valid: true},
						EndedAt:        sql.NullTime{Time: time.Now(), Valid: true},
					},
				}

				mockInterviewSessionRepo.EXPECT().ListInterviewSessionsByUserIDWithJumpPagination(ctx, mock.MatchedBy(func(req *db.ListInterviewSessionsByUserIDWithJumpPaginationParams) bool {
					return req.UserID == userID && req.Limit == 5 && req.Column2 == 1 && req.Column4 == ""
				})).
					Return(dbRows, nil)

				mockInterviewSessionRepo.EXPECT().CountInterviewSessionsByUserID(ctx, mock.MatchedBy(func(req *db.CountInterviewSessionsByUserIDParams) bool {
					return req.UserID == userID && req.Column2 == ""
				})).Return(int64(12), nil)

				return mockAuthContext, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotResp *entities.ListInterviewSessionsByUserIDResp, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, gotResp)
				assert.Len(t, gotResp.Sessions, 1)
				assert.Equal(t, 3, gotResp.TotalPages) // 12 items with page size 5 = 3 pages
				assert.Equal(t, 5, gotResp.PageSize)
				assert.NotNil(t, gotResp.NextCursor)
				assert.Nil(t, gotResp.PrevCursor)
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			mockAuthContext, mockInterviewSessionRepo := tC.setup()

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
				nil,
				nil,
				nil,
				nil,
				nil,
			)

			gotResp, gotErr := svc.ListInterviewSessionsByUserIDWithJumpPagination(ctx, tC.input)

			tC.verify(t, gotResp, gotErr)
		})
	}
}

func TestInterviewSessionService_ListFinalizingInterviewSessionByUserID(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()
	userID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	testCases := []struct {
		name   string
		setup  func() (*mockMiddleware.MockAuthContext, *mockRepositories.MockInterviewSessionRepository)
		verify func(t *testing.T, gotResp *entities.ListFinalizingInterviewSessionByUserIDResp, gotErr error)
	}{
		{
			name: "Success",
			setup: func() (*mockMiddleware.MockAuthContext, *mockRepositories.MockInterviewSessionRepository) {
				mockAuthContext := mockMiddleware.NewMockAuthContext(t)
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				mockAuthContext.EXPECT().GetAuthContext(ctx).Return(&middleware.AuthPayload{
					Payload: &utils.SignInTokenPayload{
						UserID: userID.String(),
					},
				}, nil)

				dbRows := []db.ListFinalizingInterviewSessionByUserIDRow{
					{
						ID:             userID,
						ResumeID:       userID,
						ResumeFileName: "test-resume.pdf",
						Position:       "Software Engineer",
						Status:         "completed",
						CreatedAt:      sql.NullTime{Time: time.Now(), Valid: true},
						OverallScore:   sql.NullFloat64{Float64: 85.5, Valid: true},
						StartedAt:      sql.NullTime{Time: time.Now().Add(-30 * time.Minute), Valid: true},
						EndedAt:        sql.NullTime{Time: time.Now(), Valid: true},
					},
				}

				mockInterviewSessionRepo.EXPECT().ListFinalizingInterviewSessionByUserID(ctx, mock.MatchedBy(func(req uuid.UUID) bool {
					return req == userID
				})).
					Return(dbRows, nil)

				return mockAuthContext, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotResp *entities.ListFinalizingInterviewSessionByUserIDResp, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, gotResp)
				assert.Len(t, gotResp.Sessions, 1)
				assert.Equal(t, userID.String(), gotResp.Sessions[0].ID)
				assert.Equal(t, "test-resume.pdf", gotResp.Sessions[0].ResumeFileName)
				assert.Equal(t, "Software Engineer", gotResp.Sessions[0].Position)
				assert.Equal(t, "completed", gotResp.Sessions[0].Status)
				assert.Equal(t, 1, gotResp.TotalCount)
			},
		},
		{
			name: "Error WithGetAuthContextError",
			setup: func() (*mockMiddleware.MockAuthContext, *mockRepositories.MockInterviewSessionRepository) {
				mockAuthContext := mockMiddleware.NewMockAuthContext(t)
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				mockAuthContext.EXPECT().GetAuthContext(ctx).Return(nil, errors.New("auth context error"))

				return mockAuthContext, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotResp *entities.ListFinalizingInterviewSessionByUserIDResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "auth context error")
				assert.Nil(t, gotResp)
			},
		},
		{
			name: "Error WithInvalidUserID",
			setup: func() (*mockMiddleware.MockAuthContext, *mockRepositories.MockInterviewSessionRepository) {
				mockAuthContext := mockMiddleware.NewMockAuthContext(t)
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				mockAuthContext.EXPECT().GetAuthContext(ctx).Return(&middleware.AuthPayload{
					Payload: &utils.SignInTokenPayload{
						UserID: "invalid-user-id",
					},
				}, nil)

				return mockAuthContext, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotResp *entities.ListFinalizingInterviewSessionByUserIDResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "The UUID is invalid. Please try again.")
				assert.Contains(t, gotErr.Error(), "[INS0107]")
				assert.Nil(t, gotResp)
			},
		},
		{
			name: "Error WithListFinalizingInterviewSessionByUserIDError",
			setup: func() (*mockMiddleware.MockAuthContext, *mockRepositories.MockInterviewSessionRepository) {
				mockAuthContext := mockMiddleware.NewMockAuthContext(t)
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				mockAuthContext.EXPECT().GetAuthContext(ctx).Return(&middleware.AuthPayload{
					Payload: &utils.SignInTokenPayload{
						UserID: userID.String(),
					},
				}, nil)

				mockInterviewSessionRepo.EXPECT().ListFinalizingInterviewSessionByUserID(ctx, mock.MatchedBy(func(req uuid.UUID) bool {
					return req == userID
				})).
					Return(nil, errors.New("error"))

				return mockAuthContext, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotResp *entities.ListFinalizingInterviewSessionByUserIDResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "error")
				assert.Nil(t, gotResp)
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			mockAuthContext, mockInterviewSessionRepo := tC.setup()

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
				nil,
				nil,
				nil,
				nil,
				nil,
			)

			gotResp, gotErr := svc.ListFinalizingInterviewSessionByUserID(ctx)

			tC.verify(t, gotResp, gotErr)
		})
	}
}

func TestInterviewSessionService_DeleteUserInterviewSessionByID(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()
	sessionID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	userID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	invalidSessionID := "invalid-session-id"
	invalidUserID := "invalid-user-id"

	testCases := []struct {
		name   string
		input  string
		setup  func() (*mockMiddleware.MockAuthContext, *mockRepositories.MockInterviewSessionRepository)
		verify func(t *testing.T, gotErr error)
	}{
		{
			name:  "Success",
			input: sessionID.String(),
			setup: func() (*mockMiddleware.MockAuthContext, *mockRepositories.MockInterviewSessionRepository) {
				mockAuthContext := mockMiddleware.NewMockAuthContext(t)
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				mockAuthContext.EXPECT().GetAuthContext(ctx).Return(&middleware.AuthPayload{
					Payload: &utils.SignInTokenPayload{
						UserID: userID.String(),
					},
				}, nil)

				mockInterviewSessionRepo.EXPECT().DeleteUserInterviewSessionByID(ctx, mock.MatchedBy(func(req *db.DeleteUserInterviewSessionByIDParams) bool {
					return req.UserID == userID && req.ID == sessionID
				})).
					Return(nil)

				return mockAuthContext, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotErr error) {
				assert.NoError(t, gotErr)
			},
		},
		{
			name:  "Error WithInvalidSessionID",
			input: invalidSessionID,
			setup: func() (*mockMiddleware.MockAuthContext, *mockRepositories.MockInterviewSessionRepository) {
				mockAuthContext := mockMiddleware.NewMockAuthContext(t)
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				return mockAuthContext, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "The UUID is invalid. Please try again.")
				assert.Contains(t, gotErr.Error(), "[INS0107]")
			},
		},
		{
			name:  "Error WithGetAuthContextError",
			input: sessionID.String(),
			setup: func() (*mockMiddleware.MockAuthContext, *mockRepositories.MockInterviewSessionRepository) {
				mockAuthContext := mockMiddleware.NewMockAuthContext(t)
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				mockAuthContext.EXPECT().GetAuthContext(ctx).
					Return(nil, errors.New("auth context error"))

				return mockAuthContext, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "auth context error")
			},
		},
		{
			name:  "Error WithInvalidUserID",
			input: sessionID.String(),
			setup: func() (*mockMiddleware.MockAuthContext, *mockRepositories.MockInterviewSessionRepository) {
				mockAuthContext := mockMiddleware.NewMockAuthContext(t)
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				mockAuthContext.EXPECT().GetAuthContext(ctx).Return(&middleware.AuthPayload{
					Payload: &utils.SignInTokenPayload{
						UserID: invalidUserID,
					},
				}, nil)

				return mockAuthContext, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "The UUID is invalid. Please try again.")
				assert.Contains(t, gotErr.Error(), "[INS0107]")
			},
		},
		{
			name:  "Error WithDeleteUserInterviewSessionByIDError",
			input: sessionID.String(),
			setup: func() (*mockMiddleware.MockAuthContext, *mockRepositories.MockInterviewSessionRepository) {
				mockAuthContext := mockMiddleware.NewMockAuthContext(t)
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				mockAuthContext.EXPECT().GetAuthContext(ctx).Return(&middleware.AuthPayload{
					Payload: &utils.SignInTokenPayload{
						UserID: userID.String(),
					},
				}, nil)

				mockInterviewSessionRepo.EXPECT().DeleteUserInterviewSessionByID(ctx, mock.MatchedBy(func(req *db.DeleteUserInterviewSessionByIDParams) bool {
					return req.UserID == userID && req.ID == sessionID
				})).
					Return(errors.New("error"))

				return mockAuthContext, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "error")
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			mockAuthContext, mockInterviewSessionRepo := tC.setup()

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
				nil,
				nil,
				nil,
				nil,
				nil,
			)

			gotErr := svc.DeleteUserInterviewSessionByID(ctx, tC.input)

			tC.verify(t, gotErr)
		})
	}
}

func TestInterviewSessionService_GetInterviewSessionInformationByID(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()
	sessionID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	userID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	dateNow := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC).In(time.UTC)
	invalidSessionID := "invalid-session-id"
	invalidUserID := "invalid-user-id"
	status := constants.StatusCompleted
	statusColor := "#28A745"
	statusDisplayName := "Completed"

	validDbResp := &db.GetInterviewSessionInformationByIDRow{
		UserID:         userID,
		ResumeID:       sessionID,
		ResumeFileName: "resume_file_name",
		Position:       "position",
		Status:         status,
		StartedAt:      sql.NullTime{Time: dateNow, Valid: true},
		EndedAt:        sql.NullTime{Time: dateNow, Valid: true},
		OverallScore:   sql.NullFloat64{Float64: 5, Valid: true},
		SummaryMd:      sql.NullString{String: "summary_md", Valid: true},
		CreatedAt:      sql.NullTime{Time: dateNow, Valid: true},
	}

	validResp := &entities.GetInterviewSessionInformationResp{
		ResumeID:            sessionID,
		ResumeFileName:      "resume_file_name",
		Position:            "position",
		Status:              status,
		StatusDisplayName:   statusDisplayName,
		StatusColor:         statusColor,
		TotalTime:           "",
		StartedAt:           utils.FormatNullableTimeToBangkokString(sql.NullTime{Time: dateNow, Valid: true}),
		EndedAt:             utils.FormatNullableTimeToBangkokString(sql.NullTime{Time: dateNow, Valid: true}),
		OverallScore:        5,
		OverallScorePercent: "100.0%",
		OverallScoreColor:   "#28A745",
		SummaryMd:           "summary_md",
		CreatedAt:           utils.FormatNullableTimeToBangkokString(sql.NullTime{Time: dateNow, Valid: true}),
		CreatedAtFullName:   utils.FormatNullableTimeToBangkokStringFullTimeFormat(sql.NullTime{Time: dateNow, Valid: true}),
	}

	validDbRespInvalidOverallScore := *validDbResp
	validDbRespInvalidOverallScore.OverallScore = sql.NullFloat64{Float64: 10, Valid: true}

	validDbRespInvalidStatus := *validDbResp
	validDbRespInvalidStatus.Status = "invalid"

	validDbUserRespNotMatch := *validDbResp
	validDbUserRespNotMatch.UserID = uuid.MustParse("550e8400-e29b-41d4-a716-446655440001")

	validDbRespWithNullSomeValue := &db.GetInterviewSessionInformationByIDRow{
		UserID:         userID,
		ResumeID:       sessionID,
		ResumeFileName: "resume_file_name",
		Position:       "position",
		Status:         status,
		StartedAt:      sql.NullTime{Time: dateNow, Valid: false},
		EndedAt:        sql.NullTime{Time: dateNow, Valid: false},
		OverallScore:   sql.NullFloat64{Float64: 5, Valid: false},
		SummaryMd:      sql.NullString{String: "summary_md", Valid: false},
		CreatedAt:      sql.NullTime{Time: dateNow, Valid: false},
	}

	validRespWithNullSomeValue := &entities.GetInterviewSessionInformationResp{
		ResumeID:            sessionID,
		ResumeFileName:      "resume_file_name",
		Position:            "position",
		Status:              status,
		StatusDisplayName:   statusDisplayName,
		StatusColor:         statusColor,
		TotalTime:           "", // Empty because StartedAt and EndedAt are invalid
		StartedAt:           utils.FormatNullableTimeToBangkokString(sql.NullTime{Time: dateNow, Valid: false}),
		EndedAt:             utils.FormatNullableTimeToBangkokString(sql.NullTime{Time: dateNow, Valid: false}),
		OverallScore:        0.00,
		OverallScorePercent: "0.0%",
		OverallScoreColor:   "#DC3545",
		SummaryMd:           constants.BlankOverallSummaryMd,
		CreatedAt:           utils.FormatNullableTimeToBangkokString(sql.NullTime{Time: dateNow, Valid: false}),
		CreatedAtFullName:   utils.FormatNullableTimeToBangkokStringFullTimeFormat(sql.NullTime{Time: dateNow, Valid: false}),
	}

	testCases := []struct {
		name   string
		input  string
		setup  func() (*mockMiddleware.MockAuthContext, *mockRepositories.MockInterviewSessionRepository)
		verify func(t *testing.T, gotResp *entities.GetInterviewSessionInformationResp, gotErr error)
	}{
		{
			name:  "Success",
			input: sessionID.String(),
			setup: func() (*mockMiddleware.MockAuthContext, *mockRepositories.MockInterviewSessionRepository) {
				mockAuthContext := mockMiddleware.NewMockAuthContext(t)
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				mockAuthContext.EXPECT().GetAuthContext(ctx).Return(&middleware.AuthPayload{
					Payload: &utils.SignInTokenPayload{
						UserID: userID.String(),
					},
				}, nil)

				mockInterviewSessionRepo.EXPECT().GetInterviewSessionInformationByID(ctx, sessionID).
					Return(validDbResp, nil)

				return mockAuthContext, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotResp *entities.GetInterviewSessionInformationResp, gotErr error) {
				assert.NoError(t, gotErr)
				assert.Equal(t, validResp, gotResp)
			},
		},
		{
			name:  "Success WithNullSomeValue",
			input: sessionID.String(),
			setup: func() (*mockMiddleware.MockAuthContext, *mockRepositories.MockInterviewSessionRepository) {
				mockAuthContext := mockMiddleware.NewMockAuthContext(t)
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				mockAuthContext.EXPECT().GetAuthContext(ctx).Return(&middleware.AuthPayload{
					Payload: &utils.SignInTokenPayload{
						UserID: userID.String(),
					},
				}, nil)

				mockInterviewSessionRepo.EXPECT().GetInterviewSessionInformationByID(ctx, sessionID).
					Return(validDbRespWithNullSomeValue, nil)

				return mockAuthContext, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotResp *entities.GetInterviewSessionInformationResp, gotErr error) {
				assert.NoError(t, gotErr)
				assert.Equal(t, validRespWithNullSomeValue, gotResp)
			},
		},
		{
			name:  "Error WithInvalidSessionID",
			input: invalidSessionID,
			setup: func() (*mockMiddleware.MockAuthContext, *mockRepositories.MockInterviewSessionRepository) {
				mockAuthContext := mockMiddleware.NewMockAuthContext(t)
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				return mockAuthContext, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotResp *entities.GetInterviewSessionInformationResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "The UUID is invalid. Please try again.")
				assert.Contains(t, gotErr.Error(), "[INS0107]")
			},
		},
		{
			name:  "Error WithGetAuthContextError",
			input: sessionID.String(),
			setup: func() (*mockMiddleware.MockAuthContext, *mockRepositories.MockInterviewSessionRepository) {
				mockAuthContext := mockMiddleware.NewMockAuthContext(t)
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				mockAuthContext.EXPECT().GetAuthContext(ctx).
					Return(nil, errors.New("auth context error"))

				return mockAuthContext, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotResp *entities.GetInterviewSessionInformationResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "auth context error")
			},
		},
		{
			name:  "Error WithInvalidUserID",
			input: sessionID.String(),
			setup: func() (*mockMiddleware.MockAuthContext, *mockRepositories.MockInterviewSessionRepository) {
				mockAuthContext := mockMiddleware.NewMockAuthContext(t)
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				mockAuthContext.EXPECT().GetAuthContext(ctx).Return(&middleware.AuthPayload{
					Payload: &utils.SignInTokenPayload{
						UserID: invalidUserID,
					},
				}, nil)

				return mockAuthContext, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotResp *entities.GetInterviewSessionInformationResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "The UUID is invalid. Please try again.")
				assert.Contains(t, gotErr.Error(), "[INS0107]")
			},
		},
		{
			name:  "Error WithGetInterviewSessionInformationByIDError",
			input: sessionID.String(),
			setup: func() (*mockMiddleware.MockAuthContext, *mockRepositories.MockInterviewSessionRepository) {
				mockAuthContext := mockMiddleware.NewMockAuthContext(t)
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				mockAuthContext.EXPECT().GetAuthContext(ctx).Return(&middleware.AuthPayload{
					Payload: &utils.SignInTokenPayload{
						UserID: userID.String(),
					},
				}, nil)

				mockInterviewSessionRepo.EXPECT().GetInterviewSessionInformationByID(ctx, sessionID).
					Return(nil, errors.New("error"))

				return mockAuthContext, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotResp *entities.GetInterviewSessionInformationResp, gotErr error) {
				assert.Nil(t, gotResp)
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "error")
			},
		},
		{
			name:  "Error WithUserIDNotMatch",
			input: sessionID.String(),
			setup: func() (*mockMiddleware.MockAuthContext, *mockRepositories.MockInterviewSessionRepository) {
				mockAuthContext := mockMiddleware.NewMockAuthContext(t)
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				mockAuthContext.EXPECT().GetAuthContext(ctx).Return(&middleware.AuthPayload{
					Payload: &utils.SignInTokenPayload{
						UserID: userID.String(),
					},
				}, nil)

				mockInterviewSessionRepo.EXPECT().GetInterviewSessionInformationByID(ctx, sessionID).
					Return(&validDbUserRespNotMatch, nil)

				return mockAuthContext, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotResp *entities.GetInterviewSessionInformationResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "This user does not have access to this session.")
				assert.Contains(t, gotErr.Error(), "[INS0419]")
			},
		},
		{
			name:  "Error WithInvalidOverallScore",
			input: sessionID.String(),
			setup: func() (*mockMiddleware.MockAuthContext, *mockRepositories.MockInterviewSessionRepository) {
				mockAuthContext := mockMiddleware.NewMockAuthContext(t)
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				mockAuthContext.EXPECT().GetAuthContext(ctx).Return(&middleware.AuthPayload{
					Payload: &utils.SignInTokenPayload{
						UserID: userID.String(),
					},
				}, nil)

				mockInterviewSessionRepo.EXPECT().GetInterviewSessionInformationByID(ctx, sessionID).
					Return(&validDbRespInvalidOverallScore, nil)

				return mockAuthContext, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotResp *entities.GetInterviewSessionInformationResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "The overall score is invalid.")
				assert.Contains(t, gotErr.Error(), "[INS0420]")
			},
		},
		{
			name:  "Error WithInvalidStatus",
			input: sessionID.String(),
			setup: func() (*mockMiddleware.MockAuthContext, *mockRepositories.MockInterviewSessionRepository) {
				mockAuthContext := mockMiddleware.NewMockAuthContext(t)
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				mockAuthContext.EXPECT().GetAuthContext(ctx).Return(&middleware.AuthPayload{
					Payload: &utils.SignInTokenPayload{
						UserID: userID.String(),
					},
				}, nil)

				mockInterviewSessionRepo.EXPECT().GetInterviewSessionInformationByID(ctx, sessionID).
					Return(&validDbRespInvalidStatus, nil)

				return mockAuthContext, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotResp *entities.GetInterviewSessionInformationResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "The status is invalid.")
				assert.Contains(t, gotErr.Error(), "[INS0421]")
			},
		},
		{
			name:  "Success With Time Duration",
			input: sessionID.String(),
			setup: func() (*mockMiddleware.MockAuthContext, *mockRepositories.MockInterviewSessionRepository) {
				mockAuthContext := mockMiddleware.NewMockAuthContext(t)
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				mockAuthContext.EXPECT().GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utils.SignInTokenPayload{
							UserID: userID.String(),
						},
					}, nil)

				// Create a test case with actual duration (5 minutes 30 seconds)
				startTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
				endTime := time.Date(2024, 1, 15, 10, 35, 30, 0, time.UTC) // 5 minutes 30 seconds later

				validDbRespWithDuration := &db.GetInterviewSessionInformationByIDRow{
					UserID:         userID,
					ResumeID:       sessionID,
					ResumeFileName: "resume_file_name",
					Position:       "position",
					Status:         status,
					StartedAt:      sql.NullTime{Time: startTime, Valid: true},
					EndedAt:        sql.NullTime{Time: endTime, Valid: true},
					OverallScore:   sql.NullFloat64{Float64: 5, Valid: true},
					SummaryMd:      sql.NullString{String: "summary_md", Valid: true},
					CreatedAt:      sql.NullTime{Time: dateNow, Valid: true},
				}

				mockInterviewSessionRepo.EXPECT().GetInterviewSessionInformationByID(ctx, sessionID).
					Return(validDbRespWithDuration, nil)

				return mockAuthContext, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotResp *entities.GetInterviewSessionInformationResp, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, gotResp)
				assert.Equal(t, "5 min 30 sec", gotResp.TotalTime)
				assert.Equal(t, sessionID, gotResp.ResumeID)
				assert.Equal(t, "resume_file_name", gotResp.ResumeFileName)
				assert.Equal(t, "position", gotResp.Position)
				assert.Equal(t, status, gotResp.Status)
			},
		},
		{
			name:  "Success with equal start and end times",
			input: sessionID.String(),
			setup: func() (*mockMiddleware.MockAuthContext, *mockRepositories.MockInterviewSessionRepository) {
				mockAuthContext := mockMiddleware.NewMockAuthContext(t)
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				mockAuthContext.EXPECT().GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utils.SignInTokenPayload{
							UserID: userID.String(),
						},
					}, nil)

				// Create a test case with equal start and end times
				startTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
				endTime := startTime // Same time

				validDbRespWithEqualTimes := &db.GetInterviewSessionInformationByIDRow{
					UserID:         userID,
					ResumeID:       sessionID,
					ResumeFileName: "resume_file_name",
					Position:       "position",
					Status:         status,
					StartedAt:      sql.NullTime{Time: startTime, Valid: true},
					EndedAt:        sql.NullTime{Time: endTime, Valid: true},
					OverallScore:   sql.NullFloat64{Float64: 5, Valid: true},
					SummaryMd:      sql.NullString{String: "summary_md", Valid: true},
					CreatedAt:      sql.NullTime{Time: dateNow, Valid: true},
				}

				mockInterviewSessionRepo.EXPECT().GetInterviewSessionInformationByID(ctx, sessionID).
					Return(validDbRespWithEqualTimes, nil)

				return mockAuthContext, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotResp *entities.GetInterviewSessionInformationResp, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, gotResp)
				assert.Equal(t, "", gotResp.TotalTime) // Should be empty when start and end times are equal
				assert.Equal(t, sessionID, gotResp.ResumeID)
				assert.Equal(t, "resume_file_name", gotResp.ResumeFileName)
				assert.Equal(t, "position", gotResp.Position)
				assert.Equal(t, status, gotResp.Status)
			},
		},
		{
			name:  "Success with zero duration",
			input: sessionID.String(),
			setup: func() (*mockMiddleware.MockAuthContext, *mockRepositories.MockInterviewSessionRepository) {
				mockAuthContext := mockMiddleware.NewMockAuthContext(t)
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				mockAuthContext.EXPECT().GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utils.SignInTokenPayload{
							UserID: userID.String(),
						},
					}, nil)

				// Create a test case with very small duration (less than 1 second)
				startTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
				endTime := time.Date(2024, 1, 15, 10, 30, 0, 500000000, time.UTC) // 0.5 seconds later

				validDbRespWithZeroDuration := &db.GetInterviewSessionInformationByIDRow{
					UserID:         userID,
					ResumeID:       sessionID,
					ResumeFileName: "resume_file_name",
					Position:       "position",
					Status:         status,
					StartedAt:      sql.NullTime{Time: startTime, Valid: true},
					EndedAt:        sql.NullTime{Time: endTime, Valid: true},
					OverallScore:   sql.NullFloat64{Float64: 5, Valid: true},
					SummaryMd:      sql.NullString{String: "summary_md", Valid: true},
					CreatedAt:      sql.NullTime{Time: dateNow, Valid: true},
				}

				mockInterviewSessionRepo.EXPECT().GetInterviewSessionInformationByID(ctx, sessionID).
					Return(validDbRespWithZeroDuration, nil)

				return mockAuthContext, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotResp *entities.GetInterviewSessionInformationResp, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, gotResp)
				assert.Equal(t, "0 sec", gotResp.TotalTime) // Should show "0 sec" for zero duration
				assert.Equal(t, sessionID, gotResp.ResumeID)
				assert.Equal(t, "resume_file_name", gotResp.ResumeFileName)
				assert.Equal(t, "position", gotResp.Position)
				assert.Equal(t, status, gotResp.Status)
			},
		},
		{
			name:  "Success with seconds only",
			input: sessionID.String(),
			setup: func() (*mockMiddleware.MockAuthContext, *mockRepositories.MockInterviewSessionRepository) {
				mockAuthContext := mockMiddleware.NewMockAuthContext(t)
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				mockAuthContext.EXPECT().GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utils.SignInTokenPayload{
							UserID: userID.String(),
						},
					}, nil)

				// Create a test case with only seconds (no minutes)
				startTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
				endTime := time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC) // 45 seconds later

				validDbRespWithSecondsOnly := &db.GetInterviewSessionInformationByIDRow{
					UserID:         userID,
					ResumeID:       sessionID,
					ResumeFileName: "resume_file_name",
					Position:       "position",
					Status:         status,
					StartedAt:      sql.NullTime{Time: startTime, Valid: true},
					EndedAt:        sql.NullTime{Time: endTime, Valid: true},
					OverallScore:   sql.NullFloat64{Float64: 5, Valid: true},
					SummaryMd:      sql.NullString{String: "summary_md", Valid: true},
					CreatedAt:      sql.NullTime{Time: dateNow, Valid: true},
				}

				mockInterviewSessionRepo.EXPECT().GetInterviewSessionInformationByID(ctx, sessionID).
					Return(validDbRespWithSecondsOnly, nil)

				return mockAuthContext, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotResp *entities.GetInterviewSessionInformationResp, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, gotResp)
				assert.Equal(t, "45 sec", gotResp.TotalTime) // Should show only seconds without minutes
				assert.Equal(t, sessionID, gotResp.ResumeID)
				assert.Equal(t, "resume_file_name", gotResp.ResumeFileName)
				assert.Equal(t, "position", gotResp.Position)
				assert.Equal(t, status, gotResp.Status)
			},
		},
		{
			name:  "Success with minutes only",
			input: sessionID.String(),
			setup: func() (*mockMiddleware.MockAuthContext, *mockRepositories.MockInterviewSessionRepository) {
				mockAuthContext := mockMiddleware.NewMockAuthContext(t)
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				mockAuthContext.EXPECT().GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utils.SignInTokenPayload{
							UserID: userID.String(),
						},
					}, nil)

				// Create a test case with only minutes (no seconds)
				startTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
				endTime := time.Date(2024, 1, 15, 10, 35, 0, 0, time.UTC) // 5 minutes later

				validDbRespWithMinutesOnly := &db.GetInterviewSessionInformationByIDRow{
					UserID:         userID,
					ResumeID:       sessionID,
					ResumeFileName: "resume_file_name",
					Position:       "position",
					Status:         status,
					StartedAt:      sql.NullTime{Time: startTime, Valid: true},
					EndedAt:        sql.NullTime{Time: endTime, Valid: true},
					OverallScore:   sql.NullFloat64{Float64: 5, Valid: true},
					SummaryMd:      sql.NullString{String: "summary_md", Valid: true},
					CreatedAt:      sql.NullTime{Time: dateNow, Valid: true},
				}

				mockInterviewSessionRepo.EXPECT().GetInterviewSessionInformationByID(ctx, sessionID).
					Return(validDbRespWithMinutesOnly, nil)

				return mockAuthContext, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotResp *entities.GetInterviewSessionInformationResp, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, gotResp)
				assert.Equal(t, "5 min 0 sec", gotResp.TotalTime) // Should show minutes with 0 seconds
				assert.Equal(t, sessionID, gotResp.ResumeID)
				assert.Equal(t, "resume_file_name", gotResp.ResumeFileName)
				assert.Equal(t, "position", gotResp.Position)
				assert.Equal(t, status, gotResp.Status)
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			mockAuthContext, mockInterviewSessionRepo := tC.setup()

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
				nil,
				nil,
				nil,
				nil,
				nil,
			)

			gotResp, gotErr := svc.GetInterviewSessionInformationByID(ctx, tC.input)

			tC.verify(t, gotResp, gotErr)
		})
	}
}

func TestInterviewSessionService_GetChatHistoryBySessionIDWithEvaluation(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()
	sessionID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	defaultTurnNo := int32(constants.TurnNoDefault)

	evaluationData := map[string]interface{}{
		"overall_score": "4.20",
		"summary_md":    "Good communication skills demonstrated. Clear articulation of interest in the position.",
		"scores": []map[string]interface{}{
			{
				"criterion_id":   "1",
				"criterion_name": "Communication",
				"score":          "4.00",
				"comment":        "Clear and articulate communication.",
			},
			{
				"criterion_id":   "2",
				"criterion_name": "Enthusiasm",
				"score":          "5.00",
				"comment":        "Shows genuine interest in the role.",
			},
		},
	}
	evaluationJSON, _ := json.Marshal(evaluationData)

	dbResp := db.GetChatHistoryBySessionIDWithEvaluationRow{
		TurnID:            uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"),
		TurnNo:            1,
		Actor:             "user",
		TranscriptText:    "Hello, I'm interested in this position.",
		CurrentState:      "completed",
		CorrectedSentence: sql.NullString{String: "Hello, I am interested in this position.", Valid: true},
		StartAt:           "2025-09-24T10:00:00Z",
		EndAt:             "2025-09-24T10:00:05Z",
		Evaluation:        evaluationJSON,
	}

	validResp := &entities.GetChatHistoryBySessionIDWithEvaluationResp{
		ChatHistory: []entities.ChatHistoryWithEvaluation{
			{
				ID:                dbResp.TurnID.String(),
				TurnNo:            dbResp.TurnNo,
				Actor:             dbResp.Actor,
				Content:           dbResp.TranscriptText,
				CurrentState:      dbResp.CurrentState,
				CorrectedSentence: &dbResp.CorrectedSentence.String,
				StartAt:           dbResp.StartAt,
				EndAt:             dbResp.EndAt,
				Evaluation: &entities.Evaluation{
					OverallScore: "4.20",
					OverallColor: utils.GetScoreColor(4.2),
					SummaryMd:    "Good communication skills demonstrated. Clear articulation of interest in the position.",
					Scores: []entities.CriteriaScore{
						{
							CriterionID:   "1",
							CriterionName: "Communication",
							Score:         "4.00",
							ScoreColor:    utils.GetScoreColor(4.0),
							CommentMd:     "Clear and articulate communication.",
						},
						{
							CriterionID:   "2",
							CriterionName: "Enthusiasm",
							Score:         "5.00",
							ScoreColor:    utils.GetScoreColor(5.0),
							CommentMd:     "Shows genuine interest in the role.",
						},
					},
				},
				CurrentStateColor: constants.GetColorFromState(dbResp.CurrentState),
			},
		},
		CursorTurnNext: 1,
	}

	req := &entities.GetChatHistoryBySessionIDWithEvaluationReq{
		SessionID: sessionID.String(),
		TurnNo:    &defaultTurnNo,
	}

	testCases := []struct {
		name   string
		input  *entities.GetChatHistoryBySessionIDWithEvaluationReq
		setup  func() (*mockRepositories.MockInterviewTurnsRepository, *mockRepositories.MockInterviewSessionRepository)
		verify func(t *testing.T, gotResp *entities.GetChatHistoryBySessionIDWithEvaluationResp, gotErr error)
	}{
		{
			name:  "Success",
			input: req,
			setup: func() (*mockRepositories.MockInterviewTurnsRepository, *mockRepositories.MockInterviewSessionRepository) {
				mockInterviewTurnsRepo := mockRepositories.NewMockInterviewTurnsRepository(t)
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				expectedDbReq := &db.GetChatHistoryBySessionIDWithEvaluationParams{
					SessionID: sessionID,
					TurnNo:    int64(defaultTurnNo),
					Limit:     constants.DefaultPageSize,
				}
				mockInterviewTurnsRepo.EXPECT().GetChatHistoryBySessionIDWithEvaluation(ctx, expectedDbReq).
					Return([]db.GetChatHistoryBySessionIDWithEvaluationRow{dbResp}, nil)

				return mockInterviewTurnsRepo, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotResp *entities.GetChatHistoryBySessionIDWithEvaluationResp, gotErr error) {
				assert.NoError(t, gotErr)
				assert.Equal(t, validResp, gotResp)
			},
		},
		{
			name:  "Success WithEmptyResults",
			input: req,
			setup: func() (*mockRepositories.MockInterviewTurnsRepository, *mockRepositories.MockInterviewSessionRepository) {
				mockInterviewTurnsRepo := mockRepositories.NewMockInterviewTurnsRepository(t)
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				expectedDbReq := &db.GetChatHistoryBySessionIDWithEvaluationParams{
					SessionID: sessionID,
					TurnNo:    int64(defaultTurnNo),
					Limit:     constants.DefaultPageSize,
				}
				mockInterviewTurnsRepo.EXPECT().GetChatHistoryBySessionIDWithEvaluation(ctx, expectedDbReq).
					Return([]db.GetChatHistoryBySessionIDWithEvaluationRow{}, nil)

				return mockInterviewTurnsRepo, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotResp *entities.GetChatHistoryBySessionIDWithEvaluationResp, gotErr error) {
				assert.NoError(t, gotErr)
				assert.Equal(t, &entities.GetChatHistoryBySessionIDWithEvaluationResp{
					ChatHistory:    []entities.ChatHistoryWithEvaluation{},
					CursorTurnNext: 0,
				}, gotResp)
			},
		},
		{
			name: "Error WithInvalidSessionID",
			input: &entities.GetChatHistoryBySessionIDWithEvaluationReq{
				SessionID: "invalid-session-id",
				TurnNo:    &defaultTurnNo,
			},
			setup: func() (*mockRepositories.MockInterviewTurnsRepository, *mockRepositories.MockInterviewSessionRepository) {
				mockInterviewTurnsRepo := mockRepositories.NewMockInterviewTurnsRepository(t)
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				return mockInterviewTurnsRepo, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotResp *entities.GetChatHistoryBySessionIDWithEvaluationResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "The UUID is invalid. Please try again.")
				assert.Contains(t, gotErr.Error(), "[INS0107]")
				assert.Nil(t, gotResp)
			},
		},
		{
			name:  "Error WithGetChatHistoryBySessionIDWithEvaluationError",
			input: req,
			setup: func() (*mockRepositories.MockInterviewTurnsRepository, *mockRepositories.MockInterviewSessionRepository) {
				mockInterviewTurnsRepo := mockRepositories.NewMockInterviewTurnsRepository(t)
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				expectedDbReq := &db.GetChatHistoryBySessionIDWithEvaluationParams{
					SessionID: sessionID,
					TurnNo:    int64(defaultTurnNo),
					Limit:     constants.DefaultPageSize,
				}
				mockInterviewTurnsRepo.EXPECT().GetChatHistoryBySessionIDWithEvaluation(ctx, expectedDbReq).
					Return([]db.GetChatHistoryBySessionIDWithEvaluationRow{}, errors.New("get chat history by session ID with evaluation error"))

				return mockInterviewTurnsRepo, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotResp *entities.GetChatHistoryBySessionIDWithEvaluationResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "get chat history by session ID with evaluation error")
				assert.Nil(t, gotResp)
			},
		},
		{
			name:  "Error WithOverAllScoreMissing",
			input: req,
			setup: func() (*mockRepositories.MockInterviewTurnsRepository, *mockRepositories.MockInterviewSessionRepository) {
				mockInterviewTurnsRepo := mockRepositories.NewMockInterviewTurnsRepository(t)
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				expectedDbReq := &db.GetChatHistoryBySessionIDWithEvaluationParams{
					SessionID: sessionID,
					TurnNo:    int64(defaultTurnNo),
					Limit:     constants.DefaultPageSize,
				}

				// Create evaluation without overall_score
				evaluationData := map[string]interface{}{
					"summary_md": "Good communication skills demonstrated. Clear articulation of interest in the position.",
					"scores": []map[string]interface{}{
						{
							"criterion_id":   "1",
							"criterion_name": "Communication",
							"score":          "4.00",
							"comment":        "Clear and articulate communication.",
						},
					},
				}
				evaluationJSON, _ := json.Marshal(evaluationData)

				dbResp := db.GetChatHistoryBySessionIDWithEvaluationRow{
					TurnID:            uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"),
					TurnNo:            1,
					Actor:             "user",
					TranscriptText:    "Hello, I'm interested in this position.",
					CurrentState:      "completed",
					CorrectedSentence: sql.NullString{String: "Hello, I am interested in this position.", Valid: true},
					StartAt:           "2025-09-24T10:00:00Z",
					EndAt:             "2025-09-24T10:00:05Z",
					Evaluation:        evaluationJSON,
				}

				mockInterviewTurnsRepo.EXPECT().GetChatHistoryBySessionIDWithEvaluation(ctx, expectedDbReq).
					Return([]db.GetChatHistoryBySessionIDWithEvaluationRow{dbResp}, nil)

				return mockInterviewTurnsRepo, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotResp *entities.GetChatHistoryBySessionIDWithEvaluationResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "The overall score is invalid. Please try again.")
				assert.Contains(t, gotErr.Error(), "[INS0420]")
				assert.Nil(t, gotResp)
			},
		},
		{
			name:  "Error WithOverAllScoreParseFloatError",
			input: req,
			setup: func() (*mockRepositories.MockInterviewTurnsRepository, *mockRepositories.MockInterviewSessionRepository) {
				mockInterviewTurnsRepo := mockRepositories.NewMockInterviewTurnsRepository(t)
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				expectedDbReq := &db.GetChatHistoryBySessionIDWithEvaluationParams{
					SessionID: sessionID,
					TurnNo:    int64(defaultTurnNo),
					Limit:     constants.DefaultPageSize,
				}

				// Create evaluation with invalid overall_score
				evaluationData := map[string]interface{}{
					"overall_score": "invalid",
					"summary_md":    "Good communication skills demonstrated. Clear articulation of interest in the position.",
					"scores": []map[string]interface{}{
						{
							"criterion_id":   "1",
							"criterion_name": "Communication",
							"score":          4.0,
							"comment":        "Clear and articulate communication.",
						},
					},
				}
				evaluationJSON, _ := json.Marshal(evaluationData)

				dbResp := db.GetChatHistoryBySessionIDWithEvaluationRow{
					TurnID:            uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"),
					TurnNo:            1,
					Actor:             "user",
					TranscriptText:    "Hello, I'm interested in this position.",
					CurrentState:      "completed",
					CorrectedSentence: sql.NullString{String: "Hello, I am interested in this position.", Valid: true},
					StartAt:           "2025-09-24T10:00:00Z",
					EndAt:             "2025-09-24T10:00:05Z",
					Evaluation:        evaluationJSON,
				}

				mockInterviewTurnsRepo.EXPECT().GetChatHistoryBySessionIDWithEvaluation(ctx, expectedDbReq).
					Return([]db.GetChatHistoryBySessionIDWithEvaluationRow{dbResp}, nil)

				return mockInterviewTurnsRepo, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotResp *entities.GetChatHistoryBySessionIDWithEvaluationResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "The overall score is invalid. Please try again.")
				assert.Contains(t, gotErr.Error(), "[INS0420]")
				assert.Nil(t, gotResp)
			},
		},
		{
			name:  "Error WithOverAllScoreInvalidRange",
			input: req,
			setup: func() (*mockRepositories.MockInterviewTurnsRepository, *mockRepositories.MockInterviewSessionRepository) {
				mockInterviewTurnsRepo := mockRepositories.NewMockInterviewTurnsRepository(t)
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				expectedDbReq := &db.GetChatHistoryBySessionIDWithEvaluationParams{
					SessionID: sessionID,
					TurnNo:    int64(defaultTurnNo),
					Limit:     constants.DefaultPageSize,
				}

				// Create evaluation with invalid overall_score range (> 5)
				evaluationData := map[string]interface{}{
					"overall_score": "6.00",
					"summary_md":    "Good communication skills demonstrated. Clear articulation of interest in the position.",
					"scores": []map[string]interface{}{
						{
							"criterion_id":   "1",
							"criterion_name": "Communication",
							"score":          "4.00",
							"comment":        "Clear and articulate communication.",
						},
					},
				}
				evaluationJSON, _ := json.Marshal(evaluationData)

				dbResp := db.GetChatHistoryBySessionIDWithEvaluationRow{
					TurnID:            uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"),
					TurnNo:            1,
					Actor:             "user",
					TranscriptText:    "Hello, I'm interested in this position.",
					CurrentState:      "completed",
					CorrectedSentence: sql.NullString{String: "Hello, I am interested in this position.", Valid: true},
					StartAt:           "2025-09-24T10:00:00Z",
					EndAt:             "2025-09-24T10:00:05Z",
					Evaluation:        evaluationJSON,
				}

				mockInterviewTurnsRepo.EXPECT().GetChatHistoryBySessionIDWithEvaluation(ctx, expectedDbReq).
					Return([]db.GetChatHistoryBySessionIDWithEvaluationRow{dbResp}, nil)

				return mockInterviewTurnsRepo, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotResp *entities.GetChatHistoryBySessionIDWithEvaluationResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "The overall score is invalid. Please try again.")
				assert.Contains(t, gotErr.Error(), "[INS0420]")
				assert.Nil(t, gotResp)
			},
		},
		{
			name:  "Error WithCriteriaScoreMissingScore",
			input: req,
			setup: func() (*mockRepositories.MockInterviewTurnsRepository, *mockRepositories.MockInterviewSessionRepository) {
				mockInterviewTurnsRepo := mockRepositories.NewMockInterviewTurnsRepository(t)
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				expectedDbReq := &db.GetChatHistoryBySessionIDWithEvaluationParams{
					SessionID: sessionID,
					TurnNo:    int64(defaultTurnNo),
					Limit:     constants.DefaultPageSize,
				}

				// Create evaluation with missing score field in criteria
				evaluationData := map[string]interface{}{
					"overall_score": "3.00",
					"summary_md":    "Good communication skills demonstrated. Clear articulation of interest in the position.",
					"scores": []map[string]interface{}{
						{
							"criterion_id":   "1",
							"criterion_name": "Communication",
							"comment":        "Clear and articulate communication.",
						},
					},
				}
				evaluationJSON, _ := json.Marshal(evaluationData)

				dbResp := db.GetChatHistoryBySessionIDWithEvaluationRow{
					TurnID:            uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"),
					TurnNo:            1,
					Actor:             "user",
					TranscriptText:    "Hello, I'm interested in this position.",
					CurrentState:      "completed",
					CorrectedSentence: sql.NullString{String: "Hello, I am interested in this position.", Valid: true},
					StartAt:           "2025-09-24T10:00:00Z",
					EndAt:             "2025-09-24T10:00:05Z",
					Evaluation:        evaluationJSON,
				}

				mockInterviewTurnsRepo.EXPECT().GetChatHistoryBySessionIDWithEvaluation(ctx, expectedDbReq).
					Return([]db.GetChatHistoryBySessionIDWithEvaluationRow{dbResp}, nil)

				return mockInterviewTurnsRepo, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotResp *entities.GetChatHistoryBySessionIDWithEvaluationResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "The overall score is invalid. Please try again.")
				assert.Contains(t, gotErr.Error(), "[INS0420]")
				assert.Nil(t, gotResp)
			},
		},
		{
			name:  "Error WithCriteriaScoreParseFloatError",
			input: req,
			setup: func() (*mockRepositories.MockInterviewTurnsRepository, *mockRepositories.MockInterviewSessionRepository) {
				mockInterviewTurnsRepo := mockRepositories.NewMockInterviewTurnsRepository(t)
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				expectedDbReq := &db.GetChatHistoryBySessionIDWithEvaluationParams{
					SessionID: sessionID,
					TurnNo:    int64(defaultTurnNo),
					Limit:     constants.DefaultPageSize,
				}

				// Create evaluation with invalid criteria score
				evaluationData := map[string]interface{}{
					"overall_score": "3.00",
					"summary_md":    "Good communication skills demonstrated. Clear articulation of interest in the position.",
					"scores": []map[string]interface{}{
						{
							"criterion_id":   "1",
							"criterion_name": "Communication",
							"score":          "4.00",
							"comment":        "Clear and articulate communication.",
						},
						{
							"criterion_id":   "2",
							"criterion_name": "Enthusiasm",
							"score":          "invalid",
							"comment":        "Shows genuine interest in the role.",
						},
					},
				}
				evaluationJSON, _ := json.Marshal(evaluationData)

				dbResp := db.GetChatHistoryBySessionIDWithEvaluationRow{
					TurnID:            uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"),
					TurnNo:            1,
					Actor:             "user",
					TranscriptText:    "Hello, I'm interested in this position.",
					CurrentState:      "completed",
					CorrectedSentence: sql.NullString{String: "Hello, I am interested in this position.", Valid: true},
					StartAt:           "2025-09-24T10:00:00Z",
					EndAt:             "2025-09-24T10:00:05Z",
					Evaluation:        evaluationJSON,
				}

				mockInterviewTurnsRepo.EXPECT().GetChatHistoryBySessionIDWithEvaluation(ctx, expectedDbReq).
					Return([]db.GetChatHistoryBySessionIDWithEvaluationRow{dbResp}, nil)

				return mockInterviewTurnsRepo, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotResp *entities.GetChatHistoryBySessionIDWithEvaluationResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "The overall score is invalid. Please try again.")
				assert.Contains(t, gotErr.Error(), "[INS0420]")
				assert.Nil(t, gotResp)
			},
		},
		{
			name:  "Error WithCriteriaScoreInvalidRange",
			input: req,
			setup: func() (*mockRepositories.MockInterviewTurnsRepository, *mockRepositories.MockInterviewSessionRepository) {
				mockInterviewTurnsRepo := mockRepositories.NewMockInterviewTurnsRepository(t)
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				expectedDbReq := &db.GetChatHistoryBySessionIDWithEvaluationParams{
					SessionID: sessionID,
					TurnNo:    int64(defaultTurnNo),
					Limit:     constants.DefaultPageSize,
				}

				// Create evaluation with invalid criteria score range (> 5)
				evaluationData := map[string]interface{}{
					"overall_score": "3.00",
					"summary_md":    "Good communication skills demonstrated. Clear articulation of interest in the position.",
					"scores": []map[string]interface{}{
						{
							"criterion_id":   "1",
							"criterion_name": "Communication",
							"score":          "7.00",
							"comment":        "Clear and articulate communication.",
						},
					},
				}
				evaluationJSON, _ := json.Marshal(evaluationData)

				dbResp := db.GetChatHistoryBySessionIDWithEvaluationRow{
					TurnID:            uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"),
					TurnNo:            1,
					Actor:             "user",
					TranscriptText:    "Hello, I'm interested in this position.",
					CurrentState:      "completed",
					CorrectedSentence: sql.NullString{String: "Hello, I am interested in this position.", Valid: true},
					StartAt:           "2025-09-24T10:00:00Z",
					EndAt:             "2025-09-24T10:00:05Z",
					Evaluation:        evaluationJSON,
				}

				mockInterviewTurnsRepo.EXPECT().GetChatHistoryBySessionIDWithEvaluation(ctx, expectedDbReq).
					Return([]db.GetChatHistoryBySessionIDWithEvaluationRow{dbResp}, nil)

				return mockInterviewTurnsRepo, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotResp *entities.GetChatHistoryBySessionIDWithEvaluationResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "The overall score is invalid. Please try again.")
				assert.Contains(t, gotErr.Error(), "[INS0420]")
				assert.Nil(t, gotResp)
			},
		},
		{
			name:  "Success WithEvaluationWithoutScores",
			input: req,
			setup: func() (*mockRepositories.MockInterviewTurnsRepository, *mockRepositories.MockInterviewSessionRepository) {
				mockInterviewTurnsRepo := mockRepositories.NewMockInterviewTurnsRepository(t)
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				expectedDbReq := &db.GetChatHistoryBySessionIDWithEvaluationParams{
					SessionID: sessionID,
					TurnNo:    int64(defaultTurnNo),
					Limit:     constants.DefaultPageSize,
				}

				// Create evaluation without scores array
				evaluationData := map[string]interface{}{
					"overall_score": "4.20",
					"summary_md":    "Good communication skills demonstrated. Clear articulation of interest in the position.",
				}
				evaluationJSON, _ := json.Marshal(evaluationData)

				dbResp := db.GetChatHistoryBySessionIDWithEvaluationRow{
					TurnID:            uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"),
					TurnNo:            1,
					Actor:             "user",
					TranscriptText:    "Hello, I'm interested in this position.",
					CurrentState:      "completed",
					CorrectedSentence: sql.NullString{String: "Hello, I am interested in this position.", Valid: true},
					StartAt:           "2025-09-24T10:00:00Z",
					EndAt:             "2025-09-24T10:00:05Z",
					Evaluation:        evaluationJSON,
				}

				mockInterviewTurnsRepo.EXPECT().GetChatHistoryBySessionIDWithEvaluation(ctx, expectedDbReq).
					Return([]db.GetChatHistoryBySessionIDWithEvaluationRow{dbResp}, nil)

				return mockInterviewTurnsRepo, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotResp *entities.GetChatHistoryBySessionIDWithEvaluationResp, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, gotResp)
				assert.Len(t, gotResp.ChatHistory, 1)
				assert.NotNil(t, gotResp.ChatHistory[0].Evaluation)
				assert.Len(t, gotResp.ChatHistory[0].Evaluation.Scores, 0)
			},
		},
		{
			name:  "Success WithNoEvaluation",
			input: req,
			setup: func() (*mockRepositories.MockInterviewTurnsRepository, *mockRepositories.MockInterviewSessionRepository) {
				mockInterviewTurnsRepo := mockRepositories.NewMockInterviewTurnsRepository(t)
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				expectedDbReq := &db.GetChatHistoryBySessionIDWithEvaluationParams{
					SessionID: sessionID,
					TurnNo:    int64(defaultTurnNo),
					Limit:     constants.DefaultPageSize,
				}

				// Create response without evaluation
				dbResp := db.GetChatHistoryBySessionIDWithEvaluationRow{
					TurnID:            uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"),
					TurnNo:            1,
					Actor:             "user",
					TranscriptText:    "Hello, I'm interested in this position.",
					CurrentState:      "completed",
					CorrectedSentence: sql.NullString{String: "Hello, I am interested in this position.", Valid: true},
					StartAt:           "2025-09-24T10:00:00Z",
					EndAt:             "2025-09-24T10:00:05Z",
					Evaluation:        nil,
				}

				mockInterviewTurnsRepo.EXPECT().GetChatHistoryBySessionIDWithEvaluation(ctx, expectedDbReq).
					Return([]db.GetChatHistoryBySessionIDWithEvaluationRow{dbResp}, nil)

				return mockInterviewTurnsRepo, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotResp *entities.GetChatHistoryBySessionIDWithEvaluationResp, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, gotResp)
				assert.Len(t, gotResp.ChatHistory, 1)
				assert.Nil(t, gotResp.ChatHistory[0].Evaluation)
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			mockInterviewTurnsRepo, mockInterviewSessionRepo := tC.setup()

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
				nil,
				nil,
				nil,
				mockInterviewTurnsRepo,
				nil,
			)

			gotResp, gotErr := svc.GetChatHistoryBySessionIDWithEvaluation(ctx, tC.input)

			tC.verify(t, gotResp, gotErr)
		})
	}
}

func TestInterviewSessionService_InitialFirstCurrentStateSession(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()
	sessionID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	validResp := &entities.InitialFirstCurrentStateSessionResp{
		CurrentState:   "completed",
		CurrentStateID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000").String(),
	}

	testCases := []struct {
		name   string
		input  *entities.InitialFirstCurrentStateSessionReq
		setup  func() (*mockUtils.MockGenerator, *mockRepositories.MockInterviewStateRepository)
		verify func(t *testing.T, gotResp *entities.InitialFirstCurrentStateSessionResp, gotErr error)
	}{
		{
			name: "Success",
			input: &entities.InitialFirstCurrentStateSessionReq{
				SessionID:    sessionID.String(),
				CurrentState: "completed",
			},
			setup: func() (*mockUtils.MockGenerator, *mockRepositories.MockInterviewStateRepository) {
				mockGenerator := mockUtils.NewMockGenerator(t)
				mockInterviewStateRepo := mockRepositories.NewMockInterviewStateRepository(t)

				mockGenerator.EXPECT().GenerateUUID(ctx).Return(uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"))

				mockInterviewStateRepo.EXPECT().CreateInterviewStateWithUpdateFlagSessionTx(ctx, mock.MatchedBy(func(req *repositories.CreateInterviewStateWithUpdateFlagSessionTxReq) bool {
					return req.SessionID.String() == sessionID.String() &&
						req.PhraseType == "completed" &&
						req.ID == uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
				})).
					Return(nil)

				return mockGenerator, mockInterviewStateRepo
			},
			verify: func(t *testing.T, gotResp *entities.InitialFirstCurrentStateSessionResp, gotErr error) {
				assert.NoError(t, gotErr)
				assert.Equal(t, validResp, gotResp)
			},
		},
		{
			name: "Error WithInvalidSessionID",
			input: &entities.InitialFirstCurrentStateSessionReq{
				SessionID:    "invalid-session-id",
				CurrentState: "completed",
			},
			setup: func() (*mockUtils.MockGenerator, *mockRepositories.MockInterviewStateRepository) {
				mockGenerator := mockUtils.NewMockGenerator(t)
				mockInterviewStateRepo := mockRepositories.NewMockInterviewStateRepository(t)

				return mockGenerator, mockInterviewStateRepo
			},
			verify: func(t *testing.T, gotResp *entities.InitialFirstCurrentStateSessionResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "The UUID is invalid. Please try again.")
				assert.Contains(t, gotErr.Error(), "[INS0107]")
				assert.Nil(t, gotResp)
			},
		},
		{
			name: "Error WithCreateInterviewStateWithUpdateFlagSessionTxError",
			input: &entities.InitialFirstCurrentStateSessionReq{
				SessionID:    sessionID.String(),
				CurrentState: "completed",
			},
			setup: func() (*mockUtils.MockGenerator, *mockRepositories.MockInterviewStateRepository) {
				mockGenerator := mockUtils.NewMockGenerator(t)
				mockInterviewStateRepo := mockRepositories.NewMockInterviewStateRepository(t)

				mockGenerator.EXPECT().GenerateUUID(ctx).Return(uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"))

				mockInterviewStateRepo.EXPECT().CreateInterviewStateWithUpdateFlagSessionTx(ctx, mock.MatchedBy(func(req *repositories.CreateInterviewStateWithUpdateFlagSessionTxReq) bool {
					return req.SessionID.String() == sessionID.String() &&
						req.PhraseType == "completed" &&
						req.ID == uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
				})).
					Return(errors.New("create interview state with update flag session tx error"))

				return mockGenerator, mockInterviewStateRepo
			},
			verify: func(t *testing.T, gotResp *entities.InitialFirstCurrentStateSessionResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "create interview state with update flag session tx error")
				assert.Nil(t, gotResp)
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			mockGenerator, mockInterviewStateRepo := tC.setup()

			svc := NewInterviewSessionService(
				lgr,
				nil,
				nil,
				mockGenerator,
				nil,
				nil,
				nil,
				nil,
				nil,
				nil,
				nil,
				nil,
				nil,
				mockInterviewStateRepo,
			)

			gotResp, gotErr := svc.InitialFirstCurrentStateSession(ctx, tC.input)

			tC.verify(t, gotResp, gotErr)
		})
	}
}

func TestInterviewSessionService_UpdateCurrentStateSessionAndLastTurnID(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()
	sessionID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	lastTurnID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440001")

	validReq := &entities.UpdateCurrentStateSessionAndLastTurnIDReq{
		CurrentState:      constants.InterviewStateIntro,
		SessionID:         sessionID.String(),
		OldCurrentStateID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440001").String(),
		OldCurrentState:   constants.InterviewStateGreeting,
		LastTurnID:        lastTurnID.String(),
	}

	redisKey := fmt.Sprintf("%s%s:%s", constants.RedisPrefixInterviewLastTurnID, sessionID.String(), constants.InterviewStateGreeting)

	testCases := []struct {
		name   string
		input  *entities.UpdateCurrentStateSessionAndLastTurnIDReq
		setup  func() (*mockUtils.MockGenerator, *mockDatabase.MockRedisClient, *mockRepositories.MockInterviewStateRepository)
		verify func(t *testing.T, gotResp *entities.UpdateCurrentStateSessionResp, gotErr error)
	}{
		{
			name:  "Success",
			input: validReq,
			setup: func() (*mockUtils.MockGenerator, *mockDatabase.MockRedisClient, *mockRepositories.MockInterviewStateRepository) {
				mockGenerator := mockUtils.NewMockGenerator(t)
				mockRedisClient := mockDatabase.NewMockRedisClient(t)
				mockInterviewStateRepo := mockRepositories.NewMockInterviewStateRepository(t)

				mockGenerator.EXPECT().GenerateUUID(ctx).Return(uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"))

				mockInterviewStateRepo.EXPECT().EndOldInterviewStateAndCreateNewInterviewStateWithUpdateFlagSessionTx(ctx, mock.MatchedBy(func(req *repositories.EndOldInterviewStateAndCreateNewInterviewStateWithUpdateFlagSessionTxReq) bool {
					return req.SessionID.String() == sessionID.String() &&
						req.PhraseType == constants.InterviewStateIntro &&
						req.LastTurnID == lastTurnID &&
						req.ID == uuid.MustParse("550e8400-e29b-41d4-a716-446655440001") &&
						req.NewID == uuid.MustParse("550e8400-e29b-41d4-a716-446655440000") &&
						req.SessionID == sessionID
				})).Return(nil)

				mockRedisClient.EXPECT().Set(ctx, mock.MatchedBy(func(payload database.RedisPayload) bool {
					return payload.Key == redisKey &&
						payload.Value == lastTurnID.String() &&
						payload.TTL == constants.RedisTTLInterviewLastTurnID
				})).Return(nil)

				return mockGenerator, mockRedisClient, mockInterviewStateRepo
			},
			verify: func(t *testing.T, gotResp *entities.UpdateCurrentStateSessionResp, gotErr error) {
				assert.NoError(t, gotErr)
				assert.Equal(t, validReq.CurrentState, gotResp.CurrentState)
				assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", gotResp.CurrentStateID)
			},
		},
		{
			name: "Error WithInvalidOldCurrentStateID",
			input: &entities.UpdateCurrentStateSessionAndLastTurnIDReq{
				OldCurrentStateID: "invalid-old-current-state-id",
				SessionID:         sessionID.String(),
				CurrentState:      constants.InterviewStateIntro,
				LastTurnID:        lastTurnID.String(),
			},
			setup: func() (*mockUtils.MockGenerator, *mockDatabase.MockRedisClient, *mockRepositories.MockInterviewStateRepository) {
				mockGenerator := mockUtils.NewMockGenerator(t)
				mockRedisClient := mockDatabase.NewMockRedisClient(t)
				mockInterviewStateRepo := mockRepositories.NewMockInterviewStateRepository(t)

				return mockGenerator, mockRedisClient, mockInterviewStateRepo
			},
			verify: func(t *testing.T, gotResp *entities.UpdateCurrentStateSessionResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "The UUID is invalid. Please try again.")
				assert.Contains(t, gotErr.Error(), "[INS0107]")
				assert.Nil(t, gotResp)
			},
		},
		{
			name: "Error WithInvalidSessionID",
			input: &entities.UpdateCurrentStateSessionAndLastTurnIDReq{
				OldCurrentStateID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440001").String(),
				SessionID:         "invalid-session-id",
				CurrentState:      constants.InterviewStateIntro,
				LastTurnID:        lastTurnID.String(),
			},
			setup: func() (*mockUtils.MockGenerator, *mockDatabase.MockRedisClient, *mockRepositories.MockInterviewStateRepository) {
				mockGenerator := mockUtils.NewMockGenerator(t)
				mockRedisClient := mockDatabase.NewMockRedisClient(t)
				mockInterviewStateRepo := mockRepositories.NewMockInterviewStateRepository(t)

				return mockGenerator, mockRedisClient, mockInterviewStateRepo
			},
			verify: func(t *testing.T, gotResp *entities.UpdateCurrentStateSessionResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "The UUID is invalid. Please try again.")
				assert.Contains(t, gotErr.Error(), "[INS0107]")
				assert.Nil(t, gotResp)
			},
		},
		{
			name: "Error WithInvalidLastTurnID",
			input: &entities.UpdateCurrentStateSessionAndLastTurnIDReq{
				OldCurrentStateID: lastTurnID.String(),
				SessionID:         sessionID.String(),
				CurrentState:      constants.InterviewStateIntro,
				LastTurnID:        "invalid-last-turn-id",
			},
			setup: func() (*mockUtils.MockGenerator, *mockDatabase.MockRedisClient, *mockRepositories.MockInterviewStateRepository) {
				mockGenerator := mockUtils.NewMockGenerator(t)
				mockRedisClient := mockDatabase.NewMockRedisClient(t)
				mockInterviewStateRepo := mockRepositories.NewMockInterviewStateRepository(t)

				return mockGenerator, mockRedisClient, mockInterviewStateRepo
			},
			verify: func(t *testing.T, gotResp *entities.UpdateCurrentStateSessionResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "The UUID is invalid. Please try again.")
				assert.Contains(t, gotErr.Error(), "[INS0107]")
				assert.Nil(t, gotResp)
			},
		},
		{
			name:  "Error WithEndOldInterviewStateAndCreateNewInterviewStateWithUpdateFlagSessionTxError",
			input: validReq,
			setup: func() (*mockUtils.MockGenerator, *mockDatabase.MockRedisClient, *mockRepositories.MockInterviewStateRepository) {
				mockGenerator := mockUtils.NewMockGenerator(t)
				mockRedisClient := mockDatabase.NewMockRedisClient(t)
				mockInterviewStateRepo := mockRepositories.NewMockInterviewStateRepository(t)

				mockGenerator.EXPECT().GenerateUUID(ctx).Return(uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"))

				mockInterviewStateRepo.EXPECT().EndOldInterviewStateAndCreateNewInterviewStateWithUpdateFlagSessionTx(ctx, mock.MatchedBy(func(req *repositories.EndOldInterviewStateAndCreateNewInterviewStateWithUpdateFlagSessionTxReq) bool {
					return req.SessionID.String() == sessionID.String() &&
						req.PhraseType == constants.InterviewStateIntro &&
						req.LastTurnID == lastTurnID &&
						req.ID == uuid.MustParse("550e8400-e29b-41d4-a716-446655440001") &&
						req.NewID == uuid.MustParse("550e8400-e29b-41d4-a716-446655440000") &&
						req.SessionID == sessionID
				})).Return(errors.New("end old interview state and create new interview state with update flag session tx error"))

				return mockGenerator, mockRedisClient, mockInterviewStateRepo
			},
			verify: func(t *testing.T, gotResp *entities.UpdateCurrentStateSessionResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "end old interview state and create new interview state with update flag session tx error")
				assert.Nil(t, gotResp)
			},
		},
		{
			name:  "Error WithSetRedisError",
			input: validReq,
			setup: func() (*mockUtils.MockGenerator, *mockDatabase.MockRedisClient, *mockRepositories.MockInterviewStateRepository) {
				mockGenerator := mockUtils.NewMockGenerator(t)
				mockRedisClient := mockDatabase.NewMockRedisClient(t)
				mockInterviewStateRepo := mockRepositories.NewMockInterviewStateRepository(t)

				mockGenerator.EXPECT().GenerateUUID(ctx).Return(uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"))

				mockInterviewStateRepo.EXPECT().EndOldInterviewStateAndCreateNewInterviewStateWithUpdateFlagSessionTx(ctx, mock.MatchedBy(func(req *repositories.EndOldInterviewStateAndCreateNewInterviewStateWithUpdateFlagSessionTxReq) bool {
					return req.SessionID.String() == sessionID.String() &&
						req.PhraseType == constants.InterviewStateIntro &&
						req.LastTurnID == lastTurnID &&
						req.ID == uuid.MustParse("550e8400-e29b-41d4-a716-446655440001") &&
						req.NewID == uuid.MustParse("550e8400-e29b-41d4-a716-446655440000") &&
						req.SessionID == sessionID
				})).Return(nil)

				mockRedisClient.EXPECT().Set(ctx, mock.MatchedBy(func(payload database.RedisPayload) bool {
					return payload.Key == redisKey &&
						payload.Value == lastTurnID.String() &&
						payload.TTL == constants.RedisTTLInterviewLastTurnID
				})).Return(errors.New("set redis error"))

				return mockGenerator, mockRedisClient, mockInterviewStateRepo
			},
			verify: func(t *testing.T, gotResp *entities.UpdateCurrentStateSessionResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "set redis error")
				assert.Nil(t, gotResp)
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			mockGenerator, mockRedisClient, mockInterviewStateRepo := tC.setup()

			svc := NewInterviewSessionService(
				lgr,
				nil,
				nil,
				mockGenerator,
				nil,
				nil,
				nil,
				nil,
				nil,
				mockRedisClient,
				nil,
				nil,
				nil,
				mockInterviewStateRepo,
			)

			gotResp, gotErr := svc.UpdateCurrentStateSessionAndLastTurnID(ctx, tC.input)

			tC.verify(t, gotResp, gotErr)
		})
	}
}

func TestInterviewSessionService_GetLastUserTurnIDBySessionIDAndCurrentState(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()
	sessionID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	lastTurnID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440001")

	validReq := &entities.GetLastUserTurnIDBySessionIDAndCurrentStateReq{
		CurrentState: constants.InterviewStateIntro,
		SessionID:    sessionID.String(),
	}

	testCases := []struct {
		name   string
		input  *entities.GetLastUserTurnIDBySessionIDAndCurrentStateReq
		setup  func() *mockRepositories.MockInterviewTurnsRepository
		verify func(t *testing.T, gotResp string, gotErr error)
	}{
		{
			name:  "Success",
			input: validReq,
			setup: func() *mockRepositories.MockInterviewTurnsRepository {
				mockInterviewTurnRepo := mockRepositories.NewMockInterviewTurnsRepository(t)

				mockInterviewTurnRepo.EXPECT().GetLastUserTurnIDBySessionIDAndCurrentState(ctx, mock.MatchedBy(func(req *db.GetLastUserTurnIDBySessionIDAndCurrentStateParams) bool {
					return req.SessionID.String() == sessionID.String() &&
						req.CurrentState == constants.InterviewStateIntro
				})).Return(lastTurnID, nil)

				return mockInterviewTurnRepo
			},
			verify: func(t *testing.T, gotResp string, gotErr error) {
				assert.NoError(t, gotErr)
				assert.Equal(t, lastTurnID.String(), gotResp)
			},
		},
		{
			name: "Error WithInvalidSessionID",
			input: &entities.GetLastUserTurnIDBySessionIDAndCurrentStateReq{
				SessionID:    "invalid-session-id",
				CurrentState: constants.InterviewStateIntro,
			},
			setup: func() *mockRepositories.MockInterviewTurnsRepository {
				mockInterviewTurnRepo := mockRepositories.NewMockInterviewTurnsRepository(t)

				return mockInterviewTurnRepo
			},
			verify: func(t *testing.T, gotResp string, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "The UUID is invalid. Please try again.")
				assert.Contains(t, gotErr.Error(), "[INS0107]")
				assert.Equal(t, "", gotResp)
			},
		},
		{
			name:  "Error WithGetLastUserTurnIDBySessionIDAndCurrentStateError",
			input: validReq,
			setup: func() *mockRepositories.MockInterviewTurnsRepository {
				mockInterviewTurnRepo := mockRepositories.NewMockInterviewTurnsRepository(t)

				mockInterviewTurnRepo.EXPECT().GetLastUserTurnIDBySessionIDAndCurrentState(ctx, mock.MatchedBy(func(req *db.GetLastUserTurnIDBySessionIDAndCurrentStateParams) bool {
					return req.SessionID.String() == sessionID.String() &&
						req.CurrentState == constants.InterviewStateIntro
				})).Return(uuid.UUID{}, errors.New("get last user turn ID by session ID and current state error"))

				return mockInterviewTurnRepo
			},
			verify: func(t *testing.T, gotResp string, gotErr error) {
				assert.Error(t, gotErr)
				assert.Equal(t, "", gotResp)
				assert.Contains(t, gotErr.Error(), "get last user turn ID by session ID and current state error")
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			mockInterviewTurnRepo := tC.setup()

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
				nil,
				nil,
				nil,
				mockInterviewTurnRepo,
				nil,
			)

			gotResp, gotErr := svc.GetLastUserTurnIDBySessionIDAndCurrentState(ctx, tC.input)

			tC.verify(t, gotResp, gotErr)
		})
	}
}

func TestInterviewSessionService_UpdateSessionStatus(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()
	sessionID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	testCases := []struct {
		name      string
		sessionID string
		status    string
		setup     func() *mockRepositories.MockInterviewSessionRepository
		verify    func(t *testing.T, gotErr error)
	}{
		{
			name:      "Success",
			sessionID: sessionID.String(),
			status:    constants.StatusTimedOut,
			setup: func() *mockRepositories.MockInterviewSessionRepository {
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				mockInterviewSessionRepo.EXPECT().UpdateSessionStatus(ctx, mock.MatchedBy(func(req *db.UpdateSessionStatusParams) bool {
					return req.ID.String() == sessionID.String()
				})).Return(nil)

				return mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotErr error) {
				assert.NoError(t, gotErr)
			},
		},
		{
			name:      "Error WithInvalidSessionID",
			sessionID: "invalid-session-id",
			status:    constants.StatusTimedOut,
			setup: func() *mockRepositories.MockInterviewSessionRepository {
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				return mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "The UUID is invalid. Please try again.")
				assert.Contains(t, gotErr.Error(), "[INS0107]")
			},
		},
		{
			name:      "Error WithUpdateIsTimedOutSessionError",
			sessionID: sessionID.String(),
			status:    constants.StatusTimedOut,
			setup: func() *mockRepositories.MockInterviewSessionRepository {
				mockInterviewSessionRepo := mockRepositories.NewMockInterviewSessionRepository(t)

				mockInterviewSessionRepo.EXPECT().UpdateSessionStatus(ctx, mock.MatchedBy(func(req *db.UpdateSessionStatusParams) bool {
					return req.ID.String() == sessionID.String()
				})).Return(errors.New("update session status error"))

				return mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "update session status error")
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			mockInterviewSessionRepo := tC.setup()

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
				nil,
				nil,
				nil,
				nil,
				nil,
			)

			gotErr := svc.UpdateSessionStatus(ctx, tC.sessionID, tC.status)

			tC.verify(t, gotErr)
		})
	}
}
