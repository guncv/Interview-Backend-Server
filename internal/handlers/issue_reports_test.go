package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.com/interview-simulation/interview-backend-server/internal/entities"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
	"gitlab.com/interview-simulation/interview-backend-server/internal/mocks/middleware"
	"gitlab.com/interview-simulation/interview-backend-server/internal/mocks/services"
	"gitlab.com/interview-simulation/interview-backend-server/internal/mocks/utils"
)

func TestIssueReportsHandler_CreateUserIssueReport(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log := log.Initialize("test")
	ctx := context.Background()
	mockErr := errors.New("mock error")
	invalidCategoryID := "invalid-category-id"

	categoryID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	issueReportID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")

	resp := &entities.UserIssueReport{
		ID:           issueReportID.String(),
		Description:  "test description",
		CategoryID:   categoryID.String(),
		CategoryName: "test category name",
		IsEditable:   true,
		Acknowledged: true,
		CommentCount: 1,
		CreatedAt:    "test created at",
		UpdatedAt:    "test updated at",
	}

	tests := []struct {
		name   string
		input  func() *entities.CreateUserIssueReportReq
		setup  func() (*services.MockIssueReportsService, *utils.MockValidator, *middleware.MockAuthContext)
		verify func(t *testing.T, w *httptest.ResponseRecorder)
	}{
		{
			name: "Success",
			input: func() *entities.CreateUserIssueReportReq {
				return &entities.CreateUserIssueReportReq{
					Description: "test description",
					CategoryID:  "test category id",
				}
			},
			setup: func() (*services.MockIssueReportsService, *utils.MockValidator, *middleware.MockAuthContext) {
				mockIssueReportsService := new(services.MockIssueReportsService)
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)

				mockValidator.EXPECT().
					ValidateAndBind(mock.Anything, mock.Anything, "CreateUserIssueReport").
					Return(nil)

				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, nil)

				mockIssueReportsService.EXPECT().
					CreateUserIssueReport(ctx, mock.Anything).
					Return(resp, nil)

				return mockIssueReportsService, mockValidator, mockAuthContext
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, w.Code)
				assert.JSONEq(t, `{"id":"`+resp.ID+`","description":"`+resp.Description+`","category_id":"`+resp.CategoryID+`","category_name":"`+resp.CategoryName+`","is_editable":true,"acknowledged":true,"comment_count":1,"created_at":"`+resp.CreatedAt+`","updated_at":"`+resp.UpdatedAt+`"}`, w.Body.String())
			},
		},
		{
			name: "Error - WithValidationError",
			input: func() *entities.CreateUserIssueReportReq {
				return &entities.CreateUserIssueReportReq{
					Description: "test description",
					CategoryID:  invalidCategoryID,
				}
			},
			setup: func() (*services.MockIssueReportsService, *utils.MockValidator, *middleware.MockAuthContext) {
				mockIssueReportsService := new(services.MockIssueReportsService)
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)

				mockValidator.EXPECT().
					ValidateAndBind(mock.Anything, mock.Anything, "CreateUserIssueReport").
					Return(app_error.New(mockErr, app_error.ErrCodeAuthInvalidRequest))

				return mockIssueReportsService, mockValidator, mockAuthContext
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, w.Code)
			},
		},
		{
			name: "Error - WithExtractAuthContextError",
			input: func() *entities.CreateUserIssueReportReq {
				return &entities.CreateUserIssueReportReq{
					Description: "test description",
					CategoryID:  "test category id",
				}
			},
			setup: func() (*services.MockIssueReportsService, *utils.MockValidator, *middleware.MockAuthContext) {
				mockIssueReportsService := new(services.MockIssueReportsService)
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)

				mockValidator.EXPECT().
					ValidateAndBind(mock.Anything, mock.Anything, "CreateUserIssueReport").
					Return(nil)

				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, app_error.New(mockErr, app_error.ErrCodeAuthInvalidHeader))

				return mockIssueReportsService, mockValidator, mockAuthContext
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusUnauthorized, w.Code)
			},
		},
		{
			name: "Error - WithCreateUserIssueReportError",
			input: func() *entities.CreateUserIssueReportReq {
				return &entities.CreateUserIssueReportReq{
					Description: "test description",
					CategoryID:  "test category id",
				}
			},
			setup: func() (*services.MockIssueReportsService, *utils.MockValidator, *middleware.MockAuthContext) {
				mockIssueReportsService := new(services.MockIssueReportsService)
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)

				mockValidator.EXPECT().
					ValidateAndBind(mock.Anything, mock.Anything, "CreateUserIssueReport").
					Return(nil)

				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, nil)

				mockIssueReportsService.EXPECT().
					CreateUserIssueReport(ctx, mock.Anything).
					Return(nil, mockErr)

				return mockIssueReportsService, mockValidator, mockAuthContext
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, w.Code)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.input())
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPost, "/issue-reports", bytes.NewBuffer(body))
			c.Request.Header.Set("Content-Type", "application/json")

			mockIssueReportsService, mockValidator, mockAuthContext := tt.setup()
			defer func() {
				if mockIssueReportsService != nil {
					mockIssueReportsService.AssertExpectations(t)
				}
				if mockValidator != nil {
					mockValidator.AssertExpectations(t)
				}
				if mockAuthContext != nil {
					mockAuthContext.AssertExpectations(t)
				}
			}()

			handler := NewIssueReportsHandler(log, mockIssueReportsService, mockAuthContext, mockValidator)
			handler.CreateUserIssueReport(c)

			tt.verify(t, w)
		})
	}
}

