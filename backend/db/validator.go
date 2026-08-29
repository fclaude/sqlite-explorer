package db

import (
	"strings"
	"unicode"

	"sqlite-explorer/backend/apperrors"
)

var allowedPragmas = map[string]bool{
	"table_info":       true,
	"foreign_key_list": true,
	"index_list":       true,
	"index_info":       true,
	"database_list":    true,
	"schema_version":   true,
	"integrity_check":  true,
	"quick_check":      true,
	"compile_options":  true,
	"encoding":         true,
	"application_id":   true,
	"user_version":     true,
}

var pragmasWithReadOnlyArguments = map[string]bool{
	"table_info":       true,
	"foreign_key_list": true,
	"index_list":       true,
	"index_info":       true,
	"integrity_check":  true,
	"quick_check":      true,
}

var forbiddenKeywords = map[string]bool{
	"INSERT": true, "UPDATE": true, "DELETE": true, "DROP": true,
	"ALTER": true, "CREATE": true, "REPLACE": true, "VACUUM": true,
	"ATTACH": true, "DETACH": true, "TRUNCATE": true,
}

// ValidateReadOnlySQL ensures every statement in sql is read-only.
func ValidateReadOnlySQL(sql string) error {
	trimmed := strings.TrimSpace(sql)
	if trimmed == "" {
		return apperrors.New(apperrors.CodeMalformedSQL, "SQL is empty.", "")
	}

	stmts := splitStatements(trimmed)
	if len(stmts) == 0 {
		return apperrors.New(apperrors.CodeMalformedSQL, "SQL is empty.", "")
	}

	for _, stmt := range stmts {
		if err := validateStatement(stmt); err != nil {
			return err
		}
	}
	return nil
}

func validateStatement(stmt string) error {
	s := stripComments(strings.TrimSpace(stmt))
	if s == "" {
		return nil
	}

	kw, rest := firstKeyword(s)
	if kw == "" {
		return apperrors.New(apperrors.CodeMalformedSQL, "Could not parse SQL statement.", stmt)
	}

	upper := strings.ToUpper(kw)
	switch upper {
	case "SELECT":
		return nil
	case "PRAGMA":
		return validatePragma(rest)
	case "WITH":
		return validateWithStatement(s)
	default:
		if forbiddenKeywords[upper] {
			return readOnlyViolation(upper)
		}
		return readOnlyViolation(upper)
	}
}

func validatePragma(rest string) error {
	name := pragmaName(rest)
	if name == "" {
		return apperrors.New(apperrors.CodeMalformedSQL, "Invalid PRAGMA statement.", rest)
	}
	lower := strings.ToLower(name)
	if lower == "writable_schema" {
		return readOnlyViolation("writable_schema")
	}
	if !allowedPragmas[lower] {
		return readOnlyViolation("PRAGMA " + name)
	}
	tail := strings.TrimSpace(rest[len(name):])
	if tail != "" && !pragmasWithReadOnlyArguments[lower] {
		return readOnlyViolation("PRAGMA " + name)
	}
	return nil
}

func pragmaName(rest string) string {
	rest = strings.TrimSpace(rest)
	if rest == "" {
		return ""
	}
	var b strings.Builder
	for i, r := range rest {
		if r == '(' || unicode.IsSpace(r) {
			if b.Len() > 0 {
				return b.String()
			}
			if r == '(' {
				return b.String()
			}
			continue
		}
		if i == 0 && (r == '"' || r == '\'' || r == '`') {
			return ""
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			b.WriteRune(r)
		} else {
			break
		}
	}
	return b.String()
}

