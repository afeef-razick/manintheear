package loop

import (
	"fmt"
	"time"

	"github.com/afeef-razick/manintheear/internal/script"
	"github.com/afeef-razick/manintheear/internal/session"
)

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

	// find the next phase by position, not by ID arithmetic (IDs may not be contiguous)
	next := nextPhase(s, state.CurrentPhase)
	if next != nil {
		return fmt.Sprintf("wrap, move to phase %d", next.ID)
	}
	return "wrap it up"
}

func nextPhase(s *script.Script, currentID int) *script.Phase {
	for i, p := range s.Phases {
		if p.ID == currentID && i+1 < len(s.Phases) {
			return &s.Phases[i+1]
		}
	}
	return nil
}

func containsStr(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