func TestIssueReportsHandler_ListUserIssueReports(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log := log.Initialize("test")
	ctx := context.Background()
	mockErr := errors.New("mock error")

	resp := &entities.ListUserIssueReportsResp{
		Data: []entities.UserIssueReport{
			{
				ID:           uuid.New().String(),
				Description:  "test description",
				CategoryID:   uuid.New().String(),
				CategoryName: "test category name",
				IsEditable:   true,
				Acknowledged: true,
				CommentCount: 1,
				CreatedAt:    "test created at",
				UpdatedAt:    "test updated at",
			},
		},
	}

	tests := []struct {
		name   string
		setup  func() (*services.MockIssueReportsService, *utils.MockValidator, *middleware.MockAuthContext)
		verify func(t *testing.T, w *httptest.ResponseRecorder)
	}{
		{
			name: "Success",
			setup: func() (*services.MockIssueReportsService, *utils.MockValidator, *middleware.MockAuthContext) {
				mockIssueReportsService := new(services.MockIssueReportsService)
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)

				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, nil)

				mockIssueReportsService.EXPECT().
					ListUserIssueReports(ctx).
					Return(resp, nil)

				return mockIssueReportsService, mockValidator, mockAuthContext
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, w.Code)
				assert.JSONEq(t, `{"data":[{"id":"`+resp.Data[0].ID+`","description":"`+resp.Data[0].Description+`","category_id":"`+resp.Data[0].CategoryID+`","category_name":"`+resp.Data[0].CategoryName+`","is_editable":true,"acknowledged":true,"comment_count":1,"created_at":"`+resp.Data[0].CreatedAt+`","updated_at":"`+resp.Data[0].UpdatedAt+`"}]}`, w.Body.String())
			},
		},
		{
			name: "Error - WithExtractAuthContextError",
			setup: func() (*services.MockIssueReportsService, *utils.MockValidator, *middleware.MockAuthContext) {
				mockIssueReportsService := new(services.MockIssueReportsService)
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)

				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, app_error.New(mockErr, app_error.ErrCodeAuthInvalidHeader))

				return mockIssueReportsService, mockValidator, mockAuthContext
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusUnauthorized, w.Code)
			},
		},
		{
			name: "Error - WithListUserIssueReportsError",
			setup: func() (*services.MockIssueReportsService, *utils.MockValidator, *middleware.MockAuthContext) {
				mockIssueReportsService := new(services.MockIssueReportsService)
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)

				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, nil)

				mockIssueReportsService.EXPECT().
					ListUserIssueReports(ctx).
					Return(nil, mockErr)

				return mockIssueReportsService, mockValidator, mockAuthContext
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, w.Code)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/issue-reports", nil)
			c.Request.Header.Set("Content-Type", "application/json")

			mockIssueReportsService, mockValidator, mockAuthContext := tt.setup()
			defer func() {
				if mockIssueReportsService != nil {
					mockIssueReportsService.AssertExpectations(t)
				}
				if mockValidator != nil {
					mockValidator.AssertExpectations(t)
				}
				if mockAuthContext != nil {
					mockAuthContext.AssertExpectations(t)
				}
			}()

			handler := NewIssueReportsHandler(log, mockIssueReportsService, mockAuthContext, mockValidator)
			handler.ListUserIssueReports(c)

			tt.verify(t, w)
		})
	}
}

