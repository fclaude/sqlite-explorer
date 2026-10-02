package db

import (
	"fmt"
	"sort"
	"strings"

	"sqlite-explorer/backend/apperrors"
	"sqlite-explorer/backend/model"
)

// Category groups SQL statements by what they can change. Read statements always run;
// the other categories run only when the user enables them for the SQL editor.
type Category string

const (
	CategoryRead        Category = "read"
	CategoryData        Category = "data"
	CategorySchema      Category = "schema"
	CategoryTransaction Category = "transaction"
	CategoryMaintenance Category = "maintenance"
	CategoryAttach      Category = "attach"
	CategoryPragma      Category = "pragma"
	CategoryForbidden   Category = "forbidden"
)

type categoryInfo struct {
	id          Category
	label       string
	description string
	statements  []string
}

// categories lists every category in display order. Keep descriptions in sync with README.md.
var categories = []categoryInfo{
	{
		id:          CategoryRead,
		label:       "Read queries",
		description: "Always allowed. Runs on a read-only connection.",
		statements:  []string{"SELECT", "VALUES", "WITH … SELECT", "EXPLAIN", "read-only PRAGMAs"},
	},
	{
		id:          CategoryData,
		label:       "Data changes",
		description: "Insert, update, and delete rows.",
		statements:  []string{"INSERT", "REPLACE", "UPDATE", "DELETE", "WITH … INSERT/UPDATE/DELETE"},
	},
	{
		id:          CategorySchema,
		label:       "Schema changes",
		description: "Create, alter, and drop tables, views, indexes, and triggers.",
		statements:  []string{"CREATE", "ALTER", "DROP"},
	},
	{
		id:          CategoryTransaction,
		label:       "Transactions",
		description: "Group statements in one run. A transaction left open when the run ends is rolled back.",
		statements:  []string{"BEGIN", "COMMIT", "END", "ROLLBACK", "SAVEPOINT", "RELEASE"},
	},
	{
		id:          CategoryMaintenance,
		label:       "Maintenance",
		description: "Rebuild the file or its statistics. VACUUM INTO writes a copy of the database to a new file.",
		statements:  []string{"VACUUM", "VACUUM INTO", "ANALYZE", "REINDEX"},
	},
	{
		id:          CategoryAttach,
		label:       "Attach databases",
		description: "Open other database files during a run. ATTACH can create a new file.",
		statements:  []string{"ATTACH", "DETACH"},
	},
	{
		id:          CategoryPragma,
		label:       "Other PRAGMAs",
		description: "Any PRAGMA not on the read-only list, including assignments. Connection settings last only for that run.",
		statements:  []string{"PRAGMA name = value", "PRAGMA name(value)"},
	},
	{
		id:          CategoryForbidden,
		label:       "Never allowed",
		description: "Can corrupt the database file.",
		statements:  []string{"PRAGMA writable_schema"},
	},
}

// readOnlyPragmas may run without arguments on the read-only connection.
var readOnlyPragmas = map[string]bool{
	"application_id":    true,
	"auto_vacuum":       true,
	"busy_timeout":      true,
	"cache_size":        true,
	"collation_list":    true,
	"compile_options":   true,
	"data_version":      true,
	"database_list":     true,
	"encoding":          true,
	"foreign_key_check": true,
	"foreign_key_list":  true,
	"foreign_keys":      true,
	"freelist_count":    true,
	"function_list":     true,
	"index_info":        true,
	"index_list":        true,
	"index_xinfo":       true,
	"integrity_check":   true,
	"journal_mode":      true,
	"module_list":       true,
	"page_count":        true,
	"page_size":         true,
	"pragma_list":       true,
	"quick_check":       true,
	"schema_version":    true,
	"synchronous":       true,
	"table_info":        true,
	"table_list":        true,
	"table_xinfo":       true,
	"user_version":      true,
}

// readOnlyPragmaArgs lists pragmas whose argument names an object or limit rather than a new setting.
var readOnlyPragmaArgs = map[string]bool{
	"foreign_key_check": true,
	"foreign_key_list":  true,
	"index_info":        true,
	"index_list":        true,
	"index_xinfo":       true,
	"integrity_check":   true,
	"quick_check":       true,
	"table_info":        true,
	"table_list":        true,
	"table_xinfo":       true,
}

var neverAllowedPragmas = map[string]bool{
	"writable_schema": true,
}

