package db

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"sqlite-explorer/backend/apperrors"
)

func TestMain(m *testing.M) {
	if err := EnsureFixtureDB(); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

func sampleDBPath(t *testing.T) string {
	t.Helper()
	root, err := findModuleRoot()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(root, "testdata", "sample.sqlite")
}

func TestOpen_ValidSqlite(t *testing.T) {
	conn, err := Open(sampleDBPath(t), false)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer conn.Close()

	if conn.ReadOnly() {
		t.Fatal("expected writable connection for readOnly=false")
	}
	if conn.Path() == "" {
		t.Fatal("expected non-empty path")
	}

	dsn := buildDSN(conn.Path(), false)
	if strings.Contains(dsn, "mode=ro") {
		t.Fatalf("writable DSN should not include mode=ro: %q", dsn)
	}
}

func TestOpen_ReadOnlyDSN(t *testing.T) {
	conn, err := Open(sampleDBPath(t), true)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer conn.Close()

	if !conn.ReadOnly() {
		t.Fatal("expected read-only connection")
	}
	dsn := buildDSN(conn.Path(), true)
	if !strings.Contains(dsn, "mode=ro") {
		t.Fatalf("DSN missing read-only params: %q", dsn)
	}
	if strings.Contains(dsn, "immutable=1") {
		t.Fatalf("read-only DSN must observe WAL and external changes: %q", dsn)
	}
}

func TestOpen_NotSqlite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fake.db")
	if err := os.WriteFile(path, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Open(path, true)
	if err == nil {
		t.Fatal("expected error for non-sqlite file")
	}
	appErr, ok := apperrors.As(err)
	if !ok || appErr.Code != apperrors.CodeNotSQLite {
		t.Fatalf("expected NOT_SQLITE, got %v", err)
	}
}

func TestOpen_Missing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "does-not-exist.db")
	_, err := Open(path, true)
	if err == nil {
		t.Fatal("expected error")
	}
	appErr, ok := apperrors.As(err)
	if !ok {
		t.Fatalf("expected apperrors.Error, got %v", err)
	}
	if appErr.Message != "The database file could not be found." {
		t.Fatalf("unexpected message: %q", appErr.Message)
	}
}
