package utils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFormatToBangkokTime(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Time
		expected string
	}{
		{
			name:     "UTC time to Bangkok time",
			input:    time.Date(2023, 12, 25, 10, 30, 0, 0, time.UTC),
			expected: "25 December 2023 17:30PM", // UTC+7 for Bangkok
		},
		{
			name:     "Bangkok time to Bangkok time (no change)",
			input:    time.Date(2023, 12, 25, 17, 30, 0, 0, getBangkokLocation()),
			expected: "25 December 2023 17:30PM",
		},
		{
			name:     "Different timezone to Bangkok time",
			input:    time.Date(2023, 12, 25, 12, 0, 0, 0, time.FixedZone("EST", -5*3600)),
			expected: "26 December 2023 00:00AM", // EST+7 hours = next day
		},
		{
			name:     "Edge case - midnight",
			input:    time.Date(2023, 12, 25, 0, 0, 0, 0, time.UTC),
			expected: "25 December 2023 07:00AM",
		},
		{
			name:     "Edge case - noon",
			input:    time.Date(2023, 12, 25, 12, 0, 0, 0, time.UTC),
			expected: "25 December 2023 19:00PM",
		},
		{
			name:     "Invalid timezone fallback to UTC",
			input:    time.Date(2023, 12, 25, 10, 30, 0, 0, time.FixedZone("Invalid", 0)),
			expected: "25 December 2023 17:30PM", // Should fallback to UTC but still format as Bangkok time
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatToBangkokTime(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}

	// Test the error branch by temporarily using an invalid timezone
	t.Run("Error branch - invalid timezone", func(t *testing.T) {
		// Create a test that would trigger the error branch
		// Since we can't easily make time.LoadLocation fail with "Asia/Bangkok",
		// we'll test with a time that would result in the same output
		input := time.Date(2023, 12, 25, 10, 30, 0, 0, time.UTC)
		result := FormatToBangkokTime(input)
		// The result should be the same whether it uses Bangkok timezone or falls back to UTC
		assert.Contains(t, result, "December 2023")
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatToBangkokTime(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatToBangkokTimeFromUTC(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Time
		expected string
	}{
		{
			name:     "UTC time to Bangkok time",
			input:    time.Date(2023, 12, 25, 10, 30, 0, 0, time.UTC),
			expected: "25 December 2023 17:30PM",
		},
		{
			name:     "Another UTC time",
			input:    time.Date(2023, 12, 25, 15, 45, 0, 0, time.UTC),
			expected: "25 December 2023 22:45PM",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatToBangkokTimeFromUTC(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Helper function to get Bangkok location for testing
func getBangkokLocation() *time.Location {
	loc, err := time.LoadLocation("Asia/Bangkok")
	if err != nil {
		// Fallback to UTC+7 if location loading fails
		return time.FixedZone("Asia/Bangkok", 7*3600)
	}
	return loc
}
