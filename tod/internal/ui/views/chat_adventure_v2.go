package views

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ciciliostudio/tod/internal/browser"
	"github.com/ciciliostudio/tod/internal/config"
	"github.com/ciciliostudio/tod/internal/database"
	"github.com/ciciliostudio/tod/internal/llm"
	"github.com/ciciliostudio/tod/internal/testing"
	"github.com/ciciliostudio/tod/internal/types"
)

// ChatAdventureV2View provides an enhanced conversational interface for testing
type ChatAdventureV2View struct {
	// Configuration
	config       *config.Config
	llmClient    llm.Client
	configuredURL string
	
	// Database
	db           *database.DB
	captureID    int64
	
	// Browser management
	chromeDPManager *browser.ChromeDPManager
	isConnected     bool
	currentURL      string
	currentTitle    string
	lastTitle       string // Track title changes
	beforeHTML      string // HTML before action
	beforeURL       string // URL before action
	
	// Debouncing
	lastActionTime  time.Time
	debounceDelay   time.Duration
	
	// Chat interface
	messages     []ChatMessage
	viewport     viewport.Model
	textarea     textarea.Model
	
	// Action discovery
	actionDiscovery  *testing.ActionDiscovery
	discoveredActions []testing.DiscoveredAction
	awaitingSelection bool
	
	// UI state
	width        int
	height       int
	isProcessing bool
	isNavigating bool // Track navigation state to prevent loops
	spinner      spinner.Model // Loading animation
	program      *tea.Program  // Reference to the program for sending messages
	
	// Token tracking
	totalTokens int
	totalCost   float64
	lastPromptTokens int
	lastPromptCost   float64
	
	// Styles
	userStyle      lipgloss.Style
	assistantStyle lipgloss.Style
	systemStyle    lipgloss.Style
	errorStyle     lipgloss.Style
	thinkingStyle  lipgloss.Style
	borderStyle    lipgloss.Style
	titleStyle     lipgloss.Style
	statsStyle     lipgloss.Style
}

// NewChatAdventureV2View creates a new enhanced chat-based adventure view
func NewChatAdventureV2View(cfg *config.Config) *ChatAdventureV2View {
	// Create textarea for user input
	ta := textarea.New()
	ta.Placeholder = "Type your testing request or action number... (Enter to send)"
	ta.CharLimit = 500
	ta.SetWidth(80)
	ta.SetHeight(1)  // Single line input
	ta.Focus()
	
	// Create viewport for chat history
	vp := viewport.New(80, 20)
	
	// Initialize LLM client
	var llmClient llm.Client
	if cfg.AI.APIKey != "" {
		var provider llm.Provider
		switch cfg.AI.Provider {
		case "openai":
			provider = llm.OpenAI
		case "google":
			provider = llm.Google
		case "anthropic":
			provider = llm.Anthropic
		case "openrouter":
			provider = llm.OpenRouter
		case "local":
			provider = llm.Local
		case "mock":
			provider = llm.Mock
		default:
			log.Printf("Unsupported AI provider: %s", cfg.AI.Provider)
		}
		
		if provider != "" {
			options := map[string]interface{}{
				"model": cfg.AI.Model,
			}
			llmClient, _ = llm.NewClient(provider, cfg.AI.APIKey, options)
		}
	}
	
	// Get configured URL
	env := cfg.GetCurrentEnv()
	
	view := &ChatAdventureV2View{
		config:        cfg,
		llmClient:     llmClient,
		configuredURL: env.BaseURL,
		messages:      []ChatMessage{},
		viewport:      vp,
		textarea:      ta,
		width:         80,
		height:        25,
		debounceDelay: 500 * time.Millisecond, // Prevent actions faster than 500ms
		
		// Styles
		userStyle:      lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Bold(true),
		assistantStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("250")),
		systemStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Italic(true),
		errorStyle:     lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true),
		thinkingStyle:  lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Italic(true).PaddingLeft(2),
		borderStyle:    lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("62")),
		titleStyle:     lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")).MarginBottom(1),
		statsStyle:     lipgloss.NewStyle().Foreground(lipgloss.Color("220")),
	}
	
	// Initialize action discovery if LLM is available
	if llmClient != nil {
		view.actionDiscovery = testing.NewActionDiscovery(llmClient, "playwright")
	}
	
	// Initialize spinner
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("241")) // Subtle gray color
	view.spinner = s
	
	// Initialize database
	dbPath := "tod_chat.db"
	if db, err := database.New(dbPath); err == nil {
		view.db = db
	}
	
	// Add welcome message
	view.addMessage("system", "I'm Tod, your AI testing companion. I'll help you explore and test your application.")
	view.addMessage("system", fmt.Sprintf("Target URL: %s", env.BaseURL))
	
	return view
}

// Init initializes the view
func (v *ChatAdventureV2View) Init() tea.Cmd {
	return tea.Batch(
		textarea.Blink,
		v.spinner.Tick,
		v.connectToChrome(),
	)
}

// SetProgram sets the program reference for sending messages
func (v *ChatAdventureV2View) SetProgram(p *tea.Program) {
	v.program = p
}

