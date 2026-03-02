package render

import (
	"encoding/json"
	"fmt"
	"raco/model"
	"raco/ui/theme"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

// Response panel styles: status by code (2xx green, 3xx yellow, 4xx/5xx red), body in a sub-box.
var (
	responseStatusSuccessStyle = lipgloss.NewStyle().Foreground(theme.Success).Bold(true)
	responseStatusErrorStyle   = lipgloss.NewStyle().Foreground(theme.Error).Bold(true)
	responseStatusWarningStyle = lipgloss.NewStyle().Foreground(theme.Warning).Bold(true)
	responseBodyStyle          = lipgloss.NewStyle().
					BorderStyle(lipgloss.NormalBorder()).
					BorderForeground(theme.Border).
					Padding(0, 1)
)

// Response renders the HTTP response view: status line, duration, optional headers, scrollable body.
// Body is formatted (JSON indented) and passed to responseViewport for j/k scrolling.
func Response(width, height int, isActive bool, response *model.Response, responseViewport *viewport.Model) string {
	style := theme.Box(isActive).Width(width - 2).Height(height - 2)

	if response == nil {
		return style.Render(theme.Muted().Italic(true).Render("No response — e to send"))
	}

	var content strings.Builder
	content.WriteString(theme.Title().Render("Response"))
	content.WriteString("\n\n")

	statusStyle := GetStatusStyle(response.StatusCode)
	content.WriteString(statusStyle.Render(fmt.Sprintf("%d %s", response.StatusCode, GetStatusText(response.StatusCode))))
	content.WriteString("  ")
	content.WriteString(theme.Muted().Render(fmt.Sprintf("%v", response.Duration)))
	content.WriteString("\n\n")

	if len(response.Headers) > 0 {
		content.WriteString(theme.Label().Render("Headers"))
		content.WriteString("\n")
		for key, value := range response.Headers {
			content.WriteString(theme.Muted().PaddingLeft(1).Render(key+": "+value) + "\n")
		}
		content.WriteString("\n")
	}

	content.WriteString(theme.Label().Render("Body"))
	content.WriteString("\n")
	bodyContent := FormatResponseBody(response.Body)
	responseViewport.SetContent(bodyContent)
	content.WriteString(responseBodyStyle.Width(width - 8).Height(height - 16).Render(responseViewport.View()))

	return style.Render(content.String())
}

// GetStatusStyle returns color by HTTP status range: success (2xx), warning (3xx), error (4xx/5xx).
func GetStatusStyle(code int) lipgloss.Style {
	if code >= 200 && code < 300 {
		return responseStatusSuccessStyle
	}
	if code >= 300 && code < 400 {
		return responseStatusWarningStyle
	}
	return responseStatusErrorStyle
}

// GetStatusText returns a short label for common HTTP status codes.
func GetStatusText(code int) string {
	statusTexts := map[int]string{
		200: "OK",
		201: "Created",
		204: "No Content",
		400: "Bad Request",
		401: "Unauthorized",
		403: "Forbidden",
		404: "Not Found",
		500: "Internal Server Error",
		502: "Bad Gateway",
		503: "Service Unavailable",
	}

	text, exists := statusTexts[code]
	if exists {
		return text
	}

	if code >= 200 && code < 300 {
		return "Success"
	}
	if code >= 300 && code < 400 {
		return "Redirect"
	}
	if code >= 400 && code < 500 {
		return "Client Error"
	}
	if code >= 500 {
		return "Server Error"
	}

	return "Unknown"
}

// FormatResponseBody pretty-prints JSON when possible; truncates very large bodies and appends [truncated].
func FormatResponseBody(body string) string {
	if body == "" {
		return theme.Muted().Italic(true).Render("(empty)")
	}

	maxBodyLen := 100000
	if len(body) > maxBodyLen {
		truncated := body[:maxBodyLen]
		warning := theme.Muted().Render("\n\n[truncated]")
		return truncated + warning
	}

	var parsed interface{}
	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		return body
	}

	formatted, err := json.MarshalIndent(parsed, "", "  ")
	if err != nil {
		return body
	}

	return SyntaxHighlightJSON(string(formatted))
}

// SyntaxHighlightJSON applies simple key/string/number/bool/null coloring to JSON lines for readability.
func SyntaxHighlightJSON(jsonStr string) string {
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("105"))
	stringStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	numberStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
	boolStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("170"))
	nullStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	lines := strings.Split(jsonStr, "\n")
	var highlighted strings.Builder

	for _, line := range lines {
		hasColon := strings.Contains(line, ":")
		if hasColon {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				key := parts[0]
				value := parts[1]

				highlighted.WriteString(keyStyle.Render(key))
				highlighted.WriteString(":")

				trimmedValue := strings.TrimSpace(value)
				isString := strings.HasPrefix(trimmedValue, "\"")
				if isString {
					highlighted.WriteString(stringStyle.Render(value))
				}
				if !isString {
					isBool := trimmedValue == "true" || trimmedValue == "false"
					if isBool {
						highlighted.WriteString(boolStyle.Render(value))
					}
					if !isBool {
						isNull := trimmedValue == "null"
						if isNull {
							highlighted.WriteString(nullStyle.Render(value))
						}
						if !isNull {
							highlighted.WriteString(numberStyle.Render(value))
						}
					}
				}
			}
		}
		if !hasColon {
			highlighted.WriteString(line)
		}
		highlighted.WriteString("\n")
	}

	return highlighted.String()
}