func TestIssueReportsHandler_UpdateUserIssueReportByID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log := log.Initialize("test")
	ctx := context.Background()
	mockErr := errors.New("mock error")
	invalidReportID := "invalid-report-id"

	categoryID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	issueReportID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")

	resp := &entities.UserIssueReport{
		ID:           issueReportID.String(),
		Description:  "test description",
		CategoryID:   categoryID.String(),
		CategoryName: "test category name",
		IsEditable:   true,
		Acknowledged: true,
		CommentCount: 1,
		CreatedAt:    "test created at",
		UpdatedAt:    "test updated at",
	}

	tests := []struct {
		name     string
		reportID string
		input    func() *entities.UpdateUserIssueReportByIDReq
		setup    func() (*services.MockIssueReportsService, *utils.MockValidator, *middleware.MockAuthContext)
		verify   func(t *testing.T, w *httptest.ResponseRecorder)
	}{
		{
			name:     "Success",
			reportID: issueReportID.String(),
			input: func() *entities.UpdateUserIssueReportByIDReq {
				return &entities.UpdateUserIssueReportByIDReq{
					Description: "updated test description",
					CategoryID:  uuid.New().String(),
				}
			},
			setup: func() (*services.MockIssueReportsService, *utils.MockValidator, *middleware.MockAuthContext) {
				mockIssueReportsService := new(services.MockIssueReportsService)
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)
				realValidator := validator.New()

				mockValidator.EXPECT().
					GetValidate().
					Return(realValidator).
					Once()

				mockValidator.EXPECT().
					ValidateAndBind(mock.Anything, mock.Anything, "UpdateUserIssueReportByID").
					Return(nil).
					Once()

				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, nil).
					Once()

				mockIssueReportsService.EXPECT().
					UpdateUserIssueReportByID(ctx, mock.Anything, issueReportID.String()).
					Return(resp, nil).
					Once()

				return mockIssueReportsService, mockValidator, mockAuthContext
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, w.Code)
				assert.JSONEq(t, `{"id":"`+resp.ID+`","description":"`+resp.Description+`","category_id":"`+resp.CategoryID+`","category_name":"`+resp.CategoryName+`","is_editable":true,"acknowledged":true,"comment_count":1,"created_at":"`+resp.CreatedAt+`","updated_at":"`+resp.UpdatedAt+`"}`, w.Body.String())
			},
		},
		{
			name:     "Error - Missing Report ID",
			reportID: "",
			input: func() *entities.UpdateUserIssueReportByIDReq {
				return &entities.UpdateUserIssueReportByIDReq{
					Description: "updated test description",
					CategoryID:  uuid.New().String(),
				}
			},
			setup: func() (*services.MockIssueReportsService, *utils.MockValidator, *middleware.MockAuthContext) {
				mockIssueReportsService := new(services.MockIssueReportsService)
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)

				return mockIssueReportsService, mockValidator, mockAuthContext
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, w.Code)
			},
		},
		{
			name:     "Error - Invalid Report ID Format",
			reportID: invalidReportID,
			input: func() *entities.UpdateUserIssueReportByIDReq {
				return &entities.UpdateUserIssueReportByIDReq{
					Description: "updated test description",
					CategoryID:  uuid.New().String(),
				}
			},
			setup: func() (*services.MockIssueReportsService, *utils.MockValidator, *middleware.MockAuthContext) {
				mockIssueReportsService := new(services.MockIssueReportsService)
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)
				realValidator := validator.New()

				mockValidator.EXPECT().
					GetValidate().
					Return(realValidator).
					Once()

				return mockIssueReportsService, mockValidator, mockAuthContext
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, w.Code)
			},
		},
		{
			name:     "Error - Validation Error",
			reportID: issueReportID.String(),
			input: func() *entities.UpdateUserIssueReportByIDReq {
				return &entities.UpdateUserIssueReportByIDReq{
					Description: "",
					CategoryID:  "invalid-category-id",
				}
			},
			setup: func() (*services.MockIssueReportsService, *utils.MockValidator, *middleware.MockAuthContext) {
				mockIssueReportsService := new(services.MockIssueReportsService)
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)
				realValidator := validator.New()

				mockValidator.EXPECT().
					GetValidate().
					Return(realValidator).
					Once()

				mockValidator.EXPECT().
					ValidateAndBind(mock.Anything, mock.Anything, "UpdateUserIssueReportByID").
					Return(app_error.New(mockErr, app_error.ErrCodeAuthInvalidRequest)).
					Once()

				return mockIssueReportsService, mockValidator, mockAuthContext
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, w.Code)
			},
		},
		{
			name:     "Error - Extract Auth Context Error",
			reportID: issueReportID.String(),
			input: func() *entities.UpdateUserIssueReportByIDReq {
				return &entities.UpdateUserIssueReportByIDReq{
					Description: "updated test description",
					CategoryID:  uuid.New().String(),
				}
			},
			setup: func() (*services.MockIssueReportsService, *utils.MockValidator, *middleware.MockAuthContext) {
				mockIssueReportsService := new(services.MockIssueReportsService)
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)
				realValidator := validator.New()

				mockValidator.EXPECT().
					GetValidate().
					Return(realValidator).
					Once()

				mockValidator.EXPECT().
					ValidateAndBind(mock.Anything, mock.Anything, "UpdateUserIssueReportByID").
					Return(nil).
					Once()

				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, app_error.New(mockErr, app_error.ErrCodeAuthInvalidHeader)).
					Once()

				return mockIssueReportsService, mockValidator, mockAuthContext
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusUnauthorized, w.Code)
			},
		},
		{
			name:     "Error - Service Update Error",
			reportID: issueReportID.String(),
			input: func() *entities.UpdateUserIssueReportByIDReq {
				return &entities.UpdateUserIssueReportByIDReq{
					Description: "updated test description",
					CategoryID:  uuid.New().String(),
				}
			},
			setup: func() (*services.MockIssueReportsService, *utils.MockValidator, *middleware.MockAuthContext) {
				mockIssueReportsService := new(services.MockIssueReportsService)
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)
				realValidator := validator.New()

				mockValidator.EXPECT().
					GetValidate().
					Return(realValidator).
					Once()

				mockValidator.EXPECT().
					ValidateAndBind(mock.Anything, mock.Anything, "UpdateUserIssueReportByID").
					Return(nil).
					Once()

				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, nil).
					Once()

				mockIssueReportsService.EXPECT().
					UpdateUserIssueReportByID(ctx, mock.Anything, issueReportID.String()).
					Return(nil, app_error.New(mockErr, app_error.ErrCodeIssueReportNotFound)).
					Once()

				return mockIssueReportsService, mockValidator, mockAuthContext
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, w.Code)
			},
		},
		{
			name:     "Error - Service Internal Error",
			reportID: issueReportID.String(),
			input: func() *entities.UpdateUserIssueReportByIDReq {
				return &entities.UpdateUserIssueReportByIDReq{
					Description: "updated test description",
					CategoryID:  uuid.New().String(),
				}
			},
			setup: func() (*services.MockIssueReportsService, *utils.MockValidator, *middleware.MockAuthContext) {
				mockIssueReportsService := new(services.MockIssueReportsService)
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)
				realValidator := validator.New()

				mockValidator.EXPECT().
					GetValidate().
					Return(realValidator).
					Once()

				mockValidator.EXPECT().
					ValidateAndBind(mock.Anything, mock.Anything, "UpdateUserIssueReportByID").
					Return(nil).
					Once()

				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, nil).
					Once()

				mockIssueReportsService.EXPECT().
					UpdateUserIssueReportByID(ctx, mock.Anything, issueReportID.String()).
					Return(nil, mockErr).
					Once()

				return mockIssueReportsService, mockValidator, mockAuthContext
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, w.Code)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.input())
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPatch, "/issue-reports/"+tt.reportID, bytes.NewBuffer(body))
			c.Request.Header.Set("Content-Type", "application/json")
			c.Params = []gin.Param{{Key: "issue_report_id", Value: tt.reportID}}

			mockIssueReportsService, mockValidator, mockAuthContext := tt.setup()
			defer func() {
				if mockIssueReportsService != nil {
					mockIssueReportsService.AssertExpectations(t)
				}
				if mockValidator != nil {
					mockValidator.AssertExpectations(t)
				}
				if mockAuthContext != nil {
					mockAuthContext.AssertExpectations(t)
				}
			}()

			handler := NewIssueReportsHandler(log, mockIssueReportsService, mockAuthContext, mockValidator)
			handler.UpdateUserIssueReportByID(c)

			tt.verify(t, w)
		})
	}
}

