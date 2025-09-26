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
	"gitlab.com/interview-simulation/interview-backend-server/internal/config"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	db "gitlab.com/interview-simulation/interview-backend-server/internal/db/sqlc"
	log "gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
	mockSqlc "gitlab.com/interview-simulation/interview-backend-server/internal/mocks/db/sqlc"
)

func TestInterviewSessionRepository_UpdateInterviewSessionStatus(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()
	updateID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	updateStatus := "active"

	testCases := []struct {
		name   string
		input  *db.UpdateInterviewSessionStatusParams
		setup  func() *mockSqlc.MockStore
		verify func(t *testing.T, gotErr error)
	}{
		{
			name: "Success - Update interview session status",
			input: &db.UpdateInterviewSessionStatusParams{
				ID:      updateID,
				Column2: updateStatus,
			},
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)
				mockStore.EXPECT().
					UpdateInterviewSessionStatus(ctx, db.UpdateInterviewSessionStatusParams{
						ID:      updateID,
						Column2: updateStatus,
					}).
					Return(1, nil)

				return mockStore
			},
			verify: func(t *testing.T, gotErr error) {
				assert.NoError(t, gotErr)
			},
		},
		{
			name: "Error - Update interview session status",
			input: &db.UpdateInterviewSessionStatusParams{
				ID:      updateID,
				Column2: updateStatus,
			},
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					UpdateInterviewSessionStatus(ctx, db.UpdateInterviewSessionStatusParams{
						ID:      updateID,
						Column2: updateStatus,
					}).
					Return(0, errors.New("database error"))

				return mockStore
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
			},
		},
		{
			name: "Error - Update interview session status not found",
			input: &db.UpdateInterviewSessionStatusParams{
				ID:      updateID,
				Column2: updateStatus,
			},
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					UpdateInterviewSessionStatus(ctx, db.UpdateInterviewSessionStatusParams{
						ID:      updateID,
						Column2: updateStatus,
					}).
					Return(0, nil)

				return mockStore
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Equal(
					t, gotErr.Error(),
					"[INS0401] The session was not found. Please try again. | interview session not found",
				)
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

			cfg := &config.Config{}

			svc := NewInterviewSessionRepository(lgr, mockStore, cfg)
			gotErr := svc.UpdateInterviewSessionStatus(ctx, tC.input)

			tC.verify(t, gotErr)
		})
	}
}

func TestInterviewSessionRepository_EndInterviewSession(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()
	updateID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	updateStatus := "active"
	endAt := time.Date(2025, 9, 4, 18, 35, 49, 777972000, time.FixedZone("UTC+7", 7*3600))

	testCases := []struct {
		name   string
		input  *db.EndInterviewSessionParams
		setup  func() *mockSqlc.MockStore
		verify func(t *testing.T, gotErr error)
	}{
		{
			name: "Success - Update interview session status",
			input: &db.EndInterviewSessionParams{
				ID:           updateID,
				Status:       updateStatus,
				EndedAt:      sql.NullTime{Time: endAt, Valid: true},
				OverallScore: sql.NullFloat64{Float64: 100, Valid: true},
				SummaryMd:    sql.NullString{String: "summary", Valid: true},
			},
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					EndInterviewSession(ctx, db.EndInterviewSessionParams{
						ID:           updateID,
						Status:       updateStatus,
						EndedAt:      sql.NullTime{Time: endAt, Valid: true},
						OverallScore: sql.NullFloat64{Float64: 100, Valid: true},
						SummaryMd:    sql.NullString{String: "summary", Valid: true},
					}).
					Return(1, nil)

				return mockStore
			},
			verify: func(t *testing.T, gotErr error) {
				assert.NoError(t, gotErr)
			},
		},
		{
			name: "Error - Update interview session status",
			input: &db.EndInterviewSessionParams{
				ID:           updateID,
				Status:       updateStatus,
				EndedAt:      sql.NullTime{Time: endAt, Valid: true},
				OverallScore: sql.NullFloat64{Float64: 100, Valid: true},
				SummaryMd:    sql.NullString{String: "summary", Valid: true},
			},
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					EndInterviewSession(ctx, db.EndInterviewSessionParams{
						ID:           updateID,
						Status:       updateStatus,
						EndedAt:      sql.NullTime{Time: endAt, Valid: true},
						OverallScore: sql.NullFloat64{Float64: 100, Valid: true},
						SummaryMd:    sql.NullString{String: "summary", Valid: true},
					}).
					Return(0, errors.New("database error"))

				return mockStore
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
			},
		},
		{
			name: "Error - Update interview session status not found",
			input: &db.EndInterviewSessionParams{
				ID:           updateID,
				Status:       updateStatus,
				EndedAt:      sql.NullTime{Time: endAt, Valid: true},
				OverallScore: sql.NullFloat64{Float64: 100, Valid: true},
				SummaryMd:    sql.NullString{String: "summary", Valid: true},
			},
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					EndInterviewSession(ctx, db.EndInterviewSessionParams{
						ID:           updateID,
						Status:       updateStatus,
						EndedAt:      sql.NullTime{Time: endAt, Valid: true},
						OverallScore: sql.NullFloat64{Float64: 100, Valid: true},
						SummaryMd:    sql.NullString{String: "summary", Valid: true},
					}).
					Return(0, nil)

				return mockStore
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Equal(
					t, gotErr.Error(),
					"[INS0401] The session was not found. Please try again. | interview session not found",
				)
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

			cfg := &config.Config{}

			svc := NewInterviewSessionRepository(lgr, mockStore, cfg)

			gotErr := svc.EndInterviewSession(ctx, tC.input)

			tC.verify(t, gotErr)
		})
	}
}

