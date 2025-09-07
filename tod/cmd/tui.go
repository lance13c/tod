package cmd

import (
	"errors"
	
	tea "github.com/charmbracelet/bubbletea"
	"github.com/ciciliostudio/tod/internal/config"
	"github.com/ciciliostudio/tod/internal/ui"
)

// ErrRestartConfig is returned when the user requests to restart configuration
var ErrRestartConfig = errors.New("restart_config")

// launchTUI initializes TUI components only when needed (lazy loading)
func launchTUI(todConfig *config.Config) error {
	// Launch directly into Navigation Mode instead of showing menu
	model := ui.NewModelWithInitialView(todConfig, ui.ViewNavigation)

	// Create the program with some options
	program := tea.NewProgram(
		model,
		tea.WithAltScreen(),       // Use alternate screen buffer
		tea.WithMouseCellMotion(), // Enable mouse support
	)

	// Set the program reference on the model so views can use it
	model.SetProgram(program)

	// Run the program
	finalModel, err := program.Run()
	if err != nil {
		return err
	}
	
	// Always ensure cleanup happens after the program exits
	if m, ok := finalModel.(*ui.Model); ok {
		// Check if restart was requested before cleanup
		shouldRestart := m.ShouldRestartConfig()
		
		// Clean up all resources
		m.CleanupAllViews()
		
		if shouldRestart {
			return ErrRestartConfig
		}
	}
	
	return nil
}