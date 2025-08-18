package utils

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	"gitlab.com/interview-simulation/interview-backend-server/internal/entities"
)

func TestNewSignInTokenPayload(t *testing.T) {
	tests := []struct {
		name        string
		request     *entities.TokenRequest
		expectError bool
	}{
		{
			name: "Create sign in token payload successfully",
			request: &entities.TokenRequest{
				UserID:   "user123",
				Role:     constants.UserRoleUser,
				Duration: time.Hour * 24,
			},
			expectError: false,
		},
		{
			name: "Create sign in token payload with admin role",
			request: &entities.TokenRequest{
				UserID:   "admin456",
				Role:     constants.UserRoleAdmin,
				Duration: time.Hour * 12,
			},
			expectError: false,
		},
		{
			name: "Create sign in token payload with short duration",
			request: &entities.TokenRequest{
				UserID:   "user789",
				Role:     constants.UserRoleUser,
				Duration: time.Minute * 30,
			},
			expectError: false,
		},
		{
			name: "Create sign in token payload with long duration",
			request: &entities.TokenRequest{
				UserID:   "user101",
				Role:     constants.UserRoleUser,
				Duration: time.Hour * 168, // 1 week
			},
			expectError: false,
		},
		{
			name: "Create sign in token payload with zero duration",
			request: &entities.TokenRequest{
				UserID:   "user202",
				Role:     constants.UserRoleUser,
				Duration: 0,
			},
			expectError: false,
		},
		{
			name: "Create sign in token payload with negative duration",
			request: &entities.TokenRequest{
				UserID:   "user303",
				Role:     constants.UserRoleUser,
				Duration: -time.Hour,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload, err := NewSignInTokenPayload(tt.request)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, payload)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, payload)

				// Verify payload fields
				assert.NotEqual(t, uuid.Nil, payload.ID)
				assert.Equal(t, tt.request.UserID, payload.UserID)
				assert.Equal(t, tt.request.Role, payload.Role)

				// Verify timestamps
				assert.True(t, payload.IssuedAt.Before(time.Now().Add(time.Second)) || payload.IssuedAt.Equal(time.Now()))
				assert.True(t, payload.IssuedAt.After(time.Now().Add(-time.Second)) || payload.IssuedAt.Equal(time.Now()))

				// Verify expiration
				expectedExpiry := payload.IssuedAt.Add(tt.request.Duration)
				assert.WithinDuration(t, expectedExpiry, payload.ExpiredAt, time.Second)
			}
		})
	}
}

func TestNewVerifyEmailTokenPayload(t *testing.T) {
	tests := []struct {
		name        string
		request     *entities.VerifyEmailTokenRequest
		expectError bool
	}{
		{
			name: "Create verify email token payload successfully",
			request: &entities.VerifyEmailTokenRequest{
				UserID:   "user123",
				Email:    "user@example.com",
				Duration: time.Hour * 24,
			},
			expectError: false,
		},
		{
			name: "Create verify email token payload with short duration",
			request: &entities.VerifyEmailTokenRequest{
				UserID:   "user456",
				Email:    "user456@example.com",
				Duration: time.Minute * 15,
			},
			expectError: false,
		},
		{
			name: "Create verify email token payload with long duration",
			request: &entities.VerifyEmailTokenRequest{
				UserID:   "user789",
				Email:    "user789@example.com",
				Duration: time.Hour * 168, // 1 week
			},
			expectError: false,
		},
		{
			name: "Create verify email token payload with empty email",
			request: &entities.VerifyEmailTokenRequest{
				UserID:   "user101",
				Email:    "",
				Duration: time.Hour * 24,
			},
			expectError: false,
		},
		{
			name: "Create verify email token payload with special characters in email",
			request: &entities.VerifyEmailTokenRequest{
				UserID:   "user202",
				Email:    "user+test@example.com",
				Duration: time.Hour * 24,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload, err := NewVerifyEmailTokenPayload(tt.request)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, payload)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, payload)

				// Verify payload fields
				assert.NotEqual(t, uuid.Nil, payload.ID)
				assert.Equal(t, tt.request.UserID, payload.UserID)
				assert.Equal(t, tt.request.Email, payload.Email)

				// Verify timestamps
				assert.True(t, payload.IssuedAt.Before(time.Now().Add(time.Second)) || payload.IssuedAt.Equal(time.Now()))
				assert.True(t, payload.IssuedAt.After(time.Now().Add(-time.Second)) || payload.IssuedAt.Equal(time.Now()))

				// Verify expiration
				expectedExpiry := payload.IssuedAt.Add(tt.request.Duration)
				assert.WithinDuration(t, expectedExpiry, payload.ExpiredAt, time.Second)
			}
		})
	}
}

