package apperrors

import (
	"errors"
	"os"
	"strings"
)

// UserMessage returns a user-facing message for any error. Never expose raw driver text in the UI.
func UserMessage(err error) string {
	if err == nil {
		return ""
	}
	if appErr, ok := As(err); ok {
		return appErr.Message
	}
	if errors.Is(err, os.ErrNotExist) {
		return "The database file could not be found."
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "syntax error") || strings.Contains(msg, "incomplete input") {
		return "The SQL could not be parsed. Check your syntax."
	}
	if strings.Contains(msg, "permission denied") {
		return "Permission denied opening the database file."
	}
	return "An unexpected error occurred."
}
