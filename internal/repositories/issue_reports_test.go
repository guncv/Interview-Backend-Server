package repositories

import (
	"context"
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

func TestIssueReportsRepository_UpdateUserIssueReportByID(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()
	// userID := uuid.New()
	mockErr := errors.New("error")

	fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	req := &db.UpdateUserIssueReportByIDParams{
		ID:          uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
		Description: "test description",
		CategoryID:  uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
		UpdatedAt:   fixedTime,
	}

	testCases := []struct {
		name   string
		input  *db.UpdateUserIssueReportByIDParams
		setup  func() *mockSqlc.MockStore
		verify func(t *testing.T, gotErr error)
	}{
		{
			name:  "Success",
			input: req,
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					UpdateUserIssueReportByID(ctx, *req).
					Return(int64(1), nil)

				return mockStore
			},
			verify: func(t *testing.T, gotErr error) {
				assert.NoError(t, gotErr)
			},
		},
		{
			name:  "Error - WithUpdateUserIssueReportByIDError",
			input: req,
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					UpdateUserIssueReportByID(ctx, *req).
					Return(int64(0), mockErr)

				return mockStore
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.ErrorIs(t, gotErr, mockErr)
			},
		},
		{
			name:  "Error - WithUserIssueReportNotFound",
			input: req,
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					UpdateUserIssueReportByID(ctx, *req).
					Return(int64(0), nil)

				return mockStore
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "The issue report was not found")
				assert.Contains(t, gotErr.Error(), "[ONX0700]")
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

			svc := NewIssueReportsRepository(lgr, mockStore)
			gotErr := svc.UpdateUserIssueReportByID(ctx, tC.input)

			tC.verify(t, gotErr)
		})
	}
}
