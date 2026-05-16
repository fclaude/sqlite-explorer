# SQLite Explorer — Development Progress

Living tracker for the staged build described in [`agent_prompt.md`](agent_prompt.md). Update this file as work advances: tick checkboxes, fill in `Status`, `Owner`, and `Notes`. The source of truth for *what* to build remains `agent_prompt.md`; this file tracks *where we are*.

When asked to **take 1 step**, follow [`WORKFLOW.md`](WORKFLOW.md) (Cursor rule: `.cursor/rules/one-step-at-a-time.mdc`).

## Status legend

- `Not started` — no work begun
- `In progress` — actively being worked on
- `Blocked` — work paused; reason in Notes
- `In review` — implementation done, awaiting validation
- `Done` — Definition of done satisfied and validated

## At-a-glance

| # | Stage | Status | Owner | Updated |
|---|-------|--------|-------|---------|
| 1 | Scaffold and tooling | Done | agent | 2026-05-15 |
| 2 | DB connection, file picker, identifier quoting | Done | agent | 2026-05-15 |
| 3 | Schema introspection | Done | agent | 2026-05-15 |
| 4 | Frontend shell, sidebar, status bar | Done | agent | 2026-05-15 |
| 5 | Table data browser | Done | agent | 2026-05-15 |
| 6 | SQL query runner + read-only enforcement | Not started | — | — |
| 7 | CSV export | Not started | — | — |
| 8 | UX polish, error handling, performance caps | Not started | — | — |
| 9 | Tests, sample DB, README, build | Not started | — | — |

Overall completion: 5 / 9 stages.

---

## Stage 1 — Scaffold and tooling

**Goal.** Empty repo to a runnable Wails app skeleton with `modernc.org/sqlite` wired in.

**Status:** Done
**Owner:** agent
**Started:** 2026-05-15
**Completed:** 2026-05-15

**Definition of done.**
- [x] `wails doctor` clean.
- [x] `wails dev` launches default window (not run in CI; `wails build` + packaged `.app` verified).
- [x] `wails build` produces an artifact (`build/bin/sqlite-explorer.app`).
- [x] `go vet ./...` clean.
- [x] README has prerequisites, `wails dev`, `wails build`.

**Notes.**
Wails CLI installed via `go install github.com/wailsapp/wails/v2/cmd/wails@v2.12.0`. App struct moved to `backend/`; `modernc.org/sqlite` wired in `backend/db/connection.go`.

---

## Stage 2 — DB connection, file picker, identifier quoting

**Goal.** Open a SQLite file via native picker; hold a read-only connection; ship `QuoteIdentifier` with tests.

**Status:** Done
**Owner:** agent
**Started:** 2026-05-15
**Completed:** 2026-05-15

**Definition of done.**
- [x] `OpenDatabase` / `CloseDatabase` bound and callable from the frontend.
- [x] DSN includes `mode=ro&immutable=1`.
- [x] `TestQuoteIdentifier`, `TestOpen_ValidSqlite`, `TestOpen_NotSqlite`, `TestOpen_Missing` green.
- [x] Quoting helper used everywhere identifiers are interpolated (only `identifier.go` for now; no dynamic SQL yet).

**Notes.**
`OpenDatabasePath` exposed for tests. `testdata/sample.sqlite` created by test helper. Makefile `run` target added.

---

## Stage 3 — Schema introspection

**Goal.** Return tables, views, indexes, triggers plus per-table column / PK / FK / index metadata.

**Status:** Done
**Owner:** agent
**Started:** 2026-05-15
**Completed:** 2026-05-15

**Definition of done.**
- [x] All four object types appear in the response.
- [x] FK metadata includes `from`, `to`, `table`, `on_delete`, `on_update`.
- [x] Index metadata distinguishes unique vs non-unique.
- [x] Row count is **not** computed eagerly (`GetTableRowCount` on demand).
- [x] `TestGetSchema_Fixture` green.

**Notes.**
`testdata/fixtures.sql` + `fixture.sqlite` for appendix fixture. Auto-indexes filtered from top-level index list.

---

## Stage 4 — Frontend shell, sidebar, status bar

**Goal.** Three-pane app shell that renders the schema; tabs for Data, Schema, SQL scaffolded.

**Status:** Done
**Owner:** agent
**Started:** 2026-05-15
**Completed:** 2026-05-15

**Definition of done.**
- [x] Layout renders cleanly at 1280x800 on macOS without overflow.
- [x] No console errors/warnings in dev.
- [x] Component tests (Sidebar from fixture, empty StatusBar) green.
- [x] Frontend never touches Wails `runtime` outside `api.ts`.

**Notes.**
`api.ts` Wails wrapper; React Context + useReducer state; vitest + Testing Library.

---

## Stage 5 — Table data browser

