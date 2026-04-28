# Man-in-the-Ear Tool — Product Spec

> Agent process directives live in `CLAUDE.md`. Read that first.

## What the tool does

Listens to a live talk via laptop mic. Whispers short reminders into a
bluetooth earphone if the speaker is missing points or drifting off the
planned path. Has the talk script loaded as context. Runs locally for
the duration of one talk.

## Two jobs only

1. **Coverage** — make sure every important point in the script is hit.
2. **Drift catch** — gently flag when the speaker has skipped past something
   important so they can choose to address it.

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

### Point: Hook
<!-- point_id: 1_hook, tags: [critical, joke] -->
Open with the story. Counts as covered when audience laughs.

### Point: Credibility
<!-- point_id: 1_cred, tags: [critical] -->
One line on why you are the right person to talk about this.
```

### Phases

Phases are **organisational only** — they group points for human readability
and TUI display. They play no role in AI coverage tracking or whisper decisions.

Each phase:
- `id` — phase number
- `label` — human name
- `planned_duration_seconds` — time budget (display only)
- `points` — ordered list

### Points

The atomic unit the tool tracks. A point is a single distinct moment that
needs to be made during the talk.

- `id` — stable identifier (e.g. `1_hook`)
- `label` — short human name
- `description` — what the speaker needs to say; also what "covered" means
- `tags` — optional list

### Tag vocabulary (locked)

- `critical` — must hit; recoverable slightly late if missed
- `joke` — must hit; NOT recoverable past the moment

No other tags. Adding tags is a script change, not a code change.

---

## AI loop

### When the loop fires

Loop fires when ANY of:
- `elapsed >= 6s` AND `new_words_since_last >= 20`
- `elapsed >= 18s` (hard cap; silence is signal)
- `new_words_since_last >= 60` (burst cap)

The loop does NOT fire until the speaker has produced at least 15 words of
real transcript. Pre-talk setup time is ignored entirely.

### Per-call inputs

- Full ordered list of all talk points (flat, no phase grouping in the AI context)
- Which points have been covered so far
- Which points remain (in presentation order)
- Recent transcript chunk (~last 30s of speech)

### Per-call outputs (JSON)

```json
{
  "points_covered": ["1_hook", "1_cred"],
  "points_remaining": ["2_problem", "2_evidence"],
  "whisper": "ask: what's the most popular tool in 2026?",
  "whisper_point_id": "2_question",
  "urgency": "medium"
}
```

### Coverage conservatism

A point is only marked covered when the speaker has **clearly and substantively
addressed it** — multiple sentences directly on that topic. A single word,
passing mention, or partial reference does not count. The AI errs on the side
of under-marking.

### Point skipping

If the speaker has covered 3 or more later points without addressing an earlier
one, that earlier point is silently dropped from tracking — it is too late for
a reminder to help. The tool does not nag about points the speaker has clearly
moved past.

### State persistence

AI returns state each cycle; we persist it to disk; feed it back next cycle.
Robust to ad-libs and skips. On crash/restart: auto-resume from last
persisted state file.

---

## Drift / reminder logic

Drift is handled **entirely by the AI**, not by Go-side timing rules.

The AI is given the full ordered point list and knows where the speaker is.
It whispers about the **next 1–2 uncovered points** near the speaker's current
position. It does not suggest points further ahead (which would confuse the
speaker) and does not raise points the speaker has moved past.

Phase timing (planned_duration_seconds) is not used as a trigger for drift
reminders. Phase overrun alone is not a useful signal.

---

## Whisper behaviour

### Style

- **Length**: 6–12 words.
- **Tone**: specific, direct imperative. Includes the **actual content**, not
  just the topic label.
- Good: `ask: what's the most popular tool in 2026?`
- Good: `say: this is a discussion, not a talk`
- Good: `earphone: AI is listening and whispering to you`
- Bad: `ask the question now` (too vague — which question?)
- Bad: `frame the session` (too generic — says nothing actionable)
- **Voice**: macOS `say` default voice.
- NO emoji. NO full sentences. NO supportive tone.

### Repeat behaviour

- 2 attempts max **per point** (keyed by point ID, not by whisper text).
  This prevents repeated firings when the AI words the same reminder
  slightly differently across cycles.
- Initial whisper → one re-fire if still uncovered → suppress.
- Re-fire prefixed with `again:` to distinguish from the first attempt.

### Global rate cap

Minimum 15s between any two whispers. If multiple triggers fire simultaneously,
AI picks most urgent; others re-evaluate naturally next cycle.

---

## Display (TUI)

A Bubble Tea window on the laptop:

- **Status header (fixed)**: current phase, points covered/total, points remaining
- **Status bar**: current activity (listening / transcribing / deciding / speaking),
  word count since last LLM fire, time since last STT and LLM success, whisper
  readiness (ready or blocked Xs)
- **Transcript pane**: every transcribed chunk
- **Whisper pane**: every whisper fired, urgency-coloured (red=high, yellow=medium, green=low)

TUI never truncates history within a session.

---

## Failure-mode behaviour

| Failure | Behaviour |
|---|---|
| STT API failure | Show error in status bar; keep loop running |
| LLM provider failure | Keep loop running; retry once; show error in status bar |
| Malformed JSON from LLM | Retry once with "respond in valid JSON only" suffix; if still malformed, treat as null whisper |
| Bluetooth disconnect | User handles manually; tool takes no automatic action |

---

## STT noise filtering

The STT model can hallucinate content during silence — common patterns include
repetitive short phrases and non-English text. The tool discards transcripts
that match known hallucination signatures before they reach the AI. Very short
outputs (fewer than 5 words) are treated as noise.

---

## Provider abstraction (non-negotiable — see CLAUDE.md)

| Role | Default | Fallback |
|---|---|---|
| STT | OpenAI Whisper API | local faster-whisper |
| LLM | Local AI CLI (authenticated session) | OpenAI GPT / local Ollama |
| TTS | macOS `say` | ElevenLabs / espeak |

Each interface: `transcribe`, `decide`, `speak`. Config selects implementation
at startup. Loop never imports a concrete provider.

---

## Tech stack (locked)

- **Language**: Go (single static binary)
- **Audio capture**: `gordonklaus/portaudio` (CGO, requires `brew install portaudio`)
- **STT**: OpenAI Whisper API via plain HTTP
- **LLM**: Local AI CLI via `os/exec` — auth is handled by the installed CLI session
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
OPENAI_API_KEY=xxx AI_CLI_CMD="codex exec --skip-git-repo-check -" ./manintheear talk_plan.md
```

All session artifacts written to `./sessions/<timestamp>/`:
- `manifest.json` — session metadata
- `state.json` — latest AI state (overwritten each cycle)
- `transcript.jsonl` — append-only transcript log
- `whispers.jsonl` — append-only whisper log
- `llm_calls.jsonl` — full prompt and response for every LLM cycle (for debugging)

---

## Out of scope

- Pace policing
- Phrasing critique
- Multi-language support
- Post-talk replay UI (logs are written, no playback)
- Cloud deployment
