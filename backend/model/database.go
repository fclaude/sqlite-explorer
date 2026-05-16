package model

// DatabaseInfo describes an opened SQLite database file.
type DatabaseInfo struct {
	Path      string `json:"path"`
	SizeBytes int64  `json:"sizeBytes"`
	ReadOnly  bool   `json:"readOnly"`
}
