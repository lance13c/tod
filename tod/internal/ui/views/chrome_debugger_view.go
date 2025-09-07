package views

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ciciliostudio/tod/internal/browser"
	"github.com/ciciliostudio/tod/internal/config"
	"github.com/ciciliostudio/tod/internal/database"
	"github.com/ciciliostudio/tod/internal/email"
	"github.com/ciciliostudio/tod/internal/llm"
	"github.com/ciciliostudio/tod/internal/testing"
)

// ChromeDebuggerViewState represents the current state of the Chrome debugger view
type ChromeDebuggerViewState int

const (
	ChromeDebuggerStateLaunching ChromeDebuggerViewState = iota
	ChromeDebuggerStateScanning
	ChromeDebuggerStateCapturingHTML
	ChromeDebuggerStateAnalyzingActions
	ChromeDebuggerStateShowingResults
	ChromeDebuggerStateGeneratingTests
	ChromeDebuggerStateError
)

// ChromeDebuggerView handles Chrome debugger scanning and HTML viewing
type ChromeDebuggerView struct {
	state  ChromeDebuggerViewState
	width  int
	height int

	// UI components
	targetList list.Model
	viewport   viewport.Model

	// Data
	scanResults      []browser.DebuggerScanResult
	selectedTarget   *browser.DebuggerTarget
	htmlContent      string
	htmlSimplified   string
	error            error
	savedFilename    string
	chromeDPManager  *browser.ChromeDPManager
	configuredURL    string
	chromePort       int

	// Test discovery
	discoveredActions []TestAction
	existingTests     []string
	generatedTests    string

	// Database
	db               *database.DB
	currentCaptureID int64

	// Styles
	styles ChromeDebuggerViewStyles
}

// ChromeDebuggerViewStyles holds styling for the Chrome debugger view
type ChromeDebuggerViewStyles struct {
	Title    lipgloss.Style
	Subtitle lipgloss.Style
	Loading  lipgloss.Style
	Error    lipgloss.Style
	Success  lipgloss.Style
	Info     lipgloss.Style
	Border   lipgloss.Style
	Help     lipgloss.Style
	Code     lipgloss.Style
}

// ChromeTargetItem implements list.Item for Chrome debugger targets
type ChromeTargetItem struct {
	target browser.DebuggerTarget
	port   int
}

func (c ChromeTargetItem) Title() string { return c.target.Title }
func (c ChromeTargetItem) Description() string {
	return fmt.Sprintf("Port %d • %s", c.port, c.target.URL)
}
func (c ChromeTargetItem) FilterValue() string { return c.target.Title + " " + c.target.URL }

// TestAction represents a discovered test action
type TestAction struct {
	Description string
	Element     string
	Action      string
	IsTested    bool
}

// NewChromeDebuggerView creates a new Chrome debugger view
func NewChromeDebuggerView() *ChromeDebuggerView {
	// Load config to get URL
	cwd, _ := os.Getwd()
	loader := config.NewLoader(cwd)
	todConfig, _ := loader.Load()

	var configuredURL string
	if todConfig != nil {
		env := todConfig.GetCurrentEnv()
		if env.BaseURL != "" {
			configuredURL = env.BaseURL
		}
	}

	if configuredURL == "" {
		configuredURL = "http://localhost:3000" // fallback
	}

	// Initialize database
	dbPath := filepath.Join(cwd, ".tod", "captures.db")
	db, err := database.New(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open database: %v\n", err)
		// Continue without database
		db = nil
	}
	// Create list model with default size
	targetList := list.New([]list.Item{}, list.NewDefaultDelegate(), 80, 20)
	targetList.Title = "Chrome Debugger Targets"
	targetList.SetFilteringEnabled(true)
	targetList.SetShowHelp(false)
	targetList.DisableQuitKeybindings()

	// Create viewport for HTML viewing
	vp := viewport.New(80, 20)

	// Create styles
	styles := ChromeDebuggerViewStyles{
		Title: lipgloss.NewStyle().
			Foreground(lipgloss.Color("212")).
			Bold(true),
		Subtitle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("246")),
		Loading: lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")).
			Italic(true),
		Error: lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true),
		Success: lipgloss.NewStyle().
			Foreground(lipgloss.Color("46")).
			Bold(true),
		Info: lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")),
		Border: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(1, 2),
		Help: lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")),
		Code: lipgloss.NewStyle().
			Foreground(lipgloss.Color("246")).
			Background(lipgloss.Color("236")),
	}

	return &ChromeDebuggerView{
		state:         ChromeDebuggerStateLaunching,
		targetList:    targetList,
		viewport:      vp,
		styles:        styles,
		configuredURL: configuredURL,
		chromePort:    9229,
		db:            db,
	}
}

// Init initializes the Chrome debugger view
func (v *ChromeDebuggerView) Init() tea.Cmd {
	// Launch Chrome first, then start scanning after a delay
	return v.launchChrome()
}

