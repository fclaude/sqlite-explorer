package backend

import (
	"context"
	"errors"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"sqlite-explorer/backend/apperrors"
	"sqlite-explorer/backend/db"
	"sqlite-explorer/backend/model"
)

// App is the Wails-bound application backend.
type App struct {
	ctx context.Context

	dbMu sync.RWMutex
	db   *db.DB

	queryMu     sync.Mutex
	queryID     int64
	queryCancel context.CancelFunc
	nextQueryID int64
}

// NewApp creates a new App application struct.
func NewApp() *App {
	return &App{}
}

// Startup is called when the app starts. The context is saved
// so we can call the runtime methods.
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
}

// OpenDatabase shows a native file picker and opens the selected SQLite file.
func (a *App) OpenDatabase() (model.DatabaseInfo, error) {
	if a.ctx == nil {
		return model.DatabaseInfo{}, apperrors.New(apperrors.CodeMalformedSQL, "Application not started.", "")
	}

	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Open SQLite Database",
		Filters: []runtime.FileFilter{
			{DisplayName: "SQLite databases (*.sqlite, *.sqlite3, *.db)", Pattern: "*.sqlite;*.sqlite3;*.db"},
			{DisplayName: "All files", Pattern: "*"},
		},
	})
	if err != nil {
		return model.DatabaseInfo{}, err
	}
	if path == "" {
		return model.DatabaseInfo{}, apperrors.New(apperrors.CodeCancelled, "Open database cancelled.", "")
	}

	return a.openPath(path)
}

func (a *App) openPath(path string) (model.DatabaseInfo, error) {
	conn, err := db.Open(path, false)
	if err != nil {
		if appErr, ok := apperrors.As(err); ok {
			return model.DatabaseInfo{}, appErr
		}
		return model.DatabaseInfo{}, apperrors.New(
			apperrors.CodeNotSQLite,
			apperrors.UserMessage(err),
			err.Error(),
		)
	}

	a.dbMu.Lock()
	old := a.db
	a.db = conn
	a.dbMu.Unlock()
	if old != nil {
		_ = old.Close()
	}
	return a.databaseInfo(conn)
}

// CloseDatabase closes the current database connection.
func (a *App) CloseDatabase() error {
	a.CancelQuery(0)
	a.dbMu.Lock()
	defer a.dbMu.Unlock()
	if a.db == nil {
		return apperrors.New(apperrors.CodeNoDBOpen, "No database is open.", "")
	}
	err := a.db.Close()
	a.db = nil
	return err
}

// DatabaseInfo returns metadata for the currently open database.
func (a *App) DatabaseInfo() (model.DatabaseInfo, error) {
	a.dbMu.RLock()
	defer a.dbMu.RUnlock()
	if a.db == nil {
		return model.DatabaseInfo{}, apperrors.New(apperrors.CodeNoDBOpen, "No database is open.", "")
	}
	return a.databaseInfo(a.db)
}

// GetSchema returns tables, views, indexes, and triggers for the open database.
func (a *App) GetSchema() (model.SchemaInfo, error) {
	a.dbMu.RLock()
	defer a.dbMu.RUnlock()
	if a.db == nil {
		return model.SchemaInfo{}, apperrors.New(apperrors.CodeNoDBOpen, "No database is open.", "")
	}
	start := time.Now()
	schema, err := a.db.GetSchema(context.Background())
	if err != nil {
		return model.SchemaInfo{}, err
	}
	log.Printf("GetSchema completed in %dms", time.Since(start).Milliseconds())
	return schema, nil
}

// ExportRowsToCSV exports table page or query results to a CSV file via save dialog.
func (a *App) ExportRowsToCSV(req model.ExportRequest) error {
	a.dbMu.RLock()
	defer a.dbMu.RUnlock()
	if a.db == nil {
		return apperrors.New(apperrors.CodeNoDBOpen, "No database is open.", "")
	}
	if a.ctx == nil {
		return apperrors.New(apperrors.CodeMalformedSQL, "Application not started.", "")
	}

	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Export to CSV",
		DefaultFilename: exportDefaultFilename(req),
		Filters: []runtime.FileFilter{
			{DisplayName: "CSV (*.csv)", Pattern: "*.csv"},
			{DisplayName: "All files", Pattern: "*"},
		},
	})
	if err != nil {
		return err
	}
	if path == "" {
		return nil
	}

	return a.db.ExportRowsToCSV(context.Background(), path, req)
}

