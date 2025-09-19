package repositories

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	db "gitlab.com/interview-simulation/interview-backend-server/internal/db/sqlc"
	log "gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
	mockSqlc "gitlab.com/interview-simulation/interview-backend-server/internal/mocks/db/sqlc"
)

func TestInterviewTurnsRepository_GetInterviewerLastMessage(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()
	mockErr := errors.New("error")

	reqId := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")

	validResp := db.GetInterviewerLastMessageRow{
		CurrentState:   "test current state",
		TranscriptText: "test transcript text",
	}

	testCases := []struct {
		name   string
		input  uuid.UUID
		setup  func() *mockSqlc.MockStore
		verify func(t *testing.T, gotResp *db.GetInterviewerLastMessageRow, gotErr error)
	}{
		{
			name:  "Success",
			input: reqId,
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					GetInterviewerLastMessage(ctx, reqId).
					Return(validResp, nil)

				return mockStore
			},
			verify: func(t *testing.T, gotResp *db.GetInterviewerLastMessageRow, gotErr error) {
				assert.NoError(t, gotErr)
				assert.Equal(t, &validResp, gotResp)
			},
		},
		{
			name:  "Error - WithGetInterviewerLastMessageNotFound",
			input: reqId,
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					GetInterviewerLastMessage(ctx, reqId).
					Return(db.GetInterviewerLastMessageRow{}, sql.ErrNoRows)

				return mockStore
			},
			verify: func(t *testing.T, gotResp *db.GetInterviewerLastMessageRow, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "The interviewer last message was not found. Please try again.")
				assert.Contains(t, gotErr.Error(), "[INS0600]")
				assert.Nil(t, gotResp)
			},
		},
		{
			name:  "Error - WithGetInterviewerLastMessageError",
			input: reqId,
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					GetInterviewerLastMessage(ctx, reqId).
					Return(db.GetInterviewerLastMessageRow{}, mockErr)

				return mockStore
			},
			verify: func(t *testing.T, gotResp *db.GetInterviewerLastMessageRow, gotErr error) {
				assert.Error(t, gotErr)
				assert.ErrorIs(t, gotErr, mockErr)
				assert.Nil(t, gotResp)
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

			svc := NewInterviewTurnsRepository(lgr, mockStore)
			gotResp, gotErr := svc.GetInterviewerLastMessage(ctx, tC.input)

			tC.verify(t, gotResp, gotErr)
		})
	}
}

func TestInterviewTurnsRepository_GetMaxTurnNoBySessionID(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()
	mockErr := errors.New("error")

	reqId := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")

	testCases := []struct {
		name   string
		input  uuid.UUID
		setup  func() *mockSqlc.MockStore
		verify func(t *testing.T, gotResp int64, gotErr error)
	}{
		{
			name:  "Success",
			input: reqId,
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					GetMaxTurnNoBySessionID(ctx, reqId).
					Return(int64(5), nil)

				return mockStore
			},
			verify: func(t *testing.T, gotResp int64, gotErr error) {
				assert.NoError(t, gotErr)
				assert.Equal(t, int64(5), gotResp)
			},
		},
		{
			name:  "Error - WithGetMaxTurnNoBySessionIDNotFound",
			input: reqId,
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					GetMaxTurnNoBySessionID(ctx, reqId).
					Return(int64(0), sql.ErrNoRows)

				return mockStore
			},
			verify: func(t *testing.T, gotResp int64, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "The max turn no by session ID was not found. Please try again.")
				assert.Contains(t, gotErr.Error(), "[INS0601]")
				assert.Equal(t, int64(0), gotResp)
			},
		},
		{
			name:  "Error - WithGetMaxTurnNoBySessionIDError",
			input: reqId,
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					GetMaxTurnNoBySessionID(ctx, reqId).
					Return(int64(0), mockErr)

				return mockStore
			},
			verify: func(t *testing.T, gotResp int64, gotErr error) {
				assert.Error(t, gotErr)
				assert.ErrorIs(t, gotErr, mockErr)
				assert.Equal(t, int64(0), gotResp)
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

			svc := NewInterviewTurnsRepository(lgr, mockStore)
			gotResp, gotErr := svc.GetMaxTurnNoBySessionID(ctx, tC.input)

			tC.verify(t, gotResp, gotErr)
		})
	}
}
