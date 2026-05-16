# SQLite Explorer — Staged Build Plan

You are an expert Go + Wails engineer. Build the SQLite explorer described below in nine ordered stages. Do not advance to the next stage until the current stage's *Definition of done* is satisfied.

## 1. Overview

Build a polished, practical, **local-only** desktop SQLite explorer using **Wails + Go + React/TypeScript**. The app opens a `.sqlite`/`.sqlite3`/`.db` file, browses schema (tables, views, indexes, triggers), paginates table data, runs read-only SQL, and exports results to CSV. The DB is opened **read-only by default** at both the connection layer and a statement-validator layer. Prefer **`modernc.org/sqlite`** (pure-Go, no CGO). Prioritize correctness, responsiveness on large DBs, and clean architecture over flashy UI.

## 2. Tech stack and non-negotiables

- Wails v2 desktop shell.
- Go backend; React + TypeScript frontend (Vite template).
- SQLite driver: `modernc.org/sqlite` (pure Go). Do not introduce CGO unless a stage explicitly requires it.
- Read-only by default: enforced **both** by DSN (`mode=ro&immutable=1`) and by an allow-list statement validator.
- No external/hosted services. No telemetry. No network calls.
- All identifiers (table, column, index, trigger names) MUST be passed through a quoting helper before SQL interpolation.
- All table browsing MUST be paginated with `LIMIT`/`OFFSET`; never load full tables into memory.
- All long-running queries MUST honor a `context.Context` with a timeout.
- Arbitrary query results MUST be capped at a configurable max (default 1000 rows).
- No emojis in UI or code.

## 3. Repository layout

```
/backend
  app.go
  db/
    connection.go
    schema.go
    query.go
    export.go
    identifier.go
    validator.go
  model/
    schema.go
    query.go
/frontend
  src/
    App.tsx
    components/
      Sidebar.tsx
      DataGrid.tsx
      SqlEditor.tsx
      SchemaView.tsx
      StatusBar.tsx
/testdata
  sample.sqlite           # generated; see Appendix
  fixtures.sql            # source of truth for sample.sqlite
README.md
```

Exact filenames may be adjusted, but responsibilities must stay separated as above.

## 4. Backend API surface (Wails-bound)

All methods are bound on the `App` struct and exposed to the frontend. All use typed request/response structs in `backend/model`.

```go
OpenDatabase()                        (DatabaseInfo, error)
CloseDatabase()                       error
GetSchema()                           (SchemaInfo, error)
GetTableRows(req TableRowsRequest)    (TableRowsResponse, error)
RunQuery(req QueryRequest)            (QueryResponse, error)
ExportRowsToCSV(req ExportRequest)    error
```

Suggested shapes (final names may vary, but order and types of fields must hold):

```go
type DatabaseInfo struct {
    Path      string `json:"path"`
    SizeBytes int64  `json:"sizeBytes"`
    ReadOnly  bool   `json:"readOnly"`
}

type ColumnResult struct {
    Name string `json:"name"`
    Type string `json:"type"` // declared type, or "" if dynamic
}

type CellValue struct {
    Kind  string `json:"kind"`  // "null" | "int" | "real" | "text" | "blob"
    Value any    `json:"value"` // null | number | string | { hex: string, size: int }
}

type QueryResponse struct {
    Columns    []ColumnResult `json:"columns"`
    Rows       [][]CellValue  `json:"rows"`
    RowCount   int            `json:"rowCount"`
    Truncated  bool           `json:"truncated"`
    DurationMs int64          `json:"durationMs"`
}
```

## 5. Cross-cutting requirements

- **Identifier quoting** (`backend/db/identifier.go`):
  ```go
  func QuoteIdentifier(name string) string {
      return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
  }
  ```
  Reject empty names with an explicit error.
- **Read-only enforcement** at two layers:
  1. DSN: `file:<absolute-path>?mode=ro&immutable=1` (URL-escape the path).
  2. Statement validator that tokenizes the SQL, strips leading comments/whitespace, splits on `;`, and allows only `SELECT`, `WITH ... SELECT`, and a hard-coded allow-list of harmless `PRAGMA`s (e.g. `table_info`, `foreign_key_list`, `index_list`, `index_info`, `database_list`, `schema_version`). Block all DDL/DML and `PRAGMA writable_schema`, `ATTACH`, `DETACH`, `VACUUM`.
