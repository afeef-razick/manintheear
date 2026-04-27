package loop

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/afeef-razick/manintheear/internal/script"
	"github.com/afeef-razick/manintheear/internal/session"
)

var logger = slog.Default().With("package", "loop")

type Loop struct {
	script *script.Script
	stt    STTProvider
	llm    LLMProvider
	tts    TTSProvider
	sess   *session.Session
}

func New(s *script.Script, stt STTProvider, llm LLMProvider, tts TTSProvider, sess *session.Session) (*Loop, error) {
	if s == nil || stt == nil || llm == nil || tts == nil || sess == nil {
		return nil, fmt.Errorf("loop: all dependencies are required")
	}
	return &Loop{script: s, stt: stt, llm: llm, tts: tts, sess: sess}, nil
}

func (l *Loop) Run(ctx context.Context, audioCh <-chan []byte, updateCh chan<- Update) error {
	state, err := l.sess.LoadState()
	if err != nil {
		return fmt.Errorf("loop: load state: %w", err)
	}
	if state == nil {
		state = &session.State{
			CurrentPhase:   1,
			BeatsCovered:   []string{},
			BeatsRemaining: beatIDs(l.script.AllBeats()),
		}
	}

	tr := newTrigger()
	wm := newWhisperManager()
	var window []transcriptChunk
	phaseStart := time.Now()

	st := Status{Activity: ActivityListening}

	// fires LLM even during silence, honouring the 18s hard cap when no audio arrives
	silentCheck := time.NewTicker(time.Second)
	defer silentCheck.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case audio := <-audioCh:
			st.Activity = ActivityTranscribing
			sendUpdate(updateCh, Update{State: *state, Status: currentStatus(st, tr, wm)})

			text, err := l.stt.Transcribe(ctx, audio)
			st.Activity = ActivityListening
			if err != nil {
				logger.Warn("stt error", "err", err)
				st.LastErr = "STT: " + shortErr(err)
				sendUpdate(updateCh, Update{State: *state, Status: currentStatus(st, tr, wm)})
				continue
			}
			st.LastSTTAt = time.Now()
			st.LastErr = ""

			if text == "" {
				continue
			}

			chunk := transcriptChunk{text: text, at: time.Now(), words: countWords(text)}
			if err := l.sess.AppendTranscript(session.TranscriptEntry{
				Timestamp: chunk.at,
				Text:      text,
				WordCount: chunk.words,
			}); err != nil {
				logger.Warn("transcript append failed", "err", err)
			}
			window = pruneWindow(append(window, chunk))
			tr.add(chunk.words)

			if !tr.shouldFire() {
				sendUpdate(updateCh, Update{State: *state, LastTranscript: text, Status: currentStatus(st, tr, wm)})
				continue
			}
			state, phaseStart = l.fire(ctx, state, window, wm, tr, phaseStart, updateCh, &st)
			tr.reset()
			sendUpdate(updateCh, Update{State: *state, LastTranscript: text, Status: currentStatus(st, tr, wm)})

		case <-silentCheck.C:
			// always send a status tick so the TUI timers stay fresh
			sendUpdate(updateCh, Update{State: *state, Status: currentStatus(st, tr, wm)})
			if tr.shouldFire() {
				state, phaseStart = l.fire(ctx, state, window, wm, tr, phaseStart, updateCh, &st)
				tr.reset()
				sendUpdate(updateCh, Update{State: *state, Status: currentStatus(st, tr, wm)})
			}
		}
	}
}

