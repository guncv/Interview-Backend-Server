package utils

import (
	"context"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gitlab.com/interview-simulation/interview-backend-server/internal/config"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	db "gitlab.com/interview-simulation/interview-backend-server/internal/db/sqlc"
	"gitlab.com/interview-simulation/interview-backend-server/internal/entities"
	app_error "gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
)

func TestCreateAndVerifyTokens(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	cfg := &config.Config{
		AuthConfig: config.AuthConfig{
			EncryptionSecretKey: "test_secret",
		},
	}

	ctx := context.Background()

	testCases := []struct {
		name   string
		input  func() *entities.TokenRequest
		verify func(t *testing.T, got *SignInTokenPayload, gotErr error)
	}{
		{
			name: "CreateAndVerifyToken_OK",
			input: func() *entities.TokenRequest {
				return &entities.TokenRequest{
					UserID:   "user-123",
					Role:     "user",
					Duration: time.Minute * 10,
				}
			},
			verify: func(t *testing.T, got *SignInTokenPayload, gotErr error) {
				assert.Equal(t, got.UserID, "user-123")
				assert.Equal(t, got.Role, constants.UserRole("user"))
				assert.WithinDuration(t, got.IssuedAt, time.Now(), time.Second)
				assert.WithinDuration(t, got.ExpiredAt, time.Now().Add(time.Minute*10), time.Second)
				assert.NoError(t, gotErr)
			},
		},
		{
			name: "CreateAndVerifyToken_OK",
			input: func() *entities.TokenRequest {
				return &entities.TokenRequest{
					UserID:   "user-123",
					Role:     "user",
					Duration: time.Second * -10,
				}
			},
			verify: func(t *testing.T, got *SignInTokenPayload, gotErr error) {
				assert.Equal(t, got.UserID, "user-123")
				assert.Equal(t, got.Role, constants.UserRole("user"))
				assert.WithinDuration(t, got.IssuedAt, time.Now(), time.Second)
				assert.WithinDuration(t, got.ExpiredAt, time.Now().Add(time.Second*-10), time.Second)
				assert.NoError(t, gotErr)
			},
		},
		{
			name: "CreateAndVerifyToken_Expired",
			input: func() *entities.TokenRequest {
				return &entities.TokenRequest{
					UserID:   "user-123",
					Role:     "user",
					Duration: time.Minute * -10,
				}
			},
			verify: func(t *testing.T, got *SignInTokenPayload, gotErr error) {
				assert.Nil(t, got)
				assert.Error(t, gotErr)
				assert.Equal(t, gotErr, app_error.New(constants.ErrExpiredToken, app_error.ErrCodeAuthExpiredToken)) // check the correct error
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			svc := NewJwtToken(cfg, lgr, nil)
			got, _, gotErr := svc.CreateToken(ctx, tC.input())
			assert.NoError(t, gotErr)
			gotPayload, gotErr := svc.VerifyToken(ctx, got, cfg.AuthConfig.EncryptionSecretKey)
			tC.verify(t, gotPayload, gotErr)
		})
	}
}