- **Value coercion** in `backend/db/query.go`:
  - SQLite `NULL` -> `{kind:"null", value:null}`.
  - `INTEGER` -> JSON number if it fits in JS safe integer range, otherwise `{kind:"int", value:"<decimal string>"}` (preserve magnitude).
  - `REAL` -> JSON number; `NaN`/`Inf` become `null`.
  - `TEXT` -> JSON string.
  - `BLOB` -> `{kind:"blob", value:{hex:"<first N bytes hex>", size:<total>}}` with `N <= 64`.
- **Default caps**: `MaxQueryRows = 1000` (exported `const`), `DefaultQueryTimeout = 30 * time.Second`.
- **Errors**: backend returns `error`; frontend sees a `{code, message, detail}` shape. Codes include `NO_DB_OPEN`, `NOT_SQLITE`, `READ_ONLY_VIOLATION`, `TIMEOUT`, `MALFORMED_SQL`, `PERMISSION_DENIED`, `RESULT_TRUNCATED`.

---

## Stage 1 — Scaffold and tooling

**Goal.** Empty repo to a runnable Wails app skeleton with the chosen driver wired in.

**Scope / files touched.** Project root, `wails.json`, `go.mod`, `go.sum`, `frontend/`, `.gitignore`, `README.md`.

**Implementation notes.**
- Run `wails init -n sqlite-explorer -t react-ts` in the repo root (the directory may not be empty; init into a subfolder then move, or use a temp dir and copy).
- Pin Wails CLI version in README.
- `go get modernc.org/sqlite` and add a smoke import in `backend/db/connection.go` so the dependency is required.
- Add `wails dev` and `wails build` scripts to README; add a `Makefile` target `make dev` / `make build` is optional.
- `.gitignore`: `build/bin`, `frontend/node_modules`, `frontend/dist`, `*.app`, `.DS_Store`.

**Validation — automated.**
```bash
wails doctor                # exits 0
go vet ./...                # no issues
go build ./...              # compiles
cd frontend && npm run build
```

**Validation — manual.**
- `wails dev` opens the default Wails window, no console errors.
- `wails build` produces a runnable binary under `build/bin/`.

**Definition of done.**
- [ ] `wails doctor` clean.
- [ ] `wails dev` launches default window.
- [ ] `wails build` produces an artifact.
- [ ] `go vet ./...` clean.
- [ ] README has prerequisites, `wails dev`, `wails build`.

---

## Stage 2 — DB connection, file picker, identifier quoting

**Goal.** User can open a SQLite file via a native picker; backend holds a read-only connection; identifier quoting helper exists with tests.

**Scope / files touched.** `backend/app.go`, `backend/db/connection.go`, `backend/db/identifier.go`, `backend/db/identifier_test.go`, `backend/model/schema.go`.

**Implementation notes.**
- Use `runtime.OpenFileDialog` with filters:
  - SQLite (`*.sqlite;*.sqlite3;*.db`)
  - All files (`*`)
- Build DSN with `url.PathEscape` on the absolute path.
- Connection wrapper holds `*sql.DB`, `path`, `readOnly`, `openedAt`.
- After `sql.Open`, run `PRAGMA schema_version` to confirm the file is a valid SQLite database; if it errors with "file is not a database" map to `NOT_SQLITE`.
- `CloseDatabase` closes the handle and clears app state. `OpenDatabase` returns `NO_DB_OPEN`-style errors if called in invalid order.

**Validation — automated.**
- `TestQuoteIdentifier` table-driven:
  | input | expected |
  | --- | --- |
  | `users` | `"users"` |
  | `weird name` | `"weird name"` |
  | `with"quote` | `"with""quote"` |
  | `` (empty) | error |
- `TestOpen_ValidSqlite` opens `testdata/sample.sqlite` (stub for now; full fixture in Stage 9) and asserts `DatabaseInfo.ReadOnly == true`.
- `TestOpen_NotSqlite` writes `"hello"` to a `.db` file in `t.TempDir()` and expects a `NOT_SQLITE` error.
- `TestOpen_Missing` returns a wrapped `fs.ErrNotExist`.

