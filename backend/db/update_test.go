package db

import (
	"context"
	"database/sql"
	"testing"

	"sqlite-explorer/backend/apperrors"
	"sqlite-explorer/backend/model"
)

func openUpdateTestDB(t *testing.T, schema string) *DB {
	t.Helper()
	path := t.TempDir() + "/update.sqlite"
	raw, err := sql.Open(Driver, path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := raw.Exec(schema); err != nil {
		_ = raw.Close()
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
	return conn
}

func TestUpdateTableRow_UpdatesTextColumn(t *testing.T) {
	if err := EnsureFixtureDB(); err != nil {
		t.Fatal(err)
	}
	path, err := FixtureDBPath()
	if err != nil {
		t.Fatal(err)
	}
	conn, err := Open(path, false)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	ctx := context.Background()
	_, err = conn.UpdateTableRow(ctx, model.UpdateTableRowRequest{
		Table: "customers",
		RowID: "1",
		Updates: []model.ColumnUpdate{
			{Column: "name", Text: "Ada Updated", IsNull: false},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	var name string
	if err := conn.sql.QueryRowContext(ctx, `SELECT name FROM customers WHERE rowid = 1`).Scan(&name); err != nil {
		t.Fatal(err)
	}
	if name != "Ada Updated" {
		t.Fatalf("name = %q, want Ada Updated", name)
	}

	// restore
	_, _ = conn.UpdateTableRow(ctx, model.UpdateTableRowRequest{
		Table:   "customers",
		RowID:   "1",
		Updates: []model.ColumnUpdate{{Column: "name", Text: "Example Customer 001", IsNull: false}},
	})
}

func TestUpdateTableRow_RejectsView(t *testing.T) {
	if err := EnsureFixtureDB(); err != nil {
		t.Fatal(err)
	}
	path, _ := FixtureDBPath()
	conn, err := Open(path, false)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	_, err = conn.UpdateTableRow(context.Background(), model.UpdateTableRowRequest{
		Table:   "orders_by_customer",
		RowID:   "1",
		Updates: []model.ColumnUpdate{{Column: "name", Text: "x", IsNull: false}},
	})
	appErr, ok := apperrors.As(err)
	if !ok || appErr.Code != apperrors.CodeNotEditable {
		t.Fatalf("expected NOT_EDITABLE, got %v", err)
	}
}

func TestUpdateTableRow_ReadOnlyConnection(t *testing.T) {
	if err := EnsureFixtureDB(); err != nil {
		t.Fatal(err)
	}
	path, _ := FixtureDBPath()
	conn, err := Open(path, true)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	_, err = conn.UpdateTableRow(context.Background(), model.UpdateTableRowRequest{
		Table:   "customers",
		RowID:   "1",
		Updates: []model.ColumnUpdate{{Column: "name", Text: "x", IsNull: false}},
	})
	appErr, ok := apperrors.As(err)
	if !ok || appErr.Code != apperrors.CodeReadOnlyViolation {
		t.Fatalf("expected READ_ONLY_VIOLATION, got %v", err)
	}
}

func TestUpdateTableRow_RejectsNullOnNotNullColumn(t *testing.T) {
	if err := EnsureFixtureDB(); err != nil {
		t.Fatal(err)
	}
	path, _ := FixtureDBPath()
	conn, err := Open(path, false)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	_, err = conn.UpdateTableRow(context.Background(), model.UpdateTableRowRequest{
		Table:   "customers",
		RowID:   "1",
		Updates: []model.ColumnUpdate{{Column: "name", Text: "", IsNull: true}},
	})
	appErr, ok := apperrors.As(err)
	if !ok || appErr.Code != apperrors.CodeMalformedSQL {
		t.Fatalf("expected MALFORMED_SQL, got %v", err)
	}
}

func TestGetTableRows_IncludesRowIDsWhenEditable(t *testing.T) {
	if err := EnsureFixtureDB(); err != nil {
		t.Fatal(err)
	}
	path, _ := FixtureDBPath()
	conn, err := Open(path, false)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	resp, err := conn.GetTableRows(context.Background(), model.TableRowsRequest{
		Table:     "customers",
		Page:      1,
		PageSize:  100,
		WithTotal: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Editable {
		t.Fatal("expected editable response")
	}
	if len(resp.RowIDs) != len(resp.Rows) || len(resp.RowIDs) == 0 {
		t.Fatalf("rowIds len %d, rows len %d", len(resp.RowIDs), len(resp.Rows))
	}
	if resp.RowIDs[0] != "1" {
		t.Fatalf("expected rowid 1, got %q", resp.RowIDs[0])
	}
}

func TestGetTableRows_WithoutRowIDIsNotEditable(t *testing.T) {
	conn := openUpdateTestDB(t, `
		CREATE TABLE settings (key TEXT PRIMARY KEY, value TEXT) WITHOUT ROWID;
		INSERT INTO settings VALUES ('theme', 'dark');
	`)
	resp, err := conn.GetTableRows(context.Background(), model.TableRowsRequest{
		Table: "settings", Page: 1, PageSize: 100,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Editable || len(resp.RowIDs) != 0 {
		t.Fatalf("WITHOUT ROWID table reported editable: %+v", resp)
	}
}

func TestUpdateTableRow_UsesUnshadowedRowIDAlias(t *testing.T) {
	conn := openUpdateTestDB(t, `
		CREATE TABLE shadowed (rowid TEXT, value TEXT);
		INSERT INTO shadowed (rowid, value) VALUES ('business-id', 'before');
	`)
	resp, err := conn.GetTableRows(context.Background(), model.TableRowsRequest{
		Table: "shadowed", Page: 1, PageSize: 100,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Editable || len(resp.RowIDs) != 1 || resp.RowIDs[0] != "1" {
		t.Fatalf("unexpected row identity: %+v", resp.RowIDs)
	}

	if _, err := conn.UpdateTableRow(context.Background(), model.UpdateTableRowRequest{
		Table: "shadowed", RowID: resp.RowIDs[0],
		Updates: []model.ColumnUpdate{{Column: "value", Text: "after"}},
	}); err != nil {
		t.Fatal(err)
	}
	var value string
	if err := conn.sql.QueryRow(`SELECT value FROM shadowed WHERE _rowid_ = 1`).Scan(&value); err != nil {
		t.Fatal(err)
	}
	if value != "after" {
		t.Fatalf("value = %q, want after", value)
	}
}

func TestUpdateTableRow_ReturnsChangedIntegerPrimaryKey(t *testing.T) {
	conn := openUpdateTestDB(t, `
		CREATE TABLE items (id INTEGER PRIMARY KEY, value TEXT);
		INSERT INTO items VALUES (1, 'before');
	`)
	resp, err := conn.UpdateTableRow(context.Background(), model.UpdateTableRowRequest{
		Table: "items", RowID: "1",
		Updates: []model.ColumnUpdate{{Column: "id", Text: "9"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Cells) != 2 || resp.Cells[0].Value != int64(9) {
		t.Fatalf("updated row = %+v", resp.Cells)
	}
}

func TestRowIDRoundTripsBeyondJavaScriptSafeInteger(t *testing.T) {
	conn := openUpdateTestDB(t, `
		CREATE TABLE items (value TEXT);
		INSERT INTO items (rowid, value) VALUES (9223372036854775807, 'before');
	`)
	page, err := conn.GetTableRows(context.Background(), model.TableRowsRequest{
		Table: "items", Page: 1, PageSize: 100,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.RowIDs) != 1 || page.RowIDs[0] != "9223372036854775807" {
		t.Fatalf("row id lost precision: %+v", page.RowIDs)
	}
	if _, err := conn.UpdateTableRow(context.Background(), model.UpdateTableRowRequest{
		Table: "items", RowID: page.RowIDs[0],
		Updates: []model.ColumnUpdate{{Column: "value", Text: "after"}},
	}); err != nil {
		t.Fatal(err)
	}
}

func TestUpdateTableRow_RejectsDuplicateColumns(t *testing.T) {
	conn := openUpdateTestDB(t, `
		CREATE TABLE items (value TEXT);
		INSERT INTO items VALUES ('before');
	`)
	_, err := conn.UpdateTableRow(context.Background(), model.UpdateTableRowRequest{
		Table: "items", RowID: "1",
		Updates: []model.ColumnUpdate{
			{Column: "value", Text: "first"},
			{Column: "value", Text: "second"},
		},
	})
	appErr, ok := apperrors.As(err)
	if !ok || appErr.Code != apperrors.CodeInvalidColumn {
		t.Fatalf("expected INVALID_COLUMN, got %v", err)
	}
}
