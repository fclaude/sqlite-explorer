package db

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"sqlite-explorer/backend/apperrors"
	"sqlite-explorer/backend/model"
)

func TestExportCSV_RoundTrip(t *testing.T) {
	if err := EnsureFixtureDB(); err != nil {
		t.Fatal(err)
	}
	path, _ := FixtureDBPath()
	conn, err := Open(path, true)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	outPath := filepath.Join(t.TempDir(), "roundtrip.csv")
	err = conn.ExportRowsToCSV(context.Background(), outPath, model.ExportRequest{
		Source: ExportSourceTablePage,
		TableRows: model.TableRowsRequest{
			Table: "customers", PageSize: 100, Page: 1,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	records, err := ReadCSVFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 6 {
		t.Fatalf("rows: got %d, want 6 (header+5)", len(records))
	}
	if records[0][0] != "id" || records[0][2] != "email" {
		t.Fatalf("header: %v", records[0])
	}
	if records[1][1] != "Example Customer 001" {
		t.Fatalf("first row: %v", records[1])
	}
}

func TestExportCSV_SpecialChars(t *testing.T) {
	path := filepath.Join(t.TempDir(), "special.db")
	sqlDB, err := sql.Open(Driver, path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = sqlDB.Exec(`CREATE TABLE t (txt TEXT); INSERT INTO t VALUES ('"héllo"'), ('a,b'), ('line' || char(10) || 'break')`)
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.Close()

	conn, err := Open(path, true)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	outPath := filepath.Join(t.TempDir(), "special.csv")
	err = conn.ExportRowsToCSV(context.Background(), outPath, model.ExportRequest{
		Source: ExportSourceQueryResult,
		SQL:    `SELECT txt FROM t`,
	})
	if err != nil {
		t.Fatal(err)
	}

	records, err := ReadCSVFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 4 {
		t.Fatalf("got %d records", len(records))
	}
	if records[1][0] != `"héllo"` {
		t.Fatalf("quoted: %q", records[1][0])
	}
	if records[2][0] != "a,b" {
		t.Fatalf("comma: %q", records[2][0])
	}
	if records[3][0] != "line\nbreak" {
		t.Fatalf("newline: %q", records[3][0])
	}
}

func TestExportCSV_NullsAndBlobs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "blob.db")
	sqlDB, err := sql.Open(Driver, path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = sqlDB.Exec(`CREATE TABLE t (n TEXT, b BLOB)`)
	if err != nil {
		t.Fatal(err)
	}
	blob := make([]byte, 1024)
	for i := range blob {
		blob[i] = byte(i % 256)
	}
	_, err = sqlDB.Exec(`INSERT INTO t VALUES (NULL, ?)`, blob)
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.Close()

	conn, err := Open(path, true)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	outPath := filepath.Join(t.TempDir(), "blob.csv")
	err = conn.ExportRowsToCSV(context.Background(), outPath, model.ExportRequest{
		Source: ExportSourceQueryResult,
		SQL:    `SELECT n, b FROM t`,
	})
	if err != nil {
		t.Fatal(err)
	}

	records, err := ReadCSVFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 2 {
		t.Fatal(records)
	}
	if records[1][0] != "" {
		t.Fatalf("null: %q", records[1][0])
	}
	want := FormatBlobCSV(blob)
	if records[1][1] != want {
		t.Fatalf("blob: len=%d want len=%d", len(records[1][1]), len(want))
	}
	if len(records[1][1]) != len("0x")+128+len("...(1024 bytes)") {
		t.Fatalf("blob format: %s", records[1][1][:20])
	}
}

func TestExportCSV_HonorsReadOnly(t *testing.T) {
	if err := EnsureFixtureDB(); err != nil {
		t.Fatal(err)
	}
	dbPath, _ := FixtureDBPath()
	conn, err := Open(dbPath, true)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	outPath := filepath.Join(t.TempDir(), "should-not-exist.csv")
	err = conn.ExportRowsToCSV(context.Background(), outPath, model.ExportRequest{
		Source: ExportSourceQueryResult,
		SQL:    `DELETE FROM customers`,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	appErr, ok := apperrors.As(err)
	if !ok || appErr.Code != apperrors.CodeReadOnlyViolation {
		t.Fatalf("got %v", err)
	}
	if _, err := os.Stat(outPath); !os.IsNotExist(err) {
		t.Fatal("file should not exist")
	}
}