func TestInterviewSessionRepository_CreateInterviewSessionWithNewResumeTx(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()

	resumeID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	userID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	fileName := "test.pdf"
	storageKey := "test.pdf"
	mimeType := "application/pdf"
	byteSize := int32(100)
	isDefault := false

	sessionID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	position := "test"
	status := "active"
	modality := "test"
	isConsent := true

	input := &CreateInterviewSessionTxReq{
		ResumeID:   resumeID,
		UserID:     userID,
		FileName:   fileName,
		StorageKey: storageKey,
		MimeType:   mimeType,
		ByteSize:   byteSize,
		IsDefault:  isDefault,

		SessionID: sessionID,
		Position:  position,
		Status:    status,
		Modality:  modality,
		IsConsent: isConsent,
	}

	testCases := []struct {
		name   string
		input  *CreateInterviewSessionTxReq
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

					// Mock CreateResume
					mockDBTX.EXPECT().
						ExecContext(mock.AnythingOfType(
							"context.backgroundCtx"),
							mock.AnythingOfType("string"),
							resumeID,
							userID,
							fileName,
							storageKey,
							mimeType,
							byteSize,
							isDefault).
						Return(nil, nil).Once()

					// Mock CreateInterviewSession
					mockDBTX.EXPECT().
						ExecContext(
							mock.AnythingOfType("context.backgroundCtx"),
							mock.AnythingOfType("string"),
							mock.Anything,
							mock.Anything,
							mock.Anything,
							mock.Anything,
							mock.Anything,
							mock.Anything,
							mock.Anything,
							mock.Anything,
							mock.Anything,
							mock.Anything).
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
			name:  "Error - Create interview session with new resume",
			input: input,
			setup: func() *mockSqlc.MockStore {
				mockDBTX := new(mockSqlc.MockDBTX)
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().ExecTx(ctx, mock.MatchedBy(func(fn func(*db.Queries) error) bool {
					return true
				})).Run(func(ctx context.Context, fn func(*db.Queries) error) {
					queries := db.New(mockDBTX)

					// Mock CreateResume
					mockDBTX.EXPECT().
						ExecContext(mock.AnythingOfType(
							"context.backgroundCtx"),
							mock.AnythingOfType("string"),
							resumeID,
							userID,
							fileName,
							storageKey,
							mimeType,
							byteSize,
							isDefault).
						Return(nil, errors.New("database error")).Once()

					fn(queries)
				}).Return(errors.New("database error"))

				return mockStore
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
			},
		},
		{
			name:  "Error - Create interview session with new interview session",
			input: input,
			setup: func() *mockSqlc.MockStore {
				mockDBTX := new(mockSqlc.MockDBTX)
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().ExecTx(ctx, mock.MatchedBy(func(fn func(*db.Queries) error) bool {
					return true
				})).Run(func(ctx context.Context, fn func(*db.Queries) error) {
					queries := db.New(mockDBTX)

					// Mock CreateResume
					mockDBTX.EXPECT().
						ExecContext(mock.AnythingOfType(
							"context.backgroundCtx"),
							mock.AnythingOfType("string"),
							resumeID,
							userID,
							fileName,
							storageKey,
							mimeType,
							byteSize,
							isDefault).
						Return(nil, nil).Once()

					// Mock CreateInterviewSession
					mockDBTX.EXPECT().
						ExecContext(
							mock.AnythingOfType("context.backgroundCtx"),
							mock.AnythingOfType("string"),
							mock.Anything,
							mock.Anything,
							mock.Anything,
							mock.Anything,
							mock.Anything,
							mock.Anything,
							mock.Anything,
							mock.Anything,
							mock.Anything).
						Return(nil, errors.New("database error")).Once()

					fn(queries)
				}).Return(errors.New("database error"))

				return mockStore
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
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

			cfg := &config.Config{}

			svc := NewInterviewSessionRepository(lgr, mockStore, cfg)

			gotErr := svc.CreateInterviewSessionWithNewResumeTx(ctx, tC.input)

			tC.verify(t, gotErr)
		})
	}
}

