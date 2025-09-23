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
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
	mockSqlc "gitlab.com/interview-simulation/interview-backend-server/internal/mocks/db/sqlc"
)

func TestReviewCommentRepository_CreateReviewComment(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()
	globalID := "123e4567-e89b-12d3-a456-426614174000"
	mockErr := errors.New("mock error")

	validReq := &db.CreateReviewCommentParams{
		ID:           uuid.MustParse(globalID),
		SessionID:    uuid.MustParse(globalID),
		AuthorType:   "user",
		AuthorUserID: uuid.NullUUID{UUID: uuid.MustParse(globalID), Valid: true},
		Rating:       sql.NullInt16{Int16: 5, Valid: true},
		Description:  sql.NullString{String: "Great job!", Valid: true},
	}

	testCases := []struct {
		name   string
		input  *db.CreateReviewCommentParams
		setup  func() *mockSqlc.MockStore
		verify func(t *testing.T, gotErr error)
	}{
		{
			name:  "Success",
			input: validReq,
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					CreateReviewComment(ctx, *validReq).
					Return(1, nil)

				return mockStore
			},
			verify: func(t *testing.T, gotErr error) {
				assert.NoError(t, gotErr)
			},
		},
		{
			name:  "Error WithCreateReviewCommentNotFound",
			input: validReq,
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					CreateReviewComment(ctx, *validReq).
					Return(0, nil)

				return mockStore
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "The session was not found or deleted. Please try again.")
				assert.Contains(t, gotErr.Error(), "[INS0418]")
			},
		},
		{
			name:  "Error WithCreateReviewCommentError",
			input: validReq,
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					CreateReviewComment(ctx, *validReq).
					Return(0, mockErr)

				return mockStore
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.ErrorIs(t, gotErr, mockErr)
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

			svc := NewReviewCommentRepository(lgr, mockStore)
			gotErr := svc.CreateReviewComment(ctx, tC.input)

			tC.verify(t, gotErr)
		})
	}
}
