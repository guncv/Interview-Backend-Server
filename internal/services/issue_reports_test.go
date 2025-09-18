package services

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	db "gitlab.com/interview-simulation/interview-backend-server/internal/db/sqlc"
	"gitlab.com/interview-simulation/interview-backend-server/internal/entities"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
	"gitlab.com/interview-simulation/interview-backend-server/internal/middleware"
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

	testCases := []struct {
		name   string
		input  *entities.CreateUserIssueReportReq
		setup  func() (*mockRepo.MockIssueReportsRepository, *mockRepo.MockIssueCategoriesRepository, *mockAuthContext.MockAuthContext, *mockGenerator.MockGenerator)
		verify func(t *testing.T, gotErr error)
	}{
		{
			name: "Success",
			input: &entities.CreateUserIssueReportReq{
				Description: "test description",
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
					CheckIssueCategoryExists(ctx, categoryID).
					Return(true, nil)

				mockGenerator.EXPECT().
					GenerateUUID(ctx).
					Return(issueReportID)

				mockIssueReportsRepository.EXPECT().
					CreateUserIssueReport(ctx, mock.MatchedBy(func(p *db.CreateUserIssueReportParams) bool {
						return p.UserID.UUID == userID &&
							p.UserID.Valid &&
							p.Description == "test description" &&
							p.Status == constants.IssueReportStatusOpen &&
							p.CategoryID == categoryID &&
							p.Priority == constants.IssueReportPriorityNormal
					})).
					Return(nil)

				return mockIssueReportsRepository, mockIssueCategoriesRepository, mockAuthContext, mockGenerator
			},
			verify: func(t *testing.T, gotErr error) {
				assert.NoError(t, gotErr)
			},
		},
		{
			name: "Error - WithGetAuthContextError",
			input: &entities.CreateUserIssueReportReq{
				Description: "test description",
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
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.ErrorIs(t, gotErr, mockErr)
			},
		},
		{
			name: "Error - WithInvalidUserID",
			input: &entities.CreateUserIssueReportReq{
				Description: "test description",
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
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "The UUID is invalid. Please try again.")
				assert.Contains(t, gotErr.Error(), "[ONX0107]")
			},
		},
		{
			name: "Error - WithInvalidCategoryID",
			input: &entities.CreateUserIssueReportReq{
				Description: "test description",
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
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "The UUID is invalid. Please try again.")
				assert.Contains(t, gotErr.Error(), "[ONX0107]")
			},
		},
		{
			name: "Error - WithCheckIssueCategoryExistsError",
			input: &entities.CreateUserIssueReportReq{
				Description: "test description",
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
					CheckIssueCategoryExists(ctx, categoryID).
					Return(false, mockErr)

				return mockIssueReportsRepository, mockIssueCategoriesRepository, mockAuthContext, mockGenerator
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.ErrorIs(t, gotErr, mockErr)
			},
		},
		{
			name: "Error - WithCategoryNotFound",
			input: &entities.CreateUserIssueReportReq{
				Description: "test description",
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
					CheckIssueCategoryExists(ctx, categoryID).
					Return(false, nil)

				return mockIssueReportsRepository, mockIssueCategoriesRepository, mockAuthContext, mockGenerator
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "The issue category was not found. Please try again.")
				assert.Contains(t, gotErr.Error(), "[ONX0800]")
			},
		},
		{
			name: "Error - WithCreateUserIssueReportError",
			input: &entities.CreateUserIssueReportReq{
				Description: "test description",
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
					CheckIssueCategoryExists(ctx, categoryID).
					Return(true, nil)

				mockGenerator.EXPECT().
					GenerateUUID(ctx).
					Return(issueReportID)

				mockIssueReportsRepository.EXPECT().
					CreateUserIssueReport(ctx, mock.MatchedBy(func(p *db.CreateUserIssueReportParams) bool {
						return p.UserID.UUID == userID &&
							p.UserID.Valid &&
							p.Description == "test description" &&
							p.Status == constants.IssueReportStatusOpen &&
							p.CategoryID == categoryID &&
							p.Priority == constants.IssueReportPriorityNormal
					})).
					Return(mockErr)

				return mockIssueReportsRepository, mockIssueCategoriesRepository, mockAuthContext, mockGenerator
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.ErrorIs(t, gotErr, mockErr)
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

			svc := NewIssueReportsService(lgr, mockIssueReportsRepository, mockIssueCategoriesRepository, mockAuthContext, mockGenerator)
			gotErr := svc.CreateUserIssueReport(ctx, tC.input)

			tC.verify(t, gotErr)
		})
	}
}

