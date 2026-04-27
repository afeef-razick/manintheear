# Go Code Convention

Mandatory standard for all Go packages in this repository.
All rules are authoritative.

---

## 1. Constructors

Every exported type that requires initialisation MUST use this signature:

```go
func New(deps ...Interface) (*Type, error)
```

- Return `(*Type, error)`. Never return only the struct.
- If the constructor requires `context.Context`, it MUST be the first argument.
- The constructor MUST NOT perform I/O blocking work (use `Start()` for that).

Forbidden patterns:

```go
func New() Type            // missing error
func NewWhisperProvider()  // name the factory after the package, not the type
func init() { ... }        // no init-time side effects
```

---

## 2. Context propagation (MANDATORY)

Every function that performs I/O, calls an external process, or blocks MUST
accept `ctx context.Context` as its first argument.

```go
func (l *Loop) Run(ctx context.Context) error { ... }
func (w *WhisperProvider) Transcribe(ctx context.Context, audio []byte) (string, error) { ... }
```

`context.WithCancel` and `context.WithTimeout` MUST be followed by
`defer cancel()` on the very next line:

```go
ctx, cancel := context.WithCancel(parent)
defer cancel()
```

Failure to call `cancel()` is a goroutine leak. It is a defect.

---

## 3. Provider interface rules

Provider interfaces (STT, LLM, TTS) MUST be defined in the consumer package
(`internal/loop`), not in the provider package. This is the Go idiom:
interfaces belong at the point of use.

```go
// in internal/loop/loop.go
type STTProvider interface {
    Transcribe(ctx context.Context, audio []byte) (string, error)
}
```

Interfaces MUST have 1–3 methods. An interface with more than 3 methods is a
design smell — split the responsibility.

The loop package MUST NOT import any concrete provider package. Providers are
injected via `New(...)`.

Accept interfaces, return structs:

```go
// correct: takes interface, returns concrete
func New(stt STTProvider, llm LLMProvider, tts TTSProvider) *Loop { ... }

// forbidden: takes concrete type
func New(stt *WhisperProvider) *Loop { ... }
```

---

## 4. Goroutine ownership (MANDATORY)

The goroutine that creates a goroutine owns it. The owner is responsible for:

1. Ensuring the goroutine exits.
2. Closing any channel the goroutine writes to.
3. Draining errors via a dedicated error channel or `sync.WaitGroup`.

Every goroutine's main loop MUST select on `ctx.Done()`:

```go
for {
    select {
    case <-ctx.Done():
        return ctx.Err()
    case chunk := <-transcriptCh:
        // process
    }
}
```

A goroutine that ignores `ctx.Done()` is a defect.

---

## 5. Channel conventions

- The sender closes the channel. Receivers never close.
- Channel direction MUST be typed in function signatures:

```go
func runSTT(ctx context.Context, out chan<- TranscriptChunk) error { ... }
func runTUI(ctx context.Context, in <-chan Update) error { ... }
```

- Buffered channels MUST document the buffer size rationale in the line above:

```go
// 64 slots: one per STT cycle at maximum burst rate
updates := make(chan Update, 64)
```

- Never use an unbuffered channel for inter-goroutine updates where the
  receiver might be busy (TUI redraws). Prefer buffered with a drop policy.

---

## 6. Error handling

Wrap errors with context at every layer boundary:

```go
return fmt.Errorf("stt transcribe: %w", err)
```

Never:
- Discard errors with `_`.
- Return raw errors without context across package boundaries.
- Call `log.Fatal` outside of `main()`.
- Call `os.Exit` outside of `main()`.
- Use `panic` outside of unreachable-by-design code paths.

Sentinel errors use the `Err` prefix:

```go
var ErrScriptNotFound = errors.New("script file not found")
var ErrSessionCorrupt = errors.New("session state is unreadable")
```

---

## 7. Naming

| What | Rule | Example |
|---|---|---|
| Provider implementations | `<Adjective>Provider` | `WhisperProvider`, `CLIProvider` |
| Factory functions | Named after the package | `stt.New(cfg)` |
| Goroutine entry points | Named verb | `runLoop`, `runSTT`, `runTUI` |
| Channels | Named for content | `transcriptCh`, `whisperCh`, `updateCh` |
| Context parameter | Always `ctx` | `func Foo(ctx context.Context)` |
| Sentinel errors | `Err` prefix | `ErrNoAudio` |

---

## 8. Package responsibilities

Each `internal/` package owns exactly one concern:

| Package | Owns |
|---|---|
| `audio` | Capture, ring buffer, WAV encoding |
| `stt` | Transcription interface + Whisper implementation |
| `llm` | Decision interface + AI CLI implementation |
| `tts` | Speech interface + `say` implementation |
| `loop` | AI loop, trigger rules, drift detection, whisper manager |
| `tui` | Terminal rendering only — no business logic |
| `session` | Disk persistence only |
| `script` | Parsing only |
| `config` | Env/flag loading only |

A package MUST NOT import a sibling package unless it is the consumer
(e.g. `loop` imports provider interfaces; `main` imports everything).

---

## 9. Forbidden everywhere

- Global mutable state (package-level `var` that is written after init).
- Unexported struct fields mutated from outside the package via pointer tricks.
- String formatting inside hot paths — use `slog.LogAttrs` (see logging doc).
- Shadowing the `err` variable in nested scopes.
- Anonymous functions as named business logic — give them a name.