func TestVerifyTokens(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	cfg := &config.Config{
		AuthConfig: config.AuthConfig{},
	}

	ctx := context.Background()
	svc := NewJwtToken(cfg, lgr, nil)

	testCases := []struct {
		name   string
		input  func() string
		verify func(t *testing.T, got *SignInTokenPayload, gotErr error)
	}{
		{
			name: "VerifyToken_InvalidToken",
			input: func() string {
				return "invalid_token"
			},
			verify: func(t *testing.T, got *SignInTokenPayload, gotErr error) {
				assert.Nil(t, got)
				assert.Error(t, gotErr)
				assert.Equal(t, gotErr, app_error.New(constants.ErrInvalidToken, app_error.ErrCodeAuthInvalidToken))
			},
		},
		{
			name: "VerifyToken_InvaidTokenPayload",
			input: func() string {
				token, _, err := svc.CreateToken(ctx, &entities.TokenRequest{
					UserID:   "user-123",
					Role:     constants.UserRole("user"),
					Duration: time.Minute * -10, // Expired token
				})
				assert.NoError(t, err)
				assert.NotNil(t, token)
				return token
			},
			verify: func(t *testing.T, got *SignInTokenPayload, gotErr error) {
				assert.Nil(t, got)
				assert.Error(t, gotErr)
				assert.Equal(t, gotErr, app_error.New(constants.ErrExpiredToken, app_error.ErrCodeAuthExpiredToken))
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			gotPayload, gotErr := svc.VerifyToken(ctx, tC.input(), cfg.AuthConfig.EncryptionSecretKey)
			tC.verify(t, gotPayload, gotErr)
		})
	}
}
func TestCreateToken_EdgeCases(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	cfg := &config.Config{
		AuthConfig: config.AuthConfig{
			EncryptionSecretKey: "test_secret",
		},
	}

	ctx := context.Background()
	svc := NewJwtToken(cfg, lgr, nil)

	testCases := []struct {
		name   string
		input  *entities.TokenRequest
		verify func(t *testing.T, token string, payload *SignInTokenPayload, err error)
	}{
		{
			name: "CreateToken_EmptyUserID",
			input: &entities.TokenRequest{
				UserID:   "",
				Role:     constants.UserRole("user"),
				Duration: time.Minute * 10,
			},
			verify: func(t *testing.T, token string, payload *SignInTokenPayload, err error) {
				assert.NoError(t, err) // Should still create token with empty UserID
				assert.NotEmpty(t, token)
				assert.NotNil(t, payload)
				assert.Equal(t, "", payload.UserID)
			},
		},
		{
			name: "CreateToken_ZeroDuration",
			input: &entities.TokenRequest{
				UserID:   "user-123",
				Role:     constants.UserRole("user"),
				Duration: 0,
			},
			verify: func(t *testing.T, token string, payload *SignInTokenPayload, err error) {
				assert.NoError(t, err)
				assert.NotEmpty(t, token)
				assert.NotNil(t, payload)
				// Should be expired immediately
				assert.True(t, time.Now().After(payload.ExpiredAt))
			},
		},
		{
			name: "CreateToken_LongDuration",
			input: &entities.TokenRequest{
				UserID:   "user-123",
				Role:     constants.UserRole("admin"),
				Duration: time.Hour * 24 * 365, // 1 year
			},
			verify: func(t *testing.T, token string, payload *SignInTokenPayload, err error) {
				assert.NoError(t, err)
				assert.NotEmpty(t, token)
				assert.NotNil(t, payload)
				assert.Equal(t, constants.UserRole("admin"), payload.Role)
				expectedExpiry := time.Now().Add(time.Hour * 24 * 365)
				assert.WithinDuration(t, expectedExpiry, payload.ExpiredAt, time.Minute)
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			token, payload, err := svc.CreateToken(ctx, tC.input)
			tC.verify(t, token, payload, err)
		})
	}
}

func TestVerifyToken_EdgeCases(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	cfg := &config.Config{
		AuthConfig: config.AuthConfig{
			EncryptionSecretKey: "test_secret",
		},
	}

	ctx := context.Background()
	svc := NewJwtToken(cfg, lgr, nil)

	testCases := []struct {
		name   string
		input  func() string
		verify func(t *testing.T, payload *SignInTokenPayload, err error)
	}{
		{
			name: "VerifyToken_EmptyToken",
			input: func() string {
				return ""
			},
			verify: func(t *testing.T, payload *SignInTokenPayload, err error) {
				assert.Error(t, err)
				assert.Nil(t, payload)
				assert.Equal(t, app_error.New(constants.ErrInvalidToken, app_error.ErrCodeAuthInvalidToken), err)
			},
		},
		{
			name: "VerifyToken_MalformedToken",
			input: func() string {
				return "this.is.not.a.valid.jwt.token"
			},
			verify: func(t *testing.T, payload *SignInTokenPayload, err error) {
				assert.Error(t, err)
				assert.Nil(t, payload)
				assert.Equal(t, app_error.New(constants.ErrInvalidToken, app_error.ErrCodeAuthInvalidToken), err)
			},
		},
		{
			name: "VerifyToken_TokenWithDifferentSecret",
			input: func() string {
				// Create token with different service (different secret)
				differentCfg := &config.Config{
					AuthConfig: config.AuthConfig{
						EncryptionSecretKey: "different_secret",
					},
				}
				differentSvc := NewJwtToken(differentCfg, lgr, nil)

				token, _, err := differentSvc.CreateToken(ctx, &entities.TokenRequest{
					UserID:   "user-123",
					Role:     constants.UserRole("user"),
					Duration: time.Minute * 10,
				})
				assert.NoError(t, err)
				return token
			},
			verify: func(t *testing.T, payload *SignInTokenPayload, err error) {
				assert.Error(t, err)
				assert.Nil(t, payload)
				assert.Equal(t, app_error.New(constants.ErrInvalidToken, app_error.ErrCodeAuthInvalidToken), err)
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			token := tC.input()
			payload, err := svc.VerifyToken(ctx, token, cfg.AuthConfig.EncryptionSecretKey)
			tC.verify(t, payload, err)
		})
	}
}