// Update handles updates
func (v *ChromeDebuggerView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
		v.targetList.SetSize(msg.Width-4, msg.Height-8)
		v.viewport.Width = msg.Width - 4
		v.viewport.Height = msg.Height - 8

	case tea.KeyMsg:
		return v.handleKeyMsg(msg)

	case ChromeLaunchedMsg:
		v.state = ChromeDebuggerStateScanning
		// Start SMTP monitoring if configured
		v.startSMTPMonitoring()
		// Start scanning after Chrome has launched
		return v, v.continuousScanning()

	case ChromeDebuggerFoundMsg:
		v.selectedTarget = &msg.Target
		v.state = ChromeDebuggerStateCapturingHTML
		return v, v.captureAndAnalyzeHTML(msg.Target)

	case HTMLCapturedMsg:
		v.htmlContent = msg.HTML
		v.savedFilename = msg.Filename

		// Save page capture to database
		if v.db != nil && v.selectedTarget != nil {
			capture := &database.PageCapture{
				URL:          v.selectedTarget.URL,
				Title:        v.selectedTarget.Title,
				HTMLFile:     msg.Filename,
				HTMLLength:   len(msg.HTML),
				CapturedAt:   time.Now(),
				ChromePort:   v.chromePort,
				WebSocketURL: v.selectedTarget.WebSocketDebuggerURL,
			}

			captureID, err := v.db.SavePageCapture(capture)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to save page capture to database: %v\n", err)
			} else {
				v.currentCaptureID = captureID
				fmt.Fprintf(os.Stderr, "Saved page capture to database with ID: %d\n", captureID)
			}
		}

		v.state = ChromeDebuggerStateAnalyzingActions
		return v, v.analyzeActionsWithLLM()

	case ActionsDiscoveredMsg:
		v.discoveredActions = msg.Actions

		// Save discovered actions to database
		if v.db != nil && v.currentCaptureID > 0 {
			var dbActions []database.DiscoveredAction
			for _, action := range msg.Actions {
				dbAction := database.DiscoveredAction{
					CaptureID:   v.currentCaptureID,
					Description: action.Description,
					Element:     action.Element,
					Selector:    action.Element, // Using Element as selector
					Action:      action.Action,
					IsTested:    action.IsTested,
					Priority:    "medium", // Default priority
				}
				dbActions = append(dbActions, dbAction)
			}

			if err := v.db.SaveDiscoveredActions(v.currentCaptureID, dbActions); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to save discovered actions to database: %v\n", err)
			} else {
				fmt.Fprintf(os.Stderr, "Saved %d discovered actions to database\n", len(dbActions))
			}
		}

		v.state = ChromeDebuggerStateShowingResults
		return v, nil

	case ChromeDebuggerScanCompleteMsg:
		v.scanResults = msg.Results
		if len(v.scanResults) == 0 {
			v.state = ChromeDebuggerStateError
			v.error = fmt.Errorf("no Chrome debugger instances found")
		} else {
			// Continue scanning for HTML
			v.state = ChromeDebuggerStateCapturingHTML
		}
		return v, nil

	case ChromeDebuggerHTMLFetchedMsg:
		v.htmlContent = msg.HTML
		v.savedFilename = "" // Clear any previous save filename
		v.viewport.SetContent(v.formatHTML(msg.HTML))
		return v, nil

	case ChromeDebuggerSaveSuccessMsg:
		v.savedFilename = msg.Filename
		return v, nil

	case TestsGeneratedMsg:
		v.generatedTests = msg.TestCode

		// Save test generation to database
		if v.db != nil && v.currentCaptureID > 0 && msg.TestCode != "" {
			testGen := &database.TestGeneration{
				CaptureID: v.currentCaptureID,
				Framework: "playwright", // Default, could be detected
				TestCode:  msg.TestCode,
				FileName:  fmt.Sprintf("generated_test_%d.spec.js", v.currentCaptureID),
			}

			if _, err := v.db.SaveTestGeneration(testGen); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to save test generation: %v\n", err)
			}
		}

		// Show the generated tests in a viewport
		v.state = ChromeDebuggerStateShowingResults
		v.viewport.SetContent(v.formatGeneratedTests())
		return v, nil

	case ChromeDebuggerErrorMsg:
		v.error = msg.Error
		v.state = ChromeDebuggerStateError
		return v, nil
	}

	// Update viewport if showing results
	switch v.state {
	case ChromeDebuggerStateShowingResults:
		var cmd tea.Cmd
		v.viewport, cmd = v.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return v, tea.Batch(cmds...)
}