func TestIssueReportsHandler_ListIssueCategories(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log := log.Initialize("test")
	ctx := context.Background()
	mockErr := errors.New("mock error")

	resp := &entities.ListIssueCategoriesResp{
		Data: []entities.IssueCategory{
			{
				ID:   uuid.New().String(),
				Name: "test category name",
			},
		},
	}

	tests := []struct {
		name   string
		setup  func() *services.MockIssueReportsService
		verify func(t *testing.T, w *httptest.ResponseRecorder)
	}{
		{
			name: "Success",
			setup: func() *services.MockIssueReportsService {
				mockIssueReportsService := new(services.MockIssueReportsService)

				mockIssueReportsService.EXPECT().
					ListIssueCategories(ctx).
					Return(resp, nil)

				return mockIssueReportsService
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, w.Code)
				assert.JSONEq(t, `{"data":[{"id":"`+resp.Data[0].ID+`","name":"`+resp.Data[0].Name+`"}]}`, w.Body.String())
			},
		},
		{
			name: "Error - WithListIssueCategoriesError",
			setup: func() *services.MockIssueReportsService {
				mockIssueReportsService := new(services.MockIssueReportsService)

				mockIssueReportsService.EXPECT().
					ListIssueCategories(ctx).
					Return(nil, mockErr)

				return mockIssueReportsService
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, w.Code)

			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/issue-categories", nil)
			c.Request.Header.Set("Content-Type", "application/json")

			mockIssueReportsService := tt.setup()
			defer func() {
				if mockIssueReportsService != nil {
					mockIssueReportsService.AssertExpectations(t)
				}
			}()

			handler := NewIssueReportsHandler(log, mockIssueReportsService, nil, nil)
			handler.ListIssueCategories(c)

			tt.verify(t, w)
		})
	}
}

