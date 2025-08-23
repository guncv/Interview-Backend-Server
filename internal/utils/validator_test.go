package utils

import (
	"mime/multipart"
	"reflect"
	"testing"
	"time"

	"errors"

	ut "github.com/go-playground/universal-translator"
	"github.com/stretchr/testify/assert"
	"gitlab.com/interview-simulation/interview-backend-server/internal/entities"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
)

func TestNewValidator(t *testing.T) {
	logger := log.Initialize("test")
	validator := NewValidator(logger)

	assert.NotNil(t, validator)
	assert.IsType(t, &validatorImpl{}, validator)
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

func TestValidateEmail(t *testing.T) {
	logger := log.Initialize("test")
	validator := NewValidator(logger)

	tests := []struct {
		name     string
		email    string
		expected bool
	}{
		{
			name:     "Valid email",
			email:    "user@example.com",
			expected: true,
		},
		{
			name:     "Valid email with subdomain",
			email:    "user@sub.example.com",
			expected: true,
		},
		{
			name:     "Valid email with plus",
			email:    "user+tag@example.com",
			expected: true,
		},
		{
			name:     "Valid email with dots",
			email:    "user.name@example.com",
			expected: true,
		},
		{
			name:     "Valid email with underscore",
			email:    "user_name@example.com",
			expected: true,
		},
		{
			name:     "Valid email with percent",
			email:    "user%tag@example.com",
			expected: true,
		},
		{
			name:     "Valid email with hyphen in domain",
			email:    "user@example-domain.com",
			expected: true,
		},
		{
			name:     "Invalid email - missing @",
			email:    "userexample.com",
			expected: false,
		},
		{
			name:     "Invalid email - missing domain",
			email:    "user@",
			expected: false,
		},
		{
			name:     "Invalid email - missing TLD",
			email:    "user@example",
			expected: false,
		},
		{
			name:     "Invalid email - consecutive dots",
			email:    "user..name@example.com",
			expected: false,
		},
		{
			name:     "Invalid email - @ followed by dot",
			email:    "user@.example.com",
			expected: false,
		},
		{
			name:     "Invalid email - dot followed by @",
			email:    "user.@example.com",
			expected: false,
		},
		{
			name:     "Invalid email - @ followed by hyphen",
			email:    "user@-example.com",
			expected: false,
		},
		{
			name:     "Invalid email - hyphen followed by dot",
			email:    "user@example-.com",
			expected: false,
		},
		{
			name:     "Invalid email - dot followed by hyphen",
			email:    "user@example.-com",
			expected: false,
		},
		{
			name:     "Invalid email - single character TLD",
			email:    "user@example.c",
			expected: false,
		},
		{
			name:     "Invalid email - spaces",
			email:    "user name@example.com",
			expected: false,
		},
		{
			name:     "Invalid email - multiple @",
			email:    "user@name@example.com",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test request with the email
			req := &entities.SignUpUserRequest{
				Email:       tt.email,
				Password:    "password123",
				FullName:    "Test User",
				Country:     "USA",
				Gender:      "male",
				DateOfBirth: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
			}

			// Validate the struct
			err := validator.GetValidate().Struct(req)

			if tt.expected {
				assert.NoError(t, err, "Expected valid email but got validation error")
			} else {
				assert.Error(t, err, "Expected invalid email but validation passed")
			}
		})
	}
}

func TestValidateEmailEmptyString(t *testing.T) {
	logger := log.Initialize("test")
	validator := NewValidator(logger)

	// Test that empty email is handled by required tag, not valid_email
	req := &entities.SignUpUserRequest{
		Email:       "",
		Password:    "password123",
		FullName:    "Test User",
		Country:     "USA",
		Gender:      "male",
		DateOfBirth: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	err := validator.GetValidate().Struct(req)
	assert.Error(t, err, "Empty email should fail required validation")

	// The error should be about required field, not invalid email format
	assert.Contains(t, err.Error(), "required", "Error should mention required field")
}

func TestValidateEmailEnhancedValidation(t *testing.T) {
	logger := log.Initialize("test")
	validator := NewValidator(logger)

	// Test enhanced validation scenarios
	enhancedTests := []struct {
		name     string
		email    string
		expected bool
		reason   string
	}{
		{
			name:     "Valid corporate email",
			email:    "john.doe@company-name.co.uk",
			expected: true,
			reason:   "Should accept valid corporate emails with hyphens and multiple TLDs",
		},
		{
			name:     "Valid email with numbers in domain",
			email:    "user123@domain123.com",
			expected: true,
			reason:   "Should accept domains with numbers",
		},
		{
			name:     "Valid email with underscore in local part",
			email:    "user_name@example.com",
			expected: true,
			reason:   "Should accept underscores in local part",
		},
		{
			name:     "Invalid email - starts with dot",
			email:    ".user@example.com",
			expected: false,
			reason:   "Should reject emails starting with dot",
		},
		{
			name:     "Invalid email - ends with dot",
			email:    "user.@example.com",
			expected: false,
			reason:   "Should reject emails ending with dot",
		},
		{
			name:     "Invalid email - domain starts with dot",
			email:    "user@.example.com",
			expected: false,
			reason:   "Should reject domains starting with dot",
		},
		{
			name:     "Invalid email - domain ends with dot",
			email:    "user@example.com.",
			expected: false,
			reason:   "Should reject domains ending with dot",
		},
		{
			name:     "Invalid email - consecutive hyphens",
			email:    "user@ex--ample.com",
			expected: false,
			reason:   "Should reject consecutive hyphens in domain",
		},
		{
			name:     "Invalid email - TLD too short",
			email:    "user@example.a",
			expected: false,
			reason:   "Should reject single character TLDs",
		},
		{
			name:     "Invalid email - invalid characters",
			email:    "user@example.com!",
			expected: false,
			reason:   "Should reject invalid characters",
		},
	}

	for _, tt := range enhancedTests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test request with the email
			req := &entities.SignUpUserRequest{
				Email:       tt.email,
				Password:    "password123",
				FullName:    "Test User",
				Country:     "USA",
				Gender:      "male",
				DateOfBirth: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
			}

			// Validate the struct
			err := validator.GetValidate().Struct(req)

			if tt.expected {
				assert.NoError(t, err, "Expected valid email but got validation error: %s", tt.reason)
			} else {
				assert.Error(t, err, "Expected invalid email but validation passed: %s", tt.reason)
			}
		})
	}
}

