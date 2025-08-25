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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.com/interview-simulation/interview-backend-server/internal/config"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
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

func TestUserHandler_SignUpUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log := log.Initialize("test")
	ctx := context.Background()

	tests := []struct {
		name   string
		input  func() *entities.SignUpUserRequest
		setup  func() (*services.MockUserService, *utils.MockValidator, *config.Config)
		verify func(t *testing.T, w *httptest.ResponseRecorder)
	}{
		{
			name: "Success",
			input: func() *entities.SignUpUserRequest {
				return &entities.SignUpUserRequest{
					Email:       "test@example.com",
					Password:    "password123",
					FullName:    "Test User",
					Country:     "USA",
					Gender:      "male",
					DateOfBirth: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
				}
			},
			setup: func() (*services.MockUserService, *utils.MockValidator, *config.Config) {
				mockUserService := new(services.MockUserService)
				mockValidator := new(utils.MockValidator)
				mockConfig := &config.Config{}

				mockValidator.EXPECT().
					ValidateAndBind(mock.Anything, mock.Anything, "SignUpUser").
					Return(nil)

				mockUserService.EXPECT().
					SignUpUser(ctx, mock.Anything).
					Return(&entities.SignUpUserResponse{TokenId: "token-123"}, nil)

				return mockUserService, mockValidator, mockConfig
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, w.Code)
				assert.JSONEq(t, `{"token_id":"token-123"}`, w.Body.String())
			},
		},
		{
			name: "ValidationFailed",
			input: func() *entities.SignUpUserRequest {
				return &entities.SignUpUserRequest{
					Email:       "invalid-email",
					Password:    "",
					FullName:    "",
					Country:     "",
					Gender:      "",
					DateOfBirth: time.Time{},
				}
			},
			setup: func() (*services.MockUserService, *utils.MockValidator, *config.Config) {
				mockUserService := new(services.MockUserService)
				mockValidator := new(utils.MockValidator)
				mockConfig := &config.Config{}

				mockValidator.EXPECT().
					ValidateAndBind(mock.Anything, mock.Anything, "SignUpUser").
					Return(app_error.NewWithCustomMessage(errors.New("validation failed"), app_error.ErrCodeAuthInvalidRequest, "validation failed"))

				return mockUserService, mockValidator, mockConfig
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, w.Code)
				assert.JSONEq(t, `{"code":"ONX0213","message":"validation failed"}`, w.Body.String())
			},
		},
		{
			name: "ServiceError",
			input: func() *entities.SignUpUserRequest {
				return &entities.SignUpUserRequest{
					Email:       "test@example.com",
					Password:    "password123",
					FullName:    "Test User",
					Country:     "USA",
					Gender:      "male",
					DateOfBirth: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
				}
			},
			setup: func() (*services.MockUserService, *utils.MockValidator, *config.Config) {
				mockUserService := new(services.MockUserService)
				mockValidator := new(utils.MockValidator)
				mockConfig := &config.Config{}

				mockValidator.EXPECT().
					ValidateAndBind(mock.Anything, mock.Anything, "SignUpUser").
					Return(nil)

				mockUserService.EXPECT().
					SignUpUser(ctx, mock.Anything).
					Return(nil, app_error.New(errors.New("user already exists"), app_error.ErrCodeAuthUserAlreadyExists))

				return mockUserService, mockValidator, mockConfig
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, w.Code)
				assert.JSONEq(t, `{"code":"ONX0207","message":"This email is already registered. Try logging in instead."}`, w.Body.String())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.input())
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPost, "/signup", bytes.NewBuffer(body))
			c.Request.Header.Set("Content-Type", "application/json")

			mockUserService, mockValidator, mockConfig := tt.setup()
			defer mockUserService.AssertExpectations(t)
			defer mockValidator.AssertExpectations(t)

			handler := NewUserHandler(log, mockUserService, mockConfig, nil, mockValidator, nil)
			handler.SignUpUser(c)

			tt.verify(t, w)
		})
	}
}

