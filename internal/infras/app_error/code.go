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
	ErrCodeAuthInvalidToken              ErrorCode = "ONX0201"
	ErrCodeAuthExpiredToken              ErrorCode = "ONX0202"
	ErrCodeAuthInvalidPassword           ErrorCode = "ONX0203"
	ErrCodeAuthUserNotFound              ErrorCode = "ONX0204"
	ErrCodeAuthUserAlreadyExists         ErrorCode = "ONX0205"
	ErrCodeAuthResetTokenNotFound        ErrorCode = "ONX0206"
	ErrCodeAuthResetTokenExpired         ErrorCode = "ONX0207"
	ErrCodeAuthResetTokenUsed            ErrorCode = "ONX0208"
	ErrCodeAuthPasswordSameAsOld         ErrorCode = "ONX0209"
	ErrCodeAuthInvalidHeader             ErrorCode = "ONX0210"
	ErrCodeAuthInvalidRequest            ErrorCode = "ONX0211"
	ErrCodeAuthOrganizationAlreadyExists ErrorCode = "ONX0212"
	ErrCodeAuthInvalidRole               ErrorCode = "ONX0213"
	ErrCodeAuthUserRoleAlreadyExists     ErrorCode = "ONX0214"
	ErrCodeAuthSessionNotFound           ErrorCode = "ONX0215"
	ErrCodeAuthUserRoleNotFound          ErrorCode = "ONX0216"
	ErrCodeAuthOrganizationNotFound      ErrorCode = "ONX0217"
	ErrCodeAuthMissingRole               ErrorCode = "ONX0218"

	// Category
	ErrCodeCategoriesNotFound            ErrorCode = "ONX0301"
	ErrCodeCategoriesAlreadyExists       ErrorCode = "ONX0302"
	ErrCodeCategoriesInvalidRequest      ErrorCode = "ONX0303"
	ErrCodeCategoriesForeignKeyViolation ErrorCode = "ONX0304"

	// Course
	ErrCodeCoursesNotFound       ErrorCode = "ONX0401"
	ErrCodeCoursesAlreadyExists  ErrorCode = "ONX0402"
	ErrCodeCoursesInvalidRequest ErrorCode = "ONX0403"

	// Organization
	ErrCodeOrganizationCourseNotFound       ErrorCode = "ONX0501"
	ErrCodeOrganizationCourseAlreadyExists  ErrorCode = "ONX0502"
	ErrCodeOrganizationCourseInvalidRequest ErrorCode = "ONX0503"

	// Review
	ErrCodeReviewsNotFound       ErrorCode = "ONX0601"
	ErrCodeReviewsAlreadyExists  ErrorCode = "ONX0602"
	ErrCodeReviewsInvalidRequest ErrorCode = "ONX0603"
	ErrCodeReviewsInvalidRating  ErrorCode = "ONX0604"
	ErrCodeReviewsInvalidTarget  ErrorCode = "ONX0605"

	// Course Section
	ErrCodeCourseSectionNotFound       ErrorCode = "ONX0701"
	ErrCodeCourseSectionAlreadyExists  ErrorCode = "ONX0702"
	ErrCodeCourseSectionInvalidRequest ErrorCode = "ONX0703"

	// Section Content
	ErrCodeSectionContentsNotFound       ErrorCode = "ONX0801"
	ErrCodeSectionContentsAlreadyExists  ErrorCode = "ONX0802"
	ErrCodeSectionContentsInvalidRequest ErrorCode = "ONX0803"
	ErrCodeSectionContentsInvalidType    ErrorCode = "ONX0804"
)

func (c ErrorCode) Message() string {
	msg, ok := ErrorCodeToMessage[c]
	if !ok {
		return fmt.Sprintf("no english error message for error code %q", c)
	}

	return string(msg)
}
