package app_error

type ErrorHttpCode int

const (
	ErrHttpCodeBadRequest          ErrorHttpCode = 400
	ErrHttpCodeUnauthorized        ErrorHttpCode = 401
	ErrHttpCodeForbidden           ErrorHttpCode = 403
	ErrHttpCodeNotFound            ErrorHttpCode = 404
	ErrHttpCodeInternalServerError ErrorHttpCode = 500
)
