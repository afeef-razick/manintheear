package loop

import "time"

const (
	rateCap    = 15 * time.Second
	maxAttempts = 2
)

type whisperManager struct {
	lastSpoken time.Time
	attempts   map[string]int
}

func newWhisperManager() *whisperManager {
	return &whisperManager{attempts: make(map[string]int)}
}

func (w *whisperManager) canSpeak(text string) bool {
	if time.Since(w.lastSpoken) < rateCap {
		return false
	}
	return w.attempts[text] < maxAttempts
}

// resolve returns the text to speak, prefixing with "STILL " on the second attempt.
func (w *whisperManager) resolve(text string) string {
	if w.attempts[text] == 1 {
		return "STILL " + text
	}
	return text
}

func (w *whisperManager) record(text string) {
	w.lastSpoken = time.Now()
	w.attempts[text]++
}