func TestHashTokenSHA256_EdgeCases(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()
	svc := NewJwtToken(nil, lgr, nil)

	testCases := []struct {
		name   string
		input  string
		verify func(t *testing.T, result string)
	}{
		{
			name:  "HashTokenSHA256_EmptyString",
			input: "",
			verify: func(t *testing.T, result string) {
				assert.NotEmpty(t, result)
				// SHA256 of empty string should be consistent
				assert.Equal(t, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", result)
			},
		},
		{
			name:  "HashTokenSHA256_LongString",
			input: "this_is_a_very_long_token_string_that_might_cause_issues_if_not_handled_properly_but_should_work_fine_with_sha256",
			verify: func(t *testing.T, result string) {
				assert.NotEmpty(t, result)
				assert.Len(t, result, 64) // SHA256 hex string is always 64 characters
			},
		},
		{
			name:  "HashTokenSHA256_SpecialCharacters",
			input: "token_with_special_chars!@#$%^&*()[]{}|\\:;\"'<>?,./",
			verify: func(t *testing.T, result string) {
				assert.NotEmpty(t, result)
				assert.Len(t, result, 64)
			},
		},
		{
			name:  "HashTokenSHA256_Unicode",
			input: "token_with_unicode_🚀🎉🔥",
			verify: func(t *testing.T, result string) {
				assert.NotEmpty(t, result)
				assert.Len(t, result, 64)
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			result := svc.HashTokenSHA256(ctx, tC.input)
			tC.verify(t, result)
		})
	}
}

func TestIsTokenMatch_EdgeCases(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()
	svc := NewJwtToken(nil, lgr, nil)

	testCases := []struct {
		name     string
		token1   string
		token2   string
		expected bool
	}{
		{
			name:     "IsTokenMatch_BothEmpty",
			token1:   "",
			token2:   "",
			expected: true, // Hash of empty string should match hash of empty string
		},
		{
			name:     "IsTokenMatch_IdenticalLongTokens",
			token1:   "very_long_identical_token_string_that_should_match_perfectly",
			token2:   "very_long_identical_token_string_that_should_match_perfectly",
			expected: true,
		},
		{
			name:     "IsTokenMatch_CaseSensitive",
			token1:   "TestToken",
			token2:   "testtoken",
			expected: false,
		},
		{
			name:     "IsTokenMatch_WithSpaces",
			token1:   "token with spaces",
			token2:   "token with spaces",
			expected: true,
		},
		{
			name:     "IsTokenMatch_SpecialCharacters",
			token1:   "token!@#$%^&*()",
			token2:   "token!@#$%^&*()",
			expected: true,
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			// For edge cases where we're not using hashed token2, hash it first
			hashedToken2 := svc.HashTokenSHA256(ctx, tC.token2)
			result := svc.IsTokenMatch(ctx, tC.token1, hashedToken2)
			assert.Equal(t, tC.expected, result)
		})
	}
}

func TestHashTokenSHA256(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)

	ctx := context.Background()

	testCases := []struct {
		name   string
		input  func() string
		verify func(t *testing.T, got string, gotErr error)
	}{
		{
			name: "HashTokenSHA256_OK",
			input: func() string {
				return "test_token"
			},
			verify: func(t *testing.T, got string, gotErr error) {
				assert.NotNil(t, got)
				assert.NotEqual(t, got, "test_token")
				assert.NoError(t, gotErr)
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			svc := NewJwtToken(nil, lgr, nil)
			got := svc.HashTokenSHA256(ctx, tC.input())
			tC.verify(t, got, nil)
		})
	}
}