func TestUserHandler_SignInUserByEmailAndPassword(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log := log.Initialize("test")
	ctx := context.Background()

	tests := []struct {
		name   string
		input  func() *entities.SignInUserByEmailAndPasswordRequest
		setup  func(c *gin.Context) (*services.MockUserService, *utils.MockValidator, *config.Config, *utils.MockCookies)
		verify func(t *testing.T, w *httptest.ResponseRecorder)
	}{
		{
			name: "Success",
			input: func() *entities.SignInUserByEmailAndPasswordRequest {
				return &entities.SignInUserByEmailAndPasswordRequest{
					Email:    "admin@example.com",
					Password: "password123",
				}
			},
			setup: func(c *gin.Context) (*services.MockUserService, *utils.MockValidator, *config.Config, *utils.MockCookies) {
				mockUserService := new(services.MockUserService)
				mockValidator := new(utils.MockValidator)
				mockCookies := new(utils.MockCookies)
				mockConfig := &config.Config{}

				mockValidator.EXPECT().
					ValidateAndBind(mock.Anything, mock.Anything, "SignInUserByEmailAndPassword").
					Return(nil)

				mockUserService.EXPECT().
					SignInUserByEmailAndPassword(ctx, mock.Anything).
					Return(&entities.SignInUserByEmailAndPasswordResponse{
						AccessToken:  "access-token",
						RefreshToken: "refresh-token",
					}, nil)

				mockCookies.EXPECT().
					SetRefreshTokenCookie(mock.Anything, mock.Anything)

				return mockUserService, mockValidator, mockConfig, mockCookies
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, w.Code)
				assert.JSONEq(t, `{"access_token":"access-token","refresh_token":"refresh-token"}`, w.Body.String())
			},
		},
		{
			name: "ValidationFailed",
			input: func() *entities.SignInUserByEmailAndPasswordRequest {
				return &entities.SignInUserByEmailAndPasswordRequest{
					Email:    "admin@example.com",
					Password: "",
				}
			},
			setup: func(c *gin.Context) (*services.MockUserService, *utils.MockValidator, *config.Config, *utils.MockCookies) {
				mockUserService := new(services.MockUserService)
				mockValidator := new(utils.MockValidator)
				mockCookies := new(utils.MockCookies)
				mockConfig := &config.Config{}

				mockValidator.EXPECT().
					ValidateAndBind(mock.Anything, mock.Anything, "SignInUserByEmailAndPassword").
					Return(app_error.NewWithCustomMessage(errors.New("password is required"), app_error.ErrCodeAuthInvalidRequest, "password is required"))

				return mockUserService, mockValidator, mockConfig, mockCookies
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, w.Code)
				assert.JSONEq(t, `{"code":"ONX0213","message":"password is required"}`, w.Body.String())
			},
		},
		{
			name: "ServiceError",
			input: func() *entities.SignInUserByEmailAndPasswordRequest {
				return &entities.SignInUserByEmailAndPasswordRequest{
					Email:    "admin@example.com",
					Password: "password123",
				}
			},
			setup: func(c *gin.Context) (*services.MockUserService, *utils.MockValidator, *config.Config, *utils.MockCookies) {
				mockUserService := new(services.MockUserService)
				mockValidator := new(utils.MockValidator)
				mockCookies := new(utils.MockCookies)
				mockConfig := &config.Config{}

				mockValidator.EXPECT().
					ValidateAndBind(mock.Anything, mock.Anything, "SignInUserByEmailAndPassword").
					Return(nil)

				mockUserService.EXPECT().
					SignInUserByEmailAndPassword(ctx, mock.Anything).
					Return(nil, app_error.New(errors.New("invalid credentials"), app_error.ErrCodeAuthInvalidPassword))

				return mockUserService, mockValidator, mockConfig, mockCookies
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, w.Code)
				assert.JSONEq(t, `{"code":"ONX0217","message":"Incorrect email or password. Please try again."}`, w.Body.String())
			},
		},
		{
			name: "CookieError",
			input: func() *entities.SignInUserByEmailAndPasswordRequest {
				return &entities.SignInUserByEmailAndPasswordRequest{
					Email:    "admin@example.com",
					Password: "password123",
				}
			},
			setup: func(c *gin.Context) (*services.MockUserService, *utils.MockValidator, *config.Config, *utils.MockCookies) {
				mockUserService := new(services.MockUserService)
				mockValidator := new(utils.MockValidator)
				mockCookies := new(utils.MockCookies)
				mockConfig := &config.Config{}

				mockValidator.EXPECT().
					ValidateAndBind(mock.Anything, mock.Anything, "SignInUserByEmailAndPassword").
					Return(nil)

				mockUserService.EXPECT().
					SignInUserByEmailAndPassword(ctx, mock.Anything).
					Return(&entities.SignInUserByEmailAndPasswordResponse{
						AccessToken:  "access-token",
						RefreshToken: "refresh-token",
					}, nil)

				mockCookies.EXPECT().
					SetRefreshTokenCookie(mock.Anything, mock.Anything)

				return mockUserService, mockValidator, mockConfig, mockCookies
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
			c.Request = httptest.NewRequest(http.MethodPost, "/signin", bytes.NewBuffer(body))
			c.Request.Header.Set("Content-Type", "application/json")

			mockUserService, mockValidator, mockConfig, mockCookies := tt.setup(c)
			defer mockUserService.AssertExpectations(t)
			defer mockValidator.AssertExpectations(t)
			defer mockCookies.AssertExpectations(t)

			handler := NewUserHandler(log, mockUserService, mockConfig, nil, mockValidator, mockCookies)
			handler.SignInUserByEmailAndPassword(c)

			tt.verify(t, w)
		})
	}
}

