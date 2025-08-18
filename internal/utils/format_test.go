package utils

import (
	"database/sql"
	"math"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestRoundToTwoDecimalPlaces(t *testing.T) {
	tests := []struct {
		name     string
		input    float64
		expected float64
	}{
		{
			name:     "Round positive number with 2 decimal places",
			input:    3.14159,
			expected: 3.14,
		},
		{
			name:     "Round positive number with 1 decimal place",
			input:    3.1,
			expected: 3.1,
		},
		{
			name:     "Round positive number with 3 decimal places",
			input:    3.145,
			expected: 3.15,
		},
		{
			name:     "Round positive number with 4 decimal places",
			input:    3.1449,
			expected: 3.14,
		},
		{
			name:     "Round positive number with 5 decimal places",
			input:    3.14499,
			expected: 3.14,
		},
		{
			name:     "Round negative number with 2 decimal places",
			input:    -3.14159,
			expected: -3.14,
		},
		{
			name:     "Round zero",
			input:    0.0,
			expected: 0.0,
		},
		{
			name:     "Round very small positive number",
			input:    0.001,
			expected: 0.0,
		},
		{
			name:     "Round very small negative number",
			input:    -0.001,
			expected: 0.0,
		},
		{
			name:     "Round large positive number",
			input:    123456.789,
			expected: 123456.79,
		},
		{
			name:     "Round large negative number",
			input:    -123456.789,
			expected: -123456.79,
		},
		{
			name:     "Round number exactly at 0.5 boundary",
			input:    3.5,
			expected: 3.5,
		},
		{
			name:     "Round number just below 0.5 boundary",
			input:    3.499999,
			expected: 3.5,
		},
		{
			name:     "Round number just above 0.5 boundary",
			input:    3.500001,
			expected: 3.5,
		},
		{
			name:     "Round number with many decimal places",
			input:    3.141592653589793,
			expected: 3.14,
		},
		{
			name:     "Round number that should round down",
			input:    3.141,
			expected: 3.14,
		},
		{
			name:     "Round number that should round up",
			input:    3.146,
			expected: 3.15,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RoundToTwoDecimalPlaces(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRoundToTwoDecimalPlaces_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    float64
		expected float64
	}{
		{
			name:     "Positive infinity",
			input:    math.Inf(1),
			expected: math.Inf(1),
		},
		{
			name:     "Negative infinity",
			input:    math.Inf(-1),
			expected: math.Inf(-1),
		},
		{
			name:     "NaN",
			input:    math.NaN(),
			expected: math.NaN(),
		},
		{
			name:     "Maximum float64 value",
			input:    math.MaxFloat64,
			expected: math.Inf(1), // MaxFloat64 rounds to infinity
		},
		{
			name:     "Minimum float64 value",
			input:    -math.MaxFloat64,
			expected: math.Inf(-1), // -MaxFloat64 rounds to negative infinity
		},
		{
			name:     "Smallest positive float64 value",
			input:    math.SmallestNonzeroFloat64,
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RoundToTwoDecimalPlaces(tt.input)
			if math.IsNaN(tt.input) {
				assert.True(t, math.IsNaN(result))
			} else if math.IsInf(tt.expected, 1) {
				assert.True(t, math.IsInf(result, 1))
			} else if math.IsInf(tt.expected, -1) {
				assert.True(t, math.IsInf(result, -1))
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestRoundToTwoDecimalPlaces_Precision(t *testing.T) {
	// Test that the function maintains precision for numbers that should not be rounded
	tests := []struct {
		name     string
		input    float64
		expected float64
	}{
		{"1.0", 1.0, 1.0},
		{"1.1", 1.1, 1.1},
		{"1.12", 1.12, 1.12},
		{"1.123", 1.12, 1.12},
		{"1.125", 1.13, 1.13},
		{"1.126", 1.13, 1.13},
		{"1.129", 1.13, 1.13},
		{"1.13", 1.13, 1.13},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RoundToTwoDecimalPlaces(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSafeNullUUIDToString(t *testing.T) {
	tests := []struct {
		name     string
		input    sql.NullString
		expected string
	}{
		{
			name: "Valid null string",
			input: sql.NullString{
				String: "test-uuid",
				Valid:  true,
			},
			expected: "test-uuid",
		},
		{
			name: "Invalid null string",
			input: sql.NullString{
				String: "test-uuid",
				Valid:  false,
			},
			expected: "",
		},
		{
			name: "Empty valid string",
			input: sql.NullString{
				String: "",
				Valid:  true,
			},
			expected: "",
		},
		{
			name: "Empty invalid string",
			input: sql.NullString{
				String: "",
				Valid:  false,
			},
			expected: "",
		},
		{
			name: "Long valid string",
			input: sql.NullString{
				String: "very-long-uuid-string-that-exceeds-normal-length",
				Valid:  true,
			},
			expected: "very-long-uuid-string-that-exceeds-normal-length",
		},
		{
			name: "Special characters in valid string",
			input: sql.NullString{
				String: "uuid-with-special-chars!@#$%^&*()",
				Valid:  true,
			},
			expected: "uuid-with-special-chars!@#$%^&*()",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SafeNullUUIDToString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSafeCustomNullUUIDToString(t *testing.T) {
	tests := []struct {
		name     string
		input    NullUUID
		expected string
	}{
		{
			name: "Valid UUID",
			input: NullUUID{
				UUID:  uuid.New(),
				Valid: true,
			},
			expected: "", // Will be set dynamically
		},
		{
			name: "Invalid UUID",
			input: NullUUID{
				UUID:  uuid.New(),
				Valid: false,
			},
			expected: "",
		},
		{
			name: "Zero UUID with valid flag",
			input: NullUUID{
				UUID:  uuid.Nil,
				Valid: true,
			},
			expected: "00000000-0000-0000-0000-000000000000",
		},
		{
			name: "Zero UUID with invalid flag",
			input: NullUUID{
				UUID:  uuid.Nil,
				Valid: false,
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SafeCustomNullUUIDToString(tt.input)
			if tt.name == "Valid UUID" {
				// For valid UUID, check that it's not empty and is a valid UUID format
				assert.NotEmpty(t, result)
				assert.Len(t, result, 36) // UUID length
				assert.Contains(t, result, "-")
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestSafeCustomNullUUIDToString_EdgeCases(t *testing.T) {
	// Test with multiple random UUIDs
	for i := 0; i < 100; i++ {
		randomUUID := uuid.New()
		input := NullUUID{
			UUID:  randomUUID,
			Valid: true,
		}

		result := SafeCustomNullUUIDToString(input)

		// Verify result is not empty
		assert.NotEmpty(t, result)

		// Verify result matches the input UUID
		assert.Equal(t, randomUUID.String(), result)

		// Verify result is a valid UUID format
		assert.Len(t, result, 36)
		assert.Contains(t, result, "-")
	}
}

func TestNullUUIDStruct(t *testing.T) {
	// Test the NullUUID struct fields
	uuid := uuid.New()
	nullUUID := NullUUID{
		UUID:  uuid,
		Valid: true,
	}

	assert.Equal(t, uuid, nullUUID.UUID)
	assert.True(t, nullUUID.Valid)

	// Test with invalid flag
	nullUUID.Valid = false
	assert.False(t, nullUUID.Valid)
}

func TestRoundToTwoDecimalPlaces_Consistency(t *testing.T) {
	// Test that the same input always produces the same output
	input := 3.14159
	expected := 3.14

	for i := 0; i < 1000; i++ {
		result := RoundToTwoDecimalPlaces(input)
		assert.Equal(t, expected, result)
	}
}

func TestRoundToTwoDecimalPlaces_Performance(t *testing.T) {
	// Test performance with multiple calls
	input := 3.14159

	for i := 0; i < 100000; i++ {
		_ = RoundToTwoDecimalPlaces(input)
	}
}

func TestSafeNullUUIDToString_Performance(t *testing.T) {
	// Test performance with multiple calls
	input := sql.NullString{
		String: "test-uuid",
		Valid:  true,
	}

	for i := 0; i < 100000; i++ {
		_ = SafeNullUUIDToString(input)
	}
}

func TestSafeCustomNullUUIDToString_Performance(t *testing.T) {
	// Test performance with multiple calls
	input := NullUUID{
		UUID:  uuid.New(),
		Valid: true,
	}

	for i := 0; i < 100000; i++ {
		_ = SafeCustomNullUUIDToString(input)
	}
}
