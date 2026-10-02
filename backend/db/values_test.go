package db

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"sqlite-explorer/backend/apperrors"
	"sqlite-explorer/backend/model"
)

func updateOne(t *testing.T, conn *DB, upd model.ColumnUpdate) (model.UpdateTableRowResponse, error) {
	t.Helper()
	return conn.UpdateTableRow(context.Background(), model.UpdateTableRowRequest{
		Table: "t", RowID: "1", Updates: []model.ColumnUpdate{upd},
	})
}

func storedValue(t *testing.T, conn *DB, column string) (string, any) {
	t.Helper()
	var typ string
	var v any
	q := fmt.Sprintf(`SELECT typeof(%[1]s), %[1]s FROM t WHERE rowid = 1`, column)
	if err := conn.sql.QueryRow(q).Scan(&typ, &v); err != nil {
		t.Fatal(err)
	}
	return typ, v
}

func TestUpdateTableRow_TextThatLooksLikeHexStaysText(t *testing.T) {
	conn := openUpdateTestDB(t, `CREATE TABLE t (name TEXT, n INTEGER); INSERT INTO t VALUES ('a', 1)`)
	for _, text := range []string{"0xcafe", "0x52908400098527886E0F7030069857D2E4169EE7", "0x1g"} {
		if _, err := updateOne(t, conn, model.ColumnUpdate{Column: "name", Text: text}); err != nil {
			t.Fatalf("%q: %v", text, err)
		}
		if typ, v := storedValue(t, conn, "name"); typ != "text" || v != text {
			t.Fatalf("%q stored as %s %v", text, typ, v)
		}
	}
	if _, err := updateOne(t, conn, model.ColumnUpdate{Column: "n", Text: "0x10"}); err == nil {
		t.Fatal("hex text must not be accepted as an integer")
	}
}

func TestUpdateTableRow_HexEncodedBlob(t *testing.T) {
	conn := openUpdateTestDB(t, `CREATE TABLE t (data BLOB, name TEXT); INSERT INTO t VALUES (x'00', 'a')`)

	resp, err := updateOne(t, conn, model.ColumnUpdate{Column: "data", Text: "0xDE AD\nbe ef", Encoding: EncodingHex})
	if err != nil {
		t.Fatal(err)
	}
	if typ, v := storedValue(t, conn, "data"); typ != "blob" || !bytes.Equal(v.([]byte), []byte{0xde, 0xad, 0xbe, 0xef}) {
		t.Fatalf("stored %s %v", typ, v)
	}
	if resp.Cells[0].Kind != "blob" {
		t.Fatalf("returned cell: %+v", resp.Cells[0])
	}

	// Hex encoding applies to the field, whatever the column's declared type.
	if _, err := updateOne(t, conn, model.ColumnUpdate{Column: "name", Text: "41", Encoding: EncodingHex}); err != nil {
		t.Fatal(err)
	}
	if typ, _ := storedValue(t, conn, "name"); typ != "blob" {
		t.Fatalf("name stored as %s", typ)
	}

	if _, err := updateOne(t, conn, model.ColumnUpdate{Column: "data", Text: "abc", Encoding: "base64"}); err == nil {
		t.Fatal("unknown encoding accepted")
	}
}

func TestUpdateTableRow_TruncatedBlobPreviewIsRejected(t *testing.T) {
	conn := openUpdateTestDB(t, `CREATE TABLE t (data BLOB); INSERT INTO t VALUES (randomblob(200))`)
	rows, err := conn.GetTableRows(context.Background(), model.TableRowsRequest{Table: "t", PageSize: 50})
	if err != nil {
		t.Fatal(err)
	}
	preview := rows.Rows[0][0].Value.(model.BlobValue)
	// What the cell editor showed before this fix: 64 bytes of hex plus a note.
	draft := "0x00" + preview.Hex[2:] + "\n\n[Preview: first 64 of 200 bytes.]"

	_, err = updateOne(t, conn, model.ColumnUpdate{Column: "data", Text: draft, Encoding: EncodingHex})
	if err == nil {
		t.Fatal("expected the preview note to be rejected as invalid hex")
	}
	if _, v := storedValue(t, conn, "data"); len(v.([]byte)) != 200 {
		t.Fatalf("blob length changed to %d", len(v.([]byte)))
	}
}

