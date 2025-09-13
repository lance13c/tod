package ui

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lance13c/tod/internal/config"
	"github.com/lance13c/tod/internal/ui/views"
)

// ViewState represents the current view/screen
type ViewState int

const (
	ViewMenu ViewState = iota
	ViewNavigation
)

// Model is the main application model
type Model struct {
	currentView ViewState
	width       int
	height      int
	
	// Configuration
	config *config.Config
	projectRoot string
	program *tea.Program // Reference to the program for views that need it
	
	// Views
	navigationView *views.NavigationView
	
	// Menu state
	menuList list.Model
	
	// Global styles
	styles *Styles
	
	// Special flags
	requestRestart bool // Set to true when user requests configuration restart
}

// MenuItem represents a menu item
type MenuItem struct {
	title       string
	description string
	action      func() tea.Cmd
}

func (m MenuItem) Title() string       { return m.title }
func (m MenuItem) Description() string { return m.description }
func (m MenuItem) FilterValue() string { return m.title }

// NewModel creates a new main model
func NewModel(cfg *config.Config) *Model {
	return NewModelWithRoot(cfg, ".")
}

// ShouldRestartConfig returns true if the user requested configuration restart
func (m *Model) ShouldRestartConfig() bool {
	return m.requestRestart
}

// CleanupAllViews cleans up resources from all views
func (m *Model) CleanupAllViews() {
	// Clean up navigation view
	if m.navigationView != nil {
		m.navigationView.Cleanup()
	}
}

// NewModelWithInitialView creates a new main model with a specific initial view
func NewModelWithInitialView(cfg *config.Config, initialView ViewState) *Model {
	model := NewModelWithRoot(cfg, ".")
	model.currentView = initialView
	
	// Initialize the appropriate view based on initialView
	switch initialView {
	case ViewNavigation:
		model.navigationView = views.NewNavigationView(cfg)
	}
	
	return model
}

// NewModelWithRoot creates a new main model with project root
func NewModelWithRoot(cfg *config.Config, projectRoot string) *Model {
	// Create menu items
	items := []list.Item{
		MenuItem{
			title:       "Navigation Mode",
			description: "Fast CLI-based website navigation with smart autocomplete",
			action: func() tea.Cmd {
				return showNavigationView()
			},
		},
		MenuItem{
			title:       "Exit",
			description: "Leave the realm",
		},
	}

	// Create the menu list
	menuList := list.New(items, list.NewDefaultDelegate(), 0, 0)
	menuList.Title = ""
	menuList.SetShowHelp(true)
	menuList.SetFilteringEnabled(false)
	menuList.SetShowPagination(false)

	return &Model{
		currentView: ViewMenu,
		config:      cfg,
		projectRoot: projectRoot,
		menuList:    menuList,
		styles:      NewStyles(),
	}
}


func showNavigationView() tea.Cmd {
	return func() tea.Msg {
		return ShowNavigationViewMsg{}
	}
}

// Init implements tea.Model
func (m *Model) Init() tea.Cmd {
	// Initialize the appropriate view based on current view
	switch m.currentView {
	case ViewNavigation:
		if m.navigationView != nil {
			return m.navigationView.Init()
		}
	}
	return nil
}

// Update implements tea.Model
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			// Clean up all resources before quitting
			m.CleanupAllViews()
			return m, tea.Quit
		case "esc":
			if m.currentView != ViewMenu {
				// Clean up resources when leaving views
				if m.currentView == ViewNavigation && m.navigationView != nil {
					m.navigationView.Cleanup()
				}
				m.currentView = ViewMenu
				return m, nil
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.menuList.SetWidth(msg.Width)
		m.menuList.SetHeight(msg.Height - 10) // Leave space for header/footer
		
		// Update navigation view if it exists
		if m.navigationView != nil {
			var navModel tea.Model
			navModel, _ = m.navigationView.Update(msg)
			m.navigationView = navModel.(*views.NavigationView)
		}

	
	case ShowNavigationViewMsg:
		// Initialize Navigation view if not already created
		if m.navigationView == nil {
			m.navigationView = views.NewNavigationView(m.config)
		}
		m.currentView = ViewNavigation
		return m, m.navigationView.Init()
	
	case views.ReturnToMenuMsg:
		// Clean up resources when leaving views
		if m.currentView == ViewNavigation && m.navigationView != nil {
			m.navigationView.Cleanup()
		}
		// Return to the main menu
		m.currentView = ViewMenu
		return m, nil
		
	case views.RestartConfigMsg:
		// Set flag and quit
		m.requestRestart = true
		return m, tea.Quit
	}

	// Update the current view
	switch m.currentView {
	case ViewMenu:
		m.menuList, cmd = m.menuList.Update(msg)
		
		// Handle menu selection
		if msg, ok := msg.(tea.KeyMsg); ok && msg.String() == "enter" {
			selectedItem := m.menuList.SelectedItem().(MenuItem)
			switch selectedItem.title {
			case "Exit":
				return m, tea.Quit
			case "Navigation Mode":
				return m, showNavigationView()
			}
		}
		
	case ViewNavigation:
		if m.navigationView != nil {
			var navCmd tea.Cmd
			var navModel tea.Model
			navModel, navCmd = m.navigationView.Update(msg)
			m.navigationView = navModel.(*views.NavigationView)
			return m, navCmd
		}
	}

	return m, cmd
}

// View implements tea.Model
func (m *Model) View() string {
	switch m.currentView {
	case ViewMenu:
		return m.renderMenu()
	case ViewNavigation:
		if m.navigationView != nil {
			return m.navigationView.View()
		}
		return "Navigation view not initialized"
	default:
		return "Unknown view"
	}
}

func (m *Model) renderMenu() string {
	header := m.styles.Header.Render(asciiLogo)
	
	welcome := m.styles.Welcome.Render(
		"Welcome, brave tester! Choose your path:",
	)
	
	menu := m.menuList.View()
	
	footer := m.styles.Footer.Render(
		"[↑/↓ Navigate] [Enter Select] [q Quit]",
	)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		"",
		welcome,
		"",
		menu,
		"",
		footer,
	)
}


const asciiLogo = `
╔══════════════════════════════════════════╗
║     _______  ___   _______              ║
║    |       ||   | |       |             ║
║    |_     _||   | |    ___|             ║
║      |   |  |   | |   |___              ║
║      |   |  |   | |    ___|             ║
║      |   |  |   | |   |                 ║
║      |___|  |___| |___|                 ║
║                                          ║
║   Text-adventure Interface Framework    ║
╚══════════════════════════════════════════╝
`

// SetProgram sets the program reference and passes it to views that need it
func (m *Model) SetProgram(p *tea.Program) {
	m.program = p
}

// Message types
type ShowNavigationViewMsg struct{}

