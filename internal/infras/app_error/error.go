package app_error

import (
	"database/sql"
	"errors"
	"fmt"
	"net"
	"strings"
	"syscall"

	"github.com/lib/pq"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
)

type AppError struct {
	Code    ErrorCode `json:"code"`
	Err     error     `json:"-"`
	Message string    `json:"message"`
}

type AppErrorDetail struct {
	Message           string
	OriginalErrorCode string
	OriginalErrorDesc string
}

func New(err error, code ErrorCode) *AppError {

	if err == nil {
		panic("nil error")
	}

	appErr, ok := err.(*AppError)
	if ok {
		return appErr
	}

	return &AppError{
		Code:    code,
		Err:     err,
		Message: code.Message(),
	}
}

func NewWithCustomMessage(err error, code ErrorCode, customMessage string) *AppError {
	if err == nil {
		panic("nil error")
	}

	appErr, ok := err.(*AppError)
	if ok {
		return appErr
	}

	return &AppError{
		Code:    code,
		Err:     err,
		Message: customMessage,
	}
}

func NewWithMetadata(err error, code ErrorCode, metadata ...map[string]any) *AppError {
	if err == nil {
		panic("nil error")
	}

	mergedMetadata := make(map[string]any)
	for _, m := range metadata {
		for k, v := range m {
			mergedMetadata[k] = v
		}
	}
	return &AppError{
		Code:    code,
		Err:     err,
		Message: code.Message(),
	}
}

