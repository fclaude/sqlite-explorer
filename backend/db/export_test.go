package db

import (
	"context"
	"database/sql"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
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
	_, err = conn.ExportRowsToCSV(context.Background(), outPath, model.ExportRequest{
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
	_, err = conn.ExportRowsToCSV(context.Background(), outPath, model.ExportRequest{
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
	_, err = conn.ExportRowsToCSV(context.Background(), outPath, model.ExportRequest{
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
	want := "0x" + hex.EncodeToString(blob)
	if records[1][1] != want {
		t.Fatalf("blob: len=%d want full value len=%d", len(records[1][1]), len(want))
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
	_, err = conn.ExportRowsToCSV(context.Background(), outPath, model.ExportRequest{
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

func TestExportCSV_StreamsAllRows(t *testing.T) {
	conn, _ := openScratchDB(t, `CREATE TABLE big (id INTEGER PRIMARY KEY, tag TEXT);
		WITH RECURSIVE c(x) AS (SELECT 1 UNION ALL SELECT x + 1 FROM c WHERE x < 2500)
		INSERT INTO big SELECT x, CASE WHEN x % 2 = 0 THEN 'even' ELSE 'odd' END FROM c`)
	ctx := context.Background()
	dir := t.TempDir()

	n, err := conn.ExportRowsToCSV(ctx, filepath.Join(dir, "query.csv"), model.ExportRequest{
		Source: ExportSourceQueryResult, SQL: "SELECT id FROM big ORDER BY id",
	})
	if err != nil || n != 2500 {
		t.Fatalf("query export: n=%d err=%v", n, err)
	}
	records, _ := ReadCSVFile(filepath.Join(dir, "query.csv"))
	if len(records) != 2501 || records[2500][0] != "2500" {
		t.Fatalf("query export rows: %d", len(records))
	}

	n, err = conn.ExportRowsToCSV(ctx, filepath.Join(dir, "table.csv"), model.ExportRequest{
		Source:    ExportSourceTable,
		TableRows: model.TableRowsRequest{Table: "big", Filter: "even", SortColumn: "id", SortDesc: true},
	})
	if err != nil || n != 1250 {
		t.Fatalf("table export: n=%d err=%v", n, err)
	}
	records, _ = ReadCSVFile(filepath.Join(dir, "table.csv"))
	if records[0][0] != "id" || records[1][0] != "2500" || records[1250][0] != "2" {
		t.Fatalf("table export order/filter: first=%v last=%v", records[1], records[1250])
	}

	n, err = conn.ExportRowsToCSV(ctx, filepath.Join(dir, "page.csv"), model.ExportRequest{
		Source:    ExportSourceTablePage,
		TableRows: model.TableRowsRequest{Table: "big", Page: 2, PageSize: 50},
	})
	if err != nil || n != 50 {
		t.Fatalf("page export: n=%d err=%v", n, err)
	}
	records, _ = ReadCSVFile(filepath.Join(dir, "page.csv"))
	if records[1][0] != "51" {
		t.Fatalf("page export first row: %v", records[1])
	}
}

func TestExportCSV_FailureKeepsExistingFile(t *testing.T) {
	conn, _ := openScratchDB(t, `CREATE TABLE t (a INTEGER); INSERT INTO t VALUES (1), (-9223372036854775808)`)
	out := filepath.Join(t.TempDir(), "existing.csv")
	if err := os.WriteFile(out, []byte("keep me\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// abs() of the smallest integer fails on the second row, after the first was written.
	_, err := conn.ExportRowsToCSV(context.Background(), out, model.ExportRequest{
		Source: ExportSourceQueryResult, SQL: "SELECT abs(a) FROM t",
	})
	if err == nil {
		t.Fatal("expected export error")
	}
	got, _ := os.ReadFile(out)
	if string(got) != "keep me\n" {
		t.Fatalf("existing file changed: %q", got)
	}
	entries, _ := os.ReadDir(filepath.Dir(out))
	if len(entries) != 1 {
		t.Fatalf("partial file left behind: %v", entries)
	}
}

func TestExportCSV_RejectsWritesEvenWhenEditorAllowsThem(t *testing.T) {
	err := ValidateExport(model.ExportRequest{Source: ExportSourceQueryResult, SQL: "SELECT 1; DELETE FROM t"})
	appErr, ok := apperrors.As(err)
	if !ok || appErr.Code != apperrors.CodeReadOnlyViolation || !strings.Contains(appErr.Message, "exporting runs the query again") {
		t.Fatalf("got %v", err)
	}
	if err := ValidateExport(model.ExportRequest{Source: "bogus"}); err == nil {
		t.Fatal("unknown source accepted")
	}
	if err := ValidateExport(model.ExportRequest{Source: ExportSourceTable}); err == nil {
		t.Fatal("missing table accepted")
	}
}

func TestExportCSV_Cancelled(t *testing.T) {
	conn, _ := openScratchDB(t, `CREATE TABLE t (a INTEGER)`)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	out := filepath.Join(t.TempDir(), "cancelled.csv")
	_, err := conn.ExportRowsToCSV(ctx, out, model.ExportRequest{
		Source: ExportSourceQueryResult,
		SQL:    "WITH RECURSIVE c(x) AS (SELECT 1 UNION ALL SELECT x + 1 FROM c) SELECT x FROM c",
	})
	if appErr, ok := apperrors.As(err); !ok || appErr.Code != apperrors.CodeCancelled {
		t.Fatalf("got %v", err)
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Fatal("cancelled export left a file")
	}
}
