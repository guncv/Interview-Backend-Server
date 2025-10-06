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

func TestUserHandler_GetGoogleAuthURL(t *testing.T) {
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
					GetGoogleAuthURL(ctx).
					Return(&entities.GoogleAuthURLResponse{
						AuthURL: "https://accounts.google.com/oauth/authorize?client_id=test&redirect_uri=callback&scope=email&response_type=code&state=random_state",
					}, nil)

				return mockUserService, mockConfig
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, w.Code)
				assert.JSONEq(t, `{"auth_url":"https://accounts.google.com/oauth/authorize?client_id=test&redirect_uri=callback&scope=email&response_type=code&state=random_state"}`, w.Body.String())
			},
		},
		{
			name: "ServiceError",
			setup: func() (*services.MockUserService, *config.Config) {
				mockUserService := new(services.MockUserService)
				mockConfig := &config.Config{}

				mockUserService.EXPECT().
					GetGoogleAuthURL(ctx).
					Return(nil, app_error.New(errors.New("failed to generate auth URL"), app_error.ErrCodeGeneralServerUnavailable))

				return mockUserService, mockConfig
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, w.Code)
			},
		},
		{
			name: "AuthConfigurationError",
			setup: func() (*services.MockUserService, *config.Config) {
				mockUserService := new(services.MockUserService)
				mockConfig := &config.Config{}

				mockUserService.EXPECT().
					GetGoogleAuthURL(ctx).
					Return(nil, app_error.New(errors.New("invalid OAuth configuration"), app_error.ErrCodeAuthInvalidRequest))

				return mockUserService, mockConfig
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, w.Code)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/auth/google/url", nil)

			mockUserService, mockConfig := tt.setup()
			defer mockUserService.AssertExpectations(t)

			handler := NewUserHandler(log, mockUserService, mockConfig, nil, nil, nil)
			handler.GetGoogleAuthURL(c)

			tt.verify(t, w)
		})
	}
}

