package ui

import (
	"errors"
	"log"
	"os"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ciciliostudio/tod/internal/browser"
	"github.com/ciciliostudio/tod/internal/config"
	"github.com/ciciliostudio/tod/internal/types"
	"github.com/ciciliostudio/tod/internal/manifest"
	"github.com/ciciliostudio/tod/internal/ui/views"
)

// ViewState represents the current view/screen
type ViewState int

const (
	ViewMenu ViewState = iota
	ViewAdventure
	ViewChatAdventure
	ViewNavigation
	ViewChromeDebugger
	ViewEmail
	ViewResults
	ViewSettings
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
	adventureView *views.AdventureView
	chatAdventureView tea.Model // Can be either ChatAdventureView or ChatAdventureV2View
	navigationView *views.NavigationView
	chromeDebuggerView *views.ChromeDebuggerView
	
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
	// Clean up adventure view
	if m.adventureView != nil {
		m.adventureView.Cleanup()
	}
	
	// Clean up chat adventure view
	if m.chatAdventureView != nil {
		if v, ok := m.chatAdventureView.(interface{ Cleanup() }); ok {
			v.Cleanup()
		}
	}
	
	// Clean up navigation view
	if m.navigationView != nil {
		m.navigationView.Cleanup()
	}
	
	// Clean up chrome debugger view
	// Note: ChromeDebuggerView uses the global Chrome manager, 
	// which will be cleaned up below
	
	// Ensure global Chrome is closed
	browser.CloseGlobalChromeDPManager()
}

