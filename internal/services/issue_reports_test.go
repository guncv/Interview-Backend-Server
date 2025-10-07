package services

import (
	"context"
	"errors"
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
	mockDatabase "gitlab.com/interview-simulation/interview-backend-server/internal/mocks/database"
	mockAuthContext "gitlab.com/interview-simulation/interview-backend-server/internal/mocks/middleware"
	mockRepo "gitlab.com/interview-simulation/interview-backend-server/internal/mocks/repositories"
	mockGenerator "gitlab.com/interview-simulation/interview-backend-server/internal/mocks/utils"
	utilsPkg "gitlab.com/interview-simulation/interview-backend-server/internal/utils"
)

func TestIssueReportsService_CreateUserIssueReport(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()
	categoryID := uuid.New()
	userID := uuid.New()
	issueReportID := uuid.New()
	mockErr := errors.New("error")
	invalidUserID := "invalid-user-id"
	invalidCategoryID := "invalid-category-id"
	categoryName := "test name"
	validDescription := "test description"
	fixTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	validDBResp := &db.CreateUserIssueReportRow{
		ID:           issueReportID,
		Description:  validDescription,
		CategoryID:   categoryID,
		Status:       constants.IssueReportStatusOpen,
		Acknowledged: true,
		CommentCount: 0,
		CreatedAt:    fixTime,
		UpdatedAt:    fixTime,
	}

	validResp := &entities.UserIssueReport{
		ID:           issueReportID.String(),
		Description:  validDescription,
		CategoryID:   categoryID.String(),
		CategoryName: categoryName,
		IsEditable:   true,
		Acknowledged: true,
		CommentCount: 0,
		CreatedAt:    fixTime.Format(time.RFC3339),
		UpdatedAt:    fixTime.Format(time.RFC3339),
	}

	testCases := []struct {
		name   string
		input  *entities.CreateUserIssueReportReq
		setup  func() (*mockRepo.MockIssueReportsRepository, *mockRepo.MockIssueCategoriesRepository, *mockAuthContext.MockAuthContext, *mockGenerator.MockGenerator)
		verify func(t *testing.T, gotResp *entities.UserIssueReport, gotErr error)
	}{
		{
			name: "Success",
			input: &entities.CreateUserIssueReportReq{
				Description: validDescription,
				CategoryID:  categoryID.String(),
			},
			setup: func() (*mockRepo.MockIssueReportsRepository, *mockRepo.MockIssueCategoriesRepository, *mockAuthContext.MockAuthContext, *mockGenerator.MockGenerator) {
				mockIssueReportsRepository := new(mockRepo.MockIssueReportsRepository)
				mockIssueCategoriesRepository := new(mockRepo.MockIssueCategoriesRepository)
				mockAuthContext := new(mockAuthContext.MockAuthContext)
				mockGenerator := new(mockGenerator.MockGenerator)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							ID:     userID,
							UserID: userID.String(),
							Role:   constants.UserRoleUser,
						},
					}, nil)

				mockIssueCategoriesRepository.EXPECT().
					GetIssueCategoryIfExists(ctx, categoryID).
					Return(&db.GetIssueCategoryIfExistsRow{
						ID:   categoryID,
						Name: categoryName,
					}, nil)

				mockGenerator.EXPECT().
					GenerateUUID(ctx).
					Return(issueReportID)

				mockIssueReportsRepository.EXPECT().
					CreateUserIssueReport(ctx, mock.MatchedBy(func(p *db.CreateUserIssueReportParams) bool {
						return p.UserID.UUID == userID &&
							p.UserID.Valid &&
							p.Description == validDescription &&
							p.Status == constants.IssueReportStatusOpen &&
							p.CategoryID == categoryID &&
							p.Priority == constants.IssueReportPriorityNormal
					})).
					Return(validDBResp, nil)

				return mockIssueReportsRepository, mockIssueCategoriesRepository, mockAuthContext, mockGenerator
			},
			verify: func(t *testing.T, gotResp *entities.UserIssueReport, gotErr error) {
				assert.NoError(t, gotErr)
				assert.Equal(t, validResp, gotResp)
			},
		},
		{
			name: "Error - WithGetAuthContextError",
			input: &entities.CreateUserIssueReportReq{
				Description: validDescription,
				CategoryID:  categoryID.String(),
			},
			setup: func() (*mockRepo.MockIssueReportsRepository, *mockRepo.MockIssueCategoriesRepository, *mockAuthContext.MockAuthContext, *mockGenerator.MockGenerator) {
				mockIssueReportsRepository := new(mockRepo.MockIssueReportsRepository)
				mockIssueCategoriesRepository := new(mockRepo.MockIssueCategoriesRepository)
				mockAuthContext := new(mockAuthContext.MockAuthContext)
				mockGenerator := new(mockGenerator.MockGenerator)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(nil, mockErr)

				return mockIssueReportsRepository, mockIssueCategoriesRepository, mockAuthContext, mockGenerator
			},
			verify: func(t *testing.T, gotResp *entities.UserIssueReport, gotErr error) {
				assert.Error(t, gotErr)
				assert.ErrorIs(t, gotErr, mockErr)
				assert.Nil(t, gotResp)
			},
		},
		{
			name: "Error - WithInvalidUserID",
			input: &entities.CreateUserIssueReportReq{
				Description: validDescription,
				CategoryID:  categoryID.String(),
			},
			setup: func() (*mockRepo.MockIssueReportsRepository, *mockRepo.MockIssueCategoriesRepository, *mockAuthContext.MockAuthContext, *mockGenerator.MockGenerator) {
				mockIssueReportsRepository := new(mockRepo.MockIssueReportsRepository)
				mockIssueCategoriesRepository := new(mockRepo.MockIssueCategoriesRepository)
				mockAuthContext := new(mockAuthContext.MockAuthContext)
				mockGenerator := new(mockGenerator.MockGenerator)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							ID:     userID,
							UserID: invalidUserID,
							Role:   constants.UserRoleUser,
						},
					}, nil)

				return mockIssueReportsRepository, mockIssueCategoriesRepository, mockAuthContext, mockGenerator
			},
			verify: func(t *testing.T, gotResp *entities.UserIssueReport, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "The UUID is invalid. Please try again.")
				assert.Contains(t, gotErr.Error(), "[INS0107]")
				assert.Nil(t, gotResp)
			},
		},
		{
			name: "Error - WithInvalidCategoryID",
			input: &entities.CreateUserIssueReportReq{
				Description: validDescription,
				CategoryID:  invalidCategoryID,
			},
			setup: func() (*mockRepo.MockIssueReportsRepository, *mockRepo.MockIssueCategoriesRepository, *mockAuthContext.MockAuthContext, *mockGenerator.MockGenerator) {
				mockIssueReportsRepository := new(mockRepo.MockIssueReportsRepository)
				mockIssueCategoriesRepository := new(mockRepo.MockIssueCategoriesRepository)
				mockAuthContext := new(mockAuthContext.MockAuthContext)
				mockGenerator := new(mockGenerator.MockGenerator)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							ID:     userID,
							UserID: userID.String(),
							Role:   constants.UserRoleUser,
						},
					}, nil)

				return mockIssueReportsRepository, mockIssueCategoriesRepository, mockAuthContext, mockGenerator
			},
			verify: func(t *testing.T, gotResp *entities.UserIssueReport, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "The UUID is invalid. Please try again.")
				assert.Contains(t, gotErr.Error(), "[INS0107]")
				assert.Nil(t, gotResp)
			},
		},
		{
			name: "Error - WithCheckIssueCategoryExistsError",
			input: &entities.CreateUserIssueReportReq{
				Description: validDescription,
				CategoryID:  categoryID.String(),
			},
			setup: func() (*mockRepo.MockIssueReportsRepository, *mockRepo.MockIssueCategoriesRepository, *mockAuthContext.MockAuthContext, *mockGenerator.MockGenerator) {
				mockIssueReportsRepository := new(mockRepo.MockIssueReportsRepository)
				mockIssueCategoriesRepository := new(mockRepo.MockIssueCategoriesRepository)
				mockAuthContext := new(mockAuthContext.MockAuthContext)
				mockGenerator := new(mockGenerator.MockGenerator)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							ID:     userID,
							UserID: userID.String(),
							Role:   constants.UserRoleUser,
						},
					}, nil)

				mockIssueCategoriesRepository.EXPECT().
					GetIssueCategoryIfExists(ctx, categoryID).
					Return(&db.GetIssueCategoryIfExistsRow{
						ID:   categoryID,
						Name: categoryName,
					}, mockErr)

				return mockIssueReportsRepository, mockIssueCategoriesRepository, mockAuthContext, mockGenerator
			},
			verify: func(t *testing.T, gotResp *entities.UserIssueReport, gotErr error) {
				assert.Error(t, gotErr)
				assert.ErrorIs(t, gotErr, mockErr)
				assert.Nil(t, gotResp)
			},
		},
		{
			name: "Error - WithCreateUserIssueReportError",
			input: &entities.CreateUserIssueReportReq{
				Description: validDescription,
				CategoryID:  categoryID.String(),
			},
			setup: func() (*mockRepo.MockIssueReportsRepository, *mockRepo.MockIssueCategoriesRepository, *mockAuthContext.MockAuthContext, *mockGenerator.MockGenerator) {
				mockIssueReportsRepository := new(mockRepo.MockIssueReportsRepository)
				mockIssueCategoriesRepository := new(mockRepo.MockIssueCategoriesRepository)
				mockAuthContext := new(mockAuthContext.MockAuthContext)
				mockGenerator := new(mockGenerator.MockGenerator)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							ID:     userID,
							UserID: userID.String(),
							Role:   constants.UserRoleUser,
						},
					}, nil)

				mockIssueCategoriesRepository.EXPECT().
					GetIssueCategoryIfExists(ctx, categoryID).
					Return(&db.GetIssueCategoryIfExistsRow{
						ID:   categoryID,
						Name: categoryName,
					}, nil)

				mockGenerator.EXPECT().
					GenerateUUID(ctx).
					Return(issueReportID)

				mockIssueReportsRepository.EXPECT().
					CreateUserIssueReport(ctx, mock.MatchedBy(func(p *db.CreateUserIssueReportParams) bool {
						return p.UserID.UUID == userID &&
							p.UserID.Valid &&
							p.Description == validDescription &&
							p.Status == constants.IssueReportStatusOpen &&
							p.CategoryID == categoryID &&
							p.Priority == constants.IssueReportPriorityNormal
					})).
					Return(nil, mockErr)

				return mockIssueReportsRepository, mockIssueCategoriesRepository, mockAuthContext, mockGenerator
			},
			verify: func(t *testing.T, gotResp *entities.UserIssueReport, gotErr error) {
				assert.Error(t, gotErr)
				assert.ErrorIs(t, gotErr, mockErr)
				assert.Nil(t, gotResp)
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			mockIssueReportsRepository, mockIssueCategoriesRepository, mockAuthContext, mockGenerator := tC.setup()
			defer func() {
				if mockIssueReportsRepository != nil {
					mockIssueReportsRepository.AssertExpectations(t)
				}
				if mockIssueCategoriesRepository != nil {
					mockIssueCategoriesRepository.AssertExpectations(t)
				}
				if mockAuthContext != nil {
					mockAuthContext.AssertExpectations(t)
				}
				if mockGenerator != nil {
					mockGenerator.AssertExpectations(t)
				}
			}()

			svc := NewIssueReportsService(lgr, mockIssueReportsRepository, mockIssueCategoriesRepository, mockAuthContext, nil, mockGenerator)
			gotResp, gotErr := svc.CreateUserIssueReport(ctx, tC.input)

			tC.verify(t, gotResp, gotErr)
		})
	}
}