func TestInterviewSessionRepository_UpdateStartedAtInterviewSession(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()
	updateID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	startedAt := time.Date(2025, 9, 4, 18, 35, 49, 777972000, time.FixedZone("UTC+7", 7*3600))

	testCases := []struct {
		name   string
		input  *db.UpdateStartedAtInterviewSessionParams
		setup  func() *mockSqlc.MockStore
		verify func(t *testing.T, gotErr error)
	}{
		{
			name: "Success",
			input: &db.UpdateStartedAtInterviewSessionParams{
				ID:        updateID,
				StartedAt: sql.NullTime{Time: startedAt, Valid: true},
			},
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					UpdateStartedAtInterviewSession(ctx, db.UpdateStartedAtInterviewSessionParams{
						ID:        updateID,
						StartedAt: sql.NullTime{Time: startedAt, Valid: true},
					}).
					Return(1, nil)

				return mockStore
			},
			verify: func(t *testing.T, gotErr error) {
				assert.NoError(t, gotErr)
			},
		},
		{
			name: "Error - WithUpdateStartedAtInterviewSessionNotFound",
			input: &db.UpdateStartedAtInterviewSessionParams{
				ID:        updateID,
				StartedAt: sql.NullTime{Time: startedAt, Valid: true},
			},
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					UpdateStartedAtInterviewSession(ctx, db.UpdateStartedAtInterviewSessionParams{
						ID:        updateID,
						StartedAt: sql.NullTime{Time: startedAt, Valid: true},
					}).
					Return(0, nil)

				return mockStore
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "[INS0401]")
				assert.Contains(t, gotErr.Error(), "The session was not found. Please try again.")
			},
		},
		{
			name: "Error - WithUpdateStartedAtInterviewSessionError",
			input: &db.UpdateStartedAtInterviewSessionParams{
				ID:        updateID,
				StartedAt: sql.NullTime{Time: startedAt, Valid: true},
			},
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					UpdateStartedAtInterviewSession(ctx, db.UpdateStartedAtInterviewSessionParams{
						ID:        updateID,
						StartedAt: sql.NullTime{Time: startedAt, Valid: true},
					}).
					Return(0, errors.New("error"))

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

			cfg := &config.Config{}

			svc := NewInterviewSessionRepository(lgr, mockStore, cfg)

			gotErr := svc.UpdateStartedAtInterviewSession(ctx, tC.input)

			tC.verify(t, gotErr)
		})
	}
}

func TestInterviewSessionRepository_GetStartedAndIsStartedConversationSession(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()
	updateID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	startedAt := time.Date(2025, 9, 4, 18, 35, 49, 777972000, time.FixedZone("UTC+7", 7*3600))

	testCases := []struct {
		name   string
		input  uuid.UUID
		setup  func() *mockSqlc.MockStore
		verify func(t *testing.T, gotResp *db.GetStartedAndIsStartedConversationSessionRow, gotErr error)
	}{
		{
			name:  "Success",
			input: updateID,
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					GetStartedAndIsStartedConversationSession(ctx, updateID).
					Return(db.GetStartedAndIsStartedConversationSessionRow{
						StartedAt:             sql.NullTime{Time: startedAt, Valid: true},
						IsStartedConversation: sql.NullBool{Bool: true, Valid: true},
					}, nil)

				return mockStore
			},
			verify: func(t *testing.T, gotResp *db.GetStartedAndIsStartedConversationSessionRow, gotErr error) {
				assert.NoError(t, gotErr)
				assert.Equal(t, startedAt, gotResp.StartedAt.Time)
				assert.True(t, gotResp.StartedAt.Valid)
				assert.Equal(t, true, gotResp.IsStartedConversation.Bool)
				assert.True(t, gotResp.IsStartedConversation.Valid)
			},
		},
		{
			name:  "Error - WithGetStartedAndIsStartedConversationSessionNotFound",
			input: updateID,
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					GetStartedAndIsStartedConversationSession(ctx, updateID).
					Return(db.GetStartedAndIsStartedConversationSessionRow{
						StartedAt:             sql.NullTime{},
						IsStartedConversation: sql.NullBool{Bool: true, Valid: true},
					}, sql.ErrNoRows)

				return mockStore
			},
			verify: func(t *testing.T, gotResp *db.GetStartedAndIsStartedConversationSessionRow, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "[INS0401]")
				assert.Contains(t, gotErr.Error(), "The session was not found. Please try again.")
				assert.Nil(t, gotResp)
			},
		},
		{
			name:  "Error - WithGetStartedAndIsStartedConversationSessionError",
			input: updateID,
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					GetStartedAndIsStartedConversationSession(ctx, updateID).
					Return(db.GetStartedAndIsStartedConversationSessionRow{}, errors.New("error"))

				return mockStore
			},
			verify: func(t *testing.T, gotResp *db.GetStartedAndIsStartedConversationSessionRow, gotErr error) {
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

			cfg := &config.Config{}

			svc := NewInterviewSessionRepository(lgr, mockStore, cfg)

			gotResp, gotErr := svc.GetStartedAndIsStartedConversationSession(ctx, tC.input)

			tC.verify(t, gotResp, gotErr)
		})
	}
}

