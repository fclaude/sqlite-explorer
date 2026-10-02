package db

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func findModuleRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("go.mod not found")
		}
		dir = parent
	}
}

// FixtureDBPath returns the path to testdata/sample.sqlite.
func FixtureDBPath() (string, error) {
	root, err := findModuleRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "testdata", "sample.sqlite"), nil
}

// EnsureFixtureDB creates testdata/sample.sqlite from testdata/fixtures.sql if missing.
func EnsureFixtureDB() error {
	path, err := FixtureDBPath()
	if err != nil {
		return err
	}
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	root, err := findModuleRoot()
	if err != nil {
		return err
	}
	sqlBytes, err := os.ReadFile(filepath.Join(root, "testdata", "fixtures.sql"))
	if err != nil {
		return err
	}

	sqlDB, err := sql.Open(Driver, path)
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	for _, stmt := range splitSQL(string(sqlBytes)) {
		if stmt == "" {
			continue
		}
		if _, err := sqlDB.Exec(stmt); err != nil {
			return fmt.Errorf("exec fixture SQL: %w\nstatement: %s", err, stmt)
		}
	}
	return nil
}

func splitSQL(script string) []string {
	var stmts []string
	var b strings.Builder
	depth := 0
	for _, line := range strings.Split(script, "\n") {
		trim := strings.TrimSpace(line)
		if trim == "" || strings.HasPrefix(trim, "--") {
			continue
		}
		upper := strings.ToUpper(trim)
		if upper == "BEGIN" || strings.HasPrefix(upper, "BEGIN ") {
			depth++
		}
		b.WriteString(line)
		b.WriteByte('\n')
		if strings.HasSuffix(trim, ";") {
			if strings.HasSuffix(upper, "END;") {
				depth--
			}
			if depth == 0 {
				stmts = append(stmts, strings.TrimSpace(b.String()))
				b.Reset()
			}
		}
	}
	if s := strings.TrimSpace(b.String()); s != "" {
		stmts = append(stmts, s)
	}
	return stmts
}
