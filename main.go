package main

import (
	"fmt"
	"os"
	"path/filepath"
	"raco/cli"
	"raco/ui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	if len(os.Args) > 1 {
		os.Exit(cli.Run(os.Args[1:]))
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	storagePath := filepath.Join(homeDir, ".raco")

	model := ui.NewModel(storagePath)
	program := tea.NewProgram(&model, tea.WithAltScreen(), tea.WithMouseCellMotion())

	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
