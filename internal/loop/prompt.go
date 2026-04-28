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

	sb.WriteString("TALK POINTS (in presentation order):\n")
	all := s.AllPoints()
	for i, pt := range all {
		tags := ""
		if len(pt.Tags) > 0 {
			tags = " [" + strings.Join(pt.Tags, ", ") + "]"
		}
		fmt.Fprintf(&sb, "%d. [%s] %s%s: %s\n", i+1, pt.ID, pt.Label, tags, pt.Description)
	}

	sb.WriteString("\nALREADY COVERED: ")
	if len(state.PointsCovered) == 0 {
		sb.WriteString("none\n")
	} else {
		sb.WriteString(strings.Join(state.PointsCovered, ", ") + "\n")
	}

	sb.WriteString("STILL TO COVER (in order, first = next expected): ")
	if len(state.PointsRemaining) == 0 {
		sb.WriteString("none\n")
	} else {
		sb.WriteString(strings.Join(state.PointsRemaining, ", ") + "\n")
	}

	sb.WriteString("\nRECENT TRANSCRIPT (last ~30s):\n")
	cutoff := time.Now().Add(-30 * time.Second)
	for _, c := range window {
		if c.at.After(cutoff) {
			sb.WriteString(c.text)
			sb.WriteString(" ")
		}
	}
	sb.WriteString("\n")

	sb.WriteString(`
RULES:
1. A point is covered ONLY when the speaker has clearly and substantively addressed it — multiple sentences directly on that topic. A single word, passing mention, or partial reference does NOT count. Be conservative.
2. Suggest a whisper ONLY for the next 1–2 uncovered points (the first items in "still to cover"). Do NOT suggest points further ahead — that would confuse the speaker.
3. If the speaker has clearly moved past a point by covering 3 or more later points without addressing it, silently remove it from "still to cover". Do not whisper about it — it's too late.
4. If the transcript is fewer than 15 words or clearly not real speech, output whisper: null.
5. urgency: "high" for critical points being skipped, "medium" for moderate drift, "low" otherwise.

WHISPER QUALITY — this is critical:
- The whisper must be specific enough that the speaker instantly knows what to say. Generic labels like "ask the question" or "frame the session" are useless.
- Pull the KEY specific content from the point description. Examples:
  - BAD: "ask the room the question"
  - GOOD: "ask: what's the most popular tool in 2026?"
  - BAD: "frame the session now"
  - GOOD: "say: this is a discussion, not a talk"
  - BAD: "mention the earphone"
  - GOOD: "earphone: AI is whispering to you"
- 6–12 words. Direct imperative. Include the specific phrase or idea, not just the topic name.
- Set whisper_point_id to the ID of the point being reminded about (used to prevent duplicate firings).

Respond in valid JSON only (no markdown fences):
{"points_covered":["id",...],"points_remaining":["id",...],"whisper":<string or null>,"whisper_point_id":"<point_id or empty>","urgency":"<low|medium|high>"}
`)

	return sb.String()
}