func TestUserHandler_SendVerifyEmail(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log := log.Initialize("test")
	ctx := context.Background()

	tests := []struct {
		name   string
		input  func() *entities.VerifyEmailRequest
		setup  func() (*services.MockUserService, *utils.MockValidator, *config.Config)
		verify func(t *testing.T, w *httptest.ResponseRecorder)
	}{
		{
			name: "Success",
			input: func() *entities.VerifyEmailRequest {
				return &entities.VerifyEmailRequest{
					Token: "token-123",
					Code:  "123456",
				}
			},
			setup: func() (*services.MockUserService, *utils.MockValidator, *config.Config) {
				mockUserService := new(services.MockUserService)
				mockValidator := new(utils.MockValidator)
				mockConfig := &config.Config{}

				mockValidator.EXPECT().
					ValidateAndBind(mock.Anything, mock.Anything, "SendVerifyEmail").
					Return(nil)

				mockUserService.EXPECT().
					SendVerifyEmail(ctx, mock.Anything).
					Return(nil)

				return mockUserService, mockValidator, mockConfig
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNoContent, w.Code)
			},
		},
		{
			name: "ValidationFailed",
			input: func() *entities.VerifyEmailRequest {
				return &entities.VerifyEmailRequest{
					Token: "",
					Code:  "",
				}
			},
			setup: func() (*services.MockUserService, *utils.MockValidator, *config.Config) {
				mockUserService := new(services.MockUserService)
				mockValidator := new(utils.MockValidator)
				mockConfig := &config.Config{}

				mockValidator.EXPECT().
					ValidateAndBind(mock.Anything, mock.Anything, "SendVerifyEmail").
					Return(app_error.NewWithCustomMessage(errors.New("token is required"), app_error.ErrCodeAuthInvalidRequest, "token is required"))

				return mockUserService, mockValidator, mockConfig
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, w.Code)
				assert.JSONEq(t, `{"code":"ONX0213","message":"token is required"}`, w.Body.String())
			},
		},
		{
			name: "ServiceError",
			input: func() *entities.VerifyEmailRequest {
				return &entities.VerifyEmailRequest{
					Token: "token-123",
					Code:  "123456",
				}
			},
			setup: func() (*services.MockUserService, *utils.MockValidator, *config.Config) {
				mockUserService := new(services.MockUserService)
				mockValidator := new(utils.MockValidator)
				mockConfig := &config.Config{}

				mockValidator.EXPECT().
					ValidateAndBind(mock.Anything, mock.Anything, "SendVerifyEmail").
					Return(nil)

				mockUserService.EXPECT().
					SendVerifyEmail(ctx, mock.Anything).
					Return(app_error.New(errors.New("invalid code"), app_error.ErrCodeAuthInvalidVerifyEmailCode))

				return mockUserService, mockValidator, mockConfig
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, w.Code)
				assert.JSONEq(t, `{"code":"ONX0214","message":"The verification code is invalid. Please try again."}`, w.Body.String())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.input())
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPost, "/verify-email", bytes.NewBuffer(body))
			c.Request.Header.Set("Content-Type", "application/json")

			mockUserService, mockValidator, mockConfig := tt.setup()
			defer mockUserService.AssertExpectations(t)
			defer mockValidator.AssertExpectations(t)

			handler := NewUserHandler(log, mockUserService, mockConfig, nil, mockValidator, nil)
			handler.SendVerifyEmail(c)

			tt.verify(t, w)
		})
	}
}