func TestIsTokenMatch(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()

	svc := NewJwtToken(nil, lgr, nil)
	testCases := []struct {
		name   string
		input  func() (string, string)
		verify func(t *testing.T, got bool)
	}{
		{
			name: "IsTokenMatch_OK",
			input: func() (string, string) {
				hashedToken := svc.HashTokenSHA256(ctx, "test_token")
				return "test_token", hashedToken
			},
			verify: func(t *testing.T, got bool) {
				assert.True(t, got)
			},
		},
		{
			name: "IsTokenMatch_Fail",
			input: func() (string, string) {
				hashedToken := svc.HashTokenSHA256(ctx, "different_token")
				return "test_token", hashedToken
			},
			verify: func(t *testing.T, got bool) {
				assert.False(t, got)
			},
		},
		{
			name: "IsTokenMatch_EmptyTokens",
			input: func() (string, string) {
				return "", ""
			},
			verify: func(t *testing.T, got bool) {
				assert.False(t, got)
			},
		},
		{
			name: "IsTokenMatch_OneEmptyToken",
			input: func() (string, string) {
				return "", "test_token"
			},
			verify: func(t *testing.T, got bool) {
				assert.False(t, got)
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			token1, token2 := tC.input()
			got := svc.IsTokenMatch(ctx, token1, token2)
			tC.verify(t, got)
		})
	}
}

func TestRenewVerifyEmailToken(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	cfg := &config.Config{
		AuthConfig: config.AuthConfig{
			EncryptionSecretKey:        "test_secret",
			ResetPasswordTokenDuration: time.Minute * 10,
		},
		EmailConfig: config.EmailConfig{
			EncryptionSecretKey:      "test_email_secret",
			VerifyEmailTokenDuration: time.Minute * 10,
		},
	}

	ctx := context.Background()
	svc := NewJwtToken(cfg, lgr, nil)

	testCases := []struct {
		name   string
		input  func() string
		verify func(t *testing.T, newToken string, payload *VerifyEmailTokenPayload, err error)
	}{
		{
			name: "RenewVerifyEmailToken_ValidExpiredToken",
			input: func() string {
				req := &entities.VerifyEmailTokenRequest{
					UserID:   "user-123",
					Email:    "test@example.com",
					Duration: time.Minute * 10,
				}
				_, payload, err := svc.CreateVerifyEmailToken(ctx, req)
				assert.NoError(t, err)

				payload.ExpiredAt = time.Now().Add(-time.Minute * 10)

				expiredToken := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)
				signedToken, err := expiredToken.SignedString([]byte(cfg.EmailConfig.EncryptionSecretKey))
				assert.NoError(t, err)
				return signedToken
			},
			verify: func(t *testing.T, newToken string, payload *VerifyEmailTokenPayload, err error) {
				assert.NoError(t, err)
				assert.NotEmpty(t, newToken)
				assert.NotNil(t, payload)
				assert.Equal(t, "user-123", payload.UserID)
				assert.Equal(t, "test@example.com", payload.Email)
				expectedExpiry := time.Now().Add(time.Minute * 10)
				assert.WithinDuration(t, expectedExpiry, payload.ExpiredAt, time.Minute)
			},
		},
		{
			name: "RenewVerifyEmailToken_ValidNonExpiredToken",
			input: func() string {
				req := &entities.VerifyEmailTokenRequest{
					UserID:   "user-456",
					Email:    "valid@example.com",
					Duration: time.Minute * 10,
				}
				token, _, err := svc.CreateVerifyEmailToken(ctx, req)
				assert.NoError(t, err)
				return token
			},
			verify: func(t *testing.T, newToken string, payload *VerifyEmailTokenPayload, err error) {
				assert.NoError(t, err)
				assert.NotEmpty(t, newToken)
				assert.NotNil(t, payload)
				assert.Equal(t, "user-456", payload.UserID)
				assert.Equal(t, "valid@example.com", payload.Email)
				expectedExpiry := time.Now().Add(time.Minute * 10)
				assert.WithinDuration(t, expectedExpiry, payload.ExpiredAt, time.Minute)
			},
		},
		{
			name: "RenewVerifyEmailToken_InvalidToken",
			input: func() string {
				return "invalid_token_string"
			},
			verify: func(t *testing.T, newToken string, payload *VerifyEmailTokenPayload, err error) {
				assert.Error(t, err)
				assert.Empty(t, newToken)
				assert.Nil(t, payload)
				assert.Contains(t, err.Error(), "token is invalid")
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			oldToken := tC.input()
			newToken, payload, err := svc.RenewVerifyEmailToken(ctx, oldToken)
			tC.verify(t, newToken, payload, err)
		})
	}
}

