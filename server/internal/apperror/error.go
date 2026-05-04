package apperror

type ErrorType string

const (
	ErrBadRequest   ErrorType = "bad_request"
	ErrInternal     ErrorType = "internal_error"
	ErrNotFound     ErrorType = "not_found"
	ErrUnauthorized ErrorType = "unauthorized"
	ErrForbidden    ErrorType = "forbidden"
)

type AppError struct {
	Code    int       `json:"code"`
	Type    ErrorType `json:"type"`
	Message string    `json:"message"`
}

func NewAppError(code int, errType ErrorType, message string) *AppError {
	return &AppError{
		Code:    code,
		Type:    errType,
		Message: message,
	}
}
func (e *AppError) Error() string {
	return e.Message
}
func NewBadRequestError(message string) *AppError {
	return NewAppError(400, ErrBadRequest, message)
}
func NewInternalServerError(message string) *AppError {
	return NewAppError(500, ErrInternal, message)
}
func NewNotFoundError(message string) *AppError {
	return NewAppError(404, ErrNotFound, message)
}
func NewUnauthorizedError(message string) *AppError {
	return NewAppError(401, ErrUnauthorized, message)
}
func NewForbiddenError(message string) *AppError {
	return NewAppError(403, ErrForbidden, message)
}
