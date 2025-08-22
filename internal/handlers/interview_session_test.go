package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.com/interview-simulation/interview-backend-server/internal/entities"
	app_error "gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/websocket"
	"gitlab.com/interview-simulation/interview-backend-server/internal/mocks/middleware"
	"gitlab.com/interview-simulation/interview-backend-server/internal/mocks/services"
	"gitlab.com/interview-simulation/interview-backend-server/internal/mocks/utils"
	ws "gitlab.com/interview-simulation/interview-backend-server/internal/mocks/websocket"
)

func TestInterviewSessionHandler_CreateInterviewSessionWithNewResume(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log := log.Initialize("test")
	ctx := context.Background()
	err := errors.New("mock error")

	tests := []struct {
		name   string
		input  func() *entities.CreateInterviewSessionWithNewResumeRequest
		setup  func() (*services.MockInterviewSessionService, *utils.MockValidator, *middleware.MockAuthContext, websocket.WebSocketServerInterface)
		verify func(t *testing.T, w *httptest.ResponseRecorder)
	}{
		{
			name: "Success",
			input: func() *entities.CreateInterviewSessionWithNewResumeRequest {
				return &entities.CreateInterviewSessionWithNewResumeRequest{
					Position:        "Software Engineer",
					Company:         "Tech Corp",
					WorkType:        "Full-time",
					JobRequirements: "Go, REST APIs, Microservices",
					InterviewType:   "Technical",
					Language:        "English",
					IsConsent:       true,
				}
			},
			setup: func() (*services.MockInterviewSessionService, *utils.MockValidator, *middleware.MockAuthContext, websocket.WebSocketServerInterface) {
				mockInterviewSessionService := new(services.MockInterviewSessionService)
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)
				wsServer := new(ws.MockWebSocketServerInterface)

				mockValidator.EXPECT().
					ValidateAndBind(mock.Anything, mock.Anything, "CreateInterviewSessionWithNewResume").
					Return(nil)

				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, nil)

				mockInterviewSessionService.EXPECT().
					CreateInterviewSessionWithNewResume(ctx, mock.Anything).
					Return(&entities.CreateInterviewSessionWithNewResumeResponse{
						SessionToken: "session-token-123",
					}, nil)

				return mockInterviewSessionService, mockValidator, mockAuthContext, wsServer
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusCreated, w.Code)
				assert.JSONEq(t, `{"session_token":"session-token-123"}`, w.Body.String())
			},
		},
		{
			name: "Error With Validation",
			input: func() *entities.CreateInterviewSessionWithNewResumeRequest {
				return &entities.CreateInterviewSessionWithNewResumeRequest{}
			},
			setup: func() (*services.MockInterviewSessionService, *utils.MockValidator, *middleware.MockAuthContext, websocket.WebSocketServerInterface) {
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)
				mockInterviewSessionService := new(services.MockInterviewSessionService)
				wsServer := new(ws.MockWebSocketServerInterface)

				validationErr := app_error.New(err, app_error.ErrCodeAuthInvalidRequest)
				mockValidator.EXPECT().
					ValidateAndBind(mock.Anything, mock.Anything, "CreateInterviewSessionWithNewResume").
					Return(validationErr)

				return mockInterviewSessionService, mockValidator, mockAuthContext, wsServer
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, w.Code)
				assert.Contains(t, w.Body.String(), "Something went wrong with the request")
			},
		},
		{
			name: "Error With Auth Context",
			input: func() *entities.CreateInterviewSessionWithNewResumeRequest {
				return &entities.CreateInterviewSessionWithNewResumeRequest{}
			},
			setup: func() (*services.MockInterviewSessionService, *utils.MockValidator, *middleware.MockAuthContext, websocket.WebSocketServerInterface) {
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)
				mockInterviewSessionService := new(services.MockInterviewSessionService)
				wsServer := new(ws.MockWebSocketServerInterface)

				mockValidator.EXPECT().
					ValidateAndBind(mock.Anything, mock.Anything, "CreateInterviewSessionWithNewResume").
					Return(nil)

				authErr := app_error.New(err, app_error.ErrCodeAuthInvalidHeader)
				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, authErr)

				return mockInterviewSessionService, mockValidator, mockAuthContext, wsServer
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusUnauthorized, w.Code)
				assert.Contains(t, w.Body.String(), "Please log in to continue")
			},
		},
		{
			name: "Error With Service",
			input: func() *entities.CreateInterviewSessionWithNewResumeRequest {
				return &entities.CreateInterviewSessionWithNewResumeRequest{}
			},
			setup: func() (*services.MockInterviewSessionService, *utils.MockValidator, *middleware.MockAuthContext, websocket.WebSocketServerInterface) {
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)
				mockInterviewSessionService := new(services.MockInterviewSessionService)
				wsServer := new(ws.MockWebSocketServerInterface)

				mockValidator.EXPECT().
					ValidateAndBind(mock.Anything, mock.Anything, "CreateInterviewSessionWithNewResume").
					Return(nil)

				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, nil)

				serviceErr := app_error.New(err, app_error.ErrCodeResumeUploadFailed)
				mockInterviewSessionService.EXPECT().
					CreateInterviewSessionWithNewResume(ctx, mock.Anything).
					Return(nil, serviceErr)

				return mockInterviewSessionService, mockValidator, mockAuthContext, wsServer
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, w.Code)
				assert.Contains(t, w.Body.String(), "Failed to upload the file")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a multipart form for the request
			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)

			// Add form fields
			req := tt.input()
			if req.Position != "" {
				writer.WriteField("position", req.Position)
			}
			if req.Company != "" {
				writer.WriteField("company", req.Company)
			}
			if req.WorkType != "" {
				writer.WriteField("work_type", req.WorkType)
			}
			if req.JobRequirements != "" {
				writer.WriteField("job_requirements", req.JobRequirements)
			}
			if req.InterviewType != "" {
				writer.WriteField("interview_type", req.InterviewType)
			}
			if req.Language != "" {
				writer.WriteField("language", req.Language)
			}
			if req.IsConsent {
				writer.WriteField("is_consent", "true")
			}

			// Add a dummy file
			part, _ := writer.CreateFormFile("file", "test-resume.pdf")
			part.Write([]byte("dummy resume content"))
			writer.Close()

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPost, "/interview-sessions/new-resume", body)
			c.Request.Header.Set("Content-Type", writer.FormDataContentType())

			mockInterviewSessionService, mockValidator, mockAuthContext, wsServer := tt.setup()
			defer mockInterviewSessionService.AssertExpectations(t)
			defer mockValidator.AssertExpectations(t)
			defer mockAuthContext.AssertExpectations(t)

			handler := NewInterviewSessionHandler(mockInterviewSessionService, log, mockAuthContext, mockValidator, wsServer)
			handler.CreateInterviewSessionWithNewResume(c)

			tt.verify(t, w)
		})
	}
}