func TestUpdateTableRow_TypeAffinity(t *testing.T) {
	conn := openUpdateTestDB(t, `CREATE TABLE t (
		i INTEGER, fp "FLOATING POINT", r REAL, num NUMERIC, d DATE, untyped, b BLOB
	); INSERT INTO t VALUES (0, 0, 0, 0, '2000-01-01', NULL, NULL)`)

	tests := []struct {
		column, text, wantType, wantValue string
	}{
		{"i", " 42 ", "integer", "42"},
		{"i", "1e3", "integer", "1000"},
		{"i", "2.5", "real", "2.5"},
		{"fp", "2.75", "real", "2.75"}, // INTEGER affinity in SQLite, but still a number
		{"r", "7", "real", "7.0"},
		{"num", "42", "integer", "42"},
		{"num", "n/a", "text", "n/a"},
		{"d", "2024-01-15", "text", "2024-01-15"},
		{"untyped", "0042", "text", "0042"},
		{"b", "plain text", "text", "plain text"},
	}
	for _, tt := range tests {
		if _, err := updateOne(t, conn, model.ColumnUpdate{Column: tt.column, Text: tt.text}); err != nil {
			t.Fatalf("%s=%q: %v", tt.column, tt.text, err)
		}
		var typ, value string
		q := fmt.Sprintf(`SELECT typeof(%[1]s), CAST(%[1]s AS TEXT) FROM t WHERE rowid = 1`, tt.column)
		if err := conn.sql.QueryRow(q).Scan(&typ, &value); err != nil {
			t.Fatal(err)
		}
		if typ != tt.wantType || value != tt.wantValue {
			t.Fatalf("%s=%q stored as %s %q, want %s %q", tt.column, tt.text, typ, value, tt.wantType, tt.wantValue)
		}
	}

	for _, bad := range []string{"abc", "Inf", "NaN", "0x1p3", "1e999", ""} {
		_, err := updateOne(t, conn, model.ColumnUpdate{Column: "i", Text: bad})
		if appErr, ok := apperrors.As(err); !ok || appErr.Code != apperrors.CodeMalformedSQL {
			t.Fatalf("i=%q: expected invalid number, got %v", bad, err)
		}
	}
}

