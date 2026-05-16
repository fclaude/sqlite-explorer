package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"sqlite-explorer/backend/model"
)

const schemaQueryTimeout = 5 * time.Second

// GetSchema loads tables, views, indexes, and triggers with column/FK/index metadata.
// Row counts are not computed; use GetTableRowCount on demand.
func (d *DB) GetSchema(ctx context.Context) (model.SchemaInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, schemaQueryTimeout)
	defer cancel()

	objects, err := d.loadSchemaObjects(ctx)
	if err != nil {
		return model.SchemaInfo{}, err
	}

	info := model.SchemaInfo{
		Tables:   make([]model.TableInfo, 0),
		Views:    make([]model.ViewInfo, 0),
		Indexes:  make([]model.IndexInfo, 0),
		Triggers: make([]model.TriggerInfo, 0),
	}

	for _, obj := range objects {
		switch obj.Type {
		case "table":
			t, err := d.loadTable(ctx, obj)
			if err != nil {
				return model.SchemaInfo{}, err
			}
			info.Tables = append(info.Tables, t)
		case "view":
			v, err := d.loadView(ctx, obj)
			if err != nil {
				return model.SchemaInfo{}, err
			}
			info.Views = append(info.Views, v)
		case "index":
			// User-created indexes only; PK/UNIQUE auto-indexes appear on TableInfo.Indexes.
			if strings.HasPrefix(obj.Name, "sqlite_autoindex_") {
				continue
			}
			idx, err := d.loadIndex(ctx, obj)
			if err != nil {
				return model.SchemaInfo{}, err
			}
			info.Indexes = append(info.Indexes, idx)
		case "trigger":
			info.Triggers = append(info.Triggers, model.TriggerInfo{
				Name:  obj.Name,
				Table: obj.Table,
				SQL:   obj.SQL,
			})
		}
	}

	return info, nil
}

// GetTableRowCount returns an exact row count for a table or view (lazy, on demand).
func (d *DB) GetTableRowCount(ctx context.Context, table string) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, schemaQueryTimeout)
	defer cancel()

	quoted, err := QuoteIdentifier(table)
	if err != nil {
		return 0, err
	}
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", quoted)
	var count int64
	if err := d.sql.QueryRowContext(ctx, query).Scan(&count); err != nil {
		return 0, fmt.Errorf("count rows for %s: %w", table, err)
	}
	return count, nil
}

type schemaObject struct {
	Type  string
	Name  string
	Table string
	SQL   string
}

func (d *DB) loadSchemaObjects(ctx context.Context) ([]schemaObject, error) {
	const q = `SELECT type, name, tbl_name, sql FROM sqlite_schema
		WHERE name NOT LIKE 'sqlite_%'
		ORDER BY type, name`

	rows, err := d.sql.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("query sqlite_schema: %w", err)
	}
	defer rows.Close()

	var out []schemaObject
	for rows.Next() {
		var obj schemaObject
		var sqlNull sql.NullString
		if err := rows.Scan(&obj.Type, &obj.Name, &obj.Table, &sqlNull); err != nil {
			return nil, err
		}
		if sqlNull.Valid {
			obj.SQL = sqlNull.String
		}
		out = append(out, obj)
	}
	return out, rows.Err()
}

func (d *DB) loadTable(ctx context.Context, obj schemaObject) (model.TableInfo, error) {
	cols, err := d.loadColumns(ctx, obj.Name)
	if err != nil {
		return model.TableInfo{}, err
	}
	fks, err := d.loadForeignKeys(ctx, obj.Name)
	if err != nil {
		return model.TableInfo{}, err
	}
	idxs, err := d.loadTableIndexes(ctx, obj.Name)
	if err != nil {
		return model.TableInfo{}, err
	}
	return model.TableInfo{
		Name:        obj.Name,
		SQL:         obj.SQL,
		Columns:     cols,
		ForeignKeys: fks,
		Indexes:     idxs,
	}, nil
}

func (d *DB) loadView(ctx context.Context, obj schemaObject) (model.ViewInfo, error) {
	cols, err := d.loadColumns(ctx, obj.Name)
	if err != nil {
		return model.ViewInfo{}, err
	}
	return model.ViewInfo{
		Name:    obj.Name,
		SQL:     obj.SQL,
		Columns: cols,
	}, nil
}

func (d *DB) loadIndex(ctx context.Context, obj schemaObject) (model.IndexInfo, error) {
	cols, unique, err := d.loadIndexColumns(ctx, obj.Name, obj.Table)
	if err != nil {
		return model.IndexInfo{}, err
	}
	return model.IndexInfo{
		Name:    obj.Name,
		Table:   obj.Table,
		Unique:  unique,
		SQL:     obj.SQL,
		Columns: cols,
	}, nil
}

