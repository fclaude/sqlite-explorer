package db

import (
	"context"
	"testing"
)

func TestGetObjectStats_CustomersTable(t *testing.T) {
	if err := EnsureFixtureDB(); err != nil {
		t.Fatal(err)
	}
	path, err := FixtureDBPath()
	if err != nil {
		t.Fatal(err)
	}
	conn, err := Open(path, true)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	stats, err := conn.GetObjectStats(context.Background(), "customers")
	if err != nil {
		t.Fatal(err)
	}
	if stats.Kind != "table" {
		t.Fatalf("kind = %q, want table", stats.Kind)
	}
	if stats.RowCount != 5 {
		t.Fatalf("rowCount = %d, want 5", stats.RowCount)
	}
	if stats.ColumnCount != 3 {
		t.Fatalf("columnCount = %d, want 3", stats.ColumnCount)
	}
	if stats.IndexCount < 0 {
		t.Fatal("expected non-negative index count")
	}
	if stats.DatabaseFileBytes <= 0 {
		t.Fatal("expected positive database file size")
	}
	if stats.DatabasePageSize <= 0 {
		t.Fatal("expected positive page size")
	}
}

func TestGetObjectStats_View(t *testing.T) {
	if err := EnsureFixtureDB(); err != nil {
		t.Fatal(err)
	}
	path, _ := FixtureDBPath()
	conn, err := Open(path, true)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	stats, err := conn.GetObjectStats(context.Background(), "orders_by_customer")
	if err != nil {
		t.Fatal(err)
	}
	if stats.Kind != "view" {
		t.Fatalf("kind = %q, want view", stats.Kind)
	}
	if stats.RowCount != 5 {
		t.Fatalf("rowCount = %d, want 5", stats.RowCount)
	}
}

func TestGetObjectStats_UnknownObject(t *testing.T) {
	if err := EnsureFixtureDB(); err != nil {
		t.Fatal(err)
	}
	path, _ := FixtureDBPath()
	conn, err := Open(path, true)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	_, err = conn.GetObjectStats(context.Background(), "not_a_table")
	if err == nil {
		t.Fatal("expected error for unknown object")
	}
}