// View renders the Chrome debugger view
func (v *ChromeDebuggerView) View() string {
	switch v.state {
	case ChromeDebuggerStateLaunching:
		return v.renderLaunching()
	case ChromeDebuggerStateScanning:
		return v.renderScanning()
	case ChromeDebuggerStateCapturingHTML:
		return v.renderCapturing()
	case ChromeDebuggerStateAnalyzingActions:
		return v.renderAnalyzing()
	case ChromeDebuggerStateShowingResults:
		return v.renderResults()
	case ChromeDebuggerStateGeneratingTests:
		return v.renderGeneratingTests()
	case ChromeDebuggerStateError:
		return v.renderError()
	default:
		return "Unknown state"
	}
}

// Render methods

func (v *ChromeDebuggerView) renderLaunching() string {
	title := v.styles.Title.Render("Chrome Test Discovery")
	loading := v.styles.Loading.Render("Launching Chrome in headless mode with debugging enabled...")

	content := fmt.Sprintf("%s\n\n%s\n\nOpening: %s\n\nPress Ctrl+C to cancel",
		title, loading, v.configuredURL)

	return v.styles.Border.Render(content)
}

func (v *ChromeDebuggerView) renderScanning() string {
	title := v.styles.Title.Render("Chrome Test Discovery")
	loading := v.styles.Loading.Render("Scanning for Chrome debugger...")

	content := fmt.Sprintf("%s\n\n%s\n\nWaiting for Chrome to be ready...\n\nPress Ctrl+C to cancel",
		title, loading)

	return v.styles.Border.Render(content)
}

func (v *ChromeDebuggerView) renderCapturing() string {
	title := v.styles.Title.Render("Chrome Test Discovery")
	loading := v.styles.Loading.Render("Capturing page HTML...")

	content := fmt.Sprintf("%s\n\n%s\n\nPage: %s",
		title, loading, v.configuredURL)

	return v.styles.Border.Render(content)
}

func (v *ChromeDebuggerView) renderAnalyzing() string {
	title := v.styles.Title.Render("Chrome Test Discovery")
	loading := v.styles.Loading.Render("Analyzing page for untested actions...")

	content := fmt.Sprintf("%s\n\n%s\n\nUsing AI to identify test opportunities...",
		title, loading)

	return v.styles.Border.Render(content)
}

func (v *ChromeDebuggerView) renderGeneratingTests() string {
	title := v.styles.Title.Render("Test Generation")
	loading := v.styles.Loading.Render("Generating test code...")

	content := fmt.Sprintf("%s\n\n%s\n\nCreating test scenarios for discovered actions...",
		title, loading)

	return v.styles.Border.Render(content)
}

func (v *ChromeDebuggerView) renderResults() string {
	title := v.styles.Title.Render("Test Discovery Results")

	var content strings.Builder
	content.WriteString(fmt.Sprintf("%s\n\n", title))

	// Show page info
	if v.selectedTarget != nil {
		content.WriteString(v.styles.Info.Render(fmt.Sprintf("Page: %s\n", v.selectedTarget.Title)))
		content.WriteString(v.styles.Info.Render(fmt.Sprintf("URL: %s\n\n", v.selectedTarget.URL)))
	}

	if len(v.discoveredActions) == 0 {
		content.WriteString(v.styles.Success.Render("✓ All user actions appear to be tested!"))
	} else {
		untestedCount := 0
		for _, action := range v.discoveredActions {
			if !action.IsTested {
				untestedCount++
			}
		}
		content.WriteString(v.styles.Info.Render(fmt.Sprintf("Found %d actions (%d untested):\n\n", len(v.discoveredActions), untestedCount)))
		for i, action := range v.discoveredActions {
			status := "❌"
			if action.IsTested {
				status = "✅"
			}
			content.WriteString(fmt.Sprintf("%s %d. %s\n   Element: %s\n   Action: %s\n\n",
				status, i+1, action.Description, action.Element, action.Action))
		}
	}

	// Show database stats if available
	if v.db != nil {
		stats, err := v.db.GetStatistics()
		if err == nil {
			content.WriteString("\n")
			content.WriteString(v.styles.Subtitle.Render("Database Statistics:\n"))
			content.WriteString(fmt.Sprintf("  Total Captures: %d\n", stats["total_captures"]))
			content.WriteString(fmt.Sprintf("  Total Actions: %d\n", stats["total_actions"]))
			content.WriteString(fmt.Sprintf("  Untested Actions: %d\n", stats["untested_actions"]))
			content.WriteString(fmt.Sprintf("  LLM Interactions: %d\n", stats["llm_interactions"]))
		}

		// Show LLM prompt/response info for current capture
		if v.currentCaptureID > 0 {
			llmInteractions, err := v.db.GetLLMInteractions(v.currentCaptureID)
			if err == nil && len(llmInteractions) > 0 {
				content.WriteString("\n")
				content.WriteString(v.styles.Subtitle.Render("LLM Analysis:\n"))
				for _, interaction := range llmInteractions {
					content.WriteString(fmt.Sprintf("  Type: %s\n", interaction.InteractionType))
					content.WriteString(fmt.Sprintf("  Provider: %s (%s)\n", interaction.Provider, interaction.Model))
					content.WriteString(fmt.Sprintf("  Prompt Length: %d chars\n", len(interaction.Prompt)))
					content.WriteString(fmt.Sprintf("  Response Length: %d chars\n", len(interaction.Response)))
					if interaction.Error != "" {
						content.WriteString(fmt.Sprintf("  Error: %s\n", interaction.Error))
					}
				}
			}
		}
	}

	// Show if tests have been generated
	if v.generatedTests != "" {
		content.WriteString("\n")
		content.WriteString(v.styles.Success.Render("✓ Tests generated and saved to file\n"))
	}

	content.WriteString("\n")
	if v.generatedTests == "" {
		content.WriteString(v.styles.Help.Render("Press 'g' to generate tests • 's' to save HTML • 'q' to quit"))
	} else {
		content.WriteString(v.styles.Help.Render("Press 'v' to view generated tests • 's' to save HTML • 'q' to quit"))
	}

	return v.styles.Border.Render(content.String())
}

