package app_error

import (
	"testing"

	"github.com/lib/pq"
)

func TestHandleForeignKeyViolation(t *testing.T) {
	tests := []struct {
		name           string
		constraintName string
		expectedCode   ErrorCode
		expectedMsg    string
	}{
		{
			name:           "courses_created_by_fkey",
			constraintName: "courses_created_by_fkey",
			expectedCode:   ErrCodeGeneralConstraintViolation,
			expectedMsg:    "The specified user (creator) does not exist.",
		},
		{
			name:           "categories_parent_id_fkey",
			constraintName: "categories_parent_id_fkey",
			expectedCode:   ErrCodeGeneralConstraintViolation,
			expectedMsg:    "The specified parent category does not exist.",
		},
		{
			name:           "unknown_constraint",
			constraintName: "unknown_constraint",
			expectedCode:   ErrCodeGeneralConstraintViolation,
			expectedMsg:    "The data violates database constraints.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock PostgreSQL error
			pqErr := &pq.Error{
				Code:       "23503", // Foreign key violation
				Constraint: tt.constraintName,
				Message:    "insert or update on table violates foreign key constraint",
			}

			// Handle the error
			appErr := HandleForeignKeyViolation(pqErr)

			// Check the error code
			if appErr.Code != tt.expectedCode {
				t.Errorf("Expected error code %s, got %s", tt.expectedCode, appErr.Code)
			}

			// Check the error message
			if appErr.Message != tt.expectedMsg {
				t.Errorf("Expected error message '%s', got '%s'", tt.expectedMsg, appErr.Message)
			}
		})
	}
}

func TestHandleForeignKeyViolationWithContext(t *testing.T) {
	tests := []struct {
		name           string
		constraintName string
		context        string
		expectedCode   ErrorCode
		expectedMsg    string
	}{
		{
			name:           "create_category_with_invalid_parent",
			constraintName: "categories_parent_id_fkey",
			context:        "create_category",
			expectedCode:   ErrCodeGeneralConstraintViolation,
			expectedMsg:    "Cannot create category: The specified parent category does not exist.",
		},
		{
			name:           "update_category_with_invalid_parent",
			constraintName: "categories_parent_id_fkey",
			context:        "update_category",
			expectedCode:   ErrCodeGeneralConstraintViolation,
			expectedMsg:    "Cannot update category: The specified parent category does not exist.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock PostgreSQL error
			pqErr := &pq.Error{
				Code:       "23503", // Foreign key violation
				Constraint: tt.constraintName,
				Message:    "insert or update on table violates foreign key constraint",
			}

			// Handle the error with context
			appErr := HandleDatabaseErrorWithContext(pqErr, tt.context)

			// Check the error code
			if appErr.Code != tt.expectedCode {
				t.Errorf("Expected error code %s, got %s", tt.expectedCode, appErr.Code)
			}

			// Check the error message
			if appErr.Message != tt.expectedMsg {
				t.Errorf("Expected error message '%s', got '%s'", tt.expectedMsg, appErr.Message)
			}
		})
	}
}

func TestHandleDuplicateKeyViolation(t *testing.T) {
	tests := []struct {
		name         string
		errorMsg     string
		expectedCode ErrorCode
		expectedMsg  string
	}{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock error
			err := &pq.Error{
				Code:    "23505", // Unique violation
				Message: tt.errorMsg,
			}

			// Handle the error
			appErr := handleDuplicateKeyViolation(err)

			// Check the error code
			if appErr.Code != tt.expectedCode {
				t.Errorf("Expected error code %s, got %s", tt.expectedCode, appErr.Code)
			}

			// Check the error message
			if appErr.Message != tt.expectedMsg {
				t.Errorf("Expected error message '%s', got '%s'", tt.expectedMsg, appErr.Message)
			}
		})
	}
}