```bash
go test ./backend/db/... -run 'TestQuoteIdentifier|TestOpen_' -v
```

**Validation — manual.**
- Picker shows the three SQLite extensions and an All Files option.
- Selecting a text file shows a user-readable "Not a SQLite database" message, not a stack trace.

**Definition of done.**
- [ ] `OpenDatabase` / `CloseDatabase` bound and callable from the frontend dev console.
- [ ] DSN includes `mode=ro&immutable=1`.
- [ ] All Stage 2 tests green.
- [ ] Quoting helper used everywhere identifiers are interpolated (verify via `rg`).

---

## Stage 3 — Schema introspection

**Goal.** Backend returns full schema (tables, views, indexes, triggers) with per-table column, PK, FK, and index metadata.

**Scope / files touched.** `backend/db/schema.go`, `backend/db/schema_test.go`, `backend/model/schema.go`.

**Implementation notes.**
- Query order:
  1. `SELECT type, name, tbl_name, sql FROM sqlite_schema WHERE name NOT LIKE 'sqlite_%' ORDER BY type, name;`
  2. For each table/view: `PRAGMA table_info(<quoted name>)`.
  3. For each table: `PRAGMA foreign_key_list(<quoted name>)`, `PRAGMA index_list(<quoted name>)`, then `PRAGMA index_info(<quoted index>)` per index.
- Row counts: do **not** run `SELECT COUNT(*)` eagerly. Expose `RowCountExact` lazily via a separate method `GetTableRowCount(name)` invoked on demand by the UI.
- All PRAGMA calls go through `db.QueryContext` with a 5s timeout.

**Validation — automated.**
- `TestGetSchema_Fixture` against the appendix fixture asserts:
  - exactly the expected set of tables, views, indexes, triggers (by name);
  - `customers` PK position == 1 on `id`;
  - `orders.customer_id` FK targets `customers.id` with `ON DELETE CASCADE`;
  - `customers.email` column is `NOT NULL` and has no default;
  - `notes.body` is nullable with default `NULL`;
  - the `orders_by_customer` view appears as a view, not a table.

```bash
go test ./backend/db/... -run TestGetSchema_ -v
```

**Validation — manual.**
- On the fixture DB, `GetSchema` returns in under 200 ms (log duration in dev build).

**Definition of done.**
- [ ] All four object types appear in the response.
- [ ] FK metadata includes `from`, `to`, `table`, `on_delete`, `on_update`.
- [ ] Index metadata distinguishes unique vs non-unique.
- [ ] Row count is **not** computed eagerly.

---

## Stage 4 — Frontend shell, sidebar, status bar

**Goal.** Three-pane app shell renders schema; tabs for Data, Schema, SQL scaffolded but inert.

**Scope / files touched.** `frontend/src/App.tsx`, `frontend/src/components/Sidebar.tsx`, `frontend/src/components/StatusBar.tsx`, `frontend/src/components/SchemaView.tsx`, `frontend/src/state/*`, basic CSS.

**Implementation notes.**
- Layout: left sidebar (resizable, 240px default), main panel with top-level tabs (`Data` | `Schema` | `SQL`), bottom status bar (height 28px).
- Sidebar groups: Tables, Views, Indexes, Triggers (collapsible, default expanded).
- Status bar fields: open DB path (or "No database"), last error (clickable for detail), last query duration, current page info when a table is open.
- Use a typed `WailsAPI` wrapper in `frontend/src/api.ts` over the auto-generated bindings so component code never imports `runtime` directly.
- State: lightweight (`zustand` or a single `React.Context` + `useReducer`). Do not pull in Redux.

**Validation — automated.**
- Component test (`vitest` + `@testing-library/react`) renders `<Sidebar schema={fixtureSchemaInfo} />` and asserts every fixture object name appears under its correct group.
- Component test for `<StatusBar />` verifies empty state shows "No database".

```bash
cd frontend && npm test -- --run
```

**Validation — manual.**
- Open a DB via picker (uses Stage 2). Sidebar populates. Selecting an item highlights it. Collapsing groups persists across re-renders within the session.
- With no DB open: each tab shows a clear empty state with a "Open database" CTA.