func TestUserHandler_HandleGoogleCallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log := log.Initialize("test")
	ctx := context.Background()

	tests := []struct {
		name   string
		setup  func() (*services.MockUserService, *utils.MockCookies, *config.Config)
		url    string
		verify func(t *testing.T, w *httptest.ResponseRecorder)
	}{
		{
			name: "Success",
			setup: func() (*services.MockUserService, *utils.MockCookies, *config.Config) {
				mockUserService := new(services.MockUserService)
				mockCookies := new(utils.MockCookies)
				mockConfig := &config.Config{}

				mockUserService.EXPECT().
					HandleGoogleCallback(ctx, &entities.HandleGoogleCallbackReq{
						Code:  "test_auth_code",
						State: "test_state",
					}).
					Return(&entities.HandleGoogleCallbackResp{
						AccessToken:  "test_access_token",
						RefreshToken: "test_refresh_token",
					}, nil)

				mockCookies.EXPECT().
					SetRefreshTokenCookie(mock.Anything, "test_refresh_token").
					Return()

				return mockUserService, mockCookies, mockConfig
			},
			url: "/auth/google/callback?code=test_auth_code&state=test_state",
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, w.Code)
				assert.JSONEq(t, `{"access_token":"test_access_token","refresh_token":"test_refresh_token"}`, w.Body.String())
			},
		},
		{
			name: "MissingCode",
			setup: func() (*services.MockUserService, *utils.MockCookies, *config.Config) {
				mockUserService := new(services.MockUserService)
				mockCookies := new(utils.MockCookies)
				mockConfig := &config.Config{}

				return mockUserService, mockCookies, mockConfig
			},
			url: "/auth/google/callback?state=test_state",
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, w.Code)
				assert.JSONEq(t, `{"code":"INS0213","message":"Something went wrong with the request. Please try again."}`, w.Body.String())
			},
		},
		{
			name: "MissingState",
			setup: func() (*services.MockUserService, *utils.MockCookies, *config.Config) {
				mockUserService := new(services.MockUserService)
				mockCookies := new(utils.MockCookies)
				mockConfig := &config.Config{}

				return mockUserService, mockCookies, mockConfig
			},
			url: "/auth/google/callback?code=test_auth_code",
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, w.Code)
				assert.JSONEq(t, `{"code":"INS0213","message":"Something went wrong with the request. Please try again."}`, w.Body.String())
			},
		},
		{
			name: "MissingBothCodeAndState",
			setup: func() (*services.MockUserService, *utils.MockCookies, *config.Config) {
				mockUserService := new(services.MockUserService)
				mockCookies := new(utils.MockCookies)
				mockConfig := &config.Config{}

				return mockUserService, mockCookies, mockConfig
			},
			url: "/auth/google/callback",
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, w.Code)
				assert.JSONEq(t, `{"code":"INS0213","message":"Something went wrong with the request. Please try again."}`, w.Body.String())
			},
		},
		{
			name: "EmptyCode",
			setup: func() (*services.MockUserService, *utils.MockCookies, *config.Config) {
				mockUserService := new(services.MockUserService)
				mockCookies := new(utils.MockCookies)
				mockConfig := &config.Config{}

				return mockUserService, mockCookies, mockConfig
			},
			url: "/auth/google/callback?code=&state=test_state",
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, w.Code)
				assert.JSONEq(t, `{"code":"INS0213","message":"Something went wrong with the request. Please try again."}`, w.Body.String())
			},
		},
		{
			name: "EmptyState",
			setup: func() (*services.MockUserService, *utils.MockCookies, *config.Config) {
				mockUserService := new(services.MockUserService)
				mockCookies := new(utils.MockCookies)
				mockConfig := &config.Config{}

				return mockUserService, mockCookies, mockConfig
			},
			url: "/auth/google/callback?code=test_auth_code&state=",
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, w.Code)
				assert.JSONEq(t, `{"code":"INS0213","message":"Something went wrong with the request. Please try again."}`, w.Body.String())
			},
		},
		{
			name: "ServiceError",
			setup: func() (*services.MockUserService, *utils.MockCookies, *config.Config) {
				mockUserService := new(services.MockUserService)
				mockCookies := new(utils.MockCookies)
				mockConfig := &config.Config{}

				mockUserService.EXPECT().
					HandleGoogleCallback(ctx, &entities.HandleGoogleCallbackReq{
						Code:  "test_auth_code",
						State: "test_state",
					}).
					Return(nil, app_error.New(errors.New("invalid authorization code"), app_error.ErrCodeAuthInvalidRequest))

				return mockUserService, mockCookies, mockConfig
			},
			url: "/auth/google/callback?code=test_auth_code&state=test_state",
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, w.Code)
			},
		},
		{
			name: "OAuthTokenExchangeError",
			setup: func() (*services.MockUserService, *utils.MockCookies, *config.Config) {
				mockUserService := new(services.MockUserService)
				mockCookies := new(utils.MockCookies)
				mockConfig := &config.Config{}

				mockUserService.EXPECT().
					HandleGoogleCallback(ctx, &entities.HandleGoogleCallbackReq{
						Code:  "test_auth_code",
						State: "test_state",
					}).
					Return(nil, app_error.New(errors.New("failed to exchange code for token"), app_error.ErrCodeGeneralServerUnavailable))

				return mockUserService, mockCookies, mockConfig
			},
			url: "/auth/google/callback?code=test_auth_code&state=test_state",
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, w.Code)
			},
		},
		{
			name: "UserCreationError",
			setup: func() (*services.MockUserService, *utils.MockCookies, *config.Config) {
				mockUserService := new(services.MockUserService)
				mockCookies := new(utils.MockCookies)
				mockConfig := &config.Config{}

				mockUserService.EXPECT().
					HandleGoogleCallback(ctx, &entities.HandleGoogleCallbackReq{
						Code:  "test_auth_code",
						State: "test_state",
					}).
					Return(nil, app_error.New(errors.New("failed to create user"), app_error.ErrCodeGeneralServerUnavailable))

				return mockUserService, mockCookies, mockConfig
			},
			url: "/auth/google/callback?code=test_auth_code&state=test_state",
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, w.Code)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, tt.url, nil)

			mockUserService, mockCookies, mockConfig := tt.setup()
			defer mockUserService.AssertExpectations(t)
			defer mockCookies.AssertExpectations(t)

			handler := NewUserHandler(log, mockUserService, mockConfig, nil, nil, mockCookies)
			handler.HandleGoogleCallback(c)

			tt.verify(t, w)
		})
	}
}

