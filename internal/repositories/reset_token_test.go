package repositories

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	db "gitlab.com/interview-simulation/interview-backend-server/internal/db/sqlc"
	log "gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
	mockSqlc "gitlab.com/interview-simulation/interview-backend-server/internal/mocks/db/sqlc"
)

func TestResetTokenRepository_GetResetToken(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()
	token := "1234567890"

	successResp := db.ResetTokens{
		ID:          uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
		UserID:      uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
		TokenHash:   token,
		Used:        true,
		RequestedAt: sql.NullTime{Time: time.Now(), Valid: true},
		UsedAt:      sql.NullTime{Time: time.Now(), Valid: true},
		ExpiresAt:   time.Now().Add(time.Hour * 24),
		IpAddress:   sql.NullString{String: "127.0.0.1", Valid: true},
		UserAgent:   sql.NullString{String: "Chrome/91.0.4472.124 Safari/537.36", Valid: true},
	}

	testCases := []struct {
		name   string
		input  string
		setup  func() *mockSqlc.MockStore
		verify func(t *testing.T, got *db.ResetTokens, gotErr error)
	}{
		{
			name:  "Success - Get reset token",
			input: token,
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					GetResetToken(ctx, token).
					Return(successResp, nil)

				return mockStore
			},
			verify: func(t *testing.T, got *db.ResetTokens, gotErr error) {
				assert.NoError(t, gotErr)
				assert.Equal(t, got, &successResp)
			},
		},
		{
			name:  "Error - Get reset token not found",
			input: token,
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					GetResetToken(ctx, token).
					Return(db.ResetTokens{}, sql.ErrNoRows)

				return mockStore
			},
			verify: func(t *testing.T, got *db.ResetTokens, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
				assert.Contains(t, gotErr.Error(), "[ONX0208]")
				assert.Contains(t, gotErr.Error(), "That reset link is invalid. Please request a new one.")
			},
		},
		{
			name:  "Error - Get reset token",
			input: token,
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					GetResetToken(ctx, token).
					Return(db.ResetTokens{}, errors.New("error"))

				return mockStore
			},
			verify: func(t *testing.T, got *db.ResetTokens, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			mockStore := tC.setup()

			defer func() {
				if mockStore != nil {
					mockStore.AssertExpectations(t)
				}
			}()

			svc := NewResetTokenRepository(lgr, mockStore)
			got, gotErr := svc.GetResetToken(ctx, tC.input)

			tC.verify(t, got, gotErr)
		})
	}
}