func (v *ChromeDebuggerView) renderSelection() string {
	title := v.styles.Title.Render("Chrome Debugger Targets")

	totalTargets := 0
	for _, result := range v.scanResults {
		totalTargets += len(result.Targets)
	}

	subtitle := v.styles.Subtitle.Render(fmt.Sprintf("Found %d targets across %d ports",
		totalTargets, len(v.scanResults)))

	// Ensure list has proper dimensions
	if v.targetList.Width() == 0 {
		v.targetList.SetSize(v.width-4, v.height-10)
	}

	listView := v.targetList.View()
	help := v.styles.Help.Render("↑↓: navigate • enter: view HTML • q: back • /: search")

	// Build the complete view
	var content strings.Builder
	content.WriteString(title)
	content.WriteString("\n")
	content.WriteString(subtitle)
	content.WriteString("\n\n")
	content.WriteString(listView)
	content.WriteString("\n")
	content.WriteString(help)

	return content.String()
}

func (v *ChromeDebuggerView) renderHTML() string {
	if v.selectedTarget == nil {
		return "No target selected"
	}

	title := v.styles.Title.Render("Page HTML - Full Document")
	subtitle := v.styles.Subtitle.Render(v.selectedTarget.Title)
	url := v.styles.Info.Render(v.selectedTarget.URL)

	header := fmt.Sprintf("%s\n%s\n%s\n", title, subtitle, url)

	// Show save success message if file was just saved
	if v.savedFilename != "" {
		saveMsg := v.styles.Success.Render(fmt.Sprintf("✓ Saved to: %s", v.savedFilename))
		header += saveMsg + "\n\n"
	}

	help := v.styles.Help.Render("↑↓: scroll • s: save to file • q: back to list • ctrl+c: exit")

	return header + v.viewport.View() + "\n" + help
}

func (v *ChromeDebuggerView) renderError() string {
	title := v.styles.Title.Render("Error")
	errorMsg := v.styles.Error.Render(v.error.Error())

	content := fmt.Sprintf("%s\n\n%s\n\n%s",
		title,
		errorMsg,
		v.styles.Help.Render("Press 'q' to go back • 'r' to retry"))

	return v.styles.Border.Render(content)
}

func (v *ChromeDebuggerView) renderNoTargets() string {
	title := v.styles.Title.Render("No Chrome Debugger Found")

	content := fmt.Sprintf(`%s

No Chrome debugger instances found on common ports.

To enable Chrome debugging:

Method 1: With a temporary profile (recommended):
   %s

Method 2: With your default profile:
   %s

Method 3: Keep existing Chrome and open new instance:
   %s

Note: 
- You may need to close ALL Chrome instances first
- On macOS, make sure to use the full path to Chrome
- The --user-data-dir flag creates a separate profile

%s`,
		title,
		v.styles.Code.Render("/Applications/Google\\ Chrome.app/Contents/MacOS/Google\\ Chrome --remote-debugging-port=9222 --user-data-dir=/tmp/chrome-debug"),
		v.styles.Code.Render("/Applications/Google\\ Chrome.app/Contents/MacOS/Google\\ Chrome --remote-debugging-port=9222"),
		v.styles.Code.Render("open -na 'Google Chrome' --args --remote-debugging-port=9222 --user-data-dir=/tmp/chrome-debug"),
		v.styles.Help.Render("Press 'q' to go back • 'r' to retry"))

	return v.styles.Border.Render(content)
}

// Event handlers