**Definition of done.**
- [ ] Layout matches description on macOS at 1280x800 without overflow.
- [ ] No console errors/warnings in dev.
- [ ] Component tests green.
- [ ] Frontend never touches Wails `runtime` outside `api.ts`.

---

## Stage 5 — Table data browser

**Goal.** Selecting a table or view in the sidebar shows paginated rows with sort and filter.

**Scope / files touched.** `backend/db/query.go` (`GetTableRows`), `backend/db/query_test.go`, `backend/model/query.go`, `frontend/src/components/DataGrid.tsx`, `frontend/src/components/SchemaView.tsx` (for the Data tab).

**Implementation notes.**
- Request:
  ```go
  type TableRowsRequest struct {
      Table      string `json:"table"`
      PageSize   int    `json:"pageSize"`   // 50|100|500|1000; default 100
      Page       int    `json:"page"`       // 1-indexed
      SortColumn string `json:"sortColumn"` // optional
      SortDesc   bool   `json:"sortDesc"`
      Filter     string `json:"filter"`     // optional; applied as case-insensitive LIKE across text columns
      WithTotal  bool   `json:"withTotal"`  // optional; lazy COUNT(*)
  }
  ```
- Build SQL as:
  ```
  SELECT <quoted cols> FROM <quoted table>
  [ WHERE <quoted col> LIKE ? ESCAPE '\' OR ... ]
  [ ORDER BY <quoted col> [DESC] ]
  LIMIT ? OFFSET ?
  ```
- The `Filter` value MUST be escaped: replace `\`, `%`, `_` with their `\`-escaped form before wrapping in `%...%`.
- Validate `Table` exists in schema before querying (prevents arbitrary identifier injection even with quoting).
- Validate `SortColumn` exists in the target table's column list.
- BLOB cells render as `<BLOB N bytes>` with a tooltip showing first 64 bytes as hex.
- Total row count is computed only when `WithTotal == true`; cache per `(table, filter)` in the connection wrapper for the session.

**Validation — automated.**
- `TestGetTableRows_Pagination`: generates a `t.TempDir()` SQLite with 10,000 rows in `nums(i INTEGER PRIMARY KEY, name TEXT)`; asserts page 1 returns rows 1..100, page 100 returns rows 9901..10000, page 101 returns 0 rows, `total == 10000` when `WithTotal`.
- `TestGetTableRows_Sort`: `SortColumn:"name"`, `SortDesc:true`, page 1 returns names in descending order.
- `TestGetTableRows_Filter_LikeEscape`: insert row `name="100%_off"`; filter `"100%_"` must match exactly that row, not every name containing the digits 100.
- `TestGetTableRows_BlobRendering`: insert a 1KB blob; assert `CellValue.Kind == "blob"`, `value.size == 1024`, `len(value.hex) == 128` (64 bytes hex).
- `TestGetTableRows_UnknownTable`: returns a typed error, not a SQL error string.

```bash
go test ./backend/db/... -run TestGetTableRows_ -v -count=1
```

**Validation — manual.**
- Generate a 1M-row table (script in `scripts/gen_big_db.go`). Browsing pages must stay under 150 ms per page on the dev machine (log in status bar).
- Page-size selector changes the page count immediately and resets to page 1.

**Definition of done.**
- [ ] Pagination, sort, filter all wired end-to-end.
- [ ] No identifier interpolation that bypasses validation against the live schema.
- [ ] All Stage 5 tests green.
- [ ] 1M-row browsing meets the latency target.

---

## Stage 6 — SQL query runner + read-only enforcement

**Goal.** A SQL editor pane lets the user run arbitrary read-only SQL with results, duration, error reporting, and per-session history.

**Scope / files touched.** `backend/db/query.go` (`RunQuery`), `backend/db/validator.go`, `backend/db/validator_test.go`, `backend/db/query_test.go`, `frontend/src/components/SqlEditor.tsx`, `frontend/src/state/history.ts`.

**Implementation notes.**
- Validator algorithm:
  1. Strip leading whitespace and SQL comments (`-- ...` and `/* ... */`).
  2. Split on top-level `;` (respect string literals and bracketed identifiers).
  3. For each non-empty statement, take the first identifier; allow only `SELECT`, `WITH`, or `PRAGMA <allow-listed name>`.
  4. For `WITH`, require the final keyword in the head to be `SELECT` (no `WITH ... INSERT`).
  5. Reject anything else with `READ_ONLY_VIOLATION` and the offending keyword in `detail`.
- Even though the connection is read-only, the validator gives a clear, early error message (better UX than the driver's generic "attempt to write a readonly database").
- `RunQuery` honors `DefaultQueryTimeout` and `MaxQueryRows`. When the row cap is hit, set `Truncated = true` and stop scanning further rows (do not error).
- Editor: a simple textarea with monospaced font is acceptable; Monaco/CodeMirror is optional. Cmd/Ctrl+Enter runs.
- History is a per-session in-memory list on the frontend, capped at 50, persisted only for the session.

**Validation — automated.**
- `TestReadOnlyValidator` (table-driven):
  | sql | allowed |
  | --- | --- |
  | `SELECT 1` | yes |
  | `  -- c\nSELECT 1` | yes |
  | `WITH t AS (SELECT 1) SELECT * FROM t` | yes |
  | `PRAGMA table_info(x)` | yes |
  | `PRAGMA writable_schema = 1` | no |
  | `INSERT INTO x VALUES (1)` | no |
  | `SELECT 1; DROP TABLE x` | no |
  | `ATTACH DATABASE 'x' AS y` | no |
  | `VACUUM` | no |
  | `update x set y=1` (case) | no |
  | `'; DROP TABLE x; --` | no |