var keywordCategories = map[string]Category{
	"SELECT":    CategoryRead,
	"VALUES":    CategoryRead,
	"EXPLAIN":   CategoryRead,
	"INSERT":    CategoryData,
	"REPLACE":   CategoryData,
	"UPDATE":    CategoryData,
	"DELETE":    CategoryData,
	"CREATE":    CategorySchema,
	"ALTER":     CategorySchema,
	"DROP":      CategorySchema,
	"BEGIN":     CategoryTransaction,
	"COMMIT":    CategoryTransaction,
	"END":       CategoryTransaction,
	"ROLLBACK":  CategoryTransaction,
	"SAVEPOINT": CategoryTransaction,
	"RELEASE":   CategoryTransaction,
	"VACUUM":    CategoryMaintenance,
	"ANALYZE":   CategoryMaintenance,
	"REINDEX":   CategoryMaintenance,
	"ATTACH":    CategoryAttach,
	"DETACH":    CategoryAttach,
}

// StatementCategories describes every category for the SQL editor's permissions panel.
func StatementCategories() []model.StatementCategory {
	out := make([]model.StatementCategory, 0, len(categories))
	for _, c := range categories {
		info := model.StatementCategory{
			ID:           string(c.id),
			Label:        c.label,
			Description:  c.description,
			Statements:   append([]string(nil), c.statements...),
			Configurable: c.id != CategoryRead && c.id != CategoryForbidden,
		}
		if c.id == CategoryRead {
			info.Pragmas = readOnlyPragmaNames()
		}
		out = append(out, info)
	}
	return out
}

func categoryLabel(c Category) string {
	for _, info := range categories {
		if info.id == c {
			return info.label
		}
	}
	return string(c)
}

// Policy is the set of statement categories a run may execute in addition to reads.
type Policy map[Category]bool

// ReadOnlyPolicy allows only read statements.
var ReadOnlyPolicy = Policy{}

// ParsePolicy converts category ids from the frontend into a Policy.
func ParsePolicy(allow []string) (Policy, error) {
	p := Policy{}
	for _, id := range allow {
		c := Category(id)
		configurable := false
		for _, info := range categories {
			if info.id == c && c != CategoryRead && c != CategoryForbidden {
				configurable = true
			}
		}
		if !configurable {
			return nil, apperrors.New(apperrors.CodeMalformedSQL, "Unknown statement permission.", id)
		}
		p[c] = true
	}
	return p, nil
}

func (p Policy) allows(c Category) bool {
	return c == CategoryRead || p[c]
}

// Statement is one validated statement ready to execute.
type Statement struct {
	SQL      string
	Category Category
	Label    string
}

// Plan is a validated SQL run.
type Plan struct {
	Statements []Statement
}

// Writes reports whether any statement needs the read-write connection.
func (p Plan) Writes() bool {
	for _, s := range p.Statements {
		if s.Category != CategoryRead {
			return true
		}
	}
	return false
}

func (p Plan) has(c Category) bool {
	for _, s := range p.Statements {
		if s.Category == c {
			return true
		}
	}
	return false
}

// PlanSQL splits sql into statements, classifies each one, and checks it against policy.
func PlanSQL(sql string, policy Policy) (Plan, error) {
	if strings.TrimSpace(sql) == "" {
		return Plan{}, apperrors.New(apperrors.CodeMalformedSQL, "SQL is empty.", "")
	}
	stmts, err := splitStatements(sql)
	if err != nil {
		return Plan{}, err
	}
	if len(stmts) == 0 {
		return Plan{}, apperrors.New(apperrors.CodeMalformedSQL, "SQL is empty.", "")
	}

	plan := Plan{Statements: make([]Statement, 0, len(stmts))}
	for i, stmt := range stmts {
		category, label, err := classify(stmt.tokens)
		if err != nil {
			return Plan{}, withStatementPosition(err, i, len(stmts))
		}
		if category == CategoryForbidden {
			return Plan{}, apperrors.New(
				apperrors.CodeReadOnlyViolation,
				statementPrefix(i, len(stmts))+label+" is never allowed because it can corrupt the database file.",
				label,
			)
		}
		if !policy.allows(category) {
			return Plan{}, apperrors.New(
				apperrors.CodeReadOnlyViolation,
				fmt.Sprintf("%s%s is not allowed. Enable %q under Permissions to run it.",
					statementPrefix(i, len(stmts)), label, categoryLabel(category)),
				string(category),
			)
		}
		plan.Statements = append(plan.Statements, Statement{SQL: stmt.text, Category: category, Label: label})
	}
	return plan, nil
}

