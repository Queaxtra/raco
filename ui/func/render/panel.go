package render

import (
	"fmt"
	"strings"

	"raco/ui/theme"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// panelSelectedStyle highlights the current header or file row in the request panel.
var (
	panelSelectedStyle = lipgloss.NewStyle().
				Foreground(theme.Text).
				Background(theme.BgPanel).
				PaddingLeft(1)
)

// PanelInputs holds the bubbletea input models and selection state for the request builder.
type PanelInputs struct {
	MethodInput      textinput.Model
	URLInput         textinput.Model
	HeaderKeyInput   textinput.Model
	HeaderValueInput textinput.Model
	BodyInput        textarea.Model
	FilePathInput    textinput.Model
	FileFieldInput   textinput.Model
	HeaderKeys       []string
	SelectedHeader   int
	FileKeys         []string
	SelectedFile     int
}

// Panel renders the main request builder: method, URL, headers list + add row, files list + add row, body.
// Uses theme.Box for consistent border; selected header/file row uses panelSelectedStyle.
func Panel(width, height int, isActive bool, headers map[string]string, inputs PanelInputs) string {
	if headers == nil {
		headers = make(map[string]string)
	}

	style := theme.Box(isActive).Width(width - 2).Height(height - 2)

	var b strings.Builder
	b.WriteString(theme.Title().Render("Request"))
	b.WriteString("\n\n")

	methodHint := theme.Muted().Italic(true).Render(" ←/→ or h/l")
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, theme.Label().Render("Method "), inputs.MethodInput.View(), methodHint))
	b.WriteString("\n\n")

	b.WriteString(theme.Label().Render("URL"))
	b.WriteString("\n")
	b.WriteString(inputs.URLInput.View())
	b.WriteString("\n\n")

	b.WriteString(theme.Label().Render("Headers"))
	if len(headers) > 0 {
		b.WriteString(theme.Muted().Render(fmt.Sprintf(" (%d)", len(headers))))
	}
	b.WriteString("\n")
	if len(headers) == 0 {
		b.WriteString(theme.Muted().Italic(true).PaddingLeft(1).Render("Ctrl+S add") + "\n")
	}
	for i, key := range inputs.HeaderKeys {
		value := headers[key]
		line := fmt.Sprintf("  %s: %s", key, value)
		if i == inputs.SelectedHeader {
			b.WriteString(panelSelectedStyle.Render("▸ "+line) + "\n")
		}
		if i != inputs.SelectedHeader {
			b.WriteString(theme.Muted().PaddingLeft(2).Render(line) + "\n")
		}
	}
	b.WriteString(theme.Muted().Render("  Key : Value") + "\n")
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, inputs.HeaderKeyInput.View(), theme.Muted().Render(" : "), inputs.HeaderValueInput.View()))
	b.WriteString("\n\n")

	b.WriteString(theme.Label().Render("Files"))
	if len(inputs.FileKeys) > 0 {
		b.WriteString(theme.Muted().Render(fmt.Sprintf(" (%d)", len(inputs.FileKeys))))
	}
	b.WriteString("\n")
	if len(inputs.FileKeys) == 0 {
		b.WriteString(theme.Muted().Italic(true).PaddingLeft(1).Render("Ctrl+F add") + "\n")
	}
	for i, key := range inputs.FileKeys {
		if i == inputs.SelectedFile {
			b.WriteString(panelSelectedStyle.Render("▸ "+key) + "\n")
		}
		if i != inputs.SelectedFile {
			b.WriteString(theme.Muted().PaddingLeft(2).Render(key) + "\n")
		}
	}
	b.WriteString(theme.Muted().Render("  field = path") + "\n")
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, inputs.FileFieldInput.View(), theme.Muted().Render(" = "), inputs.FilePathInput.View()))
	b.WriteString("\n\n")

	b.WriteString(theme.Label().Render("Body"))
	b.WriteString("\n")
	b.WriteString(inputs.BodyInput.View())

	return style.Render(b.String())
}

// GetPanelHelp returns the one-line shortcut hint for the request panel (Tab, e, w, Ctrl+S/D/F/X).
func GetPanelHelp() string {
	return "Tab next  Shift+Tab prev  e send  w save  h/l method  Ctrl+S/D header  Ctrl+F/X file"
}
