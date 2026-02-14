package render

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

var (
	panelBorderColor       = lipgloss.Color("240")
	panelActiveBorderColor = lipgloss.Color("255")
	panelLabelColor        = lipgloss.Color("255")
	panelValueColor        = lipgloss.Color("252")
	panelHelpColor         = lipgloss.Color("240")

	panelStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(panelBorderColor).
			Padding(1, 2)

	panelActiveStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(panelActiveBorderColor).
				Padding(1, 2)

	panelTitleStyle = lipgloss.NewStyle().
			Foreground(panelLabelColor).
			Bold(true).
			MarginBottom(1)

	panelLabelStyle = lipgloss.NewStyle().
			Foreground(panelLabelColor).
			Bold(true)

	panelSectionStyle = lipgloss.NewStyle().
				MarginTop(1)

	panelHeaderItemStyle = lipgloss.NewStyle().
				Foreground(panelValueColor).
				PaddingLeft(2)

	panelHelpStyle = lipgloss.NewStyle().
			Foreground(panelHelpColor).
			Italic(true).
			MarginTop(1)
)

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

func Panel(width, height int, isActive bool, headers map[string]string, inputs PanelInputs) string {
	if headers == nil {
		headers = make(map[string]string)
	}

	style := panelStyle
	if isActive {
		style = panelActiveStyle
	}

	var content strings.Builder

	content.WriteString(panelTitleStyle.Render("Request Builder"))
	content.WriteString("\n\n")

	methodHint := lipgloss.NewStyle().
		Foreground(panelHelpColor).
		Italic(true).
		Render(" (← → to change)")
	
	methodLine := lipgloss.JoinHorizontal(
		lipgloss.Top,
		panelLabelStyle.Render("Protocol: "),
		inputs.MethodInput.View(),
		methodHint,
	)
	content.WriteString(methodLine)
	content.WriteString("\n\n")

	content.WriteString(panelLabelStyle.Render("URL"))
	content.WriteString("\n")
	content.WriteString(inputs.URLInput.View())
	content.WriteString("\n")

	headerTitle := "Headers"
	if len(headers) > 0 {
		headerTitle = fmt.Sprintf("Headers (%d)", len(headers))
	}
	content.WriteString(panelSectionStyle.Render(panelLabelStyle.Render(headerTitle)))
	content.WriteString("\n")

	if len(headers) == 0 {
		emptyMsg := lipgloss.NewStyle().
			Foreground(panelHelpColor).
			Italic(true).
			PaddingLeft(2).
			Render("No headers (Ctrl+S to add)")
		content.WriteString(emptyMsg)
	}

	selectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("212")).
		Background(lipgloss.Color("236")).
		PaddingLeft(2)

	for i, key := range inputs.HeaderKeys {
		value := headers[key]
		headerLine := fmt.Sprintf("%s: %s", key, value)
		if i == inputs.SelectedHeader {
			content.WriteString(selectedStyle.Render("▸ " + headerLine))
		}
		if i != inputs.SelectedHeader {
			content.WriteString(panelHeaderItemStyle.Render("  " + headerLine))
		}
		content.WriteString("\n")
	}

	content.WriteString("\n")
	content.WriteString(panelLabelStyle.Render("Add Header"))
	content.WriteString("\n")

	headerInputLine := lipgloss.JoinHorizontal(
		lipgloss.Top,
		inputs.HeaderKeyInput.View(),
		lipgloss.NewStyle().Foreground(panelLabelColor).Render(" : "),
		inputs.HeaderValueInput.View(),
	)
	content.WriteString(headerInputLine)
	content.WriteString("\n")

	fileTitle := "Files"
	if len(inputs.FileKeys) > 0 {
		fileTitle = fmt.Sprintf("Files (%d)", len(inputs.FileKeys))
	}
	content.WriteString(panelSectionStyle.Render(panelLabelStyle.Render(fileTitle)))
	content.WriteString("\n")

	if len(inputs.FileKeys) == 0 {
		emptyMsg := lipgloss.NewStyle().
			Foreground(panelHelpColor).
			Italic(true).
			PaddingLeft(2).
			Render("No files (Ctrl+F to add)")
		content.WriteString(emptyMsg)
	}

	fileSelectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("212")).
		Background(lipgloss.Color("236")).
		PaddingLeft(2)

	for i, key := range inputs.FileKeys {
		fileLine := key
		if i == inputs.SelectedFile {
			content.WriteString(fileSelectedStyle.Render("▸ " + fileLine))
		}
		if i != inputs.SelectedFile {
			content.WriteString(panelHeaderItemStyle.Render("  " + fileLine))
		}
		content.WriteString("\n")
	}

	content.WriteString("\n")
	content.WriteString(panelLabelStyle.Render("Add File"))
	content.WriteString("\n")

	fileInputLine := lipgloss.JoinHorizontal(
		lipgloss.Top,
		inputs.FileFieldInput.View(),
		lipgloss.NewStyle().Foreground(panelLabelColor).Render(" = "),
		inputs.FilePathInput.View(),
	)
	content.WriteString(fileInputLine)
	content.WriteString("\n")

	content.WriteString(panelSectionStyle.Render(panelLabelStyle.Render("Body")))
	content.WriteString("\n")
	content.WriteString(inputs.BodyInput.View())
	content.WriteString("\n")

	help := GetPanelHelp()
	content.WriteString(panelHelpStyle.Render(help))

	return style.
		Width(width - 4).
		Height(height - 4).
		Render(content.String())
}

func GetPanelHelp() string {
	return "Tab: Next • Ctrl+R: Send • Ctrl+W: Save • Ctrl+S: +Header • Ctrl+D: -Header • Ctrl+F: +File • Ctrl+X: -File • Esc: Back"
}