func (v *ChromeDebuggerView) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch v.state {
	case ChromeDebuggerStateLaunching, ChromeDebuggerStateScanning, ChromeDebuggerStateCapturingHTML, ChromeDebuggerStateAnalyzingActions, ChromeDebuggerStateGeneratingTests:
		switch msg.String() {
		case "ctrl+c":
			// Clean up Chrome and database
			browser.CloseGlobalChromeDPManager()
			if v.db != nil {
				v.db.Close()
			}
			return v, tea.Quit
		case "q", "esc":
			// Clean up Chrome
			browser.CloseGlobalChromeDPManager()
			if v.db != nil {
				v.db.Close()
			}
			return v, returnToMenu()
		}

	case ChromeDebuggerStateShowingResults:
		switch msg.String() {
		case "q", "esc":
			// Clean up Chrome and database
			browser.CloseGlobalChromeDPManager()
			if v.db != nil {
				v.db.Close()
			}
			return v, returnToMenu()
		case "g":
			// Generate tests for discovered actions
			if len(v.discoveredActions) > 0 {
				v.state = ChromeDebuggerStateGeneratingTests
				return v, v.generateTestsCmd()
			}
			return v, nil
		case "v":
			// View generated tests if available
			if v.generatedTests != "" {
				v.viewport.SetContent(v.formatGeneratedTests())
			}
			return v, nil
		case "s":
			// Save HTML is already done
			return v, nil
		case "ctrl+c":
			// Clean up Chrome and database
			browser.CloseGlobalChromeDPManager()
			if v.db != nil {
				v.db.Close()
			}
			return v, tea.Quit
		}

	case ChromeDebuggerStateError:
		switch msg.String() {
		case "q", "esc":
			return v, returnToMenu()
		case "ctrl+c":
			// Clean up Chrome and database
			browser.CloseGlobalChromeDPManager()
			if v.db != nil {
				v.db.Close()
			}
			return v, tea.Quit
		case "r":
			v.state = ChromeDebuggerStateLaunching
			return v, tea.Batch(
				v.launchChrome(),
				v.continuousScanning(),
			)
		}

	default:
		switch msg.String() {
		case "q", "esc":
			return v, returnToMenu()
		case "ctrl+c":
			return v, tea.Quit
		}
	}

	return v, nil
}

// Commands

func (v *ChromeDebuggerView) launchChrome() tea.Cmd {
	return func() tea.Msg {
		// Log what we're launching
		fmt.Fprintf(os.Stderr, "Launching Chrome in headless mode with URL: %s\n", v.configuredURL)

		// Create ChromeDP manager
		manager, err := browser.GetGlobalChromeDPManager(v.configuredURL, true)
		if err != nil {
			return ChromeDebuggerErrorMsg{Error: fmt.Errorf("failed to launch Chrome: %w", err)}
		}

		v.chromeDPManager = manager

		// Update state to scanning and start continuous scanning
		return ChromeLaunchedMsg{}
	}
}

func (v *ChromeDebuggerView) continuousScanning() tea.Cmd {
	return func() tea.Msg {
		// Give Chrome a moment to fully initialize
		time.Sleep(2 * time.Second)

		// Check if ChromeDP manager is ready
		if v.chromeDPManager == nil {
			return ChromeDebuggerErrorMsg{Error: fmt.Errorf("Chrome manager not initialized")}
		}

		// Get page info
		url, title, err := v.chromeDPManager.GetPageInfo()
		if err != nil {
			return ChromeDebuggerErrorMsg{Error: fmt.Errorf("failed to get page info: %w", err)}
		}

		// Create a pseudo target for compatibility
		target := browser.DebuggerTarget{
			Type:  "page",
			URL:   url,
			Title: title,
		}

		fmt.Fprintf(os.Stderr, "Chrome connected: URL=%s, Title=%s\n", url, title)

		return ChromeDebuggerFoundMsg{Target: target}
	}
}

func (v *ChromeDebuggerView) captureAndAnalyzeHTML(target browser.DebuggerTarget) tea.Cmd {
	return func() tea.Msg {
		// Check if ChromeDP manager is ready
		if v.chromeDPManager == nil {
			return ChromeDebuggerErrorMsg{Error: fmt.Errorf("Chrome manager not initialized")}
		}

		// Capture HTML using chromedp
		html, err := v.chromeDPManager.GetPageHTML()
		if err != nil {
			return ChromeDebuggerErrorMsg{Error: fmt.Errorf("failed to capture HTML: %w", err)}
		}

		// Save HTML copy
		timestamp := time.Now().Format("2006-01-02_15-04-05")
		htmlFile := fmt.Sprintf("tod_capture_%s.html", timestamp)
		os.WriteFile(htmlFile, []byte(html), 0644)

		fmt.Fprintf(os.Stderr, "Captured HTML from %s (%d bytes)\n", target.URL, len(html))

		return HTMLCapturedMsg{HTML: html, Filename: htmlFile}
	}
}

