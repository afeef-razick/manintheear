# CLAUDE.md — Agent Process Directives

This file governs how the agent contributes to the `manintheear` codebase.
Read this fully before any other action. Every rule here is non-negotiable.

---

## The goal of this project

This codebase is **the demo content** for a live talk. Every decision —
when to enter plan mode, which skills are invoked, how commits are structured —
is visible to an audience. Build accordingly. Clarity and process discipline
matter more than speed.

---

## Mandatory skills — use these proactively, not on request

| Skill | When |
|---|---|
| `grill-me` | Before implementing anything non-trivial. Walk every decision branch. |
| `pr-review` | Before merging any PR. Run it, address every finding. |
| `unit-test` | After every implementation step. Tests must exist and pass before PR. |

Never skip a skill because the change "feels small".

---

## Plan mode discipline

1. **Enter plan mode before writing any code.** Exit only after the user
   explicitly approves the plan.
2. Plans must name: files to create/edit, interfaces to define, tests to write.
3. If requirements shift mid-implementation, return to plan mode.

---

## Code quality

Follow `docs/go-code-convention.md` and `docs/go-logging-convention.md` exactly.
Those documents are the authoritative source for all code rules.

Key reminders:
- Named functions only. No anonymous closures for business logic.
- No comments that describe WHAT the code does — only WHY when non-obvious.
- No half-finished stubs. If a feature is not done, it is not committed.

This code will be projected on a TV. Write as if that is always true.

---

## Provider abstractions — non-negotiable

STT, LLM, and TTS each live behind a small interface:

```
Transcribe(audio []byte) (string, error)
Decide(script, state, transcript) -> (AIResponse, error)
Speak(text string) error
```

Implementations are injected at startup via config. The loop never imports
a concrete provider. This is the architecture; it does not change.

---

## Build cycle — 5 PR milestones

The project is built in exactly five atomic PRs. Each PR must be
fully reviewed and merged before the next begins.

| PR | Scope |
|---|---|
| 1 | Project init: module, folder structure, script parser, example script |
| 2 | Audio layer: portaudio capture, ring buffer, WAV encoder |
| 3 | Provider interfaces + STT (Whisper API) + TTS (macOS `say`) |
| 4 | LLM provider (AI CLI) + AI loop + drift detection + whisper manager |
| 5 | TUI (Bubble Tea) + session persistence + main.go wiring |

---

## Workflow for every PR

Full details in `docs/pr-workflow.md`. Summary:

1. Create branch: `feat/<scope>` or `fix/<scope>`
2. Implement following the Go convention docs
3. Write tests — see `docs/pre-pr-checklist.md`
4. Run pre-PR checklist — all items must be green
5. Open PR with `gh pr create`
6. Run `pr-review` skill — address all findings
7. Merge only when review is clean

---

## Git convention

Full details in `docs/git-convention.md`. Summary:

- Format: `type(scope): short description`
- Types: `feat`, `fix`, `test`, `docs`, `refactor`, `chore`
- Subject line ≤ 72 characters
- Do **not** mention any AI tool or assistant name in commits or PR bodies
- Body explains WHY, not WHAT

---

## GitHub

Use the `gh` CLI for all GitHub operations (PRs, issues, checks).
Never use the GitHub web UI when `gh` can do the same thing.

---

## MCP / tooling

MCP servers are preferred over raw HTTP when a connector exists.
Fall back to plain HTTP only when no MCP connector is available.
Document the choice in the PR body.
