package utils

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
)

func TestFormatToBangkokTime(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Time
		expected string
	}{
		{
			name:     "Format UTC time to Bangkok time",
			input:    time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
			expected: "15 January 2024 17:30PM", // UTC+7
		},
		{
			name:     "Format Bangkok time to Bangkok time (no change)",
			input:    time.Date(2024, 1, 15, 17, 30, 0, 0, getBangkokLocation()),
			expected: "15 January 2024 17:30PM",
		},
		{
			name:     "Format New York time to Bangkok time",
			input:    time.Date(2024, 1, 15, 5, 30, 0, 0, getNewYorkLocation()),
			expected: "15 January 2024 17:30PM", // EST+12
		},
		{
			name:     "Format London time to Bangkok time",
			input:    time.Date(2024, 1, 15, 10, 30, 0, 0, getLondonLocation()),
			expected: "15 January 2024 17:30PM", // GMT+7
		},
		{
			name:     "Format Tokyo time to Bangkok time",
			input:    time.Date(2024, 1, 15, 19, 30, 0, 0, getTokyoLocation()),
			expected: "15 January 2024 17:30PM", // JST-2
		},
		{
			name:     "Format midnight time",
			input:    time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
			expected: "15 January 2024 07:00AM",
		},
		{
			name:     "Format noon time",
			input:    time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
			expected: "15 January 2024 19:00PM",
		},
		{
			name:     "Format end of day time",
			input:    time.Date(2024, 1, 15, 23, 59, 59, 0, time.UTC),
			expected: "16 January 2024 06:59AM",
		},
		{
			name:     "Format leap year date",
			input:    time.Date(2024, 2, 29, 15, 30, 0, 0, time.UTC),
			expected: "29 February 2024 22:30PM",
		},
		{
			name:     "Format year boundary",
			input:    time.Date(2023, 12, 31, 23, 59, 59, 0, time.UTC),
			expected: "1 January 2024 06:59AM",
		},
		{
			name:     "Format with milliseconds",
			input:    time.Date(2024, 1, 15, 10, 30, 0, 500000000, time.UTC),
			expected: "15 January 2024 17:30PM",
		},
	}

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
			name:     "Format UTC time to Bangkok time",
			input:    time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
			expected: "15 January 2024 17:30PM",
		},
		{
			name:     "Format UTC time with different date",
			input:    time.Date(2024, 6, 20, 14, 45, 30, 0, time.UTC),
			expected: "20 June 2024 21:45PM",
		},
		{
			name:     "Format UTC time with AM time",
			input:    time.Date(2024, 1, 15, 2, 15, 0, 0, time.UTC),
			expected: "15 January 2024 09:15AM",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatToBangkokTimeFromUTC(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatToBangkokTimeEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Time
		expected string
	}{
		{
			name:     "Format very old date",
			input:    time.Date(1900, 1, 1, 12, 0, 0, 0, time.UTC),
			expected: "1 January 1900 18:42PM", // UTC+6:42 for Bangkok in 1900
		},
		{
			name:     "Format very future date",
			input:    time.Date(2100, 12, 31, 12, 0, 0, 0, time.UTC),
			expected: "31 December 2100 19:00PM",
		},
		{
			name:     "Format with zero time",
			input:    time.Time{},
			expected: "1 January 0001 06:42AM", // UTC+6:42 for Bangkok in year 1
		},
		{
			name:     "Format with negative year (should handle gracefully)",
			input:    time.Date(-100, 1, 1, 12, 0, 0, 0, time.UTC),
			expected: "1 January -0100 18:42PM", // UTC+6:42 for Bangkok in -100
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatToBangkokTime(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatToBangkokTimeConsistency(t *testing.T) {
	// Test that the same input always produces the same output
	input := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	expected := "15 January 2024 17:30PM"

	for i := 0; i < 100; i++ {
		result := FormatToBangkokTime(input)
		assert.Equal(t, expected, result)
	}
}

func TestFormatToBangkokTimePerformance(t *testing.T) {
	input := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	// Test performance with multiple calls
	for i := 0; i < 10000; i++ {
		_ = FormatToBangkokTime(input)
	}
}

func TestFormatToBangkokTimeDifferentMonths(t *testing.T) {
	months := []time.Month{
		time.January, time.February, time.March, time.April,
		time.May, time.June, time.July, time.August,
		time.September, time.October, time.November, time.December,
	}

	for _, month := range months {
		t.Run(month.String(), func(t *testing.T) {
			input := time.Date(2024, month, 15, 12, 0, 0, 0, time.UTC)
			result := FormatToBangkokTime(input)

			// Verify the month is correctly formatted
			assert.Contains(t, result, month.String())
			assert.Contains(t, result, "19:00PM") // 12:00 UTC + 7 hours
		})
	}
}

func TestFormatToBangkokTimeDifferentHours(t *testing.T) {
	for hour := 0; hour < 24; hour++ {
		t.Run(fmt.Sprintf("Hour_%d", hour), func(t *testing.T) {
			input := time.Date(2024, 1, 15, hour, 0, 0, 0, time.UTC)
			result := FormatToBangkokTime(input)

			// Check if AM/PM is correct based on actual Bangkok time
			// UTC 0-11 = Bangkok 7-18 (AM/PM depending on hour)
			// UTC 12-16 = Bangkok 19-23 (PM)
			// UTC 17-23 = Bangkok 0-6 (AM)
			if hour >= 12 && hour <= 16 { // UTC 12-16 = Bangkok 19-23 (PM)
				assert.Contains(t, result, "PM")
			} else if hour >= 17 { // UTC 17-23 = Bangkok 0-6 (AM)
				assert.Contains(t, result, "AM")
			} else { // UTC 0-11 = Bangkok 7-18 (AM/PM depending on hour)
				// For UTC 0-11, Bangkok time is 7-18
				// 7-11 = AM, 12-18 = PM
				bangkokHour := hour + 7
				if bangkokHour >= 12 {
					assert.Contains(t, result, "PM")
				} else {
					assert.Contains(t, result, "AM")
				}
			}
		})
	}
}

// Helper functions to get different timezone locations
func getBangkokLocation() *time.Location {
	loc, _ := time.LoadLocation(constants.BangkokTimezone)
	if loc == nil {
		return time.UTC
	}
	return loc
}

func getNewYorkLocation() *time.Location {
	loc, _ := time.LoadLocation("America/New_York")
	if loc == nil {
		return time.UTC
	}
	return loc
}

func getLondonLocation() *time.Location {
	loc, _ := time.LoadLocation("Europe/London")
	if loc == nil {
		return time.UTC
	}
	return loc
}

func getTokyoLocation() *time.Location {
	loc, _ := time.LoadLocation("Asia/Tokyo")
	if loc == nil {
		return time.UTC
	}
	return loc
}
