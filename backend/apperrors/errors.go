package apperrors

import (
	"errors"
	"fmt"
)

// Code identifies a class of application error for the frontend.
type Code string

const (
	CodeNoDBOpen      Code = "NO_DB_OPEN"
	CodeNotSQLite     Code = "NOT_SQLITE"
	CodePermission    Code = "PERMISSION_DENIED"
	CodeCancelled     Code = "CANCELLED"
)

// Error is a user-facing error with a stable code.
type Error struct {
	Code    Code   `json:"code"`
	Message string `json:"message"`
	Detail  string `json:"detail"`
}

func (e *Error) Error() string {
	if e.Detail != "" {
		return fmt.Sprintf("%s: %s (%s)", e.Code, e.Message, e.Detail)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// New builds an application error.
func New(code Code, message, detail string) *Error {
	return &Error{Code: code, Message: message, Detail: detail}
}

// As returns the *Error if err wraps one.
func As(err error) (*Error, bool) {
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr, true
	}
	return nil, false
}
