package repositories

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	db "gitlab.com/interview-simulation/interview-backend-server/internal/db/sqlc"
	log "gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
	mockSqlc "gitlab.com/interview-simulation/interview-backend-server/internal/mocks/db/sqlc"
)

func TestResumeRepository_GetResumeByID(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()

	successResp := db.Resumes{
		ID:         uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
		UserID:     uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
		FileName:   "test.pdf",
		StorageKey: "test.pdf",
		MimeType:   "application/pdf",
		ByteSize:   1024,
		IsDefault:  true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		DeletedAt:  sql.NullTime{Time: time.Now(), Valid: true},
	}

	testCases := []struct {
		name   string
		input  uuid.UUID
		setup  func() *mockSqlc.MockStore
		verify func(t *testing.T, got *db.Resumes, gotErr error)
	}{
		{
			name:  "Success - Get resume by ID",
			input: successResp.ID,
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					GetResumeByID(ctx, successResp.ID).
					Return(successResp, nil)

				return mockStore
			},
			verify: func(t *testing.T, got *db.Resumes, gotErr error) {
				assert.NoError(t, gotErr)
				assert.Equal(t, got, &successResp)
			},
		},
		{
			name:  "Error - Get resume by ID not found",
			input: successResp.ID,
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					GetResumeByID(ctx, successResp.ID).
					Return(db.Resumes{}, sql.ErrNoRows)

				return mockStore
			},
			verify: func(t *testing.T, got *db.Resumes, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
				assert.Contains(t, gotErr.Error(), "[ONX0304]")
				assert.Contains(t, gotErr.Error(), "The resume was not found")
			},
		},
		{
			name:  "Error - Get resume by ID",
			input: successResp.ID,
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					GetResumeByID(ctx, successResp.ID).
					Return(db.Resumes{}, errors.New("error"))

				return mockStore
			},
			verify: func(t *testing.T, got *db.Resumes, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "[ONX0101]")
				assert.Contains(t, gotErr.Error(), "We're having trouble connecting to the server")
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

			svc := NewResumeRepository(lgr, mockStore, nil)
			got, gotErr := svc.GetResumeByID(ctx, tC.input)

			tC.verify(t, got, gotErr)
		})
	}
}

func TestResumeRepository_SwitchDefaultResume(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()

	oldID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	newID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	testCases := []struct {
		name  string
		input struct {
			oldID uuid.UUID
			newID uuid.UUID
		}
		setup  func() *mockSqlc.MockStore
		verify func(t *testing.T, gotErr error)
	}{
		{
			name: "Success",
			input: struct {
				oldID uuid.UUID
				newID uuid.UUID
			}{
				oldID: oldID,
				newID: newID,
			},
			setup: func() *mockSqlc.MockStore {
				mockDBTX := new(mockSqlc.MockDBTX)
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().ExecTx(ctx, mock.MatchedBy(func(fn func(*db.Queries) error) bool {
					return true
				})).Run(func(ctx context.Context, fn func(*db.Queries) error) {
					queries := db.New(mockDBTX)

					// Mock UnsetDefaultResume
					mockDBTX.EXPECT().
						ExecContext(mock.AnythingOfType(
							"context.backgroundCtx"),
							mock.AnythingOfType("string"),
							oldID,
						).
						Return(nil, nil).Once()

					// Mock SetDefaultResume
					mockDBTX.EXPECT().
						ExecContext(mock.AnythingOfType(
							"context.backgroundCtx"),
							mock.AnythingOfType("string"),
							newID,
						).
						Return(nil, nil).Once()

					fn(queries)
				}).Return(nil)

				return mockStore
			},
			verify: func(t *testing.T, gotErr error) {
				assert.NoError(t, gotErr)
			},
		},
		{
			name: "Error - UnsetDefaultResume",
			input: struct {
				oldID uuid.UUID
				newID uuid.UUID
			}{
				oldID: oldID,
				newID: newID,
			},
			setup: func() *mockSqlc.MockStore {
				mockDBTX := new(mockSqlc.MockDBTX)
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().ExecTx(ctx, mock.MatchedBy(func(fn func(*db.Queries) error) bool {
					return true
				})).Run(func(ctx context.Context, fn func(*db.Queries) error) {
					queries := db.New(mockDBTX)

					// Mock UnsetDefaultResume
					mockDBTX.EXPECT().
						ExecContext(mock.AnythingOfType(
							"context.backgroundCtx"),
							mock.AnythingOfType("string"),
							oldID,
						).
						Return(nil, errors.New("error")).Once()

					fn(queries)
				}).Return(errors.New("error"))

				return mockStore
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "[ONX0101]")
				assert.Contains(t, gotErr.Error(), "We're having trouble connecting to the server.")
			},
		},
		{
			name: "Error - SetDefaultResume",
			input: struct {
				oldID uuid.UUID
				newID uuid.UUID
			}{
				oldID: oldID,
				newID: newID,
			},
			setup: func() *mockSqlc.MockStore {
				mockDBTX := new(mockSqlc.MockDBTX)
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().ExecTx(ctx, mock.MatchedBy(func(fn func(*db.Queries) error) bool {
					return true
				})).Run(func(ctx context.Context, fn func(*db.Queries) error) {
					queries := db.New(mockDBTX)

					// Mock UnsetDefaultResume
					mockDBTX.EXPECT().
						ExecContext(mock.AnythingOfType(
							"context.backgroundCtx"),
							mock.AnythingOfType("string"),
							oldID,
						).
						Return(nil, nil).Once()

					// Mock SetDefaultResume
					mockDBTX.EXPECT().
						ExecContext(mock.AnythingOfType(
							"context.backgroundCtx"),
							mock.AnythingOfType("string"),
							newID,
						).
						Return(nil, errors.New("error")).Once()

					fn(queries)
				}).Return(errors.New("error"))

				return mockStore
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "[ONX0101]")
				assert.Contains(t, gotErr.Error(), "We're having trouble connecting to the server.")
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

			svc := NewResumeRepository(lgr, mockStore, nil)

			gotErr := svc.SwitchDefaultResume(ctx, tC.input.oldID, tC.input.newID)

			tC.verify(t, gotErr)
		})
	}
}
