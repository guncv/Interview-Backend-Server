package utils

import (
	"database/sql"
	"math"

	"github.com/google/uuid"
)

func RoundToTwoDecimalPlaces(val float64) float64 {
	return math.Round(val*100) / 100
}

type NullUUID struct {
	UUID  uuid.UUID
	Valid bool
}

func SafeNullUUIDToString(u sql.NullString) string {
	if u.Valid {
		return u.String
	}
	return ""
}

func SafeCustomNullUUIDToString(u NullUUID) string {
	if u.Valid {
		return u.UUID.String()
	}
	return ""
}
