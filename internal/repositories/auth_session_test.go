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

func TestAuthSessionRepository_GetAuthSessionByID(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()

	successResp := db.AuthSessions{
		ID:               uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
		UserID:           uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
		RefreshTokenHash: "test",
		UserAgent:        "test",
		IpAddress:        "test",
		LoginTime:        sql.NullTime{Time: time.Now(), Valid: true},
		LastActive:       sql.NullTime{Time: time.Now(), Valid: true},
		ExpiresAt:        sql.NullTime{Time: time.Now(), Valid: true},
		IsRevoked:        sql.NullBool{Bool: false, Valid: true},
		CreatedAt:        sql.NullTime{Time: time.Now(), Valid: true},
		UpdatedAt:        sql.NullTime{Time: time.Now(), Valid: true},
	}

	testCases := []struct {
		name   string
		input  uuid.UUID
		setup  func() *mockSqlc.MockStore
		verify func(t *testing.T, got *db.AuthSessions, gotErr error)
	}{
		{
			name:  "Success - Get session by ID",
			input: successResp.ID,
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					GetAuthSessionByID(ctx, successResp.ID).
					Return(successResp, nil)

				return mockStore
			},
			verify: func(t *testing.T, got *db.AuthSessions, gotErr error) {
				assert.NoError(t, gotErr)
				assert.Equal(t, got, &successResp)
			},
		},
		{
			name:  "Success - Get session by ID not found",
			input: successResp.ID,
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					GetAuthSessionByID(ctx, successResp.ID).
					Return(db.AuthSessions{}, sql.ErrNoRows)

				return mockStore
			},
			verify: func(t *testing.T, got *db.AuthSessions, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
				assert.Contains(t, gotErr.Error(), "[INS0218]")
				assert.Contains(t, gotErr.Error(), "Your session has expired or is invalid.")
			},
		},
		{
			name:  "Error - Get session by ID",
			input: successResp.ID,
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					GetAuthSessionByID(ctx, successResp.ID).
					Return(db.AuthSessions{}, errors.New("error"))

				return mockStore
			},
			verify: func(t *testing.T, got *db.AuthSessions, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
				assert.Contains(t, gotErr.Error(), "[INS0101]")
				assert.Contains(t, gotErr.Error(), "We're having trouble connecting to the server")
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

			svc := NewAuthSessionRepository(lgr, mockStore)
			got, gotErr := svc.GetAuthSessionByID(ctx, tC.input)

			tC.verify(t, got, gotErr)
		})
	}
}

func TestAuthSessionRepository_RevokeAuthSessionByID(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()

	successResp := db.AuthSessions{
		ID:               uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
		UserID:           uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
		RefreshTokenHash: "test",
		UserAgent:        "test",
		IpAddress:        "test",
		LoginTime:        sql.NullTime{Time: time.Now(), Valid: true},
		LastActive:       sql.NullTime{Time: time.Now(), Valid: true},
		ExpiresAt:        sql.NullTime{Time: time.Now(), Valid: true},
		IsRevoked:        sql.NullBool{Bool: false, Valid: true},
		CreatedAt:        sql.NullTime{Time: time.Now(), Valid: true},
		UpdatedAt:        sql.NullTime{Time: time.Now(), Valid: true},
	}

	testCases := []struct {
		name   string
		input  uuid.UUID
		setup  func() *mockSqlc.MockStore
		verify func(t *testing.T, gotErr error)
	}{
		{
			name:  "Success - Revoke session by ID",
			input: successResp.ID,
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					RevokeAuthSessionByID(ctx, successResp.ID).
					Return(nil)

				return mockStore
			},
			verify: func(t *testing.T, gotErr error) {
				assert.NoError(t, gotErr)
			},
		},
		{
			name:  "Success - Revoke session by ID not found",
			input: successResp.ID,
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					RevokeAuthSessionByID(ctx, successResp.ID).
					Return(sql.ErrNoRows)

				return mockStore
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "[INS0218]")
				assert.Contains(t, gotErr.Error(), "Your session has expired or is invalid.")
			},
		},
		{
			name:  "Error - Revoke session by ID",
			input: successResp.ID,
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					RevokeAuthSessionByID(ctx, successResp.ID).
					Return(errors.New("error"))

				return mockStore
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "[INS0101]")
				assert.Contains(t, gotErr.Error(), "We're having trouble connecting to the server")
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

			svc := NewAuthSessionRepository(lgr, mockStore)
			gotErr := svc.RevokeAuthSessionByID(ctx, tC.input)

			tC.verify(t, gotErr)
		})
	}
}
