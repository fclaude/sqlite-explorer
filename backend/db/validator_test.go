package db

import (
	"testing"

	"sqlite-explorer/backend/apperrors"
)

func TestReadOnlyValidator(t *testing.T) {
	tests := []struct {
		name    string
		sql     string
		allowed bool
	}{
		{"select", "SELECT 1", true},
		{"comment", "  -- c\nSELECT 1", true},
		{"with", "WITH t AS (SELECT 1) SELECT * FROM t", true},
		{"pragma", "PRAGMA table_info(x)", true},
		{"writable_schema", "PRAGMA writable_schema = 1", false},
		{"insert", "INSERT INTO x VALUES (1)", false},
		{"chained", "SELECT 1; DROP TABLE x", false},
		{"attach", "ATTACH DATABASE 'x' AS y", false},
		{"vacuum", "VACUUM", false},
		{"update_case", "update x set y=1", false},
		{"injection", "'; DROP TABLE x; --", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateReadOnlySQL(tt.sql)
			if tt.allowed {
				if err != nil {
					t.Fatalf("expected allowed, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("expected error")
			}
			appErr, ok := apperrors.As(err)
			if !ok {
				t.Fatalf("expected app error, got %v", err)
			}
			if appErr.Code != apperrors.CodeReadOnlyViolation && appErr.Code != apperrors.CodeMalformedSQL {
				t.Fatalf("code: %s", appErr.Code)
			}
		})
	}
}