func TestUserHandler_ForgotPassword(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log := log.Initialize("test")
	ctx := context.Background()

	tests := []struct {
		name   string
		input  func() *entities.ForgotPasswordRequest
		setup  func() (*services.MockUserService, *utils.MockValidator, *config.Config)
		verify func(t *testing.T, w *httptest.ResponseRecorder)
	}{
		{
			name: "Success",
			input: func() *entities.ForgotPasswordRequest {
				return &entities.ForgotPasswordRequest{
					Email: "test@example.com",
				}
			},
			setup: func() (*services.MockUserService, *utils.MockValidator, *config.Config) {
				mockUserService := new(services.MockUserService)
				mockValidator := new(utils.MockValidator)
				mockConfig := &config.Config{}

				mockValidator.EXPECT().
					ValidateAndBind(mock.Anything, mock.Anything, "ForgotPassword").
					Return(nil)

				mockUserService.EXPECT().
					ForgotPassword(ctx, mock.Anything).
					Return(nil)

				return mockUserService, mockValidator, mockConfig
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNoContent, w.Code)
			},
		},
		{
			name: "ValidationFailed",
			input: func() *entities.ForgotPasswordRequest {
				return &entities.ForgotPasswordRequest{
					Email: "invalid-email",
				}
			},
			setup: func() (*services.MockUserService, *utils.MockValidator, *config.Config) {
				mockUserService := new(services.MockUserService)
				mockValidator := new(utils.MockValidator)
				mockConfig := &config.Config{}

				mockValidator.EXPECT().
					ValidateAndBind(mock.Anything, mock.Anything, "ForgotPassword").
					Return(app_error.NewWithCustomMessage(errors.New("invalid email"), app_error.ErrCodeAuthInvalidRequest, "invalid email"))

				return mockUserService, mockValidator, mockConfig
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, w.Code)
				assert.JSONEq(t, `{"code":"ONX0213","message":"invalid email"}`, w.Body.String())
			},
		},
		{
			name: "ServiceError",
			input: func() *entities.ForgotPasswordRequest {
				return &entities.ForgotPasswordRequest{
					Email: "test@example.com",
				}
			},
			setup: func() (*services.MockUserService, *utils.MockValidator, *config.Config) {
				mockUserService := new(services.MockUserService)
				mockValidator := new(utils.MockValidator)
				mockConfig := &config.Config{}

				mockValidator.EXPECT().
					ValidateAndBind(mock.Anything, mock.Anything, "ForgotPassword").
					Return(nil)

				mockUserService.EXPECT().
					ForgotPassword(ctx, mock.Anything).
					Return(app_error.New(errors.New("user not found"), app_error.ErrCodeAuthUserNotFound))

				return mockUserService, mockValidator, mockConfig
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, w.Code)
				assert.JSONEq(t, `{"code":"ONX0206","message":"We couldn't find your account. Please sign up to continue."}`, w.Body.String())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.input())
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPost, "/forgot-password", bytes.NewBuffer(body))
			c.Request.Header.Set("Content-Type", "application/json")

			mockUserService, mockValidator, mockConfig := tt.setup()
			defer mockUserService.AssertExpectations(t)
			defer mockValidator.AssertExpectations(t)

			handler := NewUserHandler(log, mockUserService, mockConfig, nil, mockValidator, nil)
			handler.ForgotPassword(c)

			tt.verify(t, w)
		})
	}
}

