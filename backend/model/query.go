package model

// TableRowsRequest selects a page of rows from a table or view.
type TableRowsRequest struct {
	Table      string `json:"table"`
	PageSize   int    `json:"pageSize"`
	Page       int    `json:"page"`
	SortColumn string `json:"sortColumn"`
	SortDesc   bool   `json:"sortDesc"`
	Filter     string `json:"filter"`
	WithTotal  bool   `json:"withTotal"`
}

// ColumnResult is a result column name for grids.
type ColumnResult struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// CellValue is one cell in a result grid.
type CellValue struct {
	Kind  string `json:"kind"`
	Value any    `json:"value"`
}

// BlobValue is the JSON shape for BLOB cells.
type BlobValue struct {
	Hex  string `json:"hex"`
	Size int    `json:"size"`
}

// TableRowsResponse is a paginated page of table rows.
type TableRowsResponse struct {
	Columns    []ColumnResult `json:"columns"`
	Rows       [][]CellValue  `json:"rows"`
	RowIDs     []string       `json:"rowIds,omitempty"`
	Editable   bool           `json:"editable"`
	Page       int            `json:"page"`
	PageSize   int            `json:"pageSize"`
	TotalRows  *int64         `json:"totalRows,omitempty"`
	DurationMs int64          `json:"durationMs"`
}

// QueryRequest runs arbitrary read-only SQL.
type QueryRequest struct {
	SQL string `json:"sql"`
}

// QueryResponse holds arbitrary query results.
type QueryResponse struct {
	Columns    []ColumnResult `json:"columns"`
	Rows       [][]CellValue  `json:"rows"`
	RowCount   int            `json:"rowCount"`
	Truncated  bool           `json:"truncated"`
	DurationMs int64          `json:"durationMs"`
	QueryID    int64          `json:"queryId"`
}
