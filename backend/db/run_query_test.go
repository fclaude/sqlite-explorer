package db

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"sqlite-explorer/backend/apperrors"
)

func TestRunQuery_BlocksWriteEvenIfValidatorBypassed(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ro.db")
	sqlDB, err := sql.Open(Driver, path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = sqlDB.Exec(`CREATE TABLE t (id INTEGER)`)
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.Close()

	conn, err := Open(path, false)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	// Misclassify a write as a read: the read-only connection must still refuse it.
	plan := Plan{Statements: []Statement{{SQL: `INSERT INTO t VALUES (1)`, Category: CategoryRead, Label: "SELECT"}}}
	_, err = conn.execPlan(context.Background(), plan, func(*sql.Rows) error { return nil })
	if err == nil {
		t.Fatal("expected engine-level readonly rejection")
	}
	var count int
	if err := conn.sql.QueryRow(`SELECT COUNT(*) FROM t`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("read-only query connection wrote %d rows", count)
	}
}

func TestRunQuery_ReadOnlyConnectionSeesWrites(t *testing.T) {
	path := filepath.Join(t.TempDir(), "visible.db")
	sqlDB, err := sql.Open(Driver, path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sqlDB.Exec(`CREATE TABLE t (id INTEGER)`); err != nil {
		t.Fatal(err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatal(err)
	}

	conn, err := Open(path, false)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	if _, err := conn.sql.Exec(`INSERT INTO t VALUES (7)`); err != nil {
		t.Fatal(err)
	}
	resp, err := conn.RunQuery(context.Background(), `SELECT id FROM t`, ReadOnlyPolicy)
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Rows) != 1 || resp.Rows[0][0].Value != int64(7) {
		t.Fatalf("query connection did not observe write: %+v", resp.Rows)
	}
}

func TestRunQuery_TimesOut(t *testing.T) {
	if err := EnsureFixtureDB(); err != nil {
		t.Fatal(err)
	}
	path, _ := FixtureDBPath()
	conn, err := Open(path, true)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_, err = conn.RunQuery(ctx, `WITH RECURSIVE c(x) AS (
		SELECT 1 UNION ALL SELECT x+1 FROM c
	) SELECT count(*) FROM c`, ReadOnlyPolicy)
	if err == nil {
		t.Fatal("expected timeout")
	}
	appErr, ok := apperrors.As(err)
	if !ok || appErr.Code != apperrors.CodeTimeout {
		t.Fatalf("got %v", err)
	}
}

func TestRunQuery_Truncation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "trunc.db")
	sqlDB, err := sql.Open(Driver, path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = sqlDB.Exec(`CREATE TABLE big (id INTEGER PRIMARY KEY)`)
	if err != nil {
		t.Fatal(err)
	}
	tx, _ := sqlDB.Begin()
	stmt, _ := tx.Prepare(`INSERT INTO big (id) VALUES (?)`)
	for i := 1; i <= 5000; i++ {
		if _, err := stmt.Exec(i); err != nil {
			t.Fatal(err)
		}
	}
	stmt.Close()
	tx.Commit()
	sqlDB.Close()

	conn, err := Open(path, true)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	resp, err := conn.RunQuery(context.Background(), `SELECT id FROM big`, ReadOnlyPolicy)
	if err != nil {
		t.Fatal(err)
	}
	if resp.RowCount != MaxQueryRows {
		t.Fatalf("row count: %d", resp.RowCount)
	}
	if !resp.Truncated {
		t.Fatal("expected truncated")
	}
}

func TestRunQuery_ColumnOrderStable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "order.db")
	sqlDB, err := sql.Open(Driver, path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = sqlDB.Exec(`CREATE TABLE t (a INT, b INT); INSERT INTO t VALUES (1,2)`)
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.Close()

	conn, err := Open(path, true)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	resp, err := conn.RunQuery(context.Background(), `SELECT b, a FROM t`, ReadOnlyPolicy)
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Columns) != 2 || resp.Columns[0].Name != "b" || resp.Columns[1].Name != "a" {
		t.Fatalf("columns: %+v", resp.Columns)
	}
	if len(resp.Rows) != 1 || resp.Rows[0][0].Value != int64(2) || resp.Rows[0][1].Value != int64(1) {
		t.Fatalf("rows: %+v", resp.Rows[0])
	}
}

func TestRunQuery_AllValueKinds(t *testing.T) {
	path := filepath.Join(t.TempDir(), "kinds.db")
	sqlDB, err := sql.Open(Driver, path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = sqlDB.Exec(`CREATE TABLE t (
		i INTEGER, r REAL, txt TEXT, b BLOB
	)`)
	if err != nil {
		t.Fatal(err)
	}
	blob := []byte{0, 1, 2}
	_, err = sqlDB.Exec(`INSERT INTO t VALUES (NULL, 3.14, 'hi', ?)`, blob)
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.Close()

	conn, err := Open(path, true)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	resp, err := conn.RunQuery(context.Background(), `SELECT i, r, txt, b FROM t`, ReadOnlyPolicy)
	if err != nil {
		t.Fatal(err)
	}
	row := resp.Rows[0]
	if row[0].Kind != "null" {
		t.Fatalf("null: %+v", row[0])
	}
	if row[1].Kind != "real" {
		t.Fatalf("real: %+v", row[1])
	}
	if row[2].Kind != "text" || row[2].Value != "hi" {
		t.Fatalf("text: %+v", row[2])
	}
	if row[3].Kind != "blob" {
		t.Fatalf("blob: %+v", row[3])
	}
}

func TestRunQuery_ReadOnlyRejectsDrop(t *testing.T) {
	if err := EnsureFixtureDB(); err != nil {
		t.Fatal(err)
	}
	path, _ := FixtureDBPath()
	conn, err := Open(path, true)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	_, err = conn.RunQuery(context.Background(), `DROP TABLE customers`, ReadOnlyPolicy)
	if err == nil {
		t.Fatal("expected error")
	}
	appErr, ok := apperrors.As(err)
	if !ok || appErr.Code != apperrors.CodeReadOnlyViolation {
		t.Fatalf("got %v", err)
	}
	if !strings.Contains(appErr.Message, "DROP") {
		t.Fatalf("message: %s", appErr.Message)
	}
}
