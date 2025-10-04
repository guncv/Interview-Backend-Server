package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.com/interview-simulation/interview-backend-server/internal/config"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	db "gitlab.com/interview-simulation/interview-backend-server/internal/db/sqlc"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/aws"
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
				assert.Contains(t, gotErr.Error(), "[INS0304]")
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
				assert.Contains(t, gotErr.Error(), "[INS0101]")
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

func TestResumeRepository_GetDefaultResumeByUserID(t *testing.T) {
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
			name:  "Success - Get default resume by user ID",
			input: successResp.ID,
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					GetDefaultResumeByUserID(ctx, successResp.ID).
					Return(successResp, nil)

				return mockStore
			},
			verify: func(t *testing.T, got *db.Resumes, gotErr error) {
				assert.NoError(t, gotErr)
				assert.Equal(t, got, &successResp)
			},
		},
		{
			name:  "Error - Get default resume by user ID not found",
			input: successResp.ID,
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					GetDefaultResumeByUserID(ctx, successResp.ID).
					Return(db.Resumes{}, sql.ErrNoRows)

				return mockStore
			},
			verify: func(t *testing.T, got *db.Resumes, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
				assert.Contains(t, gotErr.Error(), "[INS0304]")
				assert.Contains(t, gotErr.Error(), "The resume was not found")
			},
		},
		{
			name:  "Error - Get default resume by user ID",
			input: successResp.ID,
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					GetDefaultResumeByUserID(ctx, successResp.ID).
					Return(db.Resumes{}, errors.New("error"))

				return mockStore
			},
			verify: func(t *testing.T, got *db.Resumes, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "[INS0101]")
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
			got, gotErr := svc.GetDefaultResumeByUserID(ctx, tC.input)

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
				assert.Contains(t, gotErr.Error(), "[INS0101]")
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
				assert.Contains(t, gotErr.Error(), "[INS0101]")
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

func TestResumeRepository_ExtractResumeJsonForRAG(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()

	sessionID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	userID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	resumeID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	validReq := &ExtractResumeJsonForRAGReq{
		SessionID: sessionID.String(),
		UserID:    userID.String(),
		ResumeID:  resumeID.String(),

		ResumeFile: aws.NewCustomFileHeader(
			"resume.pdf",
			1024,
			make(map[string][]string),
			[]byte("mock file content"),
		),
	}

	testCases := []struct {
		name           string
		input          *ExtractResumeJsonForRAGReq
		setup          func() *mockSqlc.MockStore
		serverResponse func(w http.ResponseWriter, r *http.Request)
		verify         func(t *testing.T, gotResp *ExtractResumeJsonForRAGResp, gotErr error)
	}{
		{
			name:  "Success - With resume file",
			input: validReq,
			setup: func() *mockSqlc.MockStore {
				return nil
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "/api/v1/interview/requirements", r.URL.Path)
				assert.Contains(t, r.Header.Get("Content-Type"), "multipart/form-data")

				err := r.ParseMultipartForm(10 << 20)
				assert.NoError(t, err)

				assert.Equal(t, sessionID.String(), r.FormValue("session_id"))
				assert.Equal(t, userID.String(), r.FormValue("user_id"))
				assert.Equal(t, resumeID.String(), r.FormValue("resume_id"))

				file, header, err := r.FormFile("resume_file")
				assert.NoError(t, err)
				assert.Equal(t, "resume.pdf", header.Filename)
				file.Close()
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(ExtractResumeJsonForRAGResp{
					BiasPrompt: "bias_prompt",
				})
			},
			verify: func(t *testing.T, gotResp *ExtractResumeJsonForRAGResp, gotErr error) {
				assert.NoError(t, gotErr)
				assert.Equal(t, &ExtractResumeJsonForRAGResp{
					BiasPrompt:    "bias_prompt",
					ResumeContext: json.RawMessage("null"),
				}, gotResp)
			},
		},
		{
			name: "Success - Without resume file",
			input: &ExtractResumeJsonForRAGReq{
				SessionID:  sessionID.String(),
				UserID:     userID.String(),
				ResumeID:   resumeID.String(),
				ResumeFile: nil,
			},
			setup: func() *mockSqlc.MockStore {
				return nil
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "/api/v1/interview/requirements", r.URL.Path)

				err := r.ParseMultipartForm(10 << 20)
				assert.NoError(t, err)

				assert.Equal(t, sessionID.String(), r.FormValue("session_id"))
				assert.Equal(t, userID.String(), r.FormValue("user_id"))
				assert.Equal(t, resumeID.String(), r.FormValue("resume_id"))

				_, _, err = r.FormFile("resume_file")
				assert.Error(t, err)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(ExtractResumeJsonForRAGResp{
					BiasPrompt: "bias_prompt",
				})
			},
			verify: func(t *testing.T, gotResp *ExtractResumeJsonForRAGResp, gotErr error) {
				assert.NoError(t, gotErr)
				assert.Equal(t, &ExtractResumeJsonForRAGResp{
					BiasPrompt:    "bias_prompt",
					ResumeContext: json.RawMessage("null"),
				}, gotResp)
			},
		},
		{
			name:  "Error - HTTP request failed",
			input: validReq,
			setup: func() *mockSqlc.MockStore {
				return nil
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Internal Server Error"))
			},
			verify: func(t *testing.T, gotResp *ExtractResumeJsonForRAGResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "interview agent returned status: 500 Internal Server Error")
				assert.Nil(t, gotResp)
			},
		},
		{
			name:  "Error - Bad request",
			input: validReq,
			setup: func() *mockSqlc.MockStore {
				return nil
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("Bad Request"))
			},
			verify: func(t *testing.T, gotResp *ExtractResumeJsonForRAGResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "interview agent returned status: 400 Bad Request")
				assert.Nil(t, gotResp)
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tC.serverResponse))
			defer server.Close()

			testConfig := &config.Config{
				InterviewSessionConfig: config.InterviewSessionConfig{
					InterviewAgentURL: server.URL,
				},
			}

			mockStore := tC.setup()

			defer func() {
				if mockStore != nil {
					mockStore.AssertExpectations(t)
				}
			}()

			repo := NewResumeRepository(lgr, mockStore, testConfig)
			gotResp, gotErr := repo.ExtractResumeJsonForRAG(ctx, tC.input)

			tC.verify(t, gotResp, gotErr)
		})
	}
}
