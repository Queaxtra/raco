package notification

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Msg string

type HideMsg struct{}

type State struct {
	Text    string
	Visible bool
}

func New() State {
	return State{
		Text:    "",
		Visible: false,
	}
}

func (s *State) Show(text string) {
	s.Text = text
	s.Visible = true
}

func (s *State) Hide() {
	s.Visible = false
}

func ShowCmd(text string) tea.Cmd {
	return func() tea.Msg {
		return Msg(text)
	}
}

func HideCmd() tea.Cmd {
	return tea.Tick(2*time.Second, func(_ time.Time) tea.Msg {
		return HideMsg{}
	})
}

func (s *State) Render(width int) string {
	if !s.Visible {
		return ""
	}

	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("42")).
		Background(lipgloss.Color("235")).
		Bold(true).
		Padding(0, 1)

	notification := style.Render(s.Text)

	return "\n" + lipgloss.PlaceHorizontal(width, lipgloss.Right, notification)
}
