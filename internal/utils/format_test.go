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
			expected: "15 January 2024 17:30PM",
		},
		{
			name:     "Format Bangkok time to Bangkok time (no change)",
			input:    time.Date(2024, 1, 15, 17, 30, 0, 0, getBangkokLocation()),
			expected: "15 January 2024 17:30PM",
		},
		{
			name:     "Format New York time to Bangkok time",
			input:    time.Date(2024, 1, 15, 5, 30, 0, 0, getNewYorkLocation()),
			expected: "15 January 2024 17:30PM",
		},
		{
			name:     "Format London time to Bangkok time",
			input:    time.Date(2024, 1, 15, 10, 30, 0, 0, getLondonLocation()),
			expected: "15 January 2024 17:30PM",
		},
		{
			name:     "Format Tokyo time to Bangkok time",
			input:    time.Date(2024, 1, 15, 19, 30, 0, 0, getTokyoLocation()),
			expected: "15 January 2024 17:30PM",
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
			result := FormatToBangkokFullTimeFormat(tt.input)
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
			result := FormatToBangkokFullTimeFormat(tt.input)
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
			expected: "1 January 1900 18:42PM",
		},
		{
			name:     "Format very future date",
			input:    time.Date(2100, 12, 31, 12, 0, 0, 0, time.UTC),
			expected: "31 December 2100 19:00PM",
		},
		{
			name:     "Format with zero time",
			input:    time.Time{},
			expected: "1 January 0001 06:42AM",
		},
		{
			name:     "Format with negative year (should handle gracefully)",
			input:    time.Date(-100, 1, 1, 12, 0, 0, 0, time.UTC),
			expected: "1 January -0100 18:42PM",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatToBangkokFullTimeFormat(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatToBangkokTimeConsistency(t *testing.T) {
	input := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	expected := "15 January 2024 17:30PM"

	for i := 0; i < 100; i++ {
		result := FormatToBangkokFullTimeFormat(input)
		assert.Equal(t, expected, result)
	}
}

func TestFormatToBangkokTimePerformance(t *testing.T) {
	input := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	for i := 0; i < 10000; i++ {
		_ = FormatToBangkokFullTimeFormat(input)
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
			result := FormatToBangkokFullTimeFormat(input)

			assert.Contains(t, result, month.String())
			assert.Contains(t, result, "19:00PM")
		})
	}
}

