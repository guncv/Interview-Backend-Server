package app_error

var ErrorCodeToHttpCode = map[ErrorCode]ErrorHttpCode{
	// General
	ErrCodeGeneralServerUnavailable:   ErrHttpCodeInternalServerError,
	ErrCodeGeneralRequestTimeout:      ErrHttpCodeInternalServerError,
	ErrCodeGeneralResourceNotFound:    ErrHttpCodeNotFound,
	ErrCodeGeneralDatabaseConnection:  ErrHttpCodeInternalServerError,
	ErrCodeGeneralConstraintViolation: ErrHttpCodeBadRequest,

	// Auth
	ErrCodeAuthInvalidToken:              ErrHttpCodeUnauthorized,
	ErrCodeAuthExpiredToken:              ErrHttpCodeUnauthorized,
	ErrCodeAuthInvalidPassword:           ErrHttpCodeBadRequest,
	ErrCodeAuthUserNotFound:              ErrHttpCodeNotFound,
	ErrCodeAuthUserAlreadyExists:         ErrHttpCodeBadRequest,
	ErrCodeAuthResetTokenNotFound:        ErrHttpCodeNotFound,
	ErrCodeAuthResetTokenExpired:         ErrHttpCodeNotFound,
	ErrCodeAuthResetTokenUsed:            ErrHttpCodeNotFound,
	ErrCodeAuthPasswordSameAsOld:         ErrHttpCodeBadRequest,
	ErrCodeAuthInvalidHeader:             ErrHttpCodeUnauthorized,
	ErrCodeAuthInvalidRequest:            ErrHttpCodeBadRequest,
	ErrCodeAuthOrganizationAlreadyExists: ErrHttpCodeBadRequest,
	ErrCodeAuthInvalidRole:               ErrHttpCodeBadRequest,
	ErrCodeAuthUserRoleAlreadyExists:     ErrHttpCodeBadRequest,
	ErrCodeAuthSessionNotFound:           ErrHttpCodeNotFound,
	ErrCodeAuthUserRoleNotFound:          ErrHttpCodeNotFound,
	ErrCodeAuthOrganizationNotFound:      ErrHttpCodeNotFound,
	ErrCodeAuthMissingRole:               ErrHttpCodeBadRequest,
}

var ErrorCodeToMessage = map[ErrorCode]ErrorMessage{
	// General
	ErrCodeGeneralServerUnavailable:   ErrMessageGeneralServerUnavailable,
	ErrCodeGeneralRequestTimeout:      ErrMessageGeneralRequestTimeout,
	ErrCodeGeneralResourceNotFound:    ErrMessageGeneralResourceNotFound,
	ErrCodeGeneralDatabaseConnection:  ErrMessageGeneralDatabaseConnection,
	ErrCodeGeneralConstraintViolation: ErrMessageGeneralConstraintViolation,

	// Auth
	ErrCodeAuthInvalidToken:              ErrMessageAuthInvalidToken,
	ErrCodeAuthExpiredToken:              ErrMessageAuthExpiredToken,
	ErrCodeAuthInvalidPassword:           ErrMessageAuthInvalidPassword,
	ErrCodeAuthUserNotFound:              ErrMessageAuthUserNotFound,
	ErrCodeAuthUserAlreadyExists:         ErrMessageAuthUserAlreadyExists,
	ErrCodeAuthResetTokenNotFound:        ErrMessageAuthResetTokenNotFound,
	ErrCodeAuthResetTokenExpired:         ErrMessageAuthResetTokenExpired,
	ErrCodeAuthResetTokenUsed:            ErrMessageAuthResetTokenUsed,
	ErrCodeAuthPasswordSameAsOld:         ErrMessageAuthPasswordSameAsOld,
	ErrCodeAuthInvalidHeader:             ErrMessageAuthInvalidHeader,
	ErrCodeAuthInvalidRequest:            ErrMessageAuthInvalidRequest,
	ErrCodeAuthOrganizationAlreadyExists: ErrMessageAuthOrganizationAlreadyExists,
	ErrCodeAuthInvalidRole:               ErrMessageAuthInvalidRole,
	ErrCodeAuthUserRoleAlreadyExists:     ErrMessageAuthUserRoleAlreadyExists,
	ErrCodeAuthSessionNotFound:           ErrMessageAuthSessionNotFound,
	ErrCodeAuthUserRoleNotFound:          ErrMessageAuthUserRoleNotFound,
	ErrCodeAuthOrganizationNotFound:      ErrMessageAuthOrganizationNotFound,
	ErrCodeAuthMissingRole:               ErrMessageAuthMissingRole,
}
