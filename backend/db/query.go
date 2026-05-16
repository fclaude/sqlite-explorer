package db

import (
	"context"
	"database/sql"
	"encoding/hex"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"sqlite-explorer/backend/apperrors"
	"sqlite-explorer/backend/model"
)

const (
	defaultPageSize   = 100
	maxBlobPreview    = 64
	jsMaxSafeInteger  = 9007199254740991
	tableQueryTimeout = 30 * time.Second
)

var allowedPageSizes = map[int]bool{50: true, 100: true, 500: true, 1000: true}

// GetTableRows returns a paginated page of rows from a validated table or view.
func (d *DB) GetTableRows(ctx context.Context, req model.TableRowsRequest) (model.TableRowsResponse, error) {
	start := time.Now()
	ctx, cancel := context.WithTimeout(ctx, tableQueryTimeout)
	defer cancel()

	pageSize := req.PageSize
	if pageSize == 0 {
		pageSize = defaultPageSize
	}
	if !allowedPageSizes[pageSize] {
		return model.TableRowsResponse{}, fmt.Errorf("invalid page size: %d", pageSize)
	}
	page := req.Page
	if page < 1 {
		page = 1
	}

	if err := d.validateTableOrView(ctx, req.Table); err != nil {
		return model.TableRowsResponse{}, err
	}

	columns, err := d.loadColumns(ctx, req.Table)
	if err != nil {
		return model.TableRowsResponse{}, err
	}
	if len(columns) == 0 {
		return model.TableRowsResponse{}, fmt.Errorf("table %s has no columns", req.Table)
	}

	if req.SortColumn != "" {
		if !columnExists(columns, req.SortColumn) {
			return model.TableRowsResponse{}, apperrors.New(
				apperrors.CodeInvalidColumn,
				"Sort column does not exist on this table.",
				req.SortColumn,
			)
		}
	}

	quotedTable, err := QuoteIdentifier(req.Table)
	if err != nil {
		return model.TableRowsResponse{}, err
	}

	selectCols := make([]string, len(columns))
	colTypes := make(map[string]string, len(columns))
	resultCols := make([]model.ColumnResult, len(columns))
	for i, col := range columns {
		q, err := QuoteIdentifier(col.Name)
		if err != nil {
			return model.TableRowsResponse{}, err
		}
		selectCols[i] = q
		colTypes[col.Name] = col.Type
		resultCols[i] = model.ColumnResult{Name: col.Name, Type: col.Type}
	}

	query := "SELECT " + strings.Join(selectCols, ", ") + " FROM " + quotedTable
	args := []any{}

	if req.Filter != "" {
		filterCols := filterableColumns(columns)
		if len(filterCols) > 0 {
			pattern := "%" + escapeLike(req.Filter) + "%"
			parts := make([]string, len(filterCols))
			for i, name := range filterCols {
				q, err := QuoteIdentifier(name)
				if err != nil {
					return model.TableRowsResponse{}, err
				}
				parts[i] = fmt.Sprintf("CAST(%s AS TEXT) LIKE ? ESCAPE '\\'", q)
				args = append(args, pattern)
			}
			query += " WHERE (" + strings.Join(parts, " OR ") + ")"
		}
	}

	if req.SortColumn != "" {
		sortQ, err := QuoteIdentifier(req.SortColumn)
		if err != nil {
			return model.TableRowsResponse{}, err
		}
		query += " ORDER BY " + sortQ
		if req.SortDesc {
			query += " DESC"
		} else {
			query += " ASC"
		}
	}

	offset := (page - 1) * pageSize
	query += " LIMIT ? OFFSET ?"
	args = append(args, pageSize, offset)

	rows, err := d.sql.QueryContext(ctx, query, args...)
	if err != nil {
		return model.TableRowsResponse{}, fmt.Errorf("query table rows: %w", err)
	}
	defer rows.Close()

	var result [][]model.CellValue
	for rows.Next() {
		dest := make([]any, len(columns))
		ptrs := make([]any, len(columns))
		for i := range dest {
			ptrs[i] = &dest[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return model.TableRowsResponse{}, err
		}
		row := make([]model.CellValue, len(columns))
		for i, col := range columns {
			row[i] = coerceValue(dest[i], colTypes[col.Name])
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return model.TableRowsResponse{}, err
	}

	resp := model.TableRowsResponse{
		Columns:    resultCols,
		Rows:       result,
		Page:       page,
		PageSize:   pageSize,
		DurationMs: time.Since(start).Milliseconds(),
	}

	if req.WithTotal {
		total, err := d.cachedRowCount(ctx, req.Table, req.Filter)
		if err != nil {
			return model.TableRowsResponse{}, err
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

func filterableColumns(cols []model.ColumnInfo) []string {
	var names []string
	for _, c := range cols {
		names = append(names, c.Name)
	}
	return names
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

func (d *DB) cachedRowCount(ctx context.Context, table, filter string) (int64, error) {
	key := countCacheKey{table: table, filter: filter}
	d.countMu.RLock()
	if n, ok := d.rowCountCache[key]; ok {
		d.countMu.RUnlock()
		return n, nil
	}
	d.countMu.RUnlock()

	n, err := d.countRows(ctx, table, filter)
	if err != nil {
		return 0, err
	}
	d.countMu.Lock()
	d.rowCountCache[key] = n
	d.countMu.Unlock()
	return n, nil
}

func (d *DB) countRows(ctx context.Context, table, filter string) (int64, error) {
	quotedTable, err := QuoteIdentifier(table)
	if err != nil {
		return 0, err
	}
	query := "SELECT COUNT(*) FROM " + quotedTable
	args := []any{}

	if filter != "" {
		columns, err := d.loadColumns(ctx, table)
		if err != nil {
			return 0, err
		}
		filterCols := filterableColumns(columns)
		if len(filterCols) > 0 {
			pattern := "%" + escapeLike(filter) + "%"
			parts := make([]string, len(filterCols))
			for i, name := range filterCols {
				q, err := QuoteIdentifier(name)
				if err != nil {
					return 0, err
				}
				parts[i] = fmt.Sprintf("CAST(%s AS TEXT) LIKE ? ESCAPE '\\'", q)
				args = append(args, pattern)
			}
			query += " WHERE (" + strings.Join(parts, " OR ") + ")"
		}
	}

	var count int64
	if err := d.sql.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func coerceValue(raw any, declaredType string) model.CellValue {
	if raw == nil {
		return model.CellValue{Kind: "null", Value: nil}
	}

	switch v := raw.(type) {
	case int64:
		return coerceInt(v)
	case float64:
		return coerceReal(v)
	case string:
		return model.CellValue{Kind: "text", Value: v}
	case []byte:
		if isBlob(declaredType, v) {
			return coerceBlob(v)
		}
		return model.CellValue{Kind: "text", Value: string(v)}
	default:
		return model.CellValue{Kind: "text", Value: fmt.Sprint(v)}
	}
}

func isBlob(declaredType string, b []byte) bool {
	upper := strings.ToUpper(declaredType)
	if strings.Contains(upper, "BLOB") {
		return true
	}
	// Heuristic: invalid UTF-8 often indicates binary data.
	return !isValidUTF8(b)
}

func isValidUTF8(b []byte) bool {
	return utf8.Valid(b)
}

func coerceInt(v int64) model.CellValue {
	if v > jsMaxSafeInteger || v < -jsMaxSafeInteger {
		return model.CellValue{Kind: "int", Value: strconv.FormatInt(v, 10)}
	}
	return model.CellValue{Kind: "int", Value: v}
}

func coerceReal(v float64) model.CellValue {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return model.CellValue{Kind: "null", Value: nil}
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

// FormatCellDisplay returns a human-readable cell string for the UI.
func FormatCellDisplay(cell model.CellValue) string {
	switch cell.Kind {
	case "null":
		return "NULL"
	case "blob":
		if bv, ok := cell.Value.(model.BlobValue); ok {
			return fmt.Sprintf("<BLOB %d bytes>", bv.Size)
		}
		if m, ok := cell.Value.(map[string]any); ok {
			if size, ok := m["size"].(float64); ok {
				return fmt.Sprintf("<BLOB %d bytes>", int(size))
			}
		}
		return "<BLOB>"
	case "text":
		if s, ok := cell.Value.(string); ok {
			return s
		}
	case "int", "real":
		return fmt.Sprint(cell.Value)
	}
	return fmt.Sprint(cell.Value)
}
