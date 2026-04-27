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

var logger = slog.Default().With("pkg", "loop")

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

	// silentCheck fires the LLM even when no audio arrives, honouring the 18s hard cap.
	silentCheck := time.NewTicker(time.Second)
	defer silentCheck.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case audio := <-audioCh:
			text, err := l.stt.Transcribe(ctx, audio)
			if err != nil {
				logger.Warn("stt error", "err", err)
				continue
			}
			if text == "" {
				continue
			}
			chunk := transcriptChunk{text: text, at: time.Now(), words: countWords(text)}
			_ = l.sess.AppendTranscript(session.TranscriptEntry{
				Timestamp: chunk.at,
				Text:      text,
				WordCount: chunk.words,
			})
			window = append(window, chunk)
			tr.add(chunk.words)

			if !tr.shouldFire() {
				sendUpdate(updateCh, Update{State: *state, LastTranscript: text})
				continue
			}
			state, phaseStart = l.fire(ctx, state, window, wm, phaseStart)
			tr.reset()
			sendUpdate(updateCh, Update{State: *state, LastTranscript: text})

		case <-silentCheck.C:
			if tr.shouldFire() {
				state, phaseStart = l.fire(ctx, state, window, wm, phaseStart)
				tr.reset()
				sendUpdate(updateCh, Update{State: *state})
			}
		}
	}
}

func (l *Loop) fire(
	ctx context.Context,
	state *session.State,
	window []transcriptChunk,
	wm *whisperManager,
	phaseStart time.Time,
) (*session.State, time.Time) {
	prompt := buildPrompt(l.script, *state, window)

	raw, err := l.llm.Decide(ctx, prompt)
	if err != nil {
		logger.Warn("llm error", "err", err)
		return state, phaseStart
	}

	resp, err := parseResponse(raw)
	if err != nil {
		// one retry with explicit JSON instruction
		raw2, err2 := l.llm.Decide(ctx, prompt+"\n\nRespond in valid JSON only. No markdown.")
		if err2 != nil {
			logger.Warn("llm retry error", "err", err2)
			return state, phaseStart
		}
		resp, err = parseResponse(raw2)
		if err != nil {
			logger.Warn("llm parse error after retry", "err", err)
			return state, phaseStart
		}
	}

	newState := resp.State
	_ = l.sess.SaveState(newState)

	if newState.CurrentPhase != state.CurrentPhase {
		phaseStart = time.Now()
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
		if err := l.tts.Speak(ctx, spoken); err != nil {
			logger.Warn("tts error", "err", err)
		} else {
			_ = l.sess.AppendWhisper(session.WhisperEntry{
				Timestamp: time.Now(),
				Text:      spoken,
				Urgency:   resp.Urgency,
			})
			wm.record(whisperText)
		}
	}

	return &newState, phaseStart
}

func parseResponse(raw string) (*aiResponse, error) {
	raw = strings.TrimSpace(raw)
	// strip markdown fences if present
	if strings.HasPrefix(raw, "```") {
		lines := strings.SplitN(raw, "\n", 2)
		if len(lines) == 2 {
			raw = lines[1]
		}
		raw = strings.TrimSuffix(strings.TrimSpace(raw), "```")
	}
	var resp aiResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return &resp, nil
}

func sendUpdate(ch chan<- Update, u Update) {
	select {
	case ch <- u:
	default:
	}
}
