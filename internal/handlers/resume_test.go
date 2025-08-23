package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.com/interview-simulation/interview-backend-server/internal/entities"
	app_error "gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
	"gitlab.com/interview-simulation/interview-backend-server/internal/mocks/middleware"
	"gitlab.com/interview-simulation/interview-backend-server/internal/mocks/services"
	"gitlab.com/interview-simulation/interview-backend-server/internal/mocks/utils"
)

func TestResumeHandler_ListResume(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log := log.Initialize("test")
	ctx := context.Background()
	err := errors.New("mock error")

	tests := []struct {
		name        string
		queryParams string
		setup       func() (*services.MockResumeService, *utils.MockValidator, *middleware.MockAuthContext)
		verify      func(t *testing.T, w *httptest.ResponseRecorder)
	}{
		{
			name:        "Success - No query params",
			queryParams: "",
			setup: func() (*services.MockResumeService, *utils.MockValidator, *middleware.MockAuthContext) {
				mockResumeService := new(services.MockResumeService)
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)

				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, nil)

				mockResumeService.EXPECT().
					ListResume(ctx, &entities.ListResumeRequest{}).
					Return(&entities.ListResumeResponse{
						Count:         0,
						ResumeContent: nil,
					}, nil)

				return mockResumeService, mockValidator, mockAuthContext
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, w.Code)
				assert.JSONEq(t, `{"count":0,"last_updated_at":null,"resume_content":null}`, w.Body.String())
			},
		},
		{
			name:        "Success - With updated_at query param",
			queryParams: "?updated_at=2023-01-01T00:00:00Z",
			setup: func() (*services.MockResumeService, *utils.MockValidator, *middleware.MockAuthContext) {
				mockResumeService := new(services.MockResumeService)
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)

				expectedTime, _ := time.Parse(time.RFC3339, "2023-01-01T00:00:00Z")
				expectedReq := &entities.ListResumeRequest{
					UpdatedAt: &expectedTime,
				}

				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, nil)

				mockResumeService.EXPECT().
					ListResume(ctx, expectedReq).
					Return(&entities.ListResumeResponse{
						Count: 1,
						ResumeContent: &entities.ResumeContent{
							DefaultResume: entities.GetListResumeByIdResponse{
								ID:        "",
								FileName:  "",
								MimeType:  "",
								ByteSize:  0,
								CreatedAt: "",
								UpdatedAt: "",
							},
							Resumes: nil,
						},
					}, nil)

				return mockResumeService, mockValidator, mockAuthContext
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, w.Code)
				expectedResponse := `{"count":1,"last_updated_at":null,"resume_content":{"default_resume":{"id":"","file_name":"","mime_type":"","byte_size":0,"created_at":"","updated_at":""},"resumes":null}}`
				assert.JSONEq(t, expectedResponse, w.Body.String())
			},
		},
		{
			name:        "Error - Invalid updated_at format",
			queryParams: "?updated_at=invalid-date",
			setup: func() (*services.MockResumeService, *utils.MockValidator, *middleware.MockAuthContext) {
				mockResumeService := new(services.MockResumeService)
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)

				return mockResumeService, mockValidator, mockAuthContext
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, w.Code)
				assert.Contains(t, w.Body.String(), "The request is invalid. Please try again.")
			},
		},
		{
			name:        "Error With Auth Context",
			queryParams: "",
			setup: func() (*services.MockResumeService, *utils.MockValidator, *middleware.MockAuthContext) {
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)
				mockResumeService := new(services.MockResumeService)

				authErr := app_error.New(err, app_error.ErrCodeAuthInvalidHeader)
				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, authErr)

				return mockResumeService, mockValidator, mockAuthContext
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusUnauthorized, w.Code)
				assert.Contains(t, w.Body.String(), "Please log in to continue")
			},
		},
		{
			name:        "Error With Service",
			queryParams: "",
			setup: func() (*services.MockResumeService, *utils.MockValidator, *middleware.MockAuthContext) {
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)
				mockResumeService := new(services.MockResumeService)

				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, nil)

				serviceErr := app_error.New(err, app_error.ErrCodeResumeNotFound)
				mockResumeService.EXPECT().
					ListResume(ctx, &entities.ListResumeRequest{}).
					Return(nil, serviceErr)

				return mockResumeService, mockValidator, mockAuthContext
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, w.Code)
				assert.Contains(t, w.Body.String(), "The resume was not found")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			// Use GET method with query parameters instead of POST with body
			url := "/resumes/" + tt.queryParams
			c.Request = httptest.NewRequest(http.MethodGet, url, nil)

			mockResumeService, mockValidator, mockAuthContext := tt.setup()
			defer mockResumeService.AssertExpectations(t)
			defer mockValidator.AssertExpectations(t)
			defer mockAuthContext.AssertExpectations(t)

			handler := NewResumeHandler(mockResumeService, log, mockAuthContext, mockValidator)
			handler.ListResume(c)

			tt.verify(t, w)
		})
	}
}

