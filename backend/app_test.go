package backend

import (
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
