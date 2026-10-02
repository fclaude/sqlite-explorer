# SQLite Explorer

A desktop app for browsing and editing SQLite databases, built with [Wails](https://wails.io), Go, and React. It works entirely offline: no telemetry, no update checks, no network access.

- Browse tables, views, indexes, and triggers; page, sort, and filter rows.
- Edit rows in base tables. BLOBs are edited as hex.
- Run SQL. Only read queries run until you allow more (see [SQL editor permissions](#sql-editor-permissions)).
- View row counts, indexes, and storage use per table.
- Export the current page, every matching row of a table, or a full query result to CSV.

## Install

Download a package from [Releases](https://github.com/fclaude/sqlite-explorer/releases):

| Platform | File |
|---|---|
| Debian 12, Ubuntu 22.04 or newer | `sqlite-explorer_VERSION_amd64.deb` |
| Fedora | `sqlite-explorer-VERSION-1.x86_64.rpm` |
| Other Linux with GTK3 and WebKit2GTK 4.1 | `sqlite-explorer-VERSION-linux-amd64.tar.gz` |
| macOS 12 or newer, Intel or Apple Silicon | `SQLite-Explorer-VERSION-universal-unsigned.dmg` |

`SHA256SUMS` lists a checksum for every file. The macOS app is ad-hoc signed but not notarized, so macOS may block the first launch; if you trust the download, choose **Open Anyway** under **System Settings → Privacy & Security**.

## SQL editor permissions

The SQL tab runs read queries only. Open **Permissions** in the editor toolbar to see what runs and to allow more; the choice is remembered on this computer.

| Category | Statements | Default |
|---|---|---|
| Read queries | `SELECT`, `VALUES`, `WITH … SELECT`, `EXPLAIN`, read-only `PRAGMA`s (listed in the panel) | Always allowed |
| Data changes | `INSERT`, `REPLACE`, `UPDATE`, `DELETE`, `WITH … INSERT/UPDATE/DELETE` | Off |
| Schema changes | `CREATE`, `ALTER`, `DROP` | Off |
| Transactions | `BEGIN`, `COMMIT`, `END`, `ROLLBACK`, `SAVEPOINT`, `RELEASE` | Off |
| Maintenance | `VACUUM`, `VACUUM INTO`, `ANALYZE`, `REINDEX` | Off |
| Attach databases | `ATTACH`, `DETACH` | Off |
| Other PRAGMAs | Any `PRAGMA` not on the read-only list, including assignments | Off |
| Never allowed | `PRAGMA writable_schema`, which can corrupt the file | Blocked |

- Every statement in a script is checked. The statement splitter follows SQLite's rules for quotes, comments, and `CREATE TRIGGER` bodies and is tested against SQLite's own `sqlite3_complete()`.
- Read queries run on a connection opened with `mode=ro` and `PRAGMA query_only`, so SQLite itself refuses writes.
- Anything else runs on its own connection, which is closed when the run ends: temporary tables, attached databases, and `PRAGMA` settings do not carry over, and a transaction left open is rolled back.
- **Export CSV** runs the query again, so it accepts read queries only.
- The SQLite driver parses `DATE`, `DATETIME`, and `TIMESTAMP` columns selected directly in a query and shows them as `YYYY-MM-DD HH:MM:SS`. Select `CAST(col AS TEXT)` to see the stored text. The Data tab, the row editor, and table exports always show the stored text.

## Troubleshooting

| Symptom | Cause |
|---|---|
| `file is encrypted or is not a database` | The file is not an SQLite database, or it is damaged or encrypted. |
| Permission denied opening the database | File permissions, or macOS privacy restrictions on the folder. |
| macOS will not open the app | See [Install](#install) for approving an unnotarized app. |
| Query timed out | Queries and SQL runs stop after 30 seconds. Narrow the query or add a `LIMIT`. |
| Showing the first 1000 rows | The SQL tab shows at most 1000 rows; **Export CSV** writes all of them. |
| "… is not allowed. Enable … under Permissions" | The SQL uses a statement category that is off; see [SQL editor permissions](#sql-editor-permissions). |
| "The run ended inside an open transaction" | The script ran `BEGIN` without `COMMIT`, so its changes were rolled back. |

## Development

You need Go 1.25.13 or newer, Node.js 20.19+ or 22.12+ with npm 10+, and the Wails CLI v2.15.0 (`make install-wails`). Linux builds also need the GTK3 and WebKit2GTK 4.1 development packages:

```bash
sudo apt install build-essential pkg-config libgtk-3-dev libwebkit2gtk-4.1-dev   # Debian, Ubuntu
sudo dnf install gcc pkgconf-pkg-config gtk3-devel webkit2gtk4.1-devel           # Fedora
```

| Command | Purpose |
|---|---|
| `make install-deps` | Install the locked frontend dependencies (`npm ci`) |
| `make dev` | Run the app with hot reload |
| `make test` | Build the frontend, then run the Go tests |
| `make frontend-test` | Run the frontend tests |
| `make vet` / `make audit` | Run `go vet` / `govulncheck` and `npm audit` |
| `make build` | Build for this platform into `build/bin/` |
| `make build-linux` / `make build-macos` | Build for Linux amd64 / a universal macOS app (on macOS) |
| `make gen-sample-db` | Rebuild `testdata/sample.sqlite` from `testdata/fixtures.sql` |
| `make gen-big-db` | Write a one-million-row database to `/tmp/sqlite-explorer-big.db` |

CI runs the tests (Go with `-race`), `go vet`, both audits, and the Linux and macOS package builds. [docs/RELEASING.md](docs/RELEASING.md) covers releases; notes for each release are in [docs/release-notes/](docs/release-notes/).

Code layout:

- `frontend/src/`: the React UI. Components call Go only through `api.ts`.
- `backend/app.go`: Wails bindings, dialogs, cancellation, and error reporting.
- `backend/db/`: database access, including the SQL statement classifier, row edits, and CSV export. The driver is [`modernc.org/sqlite`](https://pkg.go.dev/modernc.org/sqlite), so no CGO is needed.

## Security

- SQL from the editor runs only within the permissions above.
- Row edits use parameterized `UPDATE … WHERE rowid = ?` statements, and only base tables can be edited.
- Table and column names in generated SQL are always quoted; values are always bound parameters.
- Database contents leave the computer only when you export them.

## License

[MIT](LICENSE)
