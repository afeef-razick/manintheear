package loop

import (
	"strings"

	"github.com/afeef-razick/manintheear/internal/script"
)

func countWords(text string) int {
	return len(strings.Fields(text))
}

func beatIDs(beats []script.Beat) []string {
	ids := make([]string, len(beats))
	for i, b := range beats {
		ids[i] = b.ID
	}
	return ids
}
