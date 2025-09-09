package services

import (
	"context"
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
				Position:        "Software Engineer",
				Company:         "Tech Corp",
				WorkType:        "Full-time",
				JobRequirements: "Go, Docker, AWS",
				InterviewType:   "Technical",
				Language:        "English",
				IsConsent:       true,
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
				Position:        "Software Engineer",
				Company:         "Tech Corp",
				WorkType:        "Full-time",
				JobRequirements: "Go, Docker, AWS",
				InterviewType:   "Technical",
				Language:        "English",
				IsConsent:       true,
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
				Position:        "Software Engineer",
				Company:         "Tech Corp",
				WorkType:        "Full-time",
				JobRequirements: "Go, Docker, AWS",
				InterviewType:   "Technical",
				Language:        "English",
				IsConsent:       true,
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
				Position:        "Software Engineer",
				Company:         "Tech Corp",
				WorkType:        "Full-time",
				JobRequirements: "Go, Docker, AWS",
				InterviewType:   "Technical",
				Language:        "English",
				IsConsent:       true,
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
				Position:        "Software Engineer",
				Company:         "Tech Corp",
				WorkType:        "Full-time",
				JobRequirements: "Go, Docker, AWS",
				InterviewType:   "Technical",
				Language:        "English",
				IsConsent:       true,
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
				Position:        "Software Engineer",
				Company:         "Tech Corp",
				WorkType:        "Full-time",
				JobRequirements: "Go, Docker, AWS",
				InterviewType:   "Technical",
				Language:        "English",
				IsConsent:       true,
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
				Position:        "Software Engineer",
				Company:         "Tech Corp",
				WorkType:        "Full-time",
				JobRequirements: "Go, Docker, AWS",
				InterviewType:   "Technical",
				Language:        "English",
				IsConsent:       true,
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
				Position:        "Software Engineer",
				Company:         "Tech Corp",
				WorkType:        "Full-time",
				JobRequirements: "Go, Docker, AWS",
				InterviewType:   "Technical",
				Language:        "English",
				IsConsent:       true,
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
				Position:        "Software Engineer",
				Company:         "Tech Corp",
				WorkType:        "Full-time",
				JobRequirements: "Go, Docker, AWS",
				InterviewType:   "Technical",
				Language:        "English",
				IsConsent:       true,
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
				Position:        "Software Engineer",
				Company:         "Tech Corp",
				WorkType:        "Full-time",
				JobRequirements: "Go, Docker, AWS",
				InterviewType:   "Technical",
				Language:        "English",
				IsConsent:       true,
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
				Position:        "Software Engineer",
				Company:         "Tech Corp",
				WorkType:        "Full-time",
				JobRequirements: "Go, Docker, AWS",
				InterviewType:   "Technical",
				Language:        "English",
				IsConsent:       true,
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
				Position:        "Software Engineer",
				Company:         "Tech Corp",
				WorkType:        "Full-time",
				JobRequirements: "Go, Docker, AWS",
				InterviewType:   "Technical",
				Language:        "English",
				IsConsent:       true,
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
				Position:        "Software Engineer",
				Company:         "Tech Corp",
				WorkType:        "Full-time",
				JobRequirements: "Go, Docker, AWS",
				InterviewType:   "Technical",
				Language:        "English",
				IsConsent:       true,
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
				Position:        "Software Engineer",
				Company:         "Tech Corp",
				WorkType:        "Full-time",
				JobRequirements: "Go, Docker, AWS",
				InterviewType:   "Technical",
				Language:        "English",
				IsConsent:       true,
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
				mockResumeService,
				mockResumeRepo,
				mockGenerator,
				mockInterviewSessionRepo,
				mockJwtMaker,
				config,
				mockS3Storage,
				mockPublisher,
				mockRedisClient,
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
				ResumeID:        resumeID.String(),
				Position:        "Software Engineer",
				Company:         "Tech Corp",
				WorkType:        "Full-time",
				JobRequirements: "Go, Docker, AWS",
				InterviewType:   "Technical",
				Language:        "English",
				IsConsent:       true,
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
				ResumeID:        resumeID.String(),
				Position:        "Software Engineer",
				Company:         "Tech Corp",
				WorkType:        "Full-time",
				JobRequirements: "Go, Docker, AWS",
				InterviewType:   "Technical",
				Language:        "English",
				IsConsent:       true,
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
				ResumeID:        invalidResumeID,
				Position:        "Software Engineer",
				Company:         "Tech Corp",
				WorkType:        "Full-time",
				JobRequirements: "Go, Docker, AWS",
				InterviewType:   "Technical",
				Language:        "English",
				IsConsent:       true,
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
				ResumeID:        resumeID.String(),
				Position:        "Software Engineer",
				Company:         "Tech Corp",
				WorkType:        "Full-time",
				JobRequirements: "Go, Docker, AWS",
				InterviewType:   "Technical",
				Language:        "English",
				IsConsent:       true,
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
				ResumeID:        resumeID.String(),
				Position:        "Software Engineer",
				Company:         "Tech Corp",
				WorkType:        "Full-time",
				JobRequirements: "Go, Docker, AWS",
				InterviewType:   "Technical",
				Language:        "English",
				IsConsent:       true,
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
				ResumeID:        resumeID.String(),
				Position:        "Software Engineer",
				Company:         "Tech Corp",
				WorkType:        "Full-time",
				JobRequirements: "Go, Docker, AWS",
				InterviewType:   "Technical",
				Language:        "English",
				IsConsent:       true,
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
				ResumeID:        resumeID.String(),
				Position:        "Software Engineer",
				Company:         "Tech Corp",
				WorkType:        "Full-time",
				JobRequirements: "Go, Docker, AWS",
				InterviewType:   "Technical",
				Language:        "English",
				IsConsent:       true,
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
				ResumeID:        resumeID.String(),
				Position:        "Software Engineer",
				Company:         "Tech Corp",
				WorkType:        "Full-time",
				JobRequirements: "Go, Docker, AWS",
				InterviewType:   "Technical",
				Language:        "English",
				IsConsent:       true,
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
				ResumeID:        resumeID.String(),
				Position:        "Software Engineer",
				Company:         "Tech Corp",
				WorkType:        "Full-time",
				JobRequirements: "Go, Docker, AWS",
				InterviewType:   "Technical",
				Language:        "English",
				IsConsent:       true,
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
				ResumeID:        resumeID.String(),
				Position:        "Software Engineer",
				Company:         "Tech Corp",
				WorkType:        "Full-time",
				JobRequirements: "Go, Docker, AWS",
				InterviewType:   "Technical",
				Language:        "English",
				IsConsent:       true,
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
				ResumeID:        resumeID.String(),
				Position:        "Software Engineer",
				Company:         "Tech Corp",
				WorkType:        "Full-time",
				JobRequirements: "Go, Docker, AWS",
				InterviewType:   "Technical",
				Language:        "English",
				IsConsent:       true,
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
				mockResumeService,
				mockResumeRepo,
				mockGenerator,
				mockInterviewSessionRepo,
				mockJwtMaker,
				config,
				mockS3Storage,
				mockPublisher,
				mockRedisClient,
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
				mockResumeService,
				mockResumeRepo,
				mockGenerator,
				mockInterviewSessionRepo,
				mockJwtMaker,
				config,
				mockS3Storage,
				mockPublisher,
				mockRedisClient,
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
					Get(ctx, "valid_session_token").
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
					Get(ctx, "valid_session_token").
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
					Get(ctx, "valid_session_token").
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
					Get(ctx, "valid_session_token").
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
					Get(ctx, "valid_session_token").
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
					Get(ctx, "valid_session_token").
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
					Get(ctx, "valid_session_token").
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
				nil,
				mockInterviewSessionRepo,
				nil,
				nil,
				nil,
				nil,
				mockRedisClient,
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

	testCases := []struct {
		name   string
		input  *entities.CreateUserSessionTurnBySessionIDReq
		setup  func() (*mockUtils.MockGenerator, *mockDatabase.MockRedisClient, *mockRepositories.MockInterviewSessionRepository)
		verify func(t *testing.T, gotErr error)
	}{
		{
			name: "Success - TurnRedisTriggered",
			input: &entities.CreateUserSessionTurnBySessionIDReq{
				SessionID:  correctSessionID,
				TurnID:     correctTurnID,
				Transcript: "transcript",
			},
			setup: func() (*mockUtils.MockGenerator, *mockDatabase.MockRedisClient, *mockRepositories.MockInterviewSessionRepository) {
				mockGenerator := new(mockUtils.MockGenerator)
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewMaxTurnNo, correctSessionID)).
					Return("1", nil)

				mockRedisClient.EXPECT().
					HGetAll(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewStartEndTime, correctSessionID)).
					Return(map[string]string{
						"started_at": "0.01",
						"ended_at":   "2.34",
					}, nil)

				mockInterviewSessionRepo.EXPECT().
					CreateSessionTurnBySessionID(ctx, mock.MatchedBy(func(req *db.CreateInterviewTurnParams) bool {
						return req.SessionID.String() == correctSessionID &&
							req.ID.String() == correctTurnID &&
							req.TurnNo == 2 &&
							req.Actor == "user" &&
							req.TranscriptText.String == "transcript" &&
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

				return mockGenerator, mockRedisClient, mockInterviewSessionRepo
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
			setup: func() (*mockUtils.MockGenerator, *mockDatabase.MockRedisClient, *mockRepositories.MockInterviewSessionRepository) {

				return nil, nil, nil
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
				SessionID:  invalidID,
				TurnID:     correctTurnID,
				Transcript: "transcript",
			},
			setup: func() (*mockUtils.MockGenerator, *mockDatabase.MockRedisClient, *mockRepositories.MockInterviewSessionRepository) {

				return nil, nil, nil
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
				SessionID:  correctSessionID,
				TurnID:     correctTurnID,
				Transcript: "transcript",
			},
			setup: func() (*mockUtils.MockGenerator, *mockDatabase.MockRedisClient, *mockRepositories.MockInterviewSessionRepository) {
				mockGenerator := new(mockUtils.MockGenerator)
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewMaxTurnNo, correctSessionID)).
					Return("", redis.Nil)

				mockInterviewSessionRepo.EXPECT().
					GetMaxTurnNoBySessionID(ctx, uuid.MustParse(correctSessionID)).
					Return(int64(2), nil)

				mockRedisClient.EXPECT().
					HGetAll(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewStartEndTime, correctSessionID)).
					Return(map[string]string{
						"started_at": "0.01",
						"ended_at":   "2.34",
					}, nil)

				mockInterviewSessionRepo.EXPECT().
					CreateSessionTurnBySessionID(ctx, mock.MatchedBy(func(req *db.CreateInterviewTurnParams) bool {
						return req.SessionID.String() == correctSessionID &&
							req.ID.String() == correctTurnID &&
							req.TurnNo == 3 &&
							req.Actor == "user" &&
							req.TranscriptText.String == "transcript" &&
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

				return mockGenerator, mockRedisClient, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotErr error) {
				assert.NoError(t, gotErr)
			},
		},
		{
			name: "Success - With Warning Set Redis Error",
			input: &entities.CreateUserSessionTurnBySessionIDReq{
				SessionID:  correctSessionID,
				TurnID:     correctTurnID,
				Transcript: "transcript",
			},
			setup: func() (*mockUtils.MockGenerator, *mockDatabase.MockRedisClient, *mockRepositories.MockInterviewSessionRepository) {
				mockGenerator := new(mockUtils.MockGenerator)
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewMaxTurnNo, correctSessionID)).
					Return("1", nil)

				mockRedisClient.EXPECT().
					HGetAll(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewStartEndTime, correctSessionID)).
					Return(map[string]string{
						"started_at": "0.01",
						"ended_at":   "2.34",
					}, nil)

				mockInterviewSessionRepo.EXPECT().
					CreateSessionTurnBySessionID(ctx, mock.MatchedBy(func(req *db.CreateInterviewTurnParams) bool {
						return req.SessionID.String() == correctSessionID &&
							req.ID.String() == correctTurnID &&
							req.TurnNo == 2 &&
							req.Actor == "user" &&
							req.TranscriptText.String == "transcript" &&
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

				return mockGenerator, mockRedisClient, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotErr error) {
				assert.NoError(t, gotErr)
			},
		},
		{
			name: "Error - Redis Get Turn error",
			input: &entities.CreateUserSessionTurnBySessionIDReq{
				SessionID:  correctSessionID,
				TurnID:     correctTurnID,
				Transcript: "transcript",
			},
			setup: func() (*mockUtils.MockGenerator, *mockDatabase.MockRedisClient, *mockRepositories.MockInterviewSessionRepository) {
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewMaxTurnNo, correctSessionID)).
					Return("", errors.New("redis error"))

				return nil, mockRedisClient, nil
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
			},
		},
		{
			name: "Error - GetMaxTurnRepo Error",
			input: &entities.CreateUserSessionTurnBySessionIDReq{
				SessionID:  correctSessionID,
				TurnID:     correctTurnID,
				Transcript: "transcript",
			},
			setup: func() (*mockUtils.MockGenerator, *mockDatabase.MockRedisClient, *mockRepositories.MockInterviewSessionRepository) {
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewMaxTurnNo, correctSessionID)).
					Return("", redis.Nil)

				mockInterviewSessionRepo.EXPECT().
					GetMaxTurnNoBySessionID(ctx, uuid.MustParse(correctSessionID)).
					Return(0, errors.New("database error"))

				return nil, mockRedisClient, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
			},
		},
		{
			name: "Error - ParseIntTurnNo Error",
			input: &entities.CreateUserSessionTurnBySessionIDReq{
				SessionID:  correctSessionID,
				TurnID:     correctTurnID,
				Transcript: "transcript",
			},
			setup: func() (*mockUtils.MockGenerator, *mockDatabase.MockRedisClient, *mockRepositories.MockInterviewSessionRepository) {
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewMaxTurnNo, correctSessionID)).
					Return("invalid_turn_no", nil)

				return nil, mockRedisClient, nil
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
			},
		},
		{
			name: "Error - GetStartEndTime Redis Error",
			input: &entities.CreateUserSessionTurnBySessionIDReq{
				SessionID:  correctSessionID,
				TurnID:     correctTurnID,
				Transcript: "transcript",
			},
			setup: func() (*mockUtils.MockGenerator, *mockDatabase.MockRedisClient, *mockRepositories.MockInterviewSessionRepository) {
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewMaxTurnNo, correctSessionID)).
					Return("0", nil)

				mockRedisClient.EXPECT().
					HGetAll(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewStartEndTime, correctSessionID)).
					Return(nil, errors.New("redis error"))

				return nil, mockRedisClient, nil
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
			},
		},
		{
			name: "Error - GetStartEndTime Redis Nil",
			input: &entities.CreateUserSessionTurnBySessionIDReq{
				SessionID:  correctSessionID,
				TurnID:     correctTurnID,
				Transcript: "transcript",
			},
			setup: func() (*mockUtils.MockGenerator, *mockDatabase.MockRedisClient, *mockRepositories.MockInterviewSessionRepository) {
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewMaxTurnNo, correctSessionID)).
					Return("0", nil)

				mockRedisClient.EXPECT().
					HGetAll(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewStartEndTime, correctSessionID)).
					Return(nil, redis.Nil)

				return nil, mockRedisClient, nil
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
			},
		},
		{
			name: "Error - GetStartEndTime Not Found Any Field",
			input: &entities.CreateUserSessionTurnBySessionIDReq{
				SessionID:  correctSessionID,
				TurnID:     correctTurnID,
				Transcript: "transcript",
			},
			setup: func() (*mockUtils.MockGenerator, *mockDatabase.MockRedisClient, *mockRepositories.MockInterviewSessionRepository) {
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewMaxTurnNo, correctSessionID)).
					Return("0", nil)

				mockRedisClient.EXPECT().
					HGetAll(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewStartEndTime, correctSessionID)).
					Return(map[string]string{}, nil)

				return nil, mockRedisClient, nil
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), constants.ErrInterviewSessionStartEndTimeNotFound.Error())
			},
		},
		{
			name: "Error - GetStartEndTime StartedAt Not Found Error",
			input: &entities.CreateUserSessionTurnBySessionIDReq{
				SessionID:  correctSessionID,
				TurnID:     correctTurnID,
				Transcript: "transcript",
			},
			setup: func() (*mockUtils.MockGenerator, *mockDatabase.MockRedisClient, *mockRepositories.MockInterviewSessionRepository) {
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewMaxTurnNo, correctSessionID)).
					Return("0", nil)

				mockRedisClient.EXPECT().
					HGetAll(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewStartEndTime, correctSessionID)).
					Return(map[string]string{
						"ended_at": "2.34",
					}, nil)

				return nil, mockRedisClient, nil
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
			setup: func() (*mockUtils.MockGenerator, *mockDatabase.MockRedisClient, *mockRepositories.MockInterviewSessionRepository) {
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewMaxTurnNo, correctSessionID)).
					Return("0", nil)

				mockRedisClient.EXPECT().
					HGetAll(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewStartEndTime, correctSessionID)).
					Return(map[string]string{
						"started_at": "0.01",
					}, nil)

				return nil, mockRedisClient, nil
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), constants.ErrInterviewSessionEndTimeNotFound.Error())
			},
		},
		{
			name: "Error - CreateSessionTurnBySessionID Error",
			input: &entities.CreateUserSessionTurnBySessionIDReq{
				SessionID:  correctSessionID,
				TurnID:     correctTurnID,
				Transcript: "transcript",
			},
			setup: func() (*mockUtils.MockGenerator, *mockDatabase.MockRedisClient, *mockRepositories.MockInterviewSessionRepository) {
				mockGenerator := new(mockUtils.MockGenerator)
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewMaxTurnNo, correctSessionID)).
					Return("0", nil)

				mockRedisClient.EXPECT().
					HGetAll(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewStartEndTime, correctSessionID)).
					Return(map[string]string{
						"started_at": "0.01",
						"ended_at":   "2.34",
					}, nil)

				mockInterviewSessionRepo.EXPECT().
					CreateSessionTurnBySessionID(ctx, mock.MatchedBy(func(req *db.CreateInterviewTurnParams) bool {
						return req.SessionID.String() == correctSessionID &&
							req.ID.String() == correctTurnID &&
							req.TurnNo == 1 &&
							req.Actor == "user" &&
							req.TranscriptText.String == "transcript" &&
							req.StartAt == "0.01" &&
							req.EndAt == "2.34"
					})).
					Return(errors.New("database error"))

				return mockGenerator, mockRedisClient, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			mockGenerator, mockRedisClient, mockInterviewSessionRepo := tC.setup()
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
			}()

			svc := NewInterviewSessionService(
				lgr,
				nil,
				nil,
				nil,
				mockGenerator,
				mockInterviewSessionRepo,
				nil,
				nil,
				nil,
				nil,
				mockRedisClient,
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

	testCases := []struct {
		name   string
		input  *entities.CreateInterviewerSessionTurnBySessionIDReq
		setup  func() (*mockUtils.MockGenerator, *mockDatabase.MockRedisClient, *mockRepositories.MockInterviewSessionRepository)
		verify func(t *testing.T, gotErr error)
	}{
		{
			name: "Success - TurnRedisTriggered",
			input: &entities.CreateInterviewerSessionTurnBySessionIDReq{
				SessionID:  correctSessionID,
				TurnID:     correctTurnID,
				Transcript: "transcript",
				StartedAt:  correctStartedAt,
				EndedAt:    correctEndedAt,
			},
			setup: func() (*mockUtils.MockGenerator, *mockDatabase.MockRedisClient, *mockRepositories.MockInterviewSessionRepository) {
				mockGenerator := new(mockUtils.MockGenerator)
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewMaxTurnNo, correctSessionID)).
					Return("1", nil)

				mockInterviewSessionRepo.EXPECT().
					CreateSessionTurnBySessionID(context.Background(), mock.MatchedBy(func(req *db.CreateInterviewTurnParams) bool {
						return req.SessionID.String() == correctSessionID &&
							req.ID.String() == correctTurnID &&
							req.TurnNo == 2 &&
							req.Actor == "interviewer" &&
							req.TranscriptText.String == "transcript" &&
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

				return mockGenerator, mockRedisClient, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotErr error) {
				assert.NoError(t, gotErr)
			},
		},
		{
			name: "Error_Invalid session ID",
			input: &entities.CreateInterviewerSessionTurnBySessionIDReq{
				SessionID:  correctSessionID,
				TurnID:     invalidID,
				Transcript: "transcript",
				StartedAt:  correctStartedAt,
				EndedAt:    correctEndedAt,
			},
			setup: func() (*mockUtils.MockGenerator, *mockDatabase.MockRedisClient, *mockRepositories.MockInterviewSessionRepository) {

				return nil, nil, nil
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
				SessionID:  invalidID,
				TurnID:     correctTurnID,
				Transcript: "transcript",
				StartedAt:  correctStartedAt,
				EndedAt:    correctEndedAt,
			},
			setup: func() (*mockUtils.MockGenerator, *mockDatabase.MockRedisClient, *mockRepositories.MockInterviewSessionRepository) {

				return nil, nil, nil
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
				SessionID:  correctSessionID,
				TurnID:     correctTurnID,
				Transcript: "transcript",
				StartedAt:  correctStartedAt,
				EndedAt:    correctEndedAt,
			},
			setup: func() (*mockUtils.MockGenerator, *mockDatabase.MockRedisClient, *mockRepositories.MockInterviewSessionRepository) {
				mockGenerator := new(mockUtils.MockGenerator)
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewMaxTurnNo, correctSessionID)).
					Return("", redis.Nil)

				mockInterviewSessionRepo.EXPECT().
					GetMaxTurnNoBySessionID(ctx, uuid.MustParse(correctSessionID)).
					Return(int64(2), nil)

				mockInterviewSessionRepo.EXPECT().
					CreateSessionTurnBySessionID(context.Background(), mock.MatchedBy(func(req *db.CreateInterviewTurnParams) bool {
						return req.SessionID.String() == correctSessionID &&
							req.ID.String() == correctTurnID &&
							req.TurnNo == 3 &&
							req.Actor == "interviewer" &&
							req.TranscriptText.String == "transcript" &&
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

				return mockGenerator, mockRedisClient, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotErr error) {
				assert.NoError(t, gotErr)
			},
		},
		{
			name: "Success - With Warning Set Redis Error",
			input: &entities.CreateInterviewerSessionTurnBySessionIDReq{
				SessionID:  correctSessionID,
				TurnID:     correctTurnID,
				Transcript: "transcript",
				StartedAt:  correctStartedAt,
				EndedAt:    correctEndedAt,
			},
			setup: func() (*mockUtils.MockGenerator, *mockDatabase.MockRedisClient, *mockRepositories.MockInterviewSessionRepository) {
				mockGenerator := new(mockUtils.MockGenerator)
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewMaxTurnNo, correctSessionID)).
					Return("1", nil)

				mockInterviewSessionRepo.EXPECT().
					CreateSessionTurnBySessionID(context.Background(), mock.MatchedBy(func(req *db.CreateInterviewTurnParams) bool {
						return req.SessionID.String() == correctSessionID &&
							req.ID.String() == correctTurnID &&
							req.TurnNo == 2 &&
							req.Actor == "interviewer" &&
							req.TranscriptText.String == "transcript" &&
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

				return mockGenerator, mockRedisClient, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotErr error) {
				assert.NoError(t, gotErr)
			},
		},
		{
			name: "Error - Redis Get Turn error",
			input: &entities.CreateInterviewerSessionTurnBySessionIDReq{
				SessionID:  correctSessionID,
				TurnID:     correctTurnID,
				Transcript: "transcript",
				StartedAt:  correctStartedAt,
				EndedAt:    correctEndedAt,
			},
			setup: func() (*mockUtils.MockGenerator, *mockDatabase.MockRedisClient, *mockRepositories.MockInterviewSessionRepository) {
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewMaxTurnNo, correctSessionID)).
					Return("", errors.New("redis error"))

				return nil, mockRedisClient, nil
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
			},
		},
		{
			name: "Error - GetMaxTurnRepo Error",
			input: &entities.CreateInterviewerSessionTurnBySessionIDReq{
				SessionID:  correctSessionID,
				TurnID:     correctTurnID,
				Transcript: "transcript",
				StartedAt:  correctStartedAt,
				EndedAt:    correctEndedAt,
			},
			setup: func() (*mockUtils.MockGenerator, *mockDatabase.MockRedisClient, *mockRepositories.MockInterviewSessionRepository) {
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewMaxTurnNo, correctSessionID)).
					Return("", redis.Nil)

				mockInterviewSessionRepo.EXPECT().
					GetMaxTurnNoBySessionID(ctx, uuid.MustParse(correctSessionID)).
					Return(0, errors.New("database error"))

				return nil, mockRedisClient, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
			},
		},
		{
			name: "Error - ParseIntTurnNo Error",
			input: &entities.CreateInterviewerSessionTurnBySessionIDReq{
				SessionID:  correctSessionID,
				TurnID:     correctTurnID,
				Transcript: "transcript",
				StartedAt:  correctStartedAt,
				EndedAt:    correctEndedAt,
			},
			setup: func() (*mockUtils.MockGenerator, *mockDatabase.MockRedisClient, *mockRepositories.MockInterviewSessionRepository) {
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewMaxTurnNo, correctSessionID)).
					Return("invalid_turn_no", nil)

				return nil, mockRedisClient, nil
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
			},
		},
		{
			name: "Error - CreateSessionTurnBySessionID Error",
			input: &entities.CreateInterviewerSessionTurnBySessionIDReq{
				SessionID:  correctSessionID,
				TurnID:     correctTurnID,
				Transcript: "transcript",
				StartedAt:  correctStartedAt,
				EndedAt:    correctEndedAt,
			},
			setup: func() (*mockUtils.MockGenerator, *mockDatabase.MockRedisClient, *mockRepositories.MockInterviewSessionRepository) {
				mockGenerator := new(mockUtils.MockGenerator)
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)

				mockRedisClient.EXPECT().
					Get(ctx, fmt.Sprintf("%s%s", constants.RedisPrefixInterviewMaxTurnNo, correctSessionID)).
					Return("0", nil)

				mockInterviewSessionRepo.EXPECT().
					CreateSessionTurnBySessionID(context.Background(), mock.MatchedBy(func(req *db.CreateInterviewTurnParams) bool {
						return req.SessionID.String() == correctSessionID &&
							req.ID.String() == correctTurnID &&
							req.TurnNo == 1 &&
							req.Actor == "interviewer" &&
							req.TranscriptText.String == "transcript" &&
							req.StartAt == correctStartedAt &&
							req.EndAt == correctEndedAt
					})).
					Return(errors.New("database error"))

				return mockGenerator, mockRedisClient, mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			mockGenerator, mockRedisClient, mockInterviewSessionRepo := tC.setup()
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
			}()

			svc := NewInterviewSessionService(
				lgr,
				nil,
				nil,
				nil,
				mockGenerator,
				mockInterviewSessionRepo,
				nil,
				nil,
				nil,
				nil,
				mockRedisClient,
			)

			gotErr := svc.CreateInterviewerSessionTurnBySessionID(ctx, tC.input)

			tC.verify(t, gotErr)
		})
	}
}
