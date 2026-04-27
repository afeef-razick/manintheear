# Go Logging Convention

Mandatory standard for all observability in this repository.
Uses `log/slog` (Go stdlib ≥ 1.21). All rules are authoritative.

---

## 1. Logger initialisation (MANDATORY)

Each `internal/` package that emits logs MUST initialise a scoped logger at
package level, adding a `package` attribute so every log line is attributable:

```go
var logger = slog.Default().With("package", "loop")
```

Goroutines within a package use that package logger. They MUST NOT call
`slog.Info(...)` on the global logger directly — always use the scoped one.

---

## 2. Level rules by layer

| Level | Use for | Where |
|---|---|---|
| `DEBUG` | Goroutine lifecycle, trigger conditions, cycle counts | All packages |
| `INFO` | Significant state transitions visible to the operator | `loop`, `session` |
| `WARN` | Recoverable failures, retries, context cancellation | All packages |
| `ERROR` | Unrecoverable failure within a cycle; does not stop the program | All packages |

`os.Exit` and `log.Fatal` are not observability — they live only in `main`.

Context cancellation is shutdown, not failure. Log it at WARN:

```go
slog.Warn("goroutine exiting", "reason", "context canceled", "goroutine", "runSTT")
```

---

## 3. Structured fields (MANDATORY)

All log calls MUST use key-value pairs. Never use string interpolation in the
message to carry data.

```go
// correct
logger.Info("beat covered", "beat_id", beatID, "phase", phaseID)

// forbidden
logger.Info(fmt.Sprintf("beat %s covered in phase %d", beatID, phaseID))
```

Field names MUST be `snake_case`. Canonical field names for this project:

| Field | Type | Meaning |
|---|---|---|
| `beat_id` | string | Beat identifier from script |
| `phase` | int | Current phase number |
| `attempt` | int | Whisper attempt count (1 or 2) |
| `urgency` | string | `low`, `medium`, `high` |
| `goroutine` | string | Goroutine name for lifecycle logs |
| `provider` | string | `whisper`, `ai_cli`, `say` |
| `elapsed_ms` | int64 | Duration in milliseconds |
| `words` | int | Word count of transcript chunk |
| `err` | error | Wrapped error value |

---

## 4. What each package logs

### `loop`

MUST log:
- `DEBUG` when the AI trigger fires (elapsed, words)
- `INFO` when state transitions (phase change, beat covered)
- `INFO` when a whisper fires (beat_id, attempt, urgency)
- `WARN` on LLM retry after malformed JSON
- `ERROR` on LLM failure after retry (continues running)

MUST NOT log:
- Raw transcript text (size, privacy)
- The full script JSON

### `stt`

MUST log:
- `DEBUG` on each Whisper API call (duration, word count returned)
- `WARN` on API failure (continues; TUI shows indicator)
- `WARN` on empty transcript returned

MUST NOT log:
- The audio bytes or their size

### `llm`

MUST log:
- `DEBUG` on each LLM CLI invocation (elapsed_ms)
- `WARN` on first malformed JSON (triggering retry)
- `ERROR` on second malformed JSON (treat as null whisper)
- `WARN` on CLI process error

### `tts`

MUST log:
- `WARN` on `say` failure (non-zero exit)

MUST NOT log at `INFO` — TTS is a side effect, not a business event.

### `audio`

MUST log:
- `WARN` when no audio received for >5s
- `DEBUG` on stream start and stop

### `tui`

MUST NOT emit any `slog` calls. The TUI surfaces errors visually via
`Update` messages from the loop. It is a display layer only.

### `session`

MUST log:
- `INFO` on session create and resume (session_id, talk_id)
- `WARN` on any write failure (non-fatal, but worth knowing)

---

## 5. Hot path logging

Inside the STT ticker or AI loop cycle, use `slog.LogAttrs` to avoid
allocations:

```go
logger.LogAttrs(ctx, slog.LevelDebug, "loop trigger",
    slog.Int64("elapsed_ms", elapsed.Milliseconds()),
    slog.Int("words", wordCount),
)
```

Do not use `logger.Debug(...)` with key-value pairs inside loops that fire
every few seconds — it allocates on every call.

---

## 6. Forbidden

- `fmt.Println` or `fmt.Printf` for observability anywhere outside `main`.
- `log.Print*` family — use `slog` only.
- Logging inside provider interfaces (only implementations log).
- Logging the same event at multiple levels in the same code path.
- Emitting `ERROR` for a condition the program recovers from — use `WARN`.
- Using the message string to carry structured data.