func TestUserHandler_GetFacebookAuthURL(t *testing.T) {
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
					GetFacebookAuthURL(ctx).
					Return(&entities.FacebookAuthURLResponse{
						AuthURL: "https://www.facebook.com/v18.0/dialog/oauth?client_id=test&redirect_uri=callback&scope=email&response_type=code&state=random_state",
					}, nil)

				return mockUserService, mockConfig
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, w.Code)
				assert.JSONEq(t, `{"auth_url":"https://www.facebook.com/v18.0/dialog/oauth?client_id=test&redirect_uri=callback&scope=email&response_type=code&state=random_state"}`, w.Body.String())
			},
		},
		{
			name: "ServiceError",
			setup: func() (*services.MockUserService, *config.Config) {
				mockUserService := new(services.MockUserService)
				mockConfig := &config.Config{}

				mockUserService.EXPECT().
					GetFacebookAuthURL(ctx).
					Return(nil, app_error.New(errors.New("failed to generate auth URL"), app_error.ErrCodeGeneralServerUnavailable))

				return mockUserService, mockConfig
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, w.Code)
			},
		},
		{
			name: "AuthConfigurationError",
			setup: func() (*services.MockUserService, *config.Config) {
				mockUserService := new(services.MockUserService)
				mockConfig := &config.Config{}

				mockUserService.EXPECT().
					GetFacebookAuthURL(ctx).
					Return(nil, app_error.New(errors.New("invalid OAuth configuration"), app_error.ErrCodeAuthInvalidRequest))

				return mockUserService, mockConfig
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, w.Code)
			},
		},
		{
			name: "FacebookAPIError",
			setup: func() (*services.MockUserService, *config.Config) {
				mockUserService := new(services.MockUserService)
				mockConfig := &config.Config{}

				mockUserService.EXPECT().
					GetFacebookAuthURL(ctx).
					Return(nil, app_error.New(errors.New("Facebook API unavailable"), app_error.ErrCodeGeneralServerUnavailable))

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
			c.Request = httptest.NewRequest(http.MethodGet, "/auth/facebook/url", nil)

			mockUserService, mockConfig := tt.setup()
			defer mockUserService.AssertExpectations(t)

			handler := NewUserHandler(log, mockUserService, mockConfig, nil, nil, nil)
			handler.GetFacebookAuthURL(c)

			tt.verify(t, w)
		})
	}
}

