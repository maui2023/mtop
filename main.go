package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/maui/mtop/internal/stats"
	"github.com/maui/mtop/internal/ui"
)

func main() {
	// Initialize the stats collector
	collector := stats.NewCollector()

	// Initialize the TUI model
	model := ui.NewModel(collector)

	// Run Bubble Tea program with alternate screen buffer (so it acts like a full-screen CLI dashboard)
	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running mtop: %v\n", err)
		os.Exit(1)
	}
}
