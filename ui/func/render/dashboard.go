package render

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	dashboardBorderColor = lipgloss.Color("240")
	dashboardTitleColor  = lipgloss.Color("255")
	dashboardLabelColor  = lipgloss.Color("252")
	dashboardValueColor  = lipgloss.Color("42")
	dashboardErrorColor  = lipgloss.Color("196")

	dashboardStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(dashboardBorderColor).
			Padding(1, 2)

	dashboardTitleStyle = lipgloss.NewStyle().
				Foreground(dashboardTitleColor).
				Bold(true).
				MarginBottom(1)

	dashboardLabelStyle = lipgloss.NewStyle().
				Foreground(dashboardLabelColor)

	dashboardValueStyle = lipgloss.NewStyle().
				Foreground(dashboardValueColor).
				Bold(true)

	dashboardErrorStyle = lipgloss.NewStyle().
				Foreground(dashboardErrorColor).
				Bold(true)
)

type DashboardStats struct {
	TotalRequests   int
	SuccessCount    int
	FailureCount    int
	SuccessRate     float64
	AvgDuration     string
	MinDuration     string
	MaxDuration     string
	Sparkline       string
	SuccessRateBar  string
}

func Dashboard(width, height int, stats DashboardStats) string {
	var content strings.Builder

	content.WriteString(dashboardTitleStyle.Render("Performance Dashboard"))
	content.WriteString("\n\n")

	content.WriteString(dashboardLabelStyle.Render("Total Requests: "))
	content.WriteString(dashboardValueStyle.Render(fmt.Sprintf("%d", stats.TotalRequests)))
	content.WriteString("\n")

	content.WriteString(dashboardLabelStyle.Render("Success: "))
	content.WriteString(dashboardValueStyle.Render(fmt.Sprintf("%d", stats.SuccessCount)))
	content.WriteString(dashboardLabelStyle.Render(" | Failure: "))
	content.WriteString(dashboardErrorStyle.Render(fmt.Sprintf("%d", stats.FailureCount)))
	content.WriteString("\n\n")

	content.WriteString(dashboardLabelStyle.Render("Success Rate: "))
	content.WriteString(dashboardValueStyle.Render(fmt.Sprintf("%.1f%%", stats.SuccessRate)))
	content.WriteString("\n")
	content.WriteString(stats.SuccessRateBar)
	content.WriteString("\n\n")

	content.WriteString(dashboardLabelStyle.Render("Latency Stats"))
	content.WriteString("\n")
	content.WriteString(dashboardLabelStyle.Render("  Avg: "))
	content.WriteString(dashboardValueStyle.Render(stats.AvgDuration))
	content.WriteString(dashboardLabelStyle.Render(" | Min: "))
	content.WriteString(dashboardValueStyle.Render(stats.MinDuration))
	content.WriteString(dashboardLabelStyle.Render(" | Max: "))
	content.WriteString(dashboardValueStyle.Render(stats.MaxDuration))
	content.WriteString("\n\n")

	content.WriteString(dashboardLabelStyle.Render("Response Time Trend"))
	content.WriteString("\n")
	content.WriteString(stats.Sparkline)

	return dashboardStyle.
		Width(width - 4).
		Height(height - 4).
		Render(content.String())
}
