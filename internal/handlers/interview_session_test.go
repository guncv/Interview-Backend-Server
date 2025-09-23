package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
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
	utilsPkg "gitlab.com/interview-simulation/interview-backend-server/internal/utils"
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
					Position:  "Software Engineer",
					IsConsent: true,
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

			handler := NewInterviewSessionHandler(mockInterviewSessionService, log, mockAuthContext, mockValidator, wsServer, nil, nil)
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
					ResumeID:  "123e4567-e89b-12d3-a456-426614174000",
					Position:  "Software Engineer",
					IsConsent: true,
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

			handler := NewInterviewSessionHandler(mockInterviewSessionService, log, mockAuthContext, mockValidator, wsServer, nil, nil)
			handler.CreateInterviewSessionWithExistingResume(c)

			tt.verify(t, w)
		})
	}
}

func TestInterviewSessionHandler_OpenWsConnection(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log := log.Initialize("test")
	err := errors.New("mock error")

	tests := []struct {
		name           string
		sessionToken   string
		accessToken    string
		setup          func() (*utils.MockValidator, websocket.WebSocketServerInterface, *middleware.MockAuthMiddleware, *services.MockInterviewSessionService)
		verify         func(t *testing.T, w *httptest.ResponseRecorder)
		expectedStatus int
	}{
		{
			name:         "Success",
			sessionToken: "123e4567-e89b-12d3-a456-426614174000",
			accessToken:  "valid_access_token",
			setup: func() (*utils.MockValidator, websocket.WebSocketServerInterface, *middleware.MockAuthMiddleware, *services.MockInterviewSessionService) {
				mockValidator := new(utils.MockValidator)
				mockWsServer := new(ws.MockWebSocketServerInterface)
				mockAuthMiddleware := new(middleware.MockAuthMiddleware)
				mockInterviewSessionService := new(services.MockInterviewSessionService)
				// Create a real validator instance for UUID and required validation
				realValidator := validator.New()

				// Mock UUID validation (session token) - should pass
				mockValidator.EXPECT().
					GetValidate().
					Return(realValidator)

				// Mock required validation (access token) - should pass
				mockValidator.EXPECT().
					GetValidate().
					Return(realValidator)

				// Mock JWT token verification
				mockAuthMiddleware.EXPECT().
					VerifyAndRenewAccessToken(mock.Anything, "valid_access_token").
					Return(&utilsPkg.SignInTokenPayload{
						ID:     uuid.New(),
						UserID: "user-123",
						Role:   "user",
					}, nil)

				mockInterviewSessionService.EXPECT().
					IsSessionValid(mock.Anything, mock.Anything).
					Return(&entities.IsSessionValidResp{
						UserID:    "user-123",
						SessionID: "session-123",
					}, nil)

				// Mock WebSocket connection handling
				mockWsServer.EXPECT().
					HandleConnection(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)

				return mockValidator, mockWsServer, mockAuthMiddleware, mockInterviewSessionService
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, w.Code)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:         "Error With Empty Session Token",
			sessionToken: "",
			accessToken:  "valid_access_token",
			setup: func() (*utils.MockValidator, websocket.WebSocketServerInterface, *middleware.MockAuthMiddleware, *services.MockInterviewSessionService) {
				mockValidator := new(utils.MockValidator)
				mockWsServer := new(ws.MockWebSocketServerInterface)
				mockAuthMiddleware := new(middleware.MockAuthMiddleware)
				mockInterviewSessionService := new(services.MockInterviewSessionService)
				realValidator := validator.New()

				// Mock UUID validation (session token) - should fail for empty string
				mockValidator.EXPECT().
					GetValidate().
					Return(realValidator)

				return mockValidator, mockWsServer, mockAuthMiddleware, mockInterviewSessionService
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, w.Code)
				assert.Contains(t, w.Body.String(), "The session token is invalid. Please try again.")
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:         "Error With Invalid Session Token Format",
			sessionToken: "invalid-uuid",
			accessToken:  "valid_access_token",
			setup: func() (*utils.MockValidator, websocket.WebSocketServerInterface, *middleware.MockAuthMiddleware, *services.MockInterviewSessionService) {
				mockValidator := new(utils.MockValidator)
				mockWsServer := new(ws.MockWebSocketServerInterface)
				mockAuthMiddleware := new(middleware.MockAuthMiddleware)
				mockInterviewSessionService := new(services.MockInterviewSessionService)
				realValidator := validator.New()

				mockValidator.EXPECT().
					GetValidate().
					Return(realValidator)

				return mockValidator, mockWsServer, mockAuthMiddleware, mockInterviewSessionService
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, w.Code)
				assert.Contains(t, w.Body.String(), "The session token is invalid. Please try again.")
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:         "Error With Missing Access Token",
			sessionToken: "123e4567-e89b-12d3-a456-426614174000",
			accessToken:  "",
			setup: func() (*utils.MockValidator, websocket.WebSocketServerInterface, *middleware.MockAuthMiddleware, *services.MockInterviewSessionService) {
				mockValidator := new(utils.MockValidator)
				mockWsServer := new(ws.MockWebSocketServerInterface)
				mockAuthMiddleware := new(middleware.MockAuthMiddleware)
				mockInterviewSessionService := new(services.MockInterviewSessionService)
				realValidator := validator.New()

				// Mock UUID validation (session token) - should pass
				mockValidator.EXPECT().
					GetValidate().
					Return(realValidator)

				mockValidator.EXPECT().
					GetValidate().
					Return(realValidator)

				return mockValidator, mockWsServer, mockAuthMiddleware, mockInterviewSessionService
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusUnauthorized, w.Code)
				assert.Contains(t, w.Body.String(), "Please log in to continue")
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:         "Error With Invalid Access Token",
			sessionToken: "123e4567-e89b-12d3-a456-426614174000",
			accessToken:  "invalid_token",
			setup: func() (*utils.MockValidator, websocket.WebSocketServerInterface, *middleware.MockAuthMiddleware, *services.MockInterviewSessionService) {
				mockValidator := new(utils.MockValidator)
				mockWsServer := new(ws.MockWebSocketServerInterface)
				mockAuthMiddleware := new(middleware.MockAuthMiddleware)
				mockInterviewSessionService := new(services.MockInterviewSessionService)
				realValidator := validator.New()

				mockValidator.EXPECT().
					GetValidate().
					Return(realValidator)

				mockValidator.EXPECT().
					GetValidate().
					Return(realValidator)

				mockAuthMiddleware.EXPECT().
					VerifyAndRenewAccessToken(mock.Anything, "invalid_token").
					Return(nil, app_error.New(err, app_error.ErrCodeAuthInvalidToken))

				return mockValidator, mockWsServer, mockAuthMiddleware, mockInterviewSessionService
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusUnauthorized, w.Code)
				assert.Contains(t, w.Body.String(), "Your token is invalid. Please log in again.")
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:         "Error With Expired Access Token And Missing Refresh Token",
			sessionToken: "123e4567-e89b-12d3-a456-426614174000",
			accessToken:  "expired_token",
			setup: func() (*utils.MockValidator, websocket.WebSocketServerInterface, *middleware.MockAuthMiddleware, *services.MockInterviewSessionService) {
				mockValidator := new(utils.MockValidator)
				mockWsServer := new(ws.MockWebSocketServerInterface)
				mockAuthMiddleware := new(middleware.MockAuthMiddleware)
				mockInterviewSessionService := new(services.MockInterviewSessionService)
				// Create a real validator instance
				realValidator := validator.New()

				// Mock UUID validation (session token) - should pass
				mockValidator.EXPECT().
					GetValidate().
					Return(realValidator)

				// Mock required validation (access token) - should pass for non-empty string
				mockValidator.EXPECT().
					GetValidate().
					Return(realValidator)

				// Mock JWT token verification failure for expired token
				mockAuthMiddleware.EXPECT().
					VerifyAndRenewAccessToken(mock.Anything, "expired_token").
					Return(nil, app_error.New(err, app_error.ErrCodeAuthExpiredToken))

				return mockValidator, mockWsServer, mockAuthMiddleware, mockInterviewSessionService
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusUnauthorized, w.Code)
				assert.Contains(t, w.Body.String(), "Your token has expired. Please log in again.")
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:         "Error With VerifyAndRenewAccessToken",
			sessionToken: "123e4567-e89b-12d3-a456-426614174000",
			accessToken:  "valid_access_token",
			setup: func() (*utils.MockValidator, websocket.WebSocketServerInterface, *middleware.MockAuthMiddleware, *services.MockInterviewSessionService) {
				mockValidator := new(utils.MockValidator)
				mockWsServer := new(ws.MockWebSocketServerInterface)
				mockAuthMiddleware := new(middleware.MockAuthMiddleware)
				mockInterviewSessionService := new(services.MockInterviewSessionService)
				// Create a real validator instance
				realValidator := validator.New()

				// Mock UUID validation (session token) - should pass
				mockValidator.EXPECT().
					GetValidate().
					Return(realValidator)

				// Mock required validation (access token) - should pass
				mockValidator.EXPECT().
					GetValidate().
					Return(realValidator)

				// Mock JWT token verification
				mockAuthMiddleware.EXPECT().
					VerifyAndRenewAccessToken(mock.Anything, "valid_access_token").
					Return(nil, app_error.New(err, app_error.ErrCodeAuthExpiredToken))

				return mockValidator, mockWsServer, mockAuthMiddleware, mockInterviewSessionService
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusUnauthorized, w.Code)
				assert.Contains(t, w.Body.String(), "Your token has expired. Please log in again.")
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:         "Error With IsSessionValid",
			sessionToken: "123e4567-e89b-12d3-a456-426614174000",
			accessToken:  "valid_access_token",
			setup: func() (*utils.MockValidator, websocket.WebSocketServerInterface, *middleware.MockAuthMiddleware, *services.MockInterviewSessionService) {
				mockValidator := new(utils.MockValidator)
				mockWsServer := new(ws.MockWebSocketServerInterface)
				mockAuthMiddleware := new(middleware.MockAuthMiddleware)
				mockInterviewSessionService := new(services.MockInterviewSessionService)
				// Create a real validator instance
				realValidator := validator.New()

				// Mock UUID validation (session token) - should pass
				mockValidator.EXPECT().
					GetValidate().
					Return(realValidator)

				// Mock required validation (access token) - should pass
				mockValidator.EXPECT().
					GetValidate().
					Return(realValidator)

				// Mock JWT token verification
				mockAuthMiddleware.EXPECT().
					VerifyAndRenewAccessToken(mock.Anything, "valid_access_token").
					Return(&utilsPkg.SignInTokenPayload{
						ID:     uuid.New(),
						UserID: "user-123",
						Role:   "user",
					}, nil)

				mockInterviewSessionService.EXPECT().
					IsSessionValid(mock.Anything, mock.Anything).
					Return(nil, err)

				return mockValidator, mockWsServer, mockAuthMiddleware, mockInterviewSessionService
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Error(t, err)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:         "Error With WebSocket Connection",
			sessionToken: "123e4567-e89b-12d3-a456-426614174000",
			accessToken:  "valid_access_token",
			setup: func() (*utils.MockValidator, websocket.WebSocketServerInterface, *middleware.MockAuthMiddleware, *services.MockInterviewSessionService) {
				mockValidator := new(utils.MockValidator)
				mockWsServer := new(ws.MockWebSocketServerInterface)
				mockAuthMiddleware := new(middleware.MockAuthMiddleware)
				mockInterviewSessionService := new(services.MockInterviewSessionService)
				// Create a real validator instance
				realValidator := validator.New()

				// Mock UUID validation (session token) - should pass
				mockValidator.EXPECT().
					GetValidate().
					Return(realValidator)

				// Mock required validation (access token) - should pass
				mockValidator.EXPECT().
					GetValidate().
					Return(realValidator)

				// Mock JWT token verification
				mockAuthMiddleware.EXPECT().
					VerifyAndRenewAccessToken(mock.Anything, "valid_access_token").
					Return(&utilsPkg.SignInTokenPayload{
						ID:     uuid.New(),
						UserID: "user-123",
						Role:   "user",
					}, nil)

				mockInterviewSessionService.EXPECT().
					IsSessionValid(mock.Anything, mock.Anything).
					Return(&entities.IsSessionValidResp{
						UserID:    "user-123",
						SessionID: "session-123",
					}, nil)

				// Mock WebSocket connection handling failure
				mockWsServer.EXPECT().
					HandleConnection(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(err)

				return mockValidator, mockWsServer, mockAuthMiddleware, mockInterviewSessionService
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, w.Code)
				assert.Contains(t, w.Body.String(), "Something went wrong with the request")
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			// Set up the request with query parameters and path parameters
			url := fmt.Sprintf("/api/v1/ws/connect/%s", tt.sessionToken)
			if tt.accessToken != "" {
				url += "?access_token=" + tt.accessToken
			}

			c.Request = httptest.NewRequest(http.MethodGet, url, nil)
			c.Params = gin.Params{{Key: "id", Value: tt.sessionToken}}

			mockValidator, mockWsServer, mockAuthMiddleware, mockInterviewSessionService := tt.setup()
			defer mockValidator.AssertExpectations(t)
			defer mockAuthMiddleware.AssertExpectations(t)
			defer mockInterviewSessionService.AssertExpectations(t)

			handler := NewInterviewSessionHandler(mockInterviewSessionService, log, nil, mockValidator, mockWsServer, nil, mockAuthMiddleware)
			handler.OpenWsConnection(c)

			tt.verify(t, w)
		})
	}
}

func TestInterviewSessionHandler_GetChatHistoryBySessionToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log := log.Initialize("test")
	ctx := context.Background()
	err := errors.New("mock error")

	tests := []struct {
		name           string
		sessionToken   string
		setup          func() (*utils.MockValidator, *middleware.MockAuthContext, *services.MockInterviewSessionService)
		verify         func(t *testing.T, w *httptest.ResponseRecorder)
		expectedStatus int
	}{
		{
			name:         "Success",
			sessionToken: "123e4567-e89b-12d3-a456-426614174000",
			setup: func() (*utils.MockValidator, *middleware.MockAuthContext, *services.MockInterviewSessionService) {
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)
				mockInterviewSessionService := new(services.MockInterviewSessionService)

				realValidator := validator.New()

				mockValidator.EXPECT().
					GetValidate().
					Return(realValidator)

				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, nil)

				mockInterviewSessionService.EXPECT().
					GetChatHistoryBySessionToken(mock.Anything, mock.Anything).
					Return(&entities.GetChatHistoryBySessionTokenResp{
						ChatHistory: []entities.ChatHistory{
							{
								ID:             uuid.New(),
								TurnNo:         1,
								Actor:          "user",
								TranscriptText: "Hello, how are you?",
							},
						},
					}, nil)

				return mockValidator, mockAuthContext, mockInterviewSessionService
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, w.Code)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:         "Error - WithNonSessionTokenParam",
			sessionToken: "",
			setup: func() (*utils.MockValidator, *middleware.MockAuthContext, *services.MockInterviewSessionService) {
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)
				mockInterviewSessionService := new(services.MockInterviewSessionService)

				return mockValidator, mockAuthContext, mockInterviewSessionService
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, w.Code)
				assert.Contains(t, w.Body.String(), "The session token is invalid. Please try again.")
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:         "Error With Validation",
			sessionToken: "invalid-session-token",
			setup: func() (*utils.MockValidator, *middleware.MockAuthContext, *services.MockInterviewSessionService) {
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)
				mockInterviewSessionService := new(services.MockInterviewSessionService)

				realValidator := validator.New()

				mockValidator.EXPECT().
					GetValidate().
					Return(realValidator)

				return mockValidator, mockAuthContext, mockInterviewSessionService
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, w.Code)
				assert.Contains(t, w.Body.String(), "The session token is invalid. Please try again.")
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:         "Error - WithExtractAuthContextError",
			sessionToken: "123e4567-e89b-12d3-a456-426614174000",
			setup: func() (*utils.MockValidator, *middleware.MockAuthContext, *services.MockInterviewSessionService) {
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)
				mockInterviewSessionService := new(services.MockInterviewSessionService)

				realValidator := validator.New()

				mockValidator.EXPECT().
					GetValidate().
					Return(realValidator)

				authErr := app_error.New(errors.New("extract auth context error"), app_error.ErrCodeAuthInvalidHeader)
				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, authErr)

				return mockValidator, mockAuthContext, mockInterviewSessionService
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusUnauthorized, w.Code)
				assert.Contains(t, w.Body.String(), "Please log in to continue")
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:         "Error - WithGetChatHistoryBySessionTokenError",
			sessionToken: "123e4567-e89b-12d3-a456-426614174000",
			setup: func() (*utils.MockValidator, *middleware.MockAuthContext, *services.MockInterviewSessionService) {
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)
				mockInterviewSessionService := new(services.MockInterviewSessionService)

				realValidator := validator.New()

				mockValidator.EXPECT().
					GetValidate().
					Return(realValidator)

				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, nil)

				serviceErr := app_error.New(err, app_error.ErrCodeSessionNotFound)
				mockInterviewSessionService.EXPECT().
					GetChatHistoryBySessionToken(mock.Anything, mock.Anything).
					Return(nil, serviceErr)

				return mockValidator, mockAuthContext, mockInterviewSessionService
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, w.Code)
				assert.Contains(t, w.Body.String(), "The session was not found")
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			url := fmt.Sprintf("/api/v1/sessions/chat-history/%s", tt.sessionToken)

			c.Request = httptest.NewRequest(http.MethodGet, url, nil)
			c.Params = gin.Params{{Key: "session_token", Value: tt.sessionToken}}

			mockValidator, mockAuthContext, mockInterviewSessionService := tt.setup()
			defer mockValidator.AssertExpectations(t)
			defer mockAuthContext.AssertExpectations(t)
			defer mockInterviewSessionService.AssertExpectations(t)

			handler := NewInterviewSessionHandler(mockInterviewSessionService, log, mockAuthContext, mockValidator, nil, nil, nil)
			handler.GetChatHistoryBySessionToken(c)

			tt.verify(t, w)
		})
	}
}

