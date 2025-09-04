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

func TestResumeRepository_GetResumeJsonWithSummaryData(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()

	// Test data
	sessionID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	validReq := &GetResumeJsonWithSummaryDataReq{
		SessionID:       sessionID,
		Position:        "Software Engineer",
		Company:         "Tech Corp",
		WorkType:        "Full-time",
		JobRequirements: "Go, React, Docker",
		InterviewType:   "Technical",
		Language:        "English",
		ResumeFile: aws.NewCustomFileHeader(
			"resume.pdf",
			1024,
			make(map[string][]string),
			[]byte("mock file content"),
		),
	}

	validResponse := &GetResumeJsonWithSummaryDataResponse{
		ParsedJson: PromptInfo{
			FirstName: "John",
			LastName:  "Doe",
			Email:     "john.doe@example.com",
			Phone:     "+1234567890",
			Location:  "New York, NY",
			Experience: []Experience{
				{
					Company:     "Tech Corp",
					Position:    "Software Engineer",
					JobType:     "Full-time",
					StartDate:   "2020-01-01",
					EndDate:     "2023-12-31",
					Description: "Developed web applications",
				},
			},
			Education: []Education{
				{
					School:       "University of Technology",
					Degree:       "Bachelor of Science",
					FieldOfStudy: "Computer Science",
					StartDate:    "2016-09-01",
					EndDate:      "2020-05-31",
					Description:  "Graduated with honors",
				},
			},
			Skills:         []string{"Go", "React", "Docker"},
			Certifications: []string{"AWS Certified Developer"},
			Language:       "English",
		},
	}

	// Mock config - will be overridden in test cases

	testCases := []struct {
		name           string
		input          *GetResumeJsonWithSummaryDataReq
		setup          func() *mockSqlc.MockStore
		serverResponse func(w http.ResponseWriter, r *http.Request)
		verify         func(t *testing.T, got *GetResumeJsonWithSummaryDataResponse, gotErr error)
	}{
		{
			name:  "Success - With resume file",
			input: validReq,
			setup: func() *mockSqlc.MockStore {
				return new(mockSqlc.MockStore)
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "/api/v1/interview/requirements", r.URL.Path)
				assert.Contains(t, r.Header.Get("Content-Type"), "multipart/form-data")

				// Parse multipart form
				err := r.ParseMultipartForm(10 << 20) // 10 MB
				assert.NoError(t, err)

				// Verify form fields
				assert.Equal(t, sessionID.String(), r.FormValue("session_id"))
				assert.Equal(t, "Software Engineer", r.FormValue("position"))
				assert.Equal(t, "Tech Corp", r.FormValue("company"))
				assert.Equal(t, "Full-time", r.FormValue("work_type"))
				assert.Equal(t, "Go, React, Docker", r.FormValue("job_requirements"))
				assert.Equal(t, "Technical", r.FormValue("interview_type"))
				assert.Equal(t, "English", r.FormValue("language"))

				// Verify file upload
				file, header, err := r.FormFile("resume_file")
				assert.NoError(t, err)
				assert.Equal(t, "resume.pdf", header.Filename)
				file.Close()

				// Return success response
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(validResponse)
			},
			verify: func(t *testing.T, got *GetResumeJsonWithSummaryDataResponse, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, got)
				assert.Equal(t, validResponse.ParsedJson.FirstName, got.ParsedJson.FirstName)
				assert.Equal(t, validResponse.ParsedJson.LastName, got.ParsedJson.LastName)
				assert.Equal(t, validResponse.ParsedJson.Email, got.ParsedJson.Email)
				assert.Len(t, got.ParsedJson.Experience, 1)
				assert.Len(t, got.ParsedJson.Education, 1)
				assert.Len(t, got.ParsedJson.Skills, 3)
			},
		},
		{
			name: "Success - Without resume file",
			input: &GetResumeJsonWithSummaryDataReq{
				SessionID:       sessionID,
				Position:        "Software Engineer",
				Company:         "Tech Corp",
				WorkType:        "Full-time",
				JobRequirements: "Go, React, Docker",
				InterviewType:   "Technical",
				Language:        "English",
				ResumeFile:      nil,
			},
			setup: func() *mockSqlc.MockStore {
				return new(mockSqlc.MockStore)
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				// Verify the request
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "/api/v1/interview/requirements", r.URL.Path)

				err := r.ParseMultipartForm(10 << 20)
				assert.NoError(t, err)

				assert.Equal(t, sessionID.String(), r.FormValue("session_id"))
				assert.Equal(t, "Software Engineer", r.FormValue("position"))

				_, _, err = r.FormFile("resume_file")
				assert.Error(t, err) // Should error because no file was uploaded

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(validResponse)
			},
			verify: func(t *testing.T, got *GetResumeJsonWithSummaryDataResponse, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, got)
				assert.Equal(t, validResponse.ParsedJson.FirstName, got.ParsedJson.FirstName)
			},
		},
		{
			name:  "Error - HTTP request failed",
			input: validReq,
			setup: func() *mockSqlc.MockStore {
				return new(mockSqlc.MockStore)
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Internal Server Error"))
			},
			verify: func(t *testing.T, got *GetResumeJsonWithSummaryDataResponse, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
				assert.Contains(t, gotErr.Error(), "interview agent returned status: 500 Internal Server Error")
			},
		},
		{
			name:  "Error - Bad request",
			input: validReq,
			setup: func() *mockSqlc.MockStore {
				return new(mockSqlc.MockStore)
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("Bad Request"))
			},
			verify: func(t *testing.T, got *GetResumeJsonWithSummaryDataResponse, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
				assert.Contains(t, gotErr.Error(), "interview agent returned status: 400 Bad Request")
			},
		},
		{
			name:  "Error - Invalid JSON response",
			input: validReq,
			setup: func() *mockSqlc.MockStore {
				return new(mockSqlc.MockStore)
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("invalid json"))
			},
			verify: func(t *testing.T, got *GetResumeJsonWithSummaryDataResponse, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
				assert.Contains(t, gotErr.Error(), "invalid character")
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

			got, gotErr := repo.GetResumeJsonWithSummaryData(ctx, tC.input)

			tC.verify(t, got, gotErr)
		})
	}
}

