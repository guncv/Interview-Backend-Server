package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDerefString(t *testing.T) {
	tests := []struct {
		name     string
		input    *string
		expected string
	}{
		{
			name:     "Dereference valid string pointer",
			input:    stringPtr("hello world"),
			expected: "hello world",
		},
		{
			name:     "Dereference nil string pointer",
			input:    nil,
			expected: "",
		},
		{
			name:     "Dereference empty string pointer",
			input:    stringPtr(""),
			expected: "",
		},
		{
			name:     "Dereference long string pointer",
			input:    stringPtr("this is a very long string with many characters to test the function"),
			expected: "this is a very long string with many characters to test the function",
		},
		{
			name:     "Dereference string with special characters",
			input:    stringPtr("hello@world.com!@#$%^&*()"),
			expected: "hello@world.com!@#$%^&*()",
		},
		{
			name:     "Dereference string with unicode characters",
			input:    stringPtr("hello 世界"),
			expected: "hello 世界",
		},
		{
			name:     "Dereference string with newlines",
			input:    stringPtr("hello\nworld"),
			expected: "hello\nworld",
		},
		{
			name:     "Dereference string with tabs",
			input:    stringPtr("hello\tworld"),
			expected: "hello\tworld",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DerefString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDerefBool(t *testing.T) {
	tests := []struct {
		name     string
		input    *bool
		expected bool
	}{
		{
			name:     "Dereference true boolean pointer",
			input:    boolPtr(true),
			expected: true,
		},
		{
			name:     "Dereference false boolean pointer",
			input:    boolPtr(false),
			expected: false,
		},
		{
			name:     "Dereference nil boolean pointer",
			input:    nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DerefBool(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDerefInt(t *testing.T) {
	tests := []struct {
		name     string
		input    *int
		expected int
	}{
		{
			name:     "Dereference positive integer pointer",
			input:    intPtr(42),
			expected: 42,
		},
		{
			name:     "Dereference negative integer pointer",
			input:    intPtr(-42),
			expected: -42,
		},
		{
			name:     "Dereference zero integer pointer",
			input:    intPtr(0),
			expected: 0,
		},
		{
			name:     "Dereference nil integer pointer",
			input:    nil,
			expected: 0,
		},
		{
			name:     "Dereference large positive integer pointer",
			input:    intPtr(2147483647),
			expected: 2147483647,
		},
		{
			name:     "Dereference large negative integer pointer",
			input:    intPtr(-2147483648),
			expected: -2147483648,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DerefInt(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDerefInt32(t *testing.T) {
	tests := []struct {
		name     string
		input    *int32
		expected int32
	}{
		{
			name:     "Dereference positive int32 pointer",
			input:    int32Ptr(42),
			expected: 42,
		},
		{
			name:     "Dereference negative int32 pointer",
			input:    int32Ptr(-42),
			expected: -42,
		},
		{
			name:     "Dereference zero int32 pointer",
			input:    int32Ptr(0),
			expected: 0,
		},
		{
			name:     "Dereference nil int32 pointer",
			input:    nil,
			expected: 0,
		},
		{
			name:     "Dereference large positive int32 pointer",
			input:    int32Ptr(2147483647),
			expected: 2147483647,
		},
		{
			name:     "Dereference large negative int32 pointer",
			input:    int32Ptr(-2147483648),
			expected: -2147483648,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DerefInt32(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDerefString_EdgeCases(t *testing.T) {
	// Test with very long strings
	longString := make([]byte, 10000)
	for i := range longString {
		longString[i] = byte(i % 256)
	}
	longStringStr := string(longString)

	result := DerefString(&longStringStr)
	assert.Equal(t, longStringStr, result)

	// Test with strings containing null bytes
	nullString := "hello\x00world"
	result = DerefString(&nullString)
	assert.Equal(t, nullString, result)
}

func TestDerefBool_EdgeCases(t *testing.T) {
	// Test with alternating boolean values
	for i := 0; i < 1000; i++ {
		val := i%2 == 0
		result := DerefBool(&val)
		assert.Equal(t, val, result)
	}
}

func TestDerefInt_EdgeCases(t *testing.T) {
	// Test with various integer values
	testValues := []int{-1000, -100, -10, -1, 0, 1, 10, 100, 1000}

	for _, val := range testValues {
		result := DerefInt(&val)
		assert.Equal(t, val, result)
	}

	// Test with nil pointer multiple times
	for i := 0; i < 1000; i++ {
		result := DerefInt(nil)
		assert.Equal(t, 0, result)
	}
}

func TestDerefInt32_EdgeCases(t *testing.T) {
	// Test with various int32 values
	testValues := []int32{-1000, -100, -10, -1, 0, 1, 10, 100, 1000}

	for _, val := range testValues {
		result := DerefInt32(&val)
		assert.Equal(t, val, result)
	}

	// Test with nil pointer multiple times
	for i := 0; i < 1000; i++ {
		result := DerefInt32(nil)
		assert.Equal(t, int32(0), result)
	}
}

func TestDerefString_Consistency(t *testing.T) {
	// Test that the same input always produces the same output
	input := "test string"

	for i := 0; i < 1000; i++ {
		result := DerefString(&input)
		assert.Equal(t, input, result)
	}
}

func TestDerefBool_Consistency(t *testing.T) {
	// Test that the same input always produces the same output
	input := true

	for i := 0; i < 1000; i++ {
		result := DerefBool(&input)
		assert.Equal(t, input, result)
	}
}

func TestDerefInt_Consistency(t *testing.T) {
	// Test that the same input always produces the same output
	input := 42

	for i := 0; i < 1000; i++ {
		result := DerefInt(&input)
		assert.Equal(t, input, result)
	}
}

func TestDerefInt32_Consistency(t *testing.T) {
	// Test that the same input always produces the same output
	input := int32(42)

	for i := 0; i < 1000; i++ {
		result := DerefInt32(&input)
		assert.Equal(t, input, result)
	}
}

func TestDerefString_Performance(t *testing.T) {
	// Test performance with multiple calls
	input := "performance test string"

	for i := 0; i < 100000; i++ {
		_ = DerefString(&input)
	}
}

func TestDerefBool_Performance(t *testing.T) {
	// Test performance with multiple calls
	input := true

	for i := 0; i < 100000; i++ {
		_ = DerefBool(&input)
	}
}

func TestDerefInt_Performance(t *testing.T) {
	// Test performance with multiple calls
	input := 42

	for i := 0; i < 100000; i++ {
		_ = DerefInt(&input)
	}
}

func TestDerefInt32_Performance(t *testing.T) {
	// Test performance with multiple calls
	input := int32(42)

	for i := 0; i < 100000; i++ {
		_ = DerefInt32(&input)
	}
}

func TestDerefString_NilPointerPerformance(t *testing.T) {
	// Test performance with nil pointer
	for i := 0; i < 100000; i++ {
		_ = DerefString(nil)
	}
}

func TestDerefBool_NilPointerPerformance(t *testing.T) {
	// Test performance with nil pointer
	for i := 0; i < 100000; i++ {
		_ = DerefBool(nil)
	}
}

func TestDerefInt_NilPointerPerformance(t *testing.T) {
	// Test performance with nil pointer
	for i := 0; i < 100000; i++ {
		_ = DerefInt(nil)
	}
}

func TestDerefInt32_NilPointerPerformance(t *testing.T) {
	// Test performance with nil pointer
	for i := 0; i < 100000; i++ {
		_ = DerefInt32(nil)
	}
}

// Helper functions to create pointers for testing
func stringPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}

func intPtr(i int) *int {
	return &i
}

func int32Ptr(i int32) *int32 {
	return &i
}
