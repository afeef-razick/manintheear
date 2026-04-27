package script_test

import (
	"testing"

	"github.com/afeef-razick/manintheear/internal/script"
)

const testScript = "testdata/example_talk.md"

func TestParse_HappyPath(t *testing.T) {
	s, err := script.Parse(testScript)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if s.TalkID != "test_talk" {
		t.Errorf("TalkID = %q, want %q", s.TalkID, "test_talk")
	}
	if s.TotalDurationSeconds != 600 {
		t.Errorf("TotalDurationSeconds = %d, want 600", s.TotalDurationSeconds)
	}
	if len(s.Phases) != 2 {
		t.Fatalf("len(Phases) = %d, want 2", len(s.Phases))
	}
}

func TestParse_PhaseFields(t *testing.T) {
	s, _ := script.Parse(testScript)

	p := s.Phases[0]
	if p.ID != 1 {
		t.Errorf("Phase[0].ID = %d, want 1", p.ID)
	}
	if p.Label != "Opening" {
		t.Errorf("Phase[0].Label = %q, want %q", p.Label, "Opening")
	}
	if p.PlannedDurationSeconds != 120 {
		t.Errorf("Phase[0].PlannedDurationSeconds = %d, want 120", p.PlannedDurationSeconds)
	}
	if len(p.Beats) != 2 {
		t.Errorf("Phase[0] beat count = %d, want 2", len(p.Beats))
	}
}

func TestParse_BeatTags(t *testing.T) {
	s, _ := script.Parse(testScript)

	hook := s.BeatByID("1_hook")
	if hook == nil {
		t.Fatal("BeatByID(1_hook) returned nil")
	}
	if !hook.HasTag("critical") {
		t.Error("1_hook missing tag 'critical'")
	}
	if !hook.HasTag("joke") {
		t.Error("1_hook missing tag 'joke'")
	}

	evidence := s.BeatByID("2_evidence")
	if evidence == nil {
		t.Fatal("BeatByID(2_evidence) returned nil")
	}
	if len(evidence.Tags) != 0 {
		t.Errorf("2_evidence expected no tags, got %v", evidence.Tags)
	}
}

func TestParse_AllBeats(t *testing.T) {
	s, _ := script.Parse(testScript)

	beats := s.AllBeats()
	if len(beats) != 4 {
		t.Errorf("AllBeats() count = %d, want 4", len(beats))
	}
}

func TestParse_PhaseForBeat(t *testing.T) {
	s, _ := script.Parse(testScript)

	p := s.PhaseForBeat("2_problem")
	if p == nil {
		t.Fatal("PhaseForBeat returned nil")
	}
	if p.ID != 2 {
		t.Errorf("PhaseForBeat(2_problem).ID = %d, want 2", p.ID)
	}
}

func TestParse_MissingFile(t *testing.T) {
	_, err := script.Parse("testdata/nonexistent.md")
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}

func TestParse_MissingBeatID(t *testing.T) {
	bad := "testdata/missing_beat_id.md"
	_, err := script.Parse(bad)
	if err == nil {
		t.Error("expected error for missing beat_id, got nil")
	}
}
