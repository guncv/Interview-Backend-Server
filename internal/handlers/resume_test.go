package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/interview-simulation/interview-backend-server/internal/entities"
	app_error "gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
	"gitlab.com/interview-simulation/interview-backend-server/internal/mocks/middleware"
	"gitlab.com/interview-simulation/interview-backend-server/internal/mocks/services"
	"gitlab.com/interview-simulation/interview-backend-server/internal/mocks/utils"
)

func setupTestEnvironment(t *testing.T) (*gin.Engine, *services.MockResumeService, *middleware.MockAuthContext, *utils.MockValidator, *log.Logger) {
	gin.SetMode(gin.TestMode)

	engine := gin.New()

	mockResumeService := &services.MockResumeService{}
	mockAuthContext := &middleware.MockAuthContext{}
	mockValidator := &utils.MockValidator{}

	logger := log.Initialize("test")

	return engine, mockResumeService, mockAuthContext, mockValidator, logger
}

func createResumeHandler(mockResumeService *services.MockResumeService, mockAuthContext *middleware.MockAuthContext, mockValidator *utils.MockValidator, logger *log.Logger) *ResumeHandler {
	return NewResumeHandler(mockResumeService, logger, mockAuthContext, mockValidator)
}
func TestResumeHandler_ListResume(t *testing.T) {
	tests := []struct {
		name           string
		setupMocks     func(*services.MockResumeService, *middleware.MockAuthContext, *utils.MockValidator)
		requestBody    string
		expectedStatus int
		expectedError  bool
		expectedBody   string
	}{
		{
			name: "Success - Resume list retrieved successfully",
			setupMocks: func(mockService *services.MockResumeService, mockAuth *middleware.MockAuthContext, mockValidator *utils.MockValidator) {
				mockValidator.On("ValidateAndBind", mock.Anything, mock.Anything, "ListResume").Return(nil)
				mockAuth.On("ExtractAuthContext", mock.Anything).Return(context.Background(), nil)

				expectedResponse := &entities.ListResumeResponse{
					DefaultResume: entities.GetListResumeByIdResponse{
						ID:        "resume-1",
						FileName:  "resume.pdf",
						MimeType:  "application/pdf",
						ByteSize:  1024,
						CreatedAt: "2023-01-01T00:00:00Z",
						UpdatedAt: "2023-01-01T00:00:00Z",
					},
					Resumes: []entities.GetListResumeByIdResponse{
						{
							ID:        "resume-2",
							FileName:  "resume2.pdf",
							MimeType:  "application/pdf",
							ByteSize:  2048,
							CreatedAt: "2023-01-02T00:00:00Z",
							UpdatedAt: "2023-01-02T00:00:00Z",
						},
					},
				}
				mockService.On("ListResume", mock.Anything, mock.Anything).Return(expectedResponse, nil)
			},
			requestBody:    `{"updated_at":"2023-01-01T00:00:00Z"}`,
			expectedStatus: http.StatusOK,
			expectedError:  false,
			expectedBody:   `{"default_resume":{"id":"resume-1","file_name":"resume.pdf","mime_type":"application/pdf","byte_size":1024,"created_at":"2023-01-01T00:00:00Z","updated_at":"2023-01-01T00:00:00Z"},"resumes":[{"id":"resume-2","file_name":"resume2.pdf","mime_type":"application/pdf","byte_size":2048,"created_at":"2023-01-02T00:00:00Z","updated_at":"2023-01-02T00:00:00Z"}]}`,
		},
		{
			name: "Error - Validation failed",
			setupMocks: func(mockService *services.MockResumeService, mockAuth *middleware.MockAuthContext, mockValidator *utils.MockValidator) {
				mockValidator.On("ValidateAndBind", mock.Anything, mock.Anything, "ListResume").Return(errors.New("validation error"))
			},
			requestBody:    `{"invalid_field":"value"}`,
			expectedStatus: http.StatusInternalServerError,
			expectedError:  true,
			expectedBody:   "",
		},
		{
			name: "Error - Auth context extraction failed",
			setupMocks: func(mockService *services.MockResumeService, mockAuth *middleware.MockAuthContext, mockValidator *utils.MockValidator) {
				mockValidator.On("ValidateAndBind", mock.Anything, mock.Anything, "ListResume").Return(nil)
				mockAuth.On("ExtractAuthContext", mock.Anything).Return(nil, errors.New("auth error"))
			},
			requestBody:    `{"updated_at":"2023-01-01T00:00:00Z"}`,
			expectedStatus: http.StatusInternalServerError,
			expectedError:  true,
			expectedBody:   "",
		},
		{
			name: "Error - Service list failed",
			setupMocks: func(mockService *services.MockResumeService, mockAuth *middleware.MockAuthContext, mockValidator *utils.MockValidator) {
				mockValidator.On("ValidateAndBind", mock.Anything, mock.Anything, "ListResume").Return(nil)
				mockAuth.On("ExtractAuthContext", mock.Anything).Return(context.Background(), nil)
				mockService.On("ListResume", mock.Anything, mock.Anything).Return(nil, errors.New("service error"))
			},
			requestBody:    `{"updated_at":"2023-01-01T00:00:00Z"}`,
			expectedStatus: http.StatusInternalServerError,
			expectedError:  true,
			expectedBody:   "",
		},
		{
			name: "Success - Empty resume list",
			setupMocks: func(mockService *services.MockResumeService, mockAuth *middleware.MockAuthContext, mockValidator *utils.MockValidator) {
				mockValidator.On("ValidateAndBind", mock.Anything, mock.Anything, "ListResume").Return(nil)
				mockAuth.On("ExtractAuthContext", mock.Anything).Return(context.Background(), nil)

				expectedResponse := &entities.ListResumeResponse{
					DefaultResume: entities.GetListResumeByIdResponse{},
					Resumes:       []entities.GetListResumeByIdResponse{},
				}
				mockService.On("ListResume", mock.Anything, mock.Anything).Return(expectedResponse, nil)
			},
			requestBody:    `{}`,
			expectedStatus: http.StatusOK,
			expectedError:  false,
			expectedBody:   `{"default_resume":{"id":"","file_name":"","mime_type":"","byte_size":0,"created_at":"","updated_at":""},"resumes":[]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine, mockService, mockAuth, mockValidator, logger := setupTestEnvironment(t)
			handler := createResumeHandler(mockService, mockAuth, mockValidator, logger)

			tt.setupMocks(mockService, mockAuth, mockValidator)

			engine.POST("/resume/list", handler.ListResume)

			req, err := http.NewRequest("POST", "/resume/list", strings.NewReader(tt.requestBody))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError {
				assert.NotEmpty(t, w.Body.String())
			} else {
				// Remove whitespace for comparison
				actualBody := strings.TrimSpace(w.Body.String())
				expectedBody := strings.TrimSpace(tt.expectedBody)
				assert.Equal(t, expectedBody, actualBody)
			}

			mockService.AssertExpectations(t)
			mockAuth.AssertExpectations(t)
			mockValidator.AssertExpectations(t)
		})
	}
}

func TestResumeHandler_SwitchDefaultResume(t *testing.T) {
	tests := []struct {
		name           string
		setupMocks     func(*services.MockResumeService, *middleware.MockAuthContext, *utils.MockValidator)
		requestBody    string
		expectedStatus int
		expectedError  bool
	}{
		{
			name: "Success - Default resume switched successfully",
			setupMocks: func(mockService *services.MockResumeService, mockAuth *middleware.MockAuthContext, mockValidator *utils.MockValidator) {
				mockValidator.On("ValidateAndBind", mock.Anything, mock.Anything, "SwitchDefaultResume").Return(nil)
				mockAuth.On("ExtractAuthContext", mock.Anything).Return(context.Background(), nil)
				mockService.On("SwitchDefaultResume", mock.Anything, mock.Anything).Return(nil)
			},
			requestBody:    `{"resume_id":"resume-123"}`,
			expectedStatus: http.StatusOK,
			expectedError:  false,
		},
		{
			name: "Error - Validation failed",
			setupMocks: func(mockService *services.MockResumeService, mockAuth *middleware.MockAuthContext, mockValidator *utils.MockValidator) {
				mockValidator.On("ValidateAndBind", mock.Anything, mock.Anything, "SwitchDefaultResume").Return(errors.New("validation error"))
			},
			requestBody:    `{"resume_id":""}`,
			expectedStatus: http.StatusInternalServerError,
			expectedError:  true,
		},
		{
			name: "Error - Auth context extraction failed",
			setupMocks: func(mockService *services.MockResumeService, mockAuth *middleware.MockAuthContext, mockValidator *utils.MockValidator) {
				mockValidator.On("ValidateAndBind", mock.Anything, mock.Anything, "SwitchDefaultResume").Return(nil)
				mockAuth.On("ExtractAuthContext", mock.Anything).Return(nil, errors.New("auth error"))
			},
			requestBody:    `{"resume_id":"resume-123"}`,
			expectedStatus: http.StatusInternalServerError,
			expectedError:  true,
		},
		{
			name: "Error - Service switch failed",
			setupMocks: func(mockService *services.MockResumeService, mockAuth *middleware.MockAuthContext, mockValidator *utils.MockValidator) {
				mockValidator.On("ValidateAndBind", mock.Anything, mock.Anything, "SwitchDefaultResume").Return(nil)
				mockAuth.On("ExtractAuthContext", mock.Anything).Return(context.Background(), nil)
				mockService.On("SwitchDefaultResume", mock.Anything, mock.Anything).Return(errors.New("service error"))
			},
			requestBody:    `{"resume_id":"resume-123"}`,
			expectedStatus: http.StatusInternalServerError,
			expectedError:  true,
		},
		{
			name: "Error - Missing resume_id field",
			setupMocks: func(mockService *services.MockResumeService, mockAuth *middleware.MockAuthContext, mockValidator *utils.MockValidator) {
				mockValidator.On("ValidateAndBind", mock.Anything, mock.Anything, "SwitchDefaultResume").Return(errors.New("resume_id is required"))
			},
			requestBody:    `{}`,
			expectedStatus: http.StatusInternalServerError,
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine, mockService, mockAuth, mockValidator, logger := setupTestEnvironment(t)
			handler := createResumeHandler(mockService, mockAuth, mockValidator, logger)

			tt.setupMocks(mockService, mockAuth, mockValidator)

			engine.POST("/resume/switch-default", handler.SwitchDefaultResume)

			req, err := http.NewRequest("POST", "/resume/switch-default", strings.NewReader(tt.requestBody))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError {
				assert.NotEmpty(t, w.Body.String())
			} else {
				// For successful responses, body should be empty or contain expected data
				body := w.Body.String()
				assert.True(t, body == "" || body == "null" || body == "{}", "Expected empty body but got: %s", body)
			}

			mockService.AssertExpectations(t)
			mockAuth.AssertExpectations(t)
			mockValidator.AssertExpectations(t)
		})
	}
}

func TestResumeHandler_GetResumeByID(t *testing.T) {
	tests := []struct {
		name           string
		setupMocks     func(*services.MockResumeService, *utils.MockValidator)
		resumeID       string
		expectedStatus int
		expectedError  bool
		expectedBody   string
	}{
		{
			name: "Success - Resume retrieved successfully",
			setupMocks: func(mockService *services.MockResumeService, mockValidator *utils.MockValidator) {
				// Mock validator success for UUID validation
				validate := validator.New()
				mockValidator.On("GetValidate").Return(validate)

				// Mock service success
				expectedResponse := &entities.GetResumeByIDResponse{
					ID:        "550e8400-e29b-41d4-a716-446655440000",
					FileName:  "resume.pdf",
					MimeType:  "application/pdf",
					ByteSize:  1024,
					FileUrl:   "https://example.com/resume.pdf",
					CreatedAt: "2023-01-01T00:00:00Z",
					UpdatedAt: "2023-01-01T00:00:00Z",
				}
				mockService.On("GetResumeByID", mock.Anything, &entities.GetResumeByIDRequest{
					ResumeID: "550e8400-e29b-41d4-a716-446655440000",
				}).Return(expectedResponse, nil)
			},
			resumeID:       "550e8400-e29b-41d4-a716-446655440000",
			expectedStatus: http.StatusOK,
			expectedError:  false,
			expectedBody:   `{"id":"550e8400-e29b-41d4-a716-446655440000","file_name":"resume.pdf","mime_type":"application/pdf","byte_size":1024,"file_url":"https://example.com/resume.pdf","created_at":"2023-01-01T00:00:00Z","updated_at":"2023-01-01T00:00:00Z"}`,
		},
		{
			name: "Error - Missing resume ID (empty string)",
			setupMocks: func(mockService *services.MockResumeService, mockValidator *utils.MockValidator) {
				// No mocks needed for this case - this tests the early return when resumeId is empty
			},
			resumeID:       "",
			expectedStatus: http.StatusNotFound, // Gin router won't match empty parameter
			expectedError:  true,
			expectedBody:   "",
		},
		{
			name: "Error - Resume ID is just whitespace",
			setupMocks: func(mockService *services.MockResumeService, mockValidator *utils.MockValidator) {
				// Mock validator to return error for whitespace-only input
				validate := validator.New()
				mockValidator.On("GetValidate").Return(validate)
			},
			resumeID:       "   ",
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
			expectedBody:   "",
		},
		{
			name: "Error - Resume ID contains only spaces",
			setupMocks: func(mockService *services.MockResumeService, mockValidator *utils.MockValidator) {
				// Mock validator to return error for whitespace-only input
				validate := validator.New()
				mockValidator.On("GetValidate").Return(validate)
			},
			resumeID:       "   ",
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
			expectedBody:   "",
		},
		{
			name: "Error - Invalid UUID format (not a UUID)",
			setupMocks: func(mockService *services.MockResumeService, mockValidator *utils.MockValidator) {
				validate := validator.New()
				mockValidator.On("GetValidate").Return(validate)
			},
			resumeID:       "invalid-uuid",
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
			expectedBody:   "",
		},
		{
			name: "Error - Invalid UUID format (wrong length)",
			setupMocks: func(mockService *services.MockResumeService, mockValidator *utils.MockValidator) {
				validate := validator.New()
				mockValidator.On("GetValidate").Return(validate)
			},
			resumeID:       "123",
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
			expectedBody:   "",
		},
		{
			name: "Error - Invalid UUID format (missing hyphens)",
			setupMocks: func(mockService *services.MockResumeService, mockValidator *utils.MockValidator) {
				validate := validator.New()
				mockValidator.On("GetValidate").Return(validate)
			},
			resumeID:       "550e8400e29b41d4a716446655440000",
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
			expectedBody:   "",
		},
		{
			name: "Error - Service get failed with generic error",
			setupMocks: func(mockService *services.MockResumeService, mockValidator *utils.MockValidator) {
				validate := validator.New()
				mockValidator.On("GetValidate").Return(validate)

				mockService.On("GetResumeByID", mock.Anything, &entities.GetResumeByIDRequest{
					ResumeID: "550e8400-e29b-41d4-a716-446655440000",
				}).Return(nil, errors.New("database connection failed"))
			},
			resumeID:       "550e8400-e29b-41d4-a716-446655440000",
			expectedStatus: http.StatusInternalServerError,
			expectedError:  true,
			expectedBody:   "",
		},
		{
			name: "Error - Resume not found (404)",
			setupMocks: func(mockService *services.MockResumeService, mockValidator *utils.MockValidator) {
				validate := validator.New()
				mockValidator.On("GetValidate").Return(validate)

				appErr := app_error.New(errors.New("resume not found"), app_error.ErrCodeResumeNotFound)
				mockService.On("GetResumeByID", mock.Anything, &entities.GetResumeByIDRequest{
					ResumeID: "550e8400-e29b-41d4-a716-446655440000",
				}).Return(nil, appErr)
			},
			resumeID:       "550e8400-e29b-41d4-a716-446655440000",
			expectedStatus: http.StatusNotFound,
			expectedError:  true,
			expectedBody:   "",
		},
		{
			name: "Error - Resume invalid ID (400)",
			setupMocks: func(mockService *services.MockResumeService, mockValidator *utils.MockValidator) {
				validate := validator.New()
				mockValidator.On("GetValidate").Return(validate)

				appErr := app_error.New(errors.New("invalid resume ID"), app_error.ErrCodeResumeInvalidID)
				mockService.On("GetResumeByID", mock.Anything, &entities.GetResumeByIDRequest{
					ResumeID: "550e8400-e29b-41d4-a716-446655440000",
				}).Return(nil, appErr)
			},
			resumeID:       "550e8400-e29b-41d4-a716-446655440000",
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
			expectedBody:   "",
		},
		{
			name: "Error - Resume invalid request (400)",
			setupMocks: func(mockService *services.MockResumeService, mockValidator *utils.MockValidator) {
				validate := validator.New()
				mockValidator.On("GetValidate").Return(validate)

				appErr := app_error.New(errors.New("invalid request"), app_error.ErrCodeResumeInvalidRequest)
				mockService.On("GetResumeByID", mock.Anything, &entities.GetResumeByIDRequest{
					ResumeID: "550e8400-e29b-41d4-a716-446655440000",
				}).Return(nil, appErr)
			},
			resumeID:       "550e8400-e29b-41d4-a716-446655440000",
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
			expectedBody:   "",
		},
		{
			name: "Edge case - Very long UUID that exceeds normal length",
			setupMocks: func(mockService *services.MockResumeService, mockValidator *utils.MockValidator) {
				validate := validator.New()
				mockValidator.On("GetValidate").Return(validate)
			},
			resumeID:       "550e8400-e29b-41d4-a716-446655440000-very-long-uuid-that-exceeds-normal-length",
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
			expectedBody:   "",
		},
		{
			name: "Edge case - Special characters in UUID",
			setupMocks: func(mockService *services.MockResumeService, mockValidator *utils.MockValidator) {
				validate := validator.New()
				mockValidator.On("GetValidate").Return(validate)
			},
			resumeID:       "550e8400-e29b-41d4-a716-44665544000!",
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
			expectedBody:   "",
		},
		{
			name: "Edge case - UUID with spaces",
			setupMocks: func(mockService *services.MockResumeService, mockValidator *utils.MockValidator) {
				validate := validator.New()
				mockValidator.On("GetValidate").Return(validate)
			},
			resumeID:       "550e8400-e29b-41d4-a716-446655440000 ",
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
			expectedBody:   "",
		},
		{
			name: "Edge case - UUID with leading/trailing spaces",
			setupMocks: func(mockService *services.MockResumeService, mockValidator *utils.MockValidator) {
				validate := validator.New()
				mockValidator.On("GetValidate").Return(validate)
			},
			resumeID:       " 550e8400-e29b-41d4-a716-446655440000 ",
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
			expectedBody:   "",
		},
		{
			name: "Edge case - UUID with mixed case",
			setupMocks: func(mockService *services.MockResumeService, mockValidator *utils.MockValidator) {
				validate := validator.New()
				mockValidator.On("GetValidate").Return(validate)
			},
			resumeID:       "550E8400-E29B-41D4-A716-446655440000",
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
			expectedBody:   "",
		},
		{
			name: "Edge case - UUID with extra hyphens",
			setupMocks: func(mockService *services.MockResumeService, mockValidator *utils.MockValidator) {
				validate := validator.New()
				mockValidator.On("GetValidate").Return(validate)
			},
			resumeID:       "550e-8400-e29b-41d4-a716-446655440000",
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
			expectedBody:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine, mockService, _, mockValidator, logger := setupTestEnvironment(t)
			// Create handler without auth context for this test since GetResumeByID doesn't use it
			handler := &ResumeHandler{
				resumeService: mockService,
				log:           logger,
				validator:     mockValidator,
			}

			tt.setupMocks(mockService, mockValidator)

			engine.GET("/resume/:id", handler.GetResumeByID)

			url := "/resume/" + tt.resumeID
			req, err := http.NewRequest("GET", url, nil)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError {
				assert.NotEmpty(t, w.Body.String())
			} else {
				// Remove whitespace for comparison
				actualBody := strings.TrimSpace(w.Body.String())
				expectedBody := strings.TrimSpace(tt.expectedBody)
				assert.Equal(t, expectedBody, actualBody)
			}

			mockService.AssertExpectations(t)
			mockValidator.AssertExpectations(t)
		})
	}
}

func TestResumeHandler_EdgeCases(t *testing.T) {

	t.Run("ListResume with malformed JSON", func(t *testing.T) {
		engine, mockService, mockAuth, mockValidator, logger := setupTestEnvironment(t)
		handler := createResumeHandler(mockService, mockAuth, mockValidator, logger)

		// Mock validator to return error for malformed JSON
		mockValidator.On("ValidateAndBind", mock.Anything, mock.Anything, "ListResume").Return(errors.New("malformed JSON"))

		engine.POST("/resume/list", handler.ListResume)

		req, err := http.NewRequest("POST", "/resume/list", strings.NewReader(`{"invalid": json`))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		// Should fail due to malformed JSON
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("SwitchDefaultResume with very long resume ID", func(t *testing.T) {
		engine, mockService, mockAuth, mockValidator, logger := setupTestEnvironment(t)
		handler := createResumeHandler(mockService, mockAuth, mockValidator, logger)

		// Mock validator to return error for very long resume ID
		mockValidator.On("ValidateAndBind", mock.Anything, mock.Anything, "SwitchDefaultResume").Return(errors.New("resume ID too long"))

		engine.POST("/resume/switch-default", handler.SwitchDefaultResume)

		longResumeID := strings.Repeat("a", 1000)
		requestBody := `{"resume_id":"` + longResumeID + `"}`

		req, err := http.NewRequest("POST", "/resume/switch-default", strings.NewReader(requestBody))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		// Should fail due to validation
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestResumeHandler_ErrorHandling(t *testing.T) {

	t.Run("GetResumeByID with validation error", func(t *testing.T) {
		engine, mockService, _, mockValidator, logger := setupTestEnvironment(t)
		handler := &ResumeHandler{
			resumeService: mockService,
			log:           logger,
			validator:     mockValidator,
		}

		validate := validator.New()
		mockValidator.On("GetValidate").Return(validate)

		engine.GET("/resume/:id", handler.GetResumeByID)

		req, err := http.NewRequest("GET", "/resume/invalid-uuid", nil)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		// Should fail due to invalid UUID
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestResumeHandler_ConcurrentRequests(t *testing.T) {
	t.Run("Multiple concurrent GetResumeByID requests", func(t *testing.T) {
		engine, mockService, _, mockValidator, logger := setupTestEnvironment(t)
		handler := &ResumeHandler{
			resumeService: mockService,
			log:           logger,
			validator:     mockValidator,
		}

		validate := validator.New()
		mockValidator.On("GetValidate").Return(validate)

		expectedResponse := &entities.GetResumeByIDResponse{
			ID:        "550e8400-e29b-41d4-a716-446655440000",
			FileName:  "resume.pdf",
			MimeType:  "application/pdf",
			ByteSize:  1024,
			FileUrl:   "https://example.com/resume.pdf",
			CreatedAt: "2023-01-01T00:00:00Z",
			UpdatedAt: "2023-01-01T00:00:00Z",
		}

		// Mock service to handle multiple concurrent calls
		mockService.On("GetResumeByID", mock.Anything, mock.Anything).Return(expectedResponse, nil)

		engine.GET("/resume/:id", handler.GetResumeByID)

		// Make multiple concurrent requests
		const numRequests = 10
		results := make(chan int, numRequests)

		for i := 0; i < numRequests; i++ {
			go func() {
				req, err := http.NewRequest("GET", "/resume/550e8400-e29b-41d4-a716-446655440000", nil)
				require.NoError(t, err)

				w := httptest.NewRecorder()
				engine.ServeHTTP(w, req)
				results <- w.Code
			}()
		}

		// Collect results
		for i := 0; i < numRequests; i++ {
			statusCode := <-results
			assert.Equal(t, http.StatusOK, statusCode)
		}

		mockService.AssertExpectations(t)
		mockValidator.AssertExpectations(t)
	})
}

func TestResumeHandler_NewResumeHandler(t *testing.T) {
	t.Run("Create new resume handler successfully", func(t *testing.T) {
		mockService := &services.MockResumeService{}
		mockAuth := &middleware.MockAuthContext{}
		mockValidator := &utils.MockValidator{}
		logger := log.Initialize("test")

		handler := NewResumeHandler(mockService, logger, mockAuth, mockValidator)

		assert.NotNil(t, handler)
		assert.Equal(t, mockService, handler.resumeService)
		assert.Equal(t, logger, handler.log)
		assert.Equal(t, mockAuth, handler.authContext)
		assert.Equal(t, mockValidator, handler.validator)
	})

	t.Run("Create new resume handler with nil dependencies", func(t *testing.T) {
		handler := NewResumeHandler(nil, nil, nil, nil)

		assert.NotNil(t, handler)
		assert.Nil(t, handler.resumeService)
		assert.Nil(t, handler.log)
		assert.Nil(t, handler.authContext)
		assert.Nil(t, handler.validator)
	})
}
