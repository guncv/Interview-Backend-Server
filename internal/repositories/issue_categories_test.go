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

func TestIssueCategoriesRepository_GetIssueCategoryIfExists(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()
	mockErr := errors.New("error")

	reqId := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")

	validResp := db.GetIssueCategoryIfExistsRow{
		ID:   reqId,
		Name: "test name",
	}

	testCases := []struct {
		name   string
		input  uuid.UUID
		setup  func() *mockSqlc.MockStore
		verify func(t *testing.T, gotResp *db.GetIssueCategoryIfExistsRow, gotErr error)
	}{
		{
			name:  "Success",
			input: reqId,
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					GetIssueCategoryIfExists(ctx, reqId).
					Return(validResp, nil)

				return mockStore
			},
			verify: func(t *testing.T, gotResp *db.GetIssueCategoryIfExistsRow, gotErr error) {
				assert.NoError(t, gotErr)
				assert.Equal(t, &validResp, gotResp)
			},
		},
		{
			name:  "Error - WithUpdateUserIssueReportByIDError",
			input: reqId,
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					GetIssueCategoryIfExists(ctx, reqId).
					Return(db.GetIssueCategoryIfExistsRow{}, mockErr)

				return mockStore
			},
			verify: func(t *testing.T, gotResp *db.GetIssueCategoryIfExistsRow, gotErr error) {
				assert.Error(t, gotErr)
				assert.ErrorIs(t, gotErr, mockErr)
				assert.Nil(t, gotResp)
			},
		},
		{
			name:  "Error - WithGetIssueCategoryIfExistsNotFound",
			input: reqId,
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					GetIssueCategoryIfExists(ctx, reqId).
					Return(db.GetIssueCategoryIfExistsRow{}, sql.ErrNoRows)

				return mockStore
			},
			verify: func(t *testing.T, gotResp *db.GetIssueCategoryIfExistsRow, gotErr error) {
				assert.Error(t, gotErr)
				assert.ErrorIs(t, gotErr, sql.ErrNoRows)
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

			svc := NewIssueCategoriesRepository(lgr, mockStore)
			gotResp, gotErr := svc.GetIssueCategoryIfExists(ctx, tC.input)

			tC.verify(t, gotResp, gotErr)
		})
	}
}
