package modal

import (
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

func Collection(collectionInput textinput.Model) string {
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("255")).
		Padding(1, 2).
		Width(50).
		Background(lipgloss.Color("235"))

	title := lipgloss.NewStyle().
		Foreground(lipgloss.Color("255")).
		Bold(true).
		Render("Create New Collection")

	help := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Italic(true).
		Render("Enter: Create • Esc: Cancel")

	content := title + "\n\n" + collectionInput.View() + "\n\n" + help

	return "\n" + modalStyle.Render(content)
}
