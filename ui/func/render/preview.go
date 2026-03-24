package render

import (
	"encoding/json"
	"raco/model"
	"raco/ui/theme"
	"strings"
)

// Preview renders the dry-run snapshot shown in the TUI preview mode.
func Preview(width int, height int, preview *model.PreviewResult) string {
	style := theme.Box(true).Width(width - 2).Height(height - 2)
	if preview == nil {
		return style.Render(theme.Muted().Render("No preview available"))
	}
	data, _ := json.MarshalIndent(preview, "", "  ")
	var b strings.Builder
	b.WriteString(theme.Title().Render("Preview"))
	b.WriteString("\n\n")
	b.WriteString(dataString(width, string(data)))
	return style.Render(b.String())
}

// dataString constrains large preview payloads to the current panel width.
func dataString(width int, value string) string {
	if width <= 8 {
		return value
	}
	return theme.Label().Width(width - 8).Render(value)
}