func TestIssueReportsService_ListIssueCategories(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()
	mockErr := errors.New("error")

	dbResp := []db.ListIssueCategoriesRow{
		{
			ID:   uuid.New(),
			Name: "test category",
		},
	}

	testCases := []struct {
		name   string
		setup  func() (*mockDatabase.MockRedisClient, *mockRepo.MockIssueCategoriesRepository)
		verify func(t *testing.T, gotResp *entities.ListIssueCategoriesResp, gotErr error)
	}{
		{
			name: "Success - WithRedisHit",
			setup: func() (*mockDatabase.MockRedisClient, *mockRepo.MockIssueCategoriesRepository) {
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockIssueCategoriesRepository := new(mockRepo.MockIssueCategoriesRepository)

				mockRedisClient.EXPECT().
					Get(ctx, constants.RedisPrefixIssueCategories).
					Return(`[{"id":"`+dbResp[0].ID.String()+`","name":"`+dbResp[0].Name+`"}]`, nil)

				mockRedisClient.EXPECT().
					Set(mock.AnythingOfType("*context.timerCtx"), mock.AnythingOfType("database.RedisPayload")).
					Return(nil).Maybe()

				return mockRedisClient, mockIssueCategoriesRepository
			},
			verify: func(t *testing.T, gotResp *entities.ListIssueCategoriesResp, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, gotResp)
				assert.Equal(t, 1, len(gotResp.Data))
				assert.Equal(t, dbResp[0].ID.String(), gotResp.Data[0].ID)
				assert.Equal(t, dbResp[0].Name, gotResp.Data[0].Name)
			},
		},
		{
			name: "Success - WithRedisMiss",
			setup: func() (*mockDatabase.MockRedisClient, *mockRepo.MockIssueCategoriesRepository) {
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockIssueCategoriesRepository := new(mockRepo.MockIssueCategoriesRepository)

				mockRedisClient.EXPECT().
					Get(ctx, constants.RedisPrefixIssueCategories).
					Return("", redis.Nil)

				mockIssueCategoriesRepository.EXPECT().
					ListIssueCategories(ctx).
					Return(dbResp, nil)

				mockRedisClient.EXPECT().
					Set(mock.AnythingOfType("*context.timerCtx"), mock.AnythingOfType("database.RedisPayload")).
					Return(nil).Maybe()

				return mockRedisClient, mockIssueCategoriesRepository
			},
			verify: func(t *testing.T, gotResp *entities.ListIssueCategoriesResp, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, gotResp)
				assert.Equal(t, 1, len(gotResp.Data))
				assert.Equal(t, dbResp[0].ID.String(), gotResp.Data[0].ID)
				assert.Equal(t, dbResp[0].Name, gotResp.Data[0].Name)
			},
		},
		{
			name: "Success - WithRedisError",
			setup: func() (*mockDatabase.MockRedisClient, *mockRepo.MockIssueCategoriesRepository) {
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockIssueCategoriesRepository := new(mockRepo.MockIssueCategoriesRepository)

				mockRedisClient.EXPECT().
					Get(ctx, constants.RedisPrefixIssueCategories).
					Return("", mockErr)

				mockIssueCategoriesRepository.EXPECT().
					ListIssueCategories(ctx).
					Return(dbResp, nil)

				mockRedisClient.EXPECT().
					Set(mock.AnythingOfType("*context.timerCtx"), mock.AnythingOfType("database.RedisPayload")).
					Return(nil).Maybe()

				return mockRedisClient, mockIssueCategoriesRepository
			},
			verify: func(t *testing.T, gotResp *entities.ListIssueCategoriesResp, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, gotResp)
				assert.Equal(t, 1, len(gotResp.Data))
				assert.Equal(t, dbResp[0].ID.String(), gotResp.Data[0].ID)
				assert.Equal(t, dbResp[0].Name, gotResp.Data[0].Name)
			},
		},
		{
			name: "Success - WithEmptyResponse",
			setup: func() (*mockDatabase.MockRedisClient, *mockRepo.MockIssueCategoriesRepository) {
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockIssueCategoriesRepository := new(mockRepo.MockIssueCategoriesRepository)

				mockRedisClient.EXPECT().
					Get(ctx, constants.RedisPrefixIssueCategories).
					Return("", redis.Nil)

				mockIssueCategoriesRepository.EXPECT().
					ListIssueCategories(ctx).
					Return([]db.ListIssueCategoriesRow{}, nil)

				mockRedisClient.EXPECT().
					Set(mock.AnythingOfType("*context.timerCtx"), mock.AnythingOfType("database.RedisPayload")).
					Return(nil).Maybe()

				return mockRedisClient, mockIssueCategoriesRepository
			},
			verify: func(t *testing.T, gotResp *entities.ListIssueCategoriesResp, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, gotResp)
				assert.Equal(t, 0, len(gotResp.Data))
			},
		},
		{
			name: "Success - WithRedisSetNewDataError",
			setup: func() (*mockDatabase.MockRedisClient, *mockRepo.MockIssueCategoriesRepository) {
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockIssueCategoriesRepository := new(mockRepo.MockIssueCategoriesRepository)

				mockRedisClient.EXPECT().
					Get(ctx, constants.RedisPrefixIssueCategories).
					Return("", redis.Nil)

				mockIssueCategoriesRepository.EXPECT().
					ListIssueCategories(ctx).
					Return(dbResp, nil)

				mockRedisClient.EXPECT().
					Set(mock.AnythingOfType("*context.timerCtx"), mock.AnythingOfType("database.RedisPayload")).
					Return(mockErr).Maybe()

				return mockRedisClient, mockIssueCategoriesRepository
			},
			verify: func(t *testing.T, gotResp *entities.ListIssueCategoriesResp, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, gotResp)
				assert.Equal(t, 1, len(gotResp.Data))
				assert.Equal(t, dbResp[0].ID.String(), gotResp.Data[0].ID)
				assert.Equal(t, dbResp[0].Name, gotResp.Data[0].Name)
			},
		},
		{
			name: "Success - WithRedisHitButUnmarshalError",
			setup: func() (*mockDatabase.MockRedisClient, *mockRepo.MockIssueCategoriesRepository) {
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockIssueCategoriesRepository := new(mockRepo.MockIssueCategoriesRepository)

				mockRedisClient.EXPECT().
					Get(ctx, constants.RedisPrefixIssueCategories).
					Return(`[{"id":"`+dbResp[0].ID.String()+`","name":"`+dbResp[0].Name+`"]`, nil)

				mockIssueCategoriesRepository.EXPECT().
					ListIssueCategories(ctx).
					Return(dbResp, nil)

				mockRedisClient.EXPECT().
					Set(mock.AnythingOfType("*context.timerCtx"), mock.AnythingOfType("database.RedisPayload")).
					Return(mockErr).Maybe()

				return mockRedisClient, mockIssueCategoriesRepository
			},
			verify: func(t *testing.T, gotResp *entities.ListIssueCategoriesResp, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, gotResp)
				assert.Equal(t, 1, len(gotResp.Data))
				assert.Equal(t, dbResp[0].ID.String(), gotResp.Data[0].ID)
				assert.Equal(t, dbResp[0].Name, gotResp.Data[0].Name)
			},
		},
		{
			name: "Error - WithListIssueCategoriesError",
			setup: func() (*mockDatabase.MockRedisClient, *mockRepo.MockIssueCategoriesRepository) {
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockIssueCategoriesRepository := new(mockRepo.MockIssueCategoriesRepository)

				mockRedisClient.EXPECT().
					Get(ctx, constants.RedisPrefixIssueCategories).
					Return("", redis.Nil)

				mockIssueCategoriesRepository.EXPECT().
					ListIssueCategories(ctx).
					Return(nil, mockErr)

				return mockRedisClient, mockIssueCategoriesRepository
			},
			verify: func(t *testing.T, gotResp *entities.ListIssueCategoriesResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.ErrorIs(t, gotErr, mockErr)
			},
		},
		{
			name: "Error - WithRedisHitButUnmarshalAndListIssueCategoriesError",
			setup: func() (*mockDatabase.MockRedisClient, *mockRepo.MockIssueCategoriesRepository) {
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockIssueCategoriesRepository := new(mockRepo.MockIssueCategoriesRepository)

				mockRedisClient.EXPECT().
					Get(ctx, constants.RedisPrefixIssueCategories).
					Return(`[{"id":"`+dbResp[0].ID.String()+`","name":"`+dbResp[0].Name+`"]`, nil)

				mockIssueCategoriesRepository.EXPECT().
					ListIssueCategories(ctx).
					Return(nil, mockErr)

				return mockRedisClient, mockIssueCategoriesRepository
			},
			verify: func(t *testing.T, gotResp *entities.ListIssueCategoriesResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.ErrorIs(t, gotErr, mockErr)
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			mockRedisClient, mockIssueCategoriesRepository := tC.setup()
			defer func() {
				if mockIssueCategoriesRepository != nil {
					mockIssueCategoriesRepository.AssertExpectations(t)
				}
				if mockRedisClient != nil {
					mockRedisClient.AssertExpectations(t)
				}
			}()

			svc := NewIssueReportsService(lgr, nil, mockIssueCategoriesRepository, nil, mockRedisClient, nil)
			gotResp, gotErr := svc.ListIssueCategories(ctx)

			tC.verify(t, gotResp, gotErr)
		})
	}
}

