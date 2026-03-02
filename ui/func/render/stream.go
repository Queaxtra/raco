package render

import (
	"fmt"
	"strings"
	"time"

	"raco/ui/theme"

	"github.com/charmbracelet/lipgloss"
)

// Stream panel colors: sent (green), received (yellow), error (red).
var (
	streamSentColor     = lipgloss.Color("2")
	streamReceivedColor = lipgloss.Color("3")
	streamErrorColor    = lipgloss.Color("1")
)

// StreamMessage is one entry in the WebSocket/gRPC stream log (sent, received, system, error).
type StreamMessage struct {
	Type      string
	Data      string
	Timestamp time.Time
	Direction string
}

// Stream renders the WebSocket/gRPC view: title + [connected/disconnected], recent messages (newest at bottom),
// and the send input when active. input is the bubbletea textinput for the message line.
func Stream(width, height int, messages []StreamMessage, protocol string, active bool, input interface{}) string {
	style := theme.Box(true).Width(width - 2).Height(height - 2)
	var content strings.Builder

	title := fmt.Sprintf("%s Stream", protocol)
	status := "disconnected"
	statusStyle := lipgloss.NewStyle().Foreground(streamErrorColor)
	if active {
		status = "connected"
		statusStyle = lipgloss.NewStyle().Foreground(streamSentColor)
	}
	content.WriteString(theme.Title().Render(title))
	content.WriteString(statusStyle.Render(" ["+status+"]"))
	content.WriteString("\n\n")

	if len(messages) == 0 {
		content.WriteString(theme.Muted().Italic(true).Render("No messages yet") + "\n")
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
		ts := theme.Muted().Render("[" + msg.Timestamp.Format("15:04:05") + "]")
		var dir string
		var dataStyle lipgloss.Style
		switch msg.Direction {
		case "sent":
			dir = "→"
			dataStyle = lipgloss.NewStyle().Foreground(streamSentColor)
		case "received":
			dir = "←"
			dataStyle = lipgloss.NewStyle().Foreground(streamReceivedColor)
		case "system":
			dir = "•"
			dataStyle = theme.Muted()
		case "error":
			dir = "✗"
			dataStyle = lipgloss.NewStyle().Foreground(streamErrorColor)
		default:
			dir = "•"
			dataStyle = theme.Muted()
		}
		content.WriteString(fmt.Sprintf("%s %s %s\n", ts, dir, dataStyle.Render(truncateString(msg.Data, width-30))))
	}

	if active {
		content.WriteString("\n")
		content.WriteString(theme.Label().Render("Send: "))
		if textInput, ok := input.(interface{ View() string }); ok {
			content.WriteString(textInput.View())
		}
	}

	if !active {
		content.WriteString("\n")
		content.WriteString(theme.Muted().Italic(true).Render("e to connect"))
	}

	return style.Render(content.String())
}

// truncateString shortens long message payloads so the stream log stays readable.
func truncateString(s string, maxLen int) string {
	if maxLen < 4 {
		maxLen = 4
	}
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