func (v *ChromeDebuggerView) analyzeActionsWithLLM() tea.Cmd {
	return func() tea.Msg {
		// Get LLM client from config
		cwd, _ := os.Getwd()
		loader := config.NewLoader(cwd)
		todConfig, _ := loader.Load()

		if todConfig == nil || todConfig.AI.Provider == "" {
			// No LLM configured, use basic analysis
			return v.basicActionAnalysis()
		}

		// Create LLM client
		provider := llm.Provider(todConfig.AI.Provider)
		// Ensure model is in settings
		settings := todConfig.AI.Settings
		if settings == nil {
			settings = make(map[string]interface{})
		}
		if todConfig.AI.Model != "" {
			settings["model"] = todConfig.AI.Model
		}
		llmClient, err := llm.NewClient(provider, todConfig.AI.APIKey, settings)
		if err != nil {
			// Fall back to basic analysis
			return v.basicActionAnalysis()
		}

		// Create action discovery
		discovery := testing.NewActionDiscovery(llmClient, cwd)

		// Read existing test files
		existingTests := v.findExistingTests()

		// Discover actions (increased timeout for AI models)
		ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
		defer cancel()

		actions, llmPrompt, llmResponse, err := discovery.DiscoverActionsFromHTML(ctx, v.htmlContent, existingTests)

		// Save LLM interaction to database
		if v.db != nil && v.currentCaptureID > 0 {
			llmInteraction := &database.LLMInteraction{
				CaptureID:       v.currentCaptureID,
				InteractionType: "action_discovery",
				Provider:        todConfig.AI.Provider,
				Model:           todConfig.AI.Model,
				Prompt:          llmPrompt,
				Response:        llmResponse,
				CreatedAt:       time.Now(),
			}

			if err != nil {
				llmInteraction.Error = err.Error()
			}

			if _, saveErr := v.db.SaveLLMInteraction(llmInteraction); saveErr != nil {
				fmt.Fprintf(os.Stderr, "Failed to save LLM interaction: %v\n", saveErr)
			} else {
				fmt.Fprintf(os.Stderr, "Saved LLM interaction to database\n")
			}
		}

		if err != nil {
			return ChromeDebuggerErrorMsg{Error: fmt.Errorf("action discovery failed: %w", err)}
		}

		// Convert to view format
		var testActions []TestAction
		for _, action := range actions {
			testActions = append(testActions, TestAction{
				Description: action.Description,
				Element:     action.Selector,
				Action:      action.Action,
				IsTested:    action.IsTested,
			})
		}

		return ActionsDiscoveredMsg{Actions: testActions}
	}
}

func (v *ChromeDebuggerView) basicActionAnalysis() tea.Msg {
	// Extract interactive elements without LLM
	elements, err := browser.ExtractInteractiveElements(v.htmlContent)
	if err != nil {
		return ChromeDebuggerErrorMsg{Error: err}
	}

	var testActions []TestAction
	for _, elem := range elements {
		if elem.Selector != "" {
			action := "click"
			if elem.Tag == "input" || elem.Tag == "textarea" {
				action = "type"
			} else if elem.Tag == "select" {
				action = "select"
			}

			testActions = append(testActions, TestAction{
				Description: fmt.Sprintf("%s %s", action, elem.Text),
				Element:     elem.Selector,
				Action:      action,
				IsTested:    false,
			})
		}
	}

	return ActionsDiscoveredMsg{Actions: testActions}
}

func (v *ChromeDebuggerView) findExistingTests() []string {
	var tests []string

	// Look for common test directories
	testDirs := []string{"tests", "test", "e2e", "cypress", "__tests__"}
	for _, dir := range testDirs {
		if files, err := filepath.Glob(filepath.Join(dir, "*.spec.*")); err == nil {
			for _, file := range files {
				if content, err := os.ReadFile(file); err == nil {
					tests = append(tests, string(content))
				}
			}
		}
		if files, err := filepath.Glob(filepath.Join(dir, "*.test.*")); err == nil {
			for _, file := range files {
				if content, err := os.ReadFile(file); err == nil {
					tests = append(tests, string(content))
				}
			}
		}
	}

	return tests
}

func (v *ChromeDebuggerView) scanForDebuggers() tea.Cmd {
	return func() tea.Msg {
		results, err := browser.ScanForChromeDebugger()
		if err != nil {
			return ChromeDebuggerErrorMsg{Error: err}
		}
		return ChromeDebuggerScanCompleteMsg{Results: results}
	}
}

