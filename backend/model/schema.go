package model

// SchemaInfo is the full database schema for the explorer UI.
type SchemaInfo struct {
	Tables   []TableInfo   `json:"tables"`
	Views    []ViewInfo    `json:"views"`
	Indexes  []IndexInfo   `json:"indexes"`
	Triggers []TriggerInfo `json:"triggers"`
}

// TableInfo describes a base table and its column/FK/index metadata.
type TableInfo struct {
	Name        string          `json:"name"`
	SQL         string          `json:"sql"`
	Columns     []ColumnInfo    `json:"columns"`
	ForeignKeys []ForeignKeyInfo `json:"foreignKeys"`
	Indexes     []TableIndexRef `json:"indexes"`
}

// ViewInfo describes a view and its columns.
type ViewInfo struct {
	Name    string       `json:"name"`
	SQL     string       `json:"sql"`
	Columns []ColumnInfo `json:"columns"`
}

// IndexInfo describes an index from sqlite_schema.
type IndexInfo struct {
	Name    string            `json:"name"`
	Table   string            `json:"table"`
	Unique  bool              `json:"unique"`
	SQL     string            `json:"sql"`
	Columns []IndexColumnInfo `json:"columns"`
}

// TriggerInfo describes a trigger from sqlite_schema.
type TriggerInfo struct {
	Name  string `json:"name"`
	Table string `json:"table"`
	SQL   string `json:"sql"`
}

// ColumnInfo is one column from PRAGMA table_info.
type ColumnInfo struct {
	Name         string  `json:"name"`
	Type         string  `json:"type"`
	NotNull      bool    `json:"notNull"`
	DefaultValue *string `json:"defaultValue"`
	PrimaryKey   int     `json:"primaryKey"`
}

// ForeignKeyInfo is one row from PRAGMA foreign_key_list.
type ForeignKeyInfo struct {
	ID       int    `json:"id"`
	Seq      int    `json:"seq"`
	From     string `json:"from"`
	To       string `json:"to"`
	Table    string `json:"table"`
	OnUpdate string `json:"onUpdate"`
	OnDelete string `json:"onDelete"`
	Match    string `json:"match"`
}

// TableIndexRef is an index attached to a table (from PRAGMA index_list).
type TableIndexRef struct {
	Name    string            `json:"name"`
	Unique  bool              `json:"unique"`
	Origin  string            `json:"origin"`
	Partial bool              `json:"partial"`
	Columns []IndexColumnInfo `json:"columns"`
}

// IndexColumnInfo is one column in an index (from PRAGMA index_info).
type IndexColumnInfo struct {
	Name    string `json:"name"`
	CollSeq string `json:"collSeq,omitempty"`
}
