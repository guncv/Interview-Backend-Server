package app_error

import "fmt"

type ErrorCode string

const (
	// General
	ErrCodeGeneralServerUnavailable   ErrorCode = "INS0101"
	ErrCodeGeneralRequestTimeout      ErrorCode = "INS0102"
	ErrCodeGeneralResourceNotFound    ErrorCode = "INS0104"
	ErrCodeGeneralDatabaseConnection  ErrorCode = "INS0105"
	ErrCodeGeneralConstraintViolation ErrorCode = "INS0106"
	ErrCodeGeneralInvalidUUID         ErrorCode = "INS0107"
	ErrCodeGeneralInvalidTime         ErrorCode = "INS0108"
	ErrCodeGeneralRedisSetFailed      ErrorCode = "INS0109"
	ErrCodeGeneralRedisGetFailed      ErrorCode = "INS0110"
	ErrCodeGeneralUnmarshalFailed     ErrorCode = "INS0111"
	ErrCodeGeneralPermissionDenied    ErrorCode = "INS0112"
	ErrCodeGeneralInvalidLimit        ErrorCode = "INS0113"

	// Auth
	ErrCodeAuthInvalidToken            ErrorCode = "INS0200"
	ErrCodeAuthExpiredToken            ErrorCode = "INS0201"
	ErrCodeAuthInvalidRefreshToken     ErrorCode = "INS0202"
	ErrCodeAuthExpiredRefreshToken     ErrorCode = "INS0203"
	ErrCodeAuthUserNotFound            ErrorCode = "INS0206"
	ErrCodeAuthUserAlreadyExists       ErrorCode = "INS0207"
	ErrCodeAuthResetTokenNotFound      ErrorCode = "INS0208"
	ErrCodeAuthResetTokenExpired       ErrorCode = "INS0209"
	ErrCodeAuthResetTokenUsed          ErrorCode = "INS0210"
	ErrCodeAuthPasswordSameAsOld       ErrorCode = "INS0211"
	ErrCodeAuthInvalidHeader           ErrorCode = "INS0212"
	ErrCodeAuthInvalidRequest          ErrorCode = "INS0213"
	ErrCodeAuthInvalidVerifyEmailCode  ErrorCode = "INS0214"
	ErrCodeAuthInvalidVerifyEmailToken ErrorCode = "INS0215"
	ErrCodeAuthMaxAttemptVerifyEmail   ErrorCode = "INS0216"
	ErrCodeAuthInvalidPassword         ErrorCode = "INS0217"
	ErrCodeAuthSessionNotFound         ErrorCode = "INS0218"
	ErrCodeAuthEmailNotVerified        ErrorCode = "INS0219"

	// Resume
	ErrCodeResumeInvalidFileSize        ErrorCode = "INS0300"
	ErrCodeResumeInvalidFileContentType ErrorCode = "INS0301"
	ErrCodeResumeUploadFailed           ErrorCode = "INS0302"
	ErrCodeResumeAlreadyExists          ErrorCode = "INS0303"
	ErrCodeResumeNotFound               ErrorCode = "INS0304"
	ErrCodeResumeInvalidID              ErrorCode = "INS0305"
	ErrCodeResumeInvalidRequest         ErrorCode = "INS0306"
	ErrCodeResumeInvalidFileName        ErrorCode = "INS0307"

	// Session
	ErrCodeSessionInvalidToken                   ErrorCode = "INS0400"
	ErrCodeSessionNotFound                       ErrorCode = "INS0401"
	ErrCodeWebSocketInvalidHello                 ErrorCode = "INS0402"
	ErrCodeWebSocketInvalidSegmentStart          ErrorCode = "INS0403"
	ErrCodeWebSocketInvalidSegmentEnd            ErrorCode = "INS0404"
	ErrCodeWebSocketInvalidStopTTS               ErrorCode = "INS0405"
	ErrCodeWebSocketInvalidDBAck                 ErrorCode = "INS0406"
	ErrCodeWebSocketInvalidError                 ErrorCode = "INS0407"
	ErrCodeWebSocketInvalidPing                  ErrorCode = "INS0408"
	ErrCodeWebSocketInvalidMessage               ErrorCode = "INS0409"
	ErrCodeWebSocketInvalidUserPartialTranscript ErrorCode = "INS0410"
	ErrCodeWebSocketInvalidUserFullTranscript    ErrorCode = "INS0411"
	ErrCodeInterviewSessionStartEndTimeNotFound  ErrorCode = "INS0412"
	ErrCodeInterviewSessionStartTimeNotFound     ErrorCode = "INS0413"
	ErrCodeInterviewSessionEndTimeNotFound       ErrorCode = "INS0414"
	ErrCodeWebSocketInvalidSegmentID             ErrorCode = "INS0415"
	ErrCodeWebSocketInvalidSessionID             ErrorCode = "INS0416"

	// Evaluation
	ErrCodeEvaluationRubricNotFound         ErrorCode = "INS0500"
	ErrCodeEvaluationRubricCriteriaNotFound ErrorCode = "INS0501"

	// Interview Turns
	ErrCodeInterviewTurnLastMessageNotFound ErrorCode = "INS0600"
	ErrCodeInterviewTurnsMaxTurnNoNotFound  ErrorCode = "INS0601"

	// Issue Reports
	ErrCodeIssueReportNotFound     ErrorCode = "INS0700"
	ErrCodeIssueReportNotOpen      ErrorCode = "INS0701"
	ErrCodeIssueReportUnauthorized ErrorCode = "INS0702"
	ErrCodeIssueReportIDRequired   ErrorCode = "INS0703"

	// Issue Categories
	ErrCodeIssueCategoryNotFound ErrorCode = "INS0800"
)

func (c ErrorCode) Message() string {
	msg, ok := ErrorCodeToMessage[c]
	if !ok {
		return fmt.Sprintf("no english error message for error code %q", c)
	}

	return string(msg)
}
