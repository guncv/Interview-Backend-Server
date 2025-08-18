package utils

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
)

func TestNewGenerator(t *testing.T) {
	logger := log.Initialize(constants.TestAppEnv)
	gen := NewGenerator(logger)

	assert.NotNil(t, gen)
	assert.IsType(t, &generator{}, gen)
}

func TestGenerateUUID(t *testing.T) {
	logger := log.Initialize(constants.TestAppEnv)
	gen := NewGenerator(logger)
	ctx := context.Background()

	// Test multiple UUID generations
	for i := 0; i < 100; i++ {
		uuid := gen.GenerateUUID(ctx)

		// Verify UUID is not empty
		assert.NotEqual(t, uuid.String(), "")

		// Verify UUID is valid by checking it's not the zero value
		assert.NotEqual(t, "00000000-0000-0000-0000-000000000000", uuid.String())
	}
}

func TestGenerateUUIDWithNilContext(t *testing.T) {
	logger := log.Initialize(constants.TestAppEnv)
	gen := NewGenerator(logger)

	// Should not panic with nil context
	assert.NotPanics(t, func() {
		uuid := gen.GenerateUUID(nil)
		assert.NotEqual(t, uuid.String(), "")
		assert.NotEqual(t, "00000000-0000-0000-0000-000000000000", uuid.String())
	})
}

func TestGenerateRandomString(t *testing.T) {
	logger := log.Initialize(constants.TestAppEnv)
	gen := NewGenerator(logger)
	ctx := context.Background()

	tests := []struct {
		name     string
		length   int
		expected int
	}{
		{
			name:     "Generate random string with length 10",
			length:   10,
			expected: 10,
		},
		{
			name:     "Generate random string with length 20",
			length:   20,
			expected: 20,
		},
		{
			name:     "Generate random string with length 50",
			length:   50,
			expected: 50,
		},
		{
			name:     "Generate random string with length 100",
			length:   100,
			expected: 100,
		},
		{
			name:     "Generate random string with length 0",
			length:   0,
			expected: 0,
		},
		{
			name:     "Generate random string with length 1",
			length:   1,
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gen.GenerateRandomString(ctx, tt.length)

			// Verify length
			assert.Equal(t, tt.expected, len(result))

			// Verify content contains only valid characters
			validChars := "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
			for _, char := range result {
				assert.Contains(t, validChars, string(char))
			}
		})
	}
}

func TestGenerateRandomStringWithNegativeLength(t *testing.T) {
	logger := log.Initialize(constants.TestAppEnv)
	gen := NewGenerator(logger)
	ctx := context.Background()

	// Should panic with negative length
	assert.Panics(t, func() {
		gen.GenerateRandomString(ctx, -5)
	})
}

func TestGenerateRandomStringWithNilContext(t *testing.T) {
	logger := log.Initialize(constants.TestAppEnv)
	gen := NewGenerator(logger)

	// Should not panic with nil context
	assert.NotPanics(t, func() {
		result := gen.GenerateRandomString(nil, 10)
		assert.Equal(t, 10, len(result))
	})
}

func TestGenerateRandomStringUniqueness(t *testing.T) {
	logger := log.Initialize(constants.TestAppEnv)
	gen := NewGenerator(logger)
	ctx := context.Background()

	// Generate multiple strings and verify they are different
	length := 20
	results := make(map[string]bool)

	for i := 0; i < 1000; i++ {
		result := gen.GenerateRandomString(ctx, length)
		assert.Equal(t, length, len(result))

		// Check for uniqueness
		assert.False(t, results[result], "Duplicate string generated: %s", result)
		results[result] = true
	}
}

func TestGenerateRandomStringCharacterDistribution(t *testing.T) {
	logger := log.Initialize(constants.TestAppEnv)
	gen := NewGenerator(logger)
	ctx := context.Background()

	// Generate a long string to test character distribution
	length := 10000
	result := gen.GenerateRandomString(ctx, length)

	// Count character frequencies
	charCount := make(map[rune]int)
	for _, char := range result {
		charCount[char]++
	}

	// Verify all expected characters are present
	expectedChars := "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	for _, char := range expectedChars {
		assert.True(t, charCount[char] > 0, "Character %c not found in generated string", char)
	}

	// Verify no unexpected characters
	for char := range charCount {
		assert.Contains(t, expectedChars, string(char), "Unexpected character %c found", char)
	}
}

func TestGeneratorInterface(t *testing.T) {
	var _ Generator = (*generator)(nil)
}

func TestGenerateRandomStringEdgeCases(t *testing.T) {
	logger := log.Initialize(constants.TestAppEnv)
	gen := NewGenerator(logger)
	ctx := context.Background()

	// Test very large length
	largeResult := gen.GenerateRandomString(ctx, 1000000)
	assert.Equal(t, 1000000, len(largeResult))

	// Test single character
	singleResult := gen.GenerateRandomString(ctx, 1)
	assert.Equal(t, 1, len(singleResult))
	assert.Contains(t, "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789", singleResult)
}

func TestGenerateRandomStringPerformance(t *testing.T) {
	logger := log.Initialize(constants.TestAppEnv)
	gen := NewGenerator(logger)
	ctx := context.Background()

	// Test performance with multiple generations
	length := 100
	iterations := 1000

	for i := 0; i < iterations; i++ {
		result := gen.GenerateRandomString(ctx, length)
		assert.Equal(t, length, len(result))
	}
}

func TestGenerateRandomStringConsistency(t *testing.T) {
	logger := log.Initialize(constants.TestAppEnv)
	gen := NewGenerator(logger)
	ctx := context.Background()

	// Test that the same length always produces the expected length
	length := 25
	for i := 0; i < 100; i++ {
		result := gen.GenerateRandomString(ctx, length)
		assert.Equal(t, length, len(result))
	}
}