func TestUserHandler_RefreshToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log := log.Initialize("test")
	ctx := context.Background()

	tests := []struct {
		name   string
		setup  func(c *gin.Context) (*services.MockUserService, *config.Config)
		verify func(t *testing.T, w *httptest.ResponseRecorder)
	}{
		{
			name: "Success",
			setup: func(c *gin.Context) (*services.MockUserService, *config.Config) {
				mockUserService := new(services.MockUserService)
				mockConfig := &config.Config{}

				// Set cookie in the request
				c.Request.AddCookie(&http.Cookie{
					Name:  string(constants.RefreshTokenCookieKey),
					Value: "refresh-token-123",
				})

				mockUserService.EXPECT().
					RefreshToken(ctx, &entities.RefreshTokenRequest{
						RefreshToken: "refresh-token-123",
					}).
					Return(&entities.RefreshTokenResponse{
						AccessToken: "new-access-token",
					}, nil)

				return mockUserService, mockConfig
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, w.Code)
				assert.JSONEq(t, `{"access_token":"new-access-token"}`, w.Body.String())
			},
		},
		{
			name: "MissingCookie",
			setup: func(c *gin.Context) (*services.MockUserService, *config.Config) {
				mockUserService := new(services.MockUserService)
				mockConfig := &config.Config{}

				return mockUserService, mockConfig
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusUnauthorized, w.Code)
				assert.JSONEq(t, `{"code":"ONX0202","message":"Your refresh token is invalid. Please log in again."}`, w.Body.String())
			},
		},
		{
			name: "ServiceError",
			setup: func(c *gin.Context) (*services.MockUserService, *config.Config) {
				mockUserService := new(services.MockUserService)
				mockConfig := &config.Config{}

				// Set cookie in the request
				c.Request.AddCookie(&http.Cookie{
					Name:  string(constants.RefreshTokenCookieKey),
					Value: "refresh-token-123",
				})

				mockUserService.EXPECT().
					RefreshToken(ctx, &entities.RefreshTokenRequest{
						RefreshToken: "refresh-token-123",
					}).
					Return(nil, app_error.New(errors.New("invalid refresh token"), app_error.ErrCodeAuthInvalidRefreshToken))

				return mockUserService, mockConfig
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusUnauthorized, w.Code)
				assert.JSONEq(t, `{"code":"ONX0202","message":"Your refresh token is invalid. Please log in again."}`, w.Body.String())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPost, "/refresh-token", nil)

			mockUserService, mockConfig := tt.setup(c)
			defer mockUserService.AssertExpectations(t)

			handler := NewUserHandler(log, mockUserService, mockConfig, nil, nil, nil)
			handler.RefreshToken(c)

			tt.verify(t, w)
		})
	}
}

func TestUserHandler_ResetVerifyEmailCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log := log.Initialize("test")
	ctx := context.Background()

	tests := []struct {
		name   string
		input  func() *entities.ResetVerifyEmailCodeRequest
		setup  func() (*services.MockUserService, *utils.MockValidator, *config.Config)
		verify func(t *testing.T, w *httptest.ResponseRecorder)
	}{
		{
			name: "Success",
			input: func() *entities.ResetVerifyEmailCodeRequest {
				return &entities.ResetVerifyEmailCodeRequest{
					Token: "token-123",
				}
			},
			setup: func() (*services.MockUserService, *utils.MockValidator, *config.Config) {
				mockUserService := new(services.MockUserService)
				mockValidator := new(utils.MockValidator)
				mockConfig := &config.Config{}

				mockValidator.EXPECT().
					ValidateAndBind(mock.Anything, mock.Anything, "ResetVerifyEmailCode").
					Return(nil)

				mockUserService.EXPECT().
					ResetVerifyEmailCode(ctx, mock.Anything).
					Return(&entities.ResetVerifyEmailCodeResponse{TokenId: "new-token-123"}, nil)

				return mockUserService, mockValidator, mockConfig
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, w.Code)
				assert.JSONEq(t, `{"token_id":"new-token-123"}`, w.Body.String())
			},
		},
		{
			name: "ValidationFailed",
			input: func() *entities.ResetVerifyEmailCodeRequest {
				return &entities.ResetVerifyEmailCodeRequest{
					Token: "",
				}
			},
			setup: func() (*services.MockUserService, *utils.MockValidator, *config.Config) {
				mockUserService := new(services.MockUserService)
				mockValidator := new(utils.MockValidator)
				mockConfig := &config.Config{}

				mockValidator.EXPECT().
					ValidateAndBind(mock.Anything, mock.Anything, "ResetVerifyEmailCode").
					Return(app_error.NewWithCustomMessage(errors.New("token is required"), app_error.ErrCodeAuthInvalidRequest, "token is required"))

				return mockUserService, mockValidator, mockConfig
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, w.Code)
				assert.JSONEq(t, `{"code":"ONX0213","message":"token is required"}`, w.Body.String())
			},
		},
		{
			name: "ServiceError",
			input: func() *entities.ResetVerifyEmailCodeRequest {
				return &entities.ResetVerifyEmailCodeRequest{
					Token: "token-123",
				}
			},
			setup: func() (*services.MockUserService, *utils.MockValidator, *config.Config) {
				mockUserService := new(services.MockUserService)
				mockValidator := new(utils.MockValidator)
				mockConfig := &config.Config{}

				mockValidator.EXPECT().
					ValidateAndBind(mock.Anything, mock.Anything, "ResetVerifyEmailCode").
					Return(nil)

				mockUserService.EXPECT().
					ResetVerifyEmailCode(ctx, mock.Anything).
					Return(nil, app_error.New(errors.New("max attempts reached"), app_error.ErrCodeAuthMaxAttemptVerifyEmail))

				return mockUserService, mockValidator, mockConfig
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, w.Code)
				assert.JSONEq(t, `{"code":"ONX0216","message":"You have reached the maximum number of attempts. Please resend the new code."}`, w.Body.String())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.input())
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPost, "/reset-verify-email-code", bytes.NewBuffer(body))
			c.Request.Header.Set("Content-Type", "application/json")

			mockUserService, mockValidator, mockConfig := tt.setup()
			defer mockUserService.AssertExpectations(t)
			defer mockValidator.AssertExpectations(t)

			handler := NewUserHandler(log, mockUserService, mockConfig, nil, mockValidator, nil)
			handler.ResetVerifyEmailCode(c)

			tt.verify(t, w)
		})
	}
}

