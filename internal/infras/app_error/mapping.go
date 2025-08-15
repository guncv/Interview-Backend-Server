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

	// Category
	ErrCodeCategoriesNotFound:            ErrHttpCodeNotFound,
	ErrCodeCategoriesAlreadyExists:       ErrHttpCodeBadRequest,
	ErrCodeCategoriesInvalidRequest:      ErrHttpCodeBadRequest,
	ErrCodeCategoriesForeignKeyViolation: ErrHttpCodeBadRequest,

	// Course
	ErrCodeCoursesNotFound:       ErrHttpCodeNotFound,
	ErrCodeCoursesAlreadyExists:  ErrHttpCodeBadRequest,
	ErrCodeCoursesInvalidRequest: ErrHttpCodeBadRequest,

	// Organization
	ErrCodeOrganizationCourseNotFound:       ErrHttpCodeNotFound,
	ErrCodeOrganizationCourseAlreadyExists:  ErrHttpCodeBadRequest,
	ErrCodeOrganizationCourseInvalidRequest: ErrHttpCodeBadRequest,

	// Review
	ErrCodeReviewsNotFound:       ErrHttpCodeNotFound,
	ErrCodeReviewsAlreadyExists:  ErrHttpCodeBadRequest,
	ErrCodeReviewsInvalidRequest: ErrHttpCodeBadRequest,
	ErrCodeReviewsInvalidRating:  ErrHttpCodeBadRequest,
	ErrCodeReviewsInvalidTarget:  ErrHttpCodeBadRequest,

	// Course Section
	ErrCodeCourseSectionNotFound:       ErrHttpCodeNotFound,
	ErrCodeCourseSectionAlreadyExists:  ErrHttpCodeBadRequest,
	ErrCodeCourseSectionInvalidRequest: ErrHttpCodeBadRequest,

	// Section Content
	ErrCodeSectionContentsNotFound:       ErrHttpCodeNotFound,
	ErrCodeSectionContentsAlreadyExists:  ErrHttpCodeBadRequest,
	ErrCodeSectionContentsInvalidRequest: ErrHttpCodeBadRequest,
	ErrCodeSectionContentsInvalidType:    ErrHttpCodeBadRequest,
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

	// Category
	ErrCodeCategoriesNotFound:            ErrMessageCategoriesNotFound,
	ErrCodeCategoriesAlreadyExists:       ErrMessageCategoriesAlreadyExists,
	ErrCodeCategoriesInvalidRequest:      ErrMessageCategoriesInvalidRequest,
	ErrCodeCategoriesForeignKeyViolation: ErrMessageCategoriesForeignKeyViolation,

	// Course
	ErrCodeCoursesNotFound:       ErrMessageCoursesNotFound,
	ErrCodeCoursesAlreadyExists:  ErrMessageCoursesAlreadyExists,
	ErrCodeCoursesInvalidRequest: ErrMessageCoursesInvalidRequest,

	// Organization
	ErrCodeOrganizationCourseNotFound:       ErrMessageOrganizationCourseNotFound,
	ErrCodeOrganizationCourseAlreadyExists:  ErrMessageOrganizationCourseAlreadyExists,
	ErrCodeOrganizationCourseInvalidRequest: ErrMessageOrganizationCourseInvalidRequest,

	// Review
	ErrCodeReviewsNotFound:       ErrMessageReviewsNotFound,
	ErrCodeReviewsAlreadyExists:  ErrMessageReviewsAlreadyExists,
	ErrCodeReviewsInvalidRequest: ErrMessageReviewsInvalidRequest,
	ErrCodeReviewsInvalidRating:  ErrMessageReviewsInvalidRating,
	ErrCodeReviewsInvalidTarget:  ErrMessageReviewsInvalidTarget,

	// Course Section
	ErrCodeCourseSectionNotFound:       ErrMessageCourseSectionNotFound,
	ErrCodeCourseSectionAlreadyExists:  ErrMessageCourseSectionAlreadyExists,
	ErrCodeCourseSectionInvalidRequest: ErrMessageCourseSectionInvalidRequest,

	// Section Content
	ErrCodeSectionContentsNotFound:       ErrMessageSectionContentsNotFound,
	ErrCodeSectionContentsAlreadyExists:  ErrMessageSectionContentsAlreadyExists,
	ErrCodeSectionContentsInvalidRequest: ErrMessageSectionContentsInvalidRequest,
	ErrCodeSectionContentsInvalidType:    ErrMessageSectionContentsInvalidType,
}