// ValidateReadOnlySQL ensures every statement in sql is a read statement.
func ValidateReadOnlySQL(sql string) error {
	_, err := PlanSQL(sql, ReadOnlyPolicy)
	return err
}

func classify(tokens []token) (Category, string, error) {
	first := tokens[0]
	kw := first.keyword()
	if kw == "" {
		return "", "", apperrors.New(apperrors.CodeMalformedSQL, "Could not recognize the SQL statement.", first.text)
	}
	switch kw {
	case "WITH":
		return classifyWith(tokens)
	case "PRAGMA":
		return classifyPragma(tokens[1:])
	case "VACUUM":
		for _, t := range tokens[1:] {
			if t.is("INTO") {
				return CategoryMaintenance, "VACUUM INTO", nil
			}
		}
	}
	if c, ok := keywordCategories[kw]; ok {
		return c, kw, nil
	}
	return "", "", apperrors.New(apperrors.CodeMalformedSQL, "Unrecognized SQL statement: "+kw+".", kw)
}

// classifyWith finds the statement that follows the common table expressions. Every CTE body
// is parenthesized, so the main statement is the first top-level keyword after a closing
// parenthesis that is not AS (which follows a CTE column list).
func classifyWith(tokens []token) (Category, string, error) {
	depth := 0
	prevClose := false
	for _, t := range tokens[1:] {
		switch t.kind {
		case tokLParen:
			depth++
			prevClose = false
			continue
		case tokRParen:
			if depth > 0 {
				depth--
			}
			prevClose = depth == 0
			continue
		}
		if depth == 0 && prevClose && t.kind == tokWord && !t.is("AS") {
			switch kw := t.keyword(); kw {
			case "SELECT", "VALUES":
				return CategoryRead, "WITH … " + kw, nil
			case "INSERT", "REPLACE", "UPDATE", "DELETE":
				return CategoryData, "WITH … " + kw, nil
			default:
				return "", "", apperrors.New(apperrors.CodeMalformedSQL, "Could not recognize the statement after WITH.", kw)
			}
		}
		prevClose = false
	}
	return "", "", apperrors.New(apperrors.CodeMalformedSQL, "WITH must be followed by a statement.", "")
}

func classifyPragma(rest []token) (Category, string, error) {
	// PRAGMA [schema.]name [= value | (value)]
	if len(rest) == 0 || (rest[0].kind != tokWord && rest[0].kind != tokQuoted) {
		return "", "", apperrors.New(apperrors.CodeMalformedSQL, "Invalid PRAGMA statement.", "")
	}
	nameTok := rest[0]
	args := rest[1:]
	if len(rest) >= 3 && rest[1].kind == tokOther && rest[1].text == "." {
		nameTok = rest[2]
		args = rest[3:]
	}
	if nameTok.kind != tokWord {
		return "", "", apperrors.New(apperrors.CodeMalformedSQL, "Invalid PRAGMA name.", nameTok.text)
	}
	name := asciiLower(nameTok.text)
	label := "PRAGMA " + name
	switch {
	case neverAllowedPragmas[name]:
		return CategoryForbidden, label, nil
	case len(args) == 0 && readOnlyPragmas[name]:
		return CategoryRead, label, nil
	case len(args) > 0 && readOnlyPragmaArgs[name]:
		return CategoryRead, label, nil
	}
	return CategoryPragma, label, nil
}

func asciiLower(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c - 'A' + 'a'
		}
	}
	return string(b)
}

func statementPrefix(i, n int) string {
	if n <= 1 {
		return ""
	}
	return fmt.Sprintf("Statement %d of %d: ", i+1, n)
}

func withStatementPosition(err error, i, n int) error {
	appErr, ok := apperrors.As(err)
	if !ok || n <= 1 {
		return err
	}
	return apperrors.New(appErr.Code, statementPrefix(i, n)+appErr.Message, appErr.Detail)
}

func readOnlyPragmaNames() []string {
	names := make([]string, 0, len(readOnlyPragmas))
	for name := range readOnlyPragmas {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