func TestJwtToken_GraceWindow(t *testing.T) {
	cfg := &config.Config{
		AuthConfig: config.AuthConfig{
			EncryptionSecretKey: "test-secret-key",
			TokenGraceWindow:    time.Minute,
		},
	}

	logger := log.Initialize(constants.TestAppEnv)

	mockSessionRepo := &mockSessionRepository{}

	jwtMaker := NewJwtToken(cfg, logger, mockSessionRepo)
	ctx := context.Background()

	t.Run("Token within grace window should be valid", func(t *testing.T) {
		req := &entities.TokenRequest{
			UserID:   "test-user",
			Role:     constants.UserRoleUser,
			Duration: time.Minute,
		}

		token, payload, err := jwtMaker.CreateToken(ctx, req)
		assert.NoError(t, err)
		assert.NotEmpty(t, token)
		assert.NotNil(t, payload)

		payload.ExpiredAt = time.Now().Add(-30 * time.Second)
		t.Logf("Modified payload expiration: %v", payload.ExpiredAt)
		newToken, err := jwtMaker.CreateJWTToken(ctx, payload, cfg.AuthConfig.EncryptionSecretKey)
		assert.NoError(t, err)
		t.Logf("Created new token with expired payload: %s", newToken[:20])

		verifiedPayload, err := jwtMaker.VerifyToken(ctx, newToken, cfg.AuthConfig.EncryptionSecretKey)
		if err != nil {
			t.Logf("Verification failed: %v", err)
		}
		assert.NoError(t, err)
		assert.NotNil(t, verifiedPayload)
		assert.Equal(t, req.UserID, verifiedPayload.UserID)
	})

	t.Run("Valid token should work normally", func(t *testing.T) {
		req := &entities.TokenRequest{
			UserID:   "test-user",
			Role:     constants.UserRoleUser,
			Duration: time.Minute,
		}

		token, payload, err := jwtMaker.CreateToken(ctx, req)
		assert.NoError(t, err)
		assert.NotEmpty(t, token)
		assert.NotNil(t, payload)

		verifiedPayload, err := jwtMaker.VerifyToken(ctx, token, cfg.AuthConfig.EncryptionSecretKey)
		assert.NoError(t, err)
		assert.NotNil(t, verifiedPayload)
		assert.Equal(t, req.UserID, verifiedPayload.UserID)
	})
}

func TestTokenPayload_GraceWindow(t *testing.T) {
	t.Run("SignInTokenPayload grace window validation", func(t *testing.T) {
		payload := &SignInTokenPayload{
			ID:        uuid.New(),
			UserID:    "test-user",
			Role:      constants.UserRoleUser,
			IssuedAt:  time.Now().Add(-2 * time.Minute),
			ExpiredAt: time.Now().Add(-30 * time.Second),
		}

		err := payload.Valid()
		assert.Error(t, err)
		assert.Equal(t, constants.ErrExpiredToken, err)

		err = payload.ValidWithGraceWindow()
		assert.NoError(t, err)
	})

	t.Run("VerifyEmailTokenPayload grace window validation", func(t *testing.T) {
		payload := &VerifyEmailTokenPayload{
			ID:        uuid.New(),
			UserID:    "test-user",
			Email:     "test@example.com",
			IssuedAt:  time.Now().Add(-2 * time.Minute),
			ExpiredAt: time.Now().Add(-30 * time.Second),
		}

		err := payload.Valid()
		assert.Error(t, err)
		assert.Equal(t, constants.ErrExpiredToken, err)

		err = payload.ValidWithGraceWindow()
		assert.NoError(t, err)

	})
}

type mockSessionRepository struct{}

func (m *mockSessionRepository) CreateSession(ctx context.Context, arg *db.CreateSessionParams) error {
	return nil
}

func (m *mockSessionRepository) GetSessionByID(ctx context.Context, id uuid.UUID) (db.Sessions, error) {
	return db.Sessions{}, nil
}

func (m *mockSessionRepository) RevokeSessionByID(ctx context.Context, id uuid.UUID) error {
	return nil
}