func TestResumeHandler_SwitchDefaultResume(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log := log.Initialize("test")
	ctx := context.Background()
	err := errors.New("mock error")

	tests := []struct {
		name   string
		input  func() *entities.SwitchDefaultResumeRequest
		setup  func() (*services.MockResumeService, *utils.MockValidator, *middleware.MockAuthContext)
		verify func(t *testing.T, w *httptest.ResponseRecorder)
	}{
		{
			name: "Success",
			input: func() *entities.SwitchDefaultResumeRequest {
				return &entities.SwitchDefaultResumeRequest{
					ResumeID: "123e4567-e89b-12d3-a456-426614174000",
				}
			},
			setup: func() (*services.MockResumeService, *utils.MockValidator, *middleware.MockAuthContext) {
				mockResumeService := new(services.MockResumeService)
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)

				mockValidator.EXPECT().
					ValidateAndBind(mock.Anything, mock.Anything, "SwitchDefaultResume").
					Return(nil)

				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, nil)

				mockResumeService.EXPECT().
					SwitchDefaultResume(ctx, mock.Anything).
					Return(nil)

				return mockResumeService, mockValidator, mockAuthContext
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, w.Code)
				assert.Equal(t, "null", w.Body.String())
			},
		},
		{
			name: "Error With Validation",
			input: func() *entities.SwitchDefaultResumeRequest {
				return &entities.SwitchDefaultResumeRequest{}
			},
			setup: func() (*services.MockResumeService, *utils.MockValidator, *middleware.MockAuthContext) {
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)
				mockResumeService := new(services.MockResumeService)

				validationErr := app_error.New(err, app_error.ErrCodeAuthInvalidRequest)
				mockValidator.EXPECT().
					ValidateAndBind(mock.Anything, mock.Anything, "SwitchDefaultResume").
					Return(validationErr)

				return mockResumeService, mockValidator, mockAuthContext
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, w.Code)
				assert.Contains(t, w.Body.String(), "Something went wrong with the request")
			},
		},
		{
			name: "Error With Auth Context",
			input: func() *entities.SwitchDefaultResumeRequest {
				return &entities.SwitchDefaultResumeRequest{}
			},
			setup: func() (*services.MockResumeService, *utils.MockValidator, *middleware.MockAuthContext) {
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)
				mockResumeService := new(services.MockResumeService)

				mockValidator.EXPECT().
					ValidateAndBind(mock.Anything, mock.Anything, "SwitchDefaultResume").
					Return(nil)

				authErr := app_error.New(err, app_error.ErrCodeAuthInvalidHeader)
				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, authErr)

				return mockResumeService, mockValidator, mockAuthContext
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusUnauthorized, w.Code)
				assert.Contains(t, w.Body.String(), "Please log in to continue")
			},
		},
		{
			name: "Error With Service",
			input: func() *entities.SwitchDefaultResumeRequest {
				return &entities.SwitchDefaultResumeRequest{}
			},
			setup: func() (*services.MockResumeService, *utils.MockValidator, *middleware.MockAuthContext) {
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)
				mockResumeService := new(services.MockResumeService)

				mockValidator.EXPECT().
					ValidateAndBind(mock.Anything, mock.Anything, "SwitchDefaultResume").
					Return(nil)

				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, nil)

				serviceErr := app_error.New(err, app_error.ErrCodeResumeNotFound)
				mockResumeService.EXPECT().
					SwitchDefaultResume(ctx, mock.Anything).
					Return(serviceErr)

				return mockResumeService, mockValidator, mockAuthContext
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, w.Code)
				assert.Contains(t, w.Body.String(), "The resume was not found")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.input())
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPost, "/resumes/switch-default", bytes.NewBuffer(body))
			c.Request.Header.Set("Content-Type", "application/json")

			mockResumeService, mockValidator, mockAuthContext := tt.setup()
			defer mockResumeService.AssertExpectations(t)
			defer mockValidator.AssertExpectations(t)
			defer mockAuthContext.AssertExpectations(t)

			handler := NewResumeHandler(mockResumeService, log, mockAuthContext, mockValidator)
			handler.SwitchDefaultResume(c)

			tt.verify(t, w)
		})
	}
}