func TestFormatToBangkokTimeDifferentHours(t *testing.T) {
	for hour := 0; hour < 24; hour++ {
		t.Run(fmt.Sprintf("Hour_%d", hour), func(t *testing.T) {
			input := time.Date(2024, 1, 15, hour, 0, 0, 0, time.UTC)
			result := FormatToBangkokFullTimeFormat(input)

			if hour >= 12 && hour <= 16 {
				assert.Contains(t, result, "PM")
			} else if hour >= 17 {
				assert.Contains(t, result, "AM")
			} else {
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

func TestFormatToUTCString(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Time
		expected string
	}{
		{
			name:     "Format UTC time to UTC string",
			input:    time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC),
			expected: "2024-01-15T10:30:45Z",
		},
		{
			name:     "Format Bangkok time to UTC string",
			input:    time.Date(2024, 1, 15, 17, 30, 45, 0, time.FixedZone("Bangkok", 7*3600)),
			expected: "2024-01-15T10:30:45Z",
		},
		{
			name:     "Format with zero time",
			input:    time.Time{},
			expected: "0001-01-01T00:00:00Z",
		},
		{
			name:     "Format with milliseconds",
			input:    time.Date(2024, 1, 15, 10, 30, 45, 123456789, time.UTC),
			expected: "2024-01-15T10:30:45Z",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatToUTCString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

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

func TestFormatToBangkokTimeWithSecond(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Time
		expected string
	}{
		{
			name:     "Format UTC time to Bangkok time with seconds",
			input:    time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC),
			expected: "15/01/2024 17:30:45",
		},
		{
			name:     "Format Bangkok time to Bangkok time with seconds (no change)",
			input:    time.Date(2024, 1, 15, 17, 30, 45, 0, getBangkokLocation()),
			expected: "15/01/2024 17:30:45",
		},
		{
			name:     "Format New York time to Bangkok time with seconds",
			input:    time.Date(2024, 1, 15, 5, 30, 45, 0, getNewYorkLocation()),
			expected: "15/01/2024 17:30:45",
		},
		{
			name:     "Format London time to Bangkok time with seconds",
			input:    time.Date(2024, 1, 15, 10, 30, 45, 0, getLondonLocation()),
			expected: "15/01/2024 17:30:45",
		},
		{
			name:     "Format Tokyo time to Bangkok time with seconds",
			input:    time.Date(2024, 1, 15, 19, 30, 45, 0, getTokyoLocation()),
			expected: "15/01/2024 17:30:45",
		},
		{
			name:     "Format midnight time with seconds",
			input:    time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
			expected: "15/01/2024 07:00:00",
		},
		{
			name:     "Format noon time with seconds",
			input:    time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
			expected: "15/01/2024 19:00:00",
		},
		{
			name:     "Format end of day time with seconds",
			input:    time.Date(2024, 1, 15, 23, 59, 59, 0, time.UTC),
			expected: "16/01/2024 06:59:59",
		},
		{
			name:     "Format leap year date with seconds",
			input:    time.Date(2024, 2, 29, 15, 30, 45, 0, time.UTC),
			expected: "29/02/2024 22:30:45",
		},
		{
			name:     "Format year boundary with seconds",
			input:    time.Date(2023, 12, 31, 23, 59, 59, 0, time.UTC),
			expected: "01/01/2024 06:59:59",
		},
		{
			name:     "Format with milliseconds (should truncate)",
			input:    time.Date(2024, 1, 15, 10, 30, 45, 500000000, time.UTC),
			expected: "15/01/2024 17:30:45",
		},
		{
			name:     "Format early morning time",
			input:    time.Date(2024, 1, 15, 2, 15, 30, 0, time.UTC),
			expected: "15/01/2024 09:15:30",
		},
		{
			name:     "Format late evening time",
			input:    time.Date(2024, 1, 15, 20, 45, 15, 0, time.UTC),
			expected: "16/01/2024 03:45:15",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatBangkokDateTimeFormat(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatToBangkokTimeWithSecondEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Time
		expected string
	}{
		{
			name:     "Format very old date with seconds",
			input:    time.Date(1900, 1, 1, 12, 0, 0, 0, time.UTC),
			expected: "01/01/1900 18:42:04",
		},
		{
			name:     "Format very future date with seconds",
			input:    time.Date(2100, 12, 31, 12, 0, 0, 0, time.UTC),
			expected: "31/12/2100 19:00:00",
		},
		{
			name:     "Format with zero time",
			input:    time.Time{},
			expected: "01/01/0001 06:42:04",
		},
		{
			name:     "Format with negative year (should handle gracefully)",
			input:    time.Date(-100, 1, 1, 12, 0, 0, 0, time.UTC),
			expected: "01/01/-0100 18:42:04",
		},
		{
			name:     "Format with maximum seconds",
			input:    time.Date(2024, 1, 15, 10, 30, 59, 0, time.UTC),
			expected: "15/01/2024 17:30:59",
		},
		{
			name:     "Format with zero seconds",
			input:    time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
			expected: "15/01/2024 17:30:00",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatBangkokDateTimeFormat(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatToBangkokTimeWithSecondConsistency(t *testing.T) {
	input := time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC)
	expected := "15/01/2024 17:30:45"

	for i := 0; i < 100; i++ {
		result := FormatBangkokDateTimeFormat(input)
		assert.Equal(t, expected, result)
	}
}

func TestFormatToBangkokTimeWithSecondPerformance(t *testing.T) {
	input := time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC)

	for i := 0; i < 10000; i++ {
		_ = FormatBangkokDateTimeFormat(input)
	}
}

func TestFormatToBangkokTimeWithSecondDifferentMonths(t *testing.T) {
	months := []time.Month{
		time.January, time.February, time.March, time.April,
		time.May, time.June, time.July, time.August,
		time.September, time.October, time.November, time.December,
	}

	for _, month := range months {
		t.Run(month.String(), func(t *testing.T) {
			input := time.Date(2024, month, 15, 12, 0, 30, 0, time.UTC)
			result := FormatBangkokDateTimeFormat(input)

			// Check that the month is properly formatted as MM
			expectedMonth := fmt.Sprintf("%02d", int(month))
			assert.Contains(t, result, expectedMonth)
			assert.Contains(t, result, "19:00:30")
		})
	}
}

func TestFormatToBangkokTimeWithSecondDifferentSeconds(t *testing.T) {
	for second := 0; second < 60; second++ {
		t.Run(fmt.Sprintf("Second_%d", second), func(t *testing.T) {
			input := time.Date(2024, 1, 15, 10, 30, second, 0, time.UTC)
			result := FormatBangkokDateTimeFormat(input)

			expectedSecond := fmt.Sprintf("%02d", second)
			assert.Contains(t, result, expectedSecond)
		})
	}
}

func TestFormatToBangkokTimeWithSecondDifferentHours(t *testing.T) {
	for hour := 0; hour < 24; hour++ {
		t.Run(fmt.Sprintf("Hour_%d", hour), func(t *testing.T) {
			input := time.Date(2024, 1, 15, hour, 0, 0, 0, time.UTC)
			result := FormatBangkokDateTimeFormat(input)

			bangkokHour := hour + 7
			if bangkokHour >= 24 {
				bangkokHour -= 24
			}
			expectedHour := fmt.Sprintf("%02d", bangkokHour)
			assert.Contains(t, result, expectedHour)
		})
	}
}

func TestGetStatusColor(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected string
	}{
		{
			name:     "Pending status",
			status:   constants.StatusPending,
			expected: "#17A2B8", // Gray
		},
		{
			name:     "OnGoing status",
			status:   constants.StatusOnGoing,
			expected: "#007BFF", // Blue
		},
		{
			name:     "Completed status",
			status:   constants.StatusCompleted,
			expected: "#28A745", // Green
		},
		{
			name:     "Aborted status",
			status:   constants.StatusAborted,
			expected: "#DC3545", // Red
		},
		{
			name:     "Cancelled status",
			status:   constants.StatusCancelled,
			expected: "#FFC107", // Gray
		},
		{
			name:     "TimedOut status",
			status:   constants.StatusTimedOut,
			expected: "#FF6B35", // Orange-red
		},
		{
			name:     "Unknown status",
			status:   "unknown_status",
			expected: "#6C757D", // Default gray
		},
		{
			name:     "Empty status",
			status:   "",
			expected: "#6C757D", // Default gray
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetStatusColor(tt.status)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatScorePercentage(t *testing.T) {
	tests := []struct {
		name     string
		score    float64
		expected string
	}{
		{
			name:     "Perfect score (5.0)",
			score:    5.0,
			expected: "100.0%",
		},
		{
			name:     "High score (4.5)",
			score:    4.5,
			expected: "90.0%",
		},
		{
			name:     "Medium score (3.75)",
			score:    3.75,
			expected: "75.0%",
		},
		{
			name:     "Low score (2.25)",
			score:    2.25,
			expected: "45.0%",
		},
		{
			name:     "Zero score",
			score:    0.0,
			expected: "0.0%",
		},
		{
			name:     "Negative score (should clamp to 0)",
			score:    -0.5,
			expected: "0.0%",
		},
		{
			name:     "Score over 5 (should clamp to 5)",
			score:    7.5,
			expected: "100.0%",
		},
		{
			name:     "Decimal score",
			score:    4.35,
			expected: "87.0%",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatScorePercentage(tt.score)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetScoreColor(t *testing.T) {
	tests := []struct {
		name     string
		score    float64
		expected string
	}{
		{
			name:     "Excellent score (4.5+)",
			score:    4.75,
			expected: "#28A745", // Green
		},
		{
			name:     "Excellent score (exactly 4.5)",
			score:    4.5,
			expected: "#28A745", // Green
		},
		{
			name:     "Good score (4.0-4.49)",
			score:    4.25,
			expected: "#28A745", // Green (85%)
		},
		{
			name:     "Good score (exactly 4.0)",
			score:    4.0,
			expected: "#28A745", // Green (80%)
		},
		{
			name:     "Fair score (3.5-3.99)",
			score:    3.75,
			expected: "#20C997", // Teal (75%)
		},
		{
			name:     "Fair score (exactly 3.5)",
			score:    3.5,
			expected: "#20C997", // Teal (70%)
		},
		{
			name:     "Below average score (3.0-3.49)",
			score:    3.25,
			expected: "#20C997", // Teal (65%)
		},
		{
			name:     "Below average score (exactly 3.0)",
			score:    3.0,
			expected: "#20C997", // Teal (60%)
		},
		{
			name:     "Poor score (below 3.0)",
			score:    2.25,
			expected: "#FFC107", // Yellow (45%)
		},
		{
			name:     "Zero score",
			score:    0.0,
			expected: "#DC3545", // Red
		},
		{
			name:     "Negative score",
			score:    -0.5,
			expected: "#DC3545", // Red
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetScoreColor(tt.score)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatScoreWithColor(t *testing.T) {
	tests := []struct {
		name           string
		score          float64
		expectedFormat string
		expectedColor  string
	}{
		{
			name:           "Excellent score",
			score:          4.775,
			expectedFormat: "95.5%",
			expectedColor:  "#28A745",
		},
		{
			name:           "Good score",
			score:          4.25,
			expectedFormat: "85.0%",
			expectedColor:  "#28A745",
		},
		{
			name:           "Fair score",
			score:          3.76,
			expectedFormat: "75.2%",
			expectedColor:  "#20C997",
		},
		{
			name:           "Below average score",
			score:          3.25,
			expectedFormat: "65.0%",
			expectedColor:  "#20C997",
		},
		{
			name:           "Poor score",
			score:          2.25,
			expectedFormat: "45.0%",
			expectedColor:  "#FFC107",
		},
		{
			name:           "Zero score",
			score:          0.0,
			expectedFormat: "0.0%",
			expectedColor:  "#DC3545",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			format, color := FormatScoreWithColor(tt.score)
			assert.Equal(t, tt.expectedFormat, format)
			assert.Equal(t, tt.expectedColor, color)
		})
	}
}

func TestFormatScoreWithColorConsistency(t *testing.T) {
	score := 4.275
	expectedFormat := "85.5%"
	expectedColor := "#28A745" // 85.5% >= 80% = Green

	for i := 0; i < 100; i++ {
		format, color := FormatScoreWithColor(score)
		assert.Equal(t, expectedFormat, format)
		assert.Equal(t, expectedColor, color)
	}
}

func TestFormatScoreWithColorEdgeCases(t *testing.T) {
	tests := []struct {
		name           string
		score          float64
		expectedFormat string
		expectedColor  string
	}{
		{
			name:           "Very high score",
			score:          4.999,
			expectedFormat: "100.0%",
			expectedColor:  "#28A745",
		},
		{
			name:           "Very low score",
			score:          0.001,
			expectedFormat: "0.0%",
			expectedColor:  "#DC3545",
		},
		{
			name:           "Boundary score (exactly 4.5)",
			score:          4.5,
			expectedFormat: "90.0%",
			expectedColor:  "#28A745",
		},
		{
			name:           "Boundary score (exactly 4.0)",
			score:          4.0,
			expectedFormat: "80.0%",
			expectedColor:  "#28A745",
		},
		{
			name:           "Boundary score (exactly 3.5)",
			score:          3.5,
			expectedFormat: "70.0%",
			expectedColor:  "#20C997",
		},
		{
			name:           "Boundary score (exactly 3.0)",
			score:          3.0,
			expectedFormat: "60.0%",
			expectedColor:  "#20C997",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			format, color := FormatScoreWithColor(tt.score)
			assert.Equal(t, tt.expectedFormat, format)
			assert.Equal(t, tt.expectedColor, color)
		})
	}
}

func TestValidateScoreRange(t *testing.T) {
	tests := []struct {
		name     string
		score    float64
		expected bool
	}{
		{
			name:     "Valid score within range",
			score:    3.75,
			expected: true,
		},
		{
			name:     "Valid score at minimum",
			score:    0.0,
			expected: true,
		},
		{
			name:     "Valid score at maximum",
			score:    5.0,
			expected: true,
		},
		{
			name:     "Invalid negative score",
			score:    -0.5,
			expected: false,
		},
		{
			name:     "Invalid score over 5",
			score:    7.5,
			expected: false,
		},
		{
			name:     "Edge case: very small positive score",
			score:    0.001,
			expected: true,
		},
		{
			name:     "Edge case: score just under 5",
			score:    4.999,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateScoreRange(tt.score)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected bool
	}{
		{
			name:     "Valid pending status",
			status:   constants.StatusPending,
			expected: true,
		},
		{
			name:     "Valid ongoing status",
			status:   constants.StatusOnGoing,
			expected: true,
		},
		{
			name:     "Valid completed status",
			status:   constants.StatusCompleted,
			expected: true,
		},
		{
			name:     "Valid aborted status",
			status:   constants.StatusAborted,
			expected: true,
		},
		{
			name:     "Valid cancelled status",
			status:   constants.StatusCancelled,
			expected: true,
		},
		{
			name:     "Valid timed out status",
			status:   constants.StatusTimedOut,
			expected: true,
		},
		{
			name:     "Invalid empty status",
			status:   "",
			expected: false,
		},
		{
			name:     "Invalid unknown status",
			status:   "unknown_status",
			expected: false,
		},
		{
			name:     "Invalid case-sensitive status",
			status:   "PENDING",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateStatus(tt.status)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetStatusDisplayName(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected string
	}{
		{
			name:     "Pending status display",
			status:   constants.StatusPending,
			expected: "Pending",
		},
		{
			name:     "OnGoing status display",
			status:   constants.StatusOnGoing,
			expected: "In Progress",
		},
		{
			name:     "Completed status display",
			status:   constants.StatusCompleted,
			expected: "Completed",
		},
		{
			name:     "Aborted status display",
			status:   constants.StatusAborted,
			expected: "Aborted",
		},
		{
			name:     "Cancelled status display",
			status:   constants.StatusCancelled,
			expected: "Cancelled",
		},
		{
			name:     "TimedOut status display",
			status:   constants.StatusTimedOut,
			expected: "Timed Out",
		},
		{
			name:     "Unknown status display",
			status:   "unknown_status",
			expected: "Unknown",
		},
		{
			name:     "Empty status display",
			status:   "",
			expected: "Unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetStatusDisplayName(tt.status)
			assert.Equal(t, tt.expected, result)
		})
	}
}
