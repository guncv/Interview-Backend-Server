package repositories

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	db "gitlab.com/interview-simulation/interview-backend-server/internal/db/sqlc"
	log "gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
	mockSqlc "gitlab.com/interview-simulation/interview-backend-server/internal/mocks/db/sqlc"
)

func TestInterviewSessionRepository_UpdateInterviewSessionStatus(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()

	testCases := []struct {
		name   string
		input  *db.UpdateInterviewSessionStatusParams
		setup  func() *mockSqlc.MockStore
		verify func(t *testing.T, gotErr error)
	}{
		{
			name: "Success - Update interview session status",
			input: &db.UpdateInterviewSessionStatusParams{
				ID:      uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
				Column2: "active",
			},
			setup: func() *mockSqlc.MockStore {
				mockInterviewSessionRepo := new(mockSqlc.MockStore)

				mockInterviewSessionRepo.EXPECT().
					UpdateInterviewSessionStatus(ctx, mock.AnythingOfType("*db.UpdateInterviewSessionStatusParams")).
					Return(int64(1), nil)

				return mockInterviewSessionRepo
			},
			verify: func(t *testing.T, gotErr error) {
				assert.NoError(t, gotErr)
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {

			svc := NewInterviewSessionRepository(lgr, tC.setup())

			gotErr := svc.UpdateInterviewSessionStatus(ctx, tC.input)

			tC.verify(t, gotErr)
		})
	}
}
