package app_error

type ErrorMessage string

const (
	// General
	ErrMessageGeneralServerUnavailable     ErrorMessage = "We're having trouble connecting to the server. Please try again shortly."
	ErrMessageGeneralRequestTimeout        ErrorMessage = "The request took too long. Please check your connection and try again."
	ErrMessageGeneralResourceNotFound      ErrorMessage = "The requested resource was not found."
	ErrMessageGeneralDatabaseConnection    ErrorMessage = "Database connection error. Please try again."
	ErrMessageGeneralConstraintViolation   ErrorMessage = "The data violates database constraints."
	ErrMessageGeneralInvalidUUID           ErrorMessage = "The UUID is invalid. Please try again."
	ErrMessageGeneralInvalidTime           ErrorMessage = "The time format is invalid. Please try again."
	ErrMessageGeneralRedisSetFailed        ErrorMessage = "Failed to set the value in Redis. Please try again."
	ErrMessageGeneralRedisGetFailed        ErrorMessage = "Failed to get the value from Redis. Please try again."
	ErrMessageGeneralUnmarshalFailed       ErrorMessage = "Failed to unmarshal the value. Please try again."
	ErrMessageGeneralPermissionDenied      ErrorMessage = "You are not authorized to perform this action."
	ErrMessageGeneralInvalidLimit          ErrorMessage = "The limit is invalid. Please try again."
	ErrMessageGeneralInvalidNumber         ErrorMessage = "The number is invalid. Please try again."
	ErrMessageGeneralInvalidOffset         ErrorMessage = "The offset is invalid. Please try again."
	ErrMessageGeneralInvalidPaginationType ErrorMessage = "The pagination type is invalid. Please try again."

	// Auth
	ErrMessageAuthInvalidToken            ErrorMessage = "Your token is invalid. Please log in again."
	ErrMessageAuthExpiredToken            ErrorMessage = "Your token has expired. Please log in again."
	ErrMessageAuthInvalidRefreshToken     ErrorMessage = "Your refresh token is invalid. Please log in again."
	ErrMessageAuthExpiredRefreshToken     ErrorMessage = "Your refresh token has expired. Please log in again."
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

	// Resume
	ErrMessageResumeInvalidFileSize        ErrorMessage = "The file size is too large. Please try again with a smaller file."
	ErrMessageResumeInvalidFileContentType ErrorMessage = "The file type is not supported. Please try again with a supported file type."
	ErrMessageResumeUploadFailed           ErrorMessage = "Failed to upload the file. Please try again."
	ErrMessageResumeAlreadyExists          ErrorMessage = "You already have a resume. Please switch to the existing resume or create a new one."
	ErrMessageResumeNotFound               ErrorMessage = "The resume was not found. Please try again."
	ErrMessageResumeInvalidID              ErrorMessage = "The resume ID is invalid. Please try again."
	ErrMessageResumeInvalidRequest         ErrorMessage = "The request is invalid. Please try again."
	ErrMessageResumeInvalidFileName        ErrorMessage = "The file name is invalid. Please try again."

	// Session
	ErrMessageSessionInvalidToken ErrorMessage = "The session token is invalid. Please try again."
	ErrMessageSessionNotFound     ErrorMessage = "The session was not found. Please try again."

	// WebSocket
	ErrMessageWebSocketInvalidHello                 ErrorMessage = "The hello message is invalid. Please try again."
	ErrMessageWebSocketInvalidSegmentStart          ErrorMessage = "The segment start message is invalid. Please try again."
	ErrMessageWebSocketInvalidSegmentEnd            ErrorMessage = "The segment end message is invalid. Please try again."
	ErrMessageWebSocketInvalidStopTTS               ErrorMessage = "The stop TTS message is invalid. Please try again."
	ErrMessageWebSocketInvalidDBAck                 ErrorMessage = "The DB ack message is invalid. Please try again."
	ErrMessageWebSocketInvalidError                 ErrorMessage = "The error message is invalid. Please try again."
	ErrMessageWebSocketInvalidPing                  ErrorMessage = "The ping message is invalid. Please try again."
	ErrMessageWebSocketInvalidMessage               ErrorMessage = "The message is invalid. Please try again."
	ErrMessageWebSocketInvalidUserPartialTranscript ErrorMessage = "The user partial transcript message is invalid. Please try again."
	ErrMessageWebSocketInvalidUserFullTranscript    ErrorMessage = "The user full transcript message is invalid. Please try again."
	ErrMessageInterviewSessionStartEndTimeNotFound  ErrorMessage = "Interview session start and end time not found. Please try again."
	ErrMessageInterviewSessionStartTimeNotFound     ErrorMessage = "Interview session start time not found. Please try again."
	ErrMessageInterviewSessionEndTimeNotFound       ErrorMessage = "Interview session end time not found. Please try again."
	ErrMessageWebSocketInvalidSegmentID             ErrorMessage = "The segment ID is invalid or expired. Please try again."
	ErrMessageWebSocketInvalidSessionID             ErrorMessage = "The session ID is invalid. Please try again."
	ErrMessageSessionInvalidSessionID               ErrorMessage = "The session ID is invalid. Please try again."
	ErrMessageSessionNotFoundOrDeleted              ErrorMessage = "The session was not found or deleted. Please try again."
	ErrMessageSessionUserNotMatch                   ErrorMessage = "This user does not have access to this session. Please try again."
	// Evaluation
	ErrMessageEvaluationRubricNotFound         ErrorMessage = "The rubric was not found. Please try again."
	ErrMessageEvaluationRubricCriteriaNotFound ErrorMessage = "The rubric criteria was not found. Please try again."

	// Interview Turns
	ErrMessageInterviewTurnLastMessageNotFound ErrorMessage = "The interviewer last message was not found. Please try again."
	ErrMessageInterviewTurnsMaxTurnNoNotFound  ErrorMessage = "The max turn no by session ID was not found. Please try again."

	// Issue Reports
	ErrMessageIssueReportNotFound     ErrorMessage = "The issue report was not found. Please try again."
	ErrMessageIssueReportNotOpen      ErrorMessage = "The issue report is not open. Please try again."
	ErrMessageIssueReportUnauthorized ErrorMessage = "The issue report is unauthorized. Please try again."
	ErrMessageIssueReportIDRequired   ErrorMessage = "The issue report ID is required. Please try again."

	// Issue Categories
	ErrMessageIssueCategoryNotFound ErrorMessage = "The issue category was not found. Please try again."
)
