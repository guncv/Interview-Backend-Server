package utils

import (
	"database/sql"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestRoundToTwoDecimalPlaces(t *testing.T) {
	tests := []struct {
		name     string
		val      float64
		expected float64
	}{
		{name: "Round to two decimal places", val: 1.2345, expected: 1.23},
		{name: "Round to two decimal places", val: 1.2, expected: 1.2},
		{name: "Round to two decimal places", val: 1, expected: 1},
		{name: "Round to two decimal places", val: 0.9999, expected: 1},
		{name: "Round to two decimal places", val: 0.4, expected: 0.4},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, RoundToTwoDecimalPlaces(test.val), test.expected)
		})
	}
}

func TestSafeNullUUIDToString(t *testing.T) {
	tests := []struct {
		name     string
		val      sql.NullString
		expected string
	}{
		{name: "SafeNullUUIDToString", val: sql.NullString{String: "123e4567-e89b-12d3-a456-426614174000", Valid: true}, expected: "123e4567-e89b-12d3-a456-426614174000"},
		{name: "SafeNullUUIDToString", val: sql.NullString{String: "", Valid: false}, expected: ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, SafeNullUUIDToString(test.val), test.expected)
		})
	}
}

func TestSafeCustomNullUUIDToString(t *testing.T) {
	tests := []struct {
		name     string
		val      NullUUID
		expected string
	}{
		{name: "SafeCustomNullUUIDToString", val: NullUUID{UUID: uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"), Valid: true}, expected: "123e4567-e89b-12d3-a456-426614174000"},
		{name: "SafeCustomNullUUIDToString", val: NullUUID{UUID: uuid.Nil, Valid: false}, expected: ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, SafeCustomNullUUIDToString(test.val), test.expected)
		})
	}
}
