package render

import (
	"raco/ui/theme"

	"github.com/charmbracelet/lipgloss"
)

// StatusBar renders the single-line bottom bar: current mode on the left,
// vim-style key hints on the right. Keeps hints visible so developers can learn shortcuts.
func StatusBar(width int, mode string) string {
	left := theme.Muted().Render("raco")
	if mode != "" {
		left = theme.Muted().Render(mode)
	}

	hints := []struct{ key, desc string }{
		{"j/k", "nav"},
		{"gg/G", "top/bot"},
		{"h/l", "focus"},
		{"e", "send"},
		{"w", "save"},
		{"Tab", "next"},
		{":", "palette"},
		{"q", "quit"},
	}

	var rightParts []string
	for _, h := range hints {
		rightParts = append(rightParts, theme.KeyHint().Render(h.key)+theme.Muted().Render(" "+h.desc))
	}
	rightStr := lipgloss.JoinHorizontal(lipgloss.Top, rightParts...)
	w := width - lipgloss.Width(left) - 2
	if w < 10 {
		w = 10
	}
	right := lipgloss.NewStyle().Width(w).Align(lipgloss.Right).Render(rightStr)

	content := lipgloss.JoinHorizontal(lipgloss.Top, left, " ", right)
	return theme.StatusBar().Width(width).Render(content)
}