func (v *ChromeDebuggerView) fetchHTML(target *browser.DebuggerTarget) tea.Cmd {
	return func() tea.Msg {
		// ONLY use the direct WebSocket approach to avoid opening new windows
		html, err := browser.GetPageHTMLDirect(target.WebSocketDebuggerURL)

		if err != nil {
			// If direct WebSocket fails, return the error
			// Don't try other methods that open new windows
			return ChromeDebuggerErrorMsg{Error: fmt.Errorf("failed to get HTML via WebSocket: %w", err)}
		}

		// If we got empty or minimal HTML, add a note but still show it
		if len(html) < 100 {
			html = fmt.Sprintf("<!-- WARNING: Minimal HTML captured (%d bytes) -->\n<!-- The page might be protected or using heavy JavaScript rendering -->\n\n%s", len(html), html)
		}

		// Add metadata about the capture
		metadata := fmt.Sprintf("<!-- Captured from: %s -->\n<!-- Title: %s -->\n<!-- Type: %s -->\n<!-- Captured at: %s -->\n<!-- Content Length: %d bytes -->\n\n",
			target.URL, target.Title, target.Type, time.Now().Format(time.RFC3339), len(html))
		html = metadata + html

		return ChromeDebuggerHTMLFetchedMsg{HTML: html}
	}
}

func (v *ChromeDebuggerView) generateTestsCmd() tea.Cmd {
	return func() tea.Msg {
		// Setup logging
		logFile, err := os.OpenFile(".tod/api_calls.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			log.Printf("Failed to open log file: %v", err)
		} else {
			defer logFile.Close()
			logger := log.New(logFile, "[TEST_GEN] ", log.LstdFlags|log.Lmicroseconds)
			logger.Printf("Starting test generation...")
		}

		// Get LLM client from config
		cwd, _ := os.Getwd()
		loader := config.NewLoader(cwd)
		todConfig, _ := loader.Load()

		if todConfig == nil || todConfig.AI.Provider == "" {
			err := fmt.Errorf("no LLM configured")
			if logFile != nil {
				logger := log.New(logFile, "[TEST_GEN] ", log.LstdFlags|log.Lmicroseconds)
				logger.Printf("ERROR: %v", err)
			}
			return ChromeDebuggerErrorMsg{Error: err}
		}

		if logFile != nil {
			logger := log.New(logFile, "[TEST_GEN] ", log.LstdFlags|log.Lmicroseconds)
			logger.Printf("Using provider: %s, Model: %s", todConfig.AI.Provider, todConfig.AI.Model)
		}

		// Create LLM client
		provider := llm.Provider(todConfig.AI.Provider)
		// Ensure model is in settings
		settings := todConfig.AI.Settings
		if settings == nil {
			settings = make(map[string]interface{})
		}
		if todConfig.AI.Model != "" {
			settings["model"] = todConfig.AI.Model
		}
		llmClient, err := llm.NewClient(provider, todConfig.AI.APIKey, settings)
		if err != nil {
			if logFile != nil {
				logger := log.New(logFile, "[TEST_GEN] ", log.LstdFlags|log.Lmicroseconds)
				logger.Printf("ERROR creating LLM client: %v", err)
			}
			return ChromeDebuggerErrorMsg{Error: fmt.Errorf("failed to create LLM client: %w", err)}
		}

		if logFile != nil {
			logger := log.New(logFile, "[TEST_GEN] ", log.LstdFlags|log.Lmicroseconds)
			logger.Printf("LLM client created successfully")
		}

		// Create action discovery
		discovery := testing.NewActionDiscovery(llmClient, cwd)

		// Convert TestAction to DiscoveredAction
		var discoveredActions []testing.DiscoveredAction
		for _, action := range v.discoveredActions {
			discoveredActions = append(discoveredActions, testing.DiscoveredAction{
				Description: action.Description,
				Selector:    action.Element,
				Action:      action.Action,
				IsTested:    action.IsTested,
				Priority:    "medium",
			})
		}

		if logFile != nil {
			logger := log.New(logFile, "[TEST_GEN] ", log.LstdFlags|log.Lmicroseconds)
			logger.Printf("Processing %d discovered actions", len(discoveredActions))
		}

		// Detect testing framework
		framework := "playwright" // Default
		if _, err := os.Stat("cypress.config.js"); err == nil {
			framework = "cypress"
		}

		// Generate tests (increased timeout for AI models)
		ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
		defer cancel()

		if logFile != nil {
			logger := log.New(logFile, "[TEST_GEN] ", log.LstdFlags|log.Lmicroseconds)
			logger.Printf("Calling GenerateTestSuggestions with framework: %s", framework)
		}

		testCode, err := discovery.GenerateTestSuggestions(ctx, discoveredActions, framework)
		if err != nil {
			if logFile != nil {
				logger := log.New(logFile, "[TEST_GEN] ", log.LstdFlags|log.Lmicroseconds)
				logger.Printf("ERROR generating tests: %v", err)
			}
			return ChromeDebuggerErrorMsg{Error: fmt.Errorf("failed to generate tests: %w", err)}
		}

		if logFile != nil {
			logger := log.New(logFile, "[TEST_GEN] ", log.LstdFlags|log.Lmicroseconds)
			logger.Printf("Generated test code, length: %d characters", len(testCode))
		}

		// Save test file
		timestamp := time.Now().Format("2006-01-02_15-04-05")
		filename := fmt.Sprintf("generated_test_%s.spec.js", timestamp)
		if err := os.WriteFile(filename, []byte(testCode), 0644); err == nil {
			fmt.Fprintf(os.Stderr, "Tests saved to: %s\n", filename)
		}

		return TestsGeneratedMsg{TestCode: testCode}
	}
}

