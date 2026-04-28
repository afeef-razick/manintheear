package loop

import "time"

const (
	// 15s prevents rapid-fire interruptions that would distract the speaker mid-sentence.
	rateCap = 15 * time.Second
	// 2 attempts: initial whisper + one refire if still uncovered, then suppress to avoid nagging.
	maxAttempts = 2
)

type whisperManager struct {
	lastSpoken time.Time
	attempts   map[string]int // keyed by point ID, not text (prevents repeated firings for the same point with varied wording)
}

func newWhisperManager() *whisperManager {
	return &whisperManager{attempts: make(map[string]int)}
}

// key returns the deduplication key: point ID when available, otherwise the whisper text.
func (w *whisperManager) key(pointID, text string) string {
	if pointID != "" {
		return pointID
	}
	return text
}

func (w *whisperManager) canSpeak(pointID, text string) bool {
	if time.Since(w.lastSpoken) < rateCap {
		return false
	}
	return w.attempts[w.key(pointID, text)] < maxAttempts
}

// resolve returns the text to speak, prefixing with "again: " on the second attempt.
func (w *whisperManager) resolve(pointID, text string) string {
	if w.attempts[w.key(pointID, text)] == 1 {
		return "again: " + text
	}
	return text
}

func (w *whisperManager) record(pointID, text string) {
	w.lastSpoken = time.Now()
	w.attempts[w.key(pointID, text)]++
}

func (w *whisperManager) timeUntilReady() time.Duration {
	remaining := rateCap - time.Since(w.lastSpoken)
	if remaining < 0 {
		return 0
	}
	return remaining
}
