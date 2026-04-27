package loop

import (
	"fmt"
	"time"

	"github.com/afeef-razick/manintheear/internal/script"
	"github.com/afeef-razick/manintheear/internal/session"
)

// detectDrift returns a whisper message when the speaker has clearly overrun the
// current phase budget with beats still uncovered, or an empty string otherwise.
func detectDrift(s *script.Script, state session.State, phaseStart time.Time) string {
	if state.CurrentPhase == 0 {
		return ""
	}
	phase := s.PhaseByID(state.CurrentPhase)
	if phase == nil {
		return ""
	}

	overrun := time.Duration(phase.PlannedDurationSeconds+60) * time.Second
	if time.Since(phaseStart) < overrun {
		return ""
	}

	var uncovered int
	for _, beat := range phase.Beats {
		if !containsStr(state.BeatsCovered, beat.ID) {
			uncovered++
		}
	}
	if uncovered == 0 {
		return ""
	}

	next := s.PhaseByID(state.CurrentPhase + 1)
	if next != nil {
		return fmt.Sprintf("wrap, move to phase %d", next.ID)
	}
	return "wrap it up"
}

func containsStr(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
