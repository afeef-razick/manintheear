package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/afeef-razick/manintheear/internal/loop"
	"github.com/afeef-razick/manintheear/internal/script"
	"github.com/afeef-razick/manintheear/internal/session"
)

var (
	headerStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	labelStyle    = lipgloss.NewStyle().Faint(true)
	dividerStyle  = lipgloss.NewStyle().Faint(true)
	statusOK      = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)  // green
	statusBusy    = lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Bold(true)  // yellow
	statusSpeak   = lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true)  // cyan
	statusErr     = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)   // red
	metaStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))              // dark grey
	metaHighlight = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))             // white
	whisperHigh   = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
	whisperMed    = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	whisperLow    = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
)

type whisperLine struct {
	text    string
	urgency string
}

type model struct {
	script     *script.Script
	state      session.State
	status     loop.Status
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
			return tea.QuitMsg{}
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
		m.status = msg.Status
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

	phase := m.script.PhaseByID(m.state.CurrentPhase)
	phaseName := "—"
	if phase != nil {
		phaseName = phase.Label
	}
	total := len(m.script.AllPoints())
	covered := len(m.state.PointsCovered)
	remaining := total - covered

	sb.WriteString(headerStyle.Render(fmt.Sprintf(
		"Phase %d: %s  |  Beats %d/%d  |  Remaining: %d",
		m.state.CurrentPhase, phaseName, covered, total, remaining,
	)))
	sb.WriteString("\n")
	sb.WriteString(labelStyle.Render(fmt.Sprintf("session: %s  |  q to quit", m.sessionDir)))
	sb.WriteString("\n")

	sb.WriteString(renderStatusBar(m.status))
	sb.WriteString("\n\n")

	paneHeight := (m.height - 10) / 2
	if paneHeight < 3 {
		paneHeight = 3
	}

	sb.WriteString(dividerStyle.Render("── transcript ─────────────────────────"))
	sb.WriteString("\n")
	lines := m.transcript
	if len(lines) > paneHeight {
		lines = lines[len(lines)-paneHeight:]
	}
	for _, line := range lines {
		sb.WriteString(line)
		sb.WriteString("\n")
	}

	sb.WriteString("\n")
	sb.WriteString(dividerStyle.Render("── whispers ────────────────────────────"))
	sb.WriteString("\n")
	wlines := m.whispers
	if len(wlines) > paneHeight {
		wlines = wlines[len(wlines)-paneHeight:]
	}
	for _, w := range wlines {
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

func renderStatusBar(st loop.Status) string {
	icon, iconStyle := activityIcon(st)

	var parts []string
	parts = append(parts, iconStyle.Render(icon+" "+string(st.Activity)))

	parts = append(parts, metaStyle.Render("words: ")+metaHighlight.Render(fmt.Sprintf("%d", st.WordsSince)))

	if !st.LastSTTAt.IsZero() {
		parts = append(parts, metaStyle.Render("stt: ")+metaHighlight.Render(relTime(st.LastSTTAt)))
	} else {
		parts = append(parts, metaStyle.Render("stt: ")+metaStyle.Render("never"))
	}

	if !st.LastLLMAt.IsZero() {
		parts = append(parts, metaStyle.Render("llm: ")+metaHighlight.Render(relTime(st.LastLLMAt)))
	} else {
		parts = append(parts, metaStyle.Render("llm: ")+metaStyle.Render("never"))
	}

	if st.WhisperBlockedMs > 0 {
		secs := (st.WhisperBlockedMs + 999) / 1000
		parts = append(parts, metaStyle.Render("whisper: ")+statusBusy.Render(fmt.Sprintf("blocked %ds", secs)))
	} else {
		parts = append(parts, metaStyle.Render("whisper: ")+statusOK.Render("ready"))
	}

	line := strings.Join(parts, metaStyle.Render("  ·  "))

	if st.LastErr != "" {
		line += "\n" + statusErr.Render("  ⚠ "+st.LastErr)
	}

	return line
}

func activityIcon(st loop.Status) (string, lipgloss.Style) {
	if st.LastErr != "" {
		return "⚠", statusErr
	}
	switch st.Activity {
	case loop.ActivityTranscribing:
		return "◎", statusBusy
	case loop.ActivityDeciding:
		return "◎", statusBusy
	case loop.ActivitySpeaking:
		return "◆", statusSpeak
	default:
		return "●", statusOK
	}
}

func relTime(t time.Time) string {
	d := time.Since(t).Round(time.Second)
	if d < time.Second {
		return "now"
	}
	return fmt.Sprintf("%ds ago", int(d.Seconds()))
}

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
