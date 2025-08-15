package constants

import (
	"errors"
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

// User Role Constants
const (
	UserRoleUser  UserRole = "user"
	UserRoleAdmin UserRole = "admin"
)

// Publish Constants
const (
	CanPublish      = true
	CannotPublish   = false
	CanSubscribe    = true
	CannotSubscribe = false
)

// Auth Constants
var (
	AuthorizationHeaderKey  ContextKey = "authorization"
	AuthorizationTypeBearer ContextKey = "bearer"
	AuthorizationPayloadKey ContextKey = "authorization_payload"
	RefreshTokenCookieKey   ContextKey = "refresh_token"
	NewAccessTokenKey       ContextKey = "new_access_token"
	XAccessTokenHeaderKey   ContextKey = "X-Access-Token"
	AuthContextKey          ContextKey = "auth_context"
	RoleKey                 ContextKey = "x-active-role"
)

// Error Response Messages
var (
	ErrCategoryIDRequired           = errors.New("category id is required")
	ErrExpiredToken                 = errors.New("token has expired")
	ErrInvalidToken                 = errors.New("token is invalid")
	ErrInvalidRole                  = errors.New("invalid role")
	ErrCategoryNotFound             = errors.New("category not found")
	ErrCourseNotFound               = errors.New("course not found")
	ErrCourseAlreadyExists          = errors.New("course already exists")
	ErrOrganizationNotFound         = errors.New("organization not found")
	ErrCourseSectionNotFound        = errors.New("course section not found")
	ErrCourseSectionAlreadyExists   = errors.New("course section already exists")
	ErrCourseSectionInvalidRequest  = errors.New("course section invalid request")
	ErrSectionContentNotFound       = errors.New("section content not found")
	ErrSectionContentAlreadyExists  = errors.New("section content already exists")
	ErrSectionContentInvalidRequest = errors.New("section content invalid request")
)

// Email Constants
var (
	TaskSendResetPasswordEmail = "task:send_reset_password_email"
	TaskSendVerifyEmail        = "task:send_verify_email"
	QueueCritical              = "critical"
	QueueDefault               = "default"
	MaxRetry                   = 10

	OptionResetPasswordEmail = "option_reset_password_email"

	SubjectResetPassword = "Reset your password"
	SubjectVerifyEmail   = "Verify your email"

	CriticalQueueConcurrency = 10
	DefaultQueueConcurrency  = 5
	DefaultConcurrency       = 10
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
)

// S3 Constants
const (
	S3CourseThumbnailKey = "course/thumbnail"
)
