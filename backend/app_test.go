package backend

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"sqlite-explorer/backend/apperrors"
	"sqlite-explorer/backend/db"
	"sqlite-explorer/backend/model"
)

func TestCancelQuery_Interrupts(t *testing.T) {
	if err := db.EnsureFixtureDB(); err != nil {
		t.Fatal(err)
	}
	path, err := db.FixtureDBPath()
	if err != nil {
		t.Fatal(err)
	}

	app := NewApp()
	if _, err := app.openPath(path); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = app.CloseDatabase() }()

	longSQL := `WITH RECURSIVE c(x) AS (SELECT 1 UNION ALL SELECT x+1 FROM c) SELECT count(*) FROM c`

	var wg sync.WaitGroup
	wg.Add(1)
	var runErr error
	go func() {
		defer wg.Done()
		_, runErr = app.RunQuery(model.QueryRequest{SQL: longSQL})
	}()

	deadline := time.Now().Add(2 * time.Second)
	var id int64
	for time.Now().Before(deadline) {
		id = app.activeQueryID()
		if id != 0 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if id == 0 {
		t.Fatal("query never registered an active id")
	}

	start := time.Now()
	app.CancelQuery(id)
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("RunQuery did not return soon after cancel")
	}
	elapsed := time.Since(start)
	if elapsed > 50*time.Millisecond {
		t.Fatalf("cancel took %v, want return within 50ms", elapsed)
	}

	appErr, ok := apperrors.As(runErr)
	if !ok {
		t.Fatalf("expected apperrors.Error, got %v", runErr)
	}
	if appErr.Code != apperrors.CodeCancelled {
		t.Fatalf("expected CANCELLED, got %s", appErr.Code)
	}
}

func TestFormatError_SendsStructuredJSON(t *testing.T) {
	got, ok := FormatError(apperrors.New(apperrors.CodeMalformedSQL, "The SQL could not be parsed.", `near "FORM": syntax error`)).(string)
	if !ok {
		t.Fatal("FormatError must return a string: the Wails runtime wraps other values in new Error(value)")
	}
	var decoded apperrors.Error
	if err := json.Unmarshal([]byte(got), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Code != apperrors.CodeMalformedSQL || decoded.Detail != `near "FORM": syntax error` {
		t.Fatalf("decoded: %+v", decoded)
	}

	got = FormatError(errors.New("disk I/O error")).(string)
	if err := json.Unmarshal([]byte(got), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Code != apperrors.CodeInternal || decoded.Detail != "disk I/O error" || decoded.Message == "" {
		t.Fatalf("decoded generic error: %+v", decoded)
	}
}

func TestOpenPath_CancelsInFlightWorkInsteadOfWaiting(t *testing.T) {
	if err := db.EnsureFixtureDB(); err != nil {
		t.Fatal(err)
	}
	path, err := db.FixtureDBPath()
	if err != nil {
		t.Fatal(err)
	}
	app := NewApp()
	if _, err := app.openPath(path); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = app.CloseDatabase() }()

	runErr := make(chan error, 1)
	go func() {
		_, err := app.RunQuery(model.QueryRequest{
			SQL: `WITH RECURSIVE c(x) AS (SELECT 1 UNION ALL SELECT x+1 FROM c) SELECT count(*) FROM c`,
		})
		runErr <- err
	}()
	deadline := time.Now().Add(2 * time.Second)
	for app.activeQueryID() == 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}

	start := time.Now()
	if _, err := app.openPath(path); err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("opening a database waited %v for the running query", elapsed)
	}
	select {
	case err := <-runErr:
		if appErr, ok := apperrors.As(err); !ok || appErr.Code != apperrors.CodeCancelled {
			t.Fatalf("expected cancelled query, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("query kept running after the database was replaced")
	}

	if _, err := app.GetSchema(); err != nil {
		t.Fatalf("new database unusable: %v", err)
	}
}

func TestRunQuery_RejectsUnknownPermission(t *testing.T) {
	app := NewApp()
	_, err := app.RunQuery(model.QueryRequest{SQL: "SELECT 1", Allow: []string{"everything"}})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestExport_CancelQueryStopsExport(t *testing.T) {
	if err := db.EnsureFixtureDB(); err != nil {
		t.Fatal(err)
	}
	path, _ := db.FixtureDBPath()
	app := NewApp()
	if _, err := app.openPath(path); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = app.CloseDatabase() }()

	out := filepath.Join(t.TempDir(), "endless.csv")
	done := make(chan error, 1)
	go func() {
		_, err := app.exportToPath(out, model.ExportRequest{
			Source: db.ExportSourceQueryResult,
			SQL:    `WITH RECURSIVE c(x) AS (SELECT 1 UNION ALL SELECT x+1 FROM c) SELECT x FROM c`,
		})
		done <- err
	}()
	deadline := time.Now().Add(2 * time.Second)
	for app.activeQueryID() == 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	time.Sleep(50 * time.Millisecond)
	app.CancelQuery(0)

	select {
	case err := <-done:
		if appErr, ok := apperrors.As(err); !ok || appErr.Code != apperrors.CodeCancelled {
			t.Fatalf("expected cancelled export, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("export did not stop after CancelQuery")
	}
	entries, _ := os.ReadDir(filepath.Dir(out))
	if len(entries) != 0 {
		t.Fatalf("cancelled export left files: %v", entries)
	}
}