func (d *DB) loadColumns(ctx context.Context, table string) ([]model.ColumnInfo, error) {
	quoted, err := QuoteIdentifier(table)
	if err != nil {
		return nil, err
	}
	q := fmt.Sprintf("PRAGMA table_info(%s)", quoted)
	rows, err := d.sql.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("pragma table_info(%s): %w", table, err)
	}
	defer rows.Close()

	var cols []model.ColumnInfo
	for rows.Next() {
		var cid int
		var name, colType string
		var notNull int
		var dflt sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dflt, &pk); err != nil {
			return nil, err
		}
		col := model.ColumnInfo{
			Name:       name,
			Type:       colType,
			NotNull:    notNull != 0,
			PrimaryKey: pk,
		}
		if dflt.Valid {
			s := dflt.String
			col.DefaultValue = &s
		}
		cols = append(cols, col)
	}
	return cols, rows.Err()
}

func (d *DB) loadForeignKeys(ctx context.Context, table string) ([]model.ForeignKeyInfo, error) {
	quoted, err := QuoteIdentifier(table)
	if err != nil {
		return nil, err
	}
	q := fmt.Sprintf("PRAGMA foreign_key_list(%s)", quoted)
	rows, err := d.sql.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("pragma foreign_key_list(%s): %w", table, err)
	}
	defer rows.Close()

	var fks []model.ForeignKeyInfo
	for rows.Next() {
		var fk model.ForeignKeyInfo
		if err := rows.Scan(&fk.ID, &fk.Seq, &fk.Table, &fk.From, &fk.To, &fk.OnUpdate, &fk.OnDelete, &fk.Match); err != nil {
			return nil, err
		}
		fks = append(fks, fk)
	}
	return fks, rows.Err()
}

func (d *DB) loadTableIndexes(ctx context.Context, table string) ([]model.TableIndexRef, error) {
	quoted, err := QuoteIdentifier(table)
	if err != nil {
		return nil, err
	}
	q := fmt.Sprintf("PRAGMA index_list(%s)", quoted)
	rows, err := d.sql.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("pragma index_list(%s): %w", table, err)
	}
	defer rows.Close()

	var idxs []model.TableIndexRef
	for rows.Next() {
		var seq int
		var name, origin string
		var unique, partial int
		if err := rows.Scan(&seq, &name, &unique, &origin, &partial); err != nil {
			return nil, err
		}
		cols, _, err := d.loadIndexColumns(ctx, name, table)
		if err != nil {
			return nil, err
		}
		idxs = append(idxs, model.TableIndexRef{
			Name:    name,
			Unique:  unique != 0,
			Origin:  origin,
			Partial: partial != 0,
			Columns: cols,
		})
	}
	return idxs, rows.Err()
}

func (d *DB) loadIndexColumns(ctx context.Context, indexName, tableName string) ([]model.IndexColumnInfo, bool, error) {
	quoted, err := QuoteIdentifier(indexName)
	if err != nil {
		return nil, false, err
	}

	unique, err := d.indexUnique(ctx, tableName, indexName)
	if err != nil {
		return nil, false, err
	}

	q := fmt.Sprintf("PRAGMA index_info(%s)", quoted)
	rows, err := d.sql.QueryContext(ctx, q)
	if err != nil {
		return nil, false, fmt.Errorf("pragma index_info(%s): %w", indexName, err)
	}
	defer rows.Close()

	var cols []model.IndexColumnInfo
	for rows.Next() {
		var seqno, cid int
		var name sql.NullString
		if err := rows.Scan(&seqno, &cid, &name); err != nil {
			return nil, false, err
		}
		col := model.IndexColumnInfo{}
		if name.Valid {
			col.Name = name.String
		}
		cols = append(cols, col)
	}
	return cols, unique, rows.Err()
}

func (d *DB) indexUnique(ctx context.Context, tableName, indexName string) (bool, error) {
	quoted, err := QuoteIdentifier(tableName)
	if err != nil {
		return false, err
	}
	q := fmt.Sprintf("PRAGMA index_list(%s)", quoted)
	rows, err := d.sql.QueryContext(ctx, q)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var seq int
		var name, origin string
		var unique, partial int
		if err := rows.Scan(&seq, &name, &unique, &origin, &partial); err != nil {
			return false, err
		}
		if name == indexName {
			return unique != 0, nil
		}
	}
	return false, nil
}
