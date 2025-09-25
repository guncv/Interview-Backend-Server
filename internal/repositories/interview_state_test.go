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
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	log "gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
	mockSqlc "gitlab.com/interview-simulation/interview-backend-server/internal/mocks/db/sqlc"
)

func TestInterviewStateRepository_CreateInterviewStateWithUpdateFlagSessionTx(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()

	globalID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	sessionID := globalID
	phraseType := "Greeting"
	startedAt := time.Now()

	input := &CreateInterviewStateWithUpdateFlagSessionTxReq{
		ID:         globalID,
		SessionID:  sessionID,
		PhraseType: phraseType,
		StartedAt:  startedAt,
	}

	testCases := []struct {
		name   string
		input  *CreateInterviewStateWithUpdateFlagSessionTxReq
		setup  func() *mockSqlc.MockStore
		verify func(t *testing.T, gotErr error)
	}{
		{
			name:  "Success",
			input: input,
			setup: func() *mockSqlc.MockStore {
				mockDBTX := new(mockSqlc.MockDBTX)
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().ExecTx(ctx, mock.MatchedBy(func(fn func(*db.Queries) error) bool {
					return true
				})).Run(func(ctx context.Context, fn func(*db.Queries) error) {
					queries := db.New(mockDBTX)

					// Mock CreateInterviewState
					mockDBTX.EXPECT().
						ExecContext(mock.AnythingOfType("context.backgroundCtx"),
							mock.AnythingOfType("string"),
							globalID,
							sessionID,
							phraseType,
							startedAt,
						).
						Return(nil, nil).Once()

					// Mock UpdateCurrentStateAndIDInterviewSessionByID
					mockDBTX.EXPECT().
						ExecContext(mock.AnythingOfType("context.backgroundCtx"),
							mock.AnythingOfType("string"),
							sessionID,
							sql.NullString{String: phraseType, Valid: true},
							uuid.NullUUID{UUID: globalID, Valid: true},
						).
						Return(mockResult{affected: 1}, nil).Once()

					fn(queries)
				}).Return(nil)

				return mockStore
			},
			verify: func(t *testing.T, gotErr error) {
				assert.NoError(t, gotErr)
			},
		},
		{
			name:  "Error - CreateInterviewStateError",
			input: input,
			setup: func() *mockSqlc.MockStore {
				mockDBTX := new(mockSqlc.MockDBTX)
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().ExecTx(ctx, mock.MatchedBy(func(fn func(*db.Queries) error) bool {
					return true
				})).Run(func(ctx context.Context, fn func(*db.Queries) error) {
					queries := db.New(mockDBTX)

					// Mock CreateInterviewState with error
					mockDBTX.EXPECT().
						ExecContext(mock.AnythingOfType("context.backgroundCtx"),
							mock.AnythingOfType("string"),
							globalID,
							sessionID,
							phraseType,
							startedAt,
						).
						Return(nil, errors.New("database error")).Once()

					fn(queries)
				}).Return(errors.New("database error"))

				return mockStore
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "database error")
			},
		},
		{
			name:  "Error - UpdateCurrentStateAndIDInterviewSessionByIDError",
			input: input,
			setup: func() *mockSqlc.MockStore {
				mockDBTX := new(mockSqlc.MockDBTX)
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().ExecTx(ctx, mock.MatchedBy(func(fn func(*db.Queries) error) bool {
					return true
				})).Run(func(ctx context.Context, fn func(*db.Queries) error) {
					queries := db.New(mockDBTX)

					// Mock CreateInterviewState success
					mockDBTX.EXPECT().
						ExecContext(mock.AnythingOfType("context.backgroundCtx"),
							mock.AnythingOfType("string"),
							globalID,
							sessionID,
							phraseType,
							startedAt,
						).
						Return(nil, nil).Once()

					// Mock UpdateCurrentStateAndIDInterviewSessionByID with error
					mockDBTX.EXPECT().
						ExecContext(mock.AnythingOfType("context.backgroundCtx"),
							mock.AnythingOfType("string"),
							sessionID,
							sql.NullString{String: phraseType, Valid: true},
							uuid.NullUUID{UUID: globalID, Valid: true},
						).
						Return(nil, errors.New("database error")).Once()

					fn(queries)
				}).Return(errors.New("database error"))

				return mockStore
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "database error")
			},
		},
		{
			name:  "Error - UpdateCurrentStateAndIDInterviewSessionByIDRowsAffectedZero",
			input: input,
			setup: func() *mockSqlc.MockStore {
				mockDBTX := new(mockSqlc.MockDBTX)
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().ExecTx(ctx, mock.MatchedBy(func(fn func(*db.Queries) error) bool {
					return true
				})).Run(func(ctx context.Context, fn func(*db.Queries) error) {
					queries := db.New(mockDBTX)

					// Mock CreateInterviewState success
					mockDBTX.EXPECT().
						ExecContext(mock.AnythingOfType("context.backgroundCtx"),
							mock.AnythingOfType("string"),
							globalID,
							sessionID,
							phraseType,
							startedAt,
						).
						Return(nil, nil).Once()

					// Mock UpdateCurrentStateAndIDInterviewSessionByID with zero rows affected
					mockDBTX.EXPECT().
						ExecContext(mock.AnythingOfType("context.backgroundCtx"),
							mock.AnythingOfType("string"),
							sessionID,
							sql.NullString{String: phraseType, Valid: true},
							uuid.NullUUID{UUID: globalID, Valid: true},
						).
						Return(mockResult{affected: 0}, nil).Once()

					fn(queries)
				}).Return(app_error.New(errors.New("rows affected is 0"), app_error.ErrCodeSessionNotFound))

				return mockStore
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "[INS0401]")
				assert.Contains(t, gotErr.Error(), "The session was not found. Please try again.")
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

			repo := NewInterviewStateRepository(lgr, mockStore)

			gotErr := repo.CreateInterviewStateWithUpdateFlagSessionTx(ctx, tC.input)

			tC.verify(t, gotErr)
		})
	}
}

