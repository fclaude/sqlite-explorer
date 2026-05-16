# SQLite Explorer — Development Progress

Living tracker for the staged build described in [`agent_prompt.md`](agent_prompt.md). Update this file as work advances: tick checkboxes, fill in `Status`, `Owner`, and `Notes`. The source of truth for *what* to build remains `agent_prompt.md`; this file tracks *where we are*.

## Status legend

- `Not started` — no work begun
- `In progress` — actively being worked on
- `Blocked` — work paused; reason in Notes
- `In review` — implementation done, awaiting validation
- `Done` — Definition of done satisfied and validated

## At-a-glance

| # | Stage | Status | Owner | Updated |
|---|-------|--------|-------|---------|
| 1 | Scaffold and tooling | Not started | — | — |
| 2 | DB connection, file picker, identifier quoting | Not started | — | — |
| 3 | Schema introspection | Not started | — | — |
| 4 | Frontend shell, sidebar, status bar | Not started | — | — |
| 5 | Table data browser | Not started | — | — |
| 6 | SQL query runner + read-only enforcement | Not started | — | — |
| 7 | CSV export | Not started | — | — |
| 8 | UX polish, error handling, performance caps | Not started | — | — |
| 9 | Tests, sample DB, README, build | Not started | — | — |

Overall completion: 0 / 9 stages.

---

## Stage 1 — Scaffold and tooling

**Goal.** Empty repo to a runnable Wails app skeleton with `modernc.org/sqlite` wired in.

**Status:** Not started
**Owner:** —
**Started:** —
**Completed:** —

**Definition of done.**
- [ ] `wails doctor` clean.
- [ ] `wails dev` launches default window.
- [ ] `wails build` produces an artifact.
- [ ] `go vet ./...` clean.
- [ ] README has prerequisites, `wails dev`, `wails build`.

**Notes.**
_None._

---

## Stage 2 — DB connection, file picker, identifier quoting

**Goal.** Open a SQLite file via native picker; hold a read-only connection; ship `QuoteIdentifier` with tests.

**Status:** Not started
**Owner:** —
**Started:** —
**Completed:** —

**Definition of done.**
- [ ] `OpenDatabase` / `CloseDatabase` bound and callable from the frontend.
- [ ] DSN includes `mode=ro&immutable=1`.
- [ ] `TestQuoteIdentifier`, `TestOpen_ValidSqlite`, `TestOpen_NotSqlite`, `TestOpen_Missing` green.
- [ ] Quoting helper used everywhere identifiers are interpolated.

**Notes.**
_None._

---

## Stage 3 — Schema introspection

**Goal.** Return tables, views, indexes, triggers plus per-table column / PK / FK / index metadata.

**Status:** Not started
**Owner:** —
**Started:** —
**Completed:** —

**Definition of done.**
- [ ] All four object types appear in the response.
- [ ] FK metadata includes `from`, `to`, `table`, `on_delete`, `on_update`.
- [ ] Index metadata distinguishes unique vs non-unique.
- [ ] Row count is **not** computed eagerly.
- [ ] `TestGetSchema_Fixture` green.

**Notes.**
_None._

---

## Stage 4 — Frontend shell, sidebar, status bar

**Goal.** Three-pane app shell that renders the schema; tabs for Data, Schema, SQL scaffolded.

**Status:** Not started
**Owner:** —
**Started:** —
**Completed:** —

**Definition of done.**
- [ ] Layout renders cleanly at 1280x800 on macOS without overflow.
- [ ] No console errors/warnings in dev.
- [ ] Component tests (Sidebar from fixture, empty StatusBar) green.
- [ ] Frontend never touches Wails `runtime` outside `api.ts`.

**Notes.**
_None._

---

## Stage 5 — Table data browser

**Goal.** Paginated table grid with sort, filter, BLOB rendering, lazy total count.

**Status:** Not started
**Owner:** —
**Started:** —
**Completed:** —

**Definition of done.**
- [ ] Pagination, sort, filter wired end-to-end.
- [ ] No identifier interpolation that bypasses validation against the live schema.
- [ ] `TestGetTableRows_Pagination`, `TestGetTableRows_Sort`, `TestGetTableRows_Filter_LikeEscape`, `TestGetTableRows_BlobRendering`, `TestGetTableRows_UnknownTable` green.
- [ ] 1M-row generated DB browseable at <150ms per page.

**Notes.**
_None._

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

- _YYYY-MM-DD — example: Stage 1 -> In progress (owner: foo)._
