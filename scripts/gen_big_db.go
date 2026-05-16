//go:build ignore

// Generate a large SQLite database for manual performance testing.
// Usage: go run scripts/gen_big_db.go -rows 1000000 -out /tmp/big.db
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"

	_ "modernc.org/sqlite"
)

func main() {
	rows := flag.Int("rows", 1_000_000, "number of rows to insert")
	out := flag.String("out", "big.db", "output database path")
	flag.Parse()

	_ = os.Remove(*out)
	db, err := sql.Open("sqlite", *out)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if _, err := db.Exec(`CREATE TABLE big (id INTEGER PRIMARY KEY, name TEXT NOT NULL)`); err != nil {
		log.Fatal(err)
	}

	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}
	stmt, err := tx.Prepare(`INSERT INTO big (id, name) VALUES (?, ?)`)
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	batch := 10000
	for i := 1; i <= *rows; i++ {
		name := fmt.Sprintf("row_%d", i)
		if _, err := stmt.Exec(i, name); err != nil {
			log.Fatal(err)
		}
		if i%batch == 0 {
			fmt.Printf("inserted %d rows...\n", i)
		}
	}
	if err := tx.Commit(); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Created %s with %d rows in table big\n", *out, *rows)
}