func TestInterviewSessionHandler_CreateInterviewSessionWithExistingResume(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log := log.Initialize("test")
	ctx := context.Background()
	err := errors.New("mock error")

	tests := []struct {
		name   string
		input  func() *entities.CreateInterviewSessionWithExistingResumeReq
		setup  func() (*services.MockInterviewSessionService, *utils.MockValidator, *middleware.MockAuthContext, websocket.WebSocketServerInterface)
		verify func(t *testing.T, w *httptest.ResponseRecorder)
	}{
		{
			name: "Success",
			input: func() *entities.CreateInterviewSessionWithExistingResumeReq {
				return &entities.CreateInterviewSessionWithExistingResumeReq{
					ResumeID:        "123e4567-e89b-12d3-a456-426614174000",
					Position:        "Software Engineer",
					Company:         "Tech Corp",
					WorkType:        "Full-time",
					JobRequirements: "Go, REST APIs, Microservices",
					InterviewType:   "Technical",
					Language:        "English",
					IsConsent:       true,
				}
			},
			setup: func() (*services.MockInterviewSessionService, *utils.MockValidator, *middleware.MockAuthContext, websocket.WebSocketServerInterface) {
				mockInterviewSessionService := new(services.MockInterviewSessionService)
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)
				wsServer := new(ws.MockWebSocketServerInterface)

				mockValidator.EXPECT().
					ValidateAndBind(mock.Anything, mock.Anything, "CreateInterviewSessionWithExistingResume").
					Return(nil)

				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, nil)

				mockInterviewSessionService.EXPECT().
					CreateInterviewSessionWithExistingResume(ctx, mock.Anything).
					Return(&entities.CreateInterviewSessionWithExistingResumeResp{
						SessionToken: "session-token-456",
					}, nil)

				return mockInterviewSessionService, mockValidator, mockAuthContext, wsServer
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusCreated, w.Code)
				assert.JSONEq(t, `{"session_token":"session-token-456"}`, w.Body.String())
			},
		},
		{
			name: "Error With Validation",
			input: func() *entities.CreateInterviewSessionWithExistingResumeReq {
				return &entities.CreateInterviewSessionWithExistingResumeReq{}
			},
			setup: func() (*services.MockInterviewSessionService, *utils.MockValidator, *middleware.MockAuthContext, websocket.WebSocketServerInterface) {
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)
				mockInterviewSessionService := new(services.MockInterviewSessionService)
				wsServer := new(ws.MockWebSocketServerInterface)

				validationErr := app_error.New(err, app_error.ErrCodeAuthInvalidRequest)
				mockValidator.EXPECT().
					ValidateAndBind(mock.Anything, mock.Anything, "CreateInterviewSessionWithExistingResume").
					Return(validationErr)

				return mockInterviewSessionService, mockValidator, mockAuthContext, wsServer
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, w.Code)
				assert.Contains(t, w.Body.String(), "Something went wrong with the request")
			},
		},
		{
			name: "Error With Auth Context",
			input: func() *entities.CreateInterviewSessionWithExistingResumeReq {
				return &entities.CreateInterviewSessionWithExistingResumeReq{}
			},
			setup: func() (*services.MockInterviewSessionService, *utils.MockValidator, *middleware.MockAuthContext, websocket.WebSocketServerInterface) {
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)
				mockInterviewSessionService := new(services.MockInterviewSessionService)
				wsServer := new(ws.MockWebSocketServerInterface)

				mockValidator.EXPECT().
					ValidateAndBind(mock.Anything, mock.Anything, "CreateInterviewSessionWithExistingResume").
					Return(nil)

				authErr := app_error.New(err, app_error.ErrCodeAuthInvalidHeader)
				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, authErr)

				return mockInterviewSessionService, mockValidator, mockAuthContext, wsServer
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusUnauthorized, w.Code)
				assert.Contains(t, w.Body.String(), "Please log in to continue")
			},
		},
		{
			name: "Error With Service",
			input: func() *entities.CreateInterviewSessionWithExistingResumeReq {
				return &entities.CreateInterviewSessionWithExistingResumeReq{}
			},
			setup: func() (*services.MockInterviewSessionService, *utils.MockValidator, *middleware.MockAuthContext, websocket.WebSocketServerInterface) {
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)
				mockInterviewSessionService := new(services.MockInterviewSessionService)
				wsServer := new(ws.MockWebSocketServerInterface)

				mockValidator.EXPECT().
					ValidateAndBind(mock.Anything, mock.Anything, "CreateInterviewSessionWithExistingResume").
					Return(nil)

				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, nil)

				serviceErr := app_error.New(err, app_error.ErrCodeResumeNotFound)
				mockInterviewSessionService.EXPECT().
					CreateInterviewSessionWithExistingResume(ctx, mock.Anything).
					Return(nil, serviceErr)

				return mockInterviewSessionService, mockValidator, mockAuthContext, wsServer
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, w.Code)
				assert.Contains(t, w.Body.String(), "The resume was not found")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.input())
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPost, "/interview-sessions/existing-resume", bytes.NewBuffer(body))
			c.Request.Header.Set("Content-Type", "application/json")

			mockInterviewSessionService, mockValidator, mockAuthContext, wsServer := tt.setup()
			defer mockInterviewSessionService.AssertExpectations(t)
			defer mockValidator.AssertExpectations(t)
			defer mockAuthContext.AssertExpectations(t)

			handler := NewInterviewSessionHandler(mockInterviewSessionService, log, mockAuthContext, mockValidator, wsServer)
			handler.CreateInterviewSessionWithExistingResume(c)

			tt.verify(t, w)
		})
	}
}