func TestUpdateTableRow_ColumnNamesAreCaseInsensitive(t *testing.T) {
	conn := openUpdateTestDB(t, `CREATE TABLE t (Name TEXT); INSERT INTO t VALUES ('a')`)
	resp, err := updateOne(t, conn, model.ColumnUpdate{Column: "NAME", Text: "b"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Columns[0].Name != "Name" || resp.Cells[0].Value != "b" {
		t.Fatalf("response: %+v", resp)
	}
	_, err = conn.UpdateTableRow(context.Background(), model.UpdateTableRowRequest{
		Table: "t", RowID: "1",
		Updates: []model.ColumnUpdate{{Column: "name", Text: "c"}, {Column: "NAME", Text: "d"}},
	})
	if appErr, ok := apperrors.As(err); !ok || appErr.Code != apperrors.CodeInvalidColumn {
		t.Fatalf("expected duplicate column error, got %v", err)
	}
}

func TestDateColumnsKeepStoredText(t *testing.T) {
	conn := openUpdateTestDB(t, `CREATE TABLE t (id INTEGER PRIMARY KEY, d DATE, ts DATETIME, stamp TIMESTAMP);
		INSERT INTO t VALUES (1, '2024-01-15', '2024-01-15T10:30:00Z', '2024-01-15 10:30:00.250+02:00')`)
	ctx := context.Background()
	want := []string{"2024-01-15", "2024-01-15T10:30:00Z", "2024-01-15 10:30:00.250+02:00"}

	rows, err := conn.GetTableRows(ctx, model.TableRowsRequest{Table: "t", PageSize: 50})
	if err != nil {
		t.Fatal(err)
	}
	for i, w := range want {
		if got := rows.Rows[0][i+1]; got.Kind != "text" || got.Value != w {
			t.Fatalf("browse column %d: %+v, want %q", i+1, got, w)
		}
	}

	resp, err := conn.UpdateTableRow(ctx, model.UpdateTableRowRequest{
		Table: "t", RowID: "1", Updates: []model.ColumnUpdate{{Column: "d", Text: "2024-02-01"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Cells[1].Value != "2024-02-01" || resp.Cells[2].Value != want[1] {
		t.Fatalf("returned cells: %+v", resp.Cells)
	}

	out := filepath.Join(t.TempDir(), "dates.csv")
	if _, err := conn.ExportRowsToCSV(ctx, out, model.ExportRequest{
		Source: ExportSourceTable, TableRows: model.TableRowsRequest{Table: "t"},
	}); err != nil {
		t.Fatal(err)
	}
	records, err := ReadCSVFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(records[1], "|") != "1|2024-02-01|"+want[1]+"|"+want[2] {
		t.Fatalf("export: %v", records[1])
	}

	// Ad-hoc queries go through the driver's date parsing; the result must still be readable.
	q, err := conn.RunQuery(ctx, "SELECT ts, stamp FROM t", ReadOnlyPolicy)
	if err != nil {
		t.Fatal(err)
	}
	if q.Rows[0][0].Value != "2024-01-15 10:30:00" || q.Rows[0][1].Value != "2024-01-15 10:30:00.25+02:00" {
		t.Fatalf("query cells: %+v", q.Rows[0])
	}
}

func TestBlobCellsAreBlobsEvenWhenValidUTF8(t *testing.T) {
	conn := openUpdateTestDB(t, `CREATE TABLE t (name TEXT); INSERT INTO t VALUES (CAST('ABC' AS BLOB))`)
	rows, err := conn.GetTableRows(context.Background(), model.TableRowsRequest{Table: "t", PageSize: 50})
	if err != nil {
		t.Fatal(err)
	}
	cell := rows.Rows[0][0]
	if cell.Kind != "blob" || cell.Value.(model.BlobValue).Hex != hex.EncodeToString([]byte("ABC")) {
		t.Fatalf("cell: %+v", cell)
	}
}

func TestRowCountCacheSeesExternalWrites(t *testing.T) {
	conn, path := openScratchDB(t, `CREATE TABLE t (a TEXT); INSERT INTO t VALUES ('x'), ('y')`)
	ctx := context.Background()
	total := func(filter string) int64 {
		t.Helper()
		resp, err := conn.GetTableRows(ctx, model.TableRowsRequest{Table: "t", PageSize: 50, Filter: filter, WithTotal: true})
		if err != nil {
			t.Fatal(err)
		}
		return *resp.TotalRows
	}
	if n := total(""); n != 2 {
		t.Fatalf("total: %d", n)
	}

	other, err := sql.Open(Driver, path)
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	if _, err := other.Exec(`INSERT INTO t VALUES ('z')`); err != nil {
		t.Fatal(err)
	}
	if n := total(""); n != 3 {
		t.Fatalf("total after external insert: %d", n)
	}

	if _, err := conn.UpdateTableRow(ctx, model.UpdateTableRowRequest{
		Table: "t", RowID: "1", Updates: []model.ColumnUpdate{{Column: "a", Text: "zz"}},
	}); err != nil {
		t.Fatal(err)
	}
	if n := total("z"); n != 2 {
		t.Fatalf("filtered total after edit: %d", n)
	}

	for i := 0; i < maxCachedCounts+50; i++ {
		total(fmt.Sprintf("filter-%d", i))
	}
	conn.countMu.Lock()
	size := len(conn.rowCountCache)
	conn.countMu.Unlock()
	if size > maxCachedCounts {
		t.Fatalf("cache grew to %d entries", size)
	}
}
