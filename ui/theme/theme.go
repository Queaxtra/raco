// Package theme provides a single design system for the TUI: colors, borders, and
// text styles. All render packages use these so the app stays minimal and consistent.
package theme

import (
	"github.com/charmbracelet/lipgloss"
)

// Shared palette (xterm-256). Dark background, muted borders, one accent for highlights.
var (
	Bg       = lipgloss.Color("235")
	BgPanel  = lipgloss.Color("236")
	Border   = lipgloss.Color("240")
	BorderF  = lipgloss.Color("245")
	Text     = lipgloss.Color("252")
	TextDim  = lipgloss.Color("245")
	TextMute = lipgloss.Color("240")
	Accent   = lipgloss.Color("6")
	Success  = lipgloss.Color("2")
	Error    = lipgloss.Color("1")
	Warning  = lipgloss.Color("3")
)

// Box returns a minimal frame style for panels/sidebar. Active panel uses a brighter border.
func Box(active bool) lipgloss.Style {
	border := Border
	if active {
		border = BorderF
	}
	return lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(border).
		Padding(0, 1)
}

// Title is used for section headers (e.g. "Collections", "Request").
func Title() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(Text).
		Bold(true)
}

// Label is used for field names and secondary headings.
func Label() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(TextDim)
}

// Muted is used for hints, timestamps, and non-emphasis text.
func Muted() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(TextMute)
}

// Selected highlights the focused item in lists (sidebar, headers, files).
func Selected() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(Text).
		Background(BgPanel).
		Bold(true)
}

// StatusBar is the bottom bar style; muted text on dark background.
func StatusBar() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(TextMute).
		Background(Bg).
		Padding(0, 1)
}

// KeyHint styles shortcut keys in the status bar so they stand out from descriptions.
func KeyHint() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(Text).
		Bold(true)
}
