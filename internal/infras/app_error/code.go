package app_error

import "fmt"

type ErrorCode string

const (
	// General
	ErrCodeGeneralServerUnavailable   ErrorCode = "ONX0101"
	ErrCodeGeneralRequestTimeout      ErrorCode = "ONX0102"
	ErrCodeGeneralResourceNotFound    ErrorCode = "ONX0104"
	ErrCodeGeneralDatabaseConnection  ErrorCode = "ONX0105"
	ErrCodeGeneralConstraintViolation ErrorCode = "ONX0106"

	// Auth
	ErrCodeAuthInvalidToken            ErrorCode = "ONX0200"
	ErrCodeAuthExpiredToken            ErrorCode = "ONX0201"
	ErrCodeAuthInvalidAccessToken      ErrorCode = "ONX0202"
	ErrCodeAuthExpiredAccessToken      ErrorCode = "ONX0203"
	ErrCodeAuthInvalidRefreshToken     ErrorCode = "ONX0204"
	ErrCodeAuthExpiredRefreshToken     ErrorCode = "ONX0205"
	ErrCodeAuthUserNotFound            ErrorCode = "ONX0206"
	ErrCodeAuthUserAlreadyExists       ErrorCode = "ONX0207"
	ErrCodeAuthResetTokenNotFound      ErrorCode = "ONX0208"
	ErrCodeAuthResetTokenExpired       ErrorCode = "ONX0209"
	ErrCodeAuthResetTokenUsed          ErrorCode = "ONX0210"
	ErrCodeAuthPasswordSameAsOld       ErrorCode = "ONX0211"
	ErrCodeAuthInvalidHeader           ErrorCode = "ONX0212"
	ErrCodeAuthInvalidRequest          ErrorCode = "ONX0213"
	ErrCodeAuthInvalidVerifyEmailCode  ErrorCode = "ONX0214"
	ErrCodeAuthInvalidVerifyEmailToken ErrorCode = "ONX0215"
	ErrCodeAuthMaxAttemptVerifyEmail   ErrorCode = "ONX0216"
	ErrCodeAuthInvalidPassword         ErrorCode = "ONX0217"
	ErrCodeAuthSessionNotFound         ErrorCode = "ONX0218"
	ErrCodeAuthEmailNotVerified        ErrorCode = "ONX0219"

	// Resume
	ErrCodeResumeInvalidFileSize        ErrorCode = "ONX0300"
	ErrCodeResumeInvalidFileContentType ErrorCode = "ONX0301"
	ErrCodeResumeUploadFailed           ErrorCode = "ONX0302"
)

func (c ErrorCode) Message() string {
	msg, ok := ErrorCodeToMessage[c]
	if !ok {
		return fmt.Sprintf("no english error message for error code %q", c)
	}

	return string(msg)
}
