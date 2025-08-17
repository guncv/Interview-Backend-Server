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
	ErrCodeAuthInvalidAccessToken:      ErrHttpCodeUnauthorized,
	ErrCodeAuthExpiredAccessToken:      ErrHttpCodeUnauthorized,
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
	ErrCodeAuthInvalidAccessToken:      ErrMessageAuthInvalidAccessToken,
	ErrCodeAuthExpiredAccessToken:      ErrMessageAuthExpiredAccessToken,
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
}
