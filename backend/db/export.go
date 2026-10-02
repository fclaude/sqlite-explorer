package db

import (
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"time"

	"sqlite-explorer/backend/apperrors"
	"sqlite-explorer/backend/model"
)

const csvFlushEvery = 1000

const (
	// ExportSourceTablePage exports the current page of the table browser.
	ExportSourceTablePage = "tablePage"
	// ExportSourceTable exports every row of a table or view that matches the filter, in sort order.
	ExportSourceTable = "table"
	// ExportSourceQueryResult runs a read-only query again and exports all of its rows.
	ExportSourceQueryResult = "queryResult"
)

// ValidateExport checks an export request before the user is asked for a file name.
func ValidateExport(req model.ExportRequest) error {
	switch req.Source {
	case ExportSourceTablePage, ExportSourceTable:
		if req.TableRows.Table == "" {
			return apperrors.New(apperrors.CodeUnknownTable, "Table name is required.", "")
		}
		return nil
	case ExportSourceQueryResult:
		_, err := planExportQuery(req.SQL)
		return err
	default:
		return fmt.Errorf("unknown export source: %q", req.Source)
	}
}

// planExportQuery accepts only read statements, because exporting runs the query again.
func planExportQuery(sql string) (Plan, error) {
	plan, err := PlanSQL(sql, ReadOnlyPolicy)
	if appErr, ok := apperrors.As(err); ok && appErr.Code == apperrors.CodeReadOnlyViolation {
		return Plan{}, apperrors.New(
			apperrors.CodeReadOnlyViolation,
			"Only read queries can be exported, because exporting runs the query again.",
			appErr.Detail,
		)
	}
	return plan, err
}

// ExportRowsToCSV writes rows to path according to the export request and returns the
// number of data rows written. Rows are streamed, so exports are not limited to what the
// grid shows. The file is written next to path and renamed into place only on success.
func (d *DB) ExportRowsToCSV(ctx context.Context, path string, req model.ExportRequest) (int64, error) {
	if path == "" {
		return 0, fmt.Errorf("export path is empty")
	}
	if err := ValidateExport(req); err != nil {
		return 0, err
	}

	switch req.Source {
	case ExportSourceTablePage, ExportSourceTable:
		sel, err := d.prepareTableSelect(ctx, req.TableRows, req.Source == ExportSourceTablePage, false)
		if err != nil {
			return 0, err
		}
		header := make([]string, len(sel.resultCols))
		for i, c := range sel.resultCols {
			header[i] = c.Name
		}
		return writeCSVFile(path, func(w *csv.Writer) (int64, error) {
			rows, err := d.querySQL.QueryContext(ctx, sel.query, sel.args...)
			if err != nil {
				return 0, mapQueryError(ctx, err)
			}
			defer rows.Close()
			n, err := writeCSVRows(w, header, rows)
			if err != nil {
				return n, mapQueryError(ctx, err)
			}
			return n, nil
		})
	default:
		plan, err := planExportQuery(req.SQL)
		if err != nil {
			return 0, err
		}
		return writeCSVFile(path, func(w *csv.Writer) (int64, error) {
			var n int64
			_, err := d.execPlan(ctx, plan, func(rows *sql.Rows) error {
				header, err := rows.Columns()
				if err != nil {
					return err
				}
				n, err = writeCSVRows(w, header, rows)
				return err
			})
			return n, err
		})
	}
}

// writeCSVFile creates a temporary file next to path, fills it, and renames it over path.
func writeCSVFile(path string, fill func(*csv.Writer) (int64, error)) (int64, error) {
	tmpPath := path + ".partial-" + strconv.FormatInt(time.Now().UnixNano(), 36)
	f, err := os.OpenFile(tmpPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o666)
	if err != nil {
		return 0, fmt.Errorf("create csv file: %w", err)
	}
	w := csv.NewWriter(f)
	n, err := fill(w)
	if err == nil {
		w.Flush()
		err = w.Error()
	}
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		err = os.Rename(tmpPath, path)
	}
	if err != nil {
		_ = os.Remove(tmpPath)
		return 0, err
	}
	return n, nil
}

func writeCSVRows(w *csv.Writer, header []string, rows *sql.Rows) (int64, error) {
	if err := w.Write(header); err != nil {
		return 0, err
	}
	record := make([]string, len(header))
	var n int64
	for rows.Next() {
		values, err := scanValues(rows, len(header))
		if err != nil {
			return n, err
		}
		for i, v := range values {
			record[i] = csvField(v)
		}
		if err := w.Write(record); err != nil {
			return n, err
		}
		n++
		if n%csvFlushEvery == 0 {
			w.Flush()
			if err := w.Error(); err != nil {
				return n, err
			}
		}
	}
	if err := rows.Err(); err != nil {
		return n, err
	}
	return n, nil
}

// csvField formats a driver value for CSV. NULL is empty and BLOBs are full 0x-prefixed hex.
func csvField(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case int64:
		return strconv.FormatInt(x, 10)
	case float64:
		return strconv.FormatFloat(x, 'g', -1, 64)
	case string:
		return x
	case []byte:
		return "0x" + hex.EncodeToString(x)
	case time.Time:
		return formatSQLiteTime(x)
	default:
		return fmt.Sprint(x)
	}
}

// ReadCSVFile reads a CSV file (for tests).
func ReadCSVFile(path string) ([][]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return csv.NewReader(f).ReadAll()
}