func TestInterviewStateRepository_EndOldInterviewStateAndCreateNewInterviewStateWithUpdateFlagSessionTx(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()

	globalID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	newID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440001")
	sessionID := globalID
	phraseType := "Technical"
	endedAt := time.Now()
	startedAt := time.Now()

	input := &EndOldInterviewStateAndCreateNewInterviewStateWithUpdateFlagSessionTxReq{
		ID:         globalID,
		EndedAt:    endedAt,
		NewID:      newID,
		SessionID:  sessionID,
		PhraseType: phraseType,
		StartedAt:  startedAt,
	}

	testCases := []struct {
		name   string
		input  *EndOldInterviewStateAndCreateNewInterviewStateWithUpdateFlagSessionTxReq
		setup  func() *mockSqlc.MockStore
		verify func(t *testing.T, gotErr error)
	}{
		{
			name:  "Success",
			input: input,
			setup: func() *mockSqlc.MockStore {
				mockDBTX := new(mockSqlc.MockDBTX)
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().ExecTx(ctx, mock.MatchedBy(func(fn func(*db.Queries) error) bool {
					return true
				})).Run(func(ctx context.Context, fn func(*db.Queries) error) {
					queries := db.New(mockDBTX)

					// Mock UpdateEndedAtInterviewStateByID
					mockDBTX.EXPECT().
						ExecContext(mock.AnythingOfType("context.backgroundCtx"),
							mock.AnythingOfType("string"),
							globalID,
							sql.NullTime{Time: endedAt, Valid: true},
						).
						Return(mockResult{affected: 1}, nil).Once()

					// Mock CreateInterviewState
					mockDBTX.EXPECT().
						ExecContext(mock.AnythingOfType("context.backgroundCtx"),
							mock.AnythingOfType("string"),
							newID,
							sessionID,
							phraseType,
							startedAt,
						).
						Return(nil, nil).Once()

					// Mock UpdateCurrentStateAndIDInterviewSessionByID
					mockDBTX.EXPECT().
						ExecContext(mock.AnythingOfType("context.backgroundCtx"),
							mock.AnythingOfType("string"),
							sessionID,
							sql.NullString{String: phraseType, Valid: true},
							uuid.NullUUID{UUID: newID, Valid: true},
						).
						Return(mockResult{affected: 1}, nil).Once()

					fn(queries)
				}).Return(nil)

				return mockStore
			},
			verify: func(t *testing.T, gotErr error) {
				assert.NoError(t, gotErr)
			},
		},
		{
			name:  "Error - UpdateEndedAtInterviewStateByIDError",
			input: input,
			setup: func() *mockSqlc.MockStore {
				mockDBTX := new(mockSqlc.MockDBTX)
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().ExecTx(ctx, mock.MatchedBy(func(fn func(*db.Queries) error) bool {
					return true
				})).Run(func(ctx context.Context, fn func(*db.Queries) error) {
					queries := db.New(mockDBTX)

					// Mock UpdateEndedAtInterviewStateByID with error
					mockDBTX.EXPECT().
						ExecContext(mock.AnythingOfType("context.backgroundCtx"),
							mock.AnythingOfType("string"),
							globalID,
							sql.NullTime{Time: endedAt, Valid: true},
						).
						Return(nil, errors.New("database error")).Once()

					fn(queries)
				}).Return(errors.New("database error"))

				return mockStore
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "database error")
			},
		},
		{
			name:  "Error - UpdateEndedAtInterviewStateByIDRowsAffectedZero",
			input: input,
			setup: func() *mockSqlc.MockStore {
				mockDBTX := new(mockSqlc.MockDBTX)
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().ExecTx(ctx, mock.MatchedBy(func(fn func(*db.Queries) error) bool {
					return true
				})).Run(func(ctx context.Context, fn func(*db.Queries) error) {
					queries := db.New(mockDBTX)

					// Mock UpdateEndedAtInterviewStateByID with zero rows affected
					mockDBTX.EXPECT().
						ExecContext(mock.AnythingOfType("context.backgroundCtx"),
							mock.AnythingOfType("string"),
							globalID,
							sql.NullTime{Time: endedAt, Valid: true},
						).
						Return(mockResult{affected: 0}, nil).Once()

					fn(queries)
				}).Return(app_error.New(errors.New("rows affected is 0"), app_error.ErrCodeSessionNotFound))

				return mockStore
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "[INS0401]")
				assert.Contains(t, gotErr.Error(), "The session was not found. Please try again.")
			},
		},
		{
			name:  "Error - CreateInterviewStateError",
			input: input,
			setup: func() *mockSqlc.MockStore {
				mockDBTX := new(mockSqlc.MockDBTX)
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().ExecTx(ctx, mock.MatchedBy(func(fn func(*db.Queries) error) bool {
					return true
				})).Run(func(ctx context.Context, fn func(*db.Queries) error) {
					queries := db.New(mockDBTX)

					// Mock UpdateEndedAtInterviewStateByID success
					mockDBTX.EXPECT().
						ExecContext(mock.AnythingOfType("context.backgroundCtx"),
							mock.AnythingOfType("string"),
							globalID,
							sql.NullTime{Time: endedAt, Valid: true},
						).
						Return(mockResult{affected: 1}, nil).Once()

					// Mock CreateInterviewState with error
					mockDBTX.EXPECT().
						ExecContext(mock.AnythingOfType("context.backgroundCtx"),
							mock.AnythingOfType("string"),
							newID,
							sessionID,
							phraseType,
							startedAt,
						).
						Return(nil, errors.New("database error")).Once()

					fn(queries)
				}).Return(errors.New("database error"))

				return mockStore
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "database error")
			},
		},
		{
			name:  "Error - UpdateCurrentStateAndIDInterviewSessionByIDError",
			input: input,
			setup: func() *mockSqlc.MockStore {
				mockDBTX := new(mockSqlc.MockDBTX)
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().ExecTx(ctx, mock.MatchedBy(func(fn func(*db.Queries) error) bool {
					return true
				})).Run(func(ctx context.Context, fn func(*db.Queries) error) {
					queries := db.New(mockDBTX)

					// Mock UpdateEndedAtInterviewStateByID success
					mockDBTX.EXPECT().
						ExecContext(mock.AnythingOfType("context.backgroundCtx"),
							mock.AnythingOfType("string"),
							globalID,
							sql.NullTime{Time: endedAt, Valid: true},
						).
						Return(mockResult{affected: 1}, nil).Once()

					// Mock CreateInterviewState success
					mockDBTX.EXPECT().
						ExecContext(mock.AnythingOfType("context.backgroundCtx"),
							mock.AnythingOfType("string"),
							newID,
							sessionID,
							phraseType,
							startedAt,
						).
						Return(nil, nil).Once()

					// Mock UpdateCurrentStateAndIDInterviewSessionByID with error
					mockDBTX.EXPECT().
						ExecContext(mock.AnythingOfType("context.backgroundCtx"),
							mock.AnythingOfType("string"),
							sessionID,
							sql.NullString{String: phraseType, Valid: true},
							uuid.NullUUID{UUID: newID, Valid: true},
						).
						Return(nil, errors.New("database error")).Once()

					fn(queries)
				}).Return(errors.New("database error"))

				return mockStore
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "database error")
			},
		},
		{
			name:  "Error - UpdateCurrentStateAndIDInterviewSessionByIDRowsAffectedZero",
			input: input,
			setup: func() *mockSqlc.MockStore {
				mockDBTX := new(mockSqlc.MockDBTX)
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().ExecTx(ctx, mock.MatchedBy(func(fn func(*db.Queries) error) bool {
					return true
				})).Run(func(ctx context.Context, fn func(*db.Queries) error) {
					queries := db.New(mockDBTX)

					// Mock UpdateEndedAtInterviewStateByID success
					mockDBTX.EXPECT().
						ExecContext(mock.AnythingOfType("context.backgroundCtx"),
							mock.AnythingOfType("string"),
							globalID,
							sql.NullTime{Time: endedAt, Valid: true},
						).
						Return(mockResult{affected: 1}, nil).Once()

					// Mock CreateInterviewState success
					mockDBTX.EXPECT().
						ExecContext(mock.AnythingOfType("context.backgroundCtx"),
							mock.AnythingOfType("string"),
							newID,
							sessionID,
							phraseType,
							startedAt,
						).
						Return(nil, nil).Once()

					// Mock UpdateCurrentStateAndIDInterviewSessionByID with zero rows affected
					mockDBTX.EXPECT().
						ExecContext(mock.AnythingOfType("context.backgroundCtx"),
							mock.AnythingOfType("string"),
							sessionID,
							sql.NullString{String: phraseType, Valid: true},
							uuid.NullUUID{UUID: newID, Valid: true},
						).
						Return(mockResult{affected: 0}, nil).Once()

					fn(queries)
				}).Return(app_error.New(errors.New("rows affected is 0"), app_error.ErrCodeSessionNotFound))

				return mockStore
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "[INS0401]")
				assert.Contains(t, gotErr.Error(), "The session was not found. Please try again.")
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

			repo := NewInterviewStateRepository(lgr, mockStore)

			gotErr := repo.EndOldInterviewStateAndCreateNewInterviewStateWithUpdateFlagSessionTx(ctx, tC.input)

			tC.verify(t, gotErr)
		})
	}
}
