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
					"[ONX0401] The session was not found. Please try again. | interview session not found",
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
				OverallScore: sql.NullString{String: "100", Valid: true},
				SummaryMd:    sql.NullString{String: "summary", Valid: true},
			},
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					EndInterviewSession(ctx, db.EndInterviewSessionParams{
						ID:           updateID,
						Status:       updateStatus,
						EndedAt:      sql.NullTime{Time: endAt, Valid: true},
						OverallScore: sql.NullString{String: "100", Valid: true},
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
				OverallScore: sql.NullString{String: "100", Valid: true},
				SummaryMd:    sql.NullString{String: "summary", Valid: true},
			},
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					EndInterviewSession(ctx, db.EndInterviewSessionParams{
						ID:           updateID,
						Status:       updateStatus,
						EndedAt:      sql.NullTime{Time: endAt, Valid: true},
						OverallScore: sql.NullString{String: "100", Valid: true},
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
				OverallScore: sql.NullString{String: "100", Valid: true},
				SummaryMd:    sql.NullString{String: "summary", Valid: true},
			},
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					EndInterviewSession(ctx, db.EndInterviewSessionParams{
						ID:           updateID,
						Status:       updateStatus,
						EndedAt:      sql.NullTime{Time: endAt, Valid: true},
						OverallScore: sql.NullString{String: "100", Valid: true},
						SummaryMd:    sql.NullString{String: "summary", Valid: true},
					}).
					Return(0, nil)

				return mockStore
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Equal(
					t, gotErr.Error(),
					"[ONX0401] The session was not found. Please try again. | interview session not found",
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