// Update handles messages
func (v *ChatAdventureV2View) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
		v.viewport.Width = msg.Width - 4
		v.viewport.Height = msg.Height - 10  // Increased from -12 since we removed help text
		v.textarea.SetWidth(msg.Width - 4)
		
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc:
			return v, func() tea.Msg { return ReturnToMenuMsg{} }
		case tea.KeyCtrlC:
			v.cleanup()
			return v, tea.Quit
		case tea.KeyEnter:
			// Submit on Enter (not Ctrl+Enter)
			if !v.isProcessing {
				return v, v.handleUserInput()
			}
		}
		
	case ChromeLaunchedMsg:
		v.isConnected = true
		return v, tea.Batch(v.updateChromeStatus(), v.analyzeInitialPage())
		
	case ChromeErrorMsg:
		v.isConnected = false
		errorStr := msg.Error.Error()
		if strings.Contains(errorStr, "executable file not found") || strings.Contains(errorStr, "chrome") {
			v.addMessage("system", "❌ Chrome browser not found. Please install Chrome or Chromium")
		} else if strings.Contains(errorStr, "port") || strings.Contains(errorStr, "9229") {
			v.addMessage("system", "❌ Port 9229 is in use. Try closing other Chrome instances")
		} else if strings.Contains(errorStr, "timeout") {
			v.addMessage("system", "❌ Chrome took too long to start")
		} else {
			v.addMessage("system", fmt.Sprintf("❌ Chrome connection failed: %v", msg.Error))
		}
		v.addMessage("system", "Type 'connect' to retry")
		
	case ChromeStatusMsg:
		v.currentURL = msg.URL
		v.currentTitle = msg.Title
		
	case HTMLAnalysisMsg:
		v.isProcessing = false
		
		if msg.Error != nil {
			v.addMessage("system", fmt.Sprintf("❌ Error during analysis: %v", msg.Error))
		} else if len(msg.Actions) > 0 {
			v.discoveredActions = msg.Actions
			v.awaitingSelection = true
			v.displayDiscoveredActions(msg.Actions)
			
			// Log discovered actions to database
			if v.db != nil && v.captureID > 0 {
				dbActions := make([]database.DiscoveredAction, len(msg.Actions))
				for i, action := range msg.Actions {
					dbActions[i] = database.DiscoveredAction{
						CaptureID:   v.captureID,
						Description: action.Description,
						Element:     action.Element,
						Selector:    action.Selector,
						Action:      action.Action,
						Priority:    action.Priority,
						IsTested:    action.IsTested,
					}
				}
				v.db.SaveDiscoveredActions(v.captureID, dbActions)
			}
		}
		
		// Update cost info silently
		if msg.PromptTokens > 0 {
			v.lastPromptTokens = msg.PromptTokens
			v.lastPromptCost = msg.EstimatedCost
			v.totalTokens += msg.PromptTokens
			v.totalCost += msg.EstimatedCost
		}
		
	case ActionExecutedMsg:
		v.isProcessing = false
		if msg.Error != nil {
			v.addMessage("system", fmt.Sprintf("❌ Action failed: %v", msg.Error))
		} else {
			v.addMessage("assistant", msg.Result)
			// Re-analyze the page after action to discover new actions
			v.awaitingSelection = false
			return v, tea.Batch(v.updateChromeStatus(), v.analyzeInitialPage())
		}
		
	case ProcessingCompleteMsg:
		v.isProcessing = false
		// Also ensure navigation state is reset if there was an error
		if msg.Error != nil {
			v.isNavigating = false
			if strings.Contains(msg.Error.Error(), "Chrome not connected") {
				v.addMessage("system", "❌ Chrome not connected. Type 'connect' to connect.")
			} else {
				v.addMessage("system", fmt.Sprintf("❌ Error: %v", msg.Error))
			}
		} else if msg.Message != "" {
			v.addMessage("assistant", msg.Message)
			
			// If navigation was successful, trigger page analysis to discover new actions
			// But only if we're not already processing or navigating
			if msg.TriggerAnalysis && !v.isProcessing && !v.isNavigating {
				return v, tea.Batch(v.updateChromeStatus(), v.analyzeInitialPage())
			}
		}
		
	case NavigationStartMsg:
		v.addThinkingMessage(fmt.Sprintf("Navigating to %s...", msg.URL))
		v.addMessage("assistant", fmt.Sprintf("→ Navigating to %s...", msg.URL))
		v.isProcessing = true
		return v, v.performNavigation(msg.URL)
		
	case NavigationCompleteMsg:
		v.isProcessing = false
		v.isNavigating = false
		if msg.Error != nil {
			v.addMessage("system", fmt.Sprintf("❌ Navigation failed: %v", msg.Error))
		} else {
			v.addMessage("assistant", fmt.Sprintf("→ Successfully navigated to: %s", msg.URL))
			// Trigger page analysis after successful navigation to discover actions
			v.awaitingSelection = false
			return v, tea.Batch(v.updateChromeStatus(), v.analyzeInitialPage())
		}
		
	case IncrementalActionsMsg:
		if msg.IsInitial {
			// Handle initial actions
			if len(msg.Actions) > 0 {
				v.discoveredActions = msg.Actions
				v.awaitingSelection = true
				v.displayDiscoveredActions(msg.Actions)
				
				// Log discovered actions to database
				if v.db != nil && v.captureID > 0 {
					dbActions := make([]database.DiscoveredAction, len(msg.Actions))
					for i, action := range msg.Actions {
						dbActions[i] = database.DiscoveredAction{
							CaptureID:   v.captureID,
							Description: action.Description,
							Element:     action.Element,
							Selector:    action.Selector,
							Action:      action.Action,
							Priority:    action.Priority,
							IsTested:    action.IsTested,
						}
					}
					v.db.SaveDiscoveredActions(v.captureID, dbActions)
				}
			}
		} else {
			// Handle incremental actions
			if len(msg.Actions) > 0 {
				v.addMessage("assistant", fmt.Sprintf("🔄 Found %d new actions from dynamic content:", len(msg.Actions)))
				
				// Show only the new actions that were added
				for _, action := range msg.Actions {
					var emoji string
					switch action.Priority {
					case "high":
						emoji = "🟢"
					case "medium":
						emoji = "🟡"
					default:
						emoji = "⚪"
					}
					v.addMessage("assistant", fmt.Sprintf("   %s %s", emoji, action.Description))
				}
				
				// Update the full list for selection
				if len(msg.AllActions) > 0 {
					v.discoveredActions = msg.AllActions
				}
			}
		}
	}
	
	// Always update spinner for animations
	var spinnerCmd tea.Cmd
	v.spinner, spinnerCmd = v.spinner.Update(msg)
	cmds = append(cmds, spinnerCmd)
	
	// Update textarea
	if !v.isProcessing {
		var cmd tea.Cmd
		v.textarea, cmd = v.textarea.Update(msg)
		cmds = append(cmds, cmd)
	}
	
	// Update viewport
	v.viewport.SetContent(v.renderMessages())
	v.viewport.GotoBottom()
	
	return v, tea.Batch(cmds...)
}

// View renders the chat interface
func (v *ChatAdventureV2View) View() string {
	title := v.titleStyle.Render("🧙 Tod Adventure Mode - Your AI Testing Companion")
	
	// Status bar with detailed info
	status := v.renderEnhancedStatusBar()
	
	// Chat viewport with border
	chatView := v.borderStyle.Width(v.width - 2).Height(v.viewport.Height + 2).Render(v.viewport.View())
	
	// Input area
	inputLabel := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("Your input:")
	inputArea := v.textarea.View()
	
	// Processing indicator with animated spinner
	if v.isProcessing {
		spinnerText := fmt.Sprintf("%s Processing...", v.spinner.View())
		inputArea = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Italic(true).Render(spinnerText)
	}
	
	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		status,
		chatView,
		inputLabel,
		inputArea,
	)
}

// connectToChrome establishes Chrome connection in headless mode
func (v *ChatAdventureV2View) connectToChrome() tea.Cmd {
	return func() tea.Msg {
		// Check if already connected
		if v.chromeDPManager != nil {
			return ProcessingCompleteMsg{Message: "Chrome is already connected"}
		}
		
		// Launch in headless mode
		manager, err := browser.GetGlobalChromeDPManager(v.configuredURL, true) // headless=true
		if err != nil {
			// Provide more context about the error
			if strings.Contains(err.Error(), "exec:") {
				return ChromeErrorMsg{Error: fmt.Errorf("Chrome executable not found: %w", err)}
			} else if strings.Contains(err.Error(), "context") {
				return ChromeErrorMsg{Error: fmt.Errorf("Chrome failed to start (timeout or crash): %w", err)}
			}
			return ChromeErrorMsg{Error: err}
		}
		
		v.chromeDPManager = manager
		return ChromeLaunchedMsg{}
	}
}

