package model

// ObjectStats summarizes size and shape metrics for a table or view.
type ObjectStats struct {
	Name              string   `json:"name"`
	Kind              string   `json:"kind"`
	SQL               string   `json:"sql,omitempty"`
	RowCount          int64    `json:"rowCount"`
	ColumnCount       int      `json:"columnCount"`
	IndexCount        int      `json:"indexCount"`
	ForeignKeyCount   int      `json:"foreignKeyCount"`
	TriggerCount      int      `json:"triggerCount"`
	PrimaryKeyColumns []string `json:"primaryKeyColumns"`
	WithoutRowID      bool     `json:"withoutRowId"`
	EstimatedRowCount *int64   `json:"estimatedRowCount,omitempty"`
	StorageBytes      *int64   `json:"storageBytes,omitempty"`
	DatabaseFileBytes int64    `json:"databaseFileBytes"`
	DatabasePageCount int64    `json:"databasePageCount"`
	DatabasePageSize  int64    `json:"databasePageSize"`
	DatabaseUsedBytes int64    `json:"databaseUsedBytes"`
	DurationMs        int64    `json:"durationMs"`
}