- `TestRunQuery_BlocksWriteEvenIfValidatorBypassed`: directly call the lower-level executor with `INSERT`; the read-only DSN must still reject it.
- `TestRunQuery_TimesOut`: `WITH RECURSIVE c(x) AS (SELECT 1 UNION ALL SELECT x+1 FROM c) SELECT count(*) FROM c` with a 200ms context; expect `TIMEOUT`.
- `TestRunQuery_Truncation`: query returns 5000 rows, `MaxQueryRows=1000`, response has `RowCount==1000, Truncated==true`.
- `TestRunQuery_ColumnOrderStable`: `SELECT b, a FROM t` returns `Columns` as `[b, a]` and matching `Rows` order.
- `TestRunQuery_AllValueKinds`: a row containing NULL, INT, REAL, TEXT, BLOB round-trips correctly.

```bash
go test ./backend/db/... -run 'TestReadOnlyValidator|TestRunQuery_' -v
```

**Validation — manual.**
- Run `SELECT * FROM customers LIMIT 5` on the fixture: results render with the right columns and durations.
- Run `DROP TABLE customers`: error message reads "Read-only mode: DROP statements are not allowed." (or equivalent), no stack trace.
- Press Cmd/Ctrl+Enter executes the query. History shows the last five queries in the session.

**Definition of done.**
- [ ] Validator and read-only DSN both block writes.
- [ ] Result cap and timeout enforced.
- [ ] Errors surfaced cleanly in UI.
- [ ] All Stage 6 tests green.

---

## Stage 7 — CSV export

**Goal.** User can export the current table page or the current query result to CSV via a native save dialog.

**Scope / files touched.** `backend/db/export.go`, `backend/db/export_test.go`, `backend/app.go` (bind `ExportRowsToCSV`), frontend "Export CSV" buttons on Data and SQL tabs.

**Implementation notes.**
- Request:
  ```go
  type ExportRequest struct {
      Source  string `json:"source"`  // "tablePage" | "queryResult"
      Path    string `json:"path"`    // chosen via runtime.SaveFileDialog on frontend
      // For tablePage: include the same TableRowsRequest fields
      Table   string `json:"table,omitempty"`
      // For queryResult: include the SQL to re-run
      SQL     string `json:"sql,omitempty"`
  }
  ```
- Use `encoding/csv` with `csv.Writer`. Flush periodically every 1000 rows. Honor read-only validator for `queryResult`.
- Cell rendering:
  - `NULL` -> empty field (no quotes).
  - `INTEGER`/`REAL` -> base-10 string.
  - `TEXT` -> raw string (the csv writer handles quoting per RFC 4180).
  - `BLOB` -> `0x<hex of first 64 bytes>` followed by `...(<size> bytes)` if truncated.
