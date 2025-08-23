package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
	"gitlab.com/interview-simulation/interview-backend-server/internal/mocks/utils"
	utilsPkg "gitlab.com/interview-simulation/interview-backend-server/internal/utils"
)

func TestAuthMiddleware_AuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	lgr := log.Initialize(constants.TestAppEnv)

	testCases := []struct {
		name         string
		setup        func() (*gin.Context, *utils.MockJwtToken)
		expectStatus int
		expectAbort  bool
		verify       func(t *testing.T, c *gin.Context, mockToken *utils.MockJwtToken)
	}{
		{
			name: "AuthMiddleware_MissingAuthorizationHeader",
			setup: func() (*gin.Context, *utils.MockJwtToken) {
				w := httptest.NewRecorder()
				req := httptest.NewRequest("GET", "/test", nil)
				c, _ := gin.CreateTestContext(w)
				c.Request = req

				mockToken := &utils.MockJwtToken{}
				return c, mockToken
			},
			expectStatus: http.StatusUnauthorized,
			expectAbort:  true,
			verify: func(t *testing.T, c *gin.Context, mockToken *utils.MockJwtToken) {
				assert.True(t, c.IsAborted())
				mockToken.AssertNotCalled(t, "VerifyToken")
			},
		},
		{
			name: "AuthMiddleware_InvalidHeaderFormat",
			setup: func() (*gin.Context, *utils.MockJwtToken) {
				w := httptest.NewRecorder()
				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("authorization", "InvalidFormat")
				c, _ := gin.CreateTestContext(w)
				c.Request = req

				mockToken := &utils.MockJwtToken{}
				return c, mockToken
			},
			expectStatus: http.StatusUnauthorized,
			expectAbort:  true,
			verify: func(t *testing.T, c *gin.Context, mockToken *utils.MockJwtToken) {
				assert.True(t, c.IsAborted())
				mockToken.AssertNotCalled(t, "VerifyToken")
			},
		},
		{
			name: "AuthMiddleware_WrongAuthorizationType",
			setup: func() (*gin.Context, *utils.MockJwtToken) {
				w := httptest.NewRecorder()
				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("authorization", "Basic sometoken")
				c, _ := gin.CreateTestContext(w)
				c.Request = req

				mockToken := &utils.MockJwtToken{}
				return c, mockToken
			},
			expectStatus: http.StatusUnauthorized,
			expectAbort:  true,
			verify: func(t *testing.T, c *gin.Context, mockToken *utils.MockJwtToken) {
				assert.True(t, c.IsAborted())
				mockToken.AssertNotCalled(t, "VerifyToken")
			},
		},
		{
			name: "AuthMiddleware_ValidToken",
			setup: func() (*gin.Context, *utils.MockJwtToken) {
				w := httptest.NewRecorder()
				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("authorization", "Bearer valid_access_token")
				c, _ := gin.CreateTestContext(w)
				c.Request = req

				mockToken := &utils.MockJwtToken{}
				expectedPayload := &utilsPkg.SignInTokenPayload{
					UserID: "user123",
					Role:   constants.UserRoleAdmin,
				}
				mockToken.On("VerifyToken", mock.Anything, "valid_access_token").Return(expectedPayload, nil)
				return c, mockToken
			},
			expectStatus: http.StatusOK,
			expectAbort:  false,
			verify: func(t *testing.T, c *gin.Context, mockToken *utils.MockJwtToken) {
				assert.False(t, c.IsAborted())
				mockToken.AssertExpectations(t)

				payload, exists := c.Get(string(constants.AuthorizationPayloadKey))
				assert.True(t, exists)
				tokenPayload := payload.(*utilsPkg.SignInTokenPayload)
				assert.Equal(t, "user123", tokenPayload.UserID)
				assert.Equal(t, constants.UserRoleAdmin, tokenPayload.Role)

				_, exists = c.Get(string(constants.NewAccessTokenKey))
				assert.False(t, exists)
			},
		},
		{
			name: "AuthMiddleware_ExpiredTokenWithValidRefreshToken",
			setup: func() (*gin.Context, *utils.MockJwtToken) {
				w := httptest.NewRecorder()
				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("authorization", "Bearer expired_access_token")
				req.AddCookie(&http.Cookie{
					Name:  "refresh_token",
					Value: "valid_refresh_token",
				})
				c, _ := gin.CreateTestContext(w)
				c.Request = req

				mockToken := &utils.MockJwtToken{}
				expectedPayload := &utilsPkg.SignInTokenPayload{
					UserID: "user123",
					Role:   constants.UserRoleAdmin,
				}

				mockToken.On("VerifyToken", mock.Anything, "expired_access_token").Return(nil, app_error.New(constants.ErrExpiredToken, app_error.ErrCodeAuthExpiredToken))
				mockToken.On("RenewAccessToken", mock.Anything, "valid_refresh_token").Return("new_access_token", expectedPayload, nil)
				return c, mockToken
			},
			expectStatus: http.StatusOK,
			expectAbort:  false,
			verify: func(t *testing.T, c *gin.Context, mockToken *utils.MockJwtToken) {
				assert.False(t, c.IsAborted())
				mockToken.AssertExpectations(t)

				payload, exists := c.Get(string(constants.AuthorizationPayloadKey))
				assert.True(t, exists)
				tokenPayload := payload.(*utilsPkg.SignInTokenPayload)
				assert.Equal(t, "user123", tokenPayload.UserID)

				newToken, exists := c.Get(string(constants.NewAccessTokenKey))
				assert.True(t, exists)
				assert.Equal(t, "new_access_token", newToken)
			},
		},
		{
			name: "AuthMiddleware_ExpiredTokenWithMissingRefreshToken",
			setup: func() (*gin.Context, *utils.MockJwtToken) {
				w := httptest.NewRecorder()
				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("authorization", "Bearer expired_access_token")
				c, _ := gin.CreateTestContext(w)
				c.Request = req

				mockToken := &utils.MockJwtToken{}
				mockToken.On("VerifyToken", mock.Anything, "expired_access_token").Return(nil, app_error.New(constants.ErrExpiredToken, app_error.ErrCodeAuthExpiredToken))
				return c, mockToken
			},
			expectStatus: http.StatusUnauthorized,
			expectAbort:  true,
			verify: func(t *testing.T, c *gin.Context, mockToken *utils.MockJwtToken) {
				assert.True(t, c.IsAborted())
				mockToken.AssertExpectations(t)
				mockToken.AssertNotCalled(t, "RenewAccessToken")
			},
		},
		{
			name: "AuthMiddleware_ExpiredTokenWithInvalidRefreshToken",
			setup: func() (*gin.Context, *utils.MockJwtToken) {
				w := httptest.NewRecorder()
				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("authorization", "Bearer expired_access_token")
				req.AddCookie(&http.Cookie{
					Name:  "refresh_token",
					Value: "invalid_refresh_token",
				})
				c, _ := gin.CreateTestContext(w)
				c.Request = req

				mockToken := &utils.MockJwtToken{}
				mockToken.On("VerifyToken", mock.Anything, "expired_access_token").Return(nil, app_error.New(constants.ErrExpiredToken, app_error.ErrCodeAuthExpiredToken))
				mockToken.On("RenewAccessToken", mock.Anything, "invalid_refresh_token").Return("", nil, app_error.New(constants.ErrInvalidToken, app_error.ErrCodeAuthInvalidToken))
				return c, mockToken
			},
			expectStatus: http.StatusUnauthorized,
			expectAbort:  true,
			verify: func(t *testing.T, c *gin.Context, mockToken *utils.MockJwtToken) {
				assert.True(t, c.IsAborted())
				mockToken.AssertExpectations(t)
			},
		},
		{
			name: "AuthMiddleware_InvalidToken",
			setup: func() (*gin.Context, *utils.MockJwtToken) {
				w := httptest.NewRecorder()
				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("authorization", "Bearer invalid_access_token")
				c, _ := gin.CreateTestContext(w)
				c.Request = req

				mockToken := &utils.MockJwtToken{}
				mockToken.On("VerifyToken", mock.Anything, "invalid_access_token").Return(nil, app_error.New(constants.ErrInvalidToken, app_error.ErrCodeAuthInvalidToken))
				return c, mockToken
			},
			expectStatus: http.StatusUnauthorized,
			expectAbort:  true,
			verify: func(t *testing.T, c *gin.Context, mockToken *utils.MockJwtToken) {
				assert.True(t, c.IsAborted())
				mockToken.AssertExpectations(t)
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			c, mockToken := tC.setup()

			middleware := NewAuthMiddleware(mockToken, lgr)
			authHandler := middleware.AuthMiddleware()

			authHandler(c)

			if tC.expectAbort {
				assert.True(t, c.IsAborted())
				assert.Equal(t, tC.expectStatus, c.Writer.Status())
			} else {
				assert.False(t, c.IsAborted())
			}

			tC.verify(t, c, mockToken)
		})
	}
}

func TestNewAuthMiddleware(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	mockToken := &utils.MockJwtToken{}

	middleware := NewAuthMiddleware(mockToken, lgr)

	assert.NotNil(t, middleware)
	assert.Implements(t, (*AuthMiddleware)(nil), middleware)
}
