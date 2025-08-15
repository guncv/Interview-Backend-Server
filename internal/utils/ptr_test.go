package utils

import (
	"testing"
)

func TestDerefString(t *testing.T) {
	tests := []struct {
		name     string
		input    *string
		expected string
	}{
		{"nil", nil, ""},
		{"empty", stringPtr(""), ""},
		{"value", stringPtr("test"), "test"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DerefString(tt.input)
			if result != tt.expected {
				t.Errorf("DerefString() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestDerefBool(t *testing.T) {
	tests := []struct {
		name     string
		input    *bool
		expected bool
	}{
		{"nil", nil, false},
		{"false", boolPtr(false), false},
		{"true", boolPtr(true), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DerefBool(tt.input)
			if result != tt.expected {
				t.Errorf("DerefBool() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestDerefInt(t *testing.T) {
	tests := []struct {
		name     string
		input    *int
		expected int
	}{
		{"nil", nil, 0},
		{"zero", intPtr(0), 0},
		{"value", intPtr(42), 42},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DerefInt(tt.input)
			if result != tt.expected {
				t.Errorf("DerefInt() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestDerefInt32(t *testing.T) {
	tests := []struct {
		name     string
		input    *int32
		expected int32
	}{
		{"nil", nil, 0},
		{"zero", int32Ptr(0), 0},
		{"value", int32Ptr(42), 42},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DerefInt32(tt.input)
			if result != tt.expected {
				t.Errorf("DerefInt32() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// Helper functions for creating pointers
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
