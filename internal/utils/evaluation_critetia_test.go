package utils

import (
	"testing"
)

func TestConvertWeightToPercentage(t *testing.T) {
	tests := []struct {
		name     string
		weight   string
		expected string
	}{
		{
			name:     "valid decimal weight",
			weight:   "0.25",
			expected: "25.0",
		},
		{
			name:     "valid decimal weight with more precision",
			weight:   "0.333",
			expected: "33.3",
		},
		{
			name:     "zero weight",
			weight:   "0",
			expected: "0.0",
		},
		{
			name:     "one weight",
			weight:   "1",
			expected: "100.0",
		},
		{
			name:     "small decimal weight",
			weight:   "0.1",
			expected: "10.0",
		},
		{
			name:     "invalid weight string",
			weight:   "invalid",
			expected: "0.0",
		},
		{
			name:     "empty weight string",
			weight:   "",
			expected: "0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertWeightToPercentage(tt.weight)
			if result != tt.expected {
				t.Errorf("ConvertWeightToPercentage(%s) = %s, expected %s", tt.weight, result, tt.expected)
			}
		})
	}
}

func TestGetPercentageColor(t *testing.T) {
	tests := []struct {
		name       string
		percentage string
		expected   string
	}{
		{
			name:       "low percentage - red",
			percentage: "15.0",
			expected:   "#DC3545",
		},
		{
			name:       "exactly 20 percent - red",
			percentage: "20.0",
			expected:   "#DC3545",
		},
		{
			name:       "medium percentage - yellow",
			percentage: "25.0",
			expected:   "#FFC107",
		},
		{
			name:       "exactly 30 percent - yellow",
			percentage: "30.0",
			expected:   "#FFC107",
		},
		{
			name:       "high percentage - green",
			percentage: "35.0",
			expected:   "#28A745",
		},
		{
			name:       "very high percentage - green",
			percentage: "85.5",
			expected:   "#28A745",
		},
		{
			name:       "invalid percentage string",
			percentage: "invalid",
			expected:   "#DC3545",
		},
		{
			name:       "empty percentage string",
			percentage: "",
			expected:   "#DC3545",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetPercentageColor(tt.percentage)
			if result != tt.expected {
				t.Errorf("GetPercentageColor(%s) = %s, expected %s", tt.percentage, result, tt.expected)
			}
		})
	}
}