func TestInterviewSessionRepository_UpdateIsStartedConversationSession(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()
	updateID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	req := &db.UpdateIsStartedConversationSessionParams{
		ID:                    updateID,
		IsStartedConversation: sql.NullBool{Bool: true, Valid: true},
	}

	testCases := []struct {
		name   string
		input  *db.UpdateIsStartedConversationSessionParams
		setup  func() *mockSqlc.MockStore
		verify func(t *testing.T, gotErr error)
	}{
		{
			name:  "Success",
			input: req,
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					UpdateIsStartedConversationSession(ctx, db.UpdateIsStartedConversationSessionParams{
						ID:                    updateID,
						IsStartedConversation: sql.NullBool{Bool: true, Valid: true},
					}).
					Return(1, nil)

				return mockStore
			},
			verify: func(t *testing.T, gotErr error) {
				assert.NoError(t, gotErr)
			},
		},
		{
			name:  "Error - WithGetStartedAndIsStartedConversationSessionNotFound",
			input: req,
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					UpdateIsStartedConversationSession(ctx, db.UpdateIsStartedConversationSessionParams{
						ID:                    req.ID,
						IsStartedConversation: sql.NullBool{Bool: true, Valid: true},
					}).
					Return(0, nil)

				return mockStore
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "[INS0401]")
				assert.Contains(t, gotErr.Error(), "The session was not found. Please try again.")
			},
		},
		{
			name:  "Error - WithGetStartedAndIsStartedConversationSessionError",
			input: req,
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					UpdateIsStartedConversationSession(ctx, db.UpdateIsStartedConversationSessionParams{
						ID:                    req.ID,
						IsStartedConversation: sql.NullBool{Bool: true, Valid: true},
					}).
					Return(0, errors.New("error"))

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

			cfg := &config.Config{}

			svc := NewInterviewSessionRepository(lgr, mockStore, cfg)

			gotErr := svc.UpdateIsStartedConversationSession(ctx, tC.input)

			tC.verify(t, gotErr)
		})
	}
}

func TestInterviewSessionRepository_DeleteUserInterviewSessionByID(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()
	updateID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	userID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	req := &db.DeleteUserInterviewSessionByIDParams{
		ID:     updateID,
		UserID: userID,
	}

	testCases := []struct {
		name   string
		input  *db.DeleteUserInterviewSessionByIDParams
		setup  func() *mockSqlc.MockStore
		verify func(t *testing.T, gotErr error)
	}{
		{
			name:  "Success",
			input: req,
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					DeleteUserInterviewSessionByID(ctx, *req).
					Return(1, nil)

				return mockStore
			},
			verify: func(t *testing.T, gotErr error) {
				assert.NoError(t, gotErr)
			},
		},
		{
			name:  "Error WithDeleteUserInterviewSessionByIDNotFound",
			input: req,
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					DeleteUserInterviewSessionByID(ctx, *req).
					Return(0, nil)

				return mockStore
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "[INS0401]")
				assert.Contains(t, gotErr.Error(), "The session was not found. Please try again.")
			},
		},
		{
			name:  "Error WithDeleteUserInterviewSessionByIDError",
			input: req,
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					DeleteUserInterviewSessionByID(ctx, *req).
					Return(0, errors.New("error"))

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

			cfg := &config.Config{}

			svc := NewInterviewSessionRepository(lgr, mockStore, cfg)

			gotErr := svc.DeleteUserInterviewSessionByID(ctx, tC.input)

			tC.verify(t, gotErr)
		})
	}
}

