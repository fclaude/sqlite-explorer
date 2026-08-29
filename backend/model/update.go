package model

// ColumnUpdate is one column change in UpdateTableRow.
type ColumnUpdate struct {
	Column string `json:"column"`
	Text   string `json:"text"`
	IsNull bool   `json:"isNull"`
}

// UpdateTableRowRequest updates one row in a base table.
type UpdateTableRowRequest struct {
	Table   string         `json:"table"`
	RowID   string         `json:"rowId"`
	Updates []ColumnUpdate `json:"updates"`
}

// UpdateTableRowResponse returns the row after update.
type UpdateTableRowResponse struct {
	Columns []ColumnResult `json:"columns"`
	Cells   []CellValue    `json:"cells"`
}
