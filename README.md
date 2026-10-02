# SQLite Explorer

Local-only desktop SQLite database explorer built with [Wails](https://wails.io), Go, and React + TypeScript. Opens `.sqlite` files for browsing and editing table rows; runs SQL in the SQL tab (read-only unless you allow more, see [SQL editor permissions](#sql-editor-permissions)); exports CSV.

## Prerequisites

- **Go** 1.25.13 or newer (`go version`)
- **Node.js** 20.19+ or 22.12+ and npm 10+ (`node --version`)
- **Wails CLI** v2.15.0 (install below)

Linux builds target the modern WebKit2GTK 4.1 ABI. On Debian 12 / Ubuntu 22.04 or newer:

```bash
sudo apt install build-essential pkg-config libgtk-3-dev libwebkit2gtk-4.1-dev
```

End users need the corresponding GTK3 and WebKit2GTK 4.1 runtime libraries. Package names for other distributions are listed in the [Wails Linux support guide](https://wails.io/docs/guides/linux-distro-support/).

### Install Wails CLI

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@v2.15.0
export PATH="$(go env GOPATH)/bin:$PATH"
wails doctor
```

On macOS 12 Monterey or newer, `wails doctor` may prompt you to install Xcode command-line tools if they are missing.

## Development

```bash
# Install the exact frontend dependency lock
cd frontend && npm ci && cd ..

# Hot reload
wails dev
```

Or via Makefile:

```bash
make dev              # hot reload
make install-deps     # npm ci
make test             # go test ./...
make frontend-test    # vitest
```

## Build and run

```bash
make build          # current platform
make build-linux    # Linux amd64, WebKit2GTK 4.1
make build-macos    # universal Intel + Apple Silicon .app (run on macOS)
make run            # build and launch the current-platform app
```

Release binary: `build/bin/sqlite-explorer` (macOS: `build/bin/sqlite-explorer.app`).

Linux publishing produces `.deb` and Fedora-compatible `.rpm` packages with GTK/WebKit runtime dependencies, plus a portable `.tar.gz`. Releases also include tracked-source snapshots named `sqlite-explorer-VERSION.src.tar.gz` and `sqlite-explorer-VERSION.src.zip`. macOS publishing produces a universal, ad-hoc-signed `.dmg` without Apple notarization; Gatekeeper may require users to approve it through Privacy & Security. Tagging a version such as `v0.1.2` runs the release workflow; details and the release checklist are in [`docs/RELEASING.md`](docs/RELEASING.md).

SQLite Explorer is available under the [MIT License](LICENSE).

## Sample database

The appendix fixture lives in `testdata/fixtures.sql`. Generate `testdata/sample.sqlite`:

```bash
go run scripts/gen_sample_db.go
```

Use this file for manual walk-throughs and tests. A large DB for performance checks:

```bash
make gen-big-db   # writes /tmp/sqlite-explorer-big.db (1M rows by default)
```

## Architecture

```
+------------------+     Wails bindings (JSON)      +------------------+
|  React frontend  |  <-------------------------->  |  backend.App     |
|  api.ts wrapper  |                              |  (app.go)        |
+------------------+                              +--------+---------+
                                                           |
                                                           v
                                                  +--------+---------+
                                                  |  backend/db      |
                                                  |  SQLite file     |
                                                  +------------------+
```

- **Frontend** (`frontend/src/`): UI only; calls Go through `api.ts` (never imports `wailsjs` from components).
- **Backend** (`backend/`): file picker, schema, paginated rows, SQL statement classifier, CSV export, query cancel.
- **Driver**: [`modernc.org/sqlite`](https://pkg.go.dev/modernc.org/sqlite) (pure Go, no CGO).

## SQL editor permissions

The SQL tab runs read queries by default. **Permissions** in the editor toolbar lists what runs and lets you allow more categories; the choice is remembered on this machine.

| Category | Statements | Default |
|---|---|---|
| Read queries | `SELECT`, `VALUES`, `WITH … SELECT`, `EXPLAIN`, read-only `PRAGMA`s (listed in the panel) | Always allowed |
| Data changes | `INSERT`, `REPLACE`, `UPDATE`, `DELETE`, `WITH … INSERT/UPDATE/DELETE` | Off |
| Schema changes | `CREATE`, `ALTER`, `DROP` | Off |
| Transactions | `BEGIN`, `COMMIT`, `END`, `ROLLBACK`, `SAVEPOINT`, `RELEASE` | Off |
| Maintenance | `VACUUM`, `VACUUM INTO`, `ANALYZE`, `REINDEX` | Off |
| Attach databases | `ATTACH`, `DETACH` | Off |
| Other PRAGMAs | any `PRAGMA` not on the read-only list, including assignments | Off |
| Never allowed | `PRAGMA writable_schema` (can corrupt the file) | Blocked |

- Each statement in a script is checked separately. The splitter follows SQLite's own rules for quotes, comments, and `CREATE TRIGGER` bodies, and is tested against `sqlite3_complete()`.
- Read-only runs use a separate `mode=ro` connection with `PRAGMA query_only`, so SQLite itself refuses writes.
- Runs that change anything use a dedicated connection that is closed afterwards: temporary tables, attached databases, and `PRAGMA` settings last only for that run. A transaction left open when a run ends is rolled back.
- **Export CSV** in the SQL tab runs the query again, so it only accepts read queries.
- The SQLite driver parses `DATE`, `DATETIME`, and `TIMESTAMP` columns selected directly in a query, so the SQL tab shows them as `YYYY-MM-DD HH:MM:SS`. Select `CAST(col AS TEXT)` to see the stored text. The Data tab, row editor, and table export always use the stored text.

## Security

1. **SQL tab**: every statement is classified and checked against the permissions above before it runs; read-only runs are also enforced by SQLite (`mode=ro` plus `query_only`).
2. **Table edits**: row updates use parameterized `UPDATE` statements with quoted identifiers (`QuoteIdentifier`); only base tables (not views) are editable.
3. **Table browse**: table and column names are quoted via `QuoteIdentifier` before interpolation.
4. **Local-only operation**: the app has no analytics, telemetry, update checker, or network API. Database contents remain on the machine unless the user explicitly exports CSV.

Please report vulnerabilities through the repository's private security-advisory channel; see [`SECURITY.md`](SECURITY.md). Do not attach a private database to a public issue.

## Troubleshooting

| Symptom | Likely cause |
|--------|----------------|
| `file is encrypted or is not a database` | Wrong file selected, corrupted file, or not SQLite — not a driver bug. |
| Permission denied opening the database | OS file permissions (`chmod`) or macOS privacy restrictions on the file location. |
| App won't open on macOS (Gatekeeper) | Right-click the `.app` → Open, or allow in System Settings → Privacy & Security. |
| Query timed out | Heavy query; simplify or add limits. Default timeout is 30 seconds. |
| Showing first 1000 rows | Result cap for responsiveness; **Export CSV** writes every row. |
| "… is not allowed. Enable … under Permissions" | The SQL contains a statement category that is switched off; see [SQL editor permissions](#sql-editor-permissions). |
| "The run ended inside an open transaction" | The script ran `BEGIN` without `COMMIT`; its changes were rolled back. |

## Tests

```bash
cd frontend
npm ci
npm test -- --run
npm run build
npm audit --audit-level=low
cd ..
go test ./... -race -count=1
go vet ./...
wails build -clean -trimpath
```