func validateWithStatement(s string) error {
	// Reject mutating CTE bodies or outer statements.
	upper := strings.ToUpper(stripStringLiterals(s))
	forbidden := []string{
		" INSERT ", " UPDATE ", " DELETE ", " DROP ", " CREATE ",
		" ALTER ", " REPLACE ", " ATTACH ", " DETACH ", " VACUUM ",
	}
	padded := " " + upper + " "
	for _, f := range forbidden {
		if strings.Contains(padded, f) {
			kw := strings.TrimSpace(f)
			return readOnlyViolation(kw)
		}
	}
	// Outer query must be SELECT (not WITH ... INSERT etc. as final op).
	if !strings.Contains(upper, "SELECT") {
		return readOnlyViolation("WITH")
	}
	return nil
}

func readOnlyViolation(keyword string) error {
	return apperrors.New(
		apperrors.CodeReadOnlyViolation,
		"Read-only mode: "+keyword+" statements are not allowed.",
		keyword,
	)
}

func stripComments(s string) string {
	var b strings.Builder
	i := 0
	for i < len(s) {
		if i+1 < len(s) && s[i] == '-' && s[i+1] == '-' {
			for i < len(s) && s[i] != '\n' {
				i++
			}
			continue
		}
		if i+1 < len(s) && s[i] == '/' && s[i+1] == '*' {
			i += 2
			for i+1 < len(s) && !(s[i] == '*' && s[i+1] == '/') {
				i++
			}
			if i+1 < len(s) {
				i += 2
			}
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

func stripStringLiterals(s string) string {
	var b strings.Builder
	inSingle := false
	inDouble := false
	i := 0
	for i < len(s) {
		c := s[i]
		if inSingle {
			if c == '\'' {
				if i+1 < len(s) && s[i+1] == '\'' {
					i += 2
					continue
				}
				inSingle = false
			}
			i++
			continue
		}
		if inDouble {
			if c == '"' {
				if i+1 < len(s) && s[i+1] == '"' {
					i += 2
					continue
				}
				inDouble = false
			}
			i++
			continue
		}
		if c == '\'' {
			inSingle = true
			b.WriteByte(' ')
			i++
			continue
		}
		if c == '"' {
			inDouble = true
			b.WriteByte(' ')
			i++
			continue
		}
		b.WriteByte(c)
		i++
	}
	return b.String()
}

func firstKeyword(s string) (string, string) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", ""
	}
	i := 0
	for i < len(s) && unicode.IsSpace(rune(s[i])) {
		i++
	}
	start := i
	for i < len(s) {
		r := rune(s[i])
		if unicode.IsLetter(r) || r == '_' || (i > start && unicode.IsDigit(r)) {
			i++
			continue
		}
		break
	}
	if i == start {
		return "", s
	}
	return s[start:i], strings.TrimSpace(s[i:])
}

func splitStatements(sql string) []string {
	var stmts []string
	var b strings.Builder
	inSingle := false
	inDouble := false
	depth := 0

	for i := 0; i < len(sql); i++ {
		c := sql[i]

		if inSingle {
			b.WriteByte(c)
			if c == '\'' {
				if i+1 < len(sql) && sql[i+1] == '\'' {
					b.WriteByte(sql[i+1])
					i++
				} else {
					inSingle = false
				}
			}
			continue
		}
		if inDouble {
			b.WriteByte(c)
			if c == '"' {
				if i+1 < len(sql) && sql[i+1] == '"' {
					b.WriteByte(sql[i+1])
					i++
				} else {
					inDouble = false
				}
			}
			continue
		}

		switch c {
		case '\'':
			inSingle = true
			b.WriteByte(c)
		case '"':
			inDouble = true
			b.WriteByte(c)
		case '(':
			depth++
			b.WriteByte(c)
		case ')':
			if depth > 0 {
				depth--
			}
			b.WriteByte(c)
		case ';':
			if depth == 0 {
				if stmt := strings.TrimSpace(b.String()); stmt != "" {
					stmts = append(stmts, stmt)
				}
				b.Reset()
				continue
			}
			b.WriteByte(c)
		default:
			b.WriteByte(c)
		}
	}
	if stmt := strings.TrimSpace(b.String()); stmt != "" {
		stmts = append(stmts, stmt)
	}
	return stmts
}
