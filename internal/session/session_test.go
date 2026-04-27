package session_test

import (
	"testing"

	"github.com/afeef-razick/manintheear/internal/session"
)

func TestNew_CreatesManifest(t *testing.T) {
	dir := t.TempDir()
	sess, err := session.New(dir, "test_talk")
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	if sess.Dir() == "" {
		t.Error("Dir() is empty")
	}
}

func TestSaveAndLoadState(t *testing.T) {
	dir := t.TempDir()
	sess, err := session.New(dir, "test_talk")
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	want := session.State{
		CurrentPhase:   2,
		BeatsCovered:   []string{"1_hook", "1_cred"},
		BeatsRemaining: []string{"2_problem"},
	}
	if err := sess.SaveState(want); err != nil {
		t.Fatalf("SaveState() error: %v", err)
	}

	got, err := sess.LoadState()
	if err != nil {
		t.Fatalf("LoadState() error: %v", err)
	}
	if got == nil {
		t.Fatal("LoadState() returned nil")
	}
	if got.CurrentPhase != want.CurrentPhase {
		t.Errorf("CurrentPhase = %d, want %d", got.CurrentPhase, want.CurrentPhase)
	}
	if len(got.BeatsCovered) != len(want.BeatsCovered) {
		t.Errorf("BeatsCovered len = %d, want %d", len(got.BeatsCovered), len(want.BeatsCovered))
	}
}

func TestLoadState_NilWhenMissing(t *testing.T) {
	dir := t.TempDir()
	sess, err := session.New(dir, "test_talk")
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	st, err := sess.LoadState()
	if err != nil {
		t.Fatalf("LoadState() error: %v", err)
	}
	if st != nil {
		t.Error("LoadState() should return nil when no state.json exists")
	}
}

func TestFindLatest_ReturnsMatchingSession(t *testing.T) {
	base := t.TempDir()
	sess, err := session.New(base, "my_talk")
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	found, err := session.FindLatest(base, "my_talk")
	if err != nil {
		t.Fatalf("FindLatest() error: %v", err)
	}
	if found != sess.Dir() {
		t.Errorf("FindLatest() = %q, want %q", found, sess.Dir())
	}
}

func TestFindLatest_EmptyWhenNoMatch(t *testing.T) {
	base := t.TempDir()
	_, err := session.New(base, "talk_a")
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	found, err := session.FindLatest(base, "talk_b")
	if err != nil {
		t.Fatalf("FindLatest() error: %v", err)
	}
	if found != "" {
		t.Errorf("FindLatest() = %q, want empty string", found)
	}
}

func TestResume_FailsWithoutManifest(t *testing.T) {
	dir := t.TempDir()
	_, err := session.Resume(dir)
	if err == nil {
		t.Error("Resume() expected error when manifest missing")
	}
}