// analyzeInitialPage analyzes the initial page after Chrome launches with polling for dynamic content
func (v *ChatAdventureV2View) analyzeInitialPage() tea.Cmd {
	return func() tea.Msg {
		if v.chromeDPManager == nil {
			return HTMLAnalysisMsg{Error: fmt.Errorf("Chrome not connected")}
		}
		
		// Add thinking message about what we're analyzing
		v.addThinkingMessage("Analyzing page structure and monitoring for dynamic content...")
		
		// Get page info first
		url, title, _ := v.chromeDPManager.GetPageInfo()
		
		// Log navigation if URL changed
		if url != v.currentURL {
			log.Printf("[NAVIGATION] URL: %s", url)
			log.Printf("[NAVIGATION] Title: %s", title)
		}
		
		v.currentURL = url
		v.currentTitle = title
		
		// Start polling for changes with our timing settings
		changes := v.chromeDPManager.PollForChanges(
			3*time.Second,     // Total duration: 3 seconds
			200*time.Millisecond, // Interval: 200ms
			30*time.Millisecond,  // Initial delay: 30ms (20-40ms range)
		)
		
		var initialActions []testing.DiscoveredAction
		var charCount int
		var hasProcessedInitial bool
		
		// Process changes as they come in
		for change := range changes {
			charCount = len(change.HTML)
			
			// Save page capture to database (for initial snapshot)
			if v.db != nil && change.IsInitial {
				capture := database.PageCapture{
					URL:        url,
					Title:      title,
					HTMLLength: charCount,
					ChromePort: 9229,
				}
				captureID, err := v.db.SavePageCapture(&capture)
				if err == nil {
					v.captureID = captureID
				}
			}
			
			// If no LLM client, skip action discovery
			if v.actionDiscovery == nil {
				if change.IsInitial {
					return HTMLAnalysisMsg{CharCount: charCount}
				}
				continue
			}
			
			ctx := context.Background()
			
			if change.IsInitial {
				// Process initial snapshot
				log.Printf("Processing initial HTML snapshot (%d chars)", len(change.HTML))
				
				actions, _, _, err := v.actionDiscovery.DiscoverActionsFromHTMLWithContext(
					ctx,
					change.HTML,
					[]string{}, // existing tests
					"Analyze for testing opportunities",
				)
				
				if err != nil {
					log.Printf("Initial action discovery failed: %v", err)
					return HTMLAnalysisMsg{
						CharCount: charCount,
						Error:     err,
					}
				}
				
				initialActions = actions
				hasProcessedInitial = true
				
				// Send initial results immediately
				// This will be sent as soon as initial analysis is done
				go func() {
					v.program.Send(IncrementalActionsMsg{
						Actions:   initialActions,
						IsInitial: true,
					})
				}()
				
			} else if hasProcessedInitial {
				// Process incremental changes
				log.Printf("Processing incremental change (%d chars new content)", len(change.NewContent))
				
				if change.NewContent != "" {
					newActions, err := v.actionDiscovery.DiscoverIncrementalActions(
						ctx,
						change.NewContent,
						initialActions,
						[]string{}, // existing tests
					)
					
					if err != nil {
						log.Printf("Incremental action discovery failed: %v", err)
						continue
					}
					
					if len(newActions) > 0 {
						log.Printf("Found %d new actions from dynamic content", len(newActions))
						
						// Merge with existing actions
						mergedActions := v.actionDiscovery.MergeActions(initialActions, newActions)
						initialActions = mergedActions
						
						// Send incremental update
						go func() {
							v.program.Send(IncrementalActionsMsg{
								Actions:    newActions,
								IsInitial:  false,
								AllActions: mergedActions,
							})
						}()
					}
				}
			}
		}
		
		// Return final analysis result
		promptTokens := charCount / 4 // Rough token estimate
		costPerToken := 0.00003 // Example rate for GPT-4
		estimatedCost := float64(promptTokens) * costPerToken
		
		return HTMLAnalysisMsg{
			CharCount:     charCount,
			Actions:       initialActions,
			PromptTokens:  promptTokens,
			EstimatedCost: estimatedCost,
		}
	}
}

// buildAnalysisPrompt builds the prompt for page analysis
func (v *ChatAdventureV2View) buildAnalysisPrompt(html string) string {
	// Extract interactive elements (simplified)
	elements := v.extractInteractiveElements(html)
	
	// Truncate HTML if too long
	simplifiedHTML := html
	if len(html) > 5000 {
		simplifiedHTML = html[:5000] + "...[truncated]"
	}
	
	prompt := fmt.Sprintf(`You are a test automation expert analyzing a web page to identify user actions that should be tested.

INTERACTIVE ELEMENTS FOUND:
%s

EXISTING TEST COVERAGE:
No existing tests found.

SIMPLIFIED HTML STRUCTURE:
%s

TASK:
Analyze the page and identify important user actions that should be tested.
Focus on:
1. Form submissions and validations
2. Navigation between pages/sections
3. Interactive buttons and their expected behaviors
4. Data input and validation scenarios
5. Error handling and edge cases

For each action, provide:
- Description: What the user action does
- Element: The HTML element involved
- Selector: CSS selector to target the element
- Action: The type of interaction (click, type, select, etc.)
- Test Scenario: Brief description of what to test
- Priority: high/medium/low based on importance

Format your response as a numbered list with each action on a new line.
Example:
1. LOGIN_FORM_SUBMIT | button[data-testid='login'] | click | Submit login form with valid credentials | high
2. SIGNUP_NAVIGATION | a[href='/signup'] | click | Navigate to signup page | medium

Identify the top 10 most important actions to test:`, elements, simplifiedHTML)
	
	return prompt
}

// extractInteractiveElements extracts interactive elements from HTML
func (v *ChatAdventureV2View) extractInteractiveElements(html string) string {
	// Simple extraction of interactive elements
	var elements []string
	
	// Look for common interactive elements
	tags := []string{"button", "input", "select", "textarea", "a"}
	for _, tag := range tags {
		if strings.Contains(html, "<"+tag) {
			elements = append(elements, fmt.Sprintf("- %s elements found", tag))
		}
	}
	
	if len(elements) == 0 {
		return "No interactive elements detected"
	}
	
	return strings.Join(elements, "\n")
}