func (e *AppError) Error() string {
	return fmt.Sprintf("[%s] %s | %s", e.Code, e.Code.Message(), e.Err.Error())
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func (e *AppError) Is(target error) bool {
	return errors.Is(e.Err, target)
}

func IsForeignKeyViolation(err error) bool {
	if pqErr, ok := err.(*pq.Error); ok {
		return pqErr.Code == constants.PostgresForeignKeyViolation
	}
	return strings.Contains(err.Error(), "violates foreign key constraint")
}

func IsDuplicateKeyViolation(err error) bool {
	if pqErr, ok := err.(*pq.Error); ok {
		return pqErr.Code == constants.PostgresDuplicateKeyViolation
	}
	return strings.Contains(err.Error(), "duplicate key value violates unique constraint")
}

func HandleForeignKeyViolation(err error) *AppError {
	if !IsForeignKeyViolation(err) {
		return New(err, ErrCodeGeneralServerUnavailable)
	}

	if pqErr, ok := err.(*pq.Error); ok {
		constraintName := pqErr.Constraint
		customMessage := getForeignKeyErrorMessage(constraintName)
		if customMessage != "" {
			return NewWithCustomMessage(err, ErrCodeGeneralConstraintViolation, customMessage)
		}
	}

	errorStr := err.Error()
	customMessage := parseConstraintFromErrorMessage(errorStr)
	if customMessage != "" {
		return NewWithCustomMessage(err, ErrCodeGeneralConstraintViolation, customMessage)
	}

	return New(err, ErrCodeGeneralConstraintViolation)
}

func HandleDatabaseError(err error) *AppError {
	if errors.Is(err, sql.ErrNoRows) {
		return New(err, ErrCodeGeneralResourceNotFound)
	}

	if pqErr, ok := err.(*pq.Error); ok {
		return handlePostgreSQLError(pqErr)
	}

	if IsConnectionError(err) {
		return New(err, ErrCodeGeneralDatabaseConnection)
	}

	if IsForeignKeyViolation(err) {
		return HandleForeignKeyViolation(err)
	}

	if IsDuplicateKeyViolation(err) {
		return handleDuplicateKeyViolation(err)
	}

	return New(err, ErrCodeGeneralServerUnavailable)
}

func HandleDatabaseErrorWithContext(err error, context string) *AppError {
	if errors.Is(err, sql.ErrNoRows) {
		return handleNotFoundWithContext(err, context)
	}

	// Handle foreign key violations with context
	if IsForeignKeyViolation(err) {
		return handleForeignKeyViolationWithContext(err, context)
	}

	return HandleDatabaseError(err)
}

func handleForeignKeyViolationWithContext(err error, context string) *AppError {
	if pqErr, ok := err.(*pq.Error); ok {
		constraintName := pqErr.Constraint
		customMessage := getForeignKeyErrorMessageWithContext(constraintName, context)
		if customMessage != "" {
			// Use specific error code for category foreign key violations
			return NewWithCustomMessage(err, ErrCodeGeneralConstraintViolation, customMessage)
		}
	}

	errorStr := err.Error()
	customMessage := parseConstraintFromErrorMessageWithContext(errorStr, context)
	if customMessage != "" {
		// Use specific error code for category foreign key violations
		return NewWithCustomMessage(err, ErrCodeGeneralConstraintViolation, customMessage)
	}

	return HandleForeignKeyViolation(err)
}

func getForeignKeyErrorMessageWithContext(constraintName, context string) string {
	baseMessage := getForeignKeyErrorMessage(constraintName)
	if baseMessage == "" {
		return ""
	}

	// Add context-specific information
	switch strings.ToLower(context) {
	case "create_course":
		if constraintName == "courses_category_id_fkey" {
			return "Cannot create course: " + baseMessage
		}
		if constraintName == "courses_sku_key" {
			return "Cannot create course: " + baseMessage
		}
	case "update_course":
		if constraintName == "courses_category_id_fkey" {
			return "Cannot update course: " + baseMessage
		}
		if constraintName == "courses_sku_key" {
			return "Cannot update course: " + baseMessage
		}
	case "create_category":
		if constraintName == "categories_parent_id_fkey" {
			return "Cannot create category: " + baseMessage
		}
	case "update_category":
		if constraintName == "categories_parent_id_fkey" {
			return "Cannot update category: " + baseMessage
		}
	}

	return baseMessage
}

func parseConstraintFromErrorMessageWithContext(errorMsg, context string) string {
	baseMessage := parseConstraintFromErrorMessage(errorMsg)
	if baseMessage == "" {
		return ""
	}

	// Add context-specific information
	switch strings.ToLower(context) {
	case "create_course":
		if strings.Contains(errorMsg, "courses_category_id_fkey") {
			return "Cannot create course: " + baseMessage
		}
		if strings.Contains(errorMsg, "courses_sku_key") {
			return "Cannot create course: " + baseMessage
		}
	case "update_course":
		if strings.Contains(errorMsg, "courses_category_id_fkey") {
			return "Cannot update course: " + baseMessage
		}
		if strings.Contains(errorMsg, "courses_sku_key") {
			return "Cannot update course: " + baseMessage
		}
	case "create_category":
		if strings.Contains(errorMsg, "categories_parent_id_fkey") {
			return "Cannot create category: " + baseMessage
		}
	case "update_category":
		if strings.Contains(errorMsg, "categories_parent_id_fkey") {
			return "Cannot update category: " + baseMessage
		}
	}

	return baseMessage
}

func handleNotFoundWithContext(err error, context string) *AppError {
	switch strings.ToLower(context) {
	case "user":
		return New(err, ErrCodeAuthUserNotFound)
	case "reset_token", "resettoken", "token":
		return New(err, ErrCodeAuthResetTokenNotFound)
	default:
		return New(err, ErrCodeGeneralResourceNotFound)
	}
}

func handlePostgreSQLError(pqErr *pq.Error) *AppError {
	switch pqErr.Code {
	case "23503":
		return HandleForeignKeyViolation(pqErr)
	case "23505":
		return handleDuplicateKeyViolation(pqErr)
	case "23502":
		return NewWithCustomMessage(pqErr, ErrCodeGeneralConstraintViolation,
			"Required field is missing or empty.")
	case "23514":
		return handleCheckConstraintViolation(pqErr)
	case "23513":
		return NewWithCustomMessage(pqErr, ErrCodeGeneralConstraintViolation,
			"Data validation failed due to constraint violation.")
	case "23512":
		return NewWithCustomMessage(pqErr, ErrCodeGeneralConstraintViolation,
			"Data validation failed due to not null constraint.")
	case "23511":
		return NewWithCustomMessage(pqErr, ErrCodeGeneralConstraintViolation,
			"Data validation failed due to unique constraint.")
	case "23510":
		return NewWithCustomMessage(pqErr, ErrCodeGeneralConstraintViolation,
			"Data validation failed due to exclusion constraint.")
	case "23509":
		return NewWithCustomMessage(pqErr, ErrCodeGeneralConstraintViolation,
			"Data validation failed due to foreign key constraint.")
	case "23508":
		return NewWithCustomMessage(pqErr, ErrCodeGeneralConstraintViolation,
			"Data validation failed due to check constraint.")
	case "23507":
		return NewWithCustomMessage(pqErr, ErrCodeGeneralConstraintViolation,
			"Data validation failed due to not null constraint.")
	case "23506":
		return NewWithCustomMessage(pqErr, ErrCodeGeneralConstraintViolation,
			"Data validation failed due to unique constraint.")
	case "40001":
		return NewWithCustomMessage(pqErr, ErrCodeGeneralServerUnavailable,
			"Request conflicted with another operation. Please try again.")
	case "40P01":
		return NewWithCustomMessage(pqErr, ErrCodeGeneralServerUnavailable,
			"Request conflicted with another operation. Please try again.")
	case "57014":
		return New(pqErr, ErrCodeGeneralRequestTimeout)
	case "08003", "08006", "08001", "08004":
		return New(pqErr, ErrCodeGeneralDatabaseConnection)
	case "42P01":
		return NewWithCustomMessage(pqErr, ErrCodeGeneralServerUnavailable,
			"Database schema error. Please contact support.")
	case "42703":
		return NewWithCustomMessage(pqErr, ErrCodeGeneralServerUnavailable,
			"Database schema error. Please contact support.")
	case "42702":
		return NewWithCustomMessage(pqErr, ErrCodeGeneralServerUnavailable,
			"Database schema error. Please contact support.")
	case "42701":
		return NewWithCustomMessage(pqErr, ErrCodeGeneralServerUnavailable,
			"Database schema error. Please contact support.")
	case "42601":
		return NewWithCustomMessage(pqErr, ErrCodeGeneralServerUnavailable,
			"Database syntax error. Please contact support.")
	case "42501":
		return NewWithCustomMessage(pqErr, ErrCodeGeneralServerUnavailable,
			"Database permission error. Please contact support.")
	case "42401":
		return NewWithCustomMessage(pqErr, ErrCodeGeneralServerUnavailable,
			"Database permission error. Please contact support.")
	case "42000":
		return NewWithCustomMessage(pqErr, ErrCodeGeneralServerUnavailable,
			"Database syntax error. Please contact support.")
	case "3F000":
		return NewWithCustomMessage(pqErr, ErrCodeGeneralServerUnavailable,
			"Database schema error. Please contact support.")
	case "3D000":
		return NewWithCustomMessage(pqErr, ErrCodeGeneralServerUnavailable,
			"Database schema error. Please contact support.")
	case "28000":
		return NewWithCustomMessage(pqErr, ErrCodeGeneralServerUnavailable,
			"Database authentication error. Please contact support.")
	case "28P01":
		return NewWithCustomMessage(pqErr, ErrCodeGeneralServerUnavailable,
			"Database authentication error. Please contact support.")
	case "53300":
		return NewWithCustomMessage(pqErr, ErrCodeGeneralServerUnavailable,
			"Database connection limit exceeded. Please try again later.")
	case "53400":
		return NewWithCustomMessage(pqErr, ErrCodeGeneralServerUnavailable,
			"Database memory limit exceeded. Please try again later.")
	case "54000":
		return NewWithCustomMessage(pqErr, ErrCodeGeneralServerUnavailable,
			"Database program limit exceeded. Please try again later.")
	case "55000":
		return NewWithCustomMessage(pqErr, ErrCodeGeneralServerUnavailable,
			"Database object not in prerequisite state. Please contact support.")
	case "55006":
		return NewWithCustomMessage(pqErr, ErrCodeGeneralServerUnavailable,
			"Database object not in prerequisite state. Please contact support.")
	case "57000":
		return NewWithCustomMessage(pqErr, ErrCodeGeneralServerUnavailable,
			"Database operator intervention. Please contact support.")
	case "58000":
		return NewWithCustomMessage(pqErr, ErrCodeGeneralServerUnavailable,
			"Database system error. Please contact support.")
	case "58030":
		return NewWithCustomMessage(pqErr, ErrCodeGeneralServerUnavailable,
			"Database io error. Please contact support.")
	case "59000":
		return NewWithCustomMessage(pqErr, ErrCodeGeneralServerUnavailable,
			"Database configuration file error. Please contact support.")
	case "5A000":
		return NewWithCustomMessage(pqErr, ErrCodeGeneralServerUnavailable,
			"Database feature not supported. Please contact support.")
	case "5B000":
		return NewWithCustomMessage(pqErr, ErrCodeGeneralServerUnavailable,
			"Database savepoint error. Please contact support.")
	case "5C000":
		return NewWithCustomMessage(pqErr, ErrCodeGeneralServerUnavailable,
			"Database invalid catalog name. Please contact support.")
	case "5D000":
		return NewWithCustomMessage(pqErr, ErrCodeGeneralServerUnavailable,
			"Database invalid schema name. Please contact support.")
	case "5E000":
		return NewWithCustomMessage(pqErr, ErrCodeGeneralServerUnavailable,
			"Database invalid schema name. Please contact support.")
	case "5F000":
		return NewWithCustomMessage(pqErr, ErrCodeGeneralServerUnavailable,
			"Database triggered action exception. Please contact support.")
	case "5L000":
		return NewWithCustomMessage(pqErr, ErrCodeGeneralServerUnavailable,
			"Database invalid grantor. Please contact support.")
	case "5M000":
		return NewWithCustomMessage(pqErr, ErrCodeGeneralServerUnavailable,
			"Database invalid sql state. Please contact support.")
	case "5N000":
		return NewWithCustomMessage(pqErr, ErrCodeGeneralServerUnavailable,
			"Database invalid sql state. Please contact support.")
	case "5P000":
		return NewWithCustomMessage(pqErr, ErrCodeGeneralServerUnavailable,
			"Database invalid role specification. Please contact support.")
	case "5R000":
		return NewWithCustomMessage(pqErr, ErrCodeGeneralServerUnavailable,
			"Database invalid role specification. Please contact support.")
	case "5S000":
		return NewWithCustomMessage(pqErr, ErrCodeGeneralServerUnavailable,
			"Database invalid role specification. Please contact support.")
	case "5T000":
		return NewWithCustomMessage(pqErr, ErrCodeGeneralServerUnavailable,
			"Database invalid role specification. Please contact support.")
	case "5U000":
		return NewWithCustomMessage(pqErr, ErrCodeGeneralServerUnavailable,
			"Database invalid role specification. Please contact support.")
	case "5V000":
		return NewWithCustomMessage(pqErr, ErrCodeGeneralServerUnavailable,
			"Database invalid role specification. Please contact support.")
	case "5W000":
		return NewWithCustomMessage(pqErr, ErrCodeGeneralServerUnavailable,
			"Database invalid role specification. Please contact support.")
	case "5X000":
		return NewWithCustomMessage(pqErr, ErrCodeGeneralServerUnavailable,
			"Database invalid role specification. Please contact support.")
	case "5Y000":
		return NewWithCustomMessage(pqErr, ErrCodeGeneralServerUnavailable,
			"Database invalid role specification. Please contact support.")
	case "5Z000":
		return NewWithCustomMessage(pqErr, ErrCodeGeneralServerUnavailable,
			"Database invalid role specification. Please contact support.")
	default:
		return New(pqErr, ErrCodeGeneralServerUnavailable)
	}
}

func IsConnectionError(err error) bool {
	if err == nil {
		return false
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}

	var syscallErr *net.OpError
	if errors.As(err, &syscallErr) {
		return true
	}

	if errors.Is(err, syscall.ECONNREFUSED) ||
		errors.Is(err, syscall.ECONNRESET) ||
		errors.Is(err, syscall.ETIMEDOUT) {
		return true
	}

	errStr := strings.ToLower(err.Error())
	connectionPatterns := []string{
		"connection refused",
		"connection reset",
		"connection timeout",
		"no such host",
		"network is unreachable",
		"broken pipe",
		"driver: bad connection",
		"invalid connection",
	}

	for _, pattern := range connectionPatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}

	return false
}

