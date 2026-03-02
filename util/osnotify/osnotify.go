// Package osnotify sends OS-level desktop notifications so users see alerts
// when the terminal is in the background (e.g. after a long request or run).
package osnotify

import (
	"os/exec"
	"runtime"
	"strings"
)

// Send shows a desktop notification with the given title and body.
// It runs the platform command in a separate process and does not block.
// Unsupported platforms or missing tools (e.g. notify-send) are no-ops.
func Send(title, body string) {
	if title == "" {
		title = "Raco"
	}
	body = truncate(body, 200)

	go func() {
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "darwin":
			cmd = exec.Command("osascript", "-e", appleScriptNotification(title, body))
		case "linux":
			cmd = exec.Command("notify-send", title, body)
		default:
			return
		}
		_ = cmd.Run()
	}()
}

func truncate(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

// appleScriptNotification escapes title and body for AppleScript and returns the script.
func appleScriptNotification(title, body string) string {
	body = strings.ReplaceAll(body, "\n", " ")
	body = strings.ReplaceAll(body, "\r", " ")
	body = strings.ReplaceAll(body, `\`, `\\`)
	body = strings.ReplaceAll(body, `"`, `\"`)
	title = strings.ReplaceAll(title, `\`, `\\`)
	title = strings.ReplaceAll(title, `"`, `\"`)
	return `display notification "` + body + `" with title "` + title + `"`
}
