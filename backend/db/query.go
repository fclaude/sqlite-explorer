package db

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"sqlite-explorer/backend/apperrors"
	"sqlite-explorer/backend/model"
)

const (
	defaultPageSize     = 100
	maxBlobPreview      = 64
	jsMaxSafeInteger    = 9007199254740991
	tableQueryTimeout   = 30 * time.Second
	MaxQueryRows        = 1000
	DefaultQueryTimeout = 30 * time.Second
	cleanupTimeout      = 5 * time.Second
)

var allowedPageSizes = map[int]bool{50: true, 100: true, 500: true, 1000: true}

// tableSelect is a validated SELECT over one table or view.
type tableSelect struct {
	columns    []model.ColumnInfo
	resultCols []model.ColumnResult
	useRowID   bool
	page       int
	pageSize   int
	query      string
	args       []any
}

// prepareTableSelect validates req and builds its SELECT. With paginate, the query is limited
// to one page; otherwise it selects every row matching the filter in sort order.
func (d *DB) prepareTableSelect(ctx context.Context, req model.TableRowsRequest, paginate, withRowID bool) (tableSelect, error) {
	sel := tableSelect{page: req.Page, pageSize: req.PageSize}
	if paginate {
		if sel.pageSize == 0 {
			sel.pageSize = defaultPageSize
		}
		if !allowedPageSizes[sel.pageSize] {
			return tableSelect{}, fmt.Errorf("invalid page size: %d", sel.pageSize)
		}
		if sel.page < 1 {
			sel.page = 1
		}
	}

	if err := d.validateTableOrView(ctx, req.Table); err != nil {
		return tableSelect{}, err
	}
	columns, err := d.loadColumns(ctx, req.Table)
	if err != nil {
		return tableSelect{}, err
	}
	if len(columns) == 0 {
		return tableSelect{}, fmt.Errorf("table %s has no columns", req.Table)
	}
	if req.SortColumn != "" && !columnExists(columns, req.SortColumn) {
		return tableSelect{}, apperrors.New(
			apperrors.CodeInvalidColumn,
			"Sort column does not exist on this table.",
			req.SortColumn,
		)
	}
	quotedTable, err := QuoteIdentifier(req.Table)
	if err != nil {
		return tableSelect{}, err
	}

	selectCols := make([]string, 0, len(columns)+1)
	if withRowID {
		if alias, ok := d.editableRowIDAlias(ctx, req.Table, columns); ok {
			quotedRowID, err := QuoteIdentifier(alias)
			if err != nil {
				return tableSelect{}, err
			}
			selectCols = append(selectCols, quotedRowID)
			sel.useRowID = true
		}
	}
	sel.resultCols = make([]model.ColumnResult, len(columns))
	for i, col := range columns {
		expr, err := selectColumn(col.Name)
		if err != nil {
			return tableSelect{}, err
		}
		selectCols = append(selectCols, expr)
		sel.resultCols[i] = model.ColumnResult{Name: col.Name, Type: col.Type}
	}
	sel.columns = columns

	query := "SELECT " + strings.Join(selectCols, ", ") + " FROM " + quotedTable
	where, args, err := filterClause(columns, req.Filter)
	if err != nil {
		return tableSelect{}, err
	}
	query += where

	if req.SortColumn != "" {
		sortQ, err := QuoteIdentifier(req.SortColumn)
		if err != nil {
			return tableSelect{}, err
		}
		query += " ORDER BY " + sortQ
		if req.SortDesc {
			query += " DESC"
		} else {
			query += " ASC"
		}
	}
	if paginate {
		query += " LIMIT ? OFFSET ?"
		args = append(args, sel.pageSize, (sel.page-1)*sel.pageSize)
	}
	sel.query = query
	sel.args = args
	return sel, nil
}

// selectColumn quotes a column for a SELECT list. Unary + returns the stored value unchanged
// but hides the declared type, which stops the driver from turning TEXT in DATE, DATETIME,
// and TIMESTAMP columns into time.Time and losing the stored text.
func selectColumn(name string) (string, error) {
	q, err := QuoteIdentifier(name)
	if err != nil {
		return "", err
	}
	return "+" + q, nil
}

// filterClause matches filter as a substring of any column's text form.
func filterClause(columns []model.ColumnInfo, filter string) (string, []any, error) {
	if filter == "" || len(columns) == 0 {
		return "", nil, nil
	}
	pattern := "%" + escapeLike(filter) + "%"
	parts := make([]string, len(columns))
	args := make([]any, len(columns))
	for i, col := range columns {
		q, err := QuoteIdentifier(col.Name)
		if err != nil {
			return "", nil, err
		}
		parts[i] = fmt.Sprintf("CAST(%s AS TEXT) LIKE ? ESCAPE '\\'", q)
		args[i] = pattern
	}
	return " WHERE (" + strings.Join(parts, " OR ") + ")", args, nil
}

