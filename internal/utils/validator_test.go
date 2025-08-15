package utils

import (
	"bytes"
	"mime/multipart"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	ut "github.com/go-playground/universal-translator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/interview-simulation/interview-backend-server/internal/entities"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
)

func TestNewValidator(t *testing.T) {
	logger := log.Initialize("test")
	validator := NewValidator(logger)

	assert.NotNil(t, validator)
	assert.IsType(t, &validatorImpl{}, validator)
}

func TestValidateAndBind_Success(t *testing.T) {
	tests := []struct {
		name     string
		jsonBody string
		target   interface{}
	}{
		{
			name: "Valid AdminCreateUserRequest",
			jsonBody: `{
				"email": "test@example.com",
				"password": "password123",
				"name": "Test User",
				"role": "admin"
			}`,
			target: &entities.AdminCreateUserRequest{},
		},
		{
			name: "Valid SignInByEmailAndPasswordRequest",
			jsonBody: `{
				"email": "test@example.com",
				"password": "password123"
			}`,
			target: &entities.SignInByEmailAndPasswordRequest{},
		},
		{
			name: "Valid ForgotUserPasswordRequest",
			jsonBody: `{
				"email": "test@example.com"
			}`,
			target: &entities.ForgotUserPasswordRequest{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := log.Initialize("test")
			validator := NewValidator(logger)

			gin.SetMode(gin.TestMode)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("POST", "/test", bytes.NewBufferString(tt.jsonBody))
			c.Request.Header.Set("Content-Type", "application/json")

			err := validator.ValidateAndBind(c, tt.target, "test_handler")
			assert.NoError(t, err)
		})
	}
}

func TestValidateAndBind_ValidationErrors(t *testing.T) {
	tests := []struct {
		name           string
		jsonBody       string
		target         interface{}
		expectedErrMsg string
	}{
		{
			name: "Required field missing - email",
			jsonBody: `{
				"password": "password123",
				"name": "Test User",
				"role": "admin"
			}`,
			target:         &entities.AdminCreateUserRequest{},
			expectedErrMsg: "email is required",
		},
		{
			name: "Invalid email format",
			jsonBody: `{
				"email": "invalid-email",
				"password": "password123",
				"name": "Test User",
				"role": "admin"
			}`,
			target:         &entities.AdminCreateUserRequest{},
			expectedErrMsg: "invalid email format",
		},
		{
			name: "Password too short",
			jsonBody: `{
				"email": "test@example.com",
				"password": "123",
				"name": "Test User",
				"role": "admin"
			}`,
			target:         &entities.AdminCreateUserRequest{},
			expectedErrMsg: "password must be at least 8 characters",
		},
		{
			name: "Name too long",
			jsonBody: `{
				"email": "test@example.com",
				"password": "password123",
				"name": "` + strings.Repeat("a", 51) + `",
				"role": "admin"
			}`,
			target:         &entities.AdminCreateUserRequest{},
			expectedErrMsg: "name is too long (max 50 characters)",
		},
		{
			name: "Invalid UUID format",
			jsonBody: `{
				"email": "test@example.com",
				"password": "password123",
				"name": "Test User",
				"role": "admin"
			}`,
			target:         &entities.AdminCreateUserRequest{},
			expectedErrMsg: "invalid organizationid format",
		},
		{
			name: "Invalid role value",
			jsonBody: `{
				"email": "test@example.com",
				"password": "password123",
				"name": "Test User",
				"role": "invalid_role"
			}`,
			target:         &entities.AdminCreateUserRequest{},
			expectedErrMsg: "invalid role value",
		},
		{
			name: "Country code wrong length",
			jsonBody: `{
				"name": "Test Org",
				"address": "123 Test Street",
				"contact_email": "contact@test.com",
				"contact_phone": "1234567890",
				"country": "USA"
			}`,
			target:         &entities.CreateOrganizationRequest{},
			expectedErrMsg: "invalid country",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := log.Initialize("test")
			validator := NewValidator(logger)

			gin.SetMode(gin.TestMode)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("POST", "/test", bytes.NewBufferString(tt.jsonBody))
			c.Request.Header.Set("Content-Type", "application/json")

			err := validator.ValidateAndBind(c, tt.target, "test_handler")

			require.Error(t, err)
			appErr, ok := err.(*app_error.AppError)
			require.True(t, ok)
			assert.Equal(t, app_error.ErrCodeAuthInvalidRequest, appErr.Code)
			assert.Equal(t, tt.expectedErrMsg, appErr.Message)
		})
	}
}

