package model

// ExportRequest exports table rows or query results to a CSV file.
type ExportRequest struct {
	Source    string           `json:"source"` // "tablePage" | "table" | "queryResult"
	TableRows TableRowsRequest `json:"tableRows"`
	SQL       string           `json:"sql"`
}

// ExportResult reports a finished export. Path is empty when the user cancelled the save dialog.
type ExportResult struct {
	Path     string `json:"path"`
	RowCount int64  `json:"rowCount"`
}
