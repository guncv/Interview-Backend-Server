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
						InterviewSessionTokenDuration: 24 * time.Hour,
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
						InterviewSessionTokenDuration: 24 * time.Hour,
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
						InterviewSessionTokenDuration: 24 * time.Hour,
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

func TestInterviewSessionService_StartInterviewSession(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()
	sessionID := uuid.New()

	testCases := []struct {
		name   string
		input  *entities.StartInterviewSessionReq
		setup  func() (*mockServices.MockResumeService, *mockMiddleware.MockAuthContext, *mockRepositories.MockResumeReposity, *mockUtils.MockGenerator, *mockRepositories.MockInterviewSessionRepository, *mockUtils.MockJwtToken, *config.Config, *mockAws.MockS3Storage, *mockRepositories.MockJobRequirementRepository, *queue.MockRedisTaskPublisher, *mockDatabase.MockRedisClient)
		verify func(t *testing.T, gotErr error)
	}{
		{
			name: "Success - Start interview session",
			input: &entities.StartInterviewSessionReq{
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
					StartInterviewSession(ctx, mock.AnythingOfType("*db.StartInterviewSessionParams")).
					Return(nil)

				config := &config.Config{}

				return mockResumeService, mockAuthContext, mockResumeRepo, mockGenerator, mockInterviewSessionRepo, mockJwtMaker, config, mockS3Storage, mockJobRequirementRepo, mockPublisher, mockRedisClient
			},
			verify: func(t *testing.T, gotErr error) {
				assert.NoError(t, gotErr)
			},
		},
		{
			name: "Error - Start interview session failure",
			input: &entities.StartInterviewSessionReq{
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
					StartInterviewSession(ctx, mock.AnythingOfType("*db.StartInterviewSessionParams")).
					Return(errors.New("database error"))

				config := &config.Config{}

				return mockResumeService, mockAuthContext, mockResumeRepo, mockGenerator, mockInterviewSessionRepo, mockJwtMaker, config, mockS3Storage, mockJobRequirementRepo, mockPublisher, mockRedisClient
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Equal(t, "database error", gotErr.Error())
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

			gotErr := svc.StartInterviewSession(ctx, tC.input)

			tC.verify(t, gotErr)
		})
	}
}
