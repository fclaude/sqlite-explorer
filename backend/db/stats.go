package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"sqlite-explorer/backend/apperrors"
	"sqlite-explorer/backend/model"
)

const statsQueryTimeout = 30 * time.Second

// GetObjectStats returns row counts, column counts, and storage metrics for a table or view.
func (d *DB) GetObjectStats(ctx context.Context, name string) (model.ObjectStats, error) {
	start := time.Now()
	ctx, cancel := context.WithTimeout(ctx, statsQueryTimeout)
	defer cancel()

	if err := d.validateTableOrView(ctx, name); err != nil {
		return model.ObjectStats{}, err
	}

	kind, sqlDDL, err := d.objectKindAndSQL(ctx, name)
	if err != nil {
		return model.ObjectStats{}, err
	}

	columns, err := d.loadColumns(ctx, name)
	if err != nil {
		return model.ObjectStats{}, err
	}

	rowCount, err := d.countRows(ctx, name, "")
	if err != nil {
		return model.ObjectStats{}, err
	}

	indexCount, fkCount, err := d.tableIndexAndFKCounts(ctx, name, kind)
	if err != nil {
		return model.ObjectStats{}, err
	}

	triggerCount, err := d.triggerCountForTable(ctx, name)
	if err != nil {
		return model.ObjectStats{}, err
	}

	pkCols := primaryKeyColumnNames(columns)
	withoutRowID := kind == "table" && strings.Contains(strings.ToUpper(sqlDDL), "WITHOUT ROWID")

	fileBytes, pageCount, pageSize, usedBytes, err := d.databaseSpaceStats(ctx)
	if err != nil {
		return model.ObjectStats{}, err
	}

	stats := model.ObjectStats{
		Name:              name,
		Kind:              kind,
		SQL:               sqlDDL,
		RowCount:          rowCount,
		ColumnCount:       len(columns),
		IndexCount:        indexCount,
		ForeignKeyCount:   fkCount,
		TriggerCount:      triggerCount,
		PrimaryKeyColumns: pkCols,
		WithoutRowID:      withoutRowID,
		DatabaseFileBytes: fileBytes,
		DatabasePageCount: pageCount,
		DatabasePageSize:  pageSize,
		DatabaseUsedBytes: usedBytes,
		DurationMs:        time.Since(start).Milliseconds(),
	}

	if est := d.estimatedRowCount(ctx, name); est != nil {
		stats.EstimatedRowCount = est
	}
	if storage := d.objectStorageBytes(ctx, name); storage != nil {
		stats.StorageBytes = storage
	}

	return stats, nil
}

func (d *DB) objectKindAndSQL(ctx context.Context, name string) (kind, sqlDDL string, err error) {
	const q = `SELECT type, COALESCE(sql, '') FROM sqlite_schema WHERE name = ? AND name NOT LIKE 'sqlite_%'`
	err = d.sql.QueryRowContext(ctx, q, name).Scan(&kind, &sqlDDL)
	if err == sql.ErrNoRows {
		return "", "", apperrors.New(apperrors.CodeUnknownTable, "Table or view not found.", name)
	}
	return kind, sqlDDL, err
}

func primaryKeyColumnNames(columns []model.ColumnInfo) []string {
	var pk []string
	for _, c := range columns {
		if c.PrimaryKey > 0 {
			pk = append(pk, c.Name)
		}
	}
	return pk
}

func (d *DB) tableIndexAndFKCounts(ctx context.Context, name, kind string) (indexCount, fkCount int, err error) {
	if kind == "table" {
		fks, err := d.loadForeignKeys(ctx, name)
		if err != nil {
			return 0, 0, err
		}
		fkCount = len(fks)
		indexes, err := d.loadTableIndexes(ctx, name)
		if err != nil {
			return 0, 0, err
		}
		return len(indexes), fkCount, nil
	}
	// Views: no attached indexes in PRAGMA sense; COUNT from schema if any
	const q = `SELECT COUNT(*) FROM sqlite_schema WHERE type = 'index' AND tbl_name = ? AND name NOT LIKE 'sqlite_%'`
	var n int
	if err := d.sql.QueryRowContext(ctx, q, name).Scan(&n); err != nil {
		return 0, 0, err
	}
	return n, 0, nil
}

func (d *DB) triggerCountForTable(ctx context.Context, table string) (int, error) {
	const q = `SELECT COUNT(*) FROM sqlite_schema WHERE type = 'trigger' AND tbl_name = ?`
	var n int
	err := d.sql.QueryRowContext(ctx, q, table).Scan(&n)
	return n, err
}

func (d *DB) databaseSpaceStats(ctx context.Context) (fileBytes, pageCount, pageSize, usedBytes int64, err error) {
	info, err := os.Stat(d.path)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("stat database file: %w", err)
	}
	fileBytes = info.Size()

	if err := d.sql.QueryRowContext(ctx, `PRAGMA page_count`).Scan(&pageCount); err != nil {
		return 0, 0, 0, 0, fmt.Errorf("pragma page_count: %w", err)
	}
	if err := d.sql.QueryRowContext(ctx, `PRAGMA page_size`).Scan(&pageSize); err != nil {
		return 0, 0, 0, 0, fmt.Errorf("pragma page_size: %w", err)
	}
	usedBytes = pageCount * pageSize
	return fileBytes, pageCount, pageSize, usedBytes, nil
}

func (d *DB) estimatedRowCount(ctx context.Context, table string) *int64 {
	var stat sql.NullString
	err := d.sql.QueryRowContext(ctx,
		`SELECT stat FROM sqlite_stat1 WHERE tbl = ? ORDER BY idx LIMIT 1`,
		table,
	).Scan(&stat)
	if err != nil || !stat.Valid || stat.String == "" {
		return nil
	}
	parts := strings.Fields(stat.String)
	if len(parts) == 0 {
		return nil
	}
	n, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return nil
	}
	return &n
}

func (d *DB) objectStorageBytes(ctx context.Context, name string) *int64 {
	var total sql.NullInt64
	err := d.sql.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(pgsize), 0) FROM dbstat WHERE name = ?`,
		name,
	).Scan(&total)
	if err != nil || !total.Valid {
		return nil
	}
	v := total.Int64
	return &v
}