func TestIssueReportsService_ListUserIssueReports(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()
	userID := uuid.New()
	mockErr := errors.New("error")
	invalidUserID := "invalid-user-id"

	dbResp := []db.ListUserIssueReportsRow{
		{
			ID:           uuid.New(),
			Description:  "test description",
			CategoryID:   uuid.New(),
			CategoryName: sql.NullString{String: "test category", Valid: true},
			Status:       constants.IssueReportStatusOpen,
			Acknowledged: false,
			CommentCount: 0,
			CreatedAt:    time.Time{},
			UpdatedAt:    time.Time{},
		},
	}

	testCases := []struct {
		name   string
		setup  func() (*mockRepo.MockIssueReportsRepository, *mockAuthContext.MockAuthContext)
		verify func(t *testing.T, gotResp *entities.ListUserIssueReportsResp, gotErr error)
	}{
		{
			name: "Success",
			setup: func() (*mockRepo.MockIssueReportsRepository, *mockAuthContext.MockAuthContext) {
				mockIssueReportsRepository := new(mockRepo.MockIssueReportsRepository)
				mockAuthContext := new(mockAuthContext.MockAuthContext)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							ID:     userID,
							UserID: userID.String(),
							Role:   constants.UserRoleUser,
						},
					}, nil)

				mockIssueReportsRepository.EXPECT().
					ListUserIssueReports(ctx, userID).
					Return(dbResp, nil)

				return mockIssueReportsRepository, mockAuthContext
			},
			verify: func(t *testing.T, gotResp *entities.ListUserIssueReportsResp, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, gotResp)
				assert.Equal(t, 1, len(gotResp.Data))
				assert.Equal(t, dbResp[0].ID, uuid.MustParse(gotResp.Data[0].ID))
				assert.Equal(t, dbResp[0].Description, gotResp.Data[0].Description)
				assert.Equal(t, dbResp[0].CategoryID, uuid.MustParse(gotResp.Data[0].CategoryID))
				assert.Equal(t, dbResp[0].CategoryName.String, gotResp.Data[0].CategoryName)
				assert.Equal(t, true, gotResp.Data[0].IsEditable)
				assert.Equal(t, dbResp[0].Acknowledged, gotResp.Data[0].Acknowledged)
			},
		},
		{
			name: "Error - WithGetAuthContextError",
			setup: func() (*mockRepo.MockIssueReportsRepository, *mockAuthContext.MockAuthContext) {
				mockIssueReportsRepository := new(mockRepo.MockIssueReportsRepository)
				mockAuthContext := new(mockAuthContext.MockAuthContext)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(nil, mockErr)

				return mockIssueReportsRepository, mockAuthContext
			},
			verify: func(t *testing.T, gotResp *entities.ListUserIssueReportsResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.ErrorIs(t, gotErr, mockErr)
			},
		},
		{
			name: "Error - WithInvalidUserID",
			setup: func() (*mockRepo.MockIssueReportsRepository, *mockAuthContext.MockAuthContext) {
				mockIssueReportsRepository := new(mockRepo.MockIssueReportsRepository)
				mockAuthContext := new(mockAuthContext.MockAuthContext)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							ID:     userID,
							UserID: invalidUserID,
							Role:   constants.UserRoleUser,
						},
					}, nil)

				return mockIssueReportsRepository, mockAuthContext
			},
			verify: func(t *testing.T, gotResp *entities.ListUserIssueReportsResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "The UUID is invalid. Please try again.")
				assert.Contains(t, gotErr.Error(), "[ONX0107]")
			},
		},
		{
			name: "Error - WithListUserIssueReportsError",
			setup: func() (*mockRepo.MockIssueReportsRepository, *mockAuthContext.MockAuthContext) {
				mockIssueReportsRepository := new(mockRepo.MockIssueReportsRepository)
				mockAuthContext := new(mockAuthContext.MockAuthContext)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(&middleware.AuthPayload{
						Payload: &utilsPkg.SignInTokenPayload{
							ID:     userID,
							UserID: userID.String(),
							Role:   constants.UserRoleUser,
						},
					}, nil)

				mockIssueReportsRepository.EXPECT().
					ListUserIssueReports(ctx, userID).
					Return([]db.ListUserIssueReportsRow{}, mockErr)

				return mockIssueReportsRepository, mockAuthContext
			},
			verify: func(t *testing.T, gotResp *entities.ListUserIssueReportsResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.ErrorIs(t, gotErr, mockErr)
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			mockIssueReportsRepository, mockAuthContext := tC.setup()
			defer func() {
				if mockIssueReportsRepository != nil {
					mockIssueReportsRepository.AssertExpectations(t)
				}
				if mockAuthContext != nil {
					mockAuthContext.AssertExpectations(t)
				}
			}()

			svc := NewIssueReportsService(lgr, mockIssueReportsRepository, nil, mockAuthContext, nil)
			gotResp, gotErr := svc.ListUserIssueReports(ctx)

			tC.verify(t, gotResp, gotErr)
		})
	}
}