// displayDiscoveredActions displays discovered actions as numbered options
func (v *ChatAdventureV2View) displayDiscoveredActions(actions []testing.DiscoveredAction) {
	v.addMessage("assistant", "I've discovered the following actions:")
	
	for _, action := range actions {
		// Use emoji to indicate priority
		var emoji string
		switch action.Priority {
		case "high":
			emoji = "🟢" // Green for high priority
		case "medium":
			emoji = "🟡" // Yellow for medium priority
		default:
			emoji = "⚪" // White for low priority
		}
		
		// Just show emoji and action description
		msg := fmt.Sprintf("%s %s", emoji, action.Description)
		v.addMessage("assistant", msg)
	}
	
	v.addMessage("assistant", "\nType or describe the action you want to perform.")
}

// handleUserInput processes user input
func (v *ChatAdventureV2View) handleUserInput() tea.Cmd {
	input := strings.TrimSpace(v.textarea.Value())
	if input == "" {
		return nil
	}
	
	// Debounce - prevent too rapid actions
	if time.Since(v.lastActionTime) < v.debounceDelay {
		return nil
	}
	v.lastActionTime = time.Now()
	
	// Add user message
	v.addMessage("user", input)
	v.textarea.Reset()
	v.isProcessing = true
	
	// Log user input
	if v.db != nil {
		// Could track user inputs separately if needed
	}
	
	// Check if user is selecting an action
	if v.awaitingSelection && len(v.discoveredActions) > 0 {
		// Try to match user input to an action
		selectedAction := v.findMatchingAction(input)
		if selectedAction != nil {
			// Pass the user's input as context for better code generation
			selectedAction.UserInput = input
			return v.executeAction(*selectedAction)
		}
	}
	
	// Otherwise process as command or natural language
	return v.processUserInput(input)
}

// findMatchingAction finds an action that matches the user's input
func (v *ChatAdventureV2View) findMatchingAction(userInput string) *testing.DiscoveredAction {
	inputLower := strings.ToLower(userInput)
	
	// First check if it's a number
	if num, err := strconv.Atoi(userInput); err == nil && num > 0 && num <= len(v.discoveredActions) {
		return &v.discoveredActions[num-1]
	}
	
	// Then try to match based on keywords
	for i, action := range v.discoveredActions {
		actionLower := strings.ToLower(action.Description)
		
		// Check for significant overlap
		if strings.Contains(actionLower, inputLower) || strings.Contains(inputLower, actionLower) {
			return &v.discoveredActions[i]
		}
		
		// Check for key words match
		inputWords := strings.Fields(inputLower)
		actionWords := strings.Fields(actionLower)
		
		matches := 0
		for _, iw := range inputWords {
			if len(iw) < 3 { // Skip small words
				continue
			}
			for _, aw := range actionWords {
				if strings.Contains(aw, iw) || strings.Contains(iw, aw) {
					matches++
					break
				}
			}
		}
		
		// If we have significant word matches, consider it a match
		if matches >= 2 || (matches == 1 && len(inputWords) == 1) {
			return &v.discoveredActions[i]
		}
	}
	
	return nil
}

// executeAction executes a selected action with retry logic
func (v *ChatAdventureV2View) executeAction(action testing.DiscoveredAction) tea.Cmd {
	// Add thinking message OUTSIDE the tea.Cmd for thread safety
	v.addThinkingMessage(fmt.Sprintf("Determining the best way to execute: %s", action.Description))
	
	return func() tea.Msg {
		if v.chromeDPManager == nil {
			return ActionExecutedMsg{Error: fmt.Errorf("Chrome not connected")}
		}
		
		// Capture state before action
		v.beforeHTML, _ = v.chromeDPManager.GetPageHTML()
		v.beforeURL, _, _ = v.chromeDPManager.GetPageInfo()
		
		// Always generate fresh JavaScript based on user input
		if v.actionDiscovery != nil {
			log.Printf("Generating JavaScript code for action: %s", action.Description)
			
			// Generate executable code for this specific action with user context
			ctx := context.Background()
			if ctx == nil {
				ctx = context.TODO()
			}
			updatedAction, err := v.actionDiscovery.GenerateActionCode(ctx, action, v.beforeHTML)
			if err != nil {
				log.Printf("Failed to generate action code: %v", err)
				// Fall back to simple approach
			} else {
				action = updatedAction
				log.Printf("Generated JavaScript for user request '%s': %s", action.UserInput, action.JavaScript)
			}
		}
		
		var err error
		
		// Try executing the JavaScript if available
		if action.JavaScript != "" {
			log.Printf("[ACTION] Executing: %s", action.Description)
			log.Printf("[ACTION] JavaScript: %s", action.JavaScript)
			var success bool
			err = v.chromeDPManager.ExecuteScript(action.JavaScript, &success)
			if err == nil && success {
				log.Printf("[ACTION] JavaScript execution successful")
			} else {
				log.Printf("[ACTION] JavaScript execution failed: %v, success=%v", err, success)
				// Fall back to selector-based approach
				err = v.tryFallbackExecution(action)
				if err == nil {
					log.Printf("Fallback execution successful")
				}
			}
		} else {
			// No JavaScript available, use fallback
			err = v.tryFallbackExecution(action)
			if err == nil {
				log.Printf("Fallback execution successful")
			}
		}
		
		if err != nil {
			return ActionExecutedMsg{Error: fmt.Errorf("execution failed: %w", err)}
		}
		
		// Wait for page to update with smart polling
		afterHTML, afterURL, afterTitle := v.waitForPageUpdate()
		
		
		// Generate change description
		changeDesc := v.describeChanges(v.beforeURL, afterURL, v.beforeHTML, afterHTML, afterTitle)
		
		return ActionExecutedMsg{Result: changeDesc}
	}
}

// waitForPageUpdate intelligently waits for page changes after an action
func (v *ChatAdventureV2View) waitForPageUpdate() (afterHTML, afterURL, afterTitle string) {
	const maxWaitTime = 2 * time.Second
	const pollInterval = 100 * time.Millisecond
	
	startTime := time.Now()
	initialHTML := v.beforeHTML
	initialURL := v.beforeURL
	
	log.Printf("[NAVIGATION] Starting smart wait - Initial URL: %s", initialURL)
	
	for time.Since(startTime) < maxWaitTime {
		// Get current page state
		currentHTML, _ := v.chromeDPManager.GetPageHTML()
		currentURL, currentTitle, _ := v.chromeDPManager.GetPageInfo()
		
		// Check if URL changed (primary indicator of navigation)
		if currentURL != initialURL {
			log.Printf("[NAVIGATION] URL changed from %s to %s after %v", 
				initialURL, currentURL, time.Since(startTime))
			return currentHTML, currentURL, currentTitle
		}
		
		// Check if significant HTML changes occurred (secondary indicator)
		if len(currentHTML) > 0 && len(initialHTML) > 0 {
			// Simple heuristic: if HTML length changed significantly, content likely changed
			lenDiff := abs(len(currentHTML) - len(initialHTML))
			if lenDiff > len(initialHTML)/10 { // More than 10% change
				log.Printf("[NAVIGATION] Significant HTML change detected (%d chars) after %v", 
					lenDiff, time.Since(startTime))
				return currentHTML, currentURL, currentTitle
			}
		}
		
		time.Sleep(pollInterval)
	}
	
	// Timeout reached, get final state
	finalHTML, _ := v.chromeDPManager.GetPageHTML()
	finalURL, finalTitle, _ := v.chromeDPManager.GetPageInfo()
	
	log.Printf("[NAVIGATION] Smart wait completed after %v - Final URL: %s", 
		time.Since(startTime), finalURL)
	
	return finalHTML, finalURL, finalTitle
}

