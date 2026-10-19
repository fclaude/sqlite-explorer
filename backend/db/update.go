package db

import (
	"context"
	"database/sql"
	"encoding/hex"
	"fmt"
	"math"
	"strconv"
	"strings"

	"sqlite-explorer/backend/apperrors"
	"sqlite-explorer/backend/model"
)

// EncodingHex marks a ColumnUpdate whose text is BLOB bytes written as hex digits.
const EncodingHex = "hex"

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
	// SQLite matches column names case-insensitively (ASCII only).
	colByName := make(map[string]model.ColumnInfo, len(columns))
	for _, c := range columns {
		colByName[asciiLower(c.Name)] = c
	}

	quotedTable, err := QuoteIdentifier(req.Table)
	if err != nil {
		return model.UpdateTableRowResponse{}, err
	}

	setParts := make([]string, 0, len(req.Updates))
	args := make([]any, 0, len(req.Updates)+1)
	updatedColumns := make(map[string]bool, len(req.Updates))
	for _, upd := range req.Updates {
		columnKey := asciiLower(upd.Column)
		if updatedColumns[columnKey] {
			return model.UpdateTableRowResponse{}, apperrors.New(
				apperrors.CodeInvalidColumn,
				"Each column can only be updated once.",
				upd.Column,
			)
		}
		updatedColumns[columnKey] = true
		col, ok := colByName[columnKey]
		if !ok {
			return model.UpdateTableRowResponse{}, apperrors.New(
				apperrors.CodeInvalidColumn,
				"Column does not exist on this table.",
				upd.Column,
			)
		}
		qCol, err := QuoteIdentifier(col.Name)
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
	returningCols, resultCols, err := updateReturningColumns(columns)
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
	cells, err := scanUpdatedRow(rows, len(columns))
	if err != nil {
		return model.UpdateTableRowResponse{}, mapUpdateError(err)
	}

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
		columnNames[asciiLower(column.Name)] = true
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

type affinity int

const (
	affinityInteger affinity = iota
	affinityText
	affinityBlob
	affinityReal
	affinityNumeric
)

// columnAffinity applies SQLite's rules for deriving type affinity from a declared type, in
// the documented order: https://www.sqlite.org/datatype3.html#determination_of_column_affinity
func columnAffinity(declared string) affinity {
	t := strings.ToUpper(declared)
	switch {
	case strings.Contains(t, "INT"):
		return affinityInteger
	case strings.Contains(t, "CHAR"), strings.Contains(t, "CLOB"), strings.Contains(t, "TEXT"):
		return affinityText
	case strings.Contains(t, "BLOB"), strings.TrimSpace(t) == "":
		return affinityBlob
	case strings.Contains(t, "REAL"), strings.Contains(t, "FLOA"), strings.Contains(t, "DOUB"):
		return affinityReal
	default:
		return affinityNumeric
	}
}

// parseColumnUpdate converts an edited field into a value to bind. Hex-encoded BLOBs are
// decoded; everything else is text that SQLite converts using the column's affinity, except
// that INTEGER and REAL columns require a number so a typo cannot silently store text.
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

	switch upd.Encoding {
	case "":
	case EncodingHex:
		return decodeHexBlob(upd.Text)
	default:
		return nil, apperrors.New(apperrors.CodeMalformedSQL, "Unknown value encoding.", upd.Encoding)
	}

	switch columnAffinity(col.Type) {
	case affinityInteger, affinityReal:
		return parseNumber(upd.Text, col.Name)
	default:
		return upd.Text, nil
	}
}

// parseNumber accepts decimal integer and real literals, like SQLite's numeric conversion.
func parseNumber(text, column string) (any, error) {
	trimmed := strings.TrimSpace(text)
	if n, err := strconv.ParseInt(trimmed, 10, 64); err == nil {
		return n, nil
	}
	invalid := apperrors.New(apperrors.CodeMalformedSQL, fmt.Sprintf("Invalid number for %q.", column), text)
	// strconv also accepts hex floats, "Inf", and "NaN", which SQLite does not treat as numbers.
	for _, c := range trimmed {
		if !strings.ContainsRune("0123456789+-.eE", c) {
			return nil, invalid
		}
	}
	f, err := strconv.ParseFloat(trimmed, 64)
	if err != nil || math.IsInf(f, 0) {
		return nil, invalid
	}
	return f, nil
}

// decodeHexBlob decodes hex digits with an optional 0x prefix; ASCII whitespace is ignored.
func decodeHexBlob(text string) ([]byte, error) {
	digits := strings.Map(func(r rune) rune {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			return -1
		}
		return r
	}, text)
	if len(digits) >= 2 && digits[0] == '0' && (digits[1] == 'x' || digits[1] == 'X') {
		digits = digits[2:]
	}
	b, err := hex.DecodeString(digits)
	if err != nil {
		return nil, apperrors.New(apperrors.CodeMalformedSQL, "Invalid BLOB hex value.", err.Error())
	}
	return b, nil
}

func updateReturningColumns(columns []model.ColumnInfo) ([]string, []model.ColumnResult, error) {
	selectCols := make([]string, len(columns))
	resultCols := make([]model.ColumnResult, len(columns))
	for i, col := range columns {
		expr, err := selectColumn(col.Name)
		if err != nil {
			return nil, nil, err
		}
		selectCols[i] = expr
		resultCols[i] = model.ColumnResult{Name: col.Name, Type: col.Type}
	}
	return selectCols, resultCols, nil
}

func scanUpdatedRow(rows *sql.Rows, n int) ([]model.CellValue, error) {
	values, err := scanValues(rows, n)
	if err != nil {
		return nil, err
	}
	cells := make([]model.CellValue, n)
	for i, v := range values {
		cells[i] = coerceValue(v)
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
	return apperrors.New(apperrors.CodeMalformedSQL, "The row could not be updated.", err.Error())
}
