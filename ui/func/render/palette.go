package render

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

var (
	paletteStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(1, 2)

	paletteTitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("255")).
				Bold(true).
				MarginBottom(1)

	paletteItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252")).
				PaddingLeft(2)

	paletteSelectedStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("240")).
				Foreground(lipgloss.Color("255")).
				Bold(true).
				PaddingLeft(2)
)

func CommandPalette(width, height int, input textinput.Model, items []string, selectedIndex int) string {
	var content strings.Builder

	content.WriteString(paletteTitleStyle.Render("Command Palette"))
	content.WriteString("\n\n")

	content.WriteString(input.View())
	content.WriteString("\n\n")

	maxItems := 10
	if len(items) > maxItems {
		items = items[:maxItems]
	}

	if len(items) == 0 {
		content.WriteString(paletteItemStyle.Render("No matches found"))
	}

	for i, item := range items {
		if i == selectedIndex {
			content.WriteString(paletteSelectedStyle.Render(fmt.Sprintf("▶ %s", item)))
		}
		if i != selectedIndex {
			content.WriteString(paletteItemStyle.Render(fmt.Sprintf("  %s", item)))
		}
		content.WriteString("\n")
	}

	content.WriteString("\n")
	help := "↑/↓: Navigate • Enter: Select • Esc: Close"
	content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(help))

	return paletteStyle.
		Width(width - 4).
		Height(height - 4).
		Render(content.String())
}