// abs returns the absolute value of an integer
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// describeChanges generates a description of what changed after an action
func (v *ChatAdventureV2View) describeChanges(beforeURL, afterURL, beforeHTML, afterHTML, afterTitle string) string {
	// Debug logging to help diagnose navigation issues
	log.Printf("[CHANGE_DETECTION] Before URL: %s", beforeURL)
	log.Printf("[CHANGE_DETECTION] After URL: %s", afterURL)
	log.Printf("[CHANGE_DETECTION] After Title: %s", afterTitle)
	log.Printf("[CHANGE_DETECTION] HTML length change: %d -> %d (diff: %d)", 
		len(beforeHTML), len(afterHTML), len(afterHTML)-len(beforeHTML))
	
	// Check if URL changed (most common and clear indicator)
	if beforeURL != afterURL {
		// Log navigation for debugging
		log.Printf("[NAVIGATION] URL: %s -> %s", beforeURL, afterURL)
		log.Printf("[NAVIGATION] Title: %s", afterTitle)
		
		// Clean up the URL for display
		displayURL := afterURL
		if strings.HasPrefix(afterURL, "http://localhost:") {
			// Simplify localhost URLs
			parts := strings.Split(afterURL, "/")
			if len(parts) > 3 {
				path := strings.Join(parts[3:], "/")
				if path != "" {
					displayURL = "/" + path
				}
			}
		}
		
		if afterTitle != "" && afterTitle != "GroupUp - Organization Showcase" {
			return fmt.Sprintf("→ Navigated to %s (%s)", displayURL, afterTitle)
		}
		return fmt.Sprintf("→ Navigated to %s", displayURL)
	}
	
	// Look for specific state changes in the HTML
	beforeLower := strings.ToLower(beforeHTML)
	afterLower := strings.ToLower(afterHTML)
	
	// Check for login/logout state changes
	if strings.Contains(afterLower, "signed in") && !strings.Contains(beforeLower, "signed in") {
		return "→ Successfully signed in"
	}
	if strings.Contains(afterLower, "logged in") && !strings.Contains(beforeLower, "logged in") {
		return "→ Successfully logged in"
	}
	if strings.Contains(afterLower, "welcome") && !strings.Contains(beforeLower, "welcome") {
		return "→ Login successful - welcome screen displayed"
	}
	
	// Check for form submissions
	if strings.Contains(afterLower, "thank you") && !strings.Contains(beforeLower, "thank you") {
		return "→ Form submitted successfully"
	}
	
	// Check for modal/dialog changes
	if strings.Contains(afterLower, "modal") && !strings.Contains(beforeLower, "modal") {
		return "→ Modal dialog opened"
	}
	if !strings.Contains(afterLower, "modal") && strings.Contains(beforeLower, "modal") {
		return "→ Modal dialog closed"
	}
	
	// Check for error messages
	if strings.Contains(afterLower, "error") && !strings.Contains(beforeLower, "error") {
		return "→ Error message displayed"
	}
	
	// Check for success messages  
	if strings.Contains(afterLower, "success") && !strings.Contains(beforeLower, "success") {
		return "→ Success message displayed"
	}
	
	// Check if content size changed significantly
	sizeDiff := len(afterHTML) - len(beforeHTML)
	if sizeDiff > 5000 {
		return "→ New content loaded on page"
	} else if sizeDiff < -5000 {
		return "→ Content removed from page"
	}
	
	// Check for loading states
	if strings.Contains(afterLower, "loading") && !strings.Contains(beforeLower, "loading") {
		return "→ Loading content..."
	}
	if !strings.Contains(afterLower, "loading") && strings.Contains(beforeLower, "loading") {
		return "→ Content finished loading"
	}
	
	// Default message if no significant change detected
	log.Printf("[CHANGE_DETECTION] No significant changes detected - URLs identical, no major HTML/content changes found")
	return "→ Action completed"
}

// tryFallbackExecution tries selector-based execution with retries
func (v *ChatAdventureV2View) tryFallbackExecution(action testing.DiscoveredAction) error {
	maxRetries := 3
	var err error
	
	log.Printf("[FALLBACK] Trying alternative selector-based approach for: %s", action.Description)
	
	// Try different selector strategies
	selectors := v.generateSelectorVariations(action.Selector, action.Description)
	
	log.Printf("[FALLBACK] Trying selector-based execution for: %s", action.Description)
	log.Printf("[FALLBACK] Generated %d selector variations", len(selectors))
	
	for attempt := 1; attempt <= maxRetries; attempt++ {
		for i, selector := range selectors {
			log.Printf("[FALLBACK] Attempt %d/%d, Selector %d/%d: %s", attempt, maxRetries, i+1, len(selectors), selector)
			
			// Wait for element to be ready
			waitErr := v.chromeDPManager.WaitForElement(selector)
			if waitErr != nil {
				log.Printf("[FALLBACK] Element not found: %v", waitErr)
				continue
			}
			
			// Try to click
			err = v.chromeDPManager.Click(selector)
			if err == nil {
				log.Printf("[FALLBACK] SUCCESS with selector: %s", selector)
				return nil
			}
			log.Printf("[FALLBACK] Click failed: %v", err)
		}
		
		// Wait before retry
		if attempt < maxRetries {
			log.Printf("[FALLBACK] Waiting %d seconds before retry...", attempt)
			time.Sleep(time.Duration(attempt) * time.Second)
		}
	}
	
	// If all retries failed, try JavaScript click as last resort
	if err != nil {
		log.Printf("[FALLBACK] All selector attempts failed, trying JavaScript click as final option...")
		log.Printf("[FALLBACK] All selector attempts failed, trying JavaScript click...")
		err = v.tryJavaScriptClick(action)
		if err == nil {
			log.Printf("[FALLBACK] JavaScript click successful")
		} else {
			log.Printf("[FALLBACK] JavaScript click failed: %v", err)
		}
	}
	
	return err
}