func TestUserHandler_ResetUserPassword(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log := log.Initialize("test")
	ctx := context.Background()

	tests := []struct {
		name   string
		input  func() *entities.ResetUserPasswordRequest
		setup  func() (*services.MockUserService, *utils.MockValidator, *config.Config)
		verify func(t *testing.T, w *httptest.ResponseRecorder)
	}{
		{
			name: "Success",
			input: func() *entities.ResetUserPasswordRequest {
				return &entities.ResetUserPasswordRequest{
					Token:       "token-123",
					NewPassword: "newpassword123",
				}
			},
			setup: func() (*services.MockUserService, *utils.MockValidator, *config.Config) {
				mockUserService := new(services.MockUserService)
				mockValidator := new(utils.MockValidator)
				mockConfig := &config.Config{}

				mockValidator.EXPECT().
					ValidateAndBind(mock.Anything, mock.Anything, "ResetUserPassword").
					Return(nil)

				mockUserService.EXPECT().
					ResetUserPassword(ctx, mock.Anything).
					Return(nil)

				return mockUserService, mockValidator, mockConfig
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNoContent, w.Code)
			},
		},
		{
			name: "ValidationFailed",
			input: func() *entities.ResetUserPasswordRequest {
				return &entities.ResetUserPasswordRequest{
					Token:       "",
					NewPassword: "",
				}
			},
			setup: func() (*services.MockUserService, *utils.MockValidator, *config.Config) {
				mockUserService := new(services.MockUserService)
				mockValidator := new(utils.MockValidator)
				mockConfig := &config.Config{}

				mockValidator.EXPECT().
					ValidateAndBind(mock.Anything, mock.Anything, "ResetUserPassword").
					Return(app_error.NewWithCustomMessage(errors.New("token is required"), app_error.ErrCodeAuthInvalidRequest, "token is required"))

				return mockUserService, mockValidator, mockConfig
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, w.Code)
				assert.JSONEq(t, `{"code":"ONX0213","message":"token is required"}`, w.Body.String())
			},
		},
		{
			name: "ServiceError",
			input: func() *entities.ResetUserPasswordRequest {
				return &entities.ResetUserPasswordRequest{
					Token:       "token-123",
					NewPassword: "newpassword123",
				}
			},
			setup: func() (*services.MockUserService, *utils.MockValidator, *config.Config) {
				mockUserService := new(services.MockUserService)
				mockValidator := new(utils.MockValidator)
				mockConfig := &config.Config{}

				mockValidator.EXPECT().
					ValidateAndBind(mock.Anything, mock.Anything, "ResetUserPassword").
					Return(nil)

				mockUserService.EXPECT().
					ResetUserPassword(ctx, mock.Anything).
					Return(app_error.New(errors.New("token expired"), app_error.ErrCodeAuthResetTokenExpired))

				return mockUserService, mockValidator, mockConfig
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, w.Code)
				assert.JSONEq(t, `{"code":"ONX0209","message":"That reset link has expired. Please request a new one."}`, w.Body.String())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.input())
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPost, "/reset-password", bytes.NewBuffer(body))
			c.Request.Header.Set("Content-Type", "application/json")

			mockUserService, mockValidator, mockConfig := tt.setup()
			defer mockUserService.AssertExpectations(t)
			defer mockValidator.AssertExpectations(t)

			handler := NewUserHandler(log, mockUserService, mockConfig, nil, mockValidator, nil)
			handler.ResetUserPassword(c)

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
				assert.JSONEq(t, `{"code":"ONX0200","message":"Your token is invalid. Please log in again."}`, w.Body.String())
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
				assert.JSONEq(t, `{"code":"ONX0218","message":"Your session has expired or is invalid. Please log in again."}`, w.Body.String())
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
