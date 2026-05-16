package db

import (
	"fmt"
	"strings"
)

// QuoteIdentifier returns a double-quoted SQLite identifier with internal quotes escaped.
func QuoteIdentifier(name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("identifier name is empty")
	}
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`, nil
}