func getForeignKeyErrorMessage(constraintName string) string {
	switch constraintName {
	// User related foreign keys
	case "fk_users_organization":
		return "The specified organization does not exist."
	case "users_organization_id_fkey":
		return "The specified organization does not exist."
	case "users_created_by_fkey":
		return "The specified user (creator) does not exist."
	case "users_updated_by_fkey":
		return "The specified user (updater) does not exist."

	// Session related foreign keys
	case "fk_sessions_organization":
		return "Cannot create session: the specified organization does not exist."
	case "sessions_organization_id_fkey":
		return "The specified organization does not exist."
	case "sessions_user_id_fkey":
		return "The specified user does not exist."

	// User role related foreign keys
	case "fk_user_roles_organization":
		return "Cannot assign role: the specified organization does not exist."
	case "user_roles_organization_id_fkey":
		return "The specified organization does not exist."
	case "user_roles_user_id_fkey":
		return "The specified user does not exist."

	// Reset token related foreign keys
	case "fk_reset_tokens_organization":
		return "Cannot create reset token: the specified organization does not exist."
	case "reset_tokens_organization_id_fkey":
		return "The specified organization does not exist."
	case "reset_tokens_user_id_fkey":
		return "The specified user does not exist."

	// Security alert related foreign keys
	case "fk_security_alerts_organization":
		return "Cannot create security alert: the specified organization does not exist."
	case "security_alerts_organization_id_fkey":
		return "The specified organization does not exist."
	case "security_alerts_user_id_fkey":
		return "The specified user does not exist."

	// Room participant related foreign keys
	case "room_participants_room_id_fkey":
		return "The specified room does not exist."
	case "room_participants_user_id_fkey":
		return "The specified user does not exist."
	case "room_participants_organization_id_fkey":
		return "The specified organization does not exist."

	default:
		return ""
	}
}

