package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/afeef-razick/manintheear/internal/loop"
	"github.com/afeef-razick/manintheear/internal/script"
	"github.com/afeef-razick/manintheear/internal/session"
)

var (
	headerStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	labelStyle   = lipgloss.NewStyle().Faint(true)
	whisperHigh  = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
	whisperMed   = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	whisperLow   = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	dividerStyle = lipgloss.NewStyle().Faint(true)
)

type whisperLine struct {
	text    string
	urgency string
}

type model struct {
	script     *script.Script
	state      session.State
	transcript []string
	whispers   []whisperLine
	updateCh   <-chan loop.Update
	width      int
	height     int
	sessionDir string
}

type updateMsg loop.Update

func waitForUpdate(ch <-chan loop.Update) tea.Cmd {
	return func() tea.Msg {
		update, ok := <-ch
		if !ok {
			return tea.Quit()
		}
		return updateMsg(update)
	}
}

func newModel(s *script.Script, updateCh <-chan loop.Update, sessionDir string) model {
	return model{
		script:     s,
		updateCh:   updateCh,
		sessionDir: sessionDir,
		state:      session.State{CurrentPhase: 1},
	}
}

func (m model) Init() tea.Cmd {
	return waitForUpdate(m.updateCh)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case updateMsg:
		m.state = msg.State
		if msg.LastTranscript != "" {
			m.transcript = append(m.transcript, msg.LastTranscript)
			if len(m.transcript) > 50 {
				m.transcript = m.transcript[len(m.transcript)-50:]
			}
		}
		if msg.Whisper != "" {
			m.whispers = append(m.whispers, whisperLine{msg.Whisper, msg.Urgency})
		}
		return m, waitForUpdate(m.updateCh)

	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m model) View() string {
	if m.width == 0 {
		return "loading…"
	}

	var sb strings.Builder

	// status header
	phase := m.script.PhaseByID(m.state.CurrentPhase)
	phaseName := "—"
	if phase != nil {
		phaseName = phase.Label
	}
	total := len(m.script.AllBeats())
	covered := len(m.state.BeatsCovered)
	remaining := total - covered

	sb.WriteString(headerStyle.Render(fmt.Sprintf(
		"Phase %d: %s  |  Beats %d/%d  |  Remaining: %d",
		m.state.CurrentPhase, phaseName, covered, total, remaining,
	)))
	sb.WriteString("\n")
	sb.WriteString(labelStyle.Render(fmt.Sprintf("session: %s  |  q to quit", m.sessionDir)))
	sb.WriteString("\n\n")

	// transcript pane
	sb.WriteString(dividerStyle.Render("── transcript ─────────────────────────"))
	sb.WriteString("\n")
	transcriptLines := m.transcript
	maxLines := (m.height - 10) / 2
	if maxLines < 3 {
		maxLines = 3
	}
	if len(transcriptLines) > maxLines {
		transcriptLines = transcriptLines[len(transcriptLines)-maxLines:]
	}
	for _, line := range transcriptLines {
		sb.WriteString(line)
		sb.WriteString("\n")
	}

	sb.WriteString("\n")

	// whisper pane
	sb.WriteString(dividerStyle.Render("── whispers ────────────────────────────"))
	sb.WriteString("\n")
	maxWhispers := (m.height - 10) / 2
	if maxWhispers < 3 {
		maxWhispers = 3
	}
	whisperLines := m.whispers
	if len(whisperLines) > maxWhispers {
		whisperLines = whisperLines[len(whisperLines)-maxWhispers:]
	}
	for _, w := range whisperLines {
		var styled string
		switch w.urgency {
		case "high":
			styled = whisperHigh.Render("▶ " + w.text)
		case "medium":
			styled = whisperMed.Render("▶ " + w.text)
		default:
			styled = whisperLow.Render("▶ " + w.text)
		}
		sb.WriteString(styled)
		sb.WriteString("\n")
	}

	return sb.String()
}

// TUI wraps the Bubble Tea program for the loop display.
type TUI struct {
	script     *script.Script
	sessionDir string
}

func New(s *script.Script, sessionDir string) (*TUI, error) {
	if s == nil {
		return nil, fmt.Errorf("tui: script is required")
	}
	return &TUI{script: s, sessionDir: sessionDir}, nil
}

func (t *TUI) Run(updateCh <-chan loop.Update) error {
	m := newModel(t.script, updateCh, t.sessionDir)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
