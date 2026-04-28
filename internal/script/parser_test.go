package script_test

import (
	"testing"

	"github.com/afeef-razick/manintheear/internal/script"
)

const testScript = "testdata/example_talk.md"

func mustParse(t *testing.T) *script.Script {
	t.Helper()
	s, err := script.Parse(testScript)
	if err != nil {
		t.Fatalf("Parse() unexpected error: %v", err)
	}
	return s
}

func TestParse_HappyPath(t *testing.T) {
	s := mustParse(t)

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
	s := mustParse(t)

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
	if len(p.Points) != 2 {
		t.Errorf("Phase[0] point count = %d, want 2", len(p.Points))
	}
}

func TestParse_PointFields(t *testing.T) {
	s := mustParse(t)

	hook := s.PointByID("1_hook")
	if hook == nil {
		t.Fatal("PointByID(1_hook) returned nil")
	}
	if hook.Label != "Hook" {
		t.Errorf("point Label = %q, want %q", hook.Label, "Hook")
	}
	if hook.Description == "" {
		t.Error("point Description is empty")
	}
}

func TestParse_PointTags(t *testing.T) {
	s := mustParse(t)

	hook := s.PointByID("1_hook")
	if hook == nil {
		t.Fatal("PointByID(1_hook) returned nil")
	}
	if !hook.HasTag("critical") {
		t.Error("1_hook missing tag 'critical'")
	}
	if !hook.HasTag("joke") {
		t.Error("1_hook missing tag 'joke'")
	}

	evidence := s.PointByID("2_evidence")
	if evidence == nil {
		t.Fatal("PointByID(2_evidence) returned nil")
	}
	if len(evidence.Tags) != 0 {
		t.Errorf("2_evidence expected no tags, got %v", evidence.Tags)
	}
}

func TestParse_PointByID_Unknown(t *testing.T) {
	s := mustParse(t)
	if s.PointByID("no_such_id") != nil {
		t.Error("PointByID with unknown id should return nil")
	}
}

func TestParse_AllPoints(t *testing.T) {
	s := mustParse(t)

	if len(s.AllPoints()) != 4 {
		t.Errorf("AllPoints() count = %d, want 4", len(s.AllPoints()))
	}
}

func TestParse_PhaseByID(t *testing.T) {
	s := mustParse(t)

	p := s.PhaseByID(2)
	if p == nil {
		t.Fatal("PhaseByID(2) returned nil")
	}
	if p.Label != "Problem" {
		t.Errorf("PhaseByID(2).Label = %q, want %q", p.Label, "Problem")
	}
	if s.PhaseByID(99) != nil {
		t.Error("PhaseByID with unknown id should return nil")
	}
}

func TestParse_PhaseForPoint(t *testing.T) {
	s := mustParse(t)

	p := s.PhaseForPoint("2_problem")
	if p == nil {
		t.Fatal("PhaseForPoint returned nil")
	}
	if p.ID != 2 {
		t.Errorf("PhaseForPoint(2_problem).ID = %d, want 2", p.ID)
	}
}

func TestParse_MissingFile(t *testing.T) {
	_, err := script.Parse("testdata/nonexistent.md")
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}

func TestParse_MissingPointID(t *testing.T) {
	_, err := script.Parse("testdata/missing_point_id.md")
	if err == nil {
		t.Error("expected error for missing point_id, got nil")
	}
}