- Header row is the column names.

**Validation — automated.**
- `TestExportCSV_RoundTrip`: write a CSV from a fixture, re-read with `csv.NewReader`, assert deep equality of cell strings.
- `TestExportCSV_SpecialChars`: cells containing `"`, `,`, `\n`, and Unicode (`"héllo"`) are correctly quoted/escaped per RFC 4180.
- `TestExportCSV_NullsAndBlobs`: NULL becomes empty field; a 1024-byte blob exports as `0x<128 hex chars>...(1024 bytes)`.
- `TestExportCSV_HonorsReadOnly`: an export request with `source="queryResult"` and `sql="DELETE FROM t"` returns `READ_ONLY_VIOLATION` and writes no file.

```bash
go test ./backend/db/... -run TestExportCSV_ -v
```

**Validation — manual.**
- Export the `customers` table from the fixture, open the file in Numbers or Excel: rows align with headers, special characters intact.
- Cancelling the save dialog produces no file and no error toast.

**Definition of done.**
- [ ] Both export sources work.
- [ ] No file is left on cancel/error.
- [ ] All Stage 7 tests green.

---

## Stage 8 — UX polish, error handling, performance caps

**Goal.** Loading states, empty states, consistent error model, truncation indication, query cancellation.

**Scope / files touched.** Frontend components; `backend/app.go` for cancellation handle; `backend/db/query.go` for timeout plumbing.

**Implementation notes.**
- Spinner overlay on Data and SQL panels during in-flight requests; debounced 150ms so fast queries don't flicker.
- Empty states with concrete next action ("Open a database", "Select a table", "Type a query").
- Cancel button on SQL tab: associates an `int64` query id with a `context.CancelFunc` in the app struct; calling `CancelQuery(id)` cancels the context.
- Status bar surfaces `Truncated` with a "Showing first 1000 rows" badge linked to docs.
- Map errors to user-readable messages in a single `errors.UserMessage(err)` helper; never show a raw Go error in the UI.

**Validation — automated.**
- `TestErrors_UserMessage` covers each error code -> expected message.
- `TestCancelQuery_Interrupts` starts a long query, cancels mid-flight, asserts `RunQuery` returns within 50ms of cancel with a `CANCELED` code.

**Validation — manual matrix.** Each scenario must show a clear, non-stack-trace message:
- [ ] No DB open + click "Run Query".
- [ ] Pick a file that does not exist (rename between picker and click).
- [ ] Pick a non-SQLite text file.
- [ ] Pick a file with `chmod 000`.
- [ ] Run `SELEC * FROM x` (malformed).
- [ ] Run `INSERT INTO x VALUES (1)` (mutating).
- [ ] Run `WITH RECURSIVE c(x) AS (SELECT 1 UNION ALL SELECT x+1 FROM c) SELECT count(*) FROM c` (timeout).
- [ ] Run `SELECT * FROM big_table` where `big_table > 1000 rows` (truncation badge).
- [ ] Browse a table while running a SQL query (no UI lockup).

**Definition of done.**
- [ ] All matrix items pass.
- [ ] No raw Go errors visible in the UI.
- [ ] Cancel button works on a long query.

---

## Stage 9 — Tests, sample DB, README, build

**Goal.** Round out test coverage, ship a sample DB and generator, polish the README, verify the release build.

**Scope / files touched.** `testdata/`, `scripts/gen_sample_db.go`, `scripts/gen_big_db.go`, `README.md`, CI optional (`.github/workflows/test.yml` if convenient).

**Implementation notes.**
- `scripts/gen_sample_db.go` runs the Appendix SQL to produce `testdata/sample.sqlite`. Commit the SQL; commit the binary only if small (<256 KB) and reproducible.
- `scripts/gen_big_db.go` accepts `-rows N` and produces a multi-million-row DB out-of-tree for manual perf checks.
- README must include:
  - Prerequisites (Go version, Node version, Wails CLI install command, platform notes for macOS code signing if relevant).
  - `wails dev` for hot-reload dev.
  - `wails build` for release.
  - Troubleshooting: "file is encrypted or is not a database" (means wrong driver options or wrong file), permission denied on macOS Gatekeeper.
  - Architecture diagram (text/ASCII) of backend <-> frontend boundary.
  - Security note: read-only enforcement strategy.

