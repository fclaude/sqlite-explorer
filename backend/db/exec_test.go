package db

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"sqlite-explorer/backend/apperrors"
)

// openScratchDB creates a database from setup SQL and opens it read-write.
func openScratchDB(t *testing.T, setup string) (*DB, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "scratch.db")
	raw, err := sql.Open(Driver, path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := raw.Exec(setup); err != nil {
		t.Fatal(err)
	}
	if err := raw.Close(); err != nil {
		t.Fatal(err)
	}
	conn, err := Open(path, false)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn, path
}

func mustPolicy(t *testing.T, ids ...string) Policy {
	t.Helper()
	p, err := ParsePolicy(ids)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func countRowsIn(t *testing.T, conn *DB, table string) int64 {
	t.Helper()
	var n int64
	if err := conn.sql.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func appError(t *testing.T, err error) *apperrors.Error {
	t.Helper()
	appErr, ok := apperrors.As(err)
	if !ok {
		t.Fatalf("expected app error, got %v", err)
	}
	return appErr
}

func TestRunQuery_WriteRunReturnsLastStatementAndAffectedRows(t *testing.T) {
	conn, _ := openScratchDB(t, `CREATE TABLE t (a INTEGER)`)
	resp, err := conn.RunQuery(context.Background(),
		"INSERT INTO t VALUES (1); INSERT INTO t VALUES (2), (3); SELECT COUNT(*) AS n FROM t",
		mustPolicy(t, "data"))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatementCount != 3 || resp.RowsAffected != 3 || !resp.Changed || resp.SchemaChanged {
		t.Fatalf("response: %+v", resp)
	}
	if len(resp.Columns) != 1 || resp.Columns[0].Name != "n" || resp.Rows[0][0].Value != int64(3) {
		t.Fatalf("last statement result: %+v %+v", resp.Columns, resp.Rows)
	}

	resp, err = conn.RunQuery(context.Background(), "UPDATE t SET a = a + 1 WHERE a > 1", mustPolicy(t, "data"))
	if err != nil {
		t.Fatal(err)
	}
	if resp.RowsAffected != 2 || len(resp.Columns) != 0 {
		t.Fatalf("update response: %+v", resp)
	}

	resp, err = conn.RunQuery(context.Background(), "DELETE FROM t WHERE a = 1 RETURNING a", mustPolicy(t, "data"))
	if err != nil {
		t.Fatal(err)
	}
	if resp.RowsAffected != 1 || len(resp.Rows) != 1 || resp.Rows[0][0].Value != int64(1) {
		t.Fatalf("returning response: %+v", resp)
	}
}

func TestRunQuery_WritesRejectedWithoutPermission(t *testing.T) {
	conn, _ := openScratchDB(t, `CREATE TABLE t (a INTEGER)`)
	_, err := conn.RunQuery(context.Background(), "INSERT INTO t VALUES (1)", ReadOnlyPolicy)
	if appError(t, err).Code != apperrors.CodeReadOnlyViolation {
		t.Fatalf("got %v", err)
	}
	if n := countRowsIn(t, conn, "t"); n != 0 {
		t.Fatalf("rows written: %d", n)
	}
}

func TestRunQuery_ReadOnlyDatabaseRejectsWritesEvenWhenAllowed(t *testing.T) {
	_, path := openScratchDB(t, `CREATE TABLE t (a INTEGER)`)
	ro, err := Open(path, true)
	if err != nil {
		t.Fatal(err)
	}
	defer ro.Close()
	_, err = ro.RunQuery(context.Background(), "INSERT INTO t VALUES (1)", mustPolicy(t, "data"))
	if appError(t, err).Code != apperrors.CodeReadOnlyViolation {
		t.Fatalf("got %v", err)
	}
}

func TestRunQuery_SchemaChangeIsReported(t *testing.T) {
	conn, _ := openScratchDB(t, `CREATE TABLE t (a INTEGER)`)
	resp, err := conn.RunQuery(context.Background(), "CREATE TABLE u (b TEXT)", mustPolicy(t, "schema"))
	if err != nil {
		t.Fatal(err)
	}
	if !resp.SchemaChanged || !resp.Changed {
		t.Fatalf("response: %+v", resp)
	}
	schema, err := conn.GetSchema(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(schema.Tables) != 2 {
		t.Fatalf("tables: %+v", schema.Tables)
	}
}

func TestRunQuery_ConnectionStateDoesNotLeakBetweenRuns(t *testing.T) {
	conn, path := openScratchDB(t, `CREATE TABLE t (a INTEGER)`)
	ctx := context.Background()

	// Temporary objects exist only for the run that created them.
	resp, err := conn.RunQuery(ctx, "CREATE TEMP TABLE scratch (x); INSERT INTO scratch VALUES (1); SELECT * FROM scratch",
		mustPolicy(t, "schema", "data"))
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Rows) != 1 {
		t.Fatalf("rows: %+v", resp.Rows)
	}
	if _, err := conn.RunQuery(ctx, "CREATE TEMP TABLE scratch (x)", mustPolicy(t, "schema")); err != nil {
		t.Fatalf("temp table leaked into a later run: %v", err)
	}
	if _, err := conn.RunQuery(ctx, "SELECT * FROM temp.scratch", ReadOnlyPolicy); err == nil {
		t.Fatal("temp table visible to read-only runs")
	}

	// Attached databases are detached when the run ends.
	other := filepath.Join(filepath.Dir(path), "other.db")
	_, err = conn.RunQuery(ctx, "ATTACH '"+other+"' AS other; CREATE TABLE other.o (x)", mustPolicy(t, "attach", "schema"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(other); err != nil {
		t.Fatalf("attached database was not created: %v", err)
	}
	if _, err := conn.RunQuery(ctx, "SELECT * FROM other.o", ReadOnlyPolicy); err == nil {
		t.Fatal("attachment leaked into a later run")
	}
	if _, err := conn.RunQuery(ctx, "INSERT INTO other.o VALUES (1)", mustPolicy(t, "data")); err == nil {
		t.Fatal("attachment leaked into the read-write pool")
	}
}

func TestRunQuery_UncommittedTransactionIsRolledBack(t *testing.T) {
	conn, _ := openScratchDB(t, `CREATE TABLE t (a INTEGER)`)
	ctx := context.Background()

	_, err := conn.RunQuery(ctx, "BEGIN; INSERT INTO t VALUES (1)", mustPolicy(t, "transaction", "data"))
	if appError(t, err).Code != apperrors.CodeTransactionRolledBack {
		t.Fatalf("got %v", err)
	}
	if n := countRowsIn(t, conn, "t"); n != 0 {
		t.Fatalf("uncommitted rows kept: %d", n)
	}
	// The rolled-back connection must not hold a write lock.
	if _, err := conn.RunQuery(ctx, "BEGIN; INSERT INTO t VALUES (2); COMMIT", mustPolicy(t, "transaction", "data")); err != nil {
		t.Fatal(err)
	}
	if n := countRowsIn(t, conn, "t"); n != 1 {
		t.Fatalf("committed rows: %d", n)
	}
}

func TestRunQuery_PartialFailureExplainsWhatWasApplied(t *testing.T) {
	conn, _ := openScratchDB(t, `CREATE TABLE t (a INTEGER)`)
	ctx := context.Background()

	_, err := conn.RunQuery(ctx, "INSERT INTO t VALUES (1); INSERT INTO missing VALUES (2)", mustPolicy(t, "data"))
	appErr := appError(t, err)
	if !strings.Contains(appErr.Message, "Statement 2 of 2 (INSERT) failed") ||
		!strings.Contains(appErr.Message, "Earlier statements in this run were applied") {
		t.Fatalf("message: %q", appErr.Message)
	}
	if !strings.Contains(appErr.Detail, "no such table") {
		t.Fatalf("detail should carry the SQLite error: %q", appErr.Detail)
	}
	if n := countRowsIn(t, conn, "t"); n != 1 {
		t.Fatalf("rows: %d", n)
	}

	_, err = conn.RunQuery(ctx, "BEGIN; INSERT INTO t VALUES (3); INSERT INTO missing VALUES (4); COMMIT",
		mustPolicy(t, "data", "transaction"))
	appErr = appError(t, err)
	if !strings.Contains(appErr.Message, "Statement 3 of 4") || !strings.Contains(appErr.Message, "rolled back") {
		t.Fatalf("message: %q", appErr.Message)
	}
	if n := countRowsIn(t, conn, "t"); n != 1 {
		t.Fatalf("rows after rolled-back transaction: %d", n)
	}
}

func TestRunQuery_SyntaxErrorDetail(t *testing.T) {
	conn, _ := openScratchDB(t, `CREATE TABLE t (a INTEGER)`)
	_, err := conn.RunQuery(context.Background(), "SELECT a FORM t", ReadOnlyPolicy)
	appErr := appError(t, err)
	if appErr.Code != apperrors.CodeMalformedSQL || !strings.Contains(appErr.Detail, "syntax error") {
		t.Fatalf("got %+v", appErr)
	}
}

func TestRunQuery_MaintenanceAllowed(t *testing.T) {
	conn, path := openScratchDB(t, `CREATE TABLE t (a INTEGER); INSERT INTO t VALUES (1)`)
	out := filepath.Join(filepath.Dir(path), "copy.db")
	if _, err := conn.RunQuery(context.Background(), "VACUUM INTO '"+out+"'", mustPolicy(t, "maintenance")); err != nil {
		t.Fatal(err)
	}
	copyDB, err := Open(out, true)
	if err != nil {
		t.Fatal(err)
	}
	defer copyDB.Close()
	if n := countRowsIn(t, copyDB, "t"); n != 1 {
		t.Fatalf("copy rows: %d", n)
	}
}

// The validator is the first line of defence; the read-only connection must still refuse
// anything that slips through as a "read".
func TestExecPlan_ReadOnlyConnectionBlocksMisclassifiedStatements(t *testing.T) {
	conn, path := openScratchDB(t, `CREATE TABLE t (a INTEGER); INSERT INTO t VALUES (1)`)
	out := filepath.Join(filepath.Dir(path), "copy.db")
	for _, stmt := range []string{
		"VACUUM INTO '" + out + "'",
		"CREATE TEMP TABLE z (a)",
		"INSERT INTO t VALUES (2)",
		"PRAGMA user_version = 7",
	} {
		plan := Plan{Statements: []Statement{{SQL: stmt, Category: CategoryRead, Label: "SELECT"}}}
		if _, err := conn.execPlan(context.Background(), plan, func(*sql.Rows) error { return nil }); err == nil {
			t.Fatalf("%q ran on the read-only connection", stmt)
		}
	}
	if info, err := os.Stat(out); err == nil && info.Size() > 0 {
		t.Fatalf("VACUUM INTO wrote %d bytes from the read-only connection", info.Size())
	}
	if n := countRowsIn(t, conn, "t"); n != 1 {
		t.Fatalf("rows: %d", n)
	}
	var version int
	if err := conn.sql.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil || version != 0 {
		t.Fatalf("user_version = %d, err %v", version, err)
	}
}
