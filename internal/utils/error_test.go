package utils

import (
	"errors"
	"net/http"
	"testing"

	"net/http/httptest"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	app_error "gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
)

func TestRespondWithError(t *testing.T) {
	tests := []struct {
		name             string
		err              error
		expectedStatus   int
		expectedResponse string
	}{
		{
			name: "Respond with AppError",
			err: &app_error.AppError{
				Code:    "ONX0202",
				Err:     nil,
				Message: "Your session has expired. Please log in again.",
			},
			expectedStatus:   http.StatusUnauthorized,
			expectedResponse: `{"code":"ONX0202","message":"Your session has expired. Please log in again."}`,
		},
		{
			name:             "Respond with validation errors",
			err:              validator.ValidationErrors{}, // Empty validation errors for testing
			expectedStatus:   http.StatusBadRequest,
			expectedResponse: `{"code":"ONX0603","message":"Validation failed: "}`,
		},
		{
			name:             "Respond with single validation error",
			err:              errors.New("Field validation for 'email' failed on the 'required' tag"),
			expectedStatus:   http.StatusBadRequest,
			expectedResponse: `{"code":"ONX0603","message":"Validation failed: Field validation for 'email' failed on the 'required' tag"}`,
		},
		{
			name:             "Respond with generic error",
			err:              errors.New("Something went wrong"),
			expectedStatus:   http.StatusInternalServerError,
			expectedResponse: `{}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(rr)

			RespondWithError(ctx, tt.err)

			assert.Equal(t, tt.expectedStatus, rr.Code)
			assert.JSONEq(t, tt.expectedResponse, rr.Body.String())
		})
	}
}

func TestFormatValidationErrors(t *testing.T) {
	tests := []struct {
		name     string
		errors   validator.ValidationErrors
		expected string
	}{
		{
			name:     "Empty validation errors",
			errors:   validator.ValidationErrors{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatValidationErrors(tt.errors)
			assert.Equal(t, tt.expected, result)
		})
	}
}
