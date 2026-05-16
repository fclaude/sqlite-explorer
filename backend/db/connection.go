package db

import (
	_ "modernc.org/sqlite" // pure-Go SQLite driver (no CGO)
)

// Driver is the database/sql driver name registered by modernc.org/sqlite.
const Driver = "sqlite"
