package loop

import (
	"strings"

	"github.com/afeef-razick/manintheear/internal/script"
)

func countWords(text string) int {
	return len(strings.Fields(text))
}

func pointIDs(points []script.Point) []string {
	ids := make([]string, len(points))
	for i, p := range points {
		ids[i] = p.ID
	}
	return ids
}

func windowWords(window []transcriptChunk) int {
	n := 0
	for _, c := range window {
		n += c.words
	}
	return n
}

// isHallucination detects Whisper's known silence hallucinations.
// Whisper fills silence with phrases like "Thank you for watching" or
// non-Latin text from unrelated training data. Very short outputs (< 5 words)
// are also treated as hallucinations — real speech almost always produces more.
func isHallucination(text string) bool {
	if countWords(text) < 5 {
		return true
	}
	lower := strings.ToLower(strings.TrimSpace(text))
	known := []string{
		"thank you for watching",
		"subtitles by",
		"subtitled by",
		"transcribed by",
		"www.",
		"http",
		"like and subscribe",
	}
	for _, h := range known {
		if strings.Contains(lower, h) {
			return true
		}
	}
	// Non-ASCII characters indicate wrong-language hallucination in an English talk.
	for _, r := range text {
		if r > 127 {
			return true
		}
	}
	return false
}
