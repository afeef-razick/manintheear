package loop

import (
	"fmt"
	"strings"
	"time"

	"github.com/afeef-razick/manintheear/internal/script"
	"github.com/afeef-razick/manintheear/internal/session"
)


func buildPrompt(s *script.Script, state session.State, window []transcriptChunk) string {
	var sb strings.Builder

	sb.WriteString("You are a real-time talk coach monitoring a live presentation.\n\n")

	sb.WriteString("SCRIPT:\n")
	for _, phase := range s.Phases {
		fmt.Fprintf(&sb, "Phase %d: %s (%ds)\n", phase.ID, phase.Label, phase.PlannedDurationSeconds)
		for _, beat := range phase.Beats {
			tags := ""
			if len(beat.Tags) > 0 {
				tags = " [" + strings.Join(beat.Tags, ", ") + "]"
			}
			fmt.Fprintf(&sb, "  - %s (id: %s)%s: %s\n", beat.Label, beat.ID, tags, beat.Description)
		}
	}

	sb.WriteString("\nCURRENT STATE:\n")
	fmt.Fprintf(&sb, "  current_phase: %d\n", state.CurrentPhase)
	fmt.Fprintf(&sb, "  beats_covered: %v\n", state.BeatsCovered)
	fmt.Fprintf(&sb, "  beats_remaining: %v\n", state.BeatsRemaining)

	sb.WriteString("\nRECENT TRANSCRIPT (last ~30s):\n")
	cutoff := time.Now().Add(-30 * time.Second)
	for _, c := range window {
		if c.at.After(cutoff) {
			sb.WriteString(c.text)
			sb.WriteString(" ")
		}
	}

	sb.WriteString("\n\nRules:\n")
	sb.WriteString("- Respond in valid JSON only, no markdown fences.\n")
	sb.WriteString("- Update beats_covered when the transcript clearly addresses a beat.\n")
	sb.WriteString("- Set whisper to null if nothing needs to be said.\n")
	sb.WriteString("- If whisper is needed: 3-6 words, cryptic imperative (e.g. 'tell the joke now').\n")
	sb.WriteString("- urgency must be: low, medium, or high.\n")
	sb.WriteString("\nRespond with exactly this JSON structure:\n")
	sb.WriteString(`{"state":{"current_phase":<int>,"beats_covered":[...],"beats_remaining":[...]},"whisper":<string or null>,"urgency":"<low|medium|high>"}`)
	sb.WriteString("\n")

	return sb.String()
}

