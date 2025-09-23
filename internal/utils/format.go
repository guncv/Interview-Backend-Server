package utils

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
)

func FormatToBangkokFullTimeFormat(t time.Time) string {
	bangkokLoc, err := time.LoadLocation(constants.BangkokTimezone)
	if err != nil {
		bangkokLoc = time.UTC
	}

	bangkokTime := t.In(bangkokLoc)
	return bangkokTime.Format("2 January 2006 15:04PM")
}

func FormatBangkokDateTimeFormat(t time.Time) string {
	bangkokLoc, err := time.LoadLocation(constants.BangkokTimezone)
	if err != nil {
		bangkokLoc = time.UTC
	}

	bangkokTime := t.In(bangkokLoc)
	return bangkokTime.Format("02/01/2006 15:04:05")
}

func FormatToUTCString(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05Z")
}

func ParseToTime(t string) time.Time {
	parsedTime, err := time.Parse(time.RFC3339Nano, t)
	if err != nil {
		return time.Time{}
	}

	return parsedTime
}

func FormatSecondsToMMSS(seconds float64) string {
	minutes := int(seconds) / 60
	secs := int(seconds) % 60
	return fmt.Sprintf("%02d:%02d", minutes, secs)
}

func ParseAndFormatDurationSince(dateAt string, startTime time.Time) (string, error) {
	parsedTime, err := time.Parse(time.RFC3339Nano, dateAt)
	if err != nil {
		log.Println("Failed to parse timestamp:", err)
		return "", err
	}

	duration := parsedTime.Sub(startTime)
	result := FormatSecondsToMMSS(duration.Seconds())
	return result, nil
}

func FormatNullableTimeToBangkokString(nullTime sql.NullTime) string {
	if nullTime.Valid {
		return FormatBangkokDateTimeFormat(nullTime.Time)
	}
	return FormatBangkokDateTimeFormat(time.Now())
}

func FormatNullableTimeToUTCString(nullTime sql.NullTime) string {
	if nullTime.Valid {
		return FormatToUTCString(nullTime.Time)
	}
	return ""
}

func GetNullableFloat64(nullFloat sql.NullFloat64, defaultValue float64) float64 {
	if nullFloat.Valid {
		return nullFloat.Float64
	}
	return defaultValue
}

func GetNullableString(nullString sql.NullString, defaultValue string) string {
	if nullString.Valid {
		return nullString.String
	}
	return defaultValue
}
