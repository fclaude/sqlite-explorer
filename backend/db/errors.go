package db

import (
	"context"
	"errors"
	"strings"

	"sqlite-explorer/backend/apperrors"
)

func mapQueryError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) {
		return apperrors.New(apperrors.CodeCancelled, "Query cancelled.", "")
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return apperrors.New(apperrors.CodeTimeout, "Query timed out.", "")
	}
	if appErr, ok := apperrors.As(err); ok {
		return appErr
	}
	msg := err.Error()
	lower := strings.ToLower(msg)
	if strings.Contains(lower, "syntax error") || strings.Contains(lower, "incomplete input") {
		return apperrors.New(apperrors.CodeMalformedSQL, "The SQL could not be parsed. Check your syntax.", msg)
	}
	return apperrors.New(apperrors.CodeMalformedSQL, "The query could not be executed.", msg)
}