func (v *ChromeDebuggerView) saveHTMLToFile() tea.Cmd {
	return func() tea.Msg {
		if v.selectedTarget == nil || v.htmlContent == "" {
			return ChromeDebuggerErrorMsg{Error: fmt.Errorf("no HTML content to save")}
		}

		// Generate filename based on page title and timestamp
		timestamp := time.Now().Format("2006-01-02_15-04-05")
		safeTitle := strings.ReplaceAll(v.selectedTarget.Title, "/", "-")
		safeTitle = strings.ReplaceAll(safeTitle, " ", "_")
		if len(safeTitle) > 50 {
			safeTitle = safeTitle[:50]
		}
		filename := fmt.Sprintf("%s_%s.html", safeTitle, timestamp)

		// Save to current directory
		if err := os.WriteFile(filename, []byte(v.htmlContent), 0644); err != nil {
			return ChromeDebuggerErrorMsg{Error: fmt.Errorf("failed to save HTML: %w", err)}
		}

		return ChromeDebuggerSaveSuccessMsg{Filename: filename}
	}
}

// Helper methods

func (v *ChromeDebuggerView) updateTargetList() {
	var items []list.Item

	for _, result := range v.scanResults {
		for _, target := range result.Targets {
			items = append(items, ChromeTargetItem{
				target: target,
				port:   result.Port,
			})
		}
	}

	v.targetList.SetItems(items)
}

func (v *ChromeDebuggerView) formatHTML(html string) string {
	// Basic formatting for better readability
	// Add line numbers and proper indentation
	lines := strings.Split(html, "\n")
	var formatted strings.Builder

	for i, line := range lines {
		formatted.WriteString(fmt.Sprintf("%4d | %s\n", i+1, line))
	}

	return formatted.String()
}

func (v *ChromeDebuggerView) formatGeneratedTests() string {
	if v.generatedTests == "" {
		return "No tests generated yet."
	}

	// Format the generated test code with line numbers
	lines := strings.Split(v.generatedTests, "\n")
	var formatted strings.Builder

	formatted.WriteString(v.styles.Title.Render("Generated Test Code"))
	formatted.WriteString("\n\n")

	for i, line := range lines {
		formatted.WriteString(fmt.Sprintf("%4d | %s\n", i+1, line))
	}

	formatted.WriteString("\n")
	formatted.WriteString(v.styles.Help.Render("Tests have been saved to generated_test_*.spec.js"))

	return formatted.String()
}

// startSMTPMonitoring starts the SMTP monitoring service if configured
func (v *ChromeDebuggerView) startSMTPMonitoring() {
	// Get current working directory to find the project config
	cwd, err := os.Getwd()
	if err != nil {
		return
	}
	
	// Load config to check if SMTP monitoring is enabled
	loader := config.NewLoader(cwd)
	todConfig, err := loader.Load()
	if err != nil {
		return
	}
	
	// Check if email config exists and fetch_on_chrome_connect is enabled
	if todConfig != nil && todConfig.Email != nil {
		// Check for fetch_on_chrome_connect setting
		if fetchOnConnect, ok := todConfig.Email["fetch_on_chrome_connect"].(bool); ok && fetchOnConnect {
			// Get the monitor service
			monitorService := email.GetMonitorService(cwd)
			
			// Start background monitoring
			if err := monitorService.StartBackgroundMonitoring(); err != nil {
				// Log error but don't fail Chrome launch
				fmt.Fprintf(os.Stderr, "Failed to start SMTP monitoring: %v\n", err)
			} else {
				fmt.Fprintf(os.Stderr, "SMTP monitoring started successfully\n")
			}
		}
	}
}

// Message types

type ChromeLaunchedMsg struct{}

type ChromeDebuggerFoundMsg struct {
	Target browser.DebuggerTarget
}

type HTMLCapturedMsg struct {
	HTML     string
	Filename string
}

type ChromeDebuggerScanCompleteMsg struct {
	Results []browser.DebuggerScanResult
}

type ChromeDebuggerHTMLFetchedMsg struct {
	HTML string
}

type ChromeDebuggerErrorMsg struct {
	Error error
}

type ChromeDebuggerSaveSuccessMsg struct {
	Filename string
}

type ActionsDiscoveredMsg struct {
	Actions []TestAction
}

type TestsGeneratedMsg struct {
	TestCode string
}

// ReturnToMenuMsg signals to return to the main menu
type ReturnToMenuMsg struct{}

// Helper function to return to menu
func returnToMenu() tea.Cmd {
	return func() tea.Msg {
		return ReturnToMenuMsg{}
	}
}
