# Man-in-the-Ear Tool — Product Spec

> Process directives for Claude live in `CLAUDE.md`. Read that first.

## What the tool does

Listens to a live talk via laptop mic. Whispers short reminders into a
bluetooth earphone if the speaker is missing beats or drifting off the
planned path. Has the talk script loaded as context. Runs locally for
the duration of one talk.

## Two jobs only

1. **Coverage** — make sure every important beat in the script is hit.
2. **Drift catch** — gently flag when the speaker has wandered off the
   planned path so they can choose to pull back.

NOT a pace policer. NOT a phrasing critic. Two jobs, well done.

---

## Data model

### Script artifact

One markdown file. Doubles as the human-readable speaker script AND
the machine-readable cue sheet (single source of truth). Structured
with HTML comments so the tool can parse it as data while a human
reads it as prose.

**Format:**

```markdown
---
talk_id: my_talk
total_duration_seconds: 2700
---

## Phase 1: Opening · 120s
<!-- phase_id: 1, planned_duration_seconds: 120 -->

### Beat: Hook
<!-- beat_id: 1_hook, tags: [critical, joke] -->
Open with the story. Counts as covered when audience laughs.

### Beat: Credibility
<!-- beat_id: 1_cred, tags: [critical] -->
One line on why you are the right person to talk about this.
```

### Phases

Each talk has N phases (defined in the script). Each phase:
- `id` — phase number
- `label` — human name
- `planned_duration_seconds` — time budget
- `beats` — ordered list

### Beats

Single beat type with optional tags:
- `id` — stable identifier
- `label` — short human name
- `description` — what counts as "covered"
- `tags` — optional list

### Tag vocabulary (locked)

- `critical` — must hit, recoverable late if missed
- `joke` — must hit, NOT recoverable past the moment

No other tags. Adding tags is a script change, not a code change.

---

## AI loop

### Trigger rule

Loop fires when ANY of:
- `elapsed >= 6s` AND `new_words_since_last >= 20`
- `elapsed >= 18s` (hard cap; silence is signal)
- `new_words_since_last >= 60` (burst cap)

### Per-call inputs

- Full script (sent as context each call — phases, beats, tags)
- Recent transcript chunk (~last 30s of speech)
- Persisted state from previous cycle

### Per-call outputs (JSON)

```json
{
  "state": {
    "current_phase": 2,
    "beats_covered": ["1_hook", "1_cred"],
    "beats_remaining": ["2_problem", "2_evidence"]
  },
  "whisper": "gugsi joke now",
  "urgency": "medium"
}
```

### State persistence

AI returns state each cycle; we persist it to disk; feed it back next cycle.
Robust to ad-libs and skips. On crash/restart: auto-resume from last
persisted state file, no prompt.

---

## Drift detection (hybrid rule)

Drift fires when ANY of:

1. **Out-of-order** — speaker covered a beat from a later phase before
   completing the current phase's beats.
   → Whisper: `back to phase three`

2. **Multi-skip** — ≥2 consecutive beats uncovered while a later beat
   was covered.
   → Whisper: `you skipped gugsi`

3. **Phase overrun** — current phase exceeded `planned_duration + 60s`
   AND beats remain uncovered.
   → Whisper: `wrap, move to phase four`

---

## Whisper behaviour

### Style

- **Length**: 3–6 words.
- **Tone**: cryptic imperative.
  Examples: `gugsi joke now`, `wrap phase three`, `back to plan`
- **Voice**: macOS `say` default voice. Embraces the bit.
- NO emoji. NO full sentences. NO supportive tone.

### Repeat behaviour

- 2 attempts max per beat.
- Initial whisper → one re-fire ~30s later if still uncovered → suppress.
- Re-fire uses marginally higher urgency: `STILL gugsi` not `gugsi joke now`.

### Global rate cap

Minimum 15s between any two whispers. If multiple triggers fire simultaneously,
AI picks most urgent; others re-evaluate naturally next cycle.

---

## Display (TUI)

A Bubble Tea window on the laptop:

- **Status header (fixed)**: current phase, beats covered/total, beats remaining
- **Transcript pane (full scrollback, autoscroll)**: every transcribed chunk
- **Whisper pane (full scrollback, autoscroll)**: every whisper fired, urgency-coloured

TUI never truncates history. Mirrored to TV from Phase 7 onwards.

---

## Failure-mode behaviour

| Failure | Behaviour |
|---|---|
| STT API failure | Show error indicator; stop calling until STT recovers; speaker keeps talking |
| Claude API / CLI failure | Keep loop running; retry with backoff; show "AI silent" indicator; no stale whispers |
| Mic input dies | Heartbeat check on audio queue; surface "no audio" warning if silent >5s |
| Malformed JSON from LLM | Retry once with "respond in valid JSON only" suffix; if still malformed, treat as null whisper |
| Bluetooth disconnect | User mutes laptop speakers manually; tool takes no automatic action |

---

## Provider abstraction (non-negotiable — see CLAUDE.md)

| Role | Default | Fallback |
|---|---|---|
| STT | OpenAI Whisper API | local faster-whisper |
| LLM | Claude CLI (`claude -p`) | OpenAI GPT / local Ollama |
| TTS | macOS `say` | ElevenLabs / espeak |

Each interface: `transcribe`, `decide`, `speak`. Config selects implementation
at startup. Loop never imports a concrete provider.

---

## Tech stack (locked)

- **Language**: Go (single static binary)
- **Audio capture**: `gordonklaus/portaudio` (CGO, requires `brew install portaudio`)
- **STT**: OpenAI Whisper API via plain HTTP
- **LLM**: Claude CLI via `os/exec` — auth is handled by the local CLI session
- **TTS**: macOS `say` via `os/exec`
- **TUI**: Bubble Tea + Lip Gloss
- **Persistence**: JSON state file written every cycle

---

## STT chunking

Fixed interval: 10-second audio window, sent every 7 seconds (3s overlap).

---

## Session / crash recovery

Every cycle writes state to disk. On restart: auto-resume from last
persisted state without prompting. If script has changed (talk_id mismatch),
warn and ask.

---

## Run model

```
OPENAI_API_KEY=xxx ./manintheear talk_plan.md
```

All session artifacts written to `./sessions/<timestamp>/`:
- `manifest.json` — session metadata
- `state.json` — latest AI state (overwritten each cycle)
- `transcript.jsonl` — append-only transcript log
- `whispers.jsonl` — append-only whisper log

---

## Build sequence (5 PR milestones — see CLAUDE.md)

1. **Project init** — `go mod init`, folder structure, script parser, `example_talk.md`
2. **Audio layer** — portaudio capture, ring buffer, WAV encoder
3. **STT + TTS providers** — Whisper API, macOS `say`, provider interfaces
4. **LLM + AI loop** — Claude CLI provider, loop trigger rules, drift detection, whisper manager
5. **TUI + integration** — Bubble Tea display, session persistence, `main.go` wiring

The grill-me skill runs on item 1 of this sequence before any code is written.

---

## Out of scope

- Pace policing
- Phrasing critique
- Multi-language support
- Post-talk replay UI (logs are written, no playback)
- Cloud deployment