func (l *Loop) fire(
	ctx context.Context,
	state *session.State,
	window []transcriptChunk,
	wm *whisperManager,
	tr *trigger,
	phaseStart time.Time,
	updateCh chan<- Update,
	st *Status,
) (*session.State, time.Time) {
	elapsed := time.Since(phaseStart)
	logger.LogAttrs(ctx, slog.LevelDebug, "loop trigger",
		slog.Int64("elapsed_ms", elapsed.Milliseconds()),
		slog.Int("window_chunks", len(window)),
	)

	st.Activity = ActivityDeciding
	sendUpdate(updateCh, Update{State: *state, Status: currentStatus(*st, tr, wm)})

	prompt := buildPrompt(l.script, *state, window)
	raw, err := l.llm.Decide(ctx, prompt)
	st.Activity = ActivityListening
	if err != nil {
		logger.Warn("llm error", "err", err)
		st.LastErr = "LLM: " + shortErr(err)
		return state, phaseStart
	}
	st.LastErr = ""

	resp, err := parseResponse(raw)
	if err != nil {
		logger.Warn("llm malformed json, retrying", "err", err)
		raw2, err2 := l.llm.Decide(ctx, prompt+"\n\nRespond in valid JSON only. No markdown.")
		if err2 != nil {
			logger.Error("llm retry failed", "err", err2)
			st.LastErr = "LLM retry: " + shortErr(err2)
			return state, phaseStart
		}
		resp, err = parseResponse(raw2)
		if err != nil {
			logger.Error("llm malformed json after retry", "err", err)
			st.LastErr = "LLM parse failed"
			return state, phaseStart
		}
	}
	st.LastLLMAt = time.Now()

	newState := resp.State
	if err := l.sess.SaveState(newState); err != nil {
		logger.Warn("state save failed", "err", err)
	}

	if newState.CurrentPhase != state.CurrentPhase {
		logger.Info("phase transition", "phase", newState.CurrentPhase)
		phaseStart = time.Now()
	}
	for _, id := range newState.BeatsCovered {
		if !containsStr(state.BeatsCovered, id) {
			logger.Info("beat covered", "beat_id", id, "phase", newState.CurrentPhase)
		}
	}

	whisperText := ""
	if resp.Whisper != nil && *resp.Whisper != "" {
		whisperText = *resp.Whisper
	}
	drift := detectDrift(l.script, newState, phaseStart)
	if drift != "" {
		whisperText = drift
	}

	if whisperText != "" && wm.canSpeak(whisperText) {
		spoken := wm.resolve(whisperText)
		attempt := wm.attempts[whisperText] + 1
		st.Activity = ActivitySpeaking
		sendUpdate(updateCh, Update{State: newState, Whisper: spoken, Urgency: resp.Urgency, Status: currentStatus(*st, tr, wm)})
		if err := l.tts.Speak(ctx, spoken); err != nil {
			logger.Warn("tts error", "err", err)
		} else {
			logger.Info("whisper fired", "attempt", attempt, "urgency", resp.Urgency)
			_ = l.sess.AppendWhisper(session.WhisperEntry{
				Timestamp: time.Now(),
				Text:      spoken,
				Urgency:   resp.Urgency,
			})
			wm.record(whisperText)
		}
		st.Activity = ActivityListening
	}

	return &newState, phaseStart
}

func currentStatus(st Status, tr *trigger, wm *whisperManager) Status {
	st.WordsSince = tr.wordsSince
	st.WhisperBlockedMs = wm.timeUntilReady().Milliseconds()
	return st
}

func shortErr(err error) string {
	// OpenAI 429 bodies are verbose; cap to avoid TUI status-bar overflow
	runes := []rune(err.Error())
	if len(runes) > 60 {
		return string(runes[:60]) + "…"
	}
	return string(runes)
}

func parseResponse(raw string) (*aiResponse, error) {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "```") {
		if idx := strings.Index(raw, "\n"); idx != -1 {
			raw = raw[idx+1:]
		}
		raw = strings.TrimSuffix(strings.TrimSpace(raw), "```")
	}
	var resp aiResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return &resp, nil
}

// pruneWindow drops chunks older than 35s to bound memory over long sessions.
// 35s (not 30s) retains a small buffer beyond the prompt's 30s filter window.
func pruneWindow(window []transcriptChunk) []transcriptChunk {
	cutoff := time.Now().Add(-35 * time.Second)
	for len(window) > 0 && window[0].at.Before(cutoff) {
		window = window[1:]
	}
	return window
}

func sendUpdate(ch chan<- Update, u Update) {
	select {
	case ch <- u:
	default:
	}
}
