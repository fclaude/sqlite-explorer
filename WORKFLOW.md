# Development workflow

## One step at a time

When you ask the agent to **take 1 step** (or "next step"), it will:

1. Read [`agent_prompt.md`](agent_prompt.md) — what to build and how to validate each stage.
2. Read [`progress.md`](progress.md) — which stages are done, in progress, or blocked.
3. Pick the **next pending stage** (lowest number not marked `Done`).
4. Implement **only that stage**, run its validation criteria, then update `progress.md`.

Do not expect multiple stages in a single "take 1 step" request unless you say so explicitly.

## Files

| File | Role |
|------|------|
| `agent_prompt.md` | Full staged build plan, validation criteria, acceptance checklist |
| `progress.md` | Live status tracker — update after each step |
| `.cursor/rules/one-step-at-a-time.mdc` | Cursor rule so the agent follows this workflow automatically |

## Status values (`progress.md`)

- `Not started` — no work yet
- `In progress` — currently being worked on
- `Blocked` — paused; see Notes
- `In review` — done but validation pending
- `Done` — Definition of done satisfied