func parseConstraintFromErrorMessage(errorMsg string) string {
	// User related foreign keys
	if strings.Contains(errorMsg, "fk_users_organization") {
		return "The specified organization does not exist."
	}
	if strings.Contains(errorMsg, "users_organization_id_fkey") {
		return "The specified organization does not exist."
	}
	if strings.Contains(errorMsg, "users_created_by_fkey") {
		return "The specified user (creator) does not exist."
	}
	if strings.Contains(errorMsg, "users_updated_by_fkey") {
		return "The specified user (updater) does not exist."
	}

	// Session related foreign keys
	if strings.Contains(errorMsg, "fk_sessions_organization") {
		return "Cannot create session: the specified organization does not exist."
	}
	if strings.Contains(errorMsg, "sessions_organization_id_fkey") {
		return "The specified organization does not exist."
	}
	if strings.Contains(errorMsg, "sessions_user_id_fkey") {
		return "The specified user does not exist."
	}

	// User role related foreign keys
	if strings.Contains(errorMsg, "fk_user_roles_organization") {
		return "Cannot assign role: the specified organization does not exist."
	}
	if strings.Contains(errorMsg, "user_roles_organization_id_fkey") {
		return "The specified organization does not exist."
	}
	if strings.Contains(errorMsg, "user_roles_user_id_fkey") {
		return "The specified user does not exist."
	}

	// Reset token related foreign keys
	if strings.Contains(errorMsg, "fk_reset_tokens_organization") {
		return "Cannot create reset token: the specified organization does not exist."
	}
	if strings.Contains(errorMsg, "reset_tokens_organization_id_fkey") {
		return "The specified organization does not exist."
	}
	if strings.Contains(errorMsg, "reset_tokens_user_id_fkey") {
		return "The specified user does not exist."
	}

	// Security alert related foreign keys
	if strings.Contains(errorMsg, "fk_security_alerts_organization") {
		return "Cannot create security alert: the specified organization does not exist."
	}
	if strings.Contains(errorMsg, "security_alerts_organization_id_fkey") {
		return "The specified organization does not exist."
	}
	if strings.Contains(errorMsg, "security_alerts_user_id_fkey") {
		return "The specified user does not exist."
	}

	// Category related foreign keys
	if strings.Contains(errorMsg, "categories_created_by_fkey") {
		return "The specified user (creator) does not exist."
	}
	if strings.Contains(errorMsg, "categories_updated_by_fkey") {
		return "The specified user (updater) does not exist."
	}
	if strings.Contains(errorMsg, "categories_parent_id_fkey") {
		return "The specified parent category does not exist."
	}
	if strings.Contains(errorMsg, "categories_organization_id_fkey") {
		return "The specified organization does not exist."
	}

	// Organization course related foreign keys
	if strings.Contains(errorMsg, "organization_courses_organization_id_fkey") {
		return "The specified organization does not exist."
	}
	if strings.Contains(errorMsg, "organization_courses_course_id_fkey") {
		return "The specified course does not exist."
	}
	if strings.Contains(errorMsg, "organization_courses_give_permission_by_fkey") {
		return "The specified user (permission granter) does not exist."
	}

	// Lecture related foreign keys
	if strings.Contains(errorMsg, "lectures_course_id_fkey") {
		return "The specified course does not exist."
	}
	if strings.Contains(errorMsg, "lectures_created_by_fkey") {
		return "The specified user (creator) does not exist."
	}
	if strings.Contains(errorMsg, "lectures_updated_by_fkey") {
		return "The specified user (updater) does not exist."
	}
	if strings.Contains(errorMsg, "lectures_organization_id_fkey") {
		return "The specified organization does not exist."
	}

	// Lecture section related foreign keys
	if strings.Contains(errorMsg, "lecture_sections_lecture_id_fkey") {
		return "The specified lecture does not exist."
	}
	if strings.Contains(errorMsg, "lecture_sections_created_by_fkey") {
		return "The specified user (creator) does not exist."
	}
	if strings.Contains(errorMsg, "lecture_sections_updated_by_fkey") {
		return "The specified user (updater) does not exist."
	}
	if strings.Contains(errorMsg, "lecture_sections_organization_id_fkey") {
		return "The specified organization does not exist."
	}

	// Section content related foreign keys
	if strings.Contains(errorMsg, "section_contents_section_id_fkey") {
		return "The specified lecture section does not exist."
	}
	if strings.Contains(errorMsg, "section_contents_created_by_fkey") {
		return "The specified user (creator) does not exist."
	}
	if strings.Contains(errorMsg, "section_contents_updated_by_fkey") {
		return "The specified user (updater) does not exist."
	}
	if strings.Contains(errorMsg, "section_contents_organization_id_fkey") {
		return "The specified organization does not exist."
	}

	// Review related foreign keys
	if strings.Contains(errorMsg, "reviews_user_id_fkey") {
		return "The specified user does not exist."
	}
	if strings.Contains(errorMsg, "reviews_organization_id_fkey") {
		return "The specified organization does not exist."
	}
	if strings.Contains(errorMsg, "reviews_target_id_fkey") {
		return "The specified target (course or system) does not exist."
	}

	// Room related foreign keys
	if strings.Contains(errorMsg, "rooms_created_by_fkey") {
		return "The specified user (creator) does not exist."
	}
	if strings.Contains(errorMsg, "rooms_updated_by_fkey") {
		return "The specified user (updater) does not exist."
	}
	if strings.Contains(errorMsg, "rooms_organization_id_fkey") {
		return "The specified organization does not exist."
	}

	// Room participant related foreign keys
	if strings.Contains(errorMsg, "room_participants_room_id_fkey") {
		return "The specified room does not exist."
	}
	if strings.Contains(errorMsg, "room_participants_user_id_fkey") {
		return "The specified user does not exist."
	}
	if strings.Contains(errorMsg, "room_participants_organization_id_fkey") {
		return "The specified organization does not exist."
	}

	// Video record related foreign keys
	if strings.Contains(errorMsg, "video_records_room_id_fkey") {
		return "The specified room does not exist."
	}
	if strings.Contains(errorMsg, "video_records_created_by_fkey") {
		return "The specified user (creator) does not exist."
	}
	if strings.Contains(errorMsg, "video_records_organization_id_fkey") {
		return "The specified organization does not exist."
	}

	// OAuth user related foreign keys
	if strings.Contains(errorMsg, "oauth_users_user_id_fkey") {
		return "The specified user does not exist."
	}
	if strings.Contains(errorMsg, "oauth_users_organization_id_fkey") {
		return "The specified organization does not exist."
	}

	// Unique constraint violations
	if strings.Contains(errorMsg, "courses_sku_key") {
		return "The specified course already exists."
	}
	if strings.Contains(errorMsg, "lectures_sku_key") {
		return "The specified lecture already exists."
	}
	if strings.Contains(errorMsg, "lecture_sections_sku_key") {
		return "The specified lecture section already exists."
	}
	if strings.Contains(errorMsg, "section_contents_sku_key") {
		return "The specified section content already exists."
	}
	if strings.Contains(errorMsg, "reset_tokens_token_hash_key") {
		return "The specified reset token has already been used."
	}
	if strings.Contains(errorMsg, "oauth_users_oauth_provider_oauth_provider_id_key") {
		return "The specified OAuth user already exists."
	}
	return ""
}

