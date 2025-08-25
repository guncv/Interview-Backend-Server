package utils

import (
	"time"

	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
)

func FormatToBangkokTime(t time.Time) string {
	bangkokLoc, err := time.LoadLocation(constants.BangkokTimezone)
	if err != nil {
		bangkokLoc = time.UTC
	}

	bangkokTime := t.In(bangkokLoc)
	return bangkokTime.Format("2 January 2006 15:04PM")
}

func FormatToUTCString(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05Z")
}
