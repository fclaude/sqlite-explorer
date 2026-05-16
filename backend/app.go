package backend

import (
	"context"
	"errors"
	"log"
	"os"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"sqlite-explorer/backend/apperrors"
	"sqlite-explorer/backend/db"
	"sqlite-explorer/backend/model"
)

// App is the Wails-bound application backend.
type App struct {
	ctx context.Context
	db  *db.DB
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

// OpenDatabase shows a native file picker and opens the selected SQLite file read-only.
func (a *App) OpenDatabase() (model.DatabaseInfo, error) {
	if a.ctx == nil {
		return model.DatabaseInfo{}, errors.New("application not started")
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

// OpenDatabasePath opens a database at the given path (used by tests and tooling).
func (a *App) OpenDatabasePath(path string) (model.DatabaseInfo, error) {
	return a.openPath(path)
}

func (a *App) openPath(path string) (model.DatabaseInfo, error) {
	if a.db != nil {
		_ = a.db.Close()
		a.db = nil
	}

	conn, err := db.Open(path, true)
	if err != nil {
		return model.DatabaseInfo{}, err
	}
	a.db = conn
	return a.databaseInfo(conn)
}

// CloseDatabase closes the current database connection.
func (a *App) CloseDatabase() error {
	if a.db == nil {
		return apperrors.New(apperrors.CodeNoDBOpen, "No database is open.", "")
	}
	err := a.db.Close()
	a.db = nil
	return err
}

// DatabaseInfo returns metadata for the currently open database.
func (a *App) DatabaseInfo() (model.DatabaseInfo, error) {
	if a.db == nil {
		return model.DatabaseInfo{}, apperrors.New(apperrors.CodeNoDBOpen, "No database is open.", "")
	}
	return a.databaseInfo(a.db)
}

// GetSchema returns tables, views, indexes, and triggers for the open database.
func (a *App) GetSchema() (model.SchemaInfo, error) {
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

// RunQuery executes read-only SQL and returns results.
func (a *App) RunQuery(req model.QueryRequest) (model.QueryResponse, error) {
	if a.db == nil {
		return model.QueryResponse{}, apperrors.New(apperrors.CodeNoDBOpen, "No database is open.", "")
	}
	return a.db.RunQuery(context.Background(), req.SQL)
}

// GetTableRows returns a paginated page of rows for a table or view.
func (a *App) GetTableRows(req model.TableRowsRequest) (model.TableRowsResponse, error) {
	if a.db == nil {
		return model.TableRowsResponse{}, apperrors.New(apperrors.CodeNoDBOpen, "No database is open.", "")
	}
	return a.db.GetTableRows(context.Background(), req)
}

// GetTableRowCount returns an exact row count for a table or view (lazy, on demand).
func (a *App) GetTableRowCount(table string) (int64, error) {
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
