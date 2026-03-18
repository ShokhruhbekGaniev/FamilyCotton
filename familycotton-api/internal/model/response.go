package model

import "fmt"

type Meta struct {
	Page  int `json:"page"`
	Limit int `json:"limit"`
	Total int `json:"total"`
}

type SuccessResponse struct {
	Data any   `json:"data"`
	Meta *Meta `json:"meta,omitempty"`
}

type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

var (
	ErrNotFound     = fmt.Errorf("not found")
	ErrValidation   = fmt.Errorf("validation error")
	ErrForbidden    = fmt.Errorf("forbidden")
	ErrUnauthorized = fmt.Errorf("unauthorized")
	ErrConflict     = fmt.Errorf("conflict")
)

type AppError struct {
	Err     error
	Message string
}

func (e *AppError) Error() string { return e.Message }
func (e *AppError) Unwrap() error { return e.Err }

func NewAppError(sentinel error, msg string) *AppError {
	return &AppError{Err: sentinel, Message: msg}
}