func TestResumeHandler_GetResumeByID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log := log.Initialize("test")
	ctx := context.Background()
	err := errors.New("mock error")
	realValidator := validator.New()

	tests := []struct {
		name   string
		input  string
		setup  func() (*services.MockResumeService, *utils.MockValidator, *middleware.MockAuthContext)
		verify func(t *testing.T, w *httptest.ResponseRecorder)
	}{
		{
			name:  "Success",
			input: "123e4567-e89b-12d3-a456-426614174000",
			setup: func() (*services.MockResumeService, *utils.MockValidator, *middleware.MockAuthContext) {
				mockResumeService := new(services.MockResumeService)
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)

				mockValidator.EXPECT().
					GetValidate().
					Return(realValidator)

				mockResumeService.EXPECT().
					GetResumeByID(ctx, mock.Anything).
					Return(&entities.GetResumeByIDResponse{
						ID:        "123e4567-e89b-12d3-a456-426614174000",
						FileName:  "test-resume.pdf",
						MimeType:  "application/pdf",
						ByteSize:  1024,
						FileUrl:   "https://example.com/resume.pdf",
						CreatedAt: "2024-01-01T00:00:00Z",
						UpdatedAt: "2024-01-01T00:00:00Z",
					}, nil)

				return mockResumeService, mockValidator, mockAuthContext
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, w.Code)
				assert.Contains(t, w.Body.String(), "123e4567-e89b-12d3-a456-426614174000")
			},
		},
		{
			name:  "Error With Empty Resume ID",
			input: "",
			setup: func() (*services.MockResumeService, *utils.MockValidator, *middleware.MockAuthContext) {
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)
				mockResumeService := new(services.MockResumeService)

				return mockResumeService, mockValidator, mockAuthContext
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, w.Code)
				assert.Contains(t, w.Body.String(), "The request is invalid")
			},
		},
		{
			name:  "Error With Invalid UUID",
			input: "invalid-uuid",
			setup: func() (*services.MockResumeService, *utils.MockValidator, *middleware.MockAuthContext) {
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)
				mockResumeService := new(services.MockResumeService)

				mockValidator.EXPECT().
					GetValidate().
					Return(realValidator)

				return mockResumeService, mockValidator, mockAuthContext
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, w.Code)
				assert.Contains(t, w.Body.String(), "The resume ID is invalid")
			},
		},
		{
			name:  "Error With Service",
			input: "123e4567-e89b-12d3-a456-426614174000",
			setup: func() (*services.MockResumeService, *utils.MockValidator, *middleware.MockAuthContext) {
				mockResumeService := new(services.MockResumeService)
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)

				mockValidator.EXPECT().
					GetValidate().
					Return(realValidator)

				serviceErr := app_error.New(err, app_error.ErrCodeResumeNotFound)
				mockResumeService.EXPECT().
					GetResumeByID(ctx, mock.Anything).
					Return(nil, serviceErr)

				return mockResumeService, mockValidator, mockAuthContext
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, w.Code)
				assert.Contains(t, w.Body.String(), "The resume was not found")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/resumes/"+tt.input, nil)
			c.Params = gin.Params{{Key: "id", Value: tt.input}}

			mockResumeService, mockValidator, mockAuthContext := tt.setup()
			defer mockResumeService.AssertExpectations(t)
			defer mockValidator.AssertExpectations(t)
			defer mockAuthContext.AssertExpectations(t)

			handler := NewResumeHandler(mockResumeService, log, mockAuthContext, mockValidator)
			handler.GetResumeByID(c)

			tt.verify(t, w)
		})
	}
}