func TestInterviewSessionHandler_GetInterviewSessionInformation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log := log.Initialize("test")
	ctx := context.Background()
	err := errors.New("mock error")

	tests := []struct {
		name           string
		sessionToken   string
		setup          func() (*utils.MockValidator, *middleware.MockAuthContext, *services.MockInterviewSessionService)
		verify         func(t *testing.T, w *httptest.ResponseRecorder)
		expectedStatus int
	}{
		{
			name:         "Success",
			sessionToken: "123e4567-e89b-12d3-a456-426614174000",
			setup: func() (*utils.MockValidator, *middleware.MockAuthContext, *services.MockInterviewSessionService) {
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)
				mockInterviewSessionService := new(services.MockInterviewSessionService)

				realValidator := validator.New()

				mockValidator.EXPECT().
					GetValidate().
					Return(realValidator)

				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, nil)

				mockInterviewSessionService.EXPECT().
					GetInterviewSessionInformation(mock.Anything, mock.Anything).
					Return(&entities.GetInterviewSessionInformationResp{
						Position: "position",
					}, nil)

				return mockValidator, mockAuthContext, mockInterviewSessionService
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, w.Code)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:         "Error - WithNonSessionTokenParam",
			sessionToken: "",
			setup: func() (*utils.MockValidator, *middleware.MockAuthContext, *services.MockInterviewSessionService) {
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)
				mockInterviewSessionService := new(services.MockInterviewSessionService)

				return mockValidator, mockAuthContext, mockInterviewSessionService
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, w.Code)
				assert.Contains(t, w.Body.String(), "The session token is invalid. Please try again.")
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:         "Error With Validation",
			sessionToken: "invalid-session-token",
			setup: func() (*utils.MockValidator, *middleware.MockAuthContext, *services.MockInterviewSessionService) {
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)
				mockInterviewSessionService := new(services.MockInterviewSessionService)

				realValidator := validator.New()

				mockValidator.EXPECT().
					GetValidate().
					Return(realValidator)

				return mockValidator, mockAuthContext, mockInterviewSessionService
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, w.Code)
				assert.Contains(t, w.Body.String(), "The session token is invalid. Please try again.")
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:         "Error - WithExtractAuthContextError",
			sessionToken: "123e4567-e89b-12d3-a456-426614174000",
			setup: func() (*utils.MockValidator, *middleware.MockAuthContext, *services.MockInterviewSessionService) {
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)
				mockInterviewSessionService := new(services.MockInterviewSessionService)

				realValidator := validator.New()

				mockValidator.EXPECT().
					GetValidate().
					Return(realValidator)

				authErr := app_error.New(errors.New("extract auth context error"), app_error.ErrCodeAuthInvalidHeader)
				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, authErr)

				return mockValidator, mockAuthContext, mockInterviewSessionService
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusUnauthorized, w.Code)
				assert.Contains(t, w.Body.String(), "Please log in to continue")
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:         "Error - WithGetInterviewSessionInformationError",
			sessionToken: "123e4567-e89b-12d3-a456-426614174000",
			setup: func() (*utils.MockValidator, *middleware.MockAuthContext, *services.MockInterviewSessionService) {
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)
				mockInterviewSessionService := new(services.MockInterviewSessionService)

				realValidator := validator.New()

				mockValidator.EXPECT().
					GetValidate().
					Return(realValidator)

				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, nil)

				serviceErr := app_error.New(err, app_error.ErrCodeSessionNotFound)
				mockInterviewSessionService.EXPECT().
					GetInterviewSessionInformation(mock.Anything, mock.Anything).
					Return(nil, serviceErr)

				return mockValidator, mockAuthContext, mockInterviewSessionService
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, w.Code)
				assert.Contains(t, w.Body.String(), "The session was not found")
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			url := fmt.Sprintf("/api/v1/sessions/information/%s", tt.sessionToken)

			c.Request = httptest.NewRequest(http.MethodGet, url, nil)
			c.Params = gin.Params{{Key: "session_token", Value: tt.sessionToken}}

			mockValidator, mockAuthContext, mockInterviewSessionService := tt.setup()
			defer mockValidator.AssertExpectations(t)
			defer mockAuthContext.AssertExpectations(t)
			defer mockInterviewSessionService.AssertExpectations(t)

			handler := NewInterviewSessionHandler(mockInterviewSessionService, log, mockAuthContext, mockValidator, nil, nil, nil)
			handler.GetInterviewSessionInformation(c)

			tt.verify(t, w)
		})
	}
}

