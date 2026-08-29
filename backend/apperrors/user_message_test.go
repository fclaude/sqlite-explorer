package apperrors

import (
	"errors"
	"os"
	"testing"
)

func TestErrors_UserMessage(t *testing.T) {
	t.Parallel()
	cases := []struct {
		err  error
		want string
	}{
		{New(CodeNoDBOpen, "No database is open.", ""), "No database is open."},
		{New(CodeNotSQLite, "The selected file is not a valid SQLite database.", ""), "The selected file is not a valid SQLite database."},
		{New(CodePermission, "Permission denied opening the database file.", ""), "Permission denied opening the database file."},
		{New(CodeCancelled, "Query cancelled.", ""), "Query cancelled."},
		{New(CodeUnknownTable, "Table or view not found.", ""), "Table or view not found."},
		{New(CodeInvalidColumn, "Sort column does not exist on this table.", ""), "Sort column does not exist on this table."},
		{New(CodeReadOnlyViolation, "Read-only mode: INSERT statements are not allowed.", ""), "Read-only mode: INSERT statements are not allowed."},
		{New(CodeTimeout, "Query timed out.", ""), "Query timed out."},
		{New(CodeMalformedSQL, "Could not parse SQL statement.", ""), "Could not parse SQL statement."},
		{errors.Join(os.ErrNotExist, errors.New("missing")), "The database file could not be found."},
		{errors.New("near \"FROM\": syntax error"), "The SQL could not be parsed. Check your syntax."},
		{errors.New("something else"), "An unexpected error occurred."},
	}
	for _, tc := range cases {
		got := UserMessage(tc.err)
		if got != tc.want {
			t.Errorf("UserMessage(%v) = %q, want %q", tc.err, got, tc.want)
		}
	}
}