func TestResumeRepository_GetResumeJsonWithSummaryData_FileErrors(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"parsed_json": map[string]interface{}{
				"first_name": "John",
				"last_name":  "Doe",
			},
		})
	}))
	defer server.Close()

	validFile := aws.NewCustomFileHeader(
		"test.pdf",
		1024,
		make(map[string][]string),
		[]byte("mock file content"),
	)

	req := &GetResumeJsonWithSummaryDataReq{
		SessionID:       uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
		Position:        "Software Engineer",
		Company:         "Tech Corp",
		WorkType:        "Full-time",
		JobRequirements: "Go, React, Docker",
		InterviewType:   "Technical",
		Language:        "English",
		ResumeFile:      validFile,
	}

	mockConfig := &config.Config{
		InterviewSessionConfig: config.InterviewSessionConfig{
			InterviewAgentURL: server.URL,
		},
	}

	mockStore := new(mockSqlc.MockStore)
	repo := NewResumeRepository(lgr, mockStore, mockConfig)

	got, gotErr := repo.GetResumeJsonWithSummaryData(ctx, req)

	assert.NoError(t, gotErr)
	assert.NotNil(t, got)
}

func TestResumeRepository_GetResumeJsonWithSummaryData_EdgeCases(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()

	testCases := []struct {
		name           string
		input          *GetResumeJsonWithSummaryDataReq
		setup          func() *mockSqlc.MockStore
		serverResponse func(w http.ResponseWriter, r *http.Request)
		verify         func(t *testing.T, got *GetResumeJsonWithSummaryDataResponse, gotErr error)
	}{
		{
			name: "Error - Empty response body",
			input: &GetResumeJsonWithSummaryDataReq{
				SessionID:       uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
				Position:        "Software Engineer",
				Company:         "Tech Corp",
				WorkType:        "Full-time",
				JobRequirements: "Go, React, Docker",
				InterviewType:   "Technical",
				Language:        "English",
				ResumeFile:      nil,
			},
			setup: func() *mockSqlc.MockStore {
				return new(mockSqlc.MockStore)
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(""))
			},
			verify: func(t *testing.T, got *GetResumeJsonWithSummaryDataResponse, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
				assert.Contains(t, gotErr.Error(), "unexpected end of JSON input")
			},
		},
		{
			name: "Error - Malformed JSON response",
			input: &GetResumeJsonWithSummaryDataReq{
				SessionID:       uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
				Position:        "Software Engineer",
				Company:         "Tech Corp",
				WorkType:        "Full-time",
				JobRequirements: "Go, React, Docker",
				InterviewType:   "Technical",
				Language:        "English",
				ResumeFile:      nil,
			},
			setup: func() *mockSqlc.MockStore {
				return new(mockSqlc.MockStore)
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"parsed_json": {"first_name": "John"`))
			},
			verify: func(t *testing.T, got *GetResumeJsonWithSummaryDataResponse, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
				assert.Contains(t, gotErr.Error(), "unexpected end of JSON input")
			},
		},
		{
			name: "Error - Server timeout simulation",
			input: &GetResumeJsonWithSummaryDataReq{
				SessionID:       uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
				Position:        "Software Engineer",
				Company:         "Tech Corp",
				WorkType:        "Full-time",
				JobRequirements: "Go, React, Docker",
				InterviewType:   "Technical",
				Language:        "English",
				ResumeFile:      nil,
			},
			setup: func() *mockSqlc.MockStore {
				return new(mockSqlc.MockStore)
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(2 * time.Second)
				w.WriteHeader(http.StatusOK)
			},
			verify: func(t *testing.T, got *GetResumeJsonWithSummaryDataResponse, gotErr error) {
				if gotErr != nil {
					assert.Nil(t, got)
				}
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

			got, gotErr := repo.GetResumeJsonWithSummaryData(ctx, tC.input)

			tC.verify(t, got, gotErr)
		})
	}
}

func TestResumeRepository_GetResumeJsonWithSummaryData_FileCopyErrors(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"parsed_json": map[string]interface{}{
				"first_name": "John",
				"last_name":  "Doe",
			},
		})
	}))
	defer server.Close()

	validFile := aws.NewCustomFileHeader(
		"test.pdf",
		1024,
		make(map[string][]string),
		[]byte("mock file content"),
	)

	req := &GetResumeJsonWithSummaryDataReq{
		SessionID:       uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
		Position:        "Software Engineer",
		Company:         "Tech Corp",
		WorkType:        "Full-time",
		JobRequirements: "Go, React, Docker",
		InterviewType:   "Technical",
		Language:        "English",
		ResumeFile:      validFile,
	}

	mockConfig := &config.Config{
		InterviewSessionConfig: config.InterviewSessionConfig{
			InterviewAgentURL: server.URL,
		},
	}

	mockStore := new(mockSqlc.MockStore)
	repo := NewResumeRepository(lgr, mockStore, mockConfig)

	got, gotErr := repo.GetResumeJsonWithSummaryData(ctx, req)

	assert.NoError(t, gotErr)
	assert.NotNil(t, got)
}