// NewModelWithInitialView creates a new main model with a specific initial view
func NewModelWithInitialView(cfg *config.Config, initialView ViewState) *Model {
	model := NewModelWithRoot(cfg, ".")
	model.currentView = initialView
	
	// Initialize the appropriate view based on initialView
	switch initialView {
	case ViewChromeDebugger:
		model.chromeDebuggerView = views.NewChromeDebuggerView()
	case ViewChatAdventure:
		model.chatAdventureView = views.NewChatAdventureV2View(cfg)
		// Program reference will be set later in SetProgram method
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
			title:       "Tod Adventure Mode",
			description: "AI-powered conversational testing assistant",
		},
		MenuItem{
			title:       "Navigation Mode",
			description: "Fast CLI-based website navigation with smart autocomplete",
			action: func() tea.Cmd {
				return showNavigationView()
			},
		},
		MenuItem{
			title:       "Chrome Test Discovery",
			description: "Open Chrome debugger and discover untested actions",
			action: func() tea.Cmd {
				return showChromeDebuggerView()
			},
		},
		MenuItem{
			title:       "Start New Journey",
			description: "Begin a fresh testing adventure",
		},
		MenuItem{
			title:       "Continue Journey", 
			description: "Resume from where you left off",
		},
		MenuItem{
			title:       "Review Past Adventures",
			description: "Browse your testing history",
		},
		MenuItem{
			title:       "Generate Test Scroll",
			description: "Convert session to test code",
		},
		MenuItem{
			title:       "Configure Your Realm",
			description: "Settings and environments",
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

	// Create adventure view
	adventureView := views.NewAdventureView(cfg)

	// Initialize browser client for adventure mode
	env := cfg.GetCurrentEnv()
	if env.BaseURL != "" {
		browserClient, err := browser.NewClient(env.BaseURL)
		if err == nil {
			adventureView.SetBrowserClient(browserClient)
		} else {
			// Check if this is a permission error
			if errors.Is(err, browser.ErrScreenRecordingPermission) ||
				errors.Is(err, browser.ErrAccessibilityPermission) ||
				errors.Is(err, browser.ErrChromeNotFound) {
				// Log permission error for user awareness
				log.Printf("Browser permission error: %v", err)
				log.Printf("Instructions: %s", browser.GetPermissionInstructions(err))
			} else {
				// Log other browser errors
				log.Printf("Browser initialization error: %v", err)
			}
		}
	}

	return &Model{
		currentView:   ViewMenu,
		config:        cfg,
		projectRoot:   projectRoot,
		adventureView: adventureView,
		menuList:      menuList,
		styles:        NewStyles(),
	}
}

func showChromeDebuggerView() tea.Cmd {
	return func() tea.Msg {
		return ShowChromeDebuggerViewMsg{}
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
	case ViewChromeDebugger:
		if m.chromeDebuggerView != nil {
			return m.chromeDebuggerView.Init()
		}
	case ViewChatAdventure:
		if m.chatAdventureView != nil {
			return m.chatAdventureView.Init()
		}
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
				if m.currentView == ViewAdventure && m.adventureView != nil {
					m.adventureView.Cleanup()
				}
				if m.currentView == ViewChatAdventure && m.chatAdventureView != nil {
					// Call cleanup if the view implements it
					if v, ok := m.chatAdventureView.(interface{ Cleanup() }); ok {
						v.Cleanup()
					}
				}
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
		
		// Update adventure view dimensions
		adventureModel, _ := m.adventureView.Update(msg)
		m.adventureView = adventureModel.(*views.AdventureView)
		
		// Update chat adventure view if it exists
		if m.chatAdventureView != nil {
			m.chatAdventureView, _ = m.chatAdventureView.Update(msg)
		}
		
		// Update chrome debugger view if it exists
		if m.chromeDebuggerView != nil {
			var chromeModel tea.Model
			chromeModel, _ = m.chromeDebuggerView.Update(msg)
			m.chromeDebuggerView = chromeModel.(*views.ChromeDebuggerView)
		}
		
		// Update navigation view if it exists
		if m.navigationView != nil {
			var navModel tea.Model
			navModel, _ = m.navigationView.Update(msg)
			m.navigationView = navModel.(*views.NavigationView)
		}

	
	case ShowChromeDebuggerViewMsg:
		// Initialize Chrome debugger view if not already created
		if m.chromeDebuggerView == nil {
			m.chromeDebuggerView = views.NewChromeDebuggerView()
		}
		m.currentView = ViewChromeDebugger
		return m, m.chromeDebuggerView.Init()
	
	case ShowNavigationViewMsg:
		// Initialize Navigation view if not already created
		if m.navigationView == nil {
			m.navigationView = views.NewNavigationView(m.config)
		}
		m.currentView = ViewNavigation
		return m, m.navigationView.Init()
	
	case views.ReturnToMenuMsg:
		// Clean up resources when leaving views
		if m.currentView == ViewAdventure && m.adventureView != nil {
			m.adventureView.Cleanup()
		}
		if m.currentView == ViewChatAdventure && m.chatAdventureView != nil {
			// Call cleanup if the view implements it
			if v, ok := m.chatAdventureView.(interface{ Cleanup() }); ok {
				v.Cleanup()
			}
		}
		if m.currentView == ViewNavigation && m.navigationView != nil {
			m.navigationView.Cleanup()
		}
		// Return to the main menu
		m.currentView = ViewMenu
		return m, nil
		
	case views.RestartConfigMsg:
		// Clean up current view
		if m.chatAdventureView != nil {
			if v, ok := m.chatAdventureView.(interface{ Cleanup() }); ok {
				v.Cleanup()
			}
		}
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
			case "Tod Adventure Mode":
				if m.chatAdventureView == nil {
					// Use the enhanced V2 chat view
					m.chatAdventureView = views.NewChatAdventureV2View(m.config)
					// Set program reference if available
					if v2, ok := m.chatAdventureView.(*views.ChatAdventureV2View); ok && m.program != nil {
						v2.SetProgram(m.program)
					}
				}
				m.currentView = ViewChatAdventure
				return m, m.chatAdventureView.Init()
			case "Navigation Mode":
				return m, showNavigationView()
			case "Chrome Test Discovery":
				return m, showChromeDebuggerView()
			case "Start New Journey":
				m.currentView = ViewAdventure
				// Load available actions
				m.loadActionsForAdventure()
			}
		}
		
	case ViewAdventure:
		var adventureCmd tea.Cmd
		adventureModel, adventureCmd := m.adventureView.Update(msg)
		m.adventureView = adventureModel.(*views.AdventureView)
		return m, adventureCmd

	case ViewChatAdventure:
		if m.chatAdventureView != nil {
			var chatCmd tea.Cmd
			m.chatAdventureView, chatCmd = m.chatAdventureView.Update(msg)
			return m, chatCmd
		}
	
	case ViewNavigation:
		if m.navigationView != nil {
			var navCmd tea.Cmd
			var navModel tea.Model
			navModel, navCmd = m.navigationView.Update(msg)
			m.navigationView = navModel.(*views.NavigationView)
			return m, navCmd
		}
	
	case ViewChromeDebugger:
		if m.chromeDebuggerView != nil {
			var chromeCmd tea.Cmd
			var chromeModel tea.Model
			chromeModel, chromeCmd = m.chromeDebuggerView.Update(msg)
			m.chromeDebuggerView = chromeModel.(*views.ChromeDebuggerView)
			return m, chromeCmd
		}
	}

	return m, cmd
}

