package constants

import (
	"net/http"
	"time"
)

type ContextKey string
type UserRole string

// Context Constants
const (
	UserAgentKey ContextKey = "user-agent"
	ClientIPKey  ContextKey = "client-ip"
)

// Token Constants
var (
	TokenGraceWindow = 1 * time.Minute
)

// Test Env
var (
	TestAppEnv = "test"
)

// HTTP Status Codes
const (
	StatusOK                  = http.StatusOK
	StatusBadRequest          = http.StatusBadRequest
	StatusInternalServerError = http.StatusInternalServerError
	StatusUnauthorized        = http.StatusUnauthorized
	StatusNotFound            = http.StatusNotFound
)

// Common Response Messages
const (
	MessagePasswordResetSuccess     = "Password reset successfully"
	MessageSignedOutSuccess         = "Signed out successfully"
	MessageParticipantAlreadyInRoom = "participant already on some room"
	MessagePasswordIncorrect        = "password is incorrect"
)

// Database Error Codes
const (
	PostgresForeignKeyViolation   = "23503"
	PostgresDuplicateKeyViolation = "23505"
)

// Timezone
const (
	DefaultTimezone = "Asia/Shanghai"
	BangkokTimezone = "Asia/Bangkok"
)

// SSL Mode
const (
	SSLModeDisable = "disable"
)

// Timeout content
const (
	TimeoutContext = 10 * time.Second
	TimeoutHTTP    = 15 * time.Second
)

// S3 Constants
const (
	S3ResumeKey       = "resumes"
	S3PresignedURLTTL = 5 * time.Minute
)

// Resume Constants
var (
	ResumeAllowContentTypes = []string{
		"application/pdf",
	}
	ResumeMaxFileSize = 5 * 1024 * 1024 // 5MB
)