func handleDuplicateKeyViolation(err error) *AppError {
	errorStr := err.Error()

	// User related unique constraints
	if strings.Contains(errorStr, "users_email_key") {
		return New(err, ErrCodeAuthUserAlreadyExists)
	}
	if strings.Contains(errorStr, "users_username_key") {
		return New(err, ErrCodeAuthUserAlreadyExists)
	}

	// Reset token related unique constraints
	if strings.Contains(errorStr, "reset_tokens_token_hash_key") {
		return New(err, ErrCodeAuthResetTokenUsed)
	}

	// OAuth user related unique constraints
	if strings.Contains(errorStr, "oauth_users_oauth_provider_oauth_provider_id_key") {
		return New(err, ErrCodeAuthUserAlreadyExists)
	}

	// Session related unique constraints
	if strings.Contains(errorStr, "sessions_refresh_token_hash_key") {
		return New(err, ErrCodeAuthSessionNotFound)
	}

	if strings.Contains(errorStr, "resumes_user_id_key") {
		return New(err, ErrCodeAuthUserAlreadyExists)
	}

	// Default fallback
	return New(err, ErrCodeAuthUserAlreadyExists)
}

func handleCheckConstraintViolation(pqErr *pq.Error) *AppError {
	constraintName := pqErr.Constraint
	detail := pqErr.Detail

	// Handle specific check constraints
	switch constraintName {
	case "users_status_check":
		return NewWithCustomMessage(pqErr, ErrCodeGeneralConstraintViolation, "User status must be either 'active' or 'inactive'.")
	case "sessions_status_check":
		return NewWithCustomMessage(pqErr, ErrCodeGeneralConstraintViolation, "Session status must be either 'active' or 'expired'.")
	case "reset_tokens_status_check":
		return NewWithCustomMessage(pqErr, ErrCodeGeneralConstraintViolation, "Reset token status must be either 'active', 'used', or 'expired'.")
	default:
		// Generic check constraint violation
		if detail != "" {
			return NewWithCustomMessage(pqErr, ErrCodeGeneralConstraintViolation,
				fmt.Sprintf("Data validation failed: %s", detail))
		}
		return NewWithCustomMessage(pqErr, ErrCodeGeneralConstraintViolation,
			"Data validation failed due to check constraint violation.")
	}
}
