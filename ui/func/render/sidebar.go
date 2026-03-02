package render

import (
	"fmt"
	"raco/model"
	"raco/ui/theme"
)

// Sidebar renders the left panel: collections (expandable), their requests, and history.
// selectedIndex is the linear index over all visible items; expandedIndex is which collection is open.
// Help text at the bottom reflects vim keys (j/k, gg/G, Enter, h/l, e, w).
func Sidebar(width, height int, isActive bool, collections []*model.Collection, selectedIndex, expandedIndex int, history []*model.HistoryEntry, historyExpanded bool) string {
	if collections == nil {
		collections = []*model.Collection{}
	}
	if history == nil {
		history = []*model.HistoryEntry{}
	}

	style := theme.Box(isActive).Width(width - 2).Height(height - 2)

	content := theme.Title().Render("Collections") + "\n"

	if len(collections) == 0 {
		content += theme.Muted().Italic(true).Render("No collections") + "\n\n"
	}

	currentIdx := 0
	for colIdx, col := range collections {
		if col == nil {
			continue
		}
		isSelected := currentIdx == selectedIndex && isActive
		isExpanded := expandedIndex == colIdx

		icon := "›"
		if isExpanded {
			icon = "∨"
		}

		line := fmt.Sprintf(" %s %s (%d)", icon, col.Name, len(col.Requests))
		if isSelected {
			content += theme.Selected().Render(line)
		}
		if !isSelected {
			content += theme.Label().Render(line)
		}
		content += "\n"
		currentIdx++

		if isExpanded {
			for _, req := range col.Requests {
				reqSelected := currentIdx == selectedIndex && isActive
				reqLine := fmt.Sprintf("   %s %s", GetMethodIcon(req.Method), req.Name)
				if reqSelected {
					content += theme.Selected().Render(reqLine)
				}
				if !reqSelected {
					content += theme.Muted().PaddingLeft(1).Render(reqLine)
				}
				content += "\n"
				currentIdx++
			}
		}
	}

	content += "\n" + theme.Title().Render("History") + "\n"

	icon := "›"
	if historyExpanded {
		icon = "∨"
	}
	historyLine := fmt.Sprintf(" %s Recent (%d)", icon, len(history))
	if currentIdx == selectedIndex && isActive {
		content += theme.Selected().Render(historyLine)
	} else {
		content += theme.Label().Render(historyLine)
	}
	content += "\n"
	currentIdx++

	if historyExpanded {
		if len(history) == 0 {
			content += theme.Muted().Italic(true).PaddingLeft(2).Render("No requests yet") + "\n"
		}
		for i := len(history) - 1; i >= 0; i-- {
			entry := history[i]
			if entry == nil {
				continue
			}
			entrySelected := currentIdx == selectedIndex && isActive
			methodIcon := GetMethodIcon(entry.Method)
			if entry.Protocol == "WS" {
				methodIcon = "WS"
			}
			if entry.Protocol == "GRPC" {
				methodIcon = "GRPC"
			}
			url := entry.URL
			if len(url) > 28 {
				url = url[:25] + "..."
			}
			entryLine := fmt.Sprintf("   %s %s", methodIcon, url)
			if entrySelected {
				content += theme.Selected().Render(entryLine)
			}
			if !entrySelected {
				content += theme.Muted().PaddingLeft(1).Render(entryLine)
			}
			content += "\n"
			currentIdx++
		}
	}

	help := GetSidebarHelp(isActive)
	content += "\n" + theme.Muted().Italic(true).Render(help)

	return style.Render(content)
}

// GetMethodIcon returns a short label for HTTP/WS/gRPC for display in the sidebar.
func GetMethodIcon(method string) string {
	m := map[string]string{"GET": "GET", "POST": "POST", "PUT": "PUT", "DELETE": "DEL", "PATCH": "PATCH"}
	if s, ok := m[method]; ok {
		return s
	}
	return "REQ"
}

// GetSidebarHelp returns one-line hint: vim keys when sidebar is focused, focus panel hint otherwise.
func GetSidebarHelp(isActive bool) string {
	if isActive {
		return "j/k nav  gg/G top/bot  Enter open  h/l focus  e send  w save"
	}
	return "Tab or l → focus panel"
}
