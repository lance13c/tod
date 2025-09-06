package cmd

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/ciciliostudio/tod/internal/config"
	"github.com/ciciliostudio/tod/internal/ui"
)

// launchTUI initializes TUI components only when needed (lazy loading)
func launchTUI(todConfig *config.Config) error {
	// Initialize the main model with configuration
	model := ui.NewModel(todConfig)

	// Create the program with some options
	program := tea.NewProgram(
		model,
		tea.WithAltScreen(),       // Use alternate screen buffer
		tea.WithMouseCellMotion(), // Enable mouse support
	)

	// Run the program
	_, err := program.Run()
	return err
}