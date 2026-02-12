package modal

import (
	"raco/model"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

func Request(requestNameInput textinput.Model, collections []*model.Collection, expandedIndex int) string {
	if collections == nil {
		collections = []*model.Collection{}
	}

	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("255")).
		Padding(1, 2).
		Width(50).
		Background(lipgloss.Color("235"))

	title := lipgloss.NewStyle().
		Foreground(lipgloss.Color("255")).
		Bold(true).
		Render("Save Request")

	colName := ""
	if expandedIndex >= 0 && expandedIndex < len(collections) {
		colName = collections[expandedIndex].Name
	}
	if expandedIndex < 0 && len(collections) > 0 {
		colName = collections[0].Name
	}

	targetInfo := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")).
		Render("Target: " + colName)

	help := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Italic(true).
		Render("Enter: Save • Esc: Cancel")

	content := title + "\n\n" + targetInfo + "\n\n" + requestNameInput.View() + "\n\n" + help

	return "\n" + modalStyle.Render(content)
}
