package db

import (
	"database/sql"
	"errors"
	"sync"

	_ "modernc.org/sqlite" // register pure-Go SQLite driver
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"sqlite-explorer/backend/apperrors"
)

// Driver is the database/sql driver name registered by modernc.org/sqlite.
const Driver = "sqlite"

type countCacheKey struct {
	table  string
	filter string
}

// DB wraps a read-only SQLite connection to a single database file.
type DB struct {
	sql           *sql.DB
	path          string
	readOnly      bool
	openedAt      time.Time
	rowCountCache map[countCacheKey]int64
	countMu       sync.RWMutex
}

// Open opens a SQLite database at path. When readOnly is true, the connection uses
// mode=ro and immutable=1 so writes are rejected at the driver level.
func Open(path string, readOnly bool) (*DB, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve path: %w", err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("open database: %w", err)
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

	return &DB{
		sql:           sqlDB,
		path:          absPath,
		readOnly:      readOnly,
		openedAt:      time.Now(),
		rowCountCache: make(map[countCacheKey]int64),
	}, nil
}

// buildDSN constructs a modernc.org/sqlite DSN. The file path is URL-escaped.
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
		q.Set("immutable", "1")
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
	return d.sql.Close()
}

// Path returns the absolute filesystem path of the open database.
func (d *DB) Path() string {
	return d.path
}

// ReadOnly reports whether the connection was opened read-only.
func (d *DB) ReadOnly() bool {
	return d.readOnly
}

// SQL returns the underlying *sql.DB (for internal packages only).
func (d *DB) SQL() *sql.DB {
	return d.sql
}

// OpenedAt returns when the connection was established.
func (d *DB) OpenedAt() time.Time {
	return d.openedAt
}
