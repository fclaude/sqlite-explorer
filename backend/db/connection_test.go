package db

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"sqlite-explorer/backend/apperrors"
)

func TestMain(m *testing.M) {
	if err := ensureSampleDB(); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

func ensureSampleDB() error {
	root, err := findModuleRoot()
	if err != nil {
		return err
	}
	path := filepath.Join(root, "testdata", "sample.sqlite")
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	sqlDB, err := sql.Open(Driver, path)
	if err != nil {
		return err
	}
	defer sqlDB.Close()
	_, err = sqlDB.Exec(`CREATE TABLE sample (id INTEGER PRIMARY KEY, name TEXT NOT NULL)`)
	if err != nil {
		return err
	}
	_, err = sqlDB.Exec(`INSERT INTO sample (id, name) VALUES (1, 'one')`)
	return err
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
	conn, err := Open(sampleDBPath(t), true)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer conn.Close()

	if !conn.ReadOnly() {
		t.Fatal("expected read-only connection")
	}
	if conn.Path() == "" {
		t.Fatal("expected non-empty path")
	}

	dsn := buildDSN(conn.Path(), true)
	if !strings.Contains(dsn, "mode=ro") || !strings.Contains(dsn, "immutable=1") {
		t.Fatalf("DSN missing read-only params: %q", dsn)
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
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected fs.ErrNotExist wrap, got %v", err)
	}
}
