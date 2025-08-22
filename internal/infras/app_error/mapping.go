package app_error

var ErrorCodeToHttpCode = map[ErrorCode]ErrorHttpCode{
	// General
	ErrCodeGeneralServerUnavailable:   ErrHttpCodeInternalServerError,
	ErrCodeGeneralRequestTimeout:      ErrHttpCodeInternalServerError,
	ErrCodeGeneralResourceNotFound:    ErrHttpCodeNotFound,
	ErrCodeGeneralDatabaseConnection:  ErrHttpCodeInternalServerError,
	ErrCodeGeneralConstraintViolation: ErrHttpCodeBadRequest,

	// Auth
	ErrCodeAuthInvalidToken:            ErrHttpCodeUnauthorized,
	ErrCodeAuthExpiredToken:            ErrHttpCodeUnauthorized,
	ErrCodeAuthInvalidRefreshToken:     ErrHttpCodeUnauthorized,
	ErrCodeAuthExpiredRefreshToken:     ErrHttpCodeUnauthorized,
	ErrCodeAuthInvalidPassword:         ErrHttpCodeBadRequest,
	ErrCodeAuthUserNotFound:            ErrHttpCodeNotFound,
	ErrCodeAuthUserAlreadyExists:       ErrHttpCodeBadRequest,
	ErrCodeAuthResetTokenNotFound:      ErrHttpCodeNotFound,
	ErrCodeAuthResetTokenExpired:       ErrHttpCodeNotFound,
	ErrCodeAuthResetTokenUsed:          ErrHttpCodeNotFound,
	ErrCodeAuthPasswordSameAsOld:       ErrHttpCodeBadRequest,
	ErrCodeAuthInvalidHeader:           ErrHttpCodeUnauthorized,
	ErrCodeAuthInvalidRequest:          ErrHttpCodeBadRequest,
	ErrCodeAuthInvalidVerifyEmailCode:  ErrHttpCodeBadRequest,
	ErrCodeAuthInvalidVerifyEmailToken: ErrHttpCodeBadRequest,
	ErrCodeAuthMaxAttemptVerifyEmail:   ErrHttpCodeBadRequest,
	ErrCodeAuthSessionNotFound:         ErrHttpCodeNotFound,
	ErrCodeAuthEmailNotVerified:        ErrHttpCodeBadRequest,

	// Resume
	ErrCodeResumeInvalidFileSize:        ErrHttpCodeBadRequest,
	ErrCodeResumeInvalidFileContentType: ErrHttpCodeBadRequest,
	ErrCodeResumeUploadFailed:           ErrHttpCodeInternalServerError,
	ErrCodeResumeAlreadyExists:          ErrHttpCodeBadRequest,
	ErrCodeResumeNotFound:               ErrHttpCodeNotFound,
	ErrCodeResumeInvalidID:              ErrHttpCodeBadRequest,
	ErrCodeResumeInvalidRequest:         ErrHttpCodeBadRequest,

	// Session
	ErrCodeSessionInvalidToken: ErrHttpCodeBadRequest,
	ErrCodeSessionNotFound:     ErrHttpCodeNotFound,
}

var ErrorCodeToMessage = map[ErrorCode]ErrorMessage{
	// General
	ErrCodeGeneralServerUnavailable:   ErrMessageGeneralServerUnavailable,
	ErrCodeGeneralRequestTimeout:      ErrMessageGeneralRequestTimeout,
	ErrCodeGeneralResourceNotFound:    ErrMessageGeneralResourceNotFound,
	ErrCodeGeneralDatabaseConnection:  ErrMessageGeneralDatabaseConnection,
	ErrCodeGeneralConstraintViolation: ErrMessageGeneralConstraintViolation,

	// Auth
	ErrCodeAuthInvalidToken:            ErrMessageAuthInvalidToken,
	ErrCodeAuthExpiredToken:            ErrMessageAuthExpiredToken,
	ErrCodeAuthInvalidRefreshToken:     ErrMessageAuthInvalidRefreshToken,
	ErrCodeAuthExpiredRefreshToken:     ErrMessageAuthExpiredRefreshToken,
	ErrCodeAuthInvalidPassword:         ErrMessageAuthInvalidPassword,
	ErrCodeAuthUserNotFound:            ErrMessageAuthUserNotFound,
	ErrCodeAuthUserAlreadyExists:       ErrMessageAuthUserAlreadyExists,
	ErrCodeAuthResetTokenNotFound:      ErrMessageAuthResetTokenNotFound,
	ErrCodeAuthResetTokenExpired:       ErrMessageAuthResetTokenExpired,
	ErrCodeAuthResetTokenUsed:          ErrMessageAuthResetTokenUsed,
	ErrCodeAuthPasswordSameAsOld:       ErrMessageAuthPasswordSameAsOld,
	ErrCodeAuthInvalidHeader:           ErrMessageAuthInvalidHeader,
	ErrCodeAuthInvalidRequest:          ErrMessageAuthInvalidRequest,
	ErrCodeAuthInvalidVerifyEmailCode:  ErrMessageAuthInvalidVerifyEmailCode,
	ErrCodeAuthInvalidVerifyEmailToken: ErrMessageAuthInvalidVerifyEmailToken,
	ErrCodeAuthMaxAttemptVerifyEmail:   ErrMessageAuthMaxAttemptVerifyEmail,
	ErrCodeAuthSessionNotFound:         ErrMessageAuthSessionNotFound,
	ErrCodeAuthEmailNotVerified:        ErrMessageAuthEmailNotVerified,

	// Resume
	ErrCodeResumeInvalidFileSize:        ErrMessageResumeInvalidFileSize,
	ErrCodeResumeInvalidFileContentType: ErrMessageResumeInvalidFileContentType,
	ErrCodeResumeUploadFailed:           ErrMessageResumeUploadFailed,
	ErrCodeResumeAlreadyExists:          ErrMessageResumeAlreadyExists,
	ErrCodeResumeNotFound:               ErrMessageResumeNotFound,
	ErrCodeResumeInvalidID:              ErrMessageResumeInvalidID,
	ErrCodeResumeInvalidRequest:         ErrMessageResumeInvalidRequest,

	// Session
	ErrCodeSessionInvalidToken: ErrMessageSessionInvalidToken,
	ErrCodeSessionNotFound:     ErrMessageSessionNotFound,
}
