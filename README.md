# SQLite Explorer

Local-only desktop SQLite database explorer built with [Wails](https://wails.io), Go, and React + TypeScript. Opens `.sqlite` files for browsing and editing table rows; run validated read-only SQL in the SQL tab; export CSV.

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

Linux publishing produces `.deb` and Fedora-compatible `.rpm` packages with GTK/WebKit runtime dependencies, plus a portable `.tar.gz`. Releases also include tracked-source snapshots named `sqlite-explorer-VERSION.src.tar.gz` and `sqlite-explorer-VERSION.src.zip`. macOS publishing produces a universal, ad-hoc-signed `.dmg` without Apple notarization; Gatekeeper may require users to approve it through Privacy & Security. Tagging a version such as `v0.1.1` runs the release workflow; details and the release checklist are in [`docs/RELEASING.md`](docs/RELEASING.md).

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
- **Backend** (`backend/`): file picker, schema, paginated rows, SQL validator, CSV export, query cancel.
- **Driver**: [`modernc.org/sqlite`](https://pkg.go.dev/modernc.org/sqlite) (pure Go, no CGO).

## Security

1. **SQL tab**: every ad-hoc query is validated; only `SELECT` / read-only `WITH` / allow-listed `PRAGMA` statements run.
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
| Showing first 1000 rows | Result cap for responsiveness; use `LIMIT` or export for more. |

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
