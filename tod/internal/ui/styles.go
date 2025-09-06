package ui

import (
	"github.com/charmbracelet/lipgloss"
)

// Styles holds all the styling for the TUI
type Styles struct {
	Header      lipgloss.Style
	Welcome     lipgloss.Style
	Footer      lipgloss.Style
	AdventureBox lipgloss.Style
	ErrorBox    lipgloss.Style
	SuccessBox  lipgloss.Style
}

// NewStyles creates a new styles instance
func NewStyles() *Styles {
	return &Styles{
		Header: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(1, 2).
			MarginBottom(1),

		Welcome: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#04B575")).
			MarginBottom(1),

		Footer: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262")).
			MarginTop(1),

		AdventureBox: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#874BFD")).
			Padding(1, 2).
			MarginBottom(1).
			Width(60),

		ErrorBox: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#FF5F87")).
			Foreground(lipgloss.Color("#FF5F87")).
			Padding(1, 2).
			MarginBottom(1),

		SuccessBox: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#04B575")).
			Foreground(lipgloss.Color("#04B575")).
			Padding(1, 2).
			MarginBottom(1),
	}
}