// GetTableRows returns a paginated page of rows from a validated table or view.
func (d *DB) GetTableRows(ctx context.Context, req model.TableRowsRequest) (model.TableRowsResponse, error) {
	start := time.Now()
	ctx, cancel := context.WithTimeout(ctx, tableQueryTimeout)
	defer cancel()

	sel, err := d.prepareTableSelect(ctx, req, true, true)
	if err != nil {
		return model.TableRowsResponse{}, err
	}

	rows, err := d.sql.QueryContext(ctx, sel.query, sel.args...)
	if err != nil {
		return model.TableRowsResponse{}, mapQueryError(ctx, err)
	}
	defer rows.Close()

	var result [][]model.CellValue
	var rowIDs []string
	for rows.Next() {
		dest := make([]any, len(sel.columns))
		ptrs := make([]any, 0, len(sel.columns)+1)
		var sqliteRowID int64
		if sel.useRowID {
			ptrs = append(ptrs, &sqliteRowID)
		}
		for i := range dest {
			ptrs = append(ptrs, &dest[i])
		}
		if err := rows.Scan(ptrs...); err != nil {
			return model.TableRowsResponse{}, err
		}
		if sel.useRowID {
			rowIDs = append(rowIDs, strconv.FormatInt(sqliteRowID, 10))
		}
		row := make([]model.CellValue, len(sel.columns))
		for i := range sel.columns {
			row[i] = coerceValue(dest[i])
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return model.TableRowsResponse{}, mapQueryError(ctx, err)
	}

	resp := model.TableRowsResponse{
		Columns:    sel.resultCols,
		Rows:       result,
		RowIDs:     rowIDs,
		Editable:   sel.useRowID,
		Page:       sel.page,
		PageSize:   sel.pageSize,
		DurationMs: time.Since(start).Milliseconds(),
	}

	if req.WithTotal {
		total, err := d.cachedRowCount(ctx, req.Table, sel.columns, req.Filter)
		if err != nil {
			return model.TableRowsResponse{}, mapQueryError(ctx, err)
		}
		resp.TotalRows = &total
	}

	return resp, nil
}

func (d *DB) validateTableOrView(ctx context.Context, name string) error {
	const q = `SELECT 1 FROM sqlite_schema WHERE type IN ('table','view') AND name = ? AND name NOT LIKE 'sqlite_%'`
	var one int
	err := d.sql.QueryRowContext(ctx, q, name).Scan(&one)
	if err == sql.ErrNoRows {
		return apperrors.New(apperrors.CodeUnknownTable, "Table or view not found.", name)
	}
	if err != nil {
		return fmt.Errorf("validate table: %w", err)
	}
	return nil
}

func columnExists(cols []model.ColumnInfo, name string) bool {
	for _, c := range cols {
		if c.Name == name {
			return true
		}
	}
	return false
}

func escapeLike(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '\\', '%', '_':
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}

// cachedRowCount caches COUNT(*) results until any connection or process commits to the file.
func (d *DB) cachedRowCount(ctx context.Context, table string, columns []model.ColumnInfo, filter string) (int64, error) {
	var version int64
	if err := d.versionConn.QueryRowContext(ctx, `PRAGMA data_version`).Scan(&version); err != nil {
		return 0, fmt.Errorf("pragma data_version: %w", err)
	}

	key := countCacheKey{table: table, filter: filter}
	d.countMu.Lock()
	if version != d.countVersion {
		clear(d.rowCountCache)
		d.countVersion = version
	}
	n, ok := d.rowCountCache[key]
	d.countMu.Unlock()
	if ok {
		return n, nil
	}

	n, err := d.countRows(ctx, table, columns, filter)
	if err != nil {
		return 0, err
	}
	d.countMu.Lock()
	if d.countVersion == version {
		if len(d.rowCountCache) >= maxCachedCounts {
			clear(d.rowCountCache)
		}
		d.rowCountCache[key] = n
	}
	d.countMu.Unlock()
	return n, nil
}

func (d *DB) countRows(ctx context.Context, table string, columns []model.ColumnInfo, filter string) (int64, error) {
	quotedTable, err := QuoteIdentifier(table)
	if err != nil {
		return 0, err
	}
	where, args, err := filterClause(columns, filter)
	if err != nil {
		return 0, err
	}
	var count int64
	if err := d.sql.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+quotedTable+where, args...).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

// coerceValue converts a driver value into a grid cell. The driver returns []byte only for
// the BLOB storage class.
func coerceValue(raw any) model.CellValue {
	switch v := raw.(type) {
	case nil:
		return model.CellValue{Kind: "null", Value: nil}
	case int64:
		return coerceInt(v)
	case float64:
		return coerceReal(v)
	case string:
		return model.CellValue{Kind: "text", Value: v}
	case []byte:
		return coerceBlob(v)
	case time.Time:
		return model.CellValue{Kind: "text", Value: formatSQLiteTime(v)}
	default:
		return model.CellValue{Kind: "text", Value: fmt.Sprint(v)}
	}
}

// formatSQLiteTime renders a driver-parsed time in SQLite's datetime format. The driver
// parses TEXT from DATE, DATETIME, and TIMESTAMP result columns of ad-hoc queries, so the
// stored text itself is not available there.
func formatSQLiteTime(t time.Time) string {
	layout := "2006-01-02 15:04:05"
	if t.Nanosecond() != 0 {
		layout += ".999999999"
	}
	if _, offset := t.Zone(); offset != 0 {
		layout += "-07:00"
	}
	return t.Format(layout)
}

func coerceInt(v int64) model.CellValue {
	if v > jsMaxSafeInteger || v < -jsMaxSafeInteger {
		return model.CellValue{Kind: "int", Value: strconv.FormatInt(v, 10)}
	}
	return model.CellValue{Kind: "int", Value: v}
}

func coerceReal(v float64) model.CellValue {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return model.CellValue{Kind: "real", Value: strconv.FormatFloat(v, 'g', -1, 64)}
	}
	return model.CellValue{Kind: "real", Value: v}
}