// generateSelectorVariations creates different selector variations to try
func (v *ChatAdventureV2View) generateSelectorVariations(originalSelector, description string) []string {
	selectors := []string{originalSelector}
	
	// Clean up the selector
	cleaned := strings.TrimSpace(originalSelector)
	
	// If it's an a:contains selector, try different approaches
	if strings.HasPrefix(cleaned, "a:contains") {
		// Extract the text
		start := strings.Index(cleaned, "'")
		end := strings.LastIndex(cleaned, "'")
		if start != -1 && end != -1 && start < end {
			text := cleaned[start+1:end]
			
			// Try different selector strategies
			selectors = append(selectors,
				fmt.Sprintf(`//a[contains(text(), "%s")]`, text), // XPath
				fmt.Sprintf(`a[text*="%s"]`, text),              // CSS partial text
				fmt.Sprintf(`[aria-label*="%s"]`, text),         // ARIA label
				fmt.Sprintf(`button:has-text("%s")`, text),      // Button with text
				fmt.Sprintf(`*[role="button"]:has-text("%s")`, text), // Role button
			)
		}
	}
	
	// Add variations based on description
	descLower := strings.ToLower(description)
	if strings.Contains(descLower, "sign in") || strings.Contains(descLower, "login") {
		selectors = append(selectors,
			`a[href*="signin"]`,
			`a[href*="login"]`,
			`button[text*="Sign"]`,
			`[data-testid*="signin"]`,
			`[data-testid*="login"]`,
		)
	} else if strings.Contains(descLower, "start") || strings.Contains(descLower, "get started") {
		selectors = append(selectors,
			`a[href*="start"]`,
			`button[text*="Start"]`,
			`[data-testid*="start"]`,
			`a[href*="signup"]`,
		)
	}
	
	return selectors
}

// tryJavaScriptClick attempts to click using JavaScript
func (v *ChatAdventureV2View) tryJavaScriptClick(action testing.DiscoveredAction) error {
	// Extract text from selector if possible
	text := ""
	if strings.Contains(action.Selector, "'") {
		start := strings.Index(action.Selector, "'")
		end := strings.LastIndex(action.Selector, "'")
		if start != -1 && end != -1 && start < end {
			text = action.Selector[start+1:end]
		}
	}
	
	if text == "" {
		text = action.Description
	}
	
	// JavaScript to find and click element by text
	script := fmt.Sprintf(`
		(() => {
			const elements = document.querySelectorAll('a, button, [role="button"]');
			for (let el of elements) {
				if (el.textContent.includes('%s')) {
					el.click();
					return true;
				}
			}
			return false;
		})()
	`, text)
	
	log.Printf("[JS_CLICK] Searching for text: %s", text)
	log.Printf("[JS_CLICK] JavaScript: %s", strings.ReplaceAll(script, "\n", " "))
	
	var result bool
	err := v.chromeDPManager.ExecuteScript(script, &result)
	if err != nil {
		log.Printf("[JS_CLICK] Execution error: %v", err)
		return err
	}
	
	if !result {
		log.Printf("[JS_CLICK] Element not found with text: %s", text)
		return fmt.Errorf("element not found with text: %s", text)
	}
	
	log.Printf("[JS_CLICK] Successfully clicked element with text: %s", text)
	return nil
}

// processUserInput handles natural language input
func (v *ChatAdventureV2View) processUserInput(input string) tea.Cmd {
	lowered := strings.ToLower(input)
	
	switch {
	case lowered == "go" || lowered == "start":
		// Navigate to configured URL
		if v.chromeDPManager == nil {
			v.addMessage("system", "❌ Chrome not connected. Type 'connect' first.")
			return nil
		}
		v.addThinkingMessage("Navigating to configured homepage...")
		return v.startNavigation(v.configuredURL)
		
	case lowered == "connect":
		return v.connectToChrome()
		
	case lowered == "status":
		return v.showStatus()
		
	case strings.HasPrefix(lowered, "configure"):
		return v.restartConfiguration()
		
	case strings.HasPrefix(lowered, "analyze"):
		v.awaitingSelection = false
		return v.analyzeInitialPage()
		
	case strings.HasPrefix(lowered, "navigate "):
		url := strings.TrimSpace(strings.TrimPrefix(input, "navigate "))
		v.addThinkingMessage("Understanding navigation request...")
		return v.startNavigation(url)
		
	case strings.HasPrefix(lowered, "help"):
		return v.showHelp()
		
	default:
		// Try to interpret the command using LLM before falling back to analysis
		return v.interpretAndExecuteCommand(input)
	}
}

// interpretAndExecuteCommand interprets natural language commands and executes them
func (v *ChatAdventureV2View) interpretAndExecuteCommand(input string) tea.Cmd {
	// Add thinking message OUTSIDE the tea.Cmd function for thread safety
	v.addThinkingMessage(fmt.Sprintf("Interpreting your request: '%s'...", input))
	
	return func() tea.Msg {
		// Prevent processing if already navigating to avoid loops
		if v.isNavigating {
			return ProcessingCompleteMsg{Message: "Navigation in progress, please wait..."}
		}
		
		// If no LLM client available, fall back to analysis
		if v.llmClient == nil {
			return ProcessingCompleteMsg{Message: "AI not configured. Use specific commands like 'go', 'connect', 'status'."}
		}

		// Interpret the command using the LLM
		ctx := context.Background()
		interpretation, err := v.llmClient.InterpretCommand(ctx, input, []types.CodeAction{})
		if err != nil {
			// Fall back to context analysis if interpretation fails  
			// But avoid analysis if navigating to prevent loops
			if !v.isNavigating {
				return ProcessingCompleteMsg{Message: "Let me analyze the page for available actions..."}
			} else {
				return ProcessingCompleteMsg{Message: "Navigation in progress, please wait..."}
			}
		}

		// Execute based on interpretation
		switch interpretation.CommandType {
		case "navigation":
			target := interpretation.Parameters["target"]
			if target == "homepage" || interpretation.Parameters["page"] == "/" {
				// Return navigation start message for homepage
				return NavigationStartMsg{URL: v.configuredURL}
			} else if page, exists := interpretation.Parameters["page"]; exists && page != "" {
				// First try to find a matching navigation element instead of constructing URL
				return v.findAndNavigateToElement(page)
			} else if target != "" {
				// Try to find matching element based on target
				return v.findAndNavigateToElement(target)
			} else {
				return ProcessingCompleteMsg{Message: "I understand you want to navigate, but I'm not sure where. Try 'go to homepage' or specify a URL."}
			}

		case "authentication":
			return ProcessingCompleteMsg{Message: "Let me look for sign-in options on the current page..."}

		case "interaction":
			return ProcessingCompleteMsg{Message: "Let me analyze the page to find what you can interact with..."}

		case "form_input":
			return ProcessingCompleteMsg{Message: "Let me find form fields you can fill out..."}

		default:
			// If interpretation is unclear, provide helpful guidance
			if interpretation.Confidence < 0.5 {
				return ProcessingCompleteMsg{Message: fmt.Sprintf("I'm not sure what you want to do. Try: %s", strings.Join(interpretation.Suggestions, ", "))}
			} else {
				// Fall back to contextual analysis for unclear commands
				return ProcessingCompleteMsg{Message: "Let me analyze the page to help you with that..."}
			}
		}
	}
}