func TestInterviewSessionHandler_ListInterviewSessionsByUserIDWithCursor(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log := log.Initialize("test")
	ctx := context.Background()

	validResp := &entities.ListInterviewSessionsByUserIDResp{
		Sessions: []entities.InterviewSessionSummary{
			{
				ID:        uuid.New().String(),
				Position:  "position",
				Status:    "status",
				CreatedAt: time.Now().Format(time.RFC3339),
			},
		},
		PrevCursor: &entities.Cursor{
			ID:        uuid.New().String(),
			CreatedAt: time.Now().Format(time.RFC3339),
		},
		NextCursor: &entities.Cursor{
			ID:        uuid.New().String(),
			CreatedAt: time.Now().Format(time.RFC3339),
		},
		TotalPages: 1,
		PageSize:   1,
	}

	authError := app_error.New(errors.New("auth error"), app_error.ErrCodeAuthInvalidToken)
	serviceError := app_error.New(errors.New("service error"), app_error.ErrCodeGeneralServerUnavailable)

	tests := []struct {
		name           string
		queryParams    map[string]string
		setup          func() (*middleware.MockAuthContext, *services.MockInterviewSessionService)
		verify         func(t *testing.T, w *httptest.ResponseRecorder)
		expectedStatus int
	}{
		{
			name:        "Success - NoQueryParameters",
			queryParams: map[string]string{},
			setup: func() (*middleware.MockAuthContext, *services.MockInterviewSessionService) {
				mockAuthContext := new(middleware.MockAuthContext)
				mockInterviewSessionService := new(services.MockInterviewSessionService)

				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, nil)

				mockInterviewSessionService.EXPECT().
					ListInterviewSessionsByUserIDWithCursor(mock.Anything, mock.MatchedBy(func(req *entities.ListInterviewSessionsByUserIDWithCursorReq) bool {
						return req.SearchText == nil && req.Status == nil && req.Limit == nil && req.Cursor == nil
					})).
					Return(validResp, nil)

				return mockAuthContext, mockInterviewSessionService
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, w.Code)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "Success - WithSearchTextOnly",
			queryParams: map[string]string{"search_text": "software engineer"},
			setup: func() (*middleware.MockAuthContext, *services.MockInterviewSessionService) {
				mockAuthContext := new(middleware.MockAuthContext)
				mockInterviewSessionService := new(services.MockInterviewSessionService)

				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, nil)

				mockInterviewSessionService.EXPECT().
					ListInterviewSessionsByUserIDWithCursor(mock.Anything, mock.MatchedBy(func(req *entities.ListInterviewSessionsByUserIDWithCursorReq) bool {
						return req.SearchText != nil && *req.SearchText == "software engineer" && req.Status == nil && req.Limit == nil && req.Cursor == nil
					})).
					Return(validResp, nil)

				return mockAuthContext, mockInterviewSessionService
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, w.Code)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "Success - WithStatusOnly",
			queryParams: map[string]string{"status": "completed"},
			setup: func() (*middleware.MockAuthContext, *services.MockInterviewSessionService) {
				mockAuthContext := new(middleware.MockAuthContext)
				mockInterviewSessionService := new(services.MockInterviewSessionService)

				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, nil)

				mockInterviewSessionService.EXPECT().
					ListInterviewSessionsByUserIDWithCursor(mock.Anything, mock.MatchedBy(func(req *entities.ListInterviewSessionsByUserIDWithCursorReq) bool {
						return req.SearchText == nil && req.Status != nil && *req.Status == "completed" && req.Limit == nil && req.Cursor == nil
					})).
					Return(validResp, nil)

				return mockAuthContext, mockInterviewSessionService
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, w.Code)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "Success - WithLimitOnly",
			queryParams: map[string]string{"limit": "10"},
			setup: func() (*middleware.MockAuthContext, *services.MockInterviewSessionService) {
				mockAuthContext := new(middleware.MockAuthContext)
				mockInterviewSessionService := new(services.MockInterviewSessionService)

				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, nil)

				mockInterviewSessionService.EXPECT().
					ListInterviewSessionsByUserIDWithCursor(mock.Anything, mock.MatchedBy(func(req *entities.ListInterviewSessionsByUserIDWithCursorReq) bool {
						return req.SearchText == nil && req.Status == nil && req.Limit != nil && *req.Limit == 10 && req.Cursor == nil
					})).
					Return(validResp, nil)

				return mockAuthContext, mockInterviewSessionService
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, w.Code)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "Success - WithCursorOnly",
			queryParams: map[string]string{"cursor_id": "123", "cursor_created_at": "2023-01-01T00:00:00Z"},
			setup: func() (*middleware.MockAuthContext, *services.MockInterviewSessionService) {
				mockAuthContext := new(middleware.MockAuthContext)
				mockInterviewSessionService := new(services.MockInterviewSessionService)

				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, nil)

				mockInterviewSessionService.EXPECT().
					ListInterviewSessionsByUserIDWithCursor(mock.Anything, mock.MatchedBy(func(req *entities.ListInterviewSessionsByUserIDWithCursorReq) bool {
						return req.SearchText == nil && req.Status == nil && req.Limit == nil && req.Cursor != nil && req.Cursor.ID == "123" && req.Cursor.CreatedAt == "2023-01-01T00:00:00Z"
					})).
					Return(validResp, nil)

				return mockAuthContext, mockInterviewSessionService
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, w.Code)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Success - WithAllParameters",
			queryParams: map[string]string{
				"search_text":       "software engineer",
				"status":            "completed",
				"limit":             "5",
				"cursor_id":         "123",
				"cursor_created_at": "2023-01-01T00:00:00Z",
			},
			setup: func() (*middleware.MockAuthContext, *services.MockInterviewSessionService) {
				mockAuthContext := new(middleware.MockAuthContext)
				mockInterviewSessionService := new(services.MockInterviewSessionService)

				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, nil)

				mockInterviewSessionService.EXPECT().
					ListInterviewSessionsByUserIDWithCursor(mock.Anything, mock.MatchedBy(func(req *entities.ListInterviewSessionsByUserIDWithCursorReq) bool {
						return req.SearchText != nil && *req.SearchText == "software engineer" &&
							req.Status != nil && *req.Status == "completed" &&
							req.Limit != nil && *req.Limit == 5 &&
							req.Cursor != nil && req.Cursor.ID == "123" && req.Cursor.CreatedAt == "2023-01-01T00:00:00Z"
					})).
					Return(validResp, nil)

				return mockAuthContext, mockInterviewSessionService
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, w.Code)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "Success - WithEmptySearchText",
			queryParams: map[string]string{"search_text": ""},
			setup: func() (*middleware.MockAuthContext, *services.MockInterviewSessionService) {
				mockAuthContext := new(middleware.MockAuthContext)
				mockInterviewSessionService := new(services.MockInterviewSessionService)

				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, nil)

				mockInterviewSessionService.EXPECT().
					ListInterviewSessionsByUserIDWithCursor(mock.Anything, mock.MatchedBy(func(req *entities.ListInterviewSessionsByUserIDWithCursorReq) bool {
						return req.SearchText == nil && req.Status == nil && req.Limit == nil && req.Cursor == nil
					})).
					Return(validResp, nil)

				return mockAuthContext, mockInterviewSessionService
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, w.Code)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "Success - WithEmptyStatus",
			queryParams: map[string]string{"status": ""},
			setup: func() (*middleware.MockAuthContext, *services.MockInterviewSessionService) {
				mockAuthContext := new(middleware.MockAuthContext)
				mockInterviewSessionService := new(services.MockInterviewSessionService)

				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, nil)

				mockInterviewSessionService.EXPECT().
					ListInterviewSessionsByUserIDWithCursor(mock.Anything, mock.MatchedBy(func(req *entities.ListInterviewSessionsByUserIDWithCursorReq) bool {
						return req.SearchText == nil && req.Status == nil && req.Limit == nil && req.Cursor == nil
					})).
					Return(validResp, nil)

				return mockAuthContext, mockInterviewSessionService
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, w.Code)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "Success - WithZeroLimit",
			queryParams: map[string]string{"limit": "0"},
			setup: func() (*middleware.MockAuthContext, *services.MockInterviewSessionService) {
				mockAuthContext := new(middleware.MockAuthContext)
				mockInterviewSessionService := new(services.MockInterviewSessionService)

				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, nil)

				mockInterviewSessionService.EXPECT().
					ListInterviewSessionsByUserIDWithCursor(mock.Anything, mock.MatchedBy(func(req *entities.ListInterviewSessionsByUserIDWithCursorReq) bool {
						return req.SearchText == nil && req.Status == nil && req.Limit == nil && req.Cursor == nil
					})).
					Return(validResp, nil)

				return mockAuthContext, mockInterviewSessionService
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, w.Code)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "Success - WithNegativeLimit",
			queryParams: map[string]string{"limit": "-5"},
			setup: func() (*middleware.MockAuthContext, *services.MockInterviewSessionService) {
				mockAuthContext := new(middleware.MockAuthContext)
				mockInterviewSessionService := new(services.MockInterviewSessionService)

				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, nil)

				mockInterviewSessionService.EXPECT().
					ListInterviewSessionsByUserIDWithCursor(mock.Anything, mock.MatchedBy(func(req *entities.ListInterviewSessionsByUserIDWithCursorReq) bool {
						return req.SearchText == nil && req.Status == nil && req.Limit == nil && req.Cursor == nil
					})).
					Return(validResp, nil)

				return mockAuthContext, mockInterviewSessionService
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, w.Code)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "Success - WithPartialCursorMissingCursorCreatedAt",
			queryParams: map[string]string{"cursor_id": "123"},
			setup: func() (*middleware.MockAuthContext, *services.MockInterviewSessionService) {
				mockAuthContext := new(middleware.MockAuthContext)
				mockInterviewSessionService := new(services.MockInterviewSessionService)

				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, nil)

				mockInterviewSessionService.EXPECT().
					ListInterviewSessionsByUserIDWithCursor(mock.Anything, mock.MatchedBy(func(req *entities.ListInterviewSessionsByUserIDWithCursorReq) bool {
						return req.SearchText == nil && req.Status == nil && req.Limit == nil && req.Cursor == nil
					})).
					Return(validResp, nil)

				return mockAuthContext, mockInterviewSessionService
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, w.Code)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "Success - WithPartialCursorMissingCursorId",
			queryParams: map[string]string{"cursor_created_at": "2023-01-01T00:00:00Z"},
			setup: func() (*middleware.MockAuthContext, *services.MockInterviewSessionService) {
				mockAuthContext := new(middleware.MockAuthContext)
				mockInterviewSessionService := new(services.MockInterviewSessionService)

				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, nil)

				mockInterviewSessionService.EXPECT().
					ListInterviewSessionsByUserIDWithCursor(mock.Anything, mock.MatchedBy(func(req *entities.ListInterviewSessionsByUserIDWithCursorReq) bool {
						return req.SearchText == nil && req.Status == nil && req.Limit == nil && req.Cursor == nil
					})).
					Return(validResp, nil)

				return mockAuthContext, mockInterviewSessionService
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, w.Code)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "Error - WithInvalidLimitNonNumeric",
			queryParams: map[string]string{"limit": "invalid"},
			setup: func() (*middleware.MockAuthContext, *services.MockInterviewSessionService) {
				mockAuthContext := new(middleware.MockAuthContext)
				mockInterviewSessionService := new(services.MockInterviewSessionService)

				return mockAuthContext, mockInterviewSessionService
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, w.Code)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "Error - WithExtractAuthContextError",
			queryParams: map[string]string{},
			setup: func() (*middleware.MockAuthContext, *services.MockInterviewSessionService) {
				mockAuthContext := new(middleware.MockAuthContext)
				mockInterviewSessionService := new(services.MockInterviewSessionService)

				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, authError)

				return mockAuthContext, mockInterviewSessionService
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusUnauthorized, w.Code)
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:        "Success - WithTypeParameter",
			queryParams: map[string]string{"type": "prev", "cursor_id": "123", "cursor_created_at": "2023-01-01T00:00:00Z"},
			setup: func() (*middleware.MockAuthContext, *services.MockInterviewSessionService) {
				mockAuthContext := new(middleware.MockAuthContext)
				mockInterviewSessionService := new(services.MockInterviewSessionService)

				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, nil)

				mockInterviewSessionService.EXPECT().
					ListInterviewSessionsByUserIDWithCursor(mock.Anything, mock.MatchedBy(func(req *entities.ListInterviewSessionsByUserIDWithCursorReq) bool {
						return req.Type != nil && *req.Type == "prev" &&
							req.Cursor != nil && req.Cursor.ID == "123" && req.Cursor.CreatedAt == "2023-01-01T00:00:00Z"
					})).
					Return(validResp, nil)

				return mockAuthContext, mockInterviewSessionService
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, w.Code)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "Error - WithServiceLayerError",
			queryParams: map[string]string{},
			setup: func() (*middleware.MockAuthContext, *services.MockInterviewSessionService) {
				mockAuthContext := new(middleware.MockAuthContext)
				mockInterviewSessionService := new(services.MockInterviewSessionService)

				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, nil)

				mockInterviewSessionService.EXPECT().
					ListInterviewSessionsByUserIDWithCursor(mock.Anything, mock.Anything).
					Return(nil, serviceError)

				return mockAuthContext, mockInterviewSessionService
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, w.Code)
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			baseURL := "/api/v1/sessions"
			if len(tt.queryParams) > 0 {
				u, err := url.Parse(baseURL)
				if err != nil {
					t.Fatalf("Failed to parse base URL: %v", err)
				}
				q := u.Query()
				for key, value := range tt.queryParams {
					q.Set(key, value)
				}
				u.RawQuery = q.Encode()
				baseURL = u.String()
			}

			c.Request = httptest.NewRequest(http.MethodGet, baseURL, nil)

			mockAuthContext, mockInterviewSessionService := tt.setup()
			defer mockAuthContext.AssertExpectations(t)
			defer mockInterviewSessionService.AssertExpectations(t)

			handler := NewInterviewSessionHandler(mockInterviewSessionService, log, mockAuthContext, nil, nil, nil, nil)
			handler.ListInterviewSessionsByUserIDWithCursor(c)

			tt.verify(t, w)
		})
	}
}

