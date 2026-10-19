package backend

import (
	"context"
	"encoding/json"
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

	// dbMu guards db and dbCtx. Bound methods hold the read lock while they use the
	// database; dbCtx is cancelled before the database is replaced or closed so that
	// in-flight work stops and releases the lock promptly.
	dbMu     sync.RWMutex
	db       *db.DB
	dbCtx    context.Context
	dbCancel context.CancelFunc

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

// FormatError serializes errors returned by bound methods. The Wails runtime rejects with
// new Error(value), which only preserves strings, so the code, message, and detail travel
// as a JSON string that the frontend decodes.
func FormatError(err error) any {
	appErr, ok := apperrors.As(err)
	if !ok {
		appErr = apperrors.New(apperrors.CodeInternal, apperrors.UserMessage(err), err.Error())
	}
	b, marshalErr := json.Marshal(appErr)
	if marshalErr != nil {
		return appErr.Message
	}
	return string(b)
}

func errNoDatabase() error {
	return apperrors.New(apperrors.CodeNoDBOpen, "No database is open.", "")
}

// withDB runs fn with the open database while holding the read lock.
func withDB[T any](a *App, fn func(ctx context.Context, conn *db.DB) (T, error)) (T, error) {
	a.dbMu.RLock()
	defer a.dbMu.RUnlock()
	if a.db == nil {
		var zero T
		return zero, errNoDatabase()
	}
	return fn(a.dbCtx, a.db)
}

func (a *App) hasDatabase() bool {
	a.dbMu.RLock()
	defer a.dbMu.RUnlock()
	return a.db != nil
}

// swapDB replaces the open database with next (nil closes it).
func (a *App) swapDB(next *db.DB) error {
	// Cancel first: taking the write lock waits for every reader, and a waiting writer
	// blocks new readers, so uncancelled work would stall the whole UI.
	a.dbMu.RLock()
	cancel := a.dbCancel
	a.dbMu.RUnlock()
	if cancel != nil {
		cancel()
	}

	a.dbMu.Lock()
	old, oldCancel := a.db, a.dbCancel
	a.db = next
	a.dbCtx, a.dbCancel = nil, nil
	if next != nil {
		a.dbCtx, a.dbCancel = context.WithCancel(context.Background())
	}
	a.dbMu.Unlock()

	if oldCancel != nil {
		oldCancel()
	}
	if old != nil {
		return old.Close()
	}
	return nil
}

// OpenDatabase shows a native file picker and opens the selected SQLite file.
func (a *App) OpenDatabase() (model.DatabaseInfo, error) {
	if a.ctx == nil {
		return model.DatabaseInfo{}, apperrors.New(apperrors.CodeInternal, "Application not started.", "")
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

	if err := a.swapDB(conn); err != nil {
		log.Printf("close previous database: %v", err)
	}
	return databaseInfo(conn)
}

// CloseDatabase closes the current database connection.
func (a *App) CloseDatabase() error {
	if !a.hasDatabase() {
		return errNoDatabase()
	}
	return a.swapDB(nil)
}

// DatabaseInfo returns metadata for the currently open database.
func (a *App) DatabaseInfo() (model.DatabaseInfo, error) {
	return withDB(a, func(_ context.Context, conn *db.DB) (model.DatabaseInfo, error) {
		return databaseInfo(conn)
	})
}

// GetSchema returns tables, views, indexes, and triggers for the open database.
func (a *App) GetSchema() (model.SchemaInfo, error) {
	return withDB(a, func(ctx context.Context, conn *db.DB) (model.SchemaInfo, error) {
		start := time.Now()
		schema, err := conn.GetSchema(ctx)
		if err != nil {
			return model.SchemaInfo{}, err
		}
		log.Printf("GetSchema completed in %dms", time.Since(start).Milliseconds())
		return schema, nil
	})
}

// GetStatementCategories describes the SQL statement categories the editor can allow.
func (a *App) GetStatementCategories() []model.StatementCategory {
	return db.StatementCategories()
}

// ExportRowsToCSV exports table rows or query results to a CSV file chosen in a save dialog.
// The export counts as the active query, so CancelQuery stops it.
func (a *App) ExportRowsToCSV(req model.ExportRequest) (model.ExportResult, error) {
	if a.ctx == nil {
		return model.ExportResult{}, apperrors.New(apperrors.CodeInternal, "Application not started.", "")
	}
	if !a.hasDatabase() {
		return model.ExportResult{}, errNoDatabase()
	}
	if err := db.ValidateExport(req); err != nil {
		return model.ExportResult{}, err
	}

	// The dialog is modal and can stay open indefinitely, so no lock is held while it is shown.
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Export to CSV",
		DefaultFilename: exportDefaultFilename(req),
		Filters: []runtime.FileFilter{
			{DisplayName: "CSV (*.csv)", Pattern: "*.csv"},
			{DisplayName: "All files", Pattern: "*"},
		},
	})
	if err != nil {
		return model.ExportResult{}, err
	}
	if path == "" {
		return model.ExportResult{}, nil
	}
	return a.exportToPath(path, req)
}

