package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.com/interview-simulation/interview-backend-server/internal/entities"
	app_error "gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
	"gitlab.com/interview-simulation/interview-backend-server/internal/mocks/middleware"
	"gitlab.com/interview-simulation/interview-backend-server/internal/mocks/services"
	"gitlab.com/interview-simulation/interview-backend-server/internal/mocks/utils"
)

func TestReviewCommentHandler_CreateReviewComment(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log := log.Initialize("test")
	ctx := context.Background()
	mockErr := errors.New("mock error")
	comment := "Great job!"

	validReq := &entities.CreateReviewCommentReq{
		SessionID: "123e4567-e89b-12d3-a456-426614174000",
		Rating:    5,
		Comment:   &comment,
	}

	tests := []struct {
		name           string
		req            *entities.CreateReviewCommentReq
		setup          func() (*utils.MockValidator, *middleware.MockAuthContext, *services.MockReviewCommentService)
		verify         func(t *testing.T, w *httptest.ResponseRecorder)
		expectedStatus int
	}{
		{
			name: "Success",
			req:  validReq,
			setup: func() (*utils.MockValidator, *middleware.MockAuthContext, *services.MockReviewCommentService) {
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)
				mockReviewCommentService := new(services.MockReviewCommentService)

				mockValidator.EXPECT().
					ValidateAndBind(mock.Anything, mock.Anything, mock.Anything).
					Run(func(c *gin.Context, req interface{}, handlerName string) {
						if reqPtr, ok := req.(*entities.CreateReviewCommentReq); ok {
							*reqPtr = *validReq
						}
					}).
					Return(nil)

				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, nil)

				mockReviewCommentService.EXPECT().
					CreateReviewComment(ctx, validReq).
					Return(nil)

				return mockValidator, mockAuthContext, mockReviewCommentService
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNoContent, w.Code)
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name: "Error WithValidation",
			req:  &entities.CreateReviewCommentReq{},
			setup: func() (*utils.MockValidator, *middleware.MockAuthContext, *services.MockReviewCommentService) {
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)
				mockReviewCommentService := new(services.MockReviewCommentService)

				validationErr := app_error.New(mockErr, app_error.ErrCodeAuthInvalidRequest)
				mockValidator.EXPECT().
					ValidateAndBind(mock.Anything, mock.Anything, mock.Anything).
					Return(validationErr)

				return mockValidator, mockAuthContext, mockReviewCommentService
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, w.Code)
				assert.Contains(t, w.Body.String(), "Something went wrong with the request")
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Error WithExtractAuthContextError",
			req:  validReq,
			setup: func() (*utils.MockValidator, *middleware.MockAuthContext, *services.MockReviewCommentService) {
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)
				mockReviewCommentService := new(services.MockReviewCommentService)

				mockValidator.EXPECT().
					ValidateAndBind(mock.Anything, mock.Anything, mock.Anything).
					Run(func(c *gin.Context, req interface{}, handlerName string) {
						if reqPtr, ok := req.(*entities.CreateReviewCommentReq); ok {
							*reqPtr = *validReq
						}
					}).
					Return(nil)

				authErr := app_error.New(mockErr, app_error.ErrCodeAuthInvalidHeader)
				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, authErr)

				return mockValidator, mockAuthContext, mockReviewCommentService
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusUnauthorized, w.Code)
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "Error WithCreateReviewCommentError",
			req:  validReq,
			setup: func() (*utils.MockValidator, *middleware.MockAuthContext, *services.MockReviewCommentService) {
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)
				mockReviewCommentService := new(services.MockReviewCommentService)

				mockValidator.EXPECT().
					ValidateAndBind(mock.Anything, mock.Anything, mock.Anything).
					Run(func(c *gin.Context, req interface{}, handlerName string) {
						if reqPtr, ok := req.(*entities.CreateReviewCommentReq); ok {
							*reqPtr = *validReq
						}
					}).
					Return(nil)

				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, nil)

				mockReviewCommentService.EXPECT().
					CreateReviewComment(ctx, validReq).
					Return(mockErr)

				return mockValidator, mockAuthContext, mockReviewCommentService
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, w.Code)
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			url := "/api/v1/review-comments"

			c.Request = httptest.NewRequest(http.MethodPost, url, nil)
			c.Request.Header.Set("Content-Type", "application/json")
			body, _ := json.Marshal(tt.req)
			c.Request.Body = io.NopCloser(bytes.NewReader(body))

			mockValidator, mockAuthContext, mockReviewCommentService := tt.setup()
			defer mockValidator.AssertExpectations(t)
			defer mockAuthContext.AssertExpectations(t)
			defer mockReviewCommentService.AssertExpectations(t)

			handler := NewReviewCommentHandler(log, mockReviewCommentService, mockAuthContext, mockValidator)
			handler.CreateReviewComment(c)

			tt.verify(t, w)
		})
	}
}