**Goal.** Paginated table grid with sort, filter, BLOB rendering, lazy total count.

**Status:** Done
**Owner:** agent
**Started:** 2026-05-15
**Completed:** 2026-05-15

**Definition of done.**
- [x] Pagination, sort, filter wired end-to-end.
- [x] No identifier interpolation that bypasses validation against the live schema.
- [x] `TestGetTableRows_Pagination`, `TestGetTableRows_Sort`, `TestGetTableRows_Filter_LikeEscape`, `TestGetTableRows_BlobRendering`, `TestGetTableRows_UnknownTable` green.
- [x] 1M-row script `scripts/gen_big_db.go` (manual perf check on dev machine).

**Notes.**
Row count cached per (table, filter). BLOB cells show `<BLOB N bytes>` with hex tooltip.

---

## Stage 6 — SQL query runner + read-only enforcement

**Goal.** SQL editor + result grid + read-only validator + per-session history.

**Status:** Not started
**Owner:** —
**Started:** —
**Completed:** —

**Definition of done.**
- [ ] Validator and read-only DSN both block writes.
- [ ] `MaxQueryRows` and `DefaultQueryTimeout` enforced.
- [ ] Errors surfaced cleanly in UI (no stack traces).
- [ ] `TestReadOnlyValidator`, `TestRunQuery_BlocksWriteEvenIfValidatorBypassed`, `TestRunQuery_TimesOut`, `TestRunQuery_Truncation`, `TestRunQuery_ColumnOrderStable`, `TestRunQuery_AllValueKinds` green.

**Notes.**
_None._

---

## Stage 7 — CSV export

**Goal.** Export current table page or current query result to CSV via native save dialog.

**Status:** Not started
**Owner:** —
**Started:** —
**Completed:** —

**Definition of done.**
- [ ] Both export sources (`tablePage`, `queryResult`) work.
- [ ] No file left behind on cancel/error.
- [ ] `TestExportCSV_RoundTrip`, `TestExportCSV_SpecialChars`, `TestExportCSV_NullsAndBlobs`, `TestExportCSV_HonorsReadOnly` green.

**Notes.**
_None._

---

## Stage 8 — UX polish, error handling, performance caps

**Goal.** Loading + empty states, consistent error model, truncation indicator, query cancellation.

**Status:** Not started
**Owner:** —
**Started:** —
**Completed:** —

**Definition of done.**
- [ ] All manual matrix items (no DB open, missing file, non-SQLite file, perm denied, malformed SQL, mutating SQL, timeout, truncation, concurrent browse+query) pass.
- [ ] No raw Go errors visible in the UI.
- [ ] Cancel button interrupts a long query.
- [ ] `TestErrors_UserMessage`, `TestCancelQuery_Interrupts` green.

**Notes.**
_None._

---

## Stage 9 — Tests, sample DB, README, build

**Goal.** Round out tests, ship sample DB + generator, finalize README, verify release build.

**Status:** Not started
**Owner:** —
**Started:** —
**Completed:** —

**Definition of done.**
- [ ] `go test ./... -race -count=1` green.
- [ ] `wails build` produces a runnable artifact.
- [ ] README walk-through executed end-to-end without surprises.
- [ ] `testdata/sample.sqlite` (or generator script) committed.

**Notes.**
_None._

---

## Final acceptance checklist

- [ ] All nine stage Definitions of done satisfied.
- [ ] `go test ./... -race -count=1` passes locally.
- [ ] `wails build` succeeds and produces a runnable binary.
- [ ] Read-only mode enforced at both DSN and validator layers (covered by tests).
- [ ] Identifier quoting helper is the only path for interpolating identifiers.
- [ ] 1M-row sample database browseable without UI freeze (<150ms per page).
- [ ] No emojis in UI strings or source.
- [ ] No telemetry, network, or hosted-service calls.
- [ ] README covers prerequisites, dev, build, troubleshooting, security note.

---

## Changelog

Append a one-line entry whenever a stage's status changes. Newest entries at the top.

- 2026-05-15 — Stage 5 -> Done. GetTableRows pagination/sort/filter, DataGrid UI, backend tests.
- 2026-05-15 — Stage 4 -> Done. Three-pane shell, Sidebar, StatusBar, tabs, api.ts, component tests.
- 2026-05-15 — Stage 3 -> Done. GetSchema, GetTableRowCount, PRAGMA introspection, fixture tests.
- 2026-05-15 — Stage 2 -> Done. OpenDatabase/CloseDatabase, read-only DSN, file picker, QuoteIdentifier + tests; Makefile `run` target.
- 2026-05-15 — Stage 1 -> Done. Wails react-ts scaffold, backend package, modernc.org/sqlite, README/Makefile; `wails doctor`, `go vet`, `npm run build`, `wails build` passed.
