package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/afeef-razick/manintheear/internal/loop"
	"github.com/afeef-razick/manintheear/internal/script"
	"github.com/afeef-razick/manintheear/internal/session"
)

func testScript(t *testing.T) *script.Script {
	t.Helper()
	s, err := script.Parse("../../internal/script/testdata/example_talk.md")
	if err != nil {
		t.Fatalf("script.Parse() error: %v", err)
	}
	return s
}

func TestModel_UpdateAppliestranscript(t *testing.T) {
	ch := make(chan loop.Update, 1)
	m := newModel(testScript(t), ch, "/tmp")

	updated, _ := m.Update(updateMsg(loop.Update{LastTranscript: "hello world"}))
	got := updated.(model)
	if len(got.transcript) != 1 || got.transcript[0] != "hello world" {
		t.Errorf("transcript = %v, want [hello world]", got.transcript)
	}
}

func TestModel_UpdateAppliesWhisper(t *testing.T) {
	ch := make(chan loop.Update, 1)
	m := newModel(testScript(t), ch, "/tmp")

	updated, _ := m.Update(updateMsg(loop.Update{Whisper: "tell the joke", Urgency: "high"}))
	got := updated.(model)
	if len(got.whispers) != 1 || got.whispers[0].text != "tell the joke" {
		t.Errorf("whispers = %v, want [{tell the joke high}]", got.whispers)
	}
}

func TestModel_UpdateCapsTranscriptAt50(t *testing.T) {
	ch := make(chan loop.Update, 1)
	m := newModel(testScript(t), ch, "/tmp")

	for i := 0; i < 55; i++ {
		updated, _ := m.Update(updateMsg(loop.Update{LastTranscript: "line"}))
		m = updated.(model)
	}
	if len(m.transcript) != 50 {
		t.Errorf("transcript len = %d, want 50", len(m.transcript))
	}
}

func TestModel_UpdateStateTransition(t *testing.T) {
	ch := make(chan loop.Update, 1)
	m := newModel(testScript(t), ch, "/tmp")

	newState := session.State{CurrentPhase: 2, PointsCovered: []string{"1_hook"}}
	updated, _ := m.Update(updateMsg(loop.Update{State: newState}))
	got := updated.(model)
	if got.state.CurrentPhase != 2 {
		t.Errorf("CurrentPhase = %d, want 2", got.state.CurrentPhase)
	}
}

func TestModel_QuitOnKeyQ(t *testing.T) {
	ch := make(chan loop.Update)
	m := newModel(testScript(t), ch, "/tmp")
	m.width = 80
	m.height = 24

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("expected quit command from 'q' key")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("cmd() = %T, want tea.QuitMsg", msg)
	}
}

func TestModel_ViewRendersPhaseInfo(t *testing.T) {
	ch := make(chan loop.Update)
	m := newModel(testScript(t), ch, "/tmp")
	m.width = 80
	m.height = 24
	m.state = session.State{CurrentPhase: 1}

	view := m.View()
	if view == "" {
		t.Error("View() returned empty string")
	}
}

func TestWaitForUpdate_ClosedChannelReturnsQuitMsg(t *testing.T) {
	ch := make(chan loop.Update)
	close(ch)

	cmd := waitForUpdate(ch)
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("closed channel: got %T, want tea.QuitMsg", msg)
	}
}