func exportDefaultFilename(req model.ExportRequest) string {
	switch req.Source {
	case db.ExportSourceTablePage:
		if req.TableRows.Table != "" {
			return req.TableRows.Table + ".csv"
		}
	case db.ExportSourceQueryResult:
		return "query-results.csv"
	}
	return "export.csv"
}

// RunQuery executes read-only SQL and returns results.
func (a *App) RunQuery(req model.QueryRequest) (model.QueryResponse, error) {
	a.dbMu.RLock()
	defer a.dbMu.RUnlock()
	if a.db == nil {
		return model.QueryResponse{}, apperrors.New(apperrors.CodeNoDBOpen, "No database is open.", "")
	}

	ctx, cancel := context.WithCancel(context.Background())
	id := atomic.AddInt64(&a.nextQueryID, 1)

	a.queryMu.Lock()
	if a.queryCancel != nil {
		a.queryCancel()
	}
	a.queryID = id
	a.queryCancel = cancel
	a.queryMu.Unlock()

	defer func() {
		a.queryMu.Lock()
		if a.queryID == id {
			a.queryCancel = nil
		}
		a.queryMu.Unlock()
		cancel()
	}()

	resp, err := a.db.RunQuery(ctx, req.SQL)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return model.QueryResponse{}, apperrors.New(apperrors.CodeCancelled, "Query cancelled.", "")
		}
		if appErr, ok := apperrors.As(err); ok {
			return model.QueryResponse{}, appErr
		}
		return model.QueryResponse{}, apperrors.New(
			apperrors.CodeMalformedSQL,
			apperrors.UserMessage(err),
			err.Error(),
		)
	}
	resp.QueryID = id
	return resp, nil
}

// CancelQuery cancels an in-flight query. Pass 0 to cancel the active query.
func (a *App) CancelQuery(id int64) {
	a.queryMu.Lock()
	defer a.queryMu.Unlock()
	if id != 0 && a.queryID != id {
		return
	}
	if a.queryCancel != nil {
		a.queryCancel()
	}
}

func (a *App) activeQueryID() int64 {
	a.queryMu.Lock()
	defer a.queryMu.Unlock()
	return a.queryID
}

// GetTableRows returns a paginated page of rows for a table or view.
func (a *App) GetTableRows(req model.TableRowsRequest) (model.TableRowsResponse, error) {
	a.dbMu.RLock()
	defer a.dbMu.RUnlock()
	if a.db == nil {
		return model.TableRowsResponse{}, apperrors.New(apperrors.CodeNoDBOpen, "No database is open.", "")
	}
	return a.db.GetTableRows(context.Background(), req)
}

// UpdateTableRow saves edits to a single table row.
func (a *App) UpdateTableRow(req model.UpdateTableRowRequest) (model.UpdateTableRowResponse, error) {
	a.dbMu.RLock()
	defer a.dbMu.RUnlock()
	if a.db == nil {
		return model.UpdateTableRowResponse{}, apperrors.New(apperrors.CodeNoDBOpen, "No database is open.", "")
	}
	resp, err := a.db.UpdateTableRow(context.Background(), req)
	if err != nil {
		if appErr, ok := apperrors.As(err); ok {
			return model.UpdateTableRowResponse{}, appErr
		}
		return model.UpdateTableRowResponse{}, apperrors.New(
			apperrors.CodeMalformedSQL,
			apperrors.UserMessage(err),
			err.Error(),
		)
	}
	return resp, nil
}

// GetObjectStats returns statistics for a table or view.
func (a *App) GetObjectStats(name string) (model.ObjectStats, error) {
	a.dbMu.RLock()
	defer a.dbMu.RUnlock()
	if a.db == nil {
		return model.ObjectStats{}, apperrors.New(apperrors.CodeNoDBOpen, "No database is open.", "")
	}
	return a.db.GetObjectStats(context.Background(), name)
}

// GetTableRowCount returns an exact row count for a table or view (lazy, on demand).
func (a *App) GetTableRowCount(table string) (int64, error) {
	a.dbMu.RLock()
	defer a.dbMu.RUnlock()
	if a.db == nil {
		return 0, apperrors.New(apperrors.CodeNoDBOpen, "No database is open.", "")
	}
	return a.db.GetTableRowCount(context.Background(), table)
}

func (a *App) databaseInfo(conn *db.DB) (model.DatabaseInfo, error) {
	info, err := os.Stat(conn.Path())
	if err != nil {
		return model.DatabaseInfo{}, err
	}
	return model.DatabaseInfo{
		Path:      conn.Path(),
		SizeBytes: info.Size(),
		ReadOnly:  conn.ReadOnly(),
	}, nil
}
