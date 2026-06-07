package apperror

import "github.com/pnhatthanh/vidio-guard-be/internal/constants"

type AppError struct {
	Code    int          `json:"code"`
	Type    constants.ErrorType `json:"type"`
	Message string       `json:"message"`
}

func NewAppError(code int, errType constants.ErrorType, message string) *AppError {
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
	return NewAppError(400, constants.ErrBadRequest, message)
}
func NewInternalServerError(message string) *AppError {
	return NewAppError(500, constants.ErrInternal, message)
}
func NewNotFoundError(message string) *AppError {
	return NewAppError(404, constants.ErrNotFound, message)
}
func NewUnauthorizedError(message string) *AppError {
	return NewAppError(401, constants.ErrUnauthorized, message)
}
func NewForbiddenError(message string) *AppError {
	return NewAppError(403, constants.ErrForbidden, message)
}
func NewDuplicateError(message string) *AppError {
	return NewAppError(409, constants.ErrConflict, message)
}
