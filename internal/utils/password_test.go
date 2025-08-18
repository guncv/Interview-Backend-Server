package utils

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
)

func TestNewPassword(t *testing.T) {
	logger := log.Initialize(constants.TestAppEnv)
	passwordUtil := NewPassword(logger)

	assert.NotNil(t, passwordUtil)
	assert.IsType(t, &bcryptPassword{}, passwordUtil)
}

func TestHashPassword(t *testing.T) {
	logger := log.Initialize(constants.TestAppEnv)
	passwordUtil := NewPassword(logger)
	ctx := context.Background()

	tests := []struct {
		name        string
		password    string
		expectError bool
	}{
		{
			name:        "Hash valid password",
			password:    "validPassword123",
			expectError: false,
		},
		{
			name:        "Hash password with special characters",
			password:    "P@ssw0rd!@#$%^&*()",
			expectError: false,
		},
		{
			name:        "Hash password with unicode characters",
			password:    "password世界",
			expectError: false,
		},
		{
			name:        "Hash password with spaces",
			password:    "password with spaces",
			expectError: false,
		},
		{
			name:        "Hash password with newlines",
			password:    "password\nwith\nnewlines",
			expectError: false,
		},
		{
			name:        "Hash password with tabs",
			password:    "password\twith\ttabs",
			expectError: false,
		},
		{
			name:        "Hash empty password",
			password:    "",
			expectError: false,
		},
		{
			name:        "Hash very long password",
			password:    strings.Repeat("a", 1000),
			expectError: true,
		},
		{
			name:        "Hash password with only numbers",
			password:    "1234567890",
			expectError: false,
		},
		{
			name:        "Hash password with only letters",
			password:    "abcdefghijklmnopqrstuvwxyz",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hashedPassword, err := passwordUtil.HashPassword(ctx, tt.password)

			if tt.expectError {
				assert.Error(t, err)
				assert.Empty(t, hashedPassword)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, hashedPassword)
				assert.NotEqual(t, tt.password, hashedPassword)

				// Verify the hashed password is different from the original
				assert.True(t, len(hashedPassword) > len(tt.password))

				// Verify the hash starts with the bcrypt identifier
				assert.True(t, strings.HasPrefix(hashedPassword, "$2a$"))
			}
		})
	}
}

func TestHashPasswordWithNilContext(t *testing.T) {
	logger := log.Initialize(constants.TestAppEnv)
	passwordUtil := NewPassword(logger)

	// Should not panic with nil context
	assert.NotPanics(t, func() {
		hashedPassword, err := passwordUtil.HashPassword(nil, "testPassword")
		assert.NoError(t, err)
		assert.NotEmpty(t, hashedPassword)
	})
}

func TestCheckPassword(t *testing.T) {
	logger := log.Initialize(constants.TestAppEnv)
	passwordUtil := NewPassword(logger)
	ctx := context.Background()

	tests := []struct {
		name           string
		password       string
		hashedPassword string
		expectError    bool
	}{
		{
			name:           "Check correct password",
			password:       "correctPassword123",
			hashedPassword: "",
			expectError:    false,
		},
		{
			name:           "Check incorrect password",
			password:       "wrongPassword",
			hashedPassword: "",
			expectError:    true,
		},
		{
			name:           "Check password with special characters",
			password:       "P@ssw0rd!@#$%^&*()",
			hashedPassword: "",
			expectError:    false,
		},
		{
			name:           "Check password with unicode characters",
			password:       "password世界",
			hashedPassword: "",
			expectError:    false,
		},
		{
			name:           "Check empty password",
			password:       "",
			hashedPassword: "",
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "Check incorrect password" {
				// For incorrect password test, hash a different password first
				originalPassword := "correctPassword123"
				hashedPassword, err := passwordUtil.HashPassword(ctx, originalPassword)
				require.NoError(t, err)
				require.NotEmpty(t, hashedPassword)

				// Then check with wrong password - should return error
				err = passwordUtil.CheckPassword(ctx, tt.password, hashedPassword)
				assert.Error(t, err)
				// Verify it's the correct error type
				var appErr *app_error.AppError
				assert.ErrorAs(t, err, &appErr)
				assert.Equal(t, app_error.ErrCodeAuthInvalidPassword, appErr.Code)
			} else {
				// First hash the password
				hashedPassword, err := passwordUtil.HashPassword(ctx, tt.password)
				require.NoError(t, err)
				require.NotEmpty(t, hashedPassword)

				// Then check the password
				err = passwordUtil.CheckPassword(ctx, tt.password, hashedPassword)

				if tt.expectError {
					assert.Error(t, err)
					// Verify it's the correct error type
					var appErr *app_error.AppError
					assert.ErrorAs(t, err, &appErr)
					assert.Equal(t, app_error.ErrCodeAuthInvalidPassword, appErr.Code)
				} else {
					assert.NoError(t, err)
				}
			}
		})
	}
}