// View implements tea.Model
func (m *Model) View() string {
	switch m.currentView {
	case ViewMenu:
		return m.renderMenu()
	case ViewAdventure:
		return m.adventureView.View()
	case ViewChatAdventure:
		if m.chatAdventureView != nil {
			return m.chatAdventureView.View()
		}
		return "Chat adventure view not initialized"
	case ViewNavigation:
		if m.navigationView != nil {
			return m.navigationView.View()
		}
		return "Navigation view not initialized"
	case ViewChromeDebugger:
		if m.chromeDebuggerView != nil {
			return m.chromeDebuggerView.View()
		}
		return "Chrome debugger view not initialized"
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

// loadActionsForAdventure loads available actions from the manifest
func (m *Model) loadActionsForAdventure() {
	// Try to load actions from manifest
	cwd, err := os.Getwd()
	if err != nil {
		return
	}
	
	loader := config.NewLoader(cwd)
	if !loader.IsInitialized() {
		return
	}
	
	projectRoot, err := loader.GetProjectRoot()
	if err != nil {
		return
	}
	
	manager := manifest.NewManager(projectRoot)
	
	manifestData, err := manager.LoadManifest()
	if err == nil && len(manifestData.Actions) > 0 {
		// Set up initial page state
		env := m.config.GetCurrentEnv()
		pageState := views.PageState{
			URL:         env.BaseURL,
			Title:       "Application Home",
			Description: "Choose an action to begin your testing adventure",
		}
		
		m.adventureView.SetPageState(pageState)
		m.adventureView.SetActions(manifestData.Actions)
	} else {
		// Create some sample actions if no manifest exists
		sampleActions := []types.CodeAction{
			{
				ID:          "navigate_to_login",
				Name:        "Navigate to Login",
				Category:    "Navigation",
				Type:        "page_visit",
				Description: "Go to the login page",
				Implementation: types.TechnicalDetails{
					Endpoint: "/login",
					Method:   "GET",
				},
				Inputs: []types.UserInput{},
				Expects: types.UserExpectation{
					Success: "You should see the login form",
				},
			},
			{
				ID:          "sign_in_with_email",
				Name:        "Sign in with email",
				Category:    "Authentication", 
				Type:        "form_submit",
				Description: "Log in using email and password",
				Implementation: types.TechnicalDetails{
					Endpoint: "/api/auth/login",
					Method:   "POST",
				},
				Inputs: []types.UserInput{
					{Name: "email", Label: "Email", Type: "email", Required: true, Example: "user@example.com"},
					{Name: "password", Label: "Password", Type: "password", Required: true, Example: "password123"},
				},
				Expects: types.UserExpectation{
					Success: "You should be redirected to the dashboard",
					Failure: "An error message appears",
				},
			},
			{
				ID:          "view_dashboard",
				Name:        "View Dashboard", 
				Category:    "Navigation",
				Type:        "page_visit",
				Description: "Navigate to the main dashboard",
				Implementation: types.TechnicalDetails{
					Endpoint: "/dashboard",
					Method:   "GET",
				},
				Expects: types.UserExpectation{
					Success: "Dashboard loads with user data",
				},
			},
		}
		
		m.adventureView.SetActions(sampleActions)
	}
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
	
	// Pass the program reference to ChatAdventureV2View if it exists
	if v2, ok := m.chatAdventureView.(*views.ChatAdventureV2View); ok {
		v2.SetProgram(p)
	}
}

// Message types
type ShowChromeDebuggerViewMsg struct{}
type ShowNavigationViewMsg struct{}