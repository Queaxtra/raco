package render

import (
	"fmt"
	"raco/model"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	assertionPassStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("42")).
				Bold(true)

	assertionFailStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("196")).
				Bold(true)

	assertionLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252"))
)

func Assertions(results []model.AssertionResult, width int) string {
	if len(results) == 0 {
		return ""
	}

	var content strings.Builder

	content.WriteString(lipgloss.NewStyle().
		Foreground(lipgloss.Color("255")).
		Bold(true).
		Render("Assertions"))
	content.WriteString("\n\n")

	passCount := 0
	failCount := 0

	for _, result := range results {
		if result.Passed {
			passCount++
			content.WriteString(assertionPassStyle.Render("✓ "))
		}
		if !result.Passed {
			failCount++
			content.WriteString(assertionFailStyle.Render("✗ "))
		}

		content.WriteString(assertionLabelStyle.Render(result.Message))
		content.WriteString("\n")
	}

	content.WriteString("\n")
	summary := fmt.Sprintf("Passed: %d | Failed: %d", passCount, failCount)
	content.WriteString(assertionLabelStyle.Render(summary))

	return lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1, 2).
		Width(width - 4).
		Render(content.String())
}
