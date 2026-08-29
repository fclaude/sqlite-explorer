package db

import (
	"context"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"sqlite-explorer/backend/apperrors"
	"sqlite-explorer/backend/model"
)

// UpdateTableRow applies column updates to a single table row identified by SQLite rowid.
func (d *DB) UpdateTableRow(ctx context.Context, req model.UpdateTableRowRequest) (model.UpdateTableRowResponse, error) {
	if d.readOnly {
		return model.UpdateTableRowResponse{}, apperrors.New(
			apperrors.CodeReadOnlyViolation,
			"The database is open read-only and cannot be modified.",
			"",
		)
	}
	if req.Table == "" {
		return model.UpdateTableRowResponse{}, apperrors.New(apperrors.CodeUnknownTable, "Table name is required.", "")
	}
	rowID, err := strconv.ParseInt(req.RowID, 10, 64)
	if err != nil {
		return model.UpdateTableRowResponse{}, apperrors.New(apperrors.CodeMalformedSQL, "Invalid row id.", "")
	}
	if len(req.Updates) == 0 {
		return model.UpdateTableRowResponse{}, apperrors.New(apperrors.CodeMalformedSQL, "No columns to update.", "")
	}

	ctx, cancel := context.WithTimeout(ctx, tableQueryTimeout)
	defer cancel()

	if !d.isTable(ctx, req.Table) {
		return model.UpdateTableRowResponse{}, apperrors.New(
			apperrors.CodeNotEditable,
			"Only base tables can be edited. Views are read-only.",
			req.Table,
		)
	}

	columns, err := d.loadColumns(ctx, req.Table)
	if err != nil {
		return model.UpdateTableRowResponse{}, err
	}
	rowIDAlias, editable := d.editableRowIDAlias(ctx, req.Table, columns)
	if !editable {
		return model.UpdateTableRowResponse{}, apperrors.New(
			apperrors.CodeNotEditable,
			"This table cannot be edited safely because it has no accessible SQLite rowid.",
			req.Table,
		)
	}
	colByName := make(map[string]model.ColumnInfo, len(columns))
	for _, c := range columns {
		colByName[c.Name] = c
	}

	quotedTable, err := QuoteIdentifier(req.Table)
	if err != nil {
		return model.UpdateTableRowResponse{}, err
	}

	setParts := make([]string, 0, len(req.Updates))
	args := make([]any, 0, len(req.Updates)+1)
	updatedColumns := make(map[string]bool, len(req.Updates))
	for _, upd := range req.Updates {
		columnKey := strings.ToLower(upd.Column)
		if updatedColumns[columnKey] {
			return model.UpdateTableRowResponse{}, apperrors.New(
				apperrors.CodeInvalidColumn,
				"Each column can only be updated once.",
				upd.Column,
			)
		}
		updatedColumns[columnKey] = true
		col, ok := colByName[upd.Column]
		if !ok {
			return model.UpdateTableRowResponse{}, apperrors.New(
				apperrors.CodeInvalidColumn,
				"Column does not exist on this table.",
				upd.Column,
			)
		}
		qCol, err := QuoteIdentifier(upd.Column)
		if err != nil {
			return model.UpdateTableRowResponse{}, err
		}
		val, err := parseColumnUpdate(upd, col)
		if err != nil {
			return model.UpdateTableRowResponse{}, err
		}
		setParts = append(setParts, qCol+" = ?")
		args = append(args, val)
	}

	quotedRowID, err := QuoteIdentifier(rowIDAlias)
	if err != nil {
		return model.UpdateTableRowResponse{}, err
	}
	returningCols, resultCols, colTypes, err := updateReturningColumns(columns)
	if err != nil {
		return model.UpdateTableRowResponse{}, err
	}
	query := "UPDATE " + quotedTable + " SET " + strings.Join(setParts, ", ") +
		" WHERE " + quotedRowID + " = ? RETURNING " + strings.Join(returningCols, ", ")
	args = append(args, rowID)

	rows, err := d.sql.QueryContext(ctx, query, args...)
	if err != nil {
		return model.UpdateTableRowResponse{}, mapUpdateError(err)
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return model.UpdateTableRowResponse{}, mapUpdateError(err)
		}
		return model.UpdateTableRowResponse{}, apperrors.New(
			apperrors.CodeUnknownTable,
			"Row not found. It may have been deleted or the page changed.",
			"rowid="+req.RowID,
		)
	}
	cells, err := scanUpdatedRow(rows, columns, colTypes)
	if err != nil {
		return model.UpdateTableRowResponse{}, mapUpdateError(err)
	}

	d.invalidateRowCountCache(req.Table)

	return model.UpdateTableRowResponse{Columns: resultCols, Cells: cells}, nil
}

