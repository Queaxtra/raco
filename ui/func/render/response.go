package render

import (
	"encoding/json"
	"fmt"
	"raco/model"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

var (
	responseBorderColor      = lipgloss.Color("240")
	responseActiveColor      = lipgloss.Color("255")
	responseSuccessColor     = lipgloss.Color("42")
	responseErrorColor       = lipgloss.Color("196")
	responseWarningColor     = lipgloss.Color("220")
	responseLabelColor       = lipgloss.Color("255")
	responseValueColor       = lipgloss.Color("252")
	responseHelpColor        = lipgloss.Color("240")

	responseStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(responseBorderColor).
			Padding(1, 2)

	responseActiveStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(responseActiveColor).
				Padding(1, 2)

	responseTitleStyle = lipgloss.NewStyle().
				Foreground(responseLabelColor).
				Bold(true).
				MarginBottom(1)

	responseStatusSuccessStyle = lipgloss.NewStyle().
					Foreground(responseSuccessColor).
					Bold(true)

	responseStatusErrorStyle = lipgloss.NewStyle().
					Foreground(responseErrorColor).
					Bold(true)

	responseStatusWarningStyle = lipgloss.NewStyle().
					Foreground(responseWarningColor).
					Bold(true)

	responseLabelStyle = lipgloss.NewStyle().
				Foreground(responseLabelColor).
				Bold(true)

	responseValueStyle = lipgloss.NewStyle().
				Foreground(responseValueColor)

	responseSectionStyle = lipgloss.NewStyle().
				MarginTop(1).
				MarginBottom(1)

	responseHeaderItemStyle = lipgloss.NewStyle().
				Foreground(responseValueColor).
				PaddingLeft(2)

	responseBodyStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(responseBorderColor).
				Padding(1)

	responseHelpStyle = lipgloss.NewStyle().
				Foreground(responseHelpColor).
				Italic(true).
				MarginTop(1)
)

func Response(width, height int, isActive bool, response *model.Response, responseViewport *viewport.Model) string {
	if response == nil {
		return responseStyle.
			Width(width - 4).
			Height(height - 4).
			Render("No response yet")
	}

	style := responseStyle
	if isActive {
		style = responseActiveStyle
	}

	var content strings.Builder

	content.WriteString(responseTitleStyle.Render("Response"))
	content.WriteString("\n\n")

	statusStyle := GetStatusStyle(response.StatusCode)
	statusLine := fmt.Sprintf("Status: %d %s",
		response.StatusCode,
		GetStatusText(response.StatusCode))
	content.WriteString(statusStyle.Render(statusLine))
	content.WriteString("\n")

	durationLine := fmt.Sprintf("Duration: %v", response.Duration)
	content.WriteString(responseValueStyle.Render(durationLine))
	content.WriteString("\n")

	if len(response.Headers) > 0 {
		content.WriteString(responseSectionStyle.Render(responseLabelStyle.Render("Headers")))
		content.WriteString("\n")

		for key, value := range response.Headers {
			headerLine := fmt.Sprintf("%s: %s", key, value)
			content.WriteString(responseHeaderItemStyle.Render(headerLine))
			content.WriteString("\n")
		}
	}

	content.WriteString(responseSectionStyle.Render(responseLabelStyle.Render("Body")))
	content.WriteString("\n")

	bodyContent := FormatResponseBody(response.Body)
	responseViewport.SetContent(bodyContent)

	bodyView := responseBodyStyle.
		Width(width - 12).
		Height(height - 20).
		Render(responseViewport.View())
	content.WriteString(bodyView)
	content.WriteString("\n")

	help := "j/k: Scroll • Tab: Switch • Esc: Back • F1: Dashboard • Ctrl+P: Palette"
	content.WriteString(responseHelpStyle.Render(help))

	return style.
		Width(width - 4).
		Height(height - 4).
		Render(content.String())
}

func GetStatusStyle(code int) lipgloss.Style {
	if code >= 200 && code < 300 {
		return responseStatusSuccessStyle
	}
	if code >= 300 && code < 400 {
		return responseStatusWarningStyle
	}
	return responseStatusErrorStyle
}

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

func FormatResponseBody(body string) string {
	if body == "" {
		return lipgloss.NewStyle().
			Foreground(responseHelpColor).
			Italic(true).
			Render("Empty response body")
	}

	maxBodyLen := 100000
	if len(body) > maxBodyLen {
		truncated := body[:maxBodyLen]
		warning := lipgloss.NewStyle().
			Foreground(responseWarningColor).
			Render("\n\n[Response truncated - too large to display]")
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