func TestInterviewSessionHandler_OpenWsConnection(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log := log.Initialize("test")
	ctx := context.Background()
	err := errors.New("mock error")

	tests := []struct {
		name   string
		input  string
		body   func() *entities.OpenWsConnectionRequest
		setup  func() (*services.MockInterviewSessionService, *utils.MockValidator, *middleware.MockAuthContext, websocket.WebSocketServerInterface)
		verify func(t *testing.T, w *httptest.ResponseRecorder)
	}{
		{
			name:  "Success",
			input: "session-token-123",
			body: func() *entities.OpenWsConnectionRequest {
				return &entities.OpenWsConnectionRequest{
					SessionToken: "session-token-123",
				}
			},
			setup: func() (*services.MockInterviewSessionService, *utils.MockValidator, *middleware.MockAuthContext, websocket.WebSocketServerInterface) {
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)
				mockInterviewSessionService := new(services.MockInterviewSessionService)
				mockWsServer := new(ws.MockWebSocketServerInterface)

				mockValidator.EXPECT().
					ValidateAndBind(mock.Anything, mock.Anything, "OpenWsConnection").
					Return(nil)

				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, nil)

				mockWsServer.EXPECT().
					HandleConnection(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)

				return mockInterviewSessionService, mockValidator, mockAuthContext, mockWsServer
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, w.Code)
			},
		},
		{
			name:  "Error With Empty Session Token",
			input: "",
			body: func() *entities.OpenWsConnectionRequest {
				return &entities.OpenWsConnectionRequest{}
			},
			setup: func() (*services.MockInterviewSessionService, *utils.MockValidator, *middleware.MockAuthContext, websocket.WebSocketServerInterface) {
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)
				mockInterviewSessionService := new(services.MockInterviewSessionService)
				wsServer := new(ws.MockWebSocketServerInterface)

				return mockInterviewSessionService, mockValidator, mockAuthContext, wsServer
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, w.Code)
				assert.Contains(t, w.Body.String(), "The session token is invalid")
			},
		},
		{
			name:  "Error With Invalid JSON Body",
			input: "session-token-123",
			body: func() *entities.OpenWsConnectionRequest {
				return &entities.OpenWsConnectionRequest{}
			},
			setup: func() (*services.MockInterviewSessionService, *utils.MockValidator, *middleware.MockAuthContext, websocket.WebSocketServerInterface) {
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)
				mockInterviewSessionService := new(services.MockInterviewSessionService)
				wsServer := new(ws.MockWebSocketServerInterface)

				// Mock the validator to return an error for invalid JSON
				validationErr := app_error.New(err, app_error.ErrCodeAuthInvalidRequest)
				mockValidator.EXPECT().
					ValidateAndBind(mock.Anything, mock.Anything, "OpenWsConnection").
					Return(validationErr)

				return mockInterviewSessionService, mockValidator, mockAuthContext, wsServer
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, w.Code)
				assert.Contains(t, w.Body.String(), "Something went wrong with the request")
			},
		},
		{
			name:  "Error With Auth Context",
			input: "session-token-123",
			body: func() *entities.OpenWsConnectionRequest {
				return &entities.OpenWsConnectionRequest{
					SessionToken: "session-token-123",
				}
			},
			setup: func() (*services.MockInterviewSessionService, *utils.MockValidator, *middleware.MockAuthContext, websocket.WebSocketServerInterface) {
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)
				mockInterviewSessionService := new(services.MockInterviewSessionService)
				wsServer := new(ws.MockWebSocketServerInterface)

				mockValidator.EXPECT().
					ValidateAndBind(mock.Anything, mock.Anything, "OpenWsConnection").
					Return(nil)

				authErr := app_error.New(err, app_error.ErrCodeAuthInvalidHeader)
				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, authErr)

				return mockInterviewSessionService, mockValidator, mockAuthContext, wsServer
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusUnauthorized, w.Code)
				assert.Contains(t, w.Body.String(), "Please log in to continue")
			},
		},
		{
			name:  "Error With Websocket",
			input: "session-token-123",
			body: func() *entities.OpenWsConnectionRequest {
				return &entities.OpenWsConnectionRequest{
					SessionToken: "session-token-123",
				}
			},
			setup: func() (*services.MockInterviewSessionService, *utils.MockValidator, *middleware.MockAuthContext, websocket.WebSocketServerInterface) {
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)
				mockInterviewSessionService := new(services.MockInterviewSessionService)
				mockWsServer := new(ws.MockWebSocketServerInterface)

				mockValidator.EXPECT().
					ValidateAndBind(mock.Anything, mock.Anything, "OpenWsConnection").
					Return(nil)

				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, nil)

				mockWsServer.EXPECT().
					HandleConnection(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(err)

				return mockInterviewSessionService, mockValidator, mockAuthContext, mockWsServer
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, w.Code)
				assert.Contains(t, w.Body.String(), "Something went wrong with the request")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.body())
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPost, "/interview-sessions/"+tt.input+"/ws", bytes.NewBuffer(body))
			c.Params = gin.Params{{Key: "id", Value: tt.input}}
			c.Request.Header.Set("Content-Type", "application/json")

			mockInterviewSessionService, mockValidator, mockAuthContext, wsServer := tt.setup()
			defer mockInterviewSessionService.AssertExpectations(t)
			defer mockValidator.AssertExpectations(t)
			defer mockAuthContext.AssertExpectations(t)

			handler := NewInterviewSessionHandler(mockInterviewSessionService, log, mockAuthContext, mockValidator, wsServer)
			handler.OpenWsConnection(c)

			tt.verify(t, w)
		})
	}
}