func TestSignInTokenPayloadValid(t *testing.T) {
	tests := []struct {
		name        string
		payload     *SignInTokenPayload
		expectError bool
	}{
		{
			name: "Valid token payload",
			payload: &SignInTokenPayload{
				ID:        uuid.New(),
				UserID:    "user123",
				Role:      constants.UserRoleUser,
				IssuedAt:  time.Now().Add(-time.Hour),
				ExpiredAt: time.Now().Add(time.Hour),
			},
			expectError: false,
		},
		{
			name: "Expired token payload",
			payload: &SignInTokenPayload{
				ID:        uuid.New(),
				UserID:    "user456",
				Role:      constants.UserRoleAdmin,
				IssuedAt:  time.Now().Add(-time.Hour * 2),
				ExpiredAt: time.Now().Add(-time.Hour),
			},
			expectError: true,
		},
		{
			name: "Token payload expiring now",
			payload: &SignInTokenPayload{
				ID:        uuid.New(),
				UserID:    "user789",
				Role:      constants.UserRoleUser,
				IssuedAt:  time.Now().Add(-time.Hour),
				ExpiredAt: time.Now(),
			},
			expectError: true,
		},
		{
			name: "Token payload with zero expiration",
			payload: &SignInTokenPayload{
				ID:        uuid.New(),
				UserID:    "user101",
				Role:      constants.UserRoleUser,
				IssuedAt:  time.Now(),
				ExpiredAt: time.Time{},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.payload.Valid()

			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, constants.ErrExpiredToken, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestVerifyEmailTokenPayloadValid(t *testing.T) {
	tests := []struct {
		name        string
		payload     *VerifyEmailTokenPayload
		expectError bool
	}{
		{
			name: "Valid token payload",
			payload: &VerifyEmailTokenPayload{
				ID:        uuid.New(),
				UserID:    "user123",
				Email:     "user@example.com",
				IssuedAt:  time.Now().Add(-time.Hour),
				ExpiredAt: time.Now().Add(time.Hour),
			},
			expectError: false,
		},
		{
			name: "Expired token payload",
			payload: &VerifyEmailTokenPayload{
				ID:        uuid.New(),
				UserID:    "user456",
				Email:     "user456@example.com",
				IssuedAt:  time.Now().Add(-time.Hour * 2),
				ExpiredAt: time.Now().Add(-time.Hour),
			},
			expectError: true,
		},
		{
			name: "Token payload expiring now",
			payload: &VerifyEmailTokenPayload{
				ID:        uuid.New(),
				UserID:    "user789",
				Email:     "user789@example.com",
				IssuedAt:  time.Now().Add(-time.Hour),
				ExpiredAt: time.Now(),
			},
			expectError: true,
		},
		{
			name: "Token payload with zero expiration",
			payload: &VerifyEmailTokenPayload{
				ID:        uuid.New(),
				UserID:    "user101",
				Email:     "user101@example.com",
				IssuedAt:  time.Now(),
				ExpiredAt: time.Time{},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.payload.Valid()

			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, constants.ErrExpiredToken, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTokenPayloadConsistency(t *testing.T) {
	// Test that the same request always produces valid payloads
	request := &entities.TokenRequest{
		UserID:   "user123",
		Role:     constants.UserRoleUser,
		Duration: time.Hour * 24,
	}

	for i := 0; i < 100; i++ {
		payload, err := NewSignInTokenPayload(request)
		require.NoError(t, err)
		require.NotNil(t, payload)

		// Verify the payload is valid
		err = payload.Valid()
		assert.NoError(t, err)

		// Verify the payload has the expected values
		assert.Equal(t, request.UserID, payload.UserID)
		assert.Equal(t, request.Role, payload.Role)
		assert.True(t, payload.ExpiredAt.After(payload.IssuedAt))
	}
}

func TestTokenPayloadPerformance(t *testing.T) {
	// Test performance with multiple payload creations
	request := &entities.TokenRequest{
		UserID:   "user123",
		Role:     constants.UserRoleUser,
		Duration: time.Hour * 24,
	}

	for i := 0; i < 1000; i++ {
		payload, err := NewSignInTokenPayload(request)
		require.NoError(t, err)
		require.NotNil(t, payload)

		// Verify the payload is valid
		err = payload.Valid()
		assert.NoError(t, err)
	}
}

func TestTokenPayloadEdgeCases(t *testing.T) {
	// Test with very long user IDs
	longUserIDBytes := make([]byte, 1000)
	for i := range longUserIDBytes {
		longUserIDBytes[i] = byte(i % 256)
	}
	longUserID := string(longUserIDBytes)

	request := &entities.TokenRequest{
		UserID:   longUserID,
		Role:     constants.UserRoleUser,
		Duration: time.Hour * 24,
	}

	payload, err := NewSignInTokenPayload(request)
	require.NoError(t, err)
	require.NotNil(t, payload)

	// Verify the payload is valid
	err = payload.Valid()
	assert.NoError(t, err)

	// Verify the payload has the expected values
	assert.Equal(t, longUserID, payload.UserID)
}

func TestTokenPayloadWithDifferentRoles(t *testing.T) {
	roles := []constants.UserRole{
		constants.UserRoleUser,
		constants.UserRoleAdmin,
	}

	for _, role := range roles {
		t.Run(string(role), func(t *testing.T) {
			request := &entities.TokenRequest{
				UserID:   "user123",
				Role:     role,
				Duration: time.Hour * 24,
			}

			payload, err := NewSignInTokenPayload(request)
			require.NoError(t, err)
			require.NotNil(t, payload)

			// Verify the payload has the correct role
			assert.Equal(t, role, payload.Role)

			// Verify the payload is valid
			err = payload.Valid()
			assert.NoError(t, err)
		})
	}
}

func TestTokenPayloadWithDifferentDurations(t *testing.T) {
	durations := []time.Duration{
		time.Second,
		time.Minute,
		time.Hour,
		time.Hour * 24,
		time.Hour * 24 * 7,
		time.Hour * 24 * 30,
	}

	for _, duration := range durations {
		t.Run(duration.String(), func(t *testing.T) {
			request := &entities.TokenRequest{
				UserID:   "user123",
				Role:     constants.UserRoleUser,
				Duration: duration,
			}

			payload, err := NewSignInTokenPayload(request)
			require.NoError(t, err)
			require.NotNil(t, payload)

			// Verify the payload has the correct duration
			expectedExpiry := payload.IssuedAt.Add(duration)
			assert.WithinDuration(t, expectedExpiry, payload.ExpiredAt, time.Second)

			// Verify the payload is valid
			err = payload.Valid()
			assert.NoError(t, err)
		})
	}
}

func TestTokenPayloadUniqueness(t *testing.T) {
	// Test that each payload gets a unique ID
	request := &entities.TokenRequest{
		UserID:   "user123",
		Role:     constants.UserRoleUser,
		Duration: time.Hour * 24,
	}

	ids := make(map[uuid.UUID]bool)
	for i := 0; i < 1000; i++ {
		payload, err := NewSignInTokenPayload(request)
		require.NoError(t, err)
		require.NotNil(t, payload)

		// Each ID should be unique
		assert.False(t, ids[payload.ID], "Duplicate ID generated: %s", payload.ID)
		ids[payload.ID] = true
	}
}
