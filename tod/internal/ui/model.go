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
	ViewFlow
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
	
	// Views
	adventureView *views.AdventureView
	flowView      *views.FlowView
	
	// Menu state
	menuList list.Model
	
	// Global styles
	styles *Styles
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

// NewModelWithRoot creates a new main model with project root
func NewModelWithRoot(cfg *config.Config, projectRoot string) *Model {
	// Create menu items
	items := []list.Item{
		MenuItem{
			title:       "AI Flow Discovery",
			description: "Let AI find and run flows in your app",
			action: func() tea.Cmd {
				return showFlowView()
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

	// Initialize flow view (but don't create it until needed)
	var flowView *views.FlowView

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
		flowView:      flowView,
		menuList:      menuList,
		styles:        NewStyles(),
	}
}

// showFlowView returns a command to show the flow view
func showFlowView() tea.Cmd {
	return func() tea.Msg {
		return ShowFlowViewMsg{}
	}
}

// Init implements tea.Model
func (m *Model) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "esc":
			if m.currentView != ViewMenu {
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
		m.adventureView, _ = m.adventureView.Update(msg)
		
		// Update flow view if it exists
		if m.flowView != nil {
			var flowModel tea.Model
			flowModel, _ = m.flowView.Update(msg)
			m.flowView = flowModel.(*views.FlowView)
		}

	case ShowFlowViewMsg:
		// Initialize flow view if not already created
		if m.flowView == nil {
			var err error
			m.flowView, err = views.NewFlowView(m.config, m.projectRoot)
			if err != nil {
				// Handle error - maybe show an error view or return to menu
				return m, nil
			}
		}
		m.currentView = ViewFlow
		return m, m.flowView.Init()
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
			case "AI Flow Discovery":
				return m, showFlowView()
			case "Start New Journey":
				m.currentView = ViewAdventure
				// Load available actions
				m.loadActionsForAdventure()
			}
		}
		
	case ViewAdventure:
		var adventureCmd tea.Cmd
		m.adventureView, adventureCmd = m.adventureView.Update(msg)
		return m, adventureCmd

	case ViewFlow:
		if m.flowView != nil {
			var flowCmd tea.Cmd
			var flowModel tea.Model
			flowModel, flowCmd = m.flowView.Update(msg)
			m.flowView = flowModel.(*views.FlowView)
			return m, flowCmd
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
	case ViewFlow:
		if m.flowView != nil {
			return m.flowView.View()
		}
		return "Flow view not initialized"
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

// Message types
type ShowFlowViewMsg struct{}