package utils

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
)

func TestNewPassword(t *testing.T) {
	logger := log.Initialize(constants.TestAppEnv)
	passwordUtil := NewPassword(logger)

	assert.NotNil(t, passwordUtil)
	// Check that it implements the PasswordUtil interface
	var _ PasswordUtil = passwordUtil
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
		hashedPassword, err := passwordUtil.HashPassword(context.Background(), "testPassword")
		assert.NoError(t, err)
		assert.NotEmpty(t, hashedPassword)
	})
}

func TestIsPasswordValid(t *testing.T) {
	logger := log.Initialize(constants.TestAppEnv)
	passwordUtil := NewPassword(logger)
	ctx := context.Background()

	tests := []struct {
		name           string
		password       string
		hashedPassword string
		expectValid    bool
	}{
		{
			name:           "Check correct password",
			password:       "correctPassword123",
			hashedPassword: "",
			expectValid:    true,
		},
		{
			name:           "Check incorrect password",
			password:       "wrongPassword",
			hashedPassword: "",
			expectValid:    false,
		},
		{
			name:           "Check password with special characters",
			password:       "P@ssw0rd!@#$%^&*()",
			hashedPassword: "",
			expectValid:    true,
		},
		{
			name:           "Check password with unicode characters",
			password:       "password世界",
			hashedPassword: "",
			expectValid:    true,
		},
		{
			name:           "Check empty password",
			password:       "",
			hashedPassword: "",
			expectValid:    true,
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

				// Then check with wrong password - should return false
				isValid := passwordUtil.IsPasswordValid(ctx, tt.password, hashedPassword)
				assert.False(t, isValid)
			} else {
				// First hash the password
				hashedPassword, err := passwordUtil.HashPassword(ctx, tt.password)
				require.NoError(t, err)
				require.NotEmpty(t, hashedPassword)

				// Then check the password
				isValid := passwordUtil.IsPasswordValid(ctx, tt.password, hashedPassword)

				if tt.expectValid {
					assert.True(t, isValid)
				} else {
					assert.False(t, isValid)
				}
			}
		})
	}
}

func TestIsPasswordValidWithIncorrectHash(t *testing.T) {
	logger := log.Initialize(constants.TestAppEnv)
	passwordUtil := NewPassword(logger)
	ctx := context.Background()

	// Test with a completely wrong hash
	wrongHash := "$2a$10$wronghashformat"
	isValid := passwordUtil.IsPasswordValid(ctx, "testPassword", wrongHash)

	assert.False(t, isValid)
}

func TestIsPasswordValidWithNilContext(t *testing.T) {
	logger := log.Initialize(constants.TestAppEnv)
	passwordUtil := NewPassword(logger)

	// First hash a password
	hashedPassword, err := passwordUtil.HashPassword(context.Background(), "testPassword")
	require.NoError(t, err)

	// Should not panic with nil context
	assert.NotPanics(t, func() {
		isValid := passwordUtil.IsPasswordValid(context.Background(), "testPassword", hashedPassword)
		assert.True(t, isValid)
	})
}

func TestPasswordInterface(t *testing.T) {
	var _ PasswordUtil = (*passwordUtil)(nil)
}

func TestHashPasswordConsistency(t *testing.T) {
	logger := log.Initialize(constants.TestAppEnv)
	passwordUtil := NewPassword(logger)
	ctx := context.Background()
	password := "testPassword123"

	// Hash the same password multiple times
	hashes := make(map[string]bool)
	for i := 0; i < 10; i++ {
		hash, err := passwordUtil.HashPassword(ctx, password)
		require.NoError(t, err)

		// Each hash should be unique due to salt
		assert.False(t, hashes[hash], "Duplicate hash generated")
		hashes[hash] = true

		// Each hash should be valid
		isValid := passwordUtil.IsPasswordValid(ctx, password, hash)
		assert.True(t, isValid)
	}
}

func TestHashPasswordPerformance(t *testing.T) {
	logger := log.Initialize(constants.TestAppEnv)
	passwordUtil := NewPassword(logger)
	ctx := context.Background()
	password := "performanceTestPassword123"

	// Test performance with multiple hashes
	for i := 0; i < 10; i++ {
		hash, err := passwordUtil.HashPassword(ctx, password)
		require.NoError(t, err)
		require.NotEmpty(t, hash)
	}
}

func TestIsPasswordValidPerformance(t *testing.T) {
	logger := log.Initialize(constants.TestAppEnv)
	passwordUtil := NewPassword(logger)
	ctx := context.Background()
	password := "performanceTestPassword123"

	// Hash the password once
	hash, err := passwordUtil.HashPassword(ctx, password)
	require.NoError(t, err)

	// Test performance with multiple checks
	for i := 0; i < 10; i++ {
		isValid := passwordUtil.IsPasswordValid(ctx, password, hash)
		assert.True(t, isValid)
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
	isValid1 := passwordUtil.IsPasswordValid(ctx, password1, hash2)
	assert.False(t, isValid1)

	isValid2 := passwordUtil.IsPasswordValid(ctx, password2, hash1)
	assert.False(t, isValid2)
}

func TestPasswordWithDifferentContexts(t *testing.T) {
	logger := log.Initialize(constants.TestAppEnv)
	passwordUtil := NewPassword(logger)
	password := "testPassword123"

	// Test with different contexts
	type contextKey string
	const testKey contextKey = "key"

	contexts := []context.Context{
		context.Background(),
		context.WithValue(context.Background(), testKey, "value"),
		nil,
	}

	for _, ctx := range contexts {
		t.Run("Context test", func(t *testing.T) {
			hash, err := passwordUtil.HashPassword(ctx, password)
			if ctx != nil {
				assert.NoError(t, err)
				assert.NotEmpty(t, hash)

				isValid := passwordUtil.IsPasswordValid(ctx, password, hash)
				assert.True(t, isValid)
			} else {
				// Nil context should still work
				assert.NoError(t, err)
				assert.NotEmpty(t, hash)
			}
		})
	}
}
