package services

import (
	"context"
	"errors"
	"mime/multipart"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.com/interview-simulation/interview-backend-server/internal/config"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	db "gitlab.com/interview-simulation/interview-backend-server/internal/db/sqlc"
	"gitlab.com/interview-simulation/interview-backend-server/internal/entities"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/aws"
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

	testCases := []struct {
		name   string
		input  *entities.CreateInterviewSessionWithNewResumeRequest
		setup  func() (*mockServices.MockResumeService, *mockMiddleware.MockAuthContext, *mockRepositories.MockResumeReposity, *mockUtils.MockGenerator, *mockRepositories.MockInterviewSessionRepository, *mockUtils.MockJwtToken, *config.Config, *mockAws.MockS3Storage, *mockRepositories.MockJobRequirementRepository, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient)
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
			setup: func() (*mockServices.MockResumeService, *mockMiddleware.MockAuthContext, *mockRepositories.MockResumeReposity, *mockUtils.MockGenerator, *mockRepositories.MockInterviewSessionRepository, *mockUtils.MockJwtToken, *config.Config, *mockAws.MockS3Storage, *mockRepositories.MockJobRequirementRepository, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeService := new(mockServices.MockResumeService)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockResumeRepo := new(mockRepositories.MockResumeReposity)
				mockGenerator := new(mockUtils.MockGenerator)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockJwtMaker := new(mockUtils.MockJwtToken)
				mockS3Storage := new(mockAws.MockS3Storage)
				mockJobRequirementRepo := new(mockRepositories.MockJobRequirementRepository)
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
				sessionID1 := uuid.New()
				resumeID := uuid.New()
				jobRequirementID := uuid.New()
				sessionID2 := uuid.New()
				tokenKey := uuid.New()

				mockGenerator.EXPECT().GenerateUUID(ctx).Return(sessionID1).Times(1)
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(resumeID).Times(1)
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(jobRequirementID).Times(1)
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(sessionID2).Times(1)
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(tokenKey).Times(1)

				// Mock resume repository for JSON generation
				mockResumeRepo.EXPECT().
					GetResumeJsonWithSummaryData(ctx, mock.AnythingOfType("*repositories.GetResumeJsonWithSummaryDataReq")).
					Return(&repositories.GetResumeJsonWithSummaryDataResponse{
						ParsedJson: repositories.PromptInfo{
							FullName:   "John Doe",
							Email:      "john@example.com",
							Experience: []string{"Software Engineer at Tech Corp"},
							Skills:     []string{"Go", "Docker", "AWS"},
						},
					}, nil)

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

				return mockResumeService, mockAuthContext, mockResumeRepo, mockGenerator, mockInterviewSessionRepo, mockJwtMaker, config, mockS3Storage, mockJobRequirementRepo, mockPublisher, mockRedisClient
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
			setup: func() (*mockServices.MockResumeService, *mockMiddleware.MockAuthContext, *mockRepositories.MockResumeReposity, *mockUtils.MockGenerator, *mockRepositories.MockInterviewSessionRepository, *mockUtils.MockJwtToken, *config.Config, *mockAws.MockS3Storage, *mockRepositories.MockJobRequirementRepository, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeService := new(mockServices.MockResumeService)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockResumeRepo := new(mockRepositories.MockResumeReposity)
				mockGenerator := new(mockUtils.MockGenerator)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockJwtMaker := new(mockUtils.MockJwtToken)
				mockS3Storage := new(mockAws.MockS3Storage)
				mockJobRequirementRepo := new(mockRepositories.MockJobRequirementRepository)
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
				sessionID1 := uuid.New()
				resumeID := uuid.New()
				jobRequirementID := uuid.New()
				sessionID2 := uuid.New()
				tokenKey := uuid.New()

				mockGenerator.EXPECT().GenerateUUID(ctx).Return(sessionID1).Times(1)
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(resumeID).Times(1)
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(jobRequirementID).Times(1)
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(sessionID2).Times(1)
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(tokenKey).Times(1)

				// Mock resume repository for JSON generation
				mockResumeRepo.EXPECT().
					GetResumeJsonWithSummaryData(ctx, mock.AnythingOfType("*repositories.GetResumeJsonWithSummaryDataReq")).
					Return(&repositories.GetResumeJsonWithSummaryDataResponse{
						ParsedJson: repositories.PromptInfo{
							FullName:   "John Doe",
							Email:      "john@example.com",
							Experience: []string{"Software Engineer at Tech Corp"},
							Skills:     []string{"Go", "Docker", "AWS"},
						},
					}, nil)

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

				return mockResumeService, mockAuthContext, mockResumeRepo, mockGenerator, mockInterviewSessionRepo, mockJwtMaker, config, mockS3Storage, mockJobRequirementRepo, mockPublisher, mockRedisClient
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
			setup: func() (*mockServices.MockResumeService, *mockMiddleware.MockAuthContext, *mockRepositories.MockResumeReposity, *mockUtils.MockGenerator, *mockRepositories.MockInterviewSessionRepository, *mockUtils.MockJwtToken, *config.Config, *mockAws.MockS3Storage, *mockRepositories.MockJobRequirementRepository, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeService := new(mockServices.MockResumeService)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockResumeRepo := new(mockRepositories.MockResumeReposity)
				mockGenerator := new(mockUtils.MockGenerator)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockJwtMaker := new(mockUtils.MockJwtToken)
				mockS3Storage := new(mockAws.MockS3Storage)
				mockJobRequirementRepo := new(mockRepositories.MockJobRequirementRepository)
				mockPublisher := new(queue.MockRedisTaskPublisher)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(nil, errors.New("auth failed"))

				config := &config.Config{}

				return mockResumeService, mockAuthContext, mockResumeRepo, mockGenerator, mockInterviewSessionRepo, mockJwtMaker, config, mockS3Storage, mockJobRequirementRepo, mockPublisher, mockRedisClient
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
			setup: func() (*mockServices.MockResumeService, *mockMiddleware.MockAuthContext, *mockRepositories.MockResumeReposity, *mockUtils.MockGenerator, *mockRepositories.MockInterviewSessionRepository, *mockUtils.MockJwtToken, *config.Config, *mockAws.MockS3Storage, *mockRepositories.MockJobRequirementRepository, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeService := new(mockServices.MockResumeService)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockResumeRepo := new(mockRepositories.MockResumeReposity)
				mockGenerator := new(mockUtils.MockGenerator)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockJwtMaker := new(mockUtils.MockJwtToken)
				mockS3Storage := new(mockAws.MockS3Storage)
				mockJobRequirementRepo := new(mockRepositories.MockJobRequirementRepository)
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
				sessionID := uuid.New()
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(sessionID).Times(1)

				mockResumeRepo.EXPECT().
					GetResumeJsonWithSummaryData(ctx, mock.AnythingOfType("*repositories.GetResumeJsonWithSummaryDataReq")).
					Return(nil, errors.New("JSON generation failed"))

				config := &config.Config{}

				return mockResumeService, mockAuthContext, mockResumeRepo, mockGenerator, mockInterviewSessionRepo, mockJwtMaker, config, mockS3Storage, mockJobRequirementRepo, mockPublisher, mockRedisClient
			},
			verify: func(t *testing.T, gotResp *entities.CreateInterviewSessionWithNewResumeResponse, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
				assert.Contains(t, gotErr.Error(), "JSON generation failed")
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
			setup: func() (*mockServices.MockResumeService, *mockMiddleware.MockAuthContext, *mockRepositories.MockResumeReposity, *mockUtils.MockGenerator, *mockRepositories.MockInterviewSessionRepository, *mockUtils.MockJwtToken, *config.Config, *mockAws.MockS3Storage, *mockRepositories.MockJobRequirementRepository, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeService := new(mockServices.MockResumeService)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockResumeRepo := new(mockRepositories.MockResumeReposity)
				mockGenerator := new(mockUtils.MockGenerator)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockJwtMaker := new(mockUtils.MockJwtToken)
				mockS3Storage := new(mockAws.MockS3Storage)
				mockJobRequirementRepo := new(mockRepositories.MockJobRequirementRepository)
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
				sessionID := uuid.New()
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(sessionID).Times(1)

				mockResumeRepo.EXPECT().
					GetResumeJsonWithSummaryData(ctx, mock.AnythingOfType("*repositories.GetResumeJsonWithSummaryDataReq")).
					Return(&repositories.GetResumeJsonWithSummaryDataResponse{
						ParsedJson: repositories.PromptInfo{
							FullName:   "John Doe",
							Email:      "john@example.com",
							Experience: []string{"Software Engineer at Tech Corp"},
							Skills:     []string{"Go", "Docker", "AWS"},
						},
					}, nil)

				mockS3Storage.EXPECT().
					UploadFile(ctx, mock.AnythingOfType("*multipart.FileHeader"), constants.S3ResumeKey, userID.String()).
					Return("", errors.New("S3 upload failed"))

				config := &config.Config{}

				return mockResumeService, mockAuthContext, mockResumeRepo, mockGenerator, mockInterviewSessionRepo, mockJwtMaker, config, mockS3Storage, mockJobRequirementRepo, mockPublisher, mockRedisClient
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
			setup: func() (*mockServices.MockResumeService, *mockMiddleware.MockAuthContext, *mockRepositories.MockResumeReposity, *mockUtils.MockGenerator, *mockRepositories.MockInterviewSessionRepository, *mockUtils.MockJwtToken, *config.Config, *mockAws.MockS3Storage, *mockRepositories.MockJobRequirementRepository, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeService := new(mockServices.MockResumeService)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockResumeRepo := new(mockRepositories.MockResumeReposity)
				mockGenerator := new(mockUtils.MockGenerator)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockJwtMaker := new(mockUtils.MockJwtToken)
				mockS3Storage := new(mockAws.MockS3Storage)
				mockJobRequirementRepo := new(mockRepositories.MockJobRequirementRepository)
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
				sessionID := uuid.New()
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(sessionID).Times(1)

				mockResumeRepo.EXPECT().
					GetResumeJsonWithSummaryData(ctx, mock.AnythingOfType("*repositories.GetResumeJsonWithSummaryDataReq")).
					Return(&repositories.GetResumeJsonWithSummaryDataResponse{
						ParsedJson: repositories.PromptInfo{
							FullName:   "John Doe",
							Email:      "john@example.com",
							Experience: []string{"Software Engineer at Tech Corp"},
							Skills:     []string{"Go", "Docker", "AWS"},
						},
					}, nil)

				mockS3Storage.EXPECT().
					UploadFile(ctx, mock.AnythingOfType("*multipart.FileHeader"), constants.S3ResumeKey, userID.String()).
					Return("s3-key-123", nil)

				mockResumeRepo.EXPECT().
					CheckIsDefaultResumeExistsByUserID(ctx, userID).
					Return(false, errors.New("default resume check failed"))

				config := &config.Config{}

				return mockResumeService, mockAuthContext, mockResumeRepo, mockGenerator, mockInterviewSessionRepo, mockJwtMaker, config, mockS3Storage, mockJobRequirementRepo, mockPublisher, mockRedisClient
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
			setup: func() (*mockServices.MockResumeService, *mockMiddleware.MockAuthContext, *mockRepositories.MockResumeReposity, *mockUtils.MockGenerator, *mockRepositories.MockInterviewSessionRepository, *mockUtils.MockJwtToken, *config.Config, *mockAws.MockS3Storage, *mockRepositories.MockJobRequirementRepository, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeService := new(mockServices.MockResumeService)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockResumeRepo := new(mockRepositories.MockResumeReposity)
				mockGenerator := new(mockUtils.MockGenerator)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockJwtMaker := new(mockUtils.MockJwtToken)
				mockS3Storage := new(mockAws.MockS3Storage)
				mockJobRequirementRepo := new(mockRepositories.MockJobRequirementRepository)
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
				sessionID1 := uuid.New()
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(sessionID1).Times(1)

				mockResumeRepo.EXPECT().
					GetResumeJsonWithSummaryData(ctx, mock.AnythingOfType("*repositories.GetResumeJsonWithSummaryDataReq")).
					Return(&repositories.GetResumeJsonWithSummaryDataResponse{
						ParsedJson: repositories.PromptInfo{
							FullName:   "John Doe",
							Email:      "john@example.com",
							Experience: []string{"Software Engineer at Tech Corp"},
							Skills:     []string{"Go", "Docker", "AWS"},
						},
					}, nil)

				mockS3Storage.EXPECT().
					UploadFile(ctx, mock.AnythingOfType("*multipart.FileHeader"), constants.S3ResumeKey, userID.String()).
					Return("s3-key-123", nil)

				mockResumeRepo.EXPECT().
					CheckIsDefaultResumeExistsByUserID(ctx, userID).
					Return(false, nil)

				// Mock generator for remaining UUIDs
				resumeID := uuid.New()
				jobRequirementID := uuid.New()
				sessionID2 := uuid.New()

				mockGenerator.EXPECT().GenerateUUID(ctx).Return(resumeID).Times(1)
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(jobRequirementID).Times(1)
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(sessionID2).Times(1)

				mockInterviewSessionRepo.EXPECT().
					CreateInterviewSessionWithNewResumeTx(ctx, mock.AnythingOfType("*repositories.CreateInterviewSessionTxReq")).
					Return(errors.New("transaction failed"))

				// Mock publisher for cleanup
				mockPublisher.EXPECT().
					PublishTaskDeleteFile(ctx, mock.AnythingOfType("*aws.DeleteFilePayload")).
					Return(nil)

				config := &config.Config{}

				return mockResumeService, mockAuthContext, mockResumeRepo, mockGenerator, mockInterviewSessionRepo, mockJwtMaker, config, mockS3Storage, mockJobRequirementRepo, mockPublisher, mockRedisClient
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
			setup: func() (*mockServices.MockResumeService, *mockMiddleware.MockAuthContext, *mockRepositories.MockResumeReposity, *mockUtils.MockGenerator, *mockRepositories.MockInterviewSessionRepository, *mockUtils.MockJwtToken, *config.Config, *mockAws.MockS3Storage, *mockRepositories.MockJobRequirementRepository, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeService := new(mockServices.MockResumeService)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockResumeRepo := new(mockRepositories.MockResumeReposity)
				mockGenerator := new(mockUtils.MockGenerator)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockJwtMaker := new(mockUtils.MockJwtToken)
				mockS3Storage := new(mockAws.MockS3Storage)
				mockJobRequirementRepo := new(mockRepositories.MockJobRequirementRepository)
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
				sessionID1 := uuid.New()
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(sessionID1).Times(1)

				mockResumeRepo.EXPECT().
					GetResumeJsonWithSummaryData(ctx, mock.AnythingOfType("*repositories.GetResumeJsonWithSummaryDataReq")).
					Return(&repositories.GetResumeJsonWithSummaryDataResponse{
						ParsedJson: repositories.PromptInfo{
							FullName:   "John Doe",
							Email:      "john@example.com",
							Experience: []string{"Software Engineer at Tech Corp"},
							Skills:     []string{"Go", "Docker", "AWS"},
						},
					}, nil)

				mockS3Storage.EXPECT().
					UploadFile(ctx, mock.AnythingOfType("*multipart.FileHeader"), constants.S3ResumeKey, userID.String()).
					Return("s3-key-123", nil)

				mockResumeRepo.EXPECT().
					CheckIsDefaultResumeExistsByUserID(ctx, userID).
					Return(false, nil)

				// Mock generator for remaining UUIDs
				resumeID := uuid.New()
				jobRequirementID := uuid.New()
				sessionID2 := uuid.New()
				tokenKey := uuid.New()

				mockGenerator.EXPECT().GenerateUUID(ctx).Return(resumeID).Times(1)
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(jobRequirementID).Times(1)
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(sessionID2).Times(1)
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

				return mockResumeService, mockAuthContext, mockResumeRepo, mockGenerator, mockInterviewSessionRepo, mockJwtMaker, config, mockS3Storage, mockJobRequirementRepo, mockPublisher, mockRedisClient
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
			setup: func() (*mockServices.MockResumeService, *mockMiddleware.MockAuthContext, *mockRepositories.MockResumeReposity, *mockUtils.MockGenerator, *mockRepositories.MockInterviewSessionRepository, *mockUtils.MockJwtToken, *config.Config, *mockAws.MockS3Storage, *mockRepositories.MockJobRequirementRepository, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeService := new(mockServices.MockResumeService)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockResumeRepo := new(mockRepositories.MockResumeReposity)
				mockGenerator := new(mockUtils.MockGenerator)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockJwtMaker := new(mockUtils.MockJwtToken)
				mockS3Storage := new(mockAws.MockS3Storage)
				mockJobRequirementRepo := new(mockRepositories.MockJobRequirementRepository)
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
				sessionID1 := uuid.New()
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(sessionID1).Times(1)

				// Mock resume repository to return error
				mockResumeRepo.EXPECT().
					GetResumeJsonWithSummaryData(ctx, mock.AnythingOfType("*repositories.GetResumeJsonWithSummaryDataReq")).
					Return(nil, errors.New("resume processing failed"))

				config := &config.Config{}

				return mockResumeService, mockAuthContext, mockResumeRepo, mockGenerator, mockInterviewSessionRepo, mockJwtMaker, config, mockS3Storage, mockJobRequirementRepo, mockPublisher, mockRedisClient
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
			setup: func() (*mockServices.MockResumeService, *mockMiddleware.MockAuthContext, *mockRepositories.MockResumeReposity, *mockUtils.MockGenerator, *mockRepositories.MockInterviewSessionRepository, *mockUtils.MockJwtToken, *config.Config, *mockAws.MockS3Storage, *mockRepositories.MockJobRequirementRepository, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeService := new(mockServices.MockResumeService)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockResumeRepo := new(mockRepositories.MockResumeReposity)
				mockGenerator := new(mockUtils.MockGenerator)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockJwtMaker := new(mockUtils.MockJwtToken)
				mockS3Storage := new(mockAws.MockS3Storage)
				mockJobRequirementRepo := new(mockRepositories.MockJobRequirementRepository)
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
				sessionID1 := uuid.New()
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(sessionID1).Times(1)

				mockResumeRepo.EXPECT().
					GetResumeJsonWithSummaryData(ctx, mock.AnythingOfType("*repositories.GetResumeJsonWithSummaryDataReq")).
					Return(&repositories.GetResumeJsonWithSummaryDataResponse{
						ParsedJson: repositories.PromptInfo{
							FullName:   "John Doe",
							Email:      "john@example.com",
							Experience: []string{"Software Engineer at Tech Corp"},
							Skills:     []string{"Go", "Docker", "AWS"},
						},
					}, nil)

				mockS3Storage.EXPECT().
					UploadFile(ctx, mock.AnythingOfType("*multipart.FileHeader"), constants.S3ResumeKey, userID.String()).
					Return("s3-key-123", nil)

				mockResumeRepo.EXPECT().
					CheckIsDefaultResumeExistsByUserID(ctx, userID).
					Return(false, nil)

				// Mock generator for remaining UUIDs
				resumeID := uuid.New()
				jobRequirementID := uuid.New()
				sessionID2 := uuid.New()

				mockGenerator.EXPECT().GenerateUUID(ctx).Return(resumeID).Times(1)
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(jobRequirementID).Times(1)
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(sessionID2).Times(1)

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

				return mockResumeService, mockAuthContext, mockResumeRepo, mockGenerator, mockInterviewSessionRepo, mockJwtMaker, config, mockS3Storage, mockJobRequirementRepo, mockPublisher, mockRedisClient
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
			setup: func() (*mockServices.MockResumeService, *mockMiddleware.MockAuthContext, *mockRepositories.MockResumeReposity, *mockUtils.MockGenerator, *mockRepositories.MockInterviewSessionRepository, *mockUtils.MockJwtToken, *config.Config, *mockAws.MockS3Storage, *mockRepositories.MockJobRequirementRepository, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeService := new(mockServices.MockResumeService)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockResumeRepo := new(mockRepositories.MockResumeReposity)
				mockGenerator := new(mockUtils.MockGenerator)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockJwtMaker := new(mockUtils.MockJwtToken)
				mockS3Storage := new(mockAws.MockS3Storage)
				mockJobRequirementRepo := new(mockRepositories.MockJobRequirementRepository)
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
				sessionID := uuid.New()
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(sessionID).Times(1)

				mockResumeRepo.EXPECT().
					GetResumeJsonWithSummaryData(ctx, mock.AnythingOfType("*repositories.GetResumeJsonWithSummaryDataReq")).
					Return(nil, errors.New("invalid file: file is nil or corrupted"))

				config := &config.Config{}

				return mockResumeService, mockAuthContext, mockResumeRepo, mockGenerator, mockInterviewSessionRepo, mockJwtMaker, config, mockS3Storage, mockJobRequirementRepo, mockPublisher, mockRedisClient
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
			setup: func() (*mockServices.MockResumeService, *mockMiddleware.MockAuthContext, *mockRepositories.MockResumeReposity, *mockUtils.MockGenerator, *mockRepositories.MockInterviewSessionRepository, *mockUtils.MockJwtToken, *config.Config, *mockAws.MockS3Storage, *mockRepositories.MockJobRequirementRepository, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeService := new(mockServices.MockResumeService)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockResumeRepo := new(mockRepositories.MockResumeReposity)
				mockGenerator := new(mockUtils.MockGenerator)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockJwtMaker := new(mockUtils.MockJwtToken)
				mockS3Storage := new(mockAws.MockS3Storage)
				mockJobRequirementRepo := new(mockRepositories.MockJobRequirementRepository)
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
				sessionID1 := uuid.New()
				resumeID := uuid.New()
				jobRequirementID := uuid.New()
				sessionID2 := uuid.New()
				tokenKey := uuid.New()

				mockGenerator.EXPECT().GenerateUUID(ctx).Return(sessionID1).Times(1)
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(resumeID).Times(1)
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(jobRequirementID).Times(1)
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(sessionID2).Times(1)
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(tokenKey).Times(1)

				// Mock resume repository for JSON generation with successful file conversion
				mockResumeRepo.EXPECT().
					GetResumeJsonWithSummaryData(ctx, mock.AnythingOfType("*repositories.GetResumeJsonWithSummaryDataReq")).
					Return(&repositories.GetResumeJsonWithSummaryDataResponse{
						ParsedJson: repositories.PromptInfo{
							FullName:   "John Doe",
							Email:      "john@example.com",
							Experience: []string{"Software Engineer at Tech Corp"},
							Skills:     []string{"Go", "Docker", "AWS"},
						},
					}, nil)

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

				return mockResumeService, mockAuthContext, mockResumeRepo, mockGenerator, mockInterviewSessionRepo, mockJwtMaker, config, mockS3Storage, mockJobRequirementRepo, mockPublisher, mockRedisClient
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
			mockResumeService, mockAuthContext, mockResumeRepo, mockGenerator, mockInterviewSessionRepo, mockJwtMaker, config, mockS3Storage, mockJobRequirementRepo, mockPublisher, mockRedisClient := tC.setup()
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
				if mockJobRequirementRepo != nil {
					mockJobRequirementRepo.AssertExpectations(t)
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
				mockJobRequirementRepo,
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
	resumeID := uuid.New()

	testCases := []struct {
		name   string
		input  *entities.CreateInterviewSessionWithExistingResumeReq
		setup  func() (*mockServices.MockResumeService, *mockMiddleware.MockAuthContext, *mockRepositories.MockResumeReposity, *mockUtils.MockGenerator, *mockRepositories.MockInterviewSessionRepository, *mockUtils.MockJwtToken, *config.Config, *mockAws.MockS3Storage, *mockRepositories.MockJobRequirementRepository, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient)
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
			setup: func() (*mockServices.MockResumeService, *mockMiddleware.MockAuthContext, *mockRepositories.MockResumeReposity, *mockUtils.MockGenerator, *mockRepositories.MockInterviewSessionRepository, *mockUtils.MockJwtToken, *config.Config, *mockAws.MockS3Storage, *mockRepositories.MockJobRequirementRepository, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeService := new(mockServices.MockResumeService)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockResumeRepo := new(mockRepositories.MockResumeReposity)
				mockGenerator := new(mockUtils.MockGenerator)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockJwtMaker := new(mockUtils.MockJwtToken)
				mockS3Storage := new(mockAws.MockS3Storage)
				mockJobRequirementRepo := new(mockRepositories.MockJobRequirementRepository)
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
				sessionID1 := uuid.New()
				jobRequirementID := uuid.New()
				sessionID2 := uuid.New()
				tokenKey := uuid.New()

				mockGenerator.EXPECT().GenerateUUID(ctx).Return(sessionID1).Times(1)
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(jobRequirementID).Times(1)
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(sessionID2).Times(1)
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(tokenKey).Times(1)

				// Mock resume repository for JSON generation
				mockResumeRepo.EXPECT().
					GetResumeJsonWithSummaryData(ctx, mock.AnythingOfType("*repositories.GetResumeJsonWithSummaryDataReq")).
					Return(&repositories.GetResumeJsonWithSummaryDataResponse{
						ParsedJson: repositories.PromptInfo{
							FullName:   "John Doe",
							Email:      "john@example.com",
							Experience: []string{"Software Engineer at Tech Corp"},
							Skills:     []string{"Go", "Docker", "AWS"},
						},
					}, nil)

				// Mock interview session repository
				mockInterviewSessionRepo.EXPECT().
					CreateInterviewSessionWithExistingResumeTx(ctx, mock.AnythingOfType("*repositories.CreateInterviewSessionWithExistingResumeTxReq")).
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

				return mockResumeService, mockAuthContext, mockResumeRepo, mockGenerator, mockInterviewSessionRepo, mockJwtMaker, config, mockS3Storage, mockJobRequirementRepo, mockPublisher, mockRedisClient
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
			setup: func() (*mockServices.MockResumeService, *mockMiddleware.MockAuthContext, *mockRepositories.MockResumeReposity, *mockUtils.MockGenerator, *mockRepositories.MockInterviewSessionRepository, *mockUtils.MockJwtToken, *config.Config, *mockAws.MockS3Storage, *mockRepositories.MockJobRequirementRepository, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeService := new(mockServices.MockResumeService)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockResumeRepo := new(mockRepositories.MockResumeReposity)
				mockGenerator := new(mockUtils.MockGenerator)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockJwtMaker := new(mockUtils.MockJwtToken)
				mockS3Storage := new(mockAws.MockS3Storage)
				mockJobRequirementRepo := new(mockRepositories.MockJobRequirementRepository)
				mockPublisher := new(queue.MockRedisTaskPublisher)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(nil, errors.New("auth failed"))

				config := &config.Config{}

				return mockResumeService, mockAuthContext, mockResumeRepo, mockGenerator, mockInterviewSessionRepo, mockJwtMaker, config, mockS3Storage, mockJobRequirementRepo, mockPublisher, mockRedisClient
			},
			verify: func(t *testing.T, gotResp *entities.CreateInterviewSessionWithExistingResumeResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
				assert.Equal(t, "auth failed", gotErr.Error())
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
			setup: func() (*mockServices.MockResumeService, *mockMiddleware.MockAuthContext, *mockRepositories.MockResumeReposity, *mockUtils.MockGenerator, *mockRepositories.MockInterviewSessionRepository, *mockUtils.MockJwtToken, *config.Config, *mockAws.MockS3Storage, *mockRepositories.MockJobRequirementRepository, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeService := new(mockServices.MockResumeService)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockResumeRepo := new(mockRepositories.MockResumeReposity)
				mockGenerator := new(mockUtils.MockGenerator)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockJwtMaker := new(mockUtils.MockJwtToken)
				mockS3Storage := new(mockAws.MockS3Storage)
				mockJobRequirementRepo := new(mockRepositories.MockJobRequirementRepository)
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

				return mockResumeService, mockAuthContext, mockResumeRepo, mockGenerator, mockInterviewSessionRepo, mockJwtMaker, config, mockS3Storage, mockJobRequirementRepo, mockPublisher, mockRedisClient
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
			setup: func() (*mockServices.MockResumeService, *mockMiddleware.MockAuthContext, *mockRepositories.MockResumeReposity, *mockUtils.MockGenerator, *mockRepositories.MockInterviewSessionRepository, *mockUtils.MockJwtToken, *config.Config, *mockAws.MockS3Storage, *mockRepositories.MockJobRequirementRepository, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeService := new(mockServices.MockResumeService)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockResumeRepo := new(mockRepositories.MockResumeReposity)
				mockGenerator := new(mockUtils.MockGenerator)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockJwtMaker := new(mockUtils.MockJwtToken)
				mockS3Storage := new(mockAws.MockS3Storage)
				mockJobRequirementRepo := new(mockRepositories.MockJobRequirementRepository)
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

				return mockResumeService, mockAuthContext, mockResumeRepo, mockGenerator, mockInterviewSessionRepo, mockJwtMaker, config, mockS3Storage, mockJobRequirementRepo, mockPublisher, mockRedisClient
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
			setup: func() (*mockServices.MockResumeService, *mockMiddleware.MockAuthContext, *mockRepositories.MockResumeReposity, *mockUtils.MockGenerator, *mockRepositories.MockInterviewSessionRepository, *mockUtils.MockJwtToken, *config.Config, *mockAws.MockS3Storage, *mockRepositories.MockJobRequirementRepository, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeService := new(mockServices.MockResumeService)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockResumeRepo := new(mockRepositories.MockResumeReposity)
				mockGenerator := new(mockUtils.MockGenerator)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockJwtMaker := new(mockUtils.MockJwtToken)
				mockS3Storage := new(mockAws.MockS3Storage)
				mockJobRequirementRepo := new(mockRepositories.MockJobRequirementRepository)
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
					GetResumeJsonWithSummaryData(ctx, mock.AnythingOfType("*repositories.GetResumeJsonWithSummaryDataReq")).
					Return(nil, errors.New("JSON generation failed"))

				config := &config.Config{}

				return mockResumeService, mockAuthContext, mockResumeRepo, mockGenerator, mockInterviewSessionRepo, mockJwtMaker, config, mockS3Storage, mockJobRequirementRepo, mockPublisher, mockRedisClient
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
			setup: func() (*mockServices.MockResumeService, *mockMiddleware.MockAuthContext, *mockRepositories.MockResumeReposity, *mockUtils.MockGenerator, *mockRepositories.MockInterviewSessionRepository, *mockUtils.MockJwtToken, *config.Config, *mockAws.MockS3Storage, *mockRepositories.MockJobRequirementRepository, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeService := new(mockServices.MockResumeService)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockResumeRepo := new(mockRepositories.MockResumeReposity)
				mockGenerator := new(mockUtils.MockGenerator)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockJwtMaker := new(mockUtils.MockJwtToken)
				mockS3Storage := new(mockAws.MockS3Storage)
				mockJobRequirementRepo := new(mockRepositories.MockJobRequirementRepository)
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
					GetResumeJsonWithSummaryData(ctx, mock.AnythingOfType("*repositories.GetResumeJsonWithSummaryDataReq")).
					Return(&repositories.GetResumeJsonWithSummaryDataResponse{
						ParsedJson: repositories.PromptInfo{
							FullName:   "John Doe",
							Email:      "john@example.com",
							Experience: []string{"Software Engineer at Tech Corp"},
							Skills:     []string{"Go", "Docker", "AWS"},
						},
					}, nil)

				// Mock generator for remaining UUIDs
				jobRequirementID := uuid.New()
				sessionID2 := uuid.New()

				mockGenerator.EXPECT().GenerateUUID(ctx).Return(jobRequirementID).Times(1)
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(sessionID2).Times(1)

				// Mock interview session repository to return error
				mockInterviewSessionRepo.EXPECT().
					CreateInterviewSessionWithExistingResumeTx(ctx, mock.AnythingOfType("*repositories.CreateInterviewSessionWithExistingResumeTxReq")).
					Return(errors.New("database transaction failed"))

				config := &config.Config{}

				return mockResumeService, mockAuthContext, mockResumeRepo, mockGenerator, mockInterviewSessionRepo, mockJwtMaker, config, mockS3Storage, mockJobRequirementRepo, mockPublisher, mockRedisClient
			},
			verify: func(t *testing.T, gotResp *entities.CreateInterviewSessionWithExistingResumeResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
				assert.Contains(t, gotErr.Error(), "database transaction failed")
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
			setup: func() (*mockServices.MockResumeService, *mockMiddleware.MockAuthContext, *mockRepositories.MockResumeReposity, *mockUtils.MockGenerator, *mockRepositories.MockInterviewSessionRepository, *mockUtils.MockJwtToken, *config.Config, *mockAws.MockS3Storage, *mockRepositories.MockJobRequirementRepository, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeService := new(mockServices.MockResumeService)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockResumeRepo := new(mockRepositories.MockResumeReposity)
				mockGenerator := new(mockUtils.MockGenerator)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockJwtMaker := new(mockUtils.MockJwtToken)
				mockS3Storage := new(mockAws.MockS3Storage)
				mockJobRequirementRepo := new(mockRepositories.MockJobRequirementRepository)
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
					GetResumeJsonWithSummaryData(ctx, mock.AnythingOfType("*repositories.GetResumeJsonWithSummaryDataReq")).
					Return(&repositories.GetResumeJsonWithSummaryDataResponse{
						ParsedJson: repositories.PromptInfo{
							FullName:   "John Doe",
							Email:      "john@example.com",
							Experience: []string{"Software Engineer at Tech Corp"},
							Skills:     []string{"Go", "Docker", "AWS"},
						},
					}, nil)

				// Mock generator for remaining UUIDs
				jobRequirementID := uuid.New()
				sessionID2 := uuid.New()

				mockGenerator.EXPECT().GenerateUUID(ctx).Return(jobRequirementID).Times(1)
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(sessionID2).Times(1)

				mockInterviewSessionRepo.EXPECT().
					CreateInterviewSessionWithExistingResumeTx(ctx, mock.AnythingOfType("*repositories.CreateInterviewSessionWithExistingResumeTxReq")).
					Return(errors.New("transaction failed"))

				config := &config.Config{}

				return mockResumeService, mockAuthContext, mockResumeRepo, mockGenerator, mockInterviewSessionRepo, mockJwtMaker, config, mockS3Storage, mockJobRequirementRepo, mockPublisher, mockRedisClient
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
			setup: func() (*mockServices.MockResumeService, *mockMiddleware.MockAuthContext, *mockRepositories.MockResumeReposity, *mockUtils.MockGenerator, *mockRepositories.MockInterviewSessionRepository, *mockUtils.MockJwtToken, *config.Config, *mockAws.MockS3Storage, *mockRepositories.MockJobRequirementRepository, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeService := new(mockServices.MockResumeService)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockResumeRepo := new(mockRepositories.MockResumeReposity)
				mockGenerator := new(mockUtils.MockGenerator)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockJwtMaker := new(mockUtils.MockJwtToken)
				mockS3Storage := new(mockAws.MockS3Storage)
				mockJobRequirementRepo := new(mockRepositories.MockJobRequirementRepository)
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
					GetResumeJsonWithSummaryData(ctx, mock.AnythingOfType("*repositories.GetResumeJsonWithSummaryDataReq")).
					Return(&repositories.GetResumeJsonWithSummaryDataResponse{
						ParsedJson: repositories.PromptInfo{
							FullName:   "John Doe",
							Email:      "john@example.com",
							Experience: []string{"Software Engineer at Tech Corp"},
							Skills:     []string{"Go", "Docker", "AWS"},
						},
					}, nil)

				// Mock generator for remaining UUIDs
				jobRequirementID := uuid.New()
				sessionID2 := uuid.New()

				mockGenerator.EXPECT().GenerateUUID(ctx).Return(jobRequirementID).Times(1)
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(sessionID2).Times(1)

				mockInterviewSessionRepo.EXPECT().
					CreateInterviewSessionWithExistingResumeTx(ctx, mock.AnythingOfType("*repositories.CreateInterviewSessionWithExistingResumeTxReq")).
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

				return mockResumeService, mockAuthContext, mockResumeRepo, mockGenerator, mockInterviewSessionRepo, mockJwtMaker, config, mockS3Storage, mockJobRequirementRepo, mockPublisher, mockRedisClient
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
			setup: func() (*mockServices.MockResumeService, *mockMiddleware.MockAuthContext, *mockRepositories.MockResumeReposity, *mockUtils.MockGenerator, *mockRepositories.MockInterviewSessionRepository, *mockUtils.MockJwtToken, *config.Config, *mockAws.MockS3Storage, *mockRepositories.MockJobRequirementRepository, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeService := new(mockServices.MockResumeService)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockResumeRepo := new(mockRepositories.MockResumeReposity)
				mockGenerator := new(mockUtils.MockGenerator)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockJwtMaker := new(mockUtils.MockJwtToken)
				mockS3Storage := new(mockAws.MockS3Storage)
				mockJobRequirementRepo := new(mockRepositories.MockJobRequirementRepository)
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
					GetResumeJsonWithSummaryData(ctx, mock.AnythingOfType("*repositories.GetResumeJsonWithSummaryDataReq")).
					Return(&repositories.GetResumeJsonWithSummaryDataResponse{
						ParsedJson: repositories.PromptInfo{
							FullName:   "John Doe",
							Email:      "john@example.com",
							Experience: []string{"Software Engineer at Tech Corp"},
							Skills:     []string{"Go", "Docker", "AWS"},
						},
					}, nil)

				// Mock generator for remaining UUIDs
				jobRequirementID := uuid.New()
				sessionID2 := uuid.New()
				tokenKey := uuid.New()

				mockGenerator.EXPECT().GenerateUUID(ctx).Return(jobRequirementID).Times(1)
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(sessionID2).Times(1)
				mockGenerator.EXPECT().GenerateUUID(ctx).Return(tokenKey).Times(1)

				mockInterviewSessionRepo.EXPECT().
					CreateInterviewSessionWithExistingResumeTx(ctx, mock.AnythingOfType("*repositories.CreateInterviewSessionWithExistingResumeTxReq")).
					Return(nil)

				mockRedisClient.EXPECT().
					Set(ctx, mock.AnythingOfType("database.RedisPayload")).
					Return(errors.New("Redis set failed"))

				config := &config.Config{
					InterviewSessionConfig: config.InterviewSessionConfig{
						InterviewSessionTokenTTL: 24 * time.Hour,
					},
				}

				return mockResumeService, mockAuthContext, mockResumeRepo, mockGenerator, mockInterviewSessionRepo, mockJwtMaker, config, mockS3Storage, mockJobRequirementRepo, mockPublisher, mockRedisClient
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
			mockResumeService, mockAuthContext, mockResumeRepo, mockGenerator, mockInterviewSessionRepo, mockJwtMaker, config, mockS3Storage, mockJobRequirementRepo, mockPublisher, mockRedisClient := tC.setup()
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
				if mockJobRequirementRepo != nil {
					mockJobRequirementRepo.AssertExpectations(t)
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
				mockJobRequirementRepo,
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

	testCases := []struct {
		name   string
		input  *entities.UpdateInterviewSessionStatusReq
		setup  func() (*mockServices.MockResumeService, *mockMiddleware.MockAuthContext, *mockRepositories.MockResumeReposity, *mockUtils.MockGenerator, *mockRepositories.MockInterviewSessionRepository, *mockUtils.MockJwtToken, *config.Config, *mockAws.MockS3Storage, *mockRepositories.MockJobRequirementRepository, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient)
		verify func(t *testing.T, gotErr error)
	}{
		{
			name: "Success",
			input: &entities.UpdateInterviewSessionStatusReq{
				SessionID: sessionID.String(),
			},
			setup: func() (*mockServices.MockResumeService, *mockMiddleware.MockAuthContext, *mockRepositories.MockResumeReposity, *mockUtils.MockGenerator, *mockRepositories.MockInterviewSessionRepository, *mockUtils.MockJwtToken, *config.Config, *mockAws.MockS3Storage, *mockRepositories.MockJobRequirementRepository, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeService := new(mockServices.MockResumeService)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockResumeRepo := new(mockRepositories.MockResumeReposity)
				mockGenerator := new(mockUtils.MockGenerator)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockJwtMaker := new(mockUtils.MockJwtToken)
				mockS3Storage := new(mockAws.MockS3Storage)
				mockJobRequirementRepo := new(mockRepositories.MockJobRequirementRepository)
				mockPublisher := new(queue.MockRedisTaskPublisher)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockInterviewSessionRepo.EXPECT().
					UpdateInterviewSessionStatus(ctx, mock.AnythingOfType("*db.UpdateInterviewSessionStatusParams")).
					Return(nil)

				config := &config.Config{}

				return mockResumeService, mockAuthContext, mockResumeRepo, mockGenerator, mockInterviewSessionRepo, mockJwtMaker, config, mockS3Storage, mockJobRequirementRepo, mockPublisher, mockRedisClient
			},
			verify: func(t *testing.T, gotErr error) {
				assert.NoError(t, gotErr)
			},
		},
		{
			name: "Error - Update interview session status failure",
			input: &entities.UpdateInterviewSessionStatusReq{
				SessionID: sessionID.String(),
			},
			setup: func() (*mockServices.MockResumeService, *mockMiddleware.MockAuthContext, *mockRepositories.MockResumeReposity, *mockUtils.MockGenerator, *mockRepositories.MockInterviewSessionRepository, *mockUtils.MockJwtToken, *config.Config, *mockAws.MockS3Storage, *mockRepositories.MockJobRequirementRepository, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeService := new(mockServices.MockResumeService)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockResumeRepo := new(mockRepositories.MockResumeReposity)
				mockGenerator := new(mockUtils.MockGenerator)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockJwtMaker := new(mockUtils.MockJwtToken)
				mockS3Storage := new(mockAws.MockS3Storage)
				mockJobRequirementRepo := new(mockRepositories.MockJobRequirementRepository)
				mockPublisher := new(queue.MockRedisTaskPublisher)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				mockInterviewSessionRepo.EXPECT().
					UpdateInterviewSessionStatus(ctx, mock.AnythingOfType("*db.UpdateInterviewSessionStatusParams")).
					Return(errors.New("database error"))

				config := &config.Config{}

				return mockResumeService, mockAuthContext, mockResumeRepo, mockGenerator, mockInterviewSessionRepo, mockJwtMaker, config, mockS3Storage, mockJobRequirementRepo, mockPublisher, mockRedisClient
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Equal(t, "database error", gotErr.Error())
			},
		},
		{
			name: "Error - Invalid session ID format",
			input: &entities.UpdateInterviewSessionStatusReq{
				SessionID: "invalid-uuid",
			},
			setup: func() (*mockServices.MockResumeService, *mockMiddleware.MockAuthContext, *mockRepositories.MockResumeReposity, *mockUtils.MockGenerator, *mockRepositories.MockInterviewSessionRepository, *mockUtils.MockJwtToken, *config.Config, *mockAws.MockS3Storage, *mockRepositories.MockJobRequirementRepository, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeService := new(mockServices.MockResumeService)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockResumeRepo := new(mockRepositories.MockResumeReposity)
				mockGenerator := new(mockUtils.MockGenerator)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockJwtMaker := new(mockUtils.MockJwtToken)
				mockS3Storage := new(mockAws.MockS3Storage)
				mockJobRequirementRepo := new(mockRepositories.MockJobRequirementRepository)
				mockPublisher := new(queue.MockRedisTaskPublisher)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				config := &config.Config{}

				return mockResumeService, mockAuthContext, mockResumeRepo, mockGenerator, mockInterviewSessionRepo, mockJwtMaker, config, mockS3Storage, mockJobRequirementRepo, mockPublisher, mockRedisClient
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Panics(t, func() {
					svc := NewInterviewSessionService(
						nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
					)
					svc.UpdateInterviewSessionStatus(ctx, &entities.UpdateInterviewSessionStatusReq{
						SessionID: "invalid-uuid",
					})
				})
			},
		},
		{
			name: "Error - Empty session ID",
			input: &entities.UpdateInterviewSessionStatusReq{
				SessionID: "",
			},
			setup: func() (*mockServices.MockResumeService, *mockMiddleware.MockAuthContext, *mockRepositories.MockResumeReposity, *mockUtils.MockGenerator, *mockRepositories.MockInterviewSessionRepository, *mockUtils.MockJwtToken, *config.Config, *mockAws.MockS3Storage, *mockRepositories.MockJobRequirementRepository, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient) {
				mockResumeService := new(mockServices.MockResumeService)
				mockAuthContext := new(mockMiddleware.MockAuthContext)
				mockResumeRepo := new(mockRepositories.MockResumeReposity)
				mockGenerator := new(mockUtils.MockGenerator)
				mockInterviewSessionRepo := new(mockRepositories.MockInterviewSessionRepository)
				mockJwtMaker := new(mockUtils.MockJwtToken)
				mockS3Storage := new(mockAws.MockS3Storage)
				mockJobRequirementRepo := new(mockRepositories.MockJobRequirementRepository)
				mockPublisher := new(queue.MockRedisTaskPublisher)
				mockRedisClient := new(mockDatabase.MockRedisClient)

				config := &config.Config{}

				return mockResumeService, mockAuthContext, mockResumeRepo, mockGenerator, mockInterviewSessionRepo, mockJwtMaker, config, mockS3Storage, mockJobRequirementRepo, mockPublisher, mockRedisClient
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Panics(t, func() {
					svc := NewInterviewSessionService(
						nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
					)
					svc.UpdateInterviewSessionStatus(ctx, &entities.UpdateInterviewSessionStatusReq{
						SessionID: "",
					})
				})
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			if tC.name == "Error - Invalid session ID format" || tC.name == "Error - Empty session ID" {
				assert.Panics(t, func() {
					svc := NewInterviewSessionService(
						lgr, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
					)
					svc.UpdateInterviewSessionStatus(ctx, tC.input)
				})
				return
			}

			mockResumeService, mockAuthContext, mockResumeRepo, mockGenerator, mockInterviewSessionRepo, mockJwtMaker, config, mockS3Storage, mockJobRequirementRepo, mockPublisher, mockRedisClient := tC.setup()
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
				if mockJobRequirementRepo != nil {
					mockJobRequirementRepo.AssertExpectations(t)
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
				mockJobRequirementRepo,
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
				nil,
				mockRedisClient,
			)

			gotResp, gotErr := svc.IsSessionValid(ctx, tC.input)

			tC.verify(t, gotResp, gotErr)
		})
	}
}