**Validation — automated.**
```bash
go test ./... -race -count=1            # all green
go vet ./...                            # clean
cd frontend && npm run build            # clean
wails build                             # produces release binary
```

**Validation — manual end-to-end script** (must all pass on the release binary, not `wails dev`):
1. Launch the built app.
2. Open `testdata/sample.sqlite`.
3. Sidebar shows: tables `customers`, `orders`, `notes`; view `orders_by_customer`; index `idx_orders_customer`; trigger `notes_updated_at`.
4. Click `customers` -> Data tab shows 5 rows; sort by `email` desc works.
5. SQL tab: run `SELECT * FROM orders_by_customer;` -> 5 rows.
6. SQL tab: run `INSERT INTO customers VALUES (99, 'x', 'x@x');` -> `READ_ONLY_VIOLATION` error.
7. Data tab: Export `customers` to CSV; open in a spreadsheet app.
8. Open a non-SQLite file -> graceful error.
9. Close database -> sidebar clears, status bar shows "No database".

**Definition of done.**
- [ ] `go test ./... -race -count=1` green.
- [ ] `wails build` produces a runnable artifact.
- [ ] README walk-through executed end-to-end without surprises.
- [ ] `testdata/sample.sqlite` (or generator) committed.

---

## Final acceptance checklist

- [ ] All nine stage Definitions of done satisfied.
- [ ] `go test ./... -race -count=1` passes locally.
- [ ] `wails build` succeeds and produces a runnable binary.
- [ ] Read-only mode demonstrably enforced at both the DSN and validator layers (the relevant tests cover both paths).
- [ ] Identifier quoting helper is the only path used for interpolating table/column/index names (verified by code search).
- [ ] 1M-row sample database remains browseable without UI freeze; pages render in under 150ms on dev hardware.
- [ ] No emojis in UI strings or source.
- [ ] No telemetry, network, or hosted-service calls in the app.
- [ ] README covers prerequisites, dev, build, troubleshooting, security note.

---

## Appendix — Sample test fixture (`testdata/fixtures.sql`)

```sql
PRAGMA foreign_keys = ON;

CREATE TABLE customers (
    id    INTEGER PRIMARY KEY,
    name  TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE
);

CREATE TABLE orders (
    id          INTEGER PRIMARY KEY,
    customer_id INTEGER NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    total_cents INTEGER NOT NULL CHECK (total_cents >= 0),
    placed_at   TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE notes (
    id         INTEGER PRIMARY KEY,
    body       TEXT,
    payload    BLOB,
    updated_at TEXT
);

CREATE INDEX idx_orders_customer ON orders(customer_id);

CREATE VIEW orders_by_customer AS
SELECT c.id AS customer_id, c.name, COUNT(o.id) AS order_count, COALESCE(SUM(o.total_cents), 0) AS total_cents
FROM customers c
LEFT JOIN orders o ON o.customer_id = c.id
GROUP BY c.id, c.name;

CREATE TRIGGER notes_updated_at
AFTER UPDATE ON notes
FOR EACH ROW
BEGIN
    UPDATE notes SET updated_at = datetime('now') WHERE id = OLD.id;
END;

INSERT INTO customers (id, name, email) VALUES
    (1, 'Example Customer 001',   'customer001@example.invalid'),
    (2, 'Example Customer 002',    'customer002@example.invalid'),
    (3, 'Example Customer 003',   'customer003@example.invalid'),
    (4, 'Example Customer 004','customer004@example.invalid'),
    (5, 'Example Customer 005',   'customer005@example.invalid');

INSERT INTO orders (id, customer_id, total_cents) VALUES
    (1, 1, 1299),
    (2, 1,  499),
    (3, 2, 9999),
    (4, 3,    0),
    (5, 5, 12345);

INSERT INTO notes (id, body, payload) VALUES
    (1, 'plain text note', NULL),
    (2, NULL,              x'00010203deadbeef'),
    (3, 'note with "quotes", commas, and a newline'||x'0a'||'inside', NULL);
```