func TestIssueReportsHandler_CreateAdminIssueCategory(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log := log.Initialize("test")
	ctx := context.Background()
	mockValidationErr := app_error.New(errors.New("validation error"), app_error.ErrCodeAuthInvalidRequest)
	mockAuthErr := app_error.New(errors.New("auth error"), app_error.ErrCodeAuthInvalidHeader)
	mockServiceErr := errors.New("service error")

	req := &entities.CreateAdminIssueCategoryReq{
		Name: "test category",
	}

	tests := []struct {
		name   string
		input  *entities.CreateAdminIssueCategoryReq
		setup  func() (*services.MockIssueReportsService, *utils.MockValidator, *middleware.MockAuthContext)
		verify func(t *testing.T, w *httptest.ResponseRecorder)
	}{
		{
			name:  "Success",
			input: req,
			setup: func() (*services.MockIssueReportsService, *utils.MockValidator, *middleware.MockAuthContext) {
				mockIssueReportsService := new(services.MockIssueReportsService)
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)

				mockValidator.EXPECT().
					ValidateAndBind(mock.Anything, mock.Anything, "CreateAdminIssueCategory").
					Return(nil)

				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, nil)

				mockIssueReportsService.EXPECT().
					CreateAdminIssueCategory(ctx, mock.Anything).
					Return(nil)

				return mockIssueReportsService, mockValidator, mockAuthContext
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNoContent, w.Code)
			},
		},
		{
			name:  "Error - WithValidationError",
			input: req,
			setup: func() (*services.MockIssueReportsService, *utils.MockValidator, *middleware.MockAuthContext) {
				mockIssueReportsService := new(services.MockIssueReportsService)
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)

				mockValidator.EXPECT().
					ValidateAndBind(mock.Anything, mock.Anything, "CreateAdminIssueCategory").
					Return(mockValidationErr)

				return mockIssueReportsService, mockValidator, mockAuthContext
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, w.Code)
			},
		},
		{
			name:  "Error - WithExtractAuthContextError",
			input: req,
			setup: func() (*services.MockIssueReportsService, *utils.MockValidator, *middleware.MockAuthContext) {
				mockIssueReportsService := new(services.MockIssueReportsService)
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)

				mockValidator.EXPECT().
					ValidateAndBind(mock.Anything, mock.Anything, "CreateAdminIssueCategory").
					Return(nil)

				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, mockAuthErr)

				return mockIssueReportsService, mockValidator, mockAuthContext
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusUnauthorized, w.Code)
			},
		},
		{
			name:  "Error - WithCreateAdminIssueCategoryError",
			input: req,
			setup: func() (*services.MockIssueReportsService, *utils.MockValidator, *middleware.MockAuthContext) {
				mockIssueReportsService := new(services.MockIssueReportsService)
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)

				mockValidator.EXPECT().
					ValidateAndBind(mock.Anything, mock.Anything, "CreateAdminIssueCategory").
					Return(nil)

				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, nil)

				mockIssueReportsService.EXPECT().
					CreateAdminIssueCategory(ctx, mock.Anything).
					Return(mockServiceErr)

				return mockIssueReportsService, mockValidator, mockAuthContext
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, w.Code)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.input)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPost, "/issue-categories", bytes.NewReader(body))
			c.Request.Header.Set("Content-Type", "application/json")

			mockIssueReportsService, mockValidator, mockAuthContext := tt.setup()
			defer func() {
				if mockIssueReportsService != nil {
					mockIssueReportsService.AssertExpectations(t)
				}
				if mockValidator != nil {
					mockValidator.AssertExpectations(t)
				}
				if mockAuthContext != nil {
					mockAuthContext.AssertExpectations(t)
				}
			}()

			handler := NewIssueReportsHandler(log, mockIssueReportsService, mockAuthContext, mockValidator)
			handler.CreateAdminIssueCategory(c)

			tt.verify(t, w)
		})
	}
}
