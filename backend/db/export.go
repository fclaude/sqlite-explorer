package db

import (
	"context"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"os"

	"sqlite-explorer/backend/model"
)

const csvFlushEvery = 1000

const (
	// ExportSourceTablePage exports the current table browser page.
	ExportSourceTablePage = "tablePage"
	// ExportSourceQueryResult re-runs a read-only query for export.
	ExportSourceQueryResult = "queryResult"
)

// ExportRowsToCSV writes rows to path according to the export request.
func (d *DB) ExportRowsToCSV(ctx context.Context, path string, req model.ExportRequest) error {
	if path == "" {
		return fmt.Errorf("export path is empty")
	}

	var columns []model.ColumnResult
	var rows [][]model.CellValue

	switch req.Source {
	case ExportSourceTablePage:
		resp, err := d.GetTableRows(ctx, req.TableRows)
		if err != nil {
			return err
		}
		columns = resp.Columns
		rows = resp.Rows
	case ExportSourceQueryResult:
		if err := ValidateReadOnlySQL(req.SQL); err != nil {
			return err
		}
		resp, err := d.RunQuery(ctx, req.SQL)
		if err != nil {
			return err
		}
		columns = resp.Columns
		rows = resp.Rows
	default:
		return fmt.Errorf("unknown export source: %q", req.Source)
	}

	if err := writeCSVFile(path, columns, rows); err != nil {
		_ = os.Remove(path)
		return err
	}
	return nil
}

func writeCSVFile(path string, columns []model.ColumnResult, rows [][]model.CellValue) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create csv file: %w", err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	names := make([]string, len(columns))
	for i, c := range columns {
		names[i] = c.Name
	}
	if err := w.Write(names); err != nil {
		return err
	}

	for i, row := range rows {
		record := make([]string, len(columns))
		for j := range columns {
			var cell model.CellValue
			if j < len(row) {
				cell = row[j]
			} else {
				cell = model.CellValue{Kind: "null", Value: nil}
			}
			record[j] = cellToCSVField(cell)
		}
		if err := w.Write(record); err != nil {
			return err
		}
		if (i+1)%csvFlushEvery == 0 {
			w.Flush()
			if err := w.Error(); err != nil {
				return err
			}
		}
	}

	w.Flush()
	return w.Error()
}

func cellToCSVField(cell model.CellValue) string {
	switch cell.Kind {
	case "null":
		return ""
	case "text":
		if s, ok := cell.Value.(string); ok {
			return s
		}
	case "int", "real":
		return fmt.Sprint(cell.Value)
	case "blob":
		return blobToCSVField(cell)
	}
	return fmt.Sprint(cell.Value)
}

func blobToCSVField(cell model.CellValue) string {
	bv, ok := cell.Value.(model.BlobValue)
	if !ok {
		if m, ok := cell.Value.(map[string]any); ok {
			return blobMapToCSV(m)
		}
		return "0x"
	}
	out := "0x" + bv.Hex
	if bv.Size > maxBlobPreview {
		out += fmt.Sprintf("...(%d bytes)", bv.Size)
	}
	return out
}

func blobMapToCSV(m map[string]any) string {
	hexStr, _ := m["hex"].(string)
	size := 0
	switch v := m["size"].(type) {
	case float64:
		size = int(v)
	case int:
		size = v
	}
	out := "0x" + hexStr
	if size > maxBlobPreview {
		out += fmt.Sprintf("...(%d bytes)", size)
	}
	return out
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

// FormatBlobCSV formats a blob for expected CSV output in tests.
func FormatBlobCSV(b []byte) string {
	n := len(b)
	preview := n
	if preview > maxBlobPreview {
		preview = maxBlobPreview
	}
	out := "0x" + hex.EncodeToString(b[:preview])
	if n > maxBlobPreview {
		out += fmt.Sprintf("...(%d bytes)", n)
	}
	return out
}
