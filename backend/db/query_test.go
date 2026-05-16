package db

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"sqlite-explorer/backend/apperrors"
	"sqlite-explorer/backend/model"
)

func TestGetTableRows_Pagination(t *testing.T) {
	path := filepath.Join(t.TempDir(), "paginate.db")
	sqlDB, err := sql.Open(Driver, path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sqlDB.Exec(`CREATE TABLE nums (i INTEGER PRIMARY KEY, name TEXT NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	tx, err := sqlDB.Begin()
	if err != nil {
		t.Fatal(err)
	}
	stmt, err := tx.Prepare(`INSERT INTO nums (i, name) VALUES (?, ?)`)
	if err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 10000; i++ {
		if _, err := stmt.Exec(i, "row"); err != nil {
			t.Fatal(err)
		}
	}
	stmt.Close()
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	sqlDB.Close()

	conn, err := Open(path, true)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	ctx := context.Background()

	page1, err := conn.GetTableRows(ctx, model.TableRowsRequest{
		Table: "nums", PageSize: 100, Page: 1, WithTotal: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(page1.Rows) != 100 {
		t.Fatalf("page 1 rows: got %d, want 100", len(page1.Rows))
	}
	if page1.Rows[0][0].Kind != "int" || page1.Rows[0][0].Value != int64(1) {
		t.Fatalf("first row id: %+v", page1.Rows[0][0])
	}
	if page1.TotalRows == nil || *page1.TotalRows != 10000 {
		t.Fatalf("total: %+v", page1.TotalRows)
	}

	page100, err := conn.GetTableRows(ctx, model.TableRowsRequest{
		Table: "nums", PageSize: 100, Page: 100,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(page100.Rows) != 100 {
		t.Fatalf("page 100 rows: got %d", len(page100.Rows))
	}
	lastID := page100.Rows[99][0].Value
	if lastID != int64(10000) {
		t.Fatalf("last id on page 100: got %v", lastID)
	}

	page101, err := conn.GetTableRows(ctx, model.TableRowsRequest{
		Table: "nums", PageSize: 100, Page: 101,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(page101.Rows) != 0 {
		t.Fatalf("page 101 rows: got %d, want 0", len(page101.Rows))
	}
}

func TestGetTableRows_Sort(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sort.db")
	sqlDB, err := sql.Open(Driver, path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = sqlDB.Exec(`CREATE TABLE items (id INTEGER PRIMARY KEY, name TEXT NOT NULL)`)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"alpha", "gamma", "beta"} {
		if _, err := sqlDB.Exec(`INSERT INTO items (name) VALUES (?)`, name); err != nil {
			t.Fatal(err)
		}
	}
	sqlDB.Close()

	conn, err := Open(path, true)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	resp, err := conn.GetTableRows(context.Background(), model.TableRowsRequest{
		Table: "items", PageSize: 100, Page: 1, SortColumn: "name", SortDesc: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Rows) != 3 {
		t.Fatalf("rows: %d", len(resp.Rows))
	}
	names := []string{
		resp.Rows[0][1].Value.(string),
		resp.Rows[1][1].Value.(string),
		resp.Rows[2][1].Value.(string),
	}
	if names[0] != "gamma" || names[1] != "beta" || names[2] != "alpha" {
		t.Fatalf("sort order: %v", names)
	}
}

func TestGetTableRows_Filter_LikeEscape(t *testing.T) {
	path := filepath.Join(t.TempDir(), "filter.db")
	sqlDB, err := sql.Open(Driver, path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = sqlDB.Exec(`CREATE TABLE products (id INTEGER PRIMARY KEY, name TEXT NOT NULL)`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = sqlDB.Exec(`INSERT INTO products (name) VALUES ('100%_off'), ('100 items'), ('other')`)
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.Close()

	conn, err := Open(path, true)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	resp, err := conn.GetTableRows(context.Background(), model.TableRowsRequest{
		Table: "products", PageSize: 100, Page: 1, Filter: "100%_",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Rows) != 1 {
		t.Fatalf("filter rows: got %d, want 1", len(resp.Rows))
	}
	if resp.Rows[0][1].Value != "100%_off" {
		t.Fatalf("filtered name: %v", resp.Rows[0][1].Value)
	}
}

func TestGetTableRows_BlobRendering(t *testing.T) {
	path := filepath.Join(t.TempDir(), "blob.db")
	sqlDB, err := sql.Open(Driver, path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = sqlDB.Exec(`CREATE TABLE files (id INTEGER PRIMARY KEY, data BLOB)`)
	if err != nil {
		t.Fatal(err)
	}
	blob := make([]byte, 1024)
	for i := range blob {
		blob[i] = byte(i % 256)
	}
	_, err = sqlDB.Exec(`INSERT INTO files (data) VALUES (?)`, blob)
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.Close()

	conn, err := Open(path, true)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	resp, err := conn.GetTableRows(context.Background(), model.TableRowsRequest{
		Table: "files", PageSize: 100, Page: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	cell := resp.Rows[0][1]
	if cell.Kind != "blob" {
		t.Fatalf("kind: %s", cell.Kind)
	}
	bv, ok := cell.Value.(model.BlobValue)
	if !ok {
		t.Fatalf("value type: %T", cell.Value)
	}
	if bv.Size != 1024 {
		t.Fatalf("size: %d", bv.Size)
	}
	if len(bv.Hex) != 128 {
		t.Fatalf("hex len: %d, want 128", len(bv.Hex))
	}
}

func TestGetTableRows_UnknownTable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "unknown.db")
	sqlDB, err := sql.Open(Driver, path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = sqlDB.Exec(`CREATE TABLE t (id INTEGER)`)
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.Close()

	conn, err := Open(path, true)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	_, err = conn.GetTableRows(context.Background(), model.TableRowsRequest{
		Table: "missing", PageSize: 100, Page: 1,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	appErr, ok := apperrors.As(err)
	if !ok || appErr.Code != apperrors.CodeUnknownTable {
		t.Fatalf("got %v", err)
	}
}

func TestGetTableRows_Fixture(t *testing.T) {
	if err := EnsureFixtureDB(); err != nil {
		t.Fatal(err)
	}
	path, _ := FixtureDBPath()
	conn, err := Open(path, true)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	resp, err := conn.GetTableRows(context.Background(), model.TableRowsRequest{
		Table: "customers", PageSize: 100, Page: 1, WithTotal: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Rows) != 5 {
		t.Fatalf("customers rows: %d", len(resp.Rows))
	}
	if resp.TotalRows == nil || *resp.TotalRows != 5 {
		t.Fatalf("total: %+v", resp.TotalRows)
	}
}