// analyzeWithContextCmd is a helper to return analysis command
func (v *ChatAdventureV2View) analyzeWithContextCmd(context string) tea.Cmd {
	// Prevent analysis if already navigating to avoid loops
	if v.isNavigating {
		return func() tea.Msg {
			return ProcessingCompleteMsg{Message: "Navigation in progress, analysis will be triggered after completion"}
		}
	}
	
	// Set flag to indicate we're not awaiting selection, just providing info
	v.awaitingSelection = false
	return v.analyzeWithContext(context)
}

// analyzeWithContext analyzes the page with user-provided context
func (v *ChatAdventureV2View) analyzeWithContext(context string) tea.Cmd {
	return func() tea.Msg {
		if v.chromeDPManager == nil {
			return ProcessingCompleteMsg{Error: fmt.Errorf("Chrome not connected")}
		}
		
		if v.actionDiscovery == nil {
			return ProcessingCompleteMsg{Message: "AI not configured. Use 'analyze' to scan the page."}
		}
		
		// Get current page HTML
		html, err := v.chromeDPManager.GetPageHTML()
		if err != nil {
			return ProcessingCompleteMsg{Error: err}
		}
		
		// Discover actions with user context
		ctx := context
		actions, _, _, err := v.actionDiscovery.DiscoverActionsFromHTMLWithContext(
			nil,
			html,
			[]string{},
			ctx,
		)
		
		if err != nil {
			return HTMLAnalysisMsg{Error: err}
		}
		
		// Return as analysis result
		return HTMLAnalysisMsg{
			CharCount: len(html),
			Actions:   actions,
		}
	}
}

// startNavigation initiates navigation and returns NavigationStartMsg
func (v *ChatAdventureV2View) startNavigation(url string) tea.Cmd {
	return func() tea.Msg {
		// Check preconditions
		if v.isNavigating {
			log.Printf("[NAV] Navigation already in progress, blocking duplicate request")
			return NavigationCompleteMsg{URL: url, Success: false, Error: fmt.Errorf("navigation already in progress")}
		}
		
		if v.chromeDPManager == nil {
			log.Printf("[NAV] Chrome not connected")
			return NavigationCompleteMsg{URL: url, Success: false, Error: fmt.Errorf("Chrome not connected")}
		}
		
		// Just return the start message, actual navigation will be handled by performNavigation
		return NavigationStartMsg{URL: url}
	}
}

// performNavigation performs the actual navigation
func (v *ChatAdventureV2View) performNavigation(url string) tea.Cmd {
	return func() tea.Msg {
		// Set navigation state and ensure it gets reset
		v.isNavigating = true
		defer func() {
			v.isNavigating = false
			log.Printf("[NAV] Navigation state reset")
			if r := recover(); r != nil {
				log.Printf("[NAV] Navigation panic recovered: %v", r)
			}
		}()
		
		log.Printf("[NAV] Starting navigation to: %s", url)
		
		// Ensure URL has protocol
		if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
			url = "https://" + url
			log.Printf("[NAV] Added protocol, URL now: %s", url)
		}
		
		// Perform navigation directly
		err := v.chromeDPManager.Navigate(url)
		if err != nil {
			log.Printf("[NAV] Navigation failed: %v", err)
			return NavigationCompleteMsg{URL: url, Success: false, Error: fmt.Errorf("navigation failed: %w", err)}
		}
		
		// Get new page info
		newURL, _, _ := v.chromeDPManager.GetPageInfo()
		log.Printf("[NAV] Navigation successful, new URL: %s", newURL)
		
		// Log navigation to database
		if v.db != nil {
			// Could track navigation here
		}
		
		return NavigationCompleteMsg{URL: newURL, Success: true, Error: nil}
	}
}

// showHelp displays help
func (v *ChatAdventureV2View) showHelp() tea.Cmd {
	return func() tea.Msg {
		help := `Available Commands:
• go / start - Navigate to configured URL (%s)
• connect - Connect to Chrome browser
• status - Show current Chrome and session status
• analyze - Analyze current page for testable actions
• navigate <url> - Go to a specific URL
• configure - Restart configuration (tod init)
• help - Show this help message
• [number] - Execute a discovered action by number

Natural Language:
Describe what you want to test and I'll help!

Current Session Stats:
• Total Tokens: %d
• Total Cost: $%.4f
• Current Page: %s`
		
		return ProcessingCompleteMsg{
			Message: fmt.Sprintf(help, v.configuredURL, v.totalTokens, v.totalCost, v.currentTitle),
		}
	}
}

// showStatus displays current Chrome and session status
func (v *ChatAdventureV2View) showStatus() tea.Cmd {
	return func() tea.Msg {
		status := "📊 Current Status:\n"
		
		// Chrome connection status
		if v.isConnected && v.chromeDPManager != nil {
			status += "✅ Chrome: Connected (headless mode)\n"
			if v.currentURL != "" {
				status += fmt.Sprintf("📍 Current URL: %s\n", v.currentURL)
			}
			if v.currentTitle != "" {
				status += fmt.Sprintf("📄 Page Title: %s\n", v.currentTitle)
			}
		} else {
			status += "❌ Chrome: Not connected\n"
			status += "💡 Type 'connect' to connect to Chrome\n"
		}
		
		// Configuration
		status += fmt.Sprintf("\n⚙️ Configuration:\n")
		status += fmt.Sprintf("🎯 Target URL: %s\n", v.configuredURL)
		
		// Session stats
		if v.totalTokens > 0 {
			status += fmt.Sprintf("\n📈 Session Stats:\n")
			status += fmt.Sprintf("Tokens Used: %d\n", v.totalTokens)
			status += fmt.Sprintf("Estimated Cost: $%.4f", v.totalCost)
		}
		
		return ProcessingCompleteMsg{Message: status}
	}
}

// restartConfiguration triggers the tod init flow
func (v *ChatAdventureV2View) restartConfiguration() tea.Cmd {
	return func() tea.Msg {
		v.addMessage("system", "Restarting configuration...")
		// Return a message to trigger the init flow
		return RestartConfigMsg{}
	}
}

// updateChromeStatus updates Chrome status
func (v *ChatAdventureV2View) updateChromeStatus() tea.Cmd {
	if v.chromeDPManager == nil {
		return nil
	}
	
	return func() tea.Msg {
		url, title, err := v.chromeDPManager.GetPageInfo()
		if err != nil {
			return ChromeStatusMsg{}
		}
		return ChromeStatusMsg{URL: url, Title: title}
	}
}