func TestGetSpecificBindingErrorMessage(t *testing.T) {
	logger := log.Initialize("test")
	validator := NewValidator(logger)

	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "Date parsing error",
			err:      errors.New("parsing time \"invalid-date\" as \"2006-01-02\": cannot parse \"invalid-date\" as \"2006\""),
			expected: "invalid date format. Please use ISO 8601 format (YYYY-MM-DD)",
		},
		{
			name:     "JSON unexpected end error",
			err:      errors.New("unexpected end of JSON input"),
			expected: "invalid JSON format. Request body is incomplete or malformed",
		},
		{
			name:     "JSON invalid character error",
			err:      errors.New("invalid character 'x' looking for beginning of value"),
			expected: "invalid JSON format. Check for syntax errors in your request body",
		},
		{
			name:     "JSON unmarshal error",
			err:      errors.New("json: cannot unmarshal string into Go struct field"),
			expected: "invalid data type. Expected number but received string",
		},
		{
			name:     "Type conversion string to bool error",
			err:      errors.New("cannot unmarshal string into Go struct field .Field of type bool"),
			expected: "invalid data type. Expected boolean value but received string",
		},
		{
			name:     "Type conversion string to number error",
			err:      errors.New("cannot unmarshal string into Go struct field .Field of type int"),
			expected: "invalid data type. Expected number but received string",
		},
		{
			name:     "Type conversion number to string error",
			err:      errors.New("cannot unmarshal number into Go struct field .Field of type string"),
			expected: "invalid data type. Expected number but received string",
		},
		{
			name:     "Generic type conversion error",
			err:      errors.New("cannot unmarshal array into Go struct field"),
			expected: "invalid data type. One or more fields have incorrect data types",
		},
		{
			name:     "Field validation error",
			err:      errors.New("field validation failed"),
			expected: "invalid field value. Please check the data types and formats of your input fields",
		},
		{
			name:     "Generic error",
			err:      errors.New("some random error"),
			expected: "invalid request format. Please check your input data and try again",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.getSpecificBindingErrorMessage(tt.err)
			assert.Equal(t, tt.expected, result, "Expected error message '%s' but got '%s'", tt.expected, result)
		})
	}
}

func TestImprovedErrorHandlingDemonstration(t *testing.T) {
	logger := log.Initialize("test")
	validator := NewValidator(logger)

	// This test demonstrates how the improved error handling provides specific messages
	// instead of the generic "Something went wrong with the request. Please try again."

	t.Run("Demonstrate improved error messages", func(t *testing.T) {
		// Simulate different types of binding errors that users commonly encounter

		// 1. Date format error - user sends "2024/05/12" instead of "2024-05-12"
		dateError := errors.New("parsing time \"2024/05/12\" as \"2006-01-02\": cannot parse \"/05/12\" as \"-\"")
		dateMsg := validator.getSpecificBindingErrorMessage(dateError)
		assert.Equal(t, "invalid date format. Please use ISO 8601 format (YYYY-MM-DD)", dateMsg)

		// 2. JSON syntax error - user sends malformed JSON
		jsonError := errors.New("invalid character '}' looking for beginning of value")
		jsonMsg := validator.getSpecificBindingErrorMessage(jsonError)
		assert.Equal(t, "invalid JSON format. Check for syntax errors in your request body", jsonMsg)

		// 3. Type mismatch - user sends string where number expected
		typeError := errors.New("cannot unmarshal string \"abc\" into Go struct field .Age of type int")
		typeMsg := validator.getSpecificBindingErrorMessage(typeError)
		assert.Equal(t, "invalid data type. Expected number but received string", typeMsg)

		// 4. Incomplete JSON - user sends partial request
		incompleteError := errors.New("unexpected end of JSON input")
		incompleteMsg := validator.getSpecificBindingErrorMessage(incompleteError)
		assert.Equal(t, "invalid JSON format. Request body is incomplete or malformed", incompleteMsg)

		t.Logf("✅ Date format error: %s", dateMsg)
		t.Logf("✅ JSON syntax error: %s", jsonMsg)
		t.Logf("✅ Type mismatch error: %s", typeMsg)
		t.Logf("✅ Incomplete JSON error: %s", incompleteMsg)
		t.Logf("")
		t.Logf("🎯 Instead of generic 'Something went wrong', users now get specific guidance!")
	})
}
