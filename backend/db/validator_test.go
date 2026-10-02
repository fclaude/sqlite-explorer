package db

import (
	"strings"
	"testing"

	"sqlite-explorer/backend/apperrors"
)

func TestPlanSQL_ReadOnlyPolicy(t *testing.T) {
	tests := []struct {
		name    string
		sql     string
		allowed bool
	}{
		{"select", "SELECT 1", true},
		{"comment", "  -- c\nSELECT 1", true},
		{"with", "WITH t AS (SELECT 1) SELECT * FROM t", true},
		{"with recursive", "WITH RECURSIVE c(x) AS (SELECT 1 UNION ALL SELECT x+1 FROM c LIMIT 3) SELECT x FROM c", true},
		{"with materialized", "WITH a(x) AS NOT MATERIALIZED (SELECT 1), b AS MATERIALIZED (SELECT 2) SELECT * FROM a, b", true},
		{"with values", "WITH t AS (SELECT 1) VALUES (1)", true},
		{"values", "VALUES (1), (2)", true},
		{"explain", "EXPLAIN QUERY PLAN SELECT 1", true},
		{"explain write", "EXPLAIN DELETE FROM t", true},
		{"multiple reads", "SELECT 1; SELECT 2;", true},
		{"semicolons in identifiers", "SELECT [a;b], `c;d`, \"e;f\", 'g;h' FROM t", true},
		{"pragma table_info", "PRAGMA table_info(x)", true},
		{"pragma schema prefix", "PRAGMA main.table_info(x)", true},
		{"pragma user_version read", "PRAGMA user_version", true},
		{"pragma journal_mode read", "PRAGMA journal_mode", true},
		{"pragma foreign_key_check", "PRAGMA foreign_key_check(t)", true},

		{"insert", "INSERT INTO x VALUES (1)", false},
		{"replace", "REPLACE INTO x VALUES (1)", false},
		{"update lowercase", "update x set y=1", false},
		{"chained", "SELECT 1; DROP TABLE x", false},
		{"quote in line comment", "SELECT 1 WHERE 0 -- it's\n; VACUUM INTO 'out.db'", false},
		{"quote in block comment", "SELECT 1 /* it's */ ; DELETE FROM t", false},
		{"temp table after comment", "SELECT 1 -- it's\n; CREATE TEMP TABLE z(a)", false},
		{"with insert after newline", "WITH x AS (SELECT 1)\nINSERT INTO t SELECT * FROM x", false},
		{"with delete", "WITH x AS (SELECT 1) DELETE FROM t", false},
		{"with update column list", "WITH x(a) AS (SELECT 1) UPDATE t SET a = 1", false},
		{"pragma assignment", "PRAGMA user_version = 42", false},
		{"pragma call", "PRAGMA user_version(42)", false},
		{"pragma journal_mode write", "PRAGMA journal_mode = WAL", false},
		{"pragma with side effects", "PRAGMA wal_checkpoint", false},
		{"writable_schema", "PRAGMA writable_schema = 1", false},
		{"attach", "ATTACH DATABASE 'x' AS y", false},
		{"begin", "BEGIN", false},
		{"vacuum", "VACUUM", false},
		{"analyze", "ANALYZE", false},
		{"reindex", "REINDEX", false},
		{"injection", "'; DROP TABLE x; --", false},
		{"unterminated", "SELECT 'abc", false},
		{"unknown statement", "FROBNICATE", false},
		{"unicode lookalike keyword", "ſELECT 1", false},
		{"empty", "  ;; -- nothing\n", false},
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

func TestPlanSQL_Categories(t *testing.T) {
	tests := []struct {
		sql      string
		category Category
		label    string
	}{
		{"select 1", CategoryRead, "SELECT"},
		{"WITH x AS (SELECT 1) SELECT 1", CategoryRead, "WITH … SELECT"},
		{"WITH x AS (SELECT 1) REPLACE INTO t VALUES (1)", CategoryData, "WITH … REPLACE"},
		{"INSERT INTO t VALUES (1) ON CONFLICT DO NOTHING", CategoryData, "INSERT"},
		{"DELETE FROM t", CategoryData, "DELETE"},
		{"CREATE TEMP TABLE t(a)", CategorySchema, "CREATE"},
		{"ALTER TABLE t ADD COLUMN b", CategorySchema, "ALTER"},
		{"DROP VIEW v", CategorySchema, "DROP"},
		{"CREATE TRIGGER tr AFTER INSERT ON t BEGIN DELETE FROM u; END", CategorySchema, "CREATE"},
		{"BEGIN IMMEDIATE", CategoryTransaction, "BEGIN"},
		{"END", CategoryTransaction, "END"},
		{"SAVEPOINT a", CategoryTransaction, "SAVEPOINT"},
		{"RELEASE a", CategoryTransaction, "RELEASE"},
		{"VACUUM", CategoryMaintenance, "VACUUM"},
		{"VACUUM main INTO 'copy.db'", CategoryMaintenance, "VACUUM INTO"},
		{"ANALYZE t", CategoryMaintenance, "ANALYZE"},
		{"ATTACH 'x.db' AS x", CategoryAttach, "ATTACH"},
		{"DETACH x", CategoryAttach, "DETACH"},
		{"PRAGMA foreign_keys = ON", CategoryPragma, "PRAGMA foreign_keys"},
		{"pragma Main.User_Version = 3", CategoryPragma, "PRAGMA user_version"},
	}
	all := Policy{}
	for _, c := range []Category{CategoryData, CategorySchema, CategoryTransaction, CategoryMaintenance, CategoryAttach, CategoryPragma} {
		all[c] = true
	}
	for _, tt := range tests {
		plan, err := PlanSQL(tt.sql, all)
		if err != nil {
			t.Fatalf("%q: %v", tt.sql, err)
		}
		if len(plan.Statements) != 1 {
			t.Fatalf("%q: %d statements", tt.sql, len(plan.Statements))
		}
		st := plan.Statements[0]
		if st.Category != tt.category || st.Label != tt.label {
			t.Fatalf("%q: got %s/%q, want %s/%q", tt.sql, st.Category, st.Label, tt.category, tt.label)
		}
	}
}

func TestPlanSQL_PolicyIsPerCategory(t *testing.T) {
	policy, err := ParsePolicy([]string{"data"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := PlanSQL("DELETE FROM t; SELECT 1", policy); err != nil {
		t.Fatalf("data change should be allowed: %v", err)
	}

	_, err = PlanSQL("DELETE FROM t; VACUUM INTO 'copy.db'", policy)
	appErr, ok := apperrors.As(err)
	if !ok || appErr.Code != apperrors.CodeReadOnlyViolation {
		t.Fatalf("expected read-only violation, got %v", err)
	}
	if !strings.Contains(appErr.Message, "Statement 2 of 2") || !strings.Contains(appErr.Message, `"Maintenance"`) {
		t.Fatalf("message should name the statement and permission: %q", appErr.Message)
	}
	if appErr.Detail != string(CategoryMaintenance) {
		t.Fatalf("detail should carry the category id: %q", appErr.Detail)
	}
}

func TestPlanSQL_WritableSchemaNeverAllowed(t *testing.T) {
	policy, err := ParsePolicy([]string{"pragma"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := PlanSQL("PRAGMA writable_schema = ON", policy); err == nil {
		t.Fatal("writable_schema must be rejected even when PRAGMAs are allowed")
	}
	if _, err := PlanSQL("PRAGMA foreign_keys = ON", policy); err != nil {
		t.Fatalf("other pragmas should be allowed: %v", err)
	}
}

func TestParsePolicy_RejectsUnknownCategories(t *testing.T) {
	for _, id := range []string{"read", "forbidden", "everything", ""} {
		if _, err := ParsePolicy([]string{id}); err == nil {
			t.Fatalf("%q: expected error", id)
		}
	}
}

func TestStatementCategories(t *testing.T) {
	cats := StatementCategories()
	seen := map[string]bool{}
	for _, c := range cats {
		seen[c.ID] = true
		_, parseErr := ParsePolicy([]string{c.ID})
		if c.Configurable != (parseErr == nil) {
			t.Fatalf("%s: configurable=%v but ParsePolicy err=%v", c.ID, c.Configurable, parseErr)
		}
		if c.Label == "" || len(c.Statements) == 0 {
			t.Fatalf("%s: missing label or statements", c.ID)
		}
	}
	for _, id := range []string{"read", "data", "schema", "transaction", "maintenance", "attach", "pragma", "forbidden"} {
		if !seen[id] {
			t.Fatalf("missing category %s", id)
		}
	}
	if len(cats[0].Pragmas) == 0 {
		t.Fatal("read category should list the read-only PRAGMAs")
	}
}
