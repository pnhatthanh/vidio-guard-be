package constants

type ErrorType string

const (
	ErrBadRequest   ErrorType = "bad_request"
	ErrInternal     ErrorType = "internal_error"
	ErrNotFound     ErrorType = "not_found"
	ErrUnauthorized ErrorType = "unauthorized"
	ErrForbidden    ErrorType = "forbidden"
	ErrConflict     ErrorType = "conflict"
)