// renderEnhancedStatusBar renders detailed status bar
func (v *ChatAdventureV2View) renderEnhancedStatusBar() string {
	var parts []string
	
	// Connection status
	if v.isConnected {
		parts = append(parts, "✓ Connected")
	} else {
		parts = append(parts, "⚠ Not connected")
	}
	
	// Current page
	if v.currentTitle != "" {
		parts = append(parts, fmt.Sprintf("📄 %s", v.currentTitle))
	}
	
	// Token stats
	if v.totalTokens > 0 {
		parts = append(parts, v.statsStyle.Render(
			fmt.Sprintf("Tokens: %d ($%.4f)", v.totalTokens, v.totalCost),
		))
	}
	
	// Capture ID
	if v.captureID > 0 {
		parts = append(parts, fmt.Sprintf("Capture: #%d", v.captureID))
	}
	
	status := strings.Join(parts, " | ")
	
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Background(lipgloss.Color("235")).
		Width(v.width).
		Padding(0, 1).
		Render(status)
}

// addMessage adds a message to the chat
func (v *ChatAdventureV2View) addMessage(role, content string) {
	v.messages = append(v.messages, ChatMessage{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
	})
	// Update viewport content and scroll to bottom
	v.viewport.SetContent(v.renderMessages())
	v.viewport.GotoBottom()
}

// addThinkingMessage adds a thinking message to the chat
func (v *ChatAdventureV2View) addThinkingMessage(content string) {
	v.messages = append(v.messages, ChatMessage{
		Role:       "system",
		Content:    content,
		Timestamp:  time.Now(),
		IsThinking: true,
	})
	// Update viewport content and scroll to bottom
	v.viewport.SetContent(v.renderMessages())
	v.viewport.GotoBottom()
}

// renderMessages renders all chat messages
func (v *ChatAdventureV2View) renderMessages() string {
	var lines []string
	
	for _, msg := range v.messages {
		var style lipgloss.Style
		var content string
		
		// Handle thinking messages specially
		if msg.IsThinking {
			style = v.thinkingStyle
			content = msg.Content
		} else {
			switch msg.Role {
			case "user":
				style = v.userStyle
			case "assistant":
				style = v.assistantStyle
			case "system":
				style = v.systemStyle
			}
			content = msg.Content
		}
		
		if msg.IsError {
			style = v.errorStyle
		}
		
		// Format message without prefix
		lines = append(lines, style.Render(content))
	}
	
	return strings.Join(lines, "\n")
}

// findAndNavigateToElement looks for a matching navigation element before falling back to URL construction
func (v *ChatAdventureV2View) findAndNavigateToElement(target string) tea.Msg {
	// First, get current page elements
	if v.chromeDPManager == nil {
		return ProcessingCompleteMsg{Error: fmt.Errorf("Chrome not connected")}
	}
	
	// Extract current page interactive elements
	elements, err := v.chromeDPManager.ExtractInteractiveElements()
	if err != nil {
		// Fall back to URL construction if we can't extract elements
		return NavigationStartMsg{URL: target}
	}
	
	// Look for navigation links that match the target
	var bestMatch *browser.InteractiveElement
	var bestScore float64
	
	targetLower := strings.ToLower(target)
	
	for i, elem := range elements {
		if elem.IsNavigation && elem.Text != "" {
			// Score based on text content match
			elemTextLower := strings.ToLower(elem.Text)
			
			var score float64
			
			// Exact match gets highest score
			if elemTextLower == targetLower {
				score = 1.0
			} else if strings.Contains(elemTextLower, targetLower) || strings.Contains(targetLower, elemTextLower) {
				// Partial match
				score = 0.8
			} else {
				// Word-based matching
				targetWords := strings.Fields(targetLower)
				elemWords := strings.Fields(elemTextLower)
				
				matches := 0
				for _, tw := range targetWords {
					for _, ew := range elemWords {
						if strings.Contains(ew, tw) || strings.Contains(tw, ew) {
							matches++
							break
						}
					}
				}
				
				if matches > 0 {
					score = float64(matches) / float64(len(targetWords)) * 0.6
				}
			}
			
			if score > bestScore && score > 0.3 {
				bestScore = score
				bestMatch = &elements[i]
			}
		}
	}
	
	// If we found a good match, create a navigation action
	if bestMatch != nil {
		// If we have a full URL, we can navigate directly
		if bestMatch.FullUrl != "" {
			return NavigationStartMsg{URL: bestMatch.FullUrl}
		}
		
		// Otherwise, show what we found and suggest clicking it
		return ProcessingCompleteMsg{
			Message: fmt.Sprintf("Found navigation link: '%s' (match score: %.1f). Use 'click %s' to navigate there.", 
				bestMatch.Text, bestScore, bestMatch.Text),
		}
	}
	
	// If no match found and target looks like a URL or path, try it directly
	if strings.HasPrefix(target, "http") || strings.HasPrefix(target, "/") || strings.Contains(target, ".") {
		return NavigationStartMsg{URL: target}
	}
	
	// Show available navigation options
	var navOptions []string
	for _, elem := range elements {
		if elem.IsNavigation && elem.Text != "" && len(navOptions) < 5 {
			navOptions = append(navOptions, fmt.Sprintf("\"%s\"", elem.Text))
		}
	}
	
	if len(navOptions) > 0 {
		optionsStr := strings.Join(navOptions, ", ")
		return ProcessingCompleteMsg{
			Message: fmt.Sprintf("No navigation match found for '%s'. Available options: %s", 
				target, optionsStr),
		}
	}
	
	return ProcessingCompleteMsg{Message: fmt.Sprintf("No navigation elements found matching '%s'", target)}
}

// cleanup cleans up resources
func (v *ChatAdventureV2View) cleanup() {
	if v.chromeDPManager != nil {
		browser.CloseGlobalChromeDPManager()
		v.chromeDPManager = nil
	}
	
	if v.db != nil {
		v.addMessage("system", fmt.Sprintf(
			"Session ended. Total tokens: %d, Total cost: $%.4f",
			v.totalTokens, v.totalCost,
		))
		v.db.Close()
	}
}

// Cleanup public cleanup method
func (v *ChatAdventureV2View) Cleanup() {
	v.cleanup()
}

// Additional message types
type HTMLAnalysisMsg struct {
	CharCount     int
	Actions       []testing.DiscoveredAction
	PromptTokens  int
	EstimatedCost float64
	Error         error
}

type ActionExecutedMsg struct {
	Result string
	Error  error
}

type RestartConfigMsg struct{}

type NavigationStartMsg struct {
	URL string
}

type NavigationCompleteMsg struct {
	URL     string
	Success bool
	Error   error
}

type IncrementalActionsMsg struct {
	Actions    []testing.DiscoveredAction // New actions discovered
	IsInitial  bool                       // True if this is the initial set
	AllActions []testing.DiscoveredAction // Complete merged set (for updates)
}