func TestInterviewSessionHandler_ListInterviewSessionsByUserIDWithJumpPagination(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log := log.Initialize("test")
	ctx := context.Background()

	validResp := &entities.ListInterviewSessionsByUserIDResp{
		Sessions: []entities.InterviewSessionSummary{
			{
				ID:        uuid.New().String(),
				Position:  "position",
				Status:    "status",
				CreatedAt: time.Now().Format(time.RFC3339),
			},
		},
		PrevCursor: &entities.Cursor{
			ID:        uuid.New().String(),
			CreatedAt: time.Now().Format(time.RFC3339),
		},
		NextCursor: &entities.Cursor{
			ID:        uuid.New().String(),
			CreatedAt: time.Now().Format(time.RFC3339),
		},
		TotalPages: 1,
		PageSize:   1,
	}

	authError := app_error.New(errors.New("auth error"), app_error.ErrCodeAuthInvalidToken)
	serviceError := app_error.New(errors.New("service error"), app_error.ErrCodeGeneralServerUnavailable)

	tests := []struct {
		name           string
		queryParams    map[string]string
		setup          func() (*middleware.MockAuthContext, *services.MockInterviewSessionService)
		verify         func(t *testing.T, w *httptest.ResponseRecorder)
		expectedStatus int
	}{
		{
			name:        "Success - NoQueryParameters",
			queryParams: map[string]string{},
			setup: func() (*middleware.MockAuthContext, *services.MockInterviewSessionService) {
				mockAuthContext := new(middleware.MockAuthContext)
				mockInterviewSessionService := new(services.MockInterviewSessionService)

				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, nil)

				mockInterviewSessionService.EXPECT().
					ListInterviewSessionsByUserIDWithJumpPagination(mock.Anything, mock.MatchedBy(func(req *entities.ListInterviewSessionsByUserIDWithJumpPaginationReq) bool {
						return req.SearchText == nil && req.Status == nil && req.Limit == nil && req.Offset == nil
					})).
					Return(validResp, nil)

				return mockAuthContext, mockInterviewSessionService
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, w.Code)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "Success - WithSearchTextOnly",
			queryParams: map[string]string{"search_text": "software engineer"},
			setup: func() (*middleware.MockAuthContext, *services.MockInterviewSessionService) {
				mockAuthContext := new(middleware.MockAuthContext)
				mockInterviewSessionService := new(services.MockInterviewSessionService)

				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, nil)

				mockInterviewSessionService.EXPECT().
					ListInterviewSessionsByUserIDWithJumpPagination(mock.Anything, mock.MatchedBy(func(req *entities.ListInterviewSessionsByUserIDWithJumpPaginationReq) bool {
						return req.SearchText != nil && *req.SearchText == "software engineer" && req.Status == nil && req.Limit == nil && req.Offset == nil
					})).
					Return(validResp, nil)

				return mockAuthContext, mockInterviewSessionService
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, w.Code)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "Success - WithStatusOnly",
			queryParams: map[string]string{"status": "completed"},
			setup: func() (*middleware.MockAuthContext, *services.MockInterviewSessionService) {
				mockAuthContext := new(middleware.MockAuthContext)
				mockInterviewSessionService := new(services.MockInterviewSessionService)

				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, nil)

				mockInterviewSessionService.EXPECT().
					ListInterviewSessionsByUserIDWithJumpPagination(mock.Anything, mock.MatchedBy(func(req *entities.ListInterviewSessionsByUserIDWithJumpPaginationReq) bool {
						return req.SearchText == nil && req.Status != nil && *req.Status == "completed" && req.Limit == nil && req.Offset == nil
					})).
					Return(validResp, nil)

				return mockAuthContext, mockInterviewSessionService
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, w.Code)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "Success - WithLimitOnly",
			queryParams: map[string]string{"limit": "10"},
			setup: func() (*middleware.MockAuthContext, *services.MockInterviewSessionService) {
				mockAuthContext := new(middleware.MockAuthContext)
				mockInterviewSessionService := new(services.MockInterviewSessionService)

				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, nil)

				mockInterviewSessionService.EXPECT().
					ListInterviewSessionsByUserIDWithJumpPagination(mock.Anything, mock.MatchedBy(func(req *entities.ListInterviewSessionsByUserIDWithJumpPaginationReq) bool {
						return req.SearchText == nil && req.Status == nil && req.Limit != nil && *req.Limit == 10 && req.Offset == nil
					})).
					Return(validResp, nil)

				return mockAuthContext, mockInterviewSessionService
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, w.Code)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "Success - WithOffsetOnly",
			queryParams: map[string]string{"offset": "3"},
			setup: func() (*middleware.MockAuthContext, *services.MockInterviewSessionService) {
				mockAuthContext := new(middleware.MockAuthContext)
				mockInterviewSessionService := new(services.MockInterviewSessionService)

				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, nil)

				mockInterviewSessionService.EXPECT().
					ListInterviewSessionsByUserIDWithJumpPagination(mock.Anything, mock.MatchedBy(func(req *entities.ListInterviewSessionsByUserIDWithJumpPaginationReq) bool {
						return req.SearchText == nil && req.Status == nil && req.Limit == nil && req.Offset != nil && *req.Offset == 3
					})).
					Return(validResp, nil)

				return mockAuthContext, mockInterviewSessionService
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, w.Code)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Success - WithAllParameters",
			queryParams: map[string]string{
				"search_text":       "software engineer",
				"status":            "completed",
				"limit":             "5",
				"offset":            "3",
				"cursor_created_at": "2023-01-01T00:00:00Z",
			},
			setup: func() (*middleware.MockAuthContext, *services.MockInterviewSessionService) {
				mockAuthContext := new(middleware.MockAuthContext)
				mockInterviewSessionService := new(services.MockInterviewSessionService)

				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, nil)

				mockInterviewSessionService.EXPECT().
					ListInterviewSessionsByUserIDWithJumpPagination(mock.Anything, mock.MatchedBy(func(req *entities.ListInterviewSessionsByUserIDWithJumpPaginationReq) bool {
						return req.SearchText != nil && *req.SearchText == "software engineer" &&
							req.Status != nil && *req.Status == "completed" &&
							req.Limit != nil && *req.Limit == 5 &&
							req.Offset != nil && *req.Offset == 3
					})).
					Return(validResp, nil)

				return mockAuthContext, mockInterviewSessionService
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, w.Code)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "Success - WithEmptySearchText",
			queryParams: map[string]string{"search_text": ""},
			setup: func() (*middleware.MockAuthContext, *services.MockInterviewSessionService) {
				mockAuthContext := new(middleware.MockAuthContext)
				mockInterviewSessionService := new(services.MockInterviewSessionService)

				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, nil)

				mockInterviewSessionService.EXPECT().
					ListInterviewSessionsByUserIDWithJumpPagination(mock.Anything, mock.MatchedBy(func(req *entities.ListInterviewSessionsByUserIDWithJumpPaginationReq) bool {
						return req.SearchText == nil && req.Status == nil && req.Limit == nil && req.Offset == nil
					})).
					Return(validResp, nil)

				return mockAuthContext, mockInterviewSessionService
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, w.Code)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "Success - WithEmptyStatus",
			queryParams: map[string]string{"status": ""},
			setup: func() (*middleware.MockAuthContext, *services.MockInterviewSessionService) {
				mockAuthContext := new(middleware.MockAuthContext)
				mockInterviewSessionService := new(services.MockInterviewSessionService)

				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, nil)

				mockInterviewSessionService.EXPECT().
					ListInterviewSessionsByUserIDWithJumpPagination(mock.Anything, mock.MatchedBy(func(req *entities.ListInterviewSessionsByUserIDWithJumpPaginationReq) bool {
						return req.SearchText == nil && req.Status == nil && req.Limit == nil && req.Offset == nil
					})).
					Return(validResp, nil)

				return mockAuthContext, mockInterviewSessionService
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, w.Code)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "Success - WithZeroLimit",
			queryParams: map[string]string{"limit": "0"},
			setup: func() (*middleware.MockAuthContext, *services.MockInterviewSessionService) {
				mockAuthContext := new(middleware.MockAuthContext)
				mockInterviewSessionService := new(services.MockInterviewSessionService)

				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, nil)

				mockInterviewSessionService.EXPECT().
					ListInterviewSessionsByUserIDWithJumpPagination(mock.Anything, mock.MatchedBy(func(req *entities.ListInterviewSessionsByUserIDWithJumpPaginationReq) bool {
						return req.SearchText == nil && req.Status == nil && req.Limit == nil && req.Offset == nil
					})).
					Return(validResp, nil)

				return mockAuthContext, mockInterviewSessionService
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, w.Code)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "Success - WithNegativeLimit",
			queryParams: map[string]string{"limit": "-5"},
			setup: func() (*middleware.MockAuthContext, *services.MockInterviewSessionService) {
				mockAuthContext := new(middleware.MockAuthContext)
				mockInterviewSessionService := new(services.MockInterviewSessionService)

				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, nil)

				mockInterviewSessionService.EXPECT().
					ListInterviewSessionsByUserIDWithJumpPagination(mock.Anything, mock.MatchedBy(func(req *entities.ListInterviewSessionsByUserIDWithJumpPaginationReq) bool {
						return req.SearchText == nil && req.Status == nil && req.Limit == nil && req.Offset == nil
					})).
					Return(validResp, nil)

				return mockAuthContext, mockInterviewSessionService
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, w.Code)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "Error - WithInvalidLimitNonNumeric",
			queryParams: map[string]string{"limit": "invalid"},
			setup: func() (*middleware.MockAuthContext, *services.MockInterviewSessionService) {
				mockAuthContext := new(middleware.MockAuthContext)
				mockInterviewSessionService := new(services.MockInterviewSessionService)

				return mockAuthContext, mockInterviewSessionService
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, w.Code)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "Success - WithNegativeOffset",
			queryParams: map[string]string{"offset": "-5"},
			setup: func() (*middleware.MockAuthContext, *services.MockInterviewSessionService) {
				mockAuthContext := new(middleware.MockAuthContext)
				mockInterviewSessionService := new(services.MockInterviewSessionService)

				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, nil)

				mockInterviewSessionService.EXPECT().
					ListInterviewSessionsByUserIDWithJumpPagination(mock.Anything, mock.MatchedBy(func(req *entities.ListInterviewSessionsByUserIDWithJumpPaginationReq) bool {
						return req.SearchText == nil && req.Status == nil && req.Limit == nil && req.Offset == nil
					})).
					Return(validResp, nil)

				return mockAuthContext, mockInterviewSessionService
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, w.Code)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "Error - WithInvalidOffsetNonNumeric",
			queryParams: map[string]string{"offset": "invalid"},
			setup: func() (*middleware.MockAuthContext, *services.MockInterviewSessionService) {
				mockAuthContext := new(middleware.MockAuthContext)
				mockInterviewSessionService := new(services.MockInterviewSessionService)

				return mockAuthContext, mockInterviewSessionService
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, w.Code)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "Error - WithExtractAuthContextError",
			queryParams: map[string]string{},
			setup: func() (*middleware.MockAuthContext, *services.MockInterviewSessionService) {
				mockAuthContext := new(middleware.MockAuthContext)
				mockInterviewSessionService := new(services.MockInterviewSessionService)

				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, authError)

				return mockAuthContext, mockInterviewSessionService
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusUnauthorized, w.Code)
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:        "Error - WithServiceLayerError",
			queryParams: map[string]string{},
			setup: func() (*middleware.MockAuthContext, *services.MockInterviewSessionService) {
				mockAuthContext := new(middleware.MockAuthContext)
				mockInterviewSessionService := new(services.MockInterviewSessionService)

				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, nil)

				mockInterviewSessionService.EXPECT().
					ListInterviewSessionsByUserIDWithJumpPagination(mock.Anything, mock.Anything).
					Return(nil, serviceError)

				return mockAuthContext, mockInterviewSessionService
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, w.Code)
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			baseURL := "/api/v1/sessions"
			if len(tt.queryParams) > 0 {
				u, err := url.Parse(baseURL)
				if err != nil {
					t.Fatalf("Failed to parse base URL: %v", err)
				}
				q := u.Query()
				for key, value := range tt.queryParams {
					q.Set(key, value)
				}
				u.RawQuery = q.Encode()
				baseURL = u.String()
			}

			c.Request = httptest.NewRequest(http.MethodGet, baseURL, nil)

			mockAuthContext, mockInterviewSessionService := tt.setup()
			defer mockAuthContext.AssertExpectations(t)
			defer mockInterviewSessionService.AssertExpectations(t)

			handler := NewInterviewSessionHandler(mockInterviewSessionService, log, mockAuthContext, nil, nil, nil, nil)
			handler.ListInterviewSessionsByUserIDWithJumpPagination(c)

			tt.verify(t, w)
		})
	}
}

