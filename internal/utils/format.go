package utils

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
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

func FormatNullableTimeToBangkokStringFullTimeFormat(nullTime sql.NullTime) string {
	if nullTime.Valid {
		return FormatToBangkokFullTimeFormat(nullTime.Time)
	}
	return FormatToBangkokFullTimeFormat(time.Now())
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

func GetStatusColor(status string) string {
	switch status {
	case constants.StatusPending:
		return "#6C757D"
	case constants.StatusOnGoing:
		return "#007BFF"
	case constants.StatusCompleted:
		return "#28A745"
	case constants.StatusAborted:
		return "#DC3545"
	case constants.StatusCancelled:
		return "#6C757D"
	case constants.StatusTimedOut:
		return "#FF6B35"
	default:
		return "#6C757D"
	}
}

func FormatScorePercentage(score float64) string {
	if score < 0 {
		score = 0
	}
	if score > 5 {
		score = 5
	}
	// Convert score from 0-5 scale to percentage (multiply by 20)
	percentage := score * 20
	return fmt.Sprintf("%.1f%%", percentage)
}

func GetScoreColor(score float64) string {
	// Convert score from 0-5 scale to percentage for color comparison
	percentage := score * 20
	switch {
	case percentage >= 80:
		return "#28A745"
	case percentage >= 60:
		return "#20C997"
	case percentage >= 40:
		return "#FFC107"
	case percentage >= 20:
		return "#FD7E14"
	default:
		return "#DC3545"
	}
}

func FormatScoreWithColor(score float64) (string, string) {
	percentage := FormatScorePercentage(score)
	color := GetScoreColor(score)
	return percentage, color
}

func ValidateScoreRange(score float64) bool {
	return score >= 0 && score <= 5
}

func ValidateStatus(status string) bool {
	validStatuses := []string{
		constants.StatusPending,
		constants.StatusOnGoing,
		constants.StatusCompleted,
		constants.StatusAborted,
		constants.StatusCancelled,
		constants.StatusTimedOut,
	}

	for _, validStatus := range validStatuses {
		if status == validStatus {
			return true
		}
	}
	return false
}

func GetStatusDisplayName(status string) string {
	switch status {
	case constants.StatusPending:
		return "Pending"
	case constants.StatusOnGoing:
		return "In Progress"
	case constants.StatusCompleted:
		return "Completed"
	case constants.StatusAborted:
		return "Aborted"
	case constants.StatusCancelled:
		return "Cancelled"
	case constants.StatusTimedOut:
		return "Timed Out"
	default:
		return "Unknown"
	}
}

func FormatFloatToTwoDecimals(value float64) string {
	formatted := fmt.Sprintf("%.2f", value)

	if strings.Contains(formatted, ".") {
		formatted = strings.TrimRight(formatted, "0")
		formatted = strings.TrimRight(formatted, ".")
	}

	return formatted
}

func RoundFloatToTwoDecimals(value float64) float64 {
	return float64(int(value*100+0.5)) / 100
}
