package utils

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/interview-simulation/interview-backend-server/internal/config"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	"gitlab.com/interview-simulation/interview-backend-server/internal/entities"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
)

func TestNewCookies(t *testing.T) {
	logger := log.Initialize(constants.TestAppEnv)
	cfg := &config.Config{}

	cookieUtil := NewCookies(cfg, logger)

	assert.NotNil(t, cookieUtil)
	assert.IsType(t, &cookies{}, cookieUtil)
}

func TestSetRefreshTokenCookie(t *testing.T) {
	tests := []struct {
		name           string
		token          string
		duration       time.Duration
		domain         string
		isRejectHTTP   bool
		expectedCookie *http.Cookie
	}{
		{
			name:         "Set Refresh Token Cookie successfully",
			token:        "someRefreshToken",
			duration:     time.Hour * 24,
			domain:       "example.com",
			isRejectHTTP: false,
			expectedCookie: &http.Cookie{
				Name:     string(constants.RefreshTokenCookieKey),
				Value:    "someRefreshToken",
				Path:     "/",
				HttpOnly: true,
				Secure:   true,
				SameSite: http.SameSiteLaxMode,
				Domain:   "example.com",
				Expires:  time.Now().Add(time.Hour * 24),
			},
		},
		{
			name:         "Set Refresh Token Cookie with insecure HTTP flag",
			token:        "anotherRefreshToken",
			duration:     time.Hour * 12,
			domain:       "example.com",
			isRejectHTTP: true,
			expectedCookie: &http.Cookie{
				Name:     string(constants.RefreshTokenCookieKey),
				Value:    "anotherRefreshToken",
				Path:     "/",
				HttpOnly: true,
				Secure:   false,
				SameSite: http.SameSiteLaxMode,
				Domain:   "example.com",
				Expires:  time.Now().Add(time.Hour * 12),
			},
		},
		{
			name:         "Set Refresh Token Cookie with empty domain",
			token:        "tokenWithEmptyDomain",
			duration:     time.Hour * 6,
			domain:       "",
			isRejectHTTP: false,
			expectedCookie: &http.Cookie{
				Name:     string(constants.RefreshTokenCookieKey),
				Value:    "tokenWithEmptyDomain",
				Path:     "/",
				HttpOnly: true,
				Secure:   true,
				SameSite: http.SameSiteLaxMode,
				Domain:   "",
				Expires:  time.Now().Add(time.Hour * 6),
			},
		},
		{
			name:         "Set Refresh Token Cookie with zero duration",
			token:        "tokenWithZeroDuration",
			duration:     0,
			domain:       "localhost",
			isRejectHTTP: true,
			expectedCookie: &http.Cookie{
				Name:     string(constants.RefreshTokenCookieKey),
				Value:    "tokenWithZeroDuration",
				Path:     "/",
				HttpOnly: true,
				Secure:   false,
				SameSite: http.SameSiteLaxMode,
				Domain:   "localhost",
				Expires:  time.Now().Add(0),
			},
		},
		{
			name:         "Set Refresh Token Cookie with negative duration",
			token:        "tokenWithNegativeDuration",
			duration:     -time.Hour,
			domain:       "test.com",
			isRejectHTTP: false,
			expectedCookie: &http.Cookie{
				Name:     string(constants.RefreshTokenCookieKey),
				Value:    "tokenWithNegativeDuration",
				Path:     "/",
				HttpOnly: true,
				Secure:   true,
				SameSite: http.SameSiteLaxMode,
				Domain:   "test.com",
				Expires:  time.Now().Add(-time.Hour),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			logger := log.Initialize(constants.TestAppEnv)
			cfg := &config.Config{}
			c, _ := gin.CreateTestContext(rr)

			cookieUtil := NewCookies(cfg, logger)
			cookieUtil.SetRefreshTokenCookie(c, tt.token, tt.duration, tt.domain, tt.isRejectHTTP)

			cookies := rr.Result().Cookies()
			require.Len(t, cookies, 1)

			actualCookie := cookies[0]
			assert.Equal(t, tt.expectedCookie.Name, actualCookie.Name)
			assert.Equal(t, tt.expectedCookie.Value, actualCookie.Value)
			assert.Equal(t, tt.expectedCookie.Path, actualCookie.Path)
			assert.Equal(t, tt.expectedCookie.HttpOnly, actualCookie.HttpOnly)
			assert.Equal(t, tt.expectedCookie.Secure, actualCookie.Secure)
			assert.Equal(t, tt.expectedCookie.SameSite, actualCookie.SameSite)
			assert.Equal(t, tt.expectedCookie.Domain, actualCookie.Domain)
			assert.WithinDuration(t, tt.expectedCookie.Expires, actualCookie.Expires, time.Second)
		})
	}
}

