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

func TestNewTokenPayload(t *testing.T) {
	tests := []struct {
		name    string
		request *entities.TokenRequest
		wantErr bool
	}{
		{
			name: "Valid token request",
			request: &entities.TokenRequest{
				UserID:         "user123",
				OrganizationID: "org456",
				Role:           constants.UserRoleAdmin,
				Duration:       24 * time.Hour,
			},
			wantErr: false,
		},
		{
			name: "Valid token request with different role",
			request: &entities.TokenRequest{
				UserID:         "user789",
				OrganizationID: "org101",
				Role:           constants.UserRoleMentor,
				Duration:       12 * time.Hour,
			},
			wantErr: false,
		},
		{
			name: "Valid token request with zero duration",
			request: &entities.TokenRequest{
				UserID:         "user456",
				OrganizationID: "org789",
				Role:           constants.UserRoleTrainee,
				Duration:       0,
			},
			wantErr: false,
		},
		{
			name: "Valid token request with negative duration",
			request: &entities.TokenRequest{
				UserID:         "user999",
				OrganizationID: "org999",
				Role:           constants.UserRoleAdmin,
				Duration:       -1 * time.Hour,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload, err := NewTokenPayload(tt.request)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, payload)
			assert.NotEqual(t, uuid.Nil, payload.ID)
			assert.Equal(t, tt.request.UserID, payload.UserID)
			assert.Equal(t, tt.request.OrganizationID, payload.OrganizationID)
			assert.Equal(t, tt.request.Role, payload.Role)

			// Check that IssuedAt is recent (within 1 second)
			assert.WithinDuration(t, time.Now(), payload.IssuedAt, time.Second)

			// Check that ExpiredAt is IssuedAt + Duration
			expectedExpiry := payload.IssuedAt.Add(tt.request.Duration)
			assert.WithinDuration(t, expectedExpiry, payload.ExpiredAt, time.Second)
		})
	}
}

func TestTokenPayload_Valid(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name    string
		payload *TokenPayload
		wantErr bool
	}{
		{
			name: "Valid token - not expired",
			payload: &TokenPayload{
				ID:             uuid.New(),
				UserID:         "user123",
				OrganizationID: "org456",
				Role:           constants.UserRoleAdmin,
				IssuedAt:       now.Add(-1 * time.Hour),
				ExpiredAt:      now.Add(1 * time.Hour),
			},
			wantErr: false,
		},
		{
			name: "Expired token",
			payload: &TokenPayload{
				ID:             uuid.New(),
				UserID:         "user123",
				OrganizationID: "org456",
				Role:           constants.UserRoleAdmin,
				IssuedAt:       now.Add(-2 * time.Hour),
				ExpiredAt:      now.Add(-1 * time.Hour),
			},
			wantErr: true,
		},
		{
			name: "Token expiring now",
			payload: &TokenPayload{
				ID:             uuid.New(),
				UserID:         "user123",
				OrganizationID: "org456",
				Role:           constants.UserRoleAdmin,
				IssuedAt:       now.Add(-1 * time.Hour),
				ExpiredAt:      now,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.payload.Valid()

			if tt.wantErr {
				assert.Error(t, err)
				assert.Equal(t, constants.ErrExpiredToken, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
