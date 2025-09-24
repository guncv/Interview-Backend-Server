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

// ParseAndFormatDurationSince parses dateAt using RFC3339Nano and returns the duration
// between the parsed time and startTime formatted as "MM:SS".
//
// If dateAt cannot be parsed, the parse error is returned.
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

// FormatNullableTimeToBangkokString returns the given sql.NullTime formatted in Bangkok date-time.
// If nullTime is valid it formats nullTime.Time; otherwise it formats the current time.
// The output uses the same layout as FormatBangkokDateTimeFormat (e.g. "02/01/2006 15:04:05").
func FormatNullableTimeToBangkokString(nullTime sql.NullTime) string {
	if nullTime.Valid {
		return FormatBangkokDateTimeFormat(nullTime.Time)
	}
	return FormatBangkokDateTimeFormat(time.Now())
}

// FormatNullableTimeToBangkokStringFullTimeFormat formats a sql.NullTime into Bangkok full-time string.
// 
// If nullTime.Valid is true, it formats nullTime.Time in the Bangkok timezone using the "2 January 2006 15:04PM" full-time format.
// If nullTime.Valid is false, it formats the current time in Bangkok using the same format.
func FormatNullableTimeToBangkokStringFullTimeFormat(nullTime sql.NullTime) string {
	if nullTime.Valid {
		return FormatToBangkokFullTimeFormat(nullTime.Time)
	}
	return FormatToBangkokFullTimeFormat(time.Now())
}

// FormatNullableTimeToUTCString formats a sql.NullTime as a UTC timestamp string.
// If nullTime.Valid is true it returns the time formatted as "2006-01-02T15:04:05Z"; otherwise it returns an empty string.
func FormatNullableTimeToUTCString(nullTime sql.NullTime) string {
	if nullTime.Valid {
		return FormatToUTCString(nullTime.Time)
	}
	return ""
}

// GetNullableFloat64 returns the underlying float64 value from a sql.NullFloat64,
// or defaultValue if nullFloat is not valid.
func GetNullableFloat64(nullFloat sql.NullFloat64, defaultValue float64) float64 {
	if nullFloat.Valid {
		return nullFloat.Float64
	}
	return defaultValue
}

// GetNullableString returns the underlying string from a sql.NullString when valid;
// otherwise it returns the provided defaultValue.
func GetNullableString(nullString sql.NullString, defaultValue string) string {
	if nullString.Valid {
		return nullString.String
	}
	return defaultValue
}

// GetStatusColor returns the hex color code associated with a task status.
// Known mappings:
//   - constants.StatusPending / constants.StatusCancelled -> "#6C757D" (gray)
//   - constants.StatusOnGoing -> "#007BFF" (blue)
//   - constants.StatusCompleted -> "#28A745" (green)
//   - constants.StatusAborted -> "#DC3545" (red)
//   - constants.StatusTimedOut -> "#FF6B35" (orange)
// For any unrecognized status the function returns the default gray "#6C757D".
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

// FormatScorePercentage clamps score to the [0,5] range, converts it to a percentage
// on a 0–100 scale (score * 20), and returns the value formatted with one decimal
// place and a trailing percent sign (e.g., "80.0%").
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

// otherwise -> "#DC3545".
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

// FormatScoreWithColor returns the score as a formatted percentage string and its corresponding color.
// The first return is the score converted from a 0–5 scale into a percentage string (one decimal place, e.g. "80.0%").
// The second return is a hex color code representing the score's range.
func FormatScoreWithColor(score float64) (string, string) {
	percentage := FormatScorePercentage(score)
	color := GetScoreColor(score)
	return percentage, color
}

// ValidateScoreRange reports whether score is within the allowed rating range 0–5 (inclusive).
func ValidateScoreRange(score float64) bool {
	return score >= 0 && score <= 5
}

// ValidateStatus reports whether the provided status string matches one of the recognized status constants.
  
// It returns true if status equals one of: StatusPending, StatusOnGoing, StatusCompleted,
// StatusAborted, StatusCancelled, or StatusTimedOut; otherwise it returns false.
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

// If the provided status is not recognized, it returns "Unknown".
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
