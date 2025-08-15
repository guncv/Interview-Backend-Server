package utils

// func TestSetRefreshTokenCookie(t *testing.T) {
// 	tests := []struct {
// 		name           string
// 		token          string
// 		duration       time.Duration
// 		domain         string
// 		isRejectHTTP   bool
// 		expectedCookie *http.Cookie
// 	}{
// 		{
// 			name:         "Set Refresh Token Cookie successfully",
// 			token:        "someRefreshToken",
// 			duration:     time.Hour * 24,
// 			domain:       "example.com",
// 			isRejectHTTP: false,
// 			expectedCookie: &http.Cookie{
// 				Name:     "refresh_token",
// 				Value:    "someRefreshToken",
// 				Path:     "/",
// 				HttpOnly: true,
// 				Secure:   true,
// 				SameSite: http.SameSiteLaxMode,
// 				Domain:   "example.com",
// 				Expires:  time.Now().Add(time.Hour * 24),
// 			},
// 		},
// 		{
// 			name:         "Set Refresh Token Cookie with insecure HTTP flag",
// 			token:        "anotherRefreshToken",
// 			duration:     time.Hour * 12,
// 			domain:       "example.com",
// 			isRejectHTTP: true,
// 			expectedCookie: &http.Cookie{
// 				Name:     "refresh_token",
// 				Value:    "anotherRefreshToken",
// 				Path:     "/",
// 				HttpOnly: true,
// 				Secure:   false,
// 				SameSite: http.SameSiteLaxMode,
// 				Domain:   "example.com",
// 				Expires:  time.Now().Add(time.Hour * 12),
// 			},
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			rr := httptest.NewRecorder()
// 			logger := log.Initialize(constants.TestAppEnv)
// 			cfg := &config.Config{}
// 			c, _ := gin.CreateTestContext(rr)

// 			cookieUtil := NewCookies(cfg, logger)
// 			cookieUtil.SetRefreshTokenCookie(c, tt.token, tt.duration, tt.domain, tt.isRejectHTTP)

// 			cookies := rr.Result().Cookies()
// 			require.Len(t, cookies, 1)

// 			actualCookie := cookies[0]
// 			assert.Equal(t, tt.expectedCookie.Name, actualCookie.Name)
// 			assert.Equal(t, tt.expectedCookie.Value, actualCookie.Value)
// 			assert.Equal(t, tt.expectedCookie.Path, actualCookie.Path)
// 			assert.Equal(t, tt.expectedCookie.HttpOnly, actualCookie.HttpOnly)
// 			assert.Equal(t, tt.expectedCookie.Secure, actualCookie.Secure)
// 			assert.Equal(t, tt.expectedCookie.SameSite, actualCookie.SameSite)
// 			assert.Equal(t, tt.expectedCookie.Domain, actualCookie.Domain)
// 			assert.WithinDuration(t, tt.expectedCookie.Expires, actualCookie.Expires, time.Second)
// 		})
// 	}
// }

// func TestSetCookie(t *testing.T) {
// 	tests := []struct {
// 		name         string
// 		serviceResp  *entities.SignInServiceResponse
// 		config       *config.Config
// 		expectedResp *entities.SignInResponse
// 		expectError  bool
// 	}{
// 		{
// 			name: "Set Cookie successfully",
// 			serviceResp: &entities.SignInServiceResponse{
// 				ID:             "user123",
// 				AccessToken:    "access_token_value",
// 				RefreshToken:   "refresh_token_value",
// 				IsTempPassword: false,
// 			},
// 			config: &config.Config{
// 				AuthConfig: config.AuthConfig{
// 					RefreshTokenDuration: time.Hour * 24,
// 					CookieDomain:         "example.com",
// 					CookieRejectHTTP:     false,
// 				},
// 			},
// 			expectedResp: &entities.SignInResponse{
// 				ID:             "user123",
// 				AccessToken:    "access_token_value",
// 				IsTempPassword: false,
// 			},
// 			expectError: false,
// 		},
// 		{
// 			name: "Set Cookie with temporary password",
// 			serviceResp: &entities.SignInServiceResponse{
// 				ID:             "user456",
// 				AccessToken:    "temp_access_token",
// 				RefreshToken:   "temp_refresh_token",
// 				IsTempPassword: true,
// 			},
// 			config: &config.Config{
// 				AuthConfig: config.AuthConfig{
// 					RefreshTokenDuration: time.Hour * 12,
// 					CookieDomain:         "localhost",
// 					CookieRejectHTTP:     true,
// 				},
// 			},
// 			expectedResp: &entities.SignInResponse{
// 				ID:             "user456",
// 				AccessToken:    "temp_access_token",
// 				IsTempPassword: true,
// 			},
// 			expectError: false,
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			rr := httptest.NewRecorder()
// 			logger := log.Initialize(constants.TestAppEnv)
// 			c, _ := gin.CreateTestContext(rr)

// 			cookieUtil := NewCookies(tt.config, logger)
// 			result, err := cookieUtil.SetCookie(c, tt.serviceResp)

// 			if tt.expectError {
// 				assert.Error(t, err)
// 				assert.Nil(t, result)
// 			} else {
// 				assert.NoError(t, err)
// 				assert.NotNil(t, result)
// 				assert.Equal(t, tt.expectedResp.ID, result.ID)
// 				assert.Equal(t, tt.expectedResp.AccessToken, result.AccessToken)
// 				assert.Equal(t, tt.expectedResp.IsTempPassword, result.IsTempPassword)

// 				// Verify that refresh token cookie was set
// 				cookies := rr.Result().Cookies()
// 				require.Len(t, cookies, 1)

// 				actualCookie := cookies[0]
// 				assert.Equal(t, "refresh_token", actualCookie.Name)
// 				assert.Equal(t, tt.serviceResp.RefreshToken, actualCookie.Value)
// 				assert.Equal(t, "/", actualCookie.Path)
// 				assert.True(t, actualCookie.HttpOnly)
// 				assert.Equal(t, !tt.config.AuthConfig.CookieRejectHTTP, actualCookie.Secure)
// 				assert.Equal(t, http.SameSiteLaxMode, actualCookie.SameSite)
// 				assert.Equal(t, tt.config.AuthConfig.CookieDomain, actualCookie.Domain)
// 			}
// 		})
// 	}
// }