func TestGetSimpleErrorMessage(t *testing.T) {
	tests := []struct {
		name     string
		tag      string
		field    string
		param    string
		expected string
	}{
		{
			name:     "Required field",
			tag:      "required",
			field:    "Email",
			param:    "",
			expected: "email is required",
		},
		{
			name:     "Email validation",
			tag:      "email",
			field:    "Email",
			param:    "",
			expected: "invalid email format",
		},
		{
			name:     "Min length validation",
			tag:      "min",
			field:    "Password",
			param:    "8",
			expected: "password must be at least 8 characters",
		},
		{
			name:     "Max length validation",
			tag:      "max",
			field:    "Name",
			param:    "50",
			expected: "name is too long (max 50 characters)",
		},
		{
			name:     "UUID validation",
			tag:      "uuid",
			field:    "OrganizationID",
			param:    "",
			expected: "invalid organizationid format",
		},
		{
			name:     "OneOf validation",
			tag:      "oneof",
			field:    "Role",
			param:    "admin mentor trainee",
			expected: "invalid role value",
		},
		{
			name:     "Unknown validation tag",
			tag:      "custom",
			field:    "Field",
			param:    "",
			expected: "invalid field",
		},
		{
			name:     "File required validation",
			tag:      "file_required",
			field:    "ThumbnailURL",
			param:    "",
			expected: "thumbnail_url file is required",
		},
		{
			name:     "File required validation - other field",
			tag:      "file_required",
			field:    "ImageFile",
			param:    "",
			expected: "imagefile file is required",
		},
		{
			name:     "File optional validation",
			tag:      "file_optional",
			field:    "ThumbnailURL",
			param:    "",
			expected: "thumbnail_url must be a valid image file (JPEG, PNG, GIF, WebP, BMP, SVG)",
		},
		{
			name:     "File optional validation - other field",
			tag:      "file_optional",
			field:    "ImageFile",
			param:    "",
			expected: "imagefile must be a valid image file (JPEG, PNG, GIF, WebP, BMP, SVG)",
		},
		{
			name:     "GTE validation",
			tag:      "gte",
			field:    "Age",
			param:    "18",
			expected: "age must be greater than or equal to 18",
		},
		{
			name:     "CategoryID required validation",
			tag:      "required",
			field:    "CategoryID",
			param:    "",
			expected: "category_id is required",
		},
		{
			name:     "CategoryID min validation",
			tag:      "min",
			field:    "CategoryID",
			param:    "3",
			expected: "category_id must be at least 3 characters",
		},
		{
			name:     "CategoryID max validation",
			tag:      "max",
			field:    "CategoryID",
			param:    "50",
			expected: "category_id is too long (max 50 characters)",
		},
		{
			name:     "CategoryID UUID validation",
			tag:      "uuid",
			field:    "CategoryID",
			param:    "",
			expected: "invalid category_id format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock FieldError
			mockErr := &mockFieldError{
				tag:   tt.tag,
				field: tt.field,
				param: tt.param,
			}

			result := getSimpleErrorMessage(mockErr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Mock implementation of validator.FieldError for testing
type mockFieldError struct {
	tag   string
	field string
	param string
}

func (m *mockFieldError) Tag() string                    { return m.tag }
func (m *mockFieldError) ActualTag() string              { return m.tag }
func (m *mockFieldError) Namespace() string              { return "" }
func (m *mockFieldError) StructNamespace() string        { return "" }
func (m *mockFieldError) Field() string                  { return m.field }
func (m *mockFieldError) StructField() string            { return m.field }
func (m *mockFieldError) Value() interface{}             { return nil }
func (m *mockFieldError) Param() string                  { return m.param }
func (m *mockFieldError) Kind() reflect.Kind             { return reflect.String }
func (m *mockFieldError) Type() reflect.Type             { return reflect.TypeOf("") }
func (m *mockFieldError) Error() string                  { return "" }
func (m *mockFieldError) Translate(ut.Translator) string { return "" }

func TestValidateFileRequired(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected bool
	}{
		{
			name: "Valid file header",
			value: &multipart.FileHeader{
				Filename: "test.jpg",
				Size:     1024,
			},
			expected: true,
		},
		{
			name:     "Nil file header",
			value:    (*multipart.FileHeader)(nil),
			expected: false,
		},
		{
			name: "File header value (not pointer)",
			value: multipart.FileHeader{
				Filename: "test.jpg",
				Size:     1024,
			},
			expected: true,
		},
		{
			name:     "Non-empty string",
			value:    "test string",
			expected: true,
		},
		{
			name:     "Empty string",
			value:    "",
			expected: false,
		},
		{
			name:     "Non-empty byte slice",
			value:    []byte("test"),
			expected: true,
		},
		{
			name:     "Empty byte slice",
			value:    []byte{},
			expected: false,
		},
		{
			name:     "Non-empty uint8 slice",
			value:    []uint8{1, 2, 3},
			expected: true,
		},
		{
			name:     "Empty uint8 slice",
			value:    []uint8{},
			expected: false,
		},
		{
			name:     "Unknown type",
			value:    123,
			expected: false,
		},
		{
			name:     "Float type",
			value:    3.14,
			expected: false,
		},
		{
			name:     "Bool type",
			value:    true,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := log.Initialize("test")
			v := NewValidator(logger)
			validate := v.GetValidate()

			err := validate.Var(tt.value, "file_required")
			if tt.expected {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}
