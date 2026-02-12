package render

import (
	"fmt"
	"raco/model"

	"github.com/charmbracelet/lipgloss"
)

var (
	sidebarBorderColor       = lipgloss.Color("240")
	sidebarActiveBorderColor = lipgloss.Color("255")
	sidebarTitleColor        = lipgloss.Color("255")
	sidebarItemColor         = lipgloss.Color("252")
	sidebarSelectedColor     = lipgloss.Color("255")
	sidebarDimColor          = lipgloss.Color("240")

	sidebarStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(sidebarBorderColor).
			Padding(1, 2)

	sidebarActiveStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(sidebarActiveBorderColor).
				Padding(1, 2)

	sidebarTitleStyle = lipgloss.NewStyle().
				Foreground(sidebarTitleColor).
				Bold(true).
				MarginBottom(1)

	sidebarItemStyle = lipgloss.NewStyle().
				Foreground(sidebarItemColor)

	sidebarSelectedStyle = lipgloss.NewStyle().
				Foreground(sidebarSelectedColor).
				Bold(true).
				Background(lipgloss.Color("235"))

	sidebarRequestStyle = lipgloss.NewStyle().
				Foreground(sidebarDimColor).
				PaddingLeft(2)

	sidebarRequestSelectedStyle = lipgloss.NewStyle().
					Foreground(sidebarSelectedColor).
					PaddingLeft(2).
					Bold(true)

	sidebarHelpStyle = lipgloss.NewStyle().
				Foreground(sidebarDimColor).
				Italic(true).
				MarginTop(1)
)

func Sidebar(width, height int, isActive bool, collections []*model.Collection, selectedIndex, expandedIndex int, history []*model.HistoryEntry, historyExpanded bool) string {
	if collections == nil {
		collections = []*model.Collection{}
	}
	if history == nil {
		history = []*model.HistoryEntry{}
	}

	style := sidebarStyle
	if isActive {
		style = sidebarActiveStyle
	}

	content := sidebarTitleStyle.Render("Collections")
	content += "\n"

	if len(collections) == 0 {
		emptyMsg := lipgloss.NewStyle().
			Foreground(sidebarDimColor).
			Italic(true).
			Render("No collections found")
		content += emptyMsg
		content += "\n\n"
	}

	currentIdx := 0
	for colIdx, col := range collections {
		if col == nil {
			continue
		}
		isSelected := currentIdx == selectedIndex && isActive
		isExpanded := expandedIndex == colIdx

		icon := "▶"
		if isExpanded {
			icon = "▼"
		}

		collectionLine := fmt.Sprintf("%s %s (%d)", icon, col.Name, len(col.Requests))

		if isSelected {
			content += sidebarSelectedStyle.Render(collectionLine)
		}
		if !isSelected {
			content += sidebarItemStyle.Render(collectionLine)
		}
		content += "\n"
		currentIdx++

		if isExpanded {
			for _, req := range col.Requests {
				reqIsSelected := currentIdx == selectedIndex && isActive
				reqLine := fmt.Sprintf("  %s %s", GetMethodIcon(req.Method), req.Name)

				if reqIsSelected {
					content += sidebarRequestSelectedStyle.Render(reqLine)
				}
				if !reqIsSelected {
					content += sidebarRequestStyle.Render(reqLine)
				}
				content += "\n"
				currentIdx++
			}
		}
	}

	content += "\n"
	content += sidebarTitleStyle.Render("History")
	content += "\n"

	icon := "▶"
	if historyExpanded {
		icon = "▼"
	}
	historyLine := fmt.Sprintf("%s Recent Requests (%d)", icon, len(history))

	historyHeaderSelected := currentIdx == selectedIndex && isActive
	if historyHeaderSelected {
		content += sidebarSelectedStyle.Render(historyLine)
	}
	if !historyHeaderSelected {
		content += sidebarItemStyle.Render(historyLine)
	}
	content += "\n"
	currentIdx++

	if historyExpanded {
		if len(history) == 0 {
			emptyMsg := lipgloss.NewStyle().
				Foreground(sidebarDimColor).
				Italic(true).
				PaddingLeft(2).
				Render("No requests yet")
			content += emptyMsg
			content += "\n"
		}

		for i := len(history) - 1; i >= 0; i-- {
			entry := history[i]
			if entry == nil {
				continue
			}
			entryIsSelected := currentIdx == selectedIndex && isActive
			methodIcon := GetMethodIcon(entry.Method)
			if entry.Protocol == "WS" {
				methodIcon = "WS"
			}
			if entry.Protocol == "GRPC" {
				methodIcon = "GRPC"
			}

			url := entry.URL
			if len(url) > 30 {
				url = url[:27] + "..."
			}
			entryLine := fmt.Sprintf("  %s %s", methodIcon, url)

			if entryIsSelected {
				content += sidebarRequestSelectedStyle.Render(entryLine)
			}
			if !entryIsSelected {
				content += sidebarRequestStyle.Render(entryLine)
			}
			content += "\n"
			currentIdx++
		}
	}

	help := GetSidebarHelp(isActive)
	content += "\n"
	content += sidebarHelpStyle.Render(help)

	return style.
		Width(width - 4).
		Height(height - 4).
		Render(content)
}

func GetMethodIcon(method string) string {
	methods := map[string]string{
		"GET":    "GET",
		"POST":   "POST",
		"PUT":    "PUT",
		"DELETE": "DEL",
		"PATCH":  "PATCH",
	}

	icon, exists := methods[method]
	if exists {
		return icon
	}

	return "REQ"
}

func GetSidebarHelp(isActive bool) string {
	if isActive {
		return "j/k: Navigate • Enter: Expand/Load • Ctrl+N: New • Ctrl+P: Palette • F1: Dashboard"
	}
	return "Tab: Focus Sidebar"
}