func TestCheckPasswordWithIncorrectHash(t *testing.T) {
	logger := log.Initialize(constants.TestAppEnv)
	passwordUtil := NewPassword(logger)
	ctx := context.Background()

	// Test with a completely wrong hash
	wrongHash := "$2a$10$wronghashformat"
	err := passwordUtil.CheckPassword(ctx, "testPassword", wrongHash)

	assert.Error(t, err)
	var appErr *app_error.AppError
	assert.ErrorAs(t, err, &appErr)
	assert.Equal(t, app_error.ErrCodeAuthInvalidPassword, appErr.Code)
}

func TestCheckPasswordWithNilContext(t *testing.T) {
	logger := log.Initialize(constants.TestAppEnv)
	passwordUtil := NewPassword(logger)

	// First hash a password
	hashedPassword, err := passwordUtil.HashPassword(context.Background(), "testPassword")
	require.NoError(t, err)

	// Should not panic with nil context
	assert.NotPanics(t, func() {
		err := passwordUtil.CheckPassword(nil, "testPassword", hashedPassword)
		assert.NoError(t, err)
	})
}

func TestPasswordInterface(t *testing.T) {
	var _ Password = (*bcryptPassword)(nil)
}

func TestHashPasswordConsistency(t *testing.T) {
	logger := log.Initialize(constants.TestAppEnv)
	passwordUtil := NewPassword(logger)
	ctx := context.Background()
	password := "testPassword123"

	// Hash the same password multiple times
	hashes := make(map[string]bool)
	for i := 0; i < 100; i++ {
		hash, err := passwordUtil.HashPassword(ctx, password)
		require.NoError(t, err)

		// Each hash should be unique due to salt
		assert.False(t, hashes[hash], "Duplicate hash generated")
		hashes[hash] = true

		// Each hash should be valid
		err = passwordUtil.CheckPassword(ctx, password, hash)
		assert.NoError(t, err)
	}
}

func TestHashPasswordPerformance(t *testing.T) {
	logger := log.Initialize(constants.TestAppEnv)
	passwordUtil := NewPassword(logger)
	ctx := context.Background()
	password := "performanceTestPassword123"

	// Test performance with multiple hashes
	for i := 0; i < 100; i++ {
		hash, err := passwordUtil.HashPassword(ctx, password)
		require.NoError(t, err)
		require.NotEmpty(t, hash)
	}
}

func TestCheckPasswordPerformance(t *testing.T) {
	logger := log.Initialize(constants.TestAppEnv)
	passwordUtil := NewPassword(logger)
	ctx := context.Background()
	password := "performanceTestPassword123"

	// Hash the password once
	hash, err := passwordUtil.HashPassword(ctx, password)
	require.NoError(t, err)

	// Test performance with multiple checks
	for i := 0; i < 1000; i++ {
		err := passwordUtil.CheckPassword(ctx, password, hash)
		assert.NoError(t, err)
	}
}

func TestPasswordSecurity(t *testing.T) {
	logger := log.Initialize(constants.TestAppEnv)
	passwordUtil := NewPassword(logger)
	ctx := context.Background()

	// Test that similar passwords produce different hashes
	password1 := "password123"
	password2 := "password124"

	hash1, err := passwordUtil.HashPassword(ctx, password1)
	require.NoError(t, err)

	hash2, err := passwordUtil.HashPassword(ctx, password2)
	require.NoError(t, err)

	// Hashes should be different
	assert.NotEqual(t, hash1, hash2)

	// Each hash should only work with its own password
	err = passwordUtil.CheckPassword(ctx, password1, hash2)
	assert.Error(t, err)

	err = passwordUtil.CheckPassword(ctx, password2, hash1)
	assert.Error(t, err)
}

func TestPasswordWithDifferentContexts(t *testing.T) {
	logger := log.Initialize(constants.TestAppEnv)
	passwordUtil := NewPassword(logger)
	password := "testPassword123"

	// Test with different contexts
	contexts := []context.Context{
		context.Background(),
		context.WithValue(context.Background(), "key", "value"),
		nil,
	}

	for _, ctx := range contexts {
		t.Run("Context test", func(t *testing.T) {
			hash, err := passwordUtil.HashPassword(ctx, password)
			if ctx != nil {
				assert.NoError(t, err)
				assert.NotEmpty(t, hash)

				err = passwordUtil.CheckPassword(ctx, password, hash)
				assert.NoError(t, err)
			} else {
				// Nil context should still work
				assert.NoError(t, err)
				assert.NotEmpty(t, hash)
			}
		})
	}
}
