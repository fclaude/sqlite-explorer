package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite" // register pure-Go SQLite driver

	"sqlite-explorer/backend/apperrors"
)

// Driver is the database/sql driver name registered by modernc.org/sqlite.
const Driver = "sqlite"

type countCacheKey struct {
	table  string
	filter string
}

// maxCachedCounts bounds the row count cache; filter strings typed by the user are keys.
const maxCachedCounts = 256

// DB wraps a SQLite connection to a single database file.
type DB struct {
	sql      *sql.DB
	querySQL *sql.DB
	// versionConn is a dedicated read-only connection for PRAGMA data_version, which
	// changes whenever any other connection or process commits to the file.
	versionConn *sql.Conn
	path        string
	readOnly    bool
	openedAt    time.Time

	countMu       sync.Mutex
	countVersion  int64
	rowCountCache map[countCacheKey]int64
}

// Open opens a SQLite database at path. Ad-hoc SQL always runs through a separate
// mode=ro connection so the read-only query boundary is enforced by SQLite.
func Open(path string, readOnly bool) (*DB, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve path: %w", err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, apperrors.New(
				apperrors.CodeNotSQLite,
				"The database file could not be found.",
				absPath,
			)
		}
		if os.IsPermission(err) {
			return nil, apperrors.New(apperrors.CodePermission, "Permission denied opening the database file.", absPath)
		}
		return nil, fmt.Errorf("stat database: %w", err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("path is a directory, not a database file: %s", absPath)
	}

	dsn := buildDSN(absPath, readOnly)
	sqlDB, err := sql.Open(Driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	if err := verifySQLite(sqlDB); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}

	queryDB := sqlDB
	if !readOnly {
		queryDB, err = sql.Open(Driver, buildDSN(absPath, true))
		if err != nil {
			_ = sqlDB.Close()
			return nil, fmt.Errorf("open read-only query connection: %w", err)
		}
		if err := verifySQLite(queryDB); err != nil {
			_ = queryDB.Close()
			_ = sqlDB.Close()
			return nil, err
		}
	}

	versionConn, err := queryDB.Conn(context.Background())
	if err != nil {
		if queryDB != sqlDB {
			_ = queryDB.Close()
		}
		_ = sqlDB.Close()
		return nil, fmt.Errorf("open data version connection: %w", err)
	}

	return &DB{
		sql:           sqlDB,
		querySQL:      queryDB,
		versionConn:   versionConn,
		path:          absPath,
		readOnly:      readOnly,
		openedAt:      time.Now(),
		rowCountCache: make(map[countCacheKey]int64),
	}, nil
}

// buildDSN constructs a modernc.org/sqlite DSN. The file path is URL-escaped. Read-only
// connections also set query_only: mode=ro alone still allows VACUUM INTO and temporary objects.
func buildDSN(absPath string, readOnly bool) string {
	// SQLite file URIs use forward slashes even on Windows.
	filePath := filepath.ToSlash(absPath)
	if !strings.HasPrefix(filePath, "/") {
		// Windows drive paths: C:/foo -> /C:/foo for URI form
		if len(filePath) >= 2 && filePath[1] == ':' {
			filePath = "/" + filePath
		}
	}
	u := url.URL{
		Scheme: "file",
		Path:   filePath,
	}
	q := u.Query()
	if readOnly {
		q.Set("mode", "ro")
		q.Set("_pragma", "query_only(1)")
	}
	u.RawQuery = q.Encode()
	return u.String()
}

func verifySQLite(sqlDB *sql.DB) error {
	var version int
	err := sqlDB.QueryRow("PRAGMA schema_version").Scan(&version)
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "file is not a database") ||
			strings.Contains(msg, "file is encrypted") ||
			strings.Contains(msg, "not a database") {
			return apperrors.New(
				apperrors.CodeNotSQLite,
				"The selected file is not a valid SQLite database.",
				msg,
			)
		}
		return fmt.Errorf("verify sqlite database: %w", err)
	}
	return nil
}

// Close closes the underlying database handle.
func (d *DB) Close() error {
	if d == nil || d.sql == nil {
		return nil
	}
	var versionErr, queryErr error
	if d.versionConn != nil {
		versionErr = d.versionConn.Close()
	}
	if d.querySQL != nil && d.querySQL != d.sql {
		queryErr = d.querySQL.Close()
	}
	return errors.Join(versionErr, d.sql.Close(), queryErr)
}

// Path returns the absolute filesystem path of the open database.
func (d *DB) Path() string {
	return d.path
}

// ReadOnly reports whether the connection was opened read-only.
func (d *DB) ReadOnly() bool {
	return d.readOnly
}

// OpenedAt returns when the connection was established.
func (d *DB) OpenedAt() time.Time {
	return d.openedAt
}
