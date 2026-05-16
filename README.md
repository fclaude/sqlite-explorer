# SQLite Explorer

Local-only desktop SQLite database explorer built with [Wails](https://wails.io), Go, and React + TypeScript.

## Prerequisites

- **Go** 1.23 or newer (`go version`)
- **Node.js** 18+ and npm (`node --version`)
- **Wails CLI** v2.12.0 (install below)

### Install Wails CLI

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
export PATH="$(go env GOPATH)/bin:$PATH"
wails doctor
```

On macOS, `wails doctor` may prompt you to install Xcode command-line tools if they are missing.

## Development

```bash
# Install frontend dependencies (first time)
cd frontend && npm install && cd ..

# Run with hot reload
wails dev
```

Or via Makefile:

```bash
make dev      # hot reload
make build    # production build
make run      # build then launch the app
```

## Build and run

```bash
make build    # production build -> build/bin/sqlite-explorer.app (macOS)
make run      # build and open the app
```

Release binary: `build/bin/sqlite-explorer` (macOS: `build/bin/sqlite-explorer.app`).

## Project layout

- `backend/` — Go application logic and SQLite access
- `frontend/` — React + TypeScript UI
- `agent_prompt.md` — staged build plan
- `progress.md` — development status tracker

## SQLite driver

Uses [`modernc.org/sqlite`](https://pkg.go.dev/modernc.org/sqlite) (pure Go, no CGO).

## Staged development

See `agent_prompt.md` and `progress.md`. Ask the agent to **take 1 step** to advance one stage at a time.
