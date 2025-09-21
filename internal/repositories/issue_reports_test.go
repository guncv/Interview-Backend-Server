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

func TestIssueReportsRepository_GetUserIssueReportUserIDAndStatusByID(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()
	mockErr := errors.New("error")

	reportID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	validResp := db.GetUserIssueReportUserIDAndStatusByIDRow{
		UserID: uuid.NullUUID{UUID: uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"), Valid: true},
		Status: "test status",
	}

	testCases := []struct {
		name   string
		input  uuid.UUID
		setup  func() *mockSqlc.MockStore
		verify func(t *testing.T, gotResp *db.GetUserIssueReportUserIDAndStatusByIDRow, gotErr error)
	}{
		{
			name:  "Success",
			input: reportID,
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					GetUserIssueReportUserIDAndStatusByID(ctx, reportID).
					Return(validResp, nil)

				return mockStore
			},
			verify: func(t *testing.T, gotResp *db.GetUserIssueReportUserIDAndStatusByIDRow, gotErr error) {
				assert.NoError(t, gotErr)
				assert.Equal(t, &validResp, gotResp)
			},
		},
		{
			name:  "Error - WithGetUserIssueReportUserIDAndStatusByIDNotFound",
			input: reportID,
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					GetUserIssueReportUserIDAndStatusByID(ctx, reportID).
					Return(db.GetUserIssueReportUserIDAndStatusByIDRow{}, sql.ErrNoRows)

				return mockStore
			},
			verify: func(t *testing.T, gotResp *db.GetUserIssueReportUserIDAndStatusByIDRow, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
				assert.Contains(t, gotErr.Error(), "[INS0700]")
				assert.Contains(t, gotErr.Error(), "The issue report was not found")
			},
		},
		{
			name:  "Error - WithGetUserIssueReportUserIDAndStatusByIDError",
			input: reportID,
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					GetUserIssueReportUserIDAndStatusByID(ctx, reportID).
					Return(db.GetUserIssueReportUserIDAndStatusByIDRow{}, mockErr)

				return mockStore
			},
			verify: func(t *testing.T, gotResp *db.GetUserIssueReportUserIDAndStatusByIDRow, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "[INS0101]")
				assert.Contains(t, gotErr.Error(), "We're having trouble connecting to the server")
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

			svc := NewIssueReportsRepository(lgr, mockStore)
			gotResp, gotErr := svc.GetUserIssueReportUserIDAndStatusByID(ctx, tC.input)

			tC.verify(t, gotResp, gotErr)
		})
	}
}
