package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.com/interview-simulation/interview-backend-server/internal/config"
	"gitlab.com/interview-simulation/interview-backend-server/internal/entities"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
	"gitlab.com/interview-simulation/interview-backend-server/internal/mocks/middleware"
	"gitlab.com/interview-simulation/interview-backend-server/internal/mocks/services"
	"gitlab.com/interview-simulation/interview-backend-server/internal/mocks/utils"
)

func TestUserHandler_HealthCheck(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log := log.Initialize("test")
	ctx := context.Background()

	tests := []struct {
		name   string
		setup  func() (*services.MockUserService, *config.Config)
		verify func(t *testing.T, w *httptest.ResponseRecorder)
	}{
		{
			name: "Success",
			setup: func() (*services.MockUserService, *config.Config) {
				mockUserService := new(services.MockUserService)
				mockConfig := &config.Config{}

				mockUserService.EXPECT().
					HealthCheck(ctx).
					Return(entities.HealthCheckResponse{Status: "ok"}, nil)

				return mockUserService, mockConfig
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, w.Code)
				assert.JSONEq(t, `{"status":"ok"}`, w.Body.String())
			},
		},
		{
			name: "ServiceError",
			setup: func() (*services.MockUserService, *config.Config) {
				mockUserService := new(services.MockUserService)
				mockConfig := &config.Config{}

				mockUserService.EXPECT().
					HealthCheck(ctx).
					Return(entities.HealthCheckResponse{}, app_error.New(errors.New("service error"), app_error.ErrCodeGeneralServerUnavailable))

				return mockUserService, mockConfig
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
			c.Request = httptest.NewRequest(http.MethodGet, "/health", nil)

			mockUserService, mockConfig := tt.setup()
			defer mockUserService.AssertExpectations(t)

			handler := NewUserHandler(log, mockUserService, mockConfig, nil, nil, nil)
			handler.HealthCheck(c)

			tt.verify(t, w)
		})
	}
}

func TestUserHandler_SignOut(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log := log.Initialize("test")
	ctx := context.Background()

	tests := []struct {
		name   string
		setup  func() (*services.MockUserService, *utils.MockCookies, *middleware.MockAuthContext)
		verify func(t *testing.T, w *httptest.ResponseRecorder)
	}{
		{
			name: "Success",
			setup: func() (*services.MockUserService, *utils.MockCookies, *middleware.MockAuthContext) {
				mockUserService := new(services.MockUserService)
				mockCookies := new(utils.MockCookies)
				mockAuthContext := new(middleware.MockAuthContext)

				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, nil)

				mockUserService.EXPECT().
					SignOut(ctx).
					Return(nil)

				mockCookies.EXPECT().
					ClearRefreshTokenCookie(mock.Anything).
					Return()

				return mockUserService, mockCookies, mockAuthContext
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNoContent, w.Code)
			},
		},
		{
			name: "AuthContextError",
			setup: func() (*services.MockUserService, *utils.MockCookies, *middleware.MockAuthContext) {
				mockUserService := new(services.MockUserService)
				mockCookies := new(utils.MockCookies)
				mockAuthContext := new(middleware.MockAuthContext)

				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, app_error.New(errors.New("invalid token"), app_error.ErrCodeAuthInvalidToken))

				return mockUserService, mockCookies, mockAuthContext
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusUnauthorized, w.Code)
				assert.JSONEq(t, `{"code":"INS0200","message":"Your token is invalid. Please log in again."}`, w.Body.String())
			},
		},
		{
			name: "ServiceError",
			setup: func() (*services.MockUserService, *utils.MockCookies, *middleware.MockAuthContext) {
				mockUserService := new(services.MockUserService)
				mockCookies := new(utils.MockCookies)
				mockAuthContext := new(middleware.MockAuthContext)

				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, nil)

				mockUserService.EXPECT().
					SignOut(ctx).
					Return(app_error.New(errors.New("session not found"), app_error.ErrCodeAuthSessionNotFound))

				return mockUserService, mockCookies, mockAuthContext
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, w.Code)
				assert.JSONEq(t, `{"code":"INS0218","message":"Your session has expired or is invalid. Please log in again."}`, w.Body.String())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPost, "/signout", nil)

			mockUserService, mockCookies, mockAuthContext := tt.setup()
			defer mockUserService.AssertExpectations(t)
			defer mockCookies.AssertExpectations(t)
			defer mockAuthContext.AssertExpectations(t)

			handler := NewUserHandler(log, mockUserService, nil, mockAuthContext, nil, mockCookies)
			handler.SignOut(c)

			tt.verify(t, w)
		})
	}
}