func (a *App) exportToPath(path string, req model.ExportRequest) (model.ExportResult, error) {
	return withDB(a, func(ctx context.Context, conn *db.DB) (model.ExportResult, error) {
		ctx, _, done := a.beginQuery(ctx)
		defer done()
		n, err := conn.ExportRowsToCSV(ctx, path, req)
		if err != nil {
			return model.ExportResult{}, err
		}
		return model.ExportResult{Path: path, RowCount: n}, nil
	})
}

func exportDefaultFilename(req model.ExportRequest) string {
	switch req.Source {
	case db.ExportSourceTablePage, db.ExportSourceTable:
		if req.TableRows.Table != "" {
			return req.TableRows.Table + ".csv"
		}
	case db.ExportSourceQueryResult:
		return "query-results.csv"
	}
	return "export.csv"
}

// beginQuery registers ctx-derived work as the active query, cancelling the previous one.
// Call done when the work finishes.
func (a *App) beginQuery(parent context.Context) (context.Context, int64, func()) {
	ctx, cancel := context.WithCancel(parent)
	id := atomic.AddInt64(&a.nextQueryID, 1)

	a.queryMu.Lock()
	if a.queryCancel != nil {
		a.queryCancel()
	}
	a.queryID = id
	a.queryCancel = cancel
	a.queryMu.Unlock()

	return ctx, id, func() {
		a.queryMu.Lock()
		if a.queryID == id {
			a.queryCancel = nil
		}
		a.queryMu.Unlock()
		cancel()
	}
}

// RunQuery executes SQL from the editor. Read statements always run; req.Allow lists the
// other statement categories the user enabled.
func (a *App) RunQuery(req model.QueryRequest) (model.QueryResponse, error) {
	policy, err := db.ParsePolicy(req.Allow)
	if err != nil {
		return model.QueryResponse{}, err
	}
	return withDB(a, func(ctx context.Context, conn *db.DB) (model.QueryResponse, error) {
		ctx, id, done := a.beginQuery(ctx)
		defer done()
		resp, err := conn.RunQuery(ctx, req.SQL, policy)
		if err != nil {
			return model.QueryResponse{}, err
		}
		resp.QueryID = id
		return resp, nil
	})
}

// CancelQuery cancels an in-flight query or export. Pass 0 to cancel whatever is active.
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
	return withDB(a, func(ctx context.Context, conn *db.DB) (model.TableRowsResponse, error) {
		return conn.GetTableRows(ctx, req)
	})
}

// UpdateTableRow saves edits to a single table row.
func (a *App) UpdateTableRow(req model.UpdateTableRowRequest) (model.UpdateTableRowResponse, error) {
	return withDB(a, func(ctx context.Context, conn *db.DB) (model.UpdateTableRowResponse, error) {
		return conn.UpdateTableRow(ctx, req)
	})
}

// GetObjectStats returns statistics for a table or view.
func (a *App) GetObjectStats(name string) (model.ObjectStats, error) {
	return withDB(a, func(ctx context.Context, conn *db.DB) (model.ObjectStats, error) {
		return conn.GetObjectStats(ctx, name)
	})
}

func databaseInfo(conn *db.DB) (model.DatabaseInfo, error) {
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
