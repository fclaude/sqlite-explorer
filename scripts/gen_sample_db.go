//go:build ignore

// Generate testdata/sample.sqlite from testdata/fixtures.sql.
// Usage: go run scripts/gen_sample_db.go
package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

func main() {
	root, err := findModuleRoot()
	if err != nil {
		log.Fatal(err)
	}
	sqlPath := filepath.Join(root, "testdata", "fixtures.sql")
	outPath := filepath.Join(root, "testdata", "sample.sqlite")

	sqlBytes, err := os.ReadFile(sqlPath)
	if err != nil {
		log.Fatal(err)
	}

	_ = os.Remove(outPath)
	db, err := sql.Open("sqlite", outPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	for _, stmt := range splitSQL(string(sqlBytes)) {
		if stmt == "" {
			continue
		}
		if _, err := db.Exec(stmt); err != nil {
			log.Fatalf("exec: %v\n%s", err, stmt)
		}
	}
	fmt.Printf("Wrote %s\n", outPath)
}

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
			return "", fmt.Errorf("go.mod not found")
		}
		dir = parent
	}
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
