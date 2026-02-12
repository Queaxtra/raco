package render

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var (
	streamBorderColor    = lipgloss.Color("240")
	streamTitleColor     = lipgloss.Color("255")
	streamSentColor      = lipgloss.Color("42")
	streamReceivedColor  = lipgloss.Color("33")
	streamErrorColor     = lipgloss.Color("196")
	streamSystemColor    = lipgloss.Color("240")
	streamTimestampColor = lipgloss.Color("240")

	streamStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(streamBorderColor).
			Padding(1, 2)

	streamTitleStyle = lipgloss.NewStyle().
				Foreground(streamTitleColor).
				Bold(true).
				MarginBottom(1)
)

type StreamMessage struct {
	Type      string
	Data      string
	Timestamp time.Time
	Direction string
}

func Stream(width, height int, messages []StreamMessage, protocol string, active bool, input interface{}) string {
	var content strings.Builder

	title := fmt.Sprintf("%s Stream Monitor", protocol)
	status := "Disconnected"
	statusColor := streamErrorColor
	if active {
		status = "Connected"
		statusColor = streamSentColor
	}

	titleLine := fmt.Sprintf("%s [%s]",
		streamTitleStyle.Render(title),
		lipgloss.NewStyle().Foreground(statusColor).Render(status),
	)
	content.WriteString(titleLine)
	content.WriteString("\n\n")

	if len(messages) == 0 {
		emptyMsg := lipgloss.NewStyle().
			Foreground(streamSystemColor).
			Italic(true).
			Render("No messages yet...")
		content.WriteString(emptyMsg)
		content.WriteString("\n")
	}

	maxMessages := height - 12
	if maxMessages < 5 {
		maxMessages = 5
	}

	startIdx := 0
	if len(messages) > maxMessages {
		startIdx = len(messages) - maxMessages
	}

	for _, msg := range messages[startIdx:] {
		timestamp := msg.Timestamp.Format("15:04:05")
		timestampStr := lipgloss.NewStyle().
			Foreground(streamTimestampColor).
			Render(fmt.Sprintf("[%s]", timestamp))

		var directionStr string
		var dataStyle lipgloss.Style

		switch msg.Direction {
		case "sent":
			directionStr = "→ SENT"
			dataStyle = lipgloss.NewStyle().Foreground(streamSentColor)
		case "received":
			directionStr = "← RECV"
			dataStyle = lipgloss.NewStyle().Foreground(streamReceivedColor)
		case "system":
			directionStr = "• SYS"
			dataStyle = lipgloss.NewStyle().Foreground(streamSystemColor)
		case "error":
			directionStr = "✗ ERR"
			dataStyle = lipgloss.NewStyle().Foreground(streamErrorColor)
		default:
			directionStr = "• MSG"
			dataStyle = lipgloss.NewStyle().Foreground(streamSystemColor)
		}

		line := fmt.Sprintf("%s %s: %s",
			timestampStr,
			directionStr,
			dataStyle.Render(truncateString(msg.Data, width-30)),
		)

		content.WriteString(line)
		content.WriteString("\n")
	}

	if active {
		content.WriteString("\n")
		inputLabel := lipgloss.NewStyle().
			Foreground(streamTitleColor).
			Render("Send: ")
		content.WriteString(inputLabel)

		if textInput, ok := input.(fmt.Stringer); ok {
			content.WriteString(textInput.String())
		}
	}

	if !active {
		content.WriteString("\n")
		helpText := lipgloss.NewStyle().
			Foreground(streamSystemColor).
			Italic(true).
			Render("←/→: Change Protocol • Ctrl+R: Connect")
		content.WriteString(helpText)
	}

	return streamStyle.
		Width(width - 4).
		Height(height - 4).
		Render(content.String())
}

func truncateString(s string, maxLen int) string {
	if maxLen < 4 {
		maxLen = 4
	}
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