func TestInterviewSessionRepository_GetInterviewSessionInformationByID(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()
	updateID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	validResp := &db.GetInterviewSessionInformationByIDRow{
		UserID:         uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		ResumeID:       uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		ResumeFileName: "resume_file_name",
		Position:       "position",
		Status:         "status",
		StartedAt:      sql.NullTime{Time: time.Now(), Valid: true},
		EndedAt:        sql.NullTime{Time: time.Now(), Valid: true},
		OverallScore:   sql.NullFloat64{Float64: 100, Valid: true},
		SummaryMd:      sql.NullString{String: "summary_md", Valid: true},
		CreatedAt:      sql.NullTime{Time: time.Now(), Valid: true},
	}

	testCases := []struct {
		name   string
		input  uuid.UUID
		setup  func() *mockSqlc.MockStore
		verify func(t *testing.T, gotResp *db.GetInterviewSessionInformationByIDRow, gotErr error)
	}{
		{
			name:  "Success",
			input: updateID,
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					GetInterviewSessionInformationByID(ctx, updateID).
					Return(*validResp, nil)

				return mockStore
			},
			verify: func(t *testing.T, gotResp *db.GetInterviewSessionInformationByIDRow, gotErr error) {
				assert.NoError(t, gotErr)
				assert.Equal(t, validResp, gotResp)
			},
		},
		{
			name:  "Error WithGetInterviewSessionInformationByIDNotFound",
			input: updateID,
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					GetInterviewSessionInformationByID(ctx, updateID).
					Return(db.GetInterviewSessionInformationByIDRow{}, sql.ErrNoRows)

				return mockStore
			},
			verify: func(t *testing.T, gotResp *db.GetInterviewSessionInformationByIDRow, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "[INS0401]")
				assert.Contains(t, gotErr.Error(), "The session was not found. Please try again.")
				assert.Nil(t, gotResp)
			},
		}, {
			name:  "Error WithGetInterviewSessionInformationByIDError",
			input: updateID,
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					GetInterviewSessionInformationByID(ctx, updateID).
					Return(db.GetInterviewSessionInformationByIDRow{}, errors.New("error"))

				return mockStore
			},
			verify: func(t *testing.T, gotResp *db.GetInterviewSessionInformationByIDRow, gotErr error) {
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

			cfg := &config.Config{}

			svc := NewInterviewSessionRepository(lgr, mockStore, cfg)

			gotResp, gotErr := svc.GetInterviewSessionInformationByID(ctx, tC.input)

			tC.verify(t, gotResp, gotErr)
		})
	}
}

func TestInterviewSessionRepository_UpdateFinalizeStatusInterviewSessionByID(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()
	updateID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	finalizeReq := db.UpdateFinalizeStatusInterviewSessionByIDParams{
		ID: updateID,
		FinalizeStatus: db.NullFinalizeStatusEnum{
			FinalizeStatusEnum: db.FinalizeStatusEnumFinalizing,
			Valid:              true,
		},
	}

	testCases := []struct {
		name   string
		input  *db.UpdateFinalizeStatusInterviewSessionByIDParams
		setup  func() *mockSqlc.MockStore
		verify func(t *testing.T, gotErr error)
	}{
		{
			name:  "Success",
			input: &finalizeReq,
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					UpdateFinalizeStatusInterviewSessionByID(ctx, finalizeReq).
					Return(1, nil)

				return mockStore
			},
			verify: func(t *testing.T, gotErr error) {
				assert.NoError(t, gotErr)
			},
		},
		{
			name:  "Error WithUpdateFinalizeStatusInterviewSessionByIDNotFound",
			input: &finalizeReq,
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					UpdateFinalizeStatusInterviewSessionByID(ctx, finalizeReq).
					Return(0, nil)

				return mockStore
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "[INS0401]")
				assert.Contains(t, gotErr.Error(), "The session was not found. Please try again.")
			},
		}, {
			name:  "Error WithUpdateFinalizeStatusInterviewSessionByIDError",
			input: &finalizeReq,
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					UpdateFinalizeStatusInterviewSessionByID(ctx, finalizeReq).
					Return(0, errors.New("error"))

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

			cfg := &config.Config{}

			svc := NewInterviewSessionRepository(lgr, mockStore, cfg)

			gotErr := svc.UpdateFinalizeStatusInterviewSessionByID(ctx, tC.input)

			tC.verify(t, gotErr)
		})
	}
}
