package model

// ExportRequest exports table page or query results to a CSV file.
type ExportRequest struct {
	Source    string           `json:"source"` // "tablePage" | "queryResult"
	Path      string           `json:"path"`
	TableRows TableRowsRequest `json:"tableRows"`
	SQL       string           `json:"sql"`
}