func TestSetCookie(t *testing.T) {
	tests := []struct {
		name         string
		serviceResp  *entities.SignInUserByEmailAndPasswordResponse
		config       *config.Config
		expectedResp *entities.SignInUserByEmailAndPasswordResponse
		expectError  bool
	}{
		{
			name: "Set Cookie successfully",
			serviceResp: &entities.SignInUserByEmailAndPasswordResponse{
				AccessToken:  "access_token_value",
				RefreshToken: "refresh_token_value",
			},
			config: &config.Config{
				AuthConfig: config.AuthConfig{
					RefreshTokenDuration: time.Hour * 24,
					CookieDomain:         "example.com",
					CookieRejectHTTP:     false,
				},
			},
			expectedResp: &entities.SignInUserByEmailAndPasswordResponse{
				AccessToken:  "access_token_value",
				RefreshToken: "refresh_token_value",
			},
			expectError: false,
		},
		{
			name: "Set Cookie with temporary password",
			serviceResp: &entities.SignInUserByEmailAndPasswordResponse{
				AccessToken:  "temp_access_token",
				RefreshToken: "temp_refresh_token",
			},
			config: &config.Config{
				AuthConfig: config.AuthConfig{
					RefreshTokenDuration: time.Hour * 12,
					CookieDomain:         "localhost",
					CookieRejectHTTP:     true,
				},
			},
			expectedResp: &entities.SignInUserByEmailAndPasswordResponse{
				AccessToken:  "temp_access_token",
				RefreshToken: "temp_refresh_token",
			},
			expectError: false,
		},
		{
			name: "Set Cookie with empty refresh token",
			serviceResp: &entities.SignInUserByEmailAndPasswordResponse{
				AccessToken:  "access_token",
				RefreshToken: "",
			},
			config: &config.Config{
				AuthConfig: config.AuthConfig{
					RefreshTokenDuration: time.Hour * 24,
					CookieDomain:         "example.com",
					CookieRejectHTTP:     false,
				},
			},
			expectedResp: &entities.SignInUserByEmailAndPasswordResponse{
				AccessToken:  "access_token",
				RefreshToken: "",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			logger := log.Initialize(constants.TestAppEnv)
			c, _ := gin.CreateTestContext(rr)

			cookieUtil := NewCookies(tt.config, logger)
			err := cookieUtil.SetCookie(c, tt.serviceResp)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// Verify that refresh token cookie was set
				cookies := rr.Result().Cookies()
				require.Len(t, cookies, 1)

				actualCookie := cookies[0]
				assert.Equal(t, string(constants.RefreshTokenCookieKey), actualCookie.Name)
				assert.Equal(t, tt.serviceResp.RefreshToken, actualCookie.Value)
				assert.Equal(t, "/", actualCookie.Path)
				assert.True(t, actualCookie.HttpOnly)
				assert.Equal(t, !tt.config.AuthConfig.CookieRejectHTTP, actualCookie.Secure)
				assert.Equal(t, http.SameSiteLaxMode, actualCookie.SameSite)
				assert.Equal(t, tt.config.AuthConfig.CookieDomain, actualCookie.Domain)
			}
		})
	}
}

