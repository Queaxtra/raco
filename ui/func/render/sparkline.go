package render

import (
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var (
	sparklineSuccessColor = lipgloss.Color("42")
	sparklineErrorColor   = lipgloss.Color("196")
	sparklineNeutralColor = lipgloss.Color("240")
)

func Sparkline(durations []time.Duration, width int) string {
	if width < 1 {
		width = 1
	}
	if len(durations) == 0 {
		return strings.Repeat("·", width)
	}

	if len(durations) > width {
		durations = durations[len(durations)-width:]
	}

	maxDuration := time.Duration(0)
	for _, d := range durations {
		if d > maxDuration {
			maxDuration = d
		}
	}

	if maxDuration == 0 {
		return strings.Repeat("·", width)
	}

	bars := []string{"▁", "▂", "▃", "▄", "▅", "▆", "▇", "█"}
	var result strings.Builder

	for _, d := range durations {
		ratio := float64(d) / float64(maxDuration)
		index := int(ratio * float64(len(bars)-1))

		if index >= len(bars) {
			index = len(bars) - 1
		}

		color := sparklineSuccessColor
		if d > maxDuration/2 {
			color = sparklineErrorColor
		}

		bar := lipgloss.NewStyle().
			Foreground(color).
			Render(bars[index])

		result.WriteString(bar)
	}

	return result.String()
}

func SuccessRateBar(successCount, totalCount, width int) string {
	if width < 1 {
		width = 1
	}
	if totalCount == 0 {
		return strings.Repeat("·", width)
	}

	successRatio := float64(successCount) / float64(totalCount)
	filledWidth := int(successRatio * float64(width))

	var result strings.Builder

	for i := 0; i < width; i++ {
		if i < filledWidth {
			char := lipgloss.NewStyle().
				Foreground(sparklineSuccessColor).
				Render("█")
			result.WriteString(char)
		}
		if i >= filledWidth {
			char := lipgloss.NewStyle().
				Foreground(sparklineNeutralColor).
				Render("░")
			result.WriteString(char)
		}
	}

	return result.String()
}