func coerceBlob(b []byte) model.CellValue {
	n := len(b)
	preview := n
	if preview > maxBlobPreview {
		preview = maxBlobPreview
	}
	return model.CellValue{
		Kind: "blob",
		Value: model.BlobValue{
			Hex:  hex.EncodeToString(b[:preview]),
			Size: n,
		},
	}
}

// RunQuery checks query against policy, runs every statement, and returns up to MaxQueryRows
// rows from the last statement.
func (d *DB) RunQuery(ctx context.Context, query string, policy Policy) (model.QueryResponse, error) {
	start := time.Now()
	plan, err := PlanSQL(query, policy)
	if err != nil {
		return model.QueryResponse{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, DefaultQueryTimeout)
	defer cancel()

	var resp model.QueryResponse
	affected, err := d.execPlan(ctx, plan, func(rows *sql.Rows) error {
		cols, err := resultColumns(rows)
		if err != nil {
			return err
		}
		resp.Columns = cols
		for rows.Next() {
			if len(resp.Rows) >= MaxQueryRows {
				resp.Truncated = true
				break
			}
			values, err := scanValues(rows, len(cols))
			if err != nil {
				return err
			}
			row := make([]model.CellValue, len(values))
			for i, v := range values {
				row[i] = coerceValue(v)
			}
			resp.Rows = append(resp.Rows, row)
		}
		return rows.Err()
	})
	if err != nil {
		return model.QueryResponse{}, err
	}

	resp.RowCount = len(resp.Rows)
	resp.StatementCount = len(plan.Statements)
	resp.RowsAffected = affected
	resp.Changed = plan.Writes()
	resp.SchemaChanged = plan.has(CategorySchema)
	resp.DurationMs = time.Since(start).Milliseconds()
	return resp, nil
}

func resultColumns(rows *sql.Rows) ([]model.ColumnResult, error) {
	names, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	types, _ := rows.ColumnTypes()
	cols := make([]model.ColumnResult, len(names))
	for i, name := range names {
		cols[i] = model.ColumnResult{Name: name}
		if i < len(types) && types[i] != nil {
			cols[i].Type = types[i].DatabaseTypeName()
		}
	}
	return cols, nil
}

func scanValues(rows *sql.Rows, n int) ([]any, error) {
	dest := make([]any, n)
	ptrs := make([]any, n)
	for i := range dest {
		ptrs[i] = &dest[i]
	}
	if err := rows.Scan(ptrs...); err != nil {
		return nil, err
	}
	return dest, nil
}

// execPlan runs every statement of plan on one connection and passes the rows of the last
// statement to consume. Read-only plans use the read-only connection. Plans with writes use
// a dedicated read-write connection that is discarded afterwards, so temporary objects,
// attached databases, PRAGMA settings, and open transactions never leak into the pool.
func (d *DB) execPlan(ctx context.Context, plan Plan, consume func(*sql.Rows) error) (int64, error) {
	writes := plan.Writes()
	pool := d.querySQL
	if writes {
		if d.readOnly {
			return 0, apperrors.New(
				apperrors.CodeReadOnlyViolation,
				"The database is open read-only, so only read statements can run.",
				"",
			)
		}
		pool = d.sql
	}

	conn, err := pool.Conn(ctx)
	if err != nil {
		return 0, mapQueryError(ctx, err)
	}
	if !writes {
		defer conn.Close()
		affected, failedAt, err := runStatements(ctx, conn, plan.Statements, consume)
		return affected, statementFailure(ctx, err, plan, failedAt, "")
	}
	defer discardConn(conn)

	affected, failedAt, err := runStatements(ctx, conn, plan.Statements, consume)
	rolledBack := rollbackOpenTransaction(conn)
	if err != nil {
		note := ""
		if rolledBack {
			note = " The open transaction was rolled back."
		} else if (Plan{Statements: plan.Statements[:failedAt]}).Writes() {
			note = " Earlier statements in this run were applied."
		}
		return affected, statementFailure(ctx, err, plan, failedAt, note)
	}
	if rolledBack {
		return affected, apperrors.New(
			apperrors.CodeTransactionRolledBack,
			"The run ended inside an open transaction, so it was rolled back. End the script with COMMIT to keep its changes.",
			"",
		)
	}
	return affected, nil
}

// runStatements executes stmts in order. It returns the number of rows changed by data
// statements and, on failure, the index of the statement that failed.
func runStatements(ctx context.Context, conn *sql.Conn, stmts []Statement, consume func(*sql.Rows) error) (int64, int, error) {
	var affected int64
	for i, st := range stmts {
		if i < len(stmts)-1 {
			res, err := conn.ExecContext(ctx, st.SQL)
			if err != nil {
				return affected, i, err
			}
			if st.Category == CategoryData {
				if n, err := res.RowsAffected(); err == nil {
					affected += n
				}
			}
			continue
		}

		rows, err := conn.QueryContext(ctx, st.SQL)
		if err != nil {
			return affected, i, err
		}
		err = consume(rows)
		if closeErr := rows.Close(); err == nil {
			err = closeErr
		}
		if err != nil {
			return affected, i, err
		}
		if st.Category == CategoryData {
			var n int64
			if err := conn.QueryRowContext(ctx, `SELECT changes()`).Scan(&n); err == nil {
				affected += n
			}
		}
	}
	return affected, -1, nil
}

// rollbackOpenTransaction rolls back a transaction the run left open and reports whether
// there was one.
func rollbackOpenTransaction(conn *sql.Conn) bool {
	ctx, cancel := context.WithTimeout(context.Background(), cleanupTimeout)
	defer cancel()
	_, err := conn.ExecContext(ctx, `ROLLBACK`)
	return err == nil
}

// discardConn closes conn instead of returning it to the pool.
func discardConn(conn *sql.Conn) {
	_ = conn.Raw(func(any) error { return driver.ErrBadConn })
	_ = conn.Close()
}

func statementFailure(ctx context.Context, err error, plan Plan, failedAt int, note string) error {
	if err == nil {
		return nil
	}
	appErr, _ := apperrors.As(mapQueryError(ctx, err))
	prefix := ""
	n := len(plan.Statements)
	if n > 1 && failedAt >= 0 && appErr.Code != apperrors.CodeCancelled && appErr.Code != apperrors.CodeTimeout {
		prefix = fmt.Sprintf("Statement %d of %d (%s) failed: ", failedAt+1, n, plan.Statements[failedAt].Label)
	}
	return apperrors.New(appErr.Code, prefix+appErr.Message+note, appErr.Detail)
}

// mapQueryError converts driver and context errors into application errors.
func mapQueryError(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return apperrors.New(apperrors.CodeTimeout, "Query timed out.", "")
	}
	if errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled) {
		return apperrors.New(apperrors.CodeCancelled, "Query cancelled.", "")
	}
	if appErr, ok := apperrors.As(err); ok {
		return appErr
	}
	msg := err.Error()
	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "syntax error") || strings.Contains(lower, "incomplete input"):
		return apperrors.New(apperrors.CodeMalformedSQL, "The SQL could not be parsed. Check your syntax.", msg)
	case strings.Contains(lower, "readonly database"):
		return apperrors.New(apperrors.CodeReadOnlyViolation, "The database file is read-only.", msg)
	}
	return apperrors.New(apperrors.CodeMalformedSQL, "The query could not be executed.", msg)
}