func TestIssueReportsService_CreateAdminIssueCategory(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()
	categoryID := uuid.New()
	userID := uuid.New()
	mockErr := errors.New("error")
	invalidUserID := "invalid-user-id"
	// invalidUserID := "invalid-user-id"

	testCases := []struct {
		name   string
		input  *entities.CreateAdminIssueCategoryReq
		setup  func() (*mockDatabase.MockRedisClient, *mockRepo.MockIssueCategoriesRepository, *mockAuthContext.MockAuthContext, *mockGenerator.MockGenerator)
		verify func(t *testing.T, gotErr error)
	}{
		{
			name: "Success",
			input: &entities.CreateAdminIssueCategoryReq{
				Name: "test category",
			},
			setup: func() (*mockDatabase.MockRedisClient, *mockRepo.MockIssueCategoriesRepository, *mockAuthContext.MockAuthContext, *mockGenerator.MockGenerator) {
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockIssueCategoriesRepository := new(mockRepo.MockIssueCategoriesRepository)
				mockAuthContext := new(mockAuthContext.MockAuthContext)
				mockGenerator := new(mockGenerator.MockGenerator)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							ID:     userID,
							UserID: userID.String(),
							Role:   constants.UserRoleAdmin,
						},
					}, nil)

				mockGenerator.EXPECT().
					GenerateUUID(ctx).
					Return(categoryID)

				mockIssueCategoriesRepository.EXPECT().
					CreateAdminIssueCategory(ctx, mock.MatchedBy(func(p *db.CreateAdminIssueCategoryParams) bool {
						return p.Name == "test category" &&
							p.CreatedBy == uuid.NullUUID{UUID: userID, Valid: true} &&
							p.ID == categoryID
					})).
					Return(nil)

				mockRedisClient.EXPECT().
					Delete(ctx, constants.RedisPrefixIssueCategories).
					Return(nil)

				return mockRedisClient, mockIssueCategoriesRepository, mockAuthContext, mockGenerator
			},
			verify: func(t *testing.T, gotErr error) {
				assert.NoError(t, gotErr)
			},
		},
		{
			name: "Error - GetAuthContextError",
			input: &entities.CreateAdminIssueCategoryReq{
				Name: "test category",
			},
			setup: func() (*mockDatabase.MockRedisClient, *mockRepo.MockIssueCategoriesRepository, *mockAuthContext.MockAuthContext, *mockGenerator.MockGenerator) {
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockIssueCategoriesRepository := new(mockRepo.MockIssueCategoriesRepository)
				mockAuthContext := new(mockAuthContext.MockAuthContext)
				mockGenerator := new(mockGenerator.MockGenerator)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(nil, mockErr)

				return mockRedisClient, mockIssueCategoriesRepository, mockAuthContext, mockGenerator
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.ErrorIs(t, gotErr, mockErr)
			},
		},
		{
			name: "Error - WithPermissionDenied",
			input: &entities.CreateAdminIssueCategoryReq{
				Name: "test category",
			},
			setup: func() (*mockDatabase.MockRedisClient, *mockRepo.MockIssueCategoriesRepository, *mockAuthContext.MockAuthContext, *mockGenerator.MockGenerator) {
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockIssueCategoriesRepository := new(mockRepo.MockIssueCategoriesRepository)
				mockAuthContext := new(mockAuthContext.MockAuthContext)
				mockGenerator := new(mockGenerator.MockGenerator)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							ID:     userID,
							UserID: userID.String(),
							Role:   constants.UserRoleUser,
						},
					}, nil)

				return mockRedisClient, mockIssueCategoriesRepository, mockAuthContext, mockGenerator
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "You are not authorized to perform this action")
				assert.Contains(t, gotErr.Error(), "[INS0112]")
			},
		},
		{
			name: "Error - WithInvalidUserID",
			input: &entities.CreateAdminIssueCategoryReq{
				Name: "test category",
			},
			setup: func() (*mockDatabase.MockRedisClient, *mockRepo.MockIssueCategoriesRepository, *mockAuthContext.MockAuthContext, *mockGenerator.MockGenerator) {
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockIssueCategoriesRepository := new(mockRepo.MockIssueCategoriesRepository)
				mockAuthContext := new(mockAuthContext.MockAuthContext)
				mockGenerator := new(mockGenerator.MockGenerator)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							ID:     userID,
							UserID: invalidUserID,
							Role:   constants.UserRoleAdmin,
						},
					}, nil)

				return mockRedisClient, mockIssueCategoriesRepository, mockAuthContext, mockGenerator
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "The UUID is invalid. Please try again.")
				assert.Contains(t, gotErr.Error(), "[INS0107]")
			},
		},
		{
			name: "Error - WithCreateAdminIssueCategoryError",
			input: &entities.CreateAdminIssueCategoryReq{
				Name: "test category",
			},
			setup: func() (*mockDatabase.MockRedisClient, *mockRepo.MockIssueCategoriesRepository, *mockAuthContext.MockAuthContext, *mockGenerator.MockGenerator) {
				mockRedisClient := new(mockDatabase.MockRedisClient)
				mockIssueCategoriesRepository := new(mockRepo.MockIssueCategoriesRepository)
				mockAuthContext := new(mockAuthContext.MockAuthContext)
				mockGenerator := new(mockGenerator.MockGenerator)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							ID:     userID,
							UserID: userID.String(),
							Role:   constants.UserRoleAdmin,
						},
					}, nil)

				mockGenerator.EXPECT().
					GenerateUUID(ctx).
					Return(categoryID)

				mockIssueCategoriesRepository.EXPECT().
					CreateAdminIssueCategory(ctx, mock.MatchedBy(func(p *db.CreateAdminIssueCategoryParams) bool {
						return p.Name == "test category" &&
							p.CreatedBy == uuid.NullUUID{UUID: userID, Valid: true} &&
							p.ID == categoryID
					})).
					Return(mockErr)

				return mockRedisClient, mockIssueCategoriesRepository, mockAuthContext, mockGenerator
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.ErrorIs(t, gotErr, mockErr)
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			mockRedisClient, mockIssueCategoriesRepository, mockAuthContext, mockGenerator := tC.setup()
			defer func() {
				if mockIssueCategoriesRepository != nil {
					mockIssueCategoriesRepository.AssertExpectations(t)
				}

				if mockAuthContext != nil {
					mockAuthContext.AssertExpectations(t)
				}
				if mockGenerator != nil {
					mockGenerator.AssertExpectations(t)
				}

				if mockRedisClient != nil {
					mockRedisClient.AssertExpectations(t)
				}
			}()

			svc := NewIssueReportsService(lgr, nil, mockIssueCategoriesRepository, mockAuthContext, mockRedisClient, mockGenerator)
			gotErr := svc.CreateAdminIssueCategory(ctx, tC.input)

			tC.verify(t, gotErr)
		})
	}
}