func (d *DB) editableRowIDAlias(ctx context.Context, table string, columns []model.ColumnInfo) (string, bool) {
	if d.readOnly {
		return "", false
	}
	var kind string
	var withoutRowID int
	err := d.sql.QueryRowContext(ctx,
		`SELECT type, wr FROM pragma_table_list WHERE schema = 'main' AND name = ?`,
		table,
	).Scan(&kind, &withoutRowID)
	if err != nil || kind != "table" || withoutRowID != 0 {
		return "", false
	}

	columnNames := make(map[string]bool, len(columns))
	for _, column := range columns {
		columnNames[strings.ToLower(column.Name)] = true
	}
	for _, alias := range []string{"rowid", "_rowid_", "oid"} {
		if !columnNames[alias] {
			return alias, true
		}
	}
	return "", false
}

func (d *DB) isTable(ctx context.Context, name string) bool {
	var typ string
	err := d.sql.QueryRowContext(ctx,
		`SELECT type FROM sqlite_schema WHERE name = ? AND name NOT LIKE 'sqlite_%'`,
		name,
	).Scan(&typ)
	return err == nil && typ == "table"
}

func (d *DB) invalidateRowCountCache(table string) {
	d.countMu.Lock()
	for k := range d.rowCountCache {
		if k.table == table {
			delete(d.rowCountCache, k)
		}
	}
	d.countMu.Unlock()
}

func parseColumnUpdate(upd model.ColumnUpdate, col model.ColumnInfo) (any, error) {
	if upd.IsNull {
		if col.NotNull {
			return nil, apperrors.New(
				apperrors.CodeMalformedSQL,
				fmt.Sprintf("Column %q cannot be NULL.", col.Name),
				col.Name,
			)
		}
		return nil, nil
	}

	text := upd.Text
	upperType := strings.ToUpper(col.Type)

	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(text)), "0x") {
		hexStr := strings.TrimSpace(text)[2:]
		if idx := strings.Index(hexStr, "\n\n["); idx >= 0 {
			hexStr = hexStr[:idx]
		}
		hexStr = strings.Map(func(r rune) rune {
			if r == ' ' || r == '\n' || r == '\r' {
				return -1
			}
			return r
		}, hexStr)
		b, err := hex.DecodeString(hexStr)
		if err != nil {
			return nil, apperrors.New(apperrors.CodeMalformedSQL, "Invalid BLOB hex value.", err.Error())
		}
		return b, nil
	}

	if strings.Contains(upperType, "BLOB") {
		return []byte(text), nil
	}
	if strings.Contains(upperType, "INT") {
		n, err := strconv.ParseInt(strings.TrimSpace(text), 10, 64)
		if err != nil {
			return nil, apperrors.New(apperrors.CodeMalformedSQL, fmt.Sprintf("Invalid integer for %q.", col.Name), text)
		}
		return n, nil
	}
	if strings.Contains(upperType, "REAL") || strings.Contains(upperType, "FLOA") || strings.Contains(upperType, "DOUB") {
		f, err := strconv.ParseFloat(strings.TrimSpace(text), 64)
		if err != nil {
			return nil, apperrors.New(apperrors.CodeMalformedSQL, fmt.Sprintf("Invalid number for %q.", col.Name), text)
		}
		return f, nil
	}

	return text, nil
}

func updateReturningColumns(columns []model.ColumnInfo) ([]string, []model.ColumnResult, map[string]string, error) {
	selectCols := make([]string, len(columns))
	resultCols := make([]model.ColumnResult, len(columns))
	colTypes := make(map[string]string, len(columns))
	for i, col := range columns {
		q, err := QuoteIdentifier(col.Name)
		if err != nil {
			return nil, nil, nil, err
		}
		selectCols[i] = q
		colTypes[col.Name] = col.Type
		resultCols[i] = model.ColumnResult{Name: col.Name, Type: col.Type}
	}
	return selectCols, resultCols, colTypes, nil
}

func scanUpdatedRow(rows *sql.Rows, columns []model.ColumnInfo, colTypes map[string]string) ([]model.CellValue, error) {
	dest := make([]any, len(columns))
	ptrs := make([]any, len(columns))
	for i := range dest {
		ptrs[i] = &dest[i]
	}
	if err := rows.Scan(ptrs...); err != nil {
		return nil, err
	}
	cells := make([]model.CellValue, len(columns))
	for i, col := range columns {
		cells[i] = coerceValue(dest[i], colTypes[col.Name])
	}
	return cells, nil
}

func mapUpdateError(err error) error {
	if strings.Contains(strings.ToLower(err.Error()), "readonly") {
		return apperrors.New(
			apperrors.CodeReadOnlyViolation,
			"The database file is read-only.",
			err.Error(),
		)
	}
	return fmt.Errorf("update row: %w", err)
}
