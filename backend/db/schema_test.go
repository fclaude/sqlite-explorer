package db

import (
	"context"
	"slices"
	"testing"

	"sqlite-explorer/backend/model"
)

func TestGetSchema_Fixture(t *testing.T) {
	if err := EnsureFixtureDB(); err != nil {
		t.Fatal(err)
	}
	path, err := FixtureDBPath()
	if err != nil {
		t.Fatal(err)
	}

	conn, err := Open(path, true)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer conn.Close()

	schema, err := conn.GetSchema(context.Background())
	if err != nil {
		t.Fatalf("GetSchema: %v", err)
	}

	assertNames(t, "tables", namesFrom(schema.Tables, func(t model.TableInfo) string { return t.Name }),
		"customers", "notes", "orders")
	assertNames(t, "views", namesFrom(schema.Views, func(v model.ViewInfo) string { return v.Name }),
		"orders_by_customer")
	assertNames(t, "triggers", namesFrom(schema.Triggers, func(tr model.TriggerInfo) string { return tr.Name }),
		"notes_updated_at")
	assertNames(t, "indexes", namesFrom(schema.Indexes, func(i model.IndexInfo) string { return i.Name }),
		"idx_orders_customer")

	for _, tbl := range schema.Tables {
		if tbl.Name == "orders_by_customer" {
			t.Fatal("orders_by_customer should not appear as a table")
		}
	}

	customers := findTable(schema.Tables, "customers")
	if customers == nil {
		t.Fatal("customers table not found")
	}
	idCol := findColumn(customers.Columns, "id")
	if idCol == nil || idCol.PrimaryKey != 1 {
		t.Fatalf("customers.id primary key position: got %+v", idCol)
	}
	emailCol := findColumn(customers.Columns, "email")
	if emailCol == nil || !emailCol.NotNull || emailCol.DefaultValue != nil {
		t.Fatalf("customers.email: want NOT NULL, no default; got %+v", emailCol)
	}

	orders := findTable(schema.Tables, "orders")
	if orders == nil {
		t.Fatal("orders table not found")
	}
	var fk *model.ForeignKeyInfo
	for i := range orders.ForeignKeys {
		if orders.ForeignKeys[i].From == "customer_id" {
			fk = &orders.ForeignKeys[i]
			break
		}
	}
	if fk == nil {
		t.Fatal("orders.customer_id foreign key not found")
	}
	if fk.Table != "customers" || fk.To != "id" || fk.OnDelete != "CASCADE" {
		t.Fatalf("orders FK: got %+v", fk)
	}

	notes := findTable(schema.Tables, "notes")
	if notes == nil {
		t.Fatal("notes table not found")
	}
	bodyCol := findColumn(notes.Columns, "body")
	if bodyCol == nil || bodyCol.NotNull {
		t.Fatalf("notes.body: want nullable; got %+v", bodyCol)
	}
	if bodyCol.DefaultValue != nil && *bodyCol.DefaultValue != "NULL" {
		t.Fatalf("notes.body default: got %q", *bodyCol.DefaultValue)
	}

	idx := findIndex(schema.Indexes, "idx_orders_customer")
	if idx == nil {
		t.Fatal("idx_orders_customer not found")
	}
	if idx.Unique {
		t.Fatal("idx_orders_customer should not be unique")
	}
	if idx.Table != "orders" {
		t.Fatalf("idx_orders_customer table: got %q", idx.Table)
	}
}

func TestGetSchema_NoRowCounts(t *testing.T) {
	if err := EnsureFixtureDB(); err != nil {
		t.Fatal(err)
	}
	path, _ := FixtureDBPath()
	conn, err := Open(path, true)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	schema, err := conn.GetSchema(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(schema.Tables) == 0 {
		t.Fatal("expected tables")
	}
	count, err := conn.GetTableRowCount(context.Background(), "customers")
	if err != nil {
		t.Fatal(err)
	}
	if count != 5 {
		t.Fatalf("customers row count: got %d, want 5", count)
	}
}

func assertNames(t *testing.T, kind string, got []string, want ...string) {
	t.Helper()
	wantSorted := append([]string(nil), want...)
	slices.Sort(wantSorted)
	gotSorted := append([]string(nil), got...)
	slices.Sort(gotSorted)
	if len(gotSorted) != len(wantSorted) {
		t.Fatalf("%s names: got %v, want %v", kind, gotSorted, wantSorted)
	}
	for i := range wantSorted {
		if gotSorted[i] != wantSorted[i] {
			t.Fatalf("%s names: got %v, want %v", kind, gotSorted, wantSorted)
		}
	}
}

func namesFrom[T any](items []T, name func(T) string) []string {
	out := make([]string, len(items))
	for i, item := range items {
		out[i] = name(item)
	}
	return out
}

func findTable(tables []model.TableInfo, name string) *model.TableInfo {
	for i := range tables {
		if tables[i].Name == name {
			return &tables[i]
		}
	}
	return nil
}

func findColumn(cols []model.ColumnInfo, name string) *model.ColumnInfo {
	for i := range cols {
		if cols[i].Name == name {
			return &cols[i]
		}
	}
	return nil
}

func findIndex(indexes []model.IndexInfo, name string) *model.IndexInfo {
	for i := range indexes {
		if indexes[i].Name == name {
			return &indexes[i]
		}
	}
	return nil
}