func TestClearRefreshTokenCookie(t *testing.T) {
	tests := []struct {
		name           string
		config         *config.Config
		expectedCookie *http.Cookie
	}{
		{
			name: "Clear Refresh Token Cookie with secure config",
			config: &config.Config{
				AuthConfig: config.AuthConfig{
					CookieDomain:     "example.com",
					CookieRejectHTTP: false,
				},
			},
			expectedCookie: &http.Cookie{
				Name:     string(constants.RefreshTokenCookieKey),
				Value:    "",
				Path:     "/",
				HttpOnly: true,
				Secure:   true,
				SameSite: http.SameSiteLaxMode,
				Domain:   "example.com",
				Expires:  time.Now().Add(-time.Hour * 24),
			},
		},
		{
			name: "Clear Refresh Token Cookie with insecure config",
			config: &config.Config{
				AuthConfig: config.AuthConfig{
					CookieDomain:     "localhost",
					CookieRejectHTTP: true,
				},
			},
			expectedCookie: &http.Cookie{
				Name:     string(constants.RefreshTokenCookieKey),
				Value:    "",
				Path:     "/",
				HttpOnly: true,
				Secure:   false,
				SameSite: http.SameSiteLaxMode,
				Domain:   "localhost",
				Expires:  time.Now().Add(-time.Hour * 24),
			},
		},
		{
			name: "Clear Refresh Token Cookie with empty domain",
			config: &config.Config{
				AuthConfig: config.AuthConfig{
					CookieDomain:     "",
					CookieRejectHTTP: false,
				},
			},
			expectedCookie: &http.Cookie{
				Name:     string(constants.RefreshTokenCookieKey),
				Value:    "",
				Path:     "/",
				HttpOnly: true,
				Secure:   true,
				SameSite: http.SameSiteLaxMode,
				Domain:   "",
				Expires:  time.Now().Add(-time.Hour * 24),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			logger := log.Initialize(constants.TestAppEnv)
			c, _ := gin.CreateTestContext(rr)

			cookieUtil := NewCookies(tt.config, logger)
			cookieUtil.ClearRefreshTokenCookie(c)

			cookies := rr.Result().Cookies()
			require.Len(t, cookies, 1)

			actualCookie := cookies[0]
			assert.Equal(t, tt.expectedCookie.Name, actualCookie.Name)
			assert.Equal(t, tt.expectedCookie.Value, actualCookie.Value)
			assert.Equal(t, tt.expectedCookie.Path, actualCookie.Path)
			assert.Equal(t, tt.expectedCookie.HttpOnly, actualCookie.HttpOnly)
			assert.Equal(t, tt.expectedCookie.Secure, actualCookie.Secure)
			assert.Equal(t, tt.expectedCookie.SameSite, actualCookie.SameSite)
			assert.Equal(t, tt.expectedCookie.Domain, actualCookie.Domain)
			assert.WithinDuration(t, tt.expectedCookie.Expires, actualCookie.Expires, time.Second)
		})
	}
}

func TestCookiesInterface(t *testing.T) {
	var _ Cookies = (*cookies)(nil)
}

func TestSetRefreshTokenCookieWithNilContext(t *testing.T) {
	logger := log.Initialize(constants.TestAppEnv)
	cfg := &config.Config{}
	cookieUtil := NewCookies(cfg, logger)

	// This should panic with nil context
	assert.Panics(t, func() {
		cookieUtil.SetRefreshTokenCookie(nil, "token", time.Hour, "domain", false)
	})
}

func TestClearRefreshTokenCookieWithNilContext(t *testing.T) {
	logger := log.Initialize(constants.TestAppEnv)
	cfg := &config.Config{}
	cookieUtil := NewCookies(cfg, logger)

	// This should panic with nil context
	assert.Panics(t, func() {
		cookieUtil.ClearRefreshTokenCookie(nil)
	})
}
