package db

import (
	"math/rand"
	"strings"
	"testing"

	"modernc.org/libc"
	sqlite3 "modernc.org/sqlite/lib"
)

// sqliteComplete asks SQLite's own sqlite3_complete() whether sql ends at a statement boundary.
func sqliteComplete(t testing.TB, tls *libc.TLS, sql string) bool {
	t.Helper()
	cs, err := libc.CString(sql)
	if err != nil {
		t.Fatal(err)
	}
	defer libc.Xfree(tls, cs)
	return sqlite3.Xsqlite3_complete(tls, cs) != 0
}

// splitterComplete reports whether our splitter considers sql to end at a statement boundary.
func splitterComplete(sql string) bool {
	tokens, openComment, err := lex(sql)
	if err != nil || openComment {
		return false
	}
	state := stInvalid
	for _, tok := range tokens {
		state = state.next(tok)
	}
	return state == stStart
}

var splitterCorpus = []string{
	"SELECT 1;",
	"SELECT 1 -- it's\n; VACUUM INTO 'out.db';",
	"SELECT 1 WHERE 0 -- it's\n; VACUUM INTO 'out.db'",
	"SELECT 1 /* it's */ ; DELETE FROM t;",
	"SELECT 'a;b'; SELECT \"c;d\"; SELECT `e;f`; SELECT [g;h];",
	"SELECT 'it''s;' ; SELECT \"say \"\";\"\"\";",
	"SELECT 1; /* unterminated ;",
	"SELECT [unterminated ; DROP TABLE t;",
	"CREATE TRIGGER tr AFTER INSERT ON t BEGIN UPDATE t SET a = CASE WHEN 1 THEN 2 END; DELETE FROM u; END; SELECT 1;",
	"create temp trigger tr after insert on t begin select 1; end ; select 2;",
	"CREATE TEMPORARY TRIGGER tr BEFORE DELETE ON t BEGIN SELECT RAISE(ABORT, 'no;'); END;",
	"EXPLAIN CREATE TRIGGER tr AFTER INSERT ON t BEGIN SELECT 1; END; SELECT 2;",
	"EXPLAIN QUERY PLAN CREATE TRIGGER tr AFTER INSERT ON t BEGIN SELECT 1; END;",
	"EXPLAIN 1.create TRIGGER x; DELETE FROM t;",
	"CREATE TRIGGER x; DELETE FROM t;",
	"CREATE TABLE t(a); CREATE TRIGGER t2 AFTER INSERT ON t BEGIN SELECT 1; END",
	"SELECT 1 as end; END; SELECT :end; SELECT $end; SELECT ?1;",
	"SELECT x'00ff'; SELECT X'3B';",
	"ſELECT 1; SELECT é; SELECT  ;",
	"SELECT 1\f;\r\n\tSELECT 2\v;",
	"WITH a(x) AS NOT MATERIALIZED (SELECT 1), b AS (SELECT 2) SELECT * FROM a, b;",
	"PRAGMA main.table_info(t); PRAGMA user_version = 1;",
	";;;SELECT 1;;",
	"-- only a comment",
	"SELECT 1 -- trailing comment without newline",
}

func TestSplitterMatchesSQLiteComplete(t *testing.T) {
	tls := libc.NewTLS()
	defer tls.Close()

	check := func(sql string) {
		for n := 0; n <= len(sql); n++ {
			prefix := sql[:n]
			want := sqliteComplete(t, tls, prefix)
			if got := splitterComplete(prefix); got != want {
				t.Fatalf("statement boundary mismatch for %q: splitter=%v sqlite3_complete=%v", prefix, got, want)
			}
		}
	}
	for _, sql := range splitterCorpus {
		check(sql)
	}

	fragments := []string{
		"SELECT", "select", "DELETE FROM t", "VACUUM INTO 'x'", "CREATE", "create", "TEMP", "TEMPORARY",
		"TRIGGER", "trigger", "BEGIN", "END", "end", "EXPLAIN", "explain", "QUERY PLAN", "x", "1", "1.",
		"(", ")", ",", ";", ";", " ", "\n", "\t", "\r", "\f", "\v", "'", "''", "\"", "`", "[", "]",
		"--", "/*", "*/", "it's", "ſ", "é", "$a", "?1", ":a", "@a", "#", ".", "-", "/", "*", "0x1", "X'00'",
	}
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 4000; i++ {
		var b strings.Builder
		for j := 0; j < 1+rng.Intn(14); j++ {
			b.WriteString(fragments[rng.Intn(len(fragments))])
		}
		check(b.String())
	}
}

func FuzzSplitterMatchesSQLiteComplete(f *testing.F) {
	for _, sql := range splitterCorpus {
		f.Add(sql)
	}
	tls := libc.NewTLS()
	defer tls.Close()
	f.Fuzz(func(t *testing.T, sql string) {
		if strings.IndexByte(sql, 0) >= 0 {
			t.Skip()
		}
		if got, want := splitterComplete(sql), sqliteComplete(t, tls, sql); got != want {
			t.Fatalf("statement boundary mismatch for %q: splitter=%v sqlite3_complete=%v", sql, got, want)
		}
	})
}

func TestSplitStatements(t *testing.T) {
	tests := []struct {
		sql  string
		want []string
	}{
		{"SELECT 1", []string{"SELECT 1"}},
		{";; SELECT 1 ;; SELECT 2;", []string{"SELECT 1", "SELECT 2"}},
		{"SELECT 'a;b'; SELECT [c;d]; SELECT `e;f`", []string{"SELECT 'a;b'", "SELECT [c;d]", "SELECT `e;f`"}},
		{"SELECT 1 -- it's\n; VACUUM", []string{"SELECT 1", "VACUUM"}},
		{
			"CREATE TRIGGER tr AFTER INSERT ON t BEGIN DELETE FROM u; END; SELECT 1",
			[]string{"CREATE TRIGGER tr AFTER INSERT ON t BEGIN DELETE FROM u; END", "SELECT 1"},
		},
		{"-- comment only\n/* and another */", nil},
	}
	for _, tt := range tests {
		stmts, err := splitStatements(tt.sql)
		if err != nil {
			t.Fatalf("%q: %v", tt.sql, err)
		}
		var got []string
		for _, s := range stmts {
			got = append(got, s.text)
		}
		if strings.Join(got, "|") != strings.Join(tt.want, "|") {
			t.Fatalf("%q: got %q, want %q", tt.sql, got, tt.want)
		}
	}
}

func TestTokenizeRejectsUnterminatedAndNUL(t *testing.T) {
	for _, sql := range []string{"SELECT 'abc", `SELECT "abc`, "SELECT `abc", "SELECT [abc", "SELECT 1\x00; DROP TABLE t"} {
		if _, err := tokenize(sql); err == nil {
			t.Fatalf("%q: expected error", sql)
		}
	}
}