func TestInterviewSessionHandler_DeleteUserInterviewSessionByID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log := log.Initialize("test")
	ctx := context.Background()
	err := errors.New("mock error")

	tests := []struct {
		name           string
		sessionID      string
		setup          func() (*utils.MockValidator, *middleware.MockAuthContext, *services.MockInterviewSessionService)
		verify         func(t *testing.T, w *httptest.ResponseRecorder)
		expectedStatus int
	}{
		{
			name:      "Success",
			sessionID: "123e4567-e89b-12d3-a456-426614174000",
			setup: func() (*utils.MockValidator, *middleware.MockAuthContext, *services.MockInterviewSessionService) {
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)
				mockInterviewSessionService := new(services.MockInterviewSessionService)

				realValidator := validator.New()

				mockValidator.EXPECT().
					GetValidate().
					Return(realValidator)

				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, nil)

				mockInterviewSessionService.EXPECT().
					DeleteUserInterviewSessionByID(ctx, "123e4567-e89b-12d3-a456-426614174000").
					Return(nil)

				return mockValidator, mockAuthContext, mockInterviewSessionService
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNoContent, w.Code)
				assert.Empty(t, w.Body.String())
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name:      "Error - Empty Session ID",
			sessionID: "",
			setup: func() (*utils.MockValidator, *middleware.MockAuthContext, *services.MockInterviewSessionService) {
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)
				mockInterviewSessionService := new(services.MockInterviewSessionService)

				return mockValidator, mockAuthContext, mockInterviewSessionService
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, w.Code)
				assert.Contains(t, w.Body.String(), "The session ID is invalid. Please try again.")
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:      "Error - Invalid Session ID Format",
			sessionID: "invalid-session-id",
			setup: func() (*utils.MockValidator, *middleware.MockAuthContext, *services.MockInterviewSessionService) {
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)
				mockInterviewSessionService := new(services.MockInterviewSessionService)

				realValidator := validator.New()

				mockValidator.EXPECT().
					GetValidate().
					Return(realValidator)

				return mockValidator, mockAuthContext, mockInterviewSessionService
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, w.Code)
				assert.Contains(t, w.Body.String(), "The session ID is invalid. Please try again.")
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:      "Error - Invalid UUID Format",
			sessionID: "not-a-uuid",
			setup: func() (*utils.MockValidator, *middleware.MockAuthContext, *services.MockInterviewSessionService) {
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)
				mockInterviewSessionService := new(services.MockInterviewSessionService)

				realValidator := validator.New()

				mockValidator.EXPECT().
					GetValidate().
					Return(realValidator)

				return mockValidator, mockAuthContext, mockInterviewSessionService
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, w.Code)
				assert.Contains(t, w.Body.String(), "The session ID is invalid. Please try again.")
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:      "Error - Extract Auth Context Error",
			sessionID: "123e4567-e89b-12d3-a456-426614174000",
			setup: func() (*utils.MockValidator, *middleware.MockAuthContext, *services.MockInterviewSessionService) {
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)
				mockInterviewSessionService := new(services.MockInterviewSessionService)

				realValidator := validator.New()

				mockValidator.EXPECT().
					GetValidate().
					Return(realValidator)

				authErr := app_error.New(errors.New("extract auth context error"), app_error.ErrCodeAuthInvalidHeader)
				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, authErr)

				return mockValidator, mockAuthContext, mockInterviewSessionService
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusUnauthorized, w.Code)
				assert.Contains(t, w.Body.String(), "Please log in to continue")
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:      "Error - Service Layer Error - Session Not Found",
			sessionID: "123e4567-e89b-12d3-a456-426614174000",
			setup: func() (*utils.MockValidator, *middleware.MockAuthContext, *services.MockInterviewSessionService) {
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)
				mockInterviewSessionService := new(services.MockInterviewSessionService)

				realValidator := validator.New()

				mockValidator.EXPECT().
					GetValidate().
					Return(realValidator)

				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, nil)

				serviceErr := app_error.New(err, app_error.ErrCodeSessionNotFound)
				mockInterviewSessionService.EXPECT().
					DeleteUserInterviewSessionByID(ctx, "123e4567-e89b-12d3-a456-426614174000").
					Return(serviceErr)

				return mockValidator, mockAuthContext, mockInterviewSessionService
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, w.Code)
				assert.Contains(t, w.Body.String(), "The session was not found")
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:      "Error - Service Layer Error - Invalid UUID",
			sessionID: "123e4567-e89b-12d3-a456-426614174000",
			setup: func() (*utils.MockValidator, *middleware.MockAuthContext, *services.MockInterviewSessionService) {
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)
				mockInterviewSessionService := new(services.MockInterviewSessionService)

				realValidator := validator.New()

				mockValidator.EXPECT().
					GetValidate().
					Return(realValidator)

				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, nil)

				serviceErr := app_error.New(err, app_error.ErrCodeGeneralInvalidUUID)
				mockInterviewSessionService.EXPECT().
					DeleteUserInterviewSessionByID(ctx, "123e4567-e89b-12d3-a456-426614174000").
					Return(serviceErr)

				return mockValidator, mockAuthContext, mockInterviewSessionService
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, w.Code)
				assert.Contains(t, w.Body.String(), "The UUID is invalid. Please try again.")
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:      "Error - Service Layer Error - Permission Denied",
			sessionID: "123e4567-e89b-12d3-a456-426614174000",
			setup: func() (*utils.MockValidator, *middleware.MockAuthContext, *services.MockInterviewSessionService) {
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)
				mockInterviewSessionService := new(services.MockInterviewSessionService)

				realValidator := validator.New()

				mockValidator.EXPECT().
					GetValidate().
					Return(realValidator)

				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, nil)

				serviceErr := app_error.New(err, app_error.ErrCodeGeneralPermissionDenied)
				mockInterviewSessionService.EXPECT().
					DeleteUserInterviewSessionByID(ctx, "123e4567-e89b-12d3-a456-426614174000").
					Return(serviceErr)

				return mockValidator, mockAuthContext, mockInterviewSessionService
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusUnauthorized, w.Code)
				assert.Contains(t, w.Body.String(), "You are not authorized to perform this action")
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:      "Error - Service Layer Error - Database Connection",
			sessionID: "123e4567-e89b-12d3-a456-426614174000",
			setup: func() (*utils.MockValidator, *middleware.MockAuthContext, *services.MockInterviewSessionService) {
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)
				mockInterviewSessionService := new(services.MockInterviewSessionService)

				realValidator := validator.New()

				mockValidator.EXPECT().
					GetValidate().
					Return(realValidator)

				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, nil)

				serviceErr := app_error.New(err, app_error.ErrCodeGeneralDatabaseConnection)
				mockInterviewSessionService.EXPECT().
					DeleteUserInterviewSessionByID(ctx, "123e4567-e89b-12d3-a456-426614174000").
					Return(serviceErr)

				return mockValidator, mockAuthContext, mockInterviewSessionService
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, w.Code)
				assert.Contains(t, w.Body.String(), "Database connection error")
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:      "Error - Service Layer Error - Server Unavailable",
			sessionID: "123e4567-e89b-12d3-a456-426614174000",
			setup: func() (*utils.MockValidator, *middleware.MockAuthContext, *services.MockInterviewSessionService) {
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)
				mockInterviewSessionService := new(services.MockInterviewSessionService)

				realValidator := validator.New()

				mockValidator.EXPECT().
					GetValidate().
					Return(realValidator)

				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, nil)

				serviceErr := app_error.New(err, app_error.ErrCodeGeneralServerUnavailable)
				mockInterviewSessionService.EXPECT().
					DeleteUserInterviewSessionByID(ctx, "123e4567-e89b-12d3-a456-426614174000").
					Return(serviceErr)

				return mockValidator, mockAuthContext, mockInterviewSessionService
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, w.Code)
				assert.Contains(t, w.Body.String(), "We're having trouble connecting to the server")
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:      "Error - Service Layer Error - Generic Error",
			sessionID: "123e4567-e89b-12d3-a456-426614174000",
			setup: func() (*utils.MockValidator, *middleware.MockAuthContext, *services.MockInterviewSessionService) {
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)
				mockInterviewSessionService := new(services.MockInterviewSessionService)

				realValidator := validator.New()

				mockValidator.EXPECT().
					GetValidate().
					Return(realValidator)

				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, nil)

				serviceErr := errors.New("generic service error")
				mockInterviewSessionService.EXPECT().
					DeleteUserInterviewSessionByID(ctx, "123e4567-e89b-12d3-a456-426614174000").
					Return(serviceErr)

				return mockValidator, mockAuthContext, mockInterviewSessionService
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, w.Code)
				assert.Contains(t, w.Body.String(), "generic service error")
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			url := fmt.Sprintf("/api/v1/sessions/%s", tt.sessionID)
			c.Request = httptest.NewRequest(http.MethodDelete, url, nil)
			c.Params = gin.Params{{Key: "session_id", Value: tt.sessionID}}

			mockValidator, mockAuthContext, mockInterviewSessionService := tt.setup()
			defer mockValidator.AssertExpectations(t)
			defer mockAuthContext.AssertExpectations(t)
			defer mockInterviewSessionService.AssertExpectations(t)

			handler := NewInterviewSessionHandler(mockInterviewSessionService, log, mockAuthContext, mockValidator, nil, nil, nil)
			handler.DeleteUserInterviewSessionByID(c)

			tt.verify(t, w)
		})
	}
}