func TestUserHandler_HandleFacebookCallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log := log.Initialize("test")
	ctx := context.Background()

	tests := []struct {
		name   string
		setup  func() (*services.MockUserService, *utils.MockCookies, *config.Config)
		url    string
		verify func(t *testing.T, w *httptest.ResponseRecorder)
	}{
		{
			name: "Success",
			setup: func() (*services.MockUserService, *utils.MockCookies, *config.Config) {
				mockUserService := new(services.MockUserService)
				mockCookies := new(utils.MockCookies)
				mockConfig := &config.Config{}

				mockUserService.EXPECT().
					HandleFacebookCallback(ctx, &entities.HandleFacebookCallbackReq{
						Code:  "test_auth_code",
						State: "test_state",
					}).
					Return(&entities.HandleFacebookCallbackResp{
						AccessToken:  "test_access_token",
						RefreshToken: "test_refresh_token",
					}, nil)

				mockCookies.EXPECT().
					SetRefreshTokenCookie(mock.Anything, "test_refresh_token").
					Return()

				return mockUserService, mockCookies, mockConfig
			},
			url: "/auth/facebook/callback?code=test_auth_code&state=test_state",
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, w.Code)
				assert.JSONEq(t, `{"access_token":"test_access_token","refresh_token":"test_refresh_token"}`, w.Body.String())
			},
		},
		{
			name: "MissingCode",
			setup: func() (*services.MockUserService, *utils.MockCookies, *config.Config) {
				mockUserService := new(services.MockUserService)
				mockCookies := new(utils.MockCookies)
				mockConfig := &config.Config{}

				return mockUserService, mockCookies, mockConfig
			},
			url: "/auth/facebook/callback?state=test_state",
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, w.Code)
				assert.JSONEq(t, `{"code":"INS0213","message":"Something went wrong with the request. Please try again."}`, w.Body.String())
			},
		},
		{
			name: "MissingState",
			setup: func() (*services.MockUserService, *utils.MockCookies, *config.Config) {
				mockUserService := new(services.MockUserService)
				mockCookies := new(utils.MockCookies)
				mockConfig := &config.Config{}

				return mockUserService, mockCookies, mockConfig
			},
			url: "/auth/facebook/callback?code=test_auth_code",
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, w.Code)
				assert.JSONEq(t, `{"code":"INS0213","message":"Something went wrong with the request. Please try again."}`, w.Body.String())
			},
		},
		{
			name: "MissingBothCodeAndState",
			setup: func() (*services.MockUserService, *utils.MockCookies, *config.Config) {
				mockUserService := new(services.MockUserService)
				mockCookies := new(utils.MockCookies)
				mockConfig := &config.Config{}

				return mockUserService, mockCookies, mockConfig
			},
			url: "/auth/facebook/callback",
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, w.Code)
				assert.JSONEq(t, `{"code":"INS0213","message":"Something went wrong with the request. Please try again."}`, w.Body.String())
			},
		},
		{
			name: "EmptyCode",
			setup: func() (*services.MockUserService, *utils.MockCookies, *config.Config) {
				mockUserService := new(services.MockUserService)
				mockCookies := new(utils.MockCookies)
				mockConfig := &config.Config{}

				return mockUserService, mockCookies, mockConfig
			},
			url: "/auth/facebook/callback?code=&state=test_state",
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, w.Code)
				assert.JSONEq(t, `{"code":"INS0213","message":"Something went wrong with the request. Please try again."}`, w.Body.String())
			},
		},
		{
			name: "EmptyState",
			setup: func() (*services.MockUserService, *utils.MockCookies, *config.Config) {
				mockUserService := new(services.MockUserService)
				mockCookies := new(utils.MockCookies)
				mockConfig := &config.Config{}

				return mockUserService, mockCookies, mockConfig
			},
			url: "/auth/facebook/callback?code=test_auth_code&state=",
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, w.Code)
				assert.JSONEq(t, `{"code":"INS0213","message":"Something went wrong with the request. Please try again."}`, w.Body.String())
			},
		},
		{
			name: "ServiceError",
			setup: func() (*services.MockUserService, *utils.MockCookies, *config.Config) {
				mockUserService := new(services.MockUserService)
				mockCookies := new(utils.MockCookies)
				mockConfig := &config.Config{}

				mockUserService.EXPECT().
					HandleFacebookCallback(ctx, &entities.HandleFacebookCallbackReq{
						Code:  "test_auth_code",
						State: "test_state",
					}).
					Return(nil, app_error.New(errors.New("invalid authorization code"), app_error.ErrCodeAuthInvalidRequest))

				return mockUserService, mockCookies, mockConfig
			},
			url: "/auth/facebook/callback?code=test_auth_code&state=test_state",
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, w.Code)
			},
		},
		{
			name: "OAuthTokenExchangeError",
			setup: func() (*services.MockUserService, *utils.MockCookies, *config.Config) {
				mockUserService := new(services.MockUserService)
				mockCookies := new(utils.MockCookies)
				mockConfig := &config.Config{}

				mockUserService.EXPECT().
					HandleFacebookCallback(ctx, &entities.HandleFacebookCallbackReq{
						Code:  "test_auth_code",
						State: "test_state",
					}).
					Return(nil, app_error.New(errors.New("failed to exchange code for token"), app_error.ErrCodeGeneralServerUnavailable))

				return mockUserService, mockCookies, mockConfig
			},
			url: "/auth/facebook/callback?code=test_auth_code&state=test_state",
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, w.Code)
			},
		},
		{
			name: "UserCreationError",
			setup: func() (*services.MockUserService, *utils.MockCookies, *config.Config) {
				mockUserService := new(services.MockUserService)
				mockCookies := new(utils.MockCookies)
				mockConfig := &config.Config{}

				mockUserService.EXPECT().
					HandleFacebookCallback(ctx, &entities.HandleFacebookCallbackReq{
						Code:  "test_auth_code",
						State: "test_state",
					}).
					Return(nil, app_error.New(errors.New("failed to create user"), app_error.ErrCodeGeneralServerUnavailable))

				return mockUserService, mockCookies, mockConfig
			},
			url: "/auth/facebook/callback?code=test_auth_code&state=test_state",
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, w.Code)
			},
		},
		{
			name: "FacebookAPIError",
			setup: func() (*services.MockUserService, *utils.MockCookies, *config.Config) {
				mockUserService := new(services.MockUserService)
				mockCookies := new(utils.MockCookies)
				mockConfig := &config.Config{}

				mockUserService.EXPECT().
					HandleFacebookCallback(ctx, &entities.HandleFacebookCallbackReq{
						Code:  "test_auth_code",
						State: "test_state",
					}).
					Return(nil, app_error.New(errors.New("Facebook API error"), app_error.ErrCodeGeneralServerUnavailable))

				return mockUserService, mockCookies, mockConfig
			},
			url: "/auth/facebook/callback?code=test_auth_code&state=test_state",
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, w.Code)
			},
		},
		{
			name: "InvalidStateError",
			setup: func() (*services.MockUserService, *utils.MockCookies, *config.Config) {
				mockUserService := new(services.MockUserService)
				mockCookies := new(utils.MockCookies)
				mockConfig := &config.Config{}

				mockUserService.EXPECT().
					HandleFacebookCallback(ctx, &entities.HandleFacebookCallbackReq{
						Code:  "test_auth_code",
						State: "invalid_state",
					}).
					Return(nil, app_error.New(errors.New("invalid state parameter"), app_error.ErrCodeAuthInvalidRequest))

				return mockUserService, mockCookies, mockConfig
			},
			url: "/auth/facebook/callback?code=test_auth_code&state=invalid_state",
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, w.Code)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, tt.url, nil)

			mockUserService, mockCookies, mockConfig := tt.setup()
			defer mockUserService.AssertExpectations(t)
			defer mockCookies.AssertExpectations(t)

			handler := NewUserHandler(log, mockUserService, mockConfig, nil, nil, mockCookies)
			handler.HandleFacebookCallback(c)

			tt.verify(t, w)
		})
	}
}
