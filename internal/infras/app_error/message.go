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
	ErrMessageAuthInvalidToken            ErrorMessage = "Your session is invalid. Please log in again."
	ErrMessageAuthExpiredToken            ErrorMessage = "Your session has expired. Please log in again."
	ErrMessageAuthInvalidPassword         ErrorMessage = "Incorrect email or password. Please try again."
	ErrMessageAuthUserAlreadyExists       ErrorMessage = "This email is already registered. Try logging in instead."
	ErrMessageAuthUserNotFound            ErrorMessage = "We couldn't find your account. Please sign up to continue."
	ErrMessageAuthResetTokenNotFound      ErrorMessage = "That reset link is invalid. Please request a new one."
	ErrMessageAuthResetTokenExpired       ErrorMessage = "That reset link has expired. Please request a new one."
	ErrMessageAuthResetTokenUsed          ErrorMessage = "That reset link has already been used. Please request a new one."
	ErrMessageAuthPasswordSameAsOld       ErrorMessage = "Your new password can't be the same as the old one. Try a different password."
	ErrMessageAuthInvalidHeader           ErrorMessage = "Please log in to continue."
	ErrMessageAuthInvalidRequest          ErrorMessage = "Something went wrong with the request. Please try again."
	ErrMessageAuthSessionNotFound         ErrorMessage = "Your session has expired or is invalid. Please log in again."
	ErrMessageAuthInvalidVerifyEmailCode  ErrorMessage = "The verification code is invalid. Please try again."
	ErrMessageAuthInvalidVerifyEmailToken ErrorMessage = "The verification email link is invalid. Please request a new one."
	ErrMessageAuthMaxAttemptVerifyEmail   ErrorMessage = "You have reached the maximum number of attempts. Please resend the new code."
	ErrMessageAuthEmailNotVerified        ErrorMessage = "This email is not verified. Please verify your email to continue."
)
