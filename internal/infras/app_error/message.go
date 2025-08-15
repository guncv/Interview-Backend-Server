package app_error

type ErrorMessage string

const (
	// General
	ErrMessageGeneralServerUnavailable   ErrorMessage = "We're having trouble connecting to the server. Please try again shortly."
	ErrMessageGeneralRequestTimeout      ErrorMessage = "The request took too long. Please check your connection and try again."
	ErrMessageGeneralResourceNotFound    ErrorMessage = "The requested resource was not found."
	ErrMessageGeneralDatabaseConnection  ErrorMessage = "Database connection error. Please try again."
	ErrMessageGeneralConstraintViolation ErrorMessage = "The data violates database constraints."

	// Auth
	ErrMessageAuthInvalidToken              ErrorMessage = "Your session is invalid. Please log in again."
	ErrMessageAuthExpiredToken              ErrorMessage = "Your session has expired. Please log in again."
	ErrMessageAuthInvalidPassword           ErrorMessage = "Incorrect email or password. Please try again."
	ErrMessageAuthUserAlreadyExists         ErrorMessage = "This email is already registered. Try logging in instead."
	ErrMessageAuthUserNotFound              ErrorMessage = "We couldn't find your account. Please sign up to continue."
	ErrMessageAuthResetTokenNotFound        ErrorMessage = "That reset link is invalid. Please request a new one."
	ErrMessageAuthResetTokenExpired         ErrorMessage = "That reset link has expired. Please request a new one."
	ErrMessageAuthResetTokenUsed            ErrorMessage = "That reset link has already been used. Please request a new one."
	ErrMessageAuthPasswordSameAsOld         ErrorMessage = "Your new password can't be the same as the old one. Try a different password."
	ErrMessageAuthInvalidHeader             ErrorMessage = "Please log in to continue."
	ErrMessageAuthInvalidRequest            ErrorMessage = "Something went wrong with the request. Please try again."
	ErrMessageAuthOrganizationAlreadyExists ErrorMessage = "This organization already exists. Please try a different name."
	ErrMessageAuthInvalidRole               ErrorMessage = "You are not authorized to perform this action."
	ErrMessageAuthUserRoleAlreadyExists     ErrorMessage = "This user already has this role in the organization."
	ErrMessageAuthSessionNotFound           ErrorMessage = "Your session has expired or is invalid. Please log in again."
	ErrMessageAuthUserRoleNotFound          ErrorMessage = "The user role was not found."
	ErrMessageAuthOrganizationNotFound      ErrorMessage = "The organization was not found."
	ErrMessageAuthMissingRole               ErrorMessage = "Please provide a role to continue."

	// Category
	ErrMessageCategoriesNotFound            ErrorMessage = "This category was not found."
	ErrMessageCategoriesAlreadyExists       ErrorMessage = "This category already exists."
	ErrMessageCategoriesInvalidRequest      ErrorMessage = "This category request is invalid."
	ErrMessageCategoriesForeignKeyViolation ErrorMessage = "The specified category does not exist. Please select a valid category."

	// Course
	ErrMessageCoursesNotFound       ErrorMessage = "This course was not found."
	ErrMessageCoursesAlreadyExists  ErrorMessage = "This course already exists."
	ErrMessageCoursesInvalidRequest ErrorMessage = "This course request is invalid."

	// Organization
	ErrMessageOrganizationCourseNotFound       ErrorMessage = "This organization course was not found."
	ErrMessageOrganizationCourseAlreadyExists  ErrorMessage = "This organization already has permission to access this course."
	ErrMessageOrganizationCourseInvalidRequest ErrorMessage = "This organization course request is invalid."

	// Review
	ErrMessageReviewsNotFound       ErrorMessage = "This review was not found."
	ErrMessageReviewsAlreadyExists  ErrorMessage = "You have already reviewed this item. You can only submit one review per item."
	ErrMessageReviewsInvalidRequest ErrorMessage = "This review request is invalid."
	ErrMessageReviewsInvalidRating  ErrorMessage = "Rating must be between 1 and 5."
	ErrMessageReviewsInvalidTarget  ErrorMessage = "Invalid review target. Please select a valid course or system."

	// Course Section
	ErrMessageCourseSectionNotFound       ErrorMessage = "This course section was not found."
	ErrMessageCourseSectionAlreadyExists  ErrorMessage = "A section with this title already exists in this course."
	ErrMessageCourseSectionInvalidRequest ErrorMessage = "This course section request is invalid."

	// Section Content
	ErrMessageSectionContentsNotFound       ErrorMessage = "This section content was not found."
	ErrMessageSectionContentsAlreadyExists  ErrorMessage = "Content with this title already exists in this section."
	ErrMessageSectionContentsInvalidRequest ErrorMessage = "This section content request is invalid."
	ErrMessageSectionContentsInvalidType    ErrorMessage = "Content type must be either 'video', 'quiz', or 'pdf'."
)
