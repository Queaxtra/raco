package render

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	statusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Background(lipgloss.Color("235")).
			Padding(0, 1)

	statusBarKeyStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("255")).
				Bold(true)

	statusBarSeparatorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("240"))
)

func StatusBar(width int) string {
	shortcuts := []struct {
		key  string
		desc string
	}{
		{"Ctrl+N", "New Coll."},
		{"Ctrl+W", "Save Req."},
		{"Ctrl+R", "Send"},
		{"F1", "Dashboard"},
		{"Ctrl+P", "Palette"},
		{"Tab", "Switch"},
		{"Ctrl+F", "Add File"},
		{"Ctrl+X", "Del File"},
		{"Esc", "Back"},
		{"Ctrl+C", "Quit"},
	}

	var items []string
	for _, s := range shortcuts {
		key := statusBarKeyStyle.Render(s.key)
		items = append(items, key+" "+s.desc)
	}

	separator := statusBarSeparatorStyle.Render(" ")
	content := lipgloss.JoinHorizontal(lipgloss.Left, items[0])
	for i := 1; i < len(items); i++ {
		content += separator + items[i]
	}

	return statusBarStyle.
		Width(width - 2).
		Render(content)
}