func TestIssueReportsService_UpdateUserIssueReportByID(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()
	categoryID := uuid.New()
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000000")
	otherUserID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	issueReportID := uuid.New()
	mockErr := errors.New("error")
	invalidUserID := "invalid-user-id"
	invalidCategoryID := "invalid-category-id"
	invalidIssueReportID := "invalid-issue-report-id"

	testCases := []struct {
		name          string
		input         *entities.UpdateUserIssueReportByIDReq
		issueReportID string
		setup         func() (*mockRepo.MockIssueReportsRepository, *mockRepo.MockIssueCategoriesRepository, *mockAuthContext.MockAuthContext, *mockGenerator.MockGenerator)
		verify        func(t *testing.T, gotErr error)
	}{
		{
			name: "Success",
			input: &entities.UpdateUserIssueReportByIDReq{
				Description: "test description",
				CategoryID:  categoryID.String(),
			},
			issueReportID: issueReportID.String(),
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
					CheckIssueCategoryExists(ctx, categoryID).
					Return(true, nil)

				mockIssueReportsRepository.EXPECT().
					GetUserIssueReportUserIDAndStatusByID(ctx, issueReportID).
					Return(&db.GetUserIssueReportUserIDAndStatusByIDRow{
						UserID: uuid.NullUUID{UUID: userID, Valid: true},
						Status: constants.IssueReportStatusOpen,
					}, nil)

				mockIssueReportsRepository.EXPECT().
					UpdateUserIssueReportByID(ctx, mock.MatchedBy(func(p *db.UpdateUserIssueReportByIDParams) bool {
						return p.ID == issueReportID &&
							p.Description == "test description" &&
							p.CategoryID == categoryID
					})).
					Return(nil)

				return mockIssueReportsRepository, mockIssueCategoriesRepository, mockAuthContext, mockGenerator
			},
			verify: func(t *testing.T, gotErr error) {
				assert.NoError(t, gotErr)
			},
		},
		{
			name: "Error - WithGetAuthContextError",
			input: &entities.UpdateUserIssueReportByIDReq{
				Description: "test description",
				CategoryID:  categoryID.String(),
			},
			issueReportID: issueReportID.String(),
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
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.ErrorIs(t, gotErr, mockErr)
			},
		},
		{
			name: "Error - WithInvalidUserID",
			input: &entities.UpdateUserIssueReportByIDReq{
				Description: "test description",
				CategoryID:  categoryID.String(),
			},
			issueReportID: issueReportID.String(),
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
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "The UUID is invalid. Please try again.")
				assert.Contains(t, gotErr.Error(), "[ONX0107]")
			},
		},
		{
			name: "Error - WithInvalidIssueReportID",
			input: &entities.UpdateUserIssueReportByIDReq{
				Description: "test description",
				CategoryID:  categoryID.String(),
			},
			issueReportID: invalidIssueReportID,
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
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "The UUID is invalid. Please try again.")
				assert.Contains(t, gotErr.Error(), "[ONX0107]")
			},
		},
		{
			name: "Error - WithInvalidCategoryID",
			input: &entities.UpdateUserIssueReportByIDReq{
				Description: "test description",
				CategoryID:  invalidCategoryID,
			},
			issueReportID: issueReportID.String(),
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
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "The UUID is invalid. Please try again.")
				assert.Contains(t, gotErr.Error(), "[ONX0107]")
			},
		},
		{
			name: "Error - WithCheckIssueCategoryExistsError",
			input: &entities.UpdateUserIssueReportByIDReq{
				Description: "test description",
				CategoryID:  categoryID.String(),
			},
			issueReportID: issueReportID.String(),
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
					CheckIssueCategoryExists(ctx, categoryID).
					Return(false, mockErr)

				return mockIssueReportsRepository, mockIssueCategoriesRepository, mockAuthContext, mockGenerator
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.ErrorIs(t, gotErr, mockErr)
			},
		},
		{
			name: "Error - WithCheckIssueCategoryExistsDoesNotExist",
			input: &entities.UpdateUserIssueReportByIDReq{
				Description: "test description",
				CategoryID:  categoryID.String(),
			},
			issueReportID: issueReportID.String(),
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
					CheckIssueCategoryExists(ctx, categoryID).
					Return(false, nil)

				return mockIssueReportsRepository, mockIssueCategoriesRepository, mockAuthContext, mockGenerator
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "The issue category was not found. Please try again.")
				assert.Contains(t, gotErr.Error(), "[ONX0800]")
			},
		},
		{
			name: "Error - WithGetUserIssueReportUserIDAndStatusByIDError",
			input: &entities.UpdateUserIssueReportByIDReq{
				Description: "test description",
				CategoryID:  categoryID.String(),
			},
			issueReportID: issueReportID.String(),
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
					CheckIssueCategoryExists(ctx, categoryID).
					Return(true, nil)

				mockIssueReportsRepository.EXPECT().
					GetUserIssueReportUserIDAndStatusByID(ctx, issueReportID).
					Return(nil, mockErr)

				return mockIssueReportsRepository, mockIssueCategoriesRepository, mockAuthContext, mockGenerator
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.ErrorIs(t, gotErr, mockErr)
			},
		},
		{
			name: "Error - WithUserIDMismatch",
			input: &entities.UpdateUserIssueReportByIDReq{
				Description: "test description",
				CategoryID:  categoryID.String(),
			},
			issueReportID: issueReportID.String(),
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
					CheckIssueCategoryExists(ctx, categoryID).
					Return(true, nil)

				mockIssueReportsRepository.EXPECT().
					GetUserIssueReportUserIDAndStatusByID(ctx, issueReportID).
					Return(&db.GetUserIssueReportUserIDAndStatusByIDRow{
						UserID: uuid.NullUUID{UUID: otherUserID, Valid: true},
						Status: constants.IssueReportStatusOpen,
					}, nil)

				return mockIssueReportsRepository, mockIssueCategoriesRepository, mockAuthContext, mockGenerator
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "this user is not the owner of the issue report")
				assert.Contains(t, gotErr.Error(), "[ONX0702]")
			},
		},
		{
			name: "Error - WithIssueReportNotOpen",
			input: &entities.UpdateUserIssueReportByIDReq{
				Description: "test description",
				CategoryID:  categoryID.String(),
			},
			issueReportID: issueReportID.String(),
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
					CheckIssueCategoryExists(ctx, categoryID).
					Return(true, nil)

				mockIssueReportsRepository.EXPECT().
					GetUserIssueReportUserIDAndStatusByID(ctx, issueReportID).
					Return(&db.GetUserIssueReportUserIDAndStatusByIDRow{
						UserID: uuid.NullUUID{UUID: userID, Valid: true},
						Status: constants.IssueReportStatusClosed,
					}, nil)

				return mockIssueReportsRepository, mockIssueCategoriesRepository, mockAuthContext, mockGenerator
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "issue report not open")
				assert.Contains(t, gotErr.Error(), "[ONX0701]")
			},
		},
		{
			name: "Error - WithUpdateUserIssueReportByIDError",
			input: &entities.UpdateUserIssueReportByIDReq{
				Description: "test description",
				CategoryID:  categoryID.String(),
			},
			issueReportID: issueReportID.String(),
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
					CheckIssueCategoryExists(ctx, categoryID).
					Return(true, nil)

				mockIssueReportsRepository.EXPECT().
					GetUserIssueReportUserIDAndStatusByID(ctx, issueReportID).
					Return(&db.GetUserIssueReportUserIDAndStatusByIDRow{
						UserID: uuid.NullUUID{UUID: userID, Valid: true},
						Status: constants.IssueReportStatusOpen,
					}, nil)

				mockIssueReportsRepository.EXPECT().
					UpdateUserIssueReportByID(ctx, mock.MatchedBy(func(p *db.UpdateUserIssueReportByIDParams) bool {
						return p.ID == issueReportID &&
							p.Description == "test description" &&
							p.CategoryID == categoryID
					})).
					Return(mockErr)

				return mockIssueReportsRepository, mockIssueCategoriesRepository, mockAuthContext, mockGenerator
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.ErrorIs(t, gotErr, mockErr)
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

			svc := NewIssueReportsService(lgr, mockIssueReportsRepository, mockIssueCategoriesRepository, mockAuthContext, mockGenerator)
			gotErr := svc.UpdateUserIssueReportByID(ctx, tC.input, tC.issueReportID)

			tC.verify(t, gotErr)
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
		setup  func() *mockRepo.MockIssueCategoriesRepository
		verify func(t *testing.T, gotResp *entities.ListIssueCategoriesResp, gotErr error)
	}{
		{
			name: "Success",
			setup: func() *mockRepo.MockIssueCategoriesRepository {
				mockIssueCategoriesRepository := new(mockRepo.MockIssueCategoriesRepository)

				mockIssueCategoriesRepository.EXPECT().
					ListIssueCategories(ctx).
					Return(dbResp, nil)

				return mockIssueCategoriesRepository
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
			setup: func() *mockRepo.MockIssueCategoriesRepository {
				mockIssueCategoriesRepository := new(mockRepo.MockIssueCategoriesRepository)

				mockIssueCategoriesRepository.EXPECT().
					ListIssueCategories(ctx).
					Return([]db.ListIssueCategoriesRow{}, nil)

				return mockIssueCategoriesRepository
			},
			verify: func(t *testing.T, gotResp *entities.ListIssueCategoriesResp, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, gotResp)
				assert.Equal(t, 0, len(gotResp.Data))
			},
		},
		{
			name: "Error - WithListIssueCategoriesError",
			setup: func() *mockRepo.MockIssueCategoriesRepository {
				mockIssueCategoriesRepository := new(mockRepo.MockIssueCategoriesRepository)

				mockIssueCategoriesRepository.EXPECT().
					ListIssueCategories(ctx).
					Return(nil, mockErr)

				return mockIssueCategoriesRepository
			},
			verify: func(t *testing.T, gotResp *entities.ListIssueCategoriesResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.ErrorIs(t, gotErr, mockErr)
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			mockIssueCategoriesRepository := tC.setup()
			defer func() {
				if mockIssueCategoriesRepository != nil {
					mockIssueCategoriesRepository.AssertExpectations(t)
				}
			}()

			svc := NewIssueReportsService(lgr, nil, mockIssueCategoriesRepository, nil, nil)
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
		setup  func() (*mockRepo.MockIssueCategoriesRepository, *mockAuthContext.MockAuthContext, *mockGenerator.MockGenerator)
		verify func(t *testing.T, gotErr error)
	}{
		{
			name: "Success",
			input: &entities.CreateAdminIssueCategoryReq{
				Name: "test category",
			},
			setup: func() (*mockRepo.MockIssueCategoriesRepository, *mockAuthContext.MockAuthContext, *mockGenerator.MockGenerator) {
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
							p.CreatedBy == userID &&
							p.ID == categoryID
					})).
					Return(nil)

				return mockIssueCategoriesRepository, mockAuthContext, mockGenerator
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
			setup: func() (*mockRepo.MockIssueCategoriesRepository, *mockAuthContext.MockAuthContext, *mockGenerator.MockGenerator) {
				mockIssueCategoriesRepository := new(mockRepo.MockIssueCategoriesRepository)
				mockAuthContext := new(mockAuthContext.MockAuthContext)
				mockGenerator := new(mockGenerator.MockGenerator)

				mockAuthContext.EXPECT().
					GetAuthContext(ctx).
					Return(nil, mockErr)

				return mockIssueCategoriesRepository, mockAuthContext, mockGenerator
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
			setup: func() (*mockRepo.MockIssueCategoriesRepository, *mockAuthContext.MockAuthContext, *mockGenerator.MockGenerator) {
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

				return mockIssueCategoriesRepository, mockAuthContext, mockGenerator
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "You are not authorized to perform this action")
				assert.Contains(t, gotErr.Error(), "[ONX0111]")
			},
		},
		{
			name: "Error - WithInvalidUserID",
			input: &entities.CreateAdminIssueCategoryReq{
				Name: "test category",
			},
			setup: func() (*mockRepo.MockIssueCategoriesRepository, *mockAuthContext.MockAuthContext, *mockGenerator.MockGenerator) {
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

				return mockIssueCategoriesRepository, mockAuthContext, mockGenerator
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "The UUID is invalid. Please try again.")
				assert.Contains(t, gotErr.Error(), "[ONX0107]")
			},
		},
		{
			name: "Error - WithCreateAdminIssueCategoryError",
			input: &entities.CreateAdminIssueCategoryReq{
				Name: "test category",
			},
			setup: func() (*mockRepo.MockIssueCategoriesRepository, *mockAuthContext.MockAuthContext, *mockGenerator.MockGenerator) {
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
							p.CreatedBy == userID &&
							p.ID == categoryID
					})).
					Return(mockErr)

				return mockIssueCategoriesRepository, mockAuthContext, mockGenerator
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.ErrorIs(t, gotErr, mockErr)
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			mockIssueCategoriesRepository, mockAuthContext, mockGenerator := tC.setup()
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
			}()

			svc := NewIssueReportsService(lgr, nil, mockIssueCategoriesRepository, mockAuthContext, mockGenerator)
			gotErr := svc.CreateAdminIssueCategory(ctx, tC.input)

			tC.verify(t, gotErr)
		})
	}
}
