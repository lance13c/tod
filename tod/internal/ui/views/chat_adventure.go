package views

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ciciliostudio/tod/internal/browser"
	"github.com/ciciliostudio/tod/internal/config"
	"github.com/ciciliostudio/tod/internal/llm"
	"github.com/ciciliostudio/tod/internal/testing"
)

// ChatMessage represents a single message in the chat
type ChatMessage struct {
	Role       string    // "user", "assistant", "system"
	Content    string
	Timestamp  time.Time
	IsError    bool
	IsThinking bool      // indicates if this is a thinking message
}

// ChatAdventureView provides a conversational interface for testing
type ChatAdventureView struct {
	// Configuration
	config       *config.Config
	llmClient    llm.Client
	configuredURL string
	
	// Browser management
	chromeDPManager *browser.ChromeDPManager
	isConnected     bool
	currentURL      string
	currentTitle    string
	
	// Chat interface
	messages     []ChatMessage
	viewport     viewport.Model
	textarea     textarea.Model
	
	// Action discovery
	actionDiscovery *testing.ActionDiscovery
	
	// UI state
	width        int
	height       int
	isProcessing bool
	
	// Styles
	userStyle      lipgloss.Style
	assistantStyle lipgloss.Style
	systemStyle    lipgloss.Style
	errorStyle     lipgloss.Style
	borderStyle    lipgloss.Style
	titleStyle     lipgloss.Style
}

// NewChatAdventureView creates a new chat-based adventure view
func NewChatAdventureView(cfg *config.Config) *ChatAdventureView {
	// Create textarea for user input
	ta := textarea.New()
	ta.Placeholder = "Type your testing request... (Ctrl+Enter to send)"
	ta.CharLimit = 500
	ta.SetWidth(80)
	ta.SetHeight(3)
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
	
	view := &ChatAdventureView{
		config:        cfg,
		llmClient:     llmClient,
		configuredURL: env.BaseURL,
		messages:      []ChatMessage{},
		viewport:      vp,
		textarea:      ta,
		width:         80,
		height:        25,
		
		// Styles
		userStyle:      lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Bold(true),
		assistantStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("250")),
		systemStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Italic(true),
		errorStyle:     lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true),
		borderStyle:    lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("62")),
		titleStyle:     lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")).MarginBottom(1),
	}
	
	// Initialize action discovery if LLM is available
	if llmClient != nil {
		view.actionDiscovery = testing.NewActionDiscovery(llmClient, "playwright")
	}
	
	// Add welcome message
	view.addMessage("system", "Welcome to Chat Adventure Mode! I'm your AI testing assistant.")
	view.addMessage("system", "I can help you test your application interactively. Just tell me what you'd like to test!")
	view.addMessage("system", fmt.Sprintf("Configured URL: %s", env.BaseURL))
	view.addMessage("system", "Commands: 'connect' to launch Chrome, 'analyze' to discover actions, 'navigate <url>' to go to a page")
	
	return view
}

// Init initializes the view
func (v *ChatAdventureView) Init() tea.Cmd {
	return tea.Batch(
		textarea.Blink,
		v.connectToChrome(),
	)
}

// Update handles messages
func (v *ChatAdventureView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
		v.viewport.Width = msg.Width - 4
		v.viewport.Height = msg.Height - 10
		v.textarea.SetWidth(msg.Width - 4)
		
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc:
			return v, func() tea.Msg { return ReturnToMenuMsg{} }
		case tea.KeyCtrlC:
			browser.CloseGlobalChromeDPManager()
			return v, tea.Quit
		case tea.KeyCtrlS: // Ctrl+S to send message (alternative to Ctrl+Enter)
			if !v.isProcessing {
				return v, v.handleUserInput()
			}
		case tea.KeyEnter:
			if msg.Alt { // Alt+Enter to send
				if !v.isProcessing {
					return v, v.handleUserInput()
				}
			}
		}
		
	case ChromeLaunchedMsg:
		v.isConnected = true
		v.addMessage("system", "✓ Chrome connected successfully!")
		return v, v.updateChromeStatus()
		
	case ChromeStatusMsg:
		v.currentURL = msg.URL
		v.currentTitle = msg.Title
		if msg.URL != "" {
			v.addMessage("system", fmt.Sprintf("Current page: %s", msg.Title))
		}
		
	case ChromeErrorMsg:
		v.addMessage("system", fmt.Sprintf("Chrome error: %v", msg.Error))
		
	case AnalyzeResultMsg:
		v.isProcessing = false
		if msg.Error != nil {
			v.addMessage("assistant", fmt.Sprintf("Error analyzing page: %v", msg.Error))
		} else {
			if len(msg.Actions) > 0 {
				v.addMessage("assistant", "Discovered actions:")
				for i, action := range msg.Actions {
					v.addMessage("assistant", fmt.Sprintf("%d. %s - %s", i+1, action.Selector, action.Description))
				}
			}
		}
		
	case ProcessingCompleteMsg:
		v.isProcessing = false
		if msg.Error != nil {
			v.addMessage("assistant", fmt.Sprintf("Error: %v", msg.Error))
		} else if msg.Message != "" {
			v.addMessage("assistant", msg.Message)
		}
	}
	
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
func (v *ChatAdventureView) View() string {
	title := v.titleStyle.Render("🤖 Chat Adventure Mode - AI Testing Assistant")
	
	// Status bar
	status := v.renderStatusBar()
	
	// Chat viewport with border
	chatView := v.borderStyle.Width(v.width - 2).Height(v.viewport.Height + 2).Render(v.viewport.View())
	
	// Input area
	inputLabel := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("Your message:")
	inputArea := v.textarea.View()
	
	// Processing indicator
	if v.isProcessing {
		inputArea = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Italic(true).Render("Processing...")
	}
	
	// Help text
	help := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("[Ctrl+Enter to send] [Esc to return to menu]")
	
	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		status,
		chatView,
		inputLabel,
		inputArea,
		help,
	)
}

// handleUserInput processes user input
func (v *ChatAdventureView) handleUserInput() tea.Cmd {
	input := strings.TrimSpace(v.textarea.Value())
	if input == "" {
		return nil
	}
	
	// Add user message
	v.addMessage("user", input)
	v.textarea.Reset()
	v.isProcessing = true
	
	// Parse commands or process with AI
	return v.processUserInput(input)
}

// processUserInput handles the user's input
func (v *ChatAdventureView) processUserInput(input string) tea.Cmd {
	lowered := strings.ToLower(input)
	
	// Handle commands
	switch {
	case strings.HasPrefix(lowered, "connect"):
		return v.connectToChrome()
		
	case strings.HasPrefix(lowered, "navigate "):
		url := strings.TrimSpace(strings.TrimPrefix(input, "navigate "))
		return v.navigateToURL(url)
		
	case strings.HasPrefix(lowered, "analyze"):
		return v.analyzePage(input)
		
	case strings.HasPrefix(lowered, "help"):
		return v.showHelp()
		
	default:
		// Process with AI for natural language requests
		return v.processWithAI(input)
	}
}

// connectToChrome establishes Chrome connection
func (v *ChatAdventureView) connectToChrome() tea.Cmd {
	return func() tea.Msg {
		if v.chromeDPManager != nil {
			return ProcessingCompleteMsg{Message: "Chrome is already connected"}
		}
		
		manager, err := browser.GetGlobalChromeDPManager(v.configuredURL, false) // headless=false for chat mode
		if err != nil {
			return ChromeErrorMsg{Error: err}
		}
		
		v.chromeDPManager = manager
		return ChromeLaunchedMsg{}
	}
}

// navigateToURL navigates Chrome to a specific URL
func (v *ChatAdventureView) navigateToURL(url string) tea.Cmd {
	return func() tea.Msg {
		if v.chromeDPManager == nil {
			return ProcessingCompleteMsg{Error: fmt.Errorf("Chrome not connected. Use 'connect' first")}
		}
		
		// Ensure URL has protocol
		if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
			url = "https://" + url
		}
		
		if err := v.chromeDPManager.Navigate(url); err != nil {
			return ProcessingCompleteMsg{Error: fmt.Errorf("navigation failed: %w", err)}
		}
		
		// Get page info after navigation
		pageURL, title, err := v.chromeDPManager.GetPageInfo()
		if err != nil {
			return ProcessingCompleteMsg{Message: fmt.Sprintf("Navigated to %s", url)}
		}
		
		return ProcessingCompleteMsg{Message: fmt.Sprintf("Navigated to: %s\nTitle: %s", pageURL, title)}
	}
}

// analyzePage analyzes the current page for testing opportunities
func (v *ChatAdventureView) analyzePage(userContext string) tea.Cmd {
	return func() tea.Msg {
		if v.chromeDPManager == nil {
			return AnalyzeResultMsg{Error: fmt.Errorf("Chrome not connected")}
		}
		
		if v.actionDiscovery == nil {
			return AnalyzeResultMsg{Error: fmt.Errorf("AI not configured")}
		}
		
		// Get page HTML
		html, err := v.chromeDPManager.GetPageHTML()
		if err != nil {
			return AnalyzeResultMsg{Error: fmt.Errorf("failed to get page HTML: %w", err)}
		}
		
		// Extract context from user input
		context := ""
		if strings.HasPrefix(strings.ToLower(userContext), "analyze ") {
			context = strings.TrimSpace(strings.TrimPrefix(userContext, "analyze "))
		}
		
		// Discover actions
		actions, summary, _, err := v.actionDiscovery.DiscoverActionsFromHTMLWithContext(
			nil, // context.Context would go here
			html,
			[]string{},
			context,
		)
		
		if err != nil {
			return AnalyzeResultMsg{Error: err}
		}
		
		// Add summary as a message if available
		if summary != "" {
			return ProcessingCompleteMsg{Message: summary}
		}
		
		return AnalyzeResultMsg{
			Actions: actions,
		}
	}
}

// processWithAI handles natural language requests with AI
func (v *ChatAdventureView) processWithAI(input string) tea.Cmd {
	return func() tea.Msg {
		if v.llmClient == nil {
			return ProcessingCompleteMsg{Message: "I understand you want to: " + input + "\n\nHowever, AI is not configured. You can use commands like 'navigate', 'analyze', or 'help' instead."}
		}
		
		// Build context from recent messages
		context := v.buildAIContext()
		
		// Create prompt for AI (currently unused, would need proper LLM integration)
		_ = fmt.Sprintf(`You are a helpful testing assistant for a web application. 
The user is currently testing: %s
Current page: %s

Recent conversation:
%s

User request: %s

Provide helpful guidance for testing. If they want to perform specific actions, suggest the appropriate commands.
Available commands: connect, navigate <url>, analyze [context], help

Be concise and helpful.`, v.configuredURL, v.currentTitle, context, input)
		
		// Get AI response (this would need proper context handling)
		// For now, return a helpful message
		response := v.generateAIResponse(input)
		
		return ProcessingCompleteMsg{Message: response}
	}
}

// generateAIResponse creates a helpful response based on input
func (v *ChatAdventureView) generateAIResponse(input string) string {
	lowered := strings.ToLower(input)
	
	switch {
	case strings.Contains(lowered, "test") && strings.Contains(lowered, "login"):
		return "To test login functionality:\n1. Navigate to the login page with 'navigate /login'\n2. Use 'analyze login form' to discover form elements\n3. I can help you create test scenarios for different login cases"
		
	case strings.Contains(lowered, "find") && strings.Contains(lowered, "bug"):
		return "To find bugs:\n1. Use 'analyze' to discover all interactive elements\n2. Try edge cases like empty inputs, special characters\n3. Check error handling and validation messages\n4. Test navigation flows and back button behavior"
		
	case strings.Contains(lowered, "start"):
		return "Let's start testing! First:\n1. Make sure Chrome is connected (it should auto-connect)\n2. Navigate to the page you want to test\n3. Use 'analyze' to see what actions are available\n4. Tell me what specific functionality you'd like to test"
		
	default:
		return fmt.Sprintf("I understand you want to: %s\n\nTry using:\n- 'navigate <url>' to go to a specific page\n- 'analyze' to discover testable elements\n- 'help' for more commands", input)
	}
}

// showHelp displays help information
func (v *ChatAdventureView) showHelp() tea.Cmd {
	return func() tea.Msg {
		help := `Available Commands:
• connect - Connect to Chrome browser
• navigate <url> - Go to a specific URL
• analyze [context] - Discover testable actions on current page
• help - Show this help message

Natural Language:
You can also describe what you want to test in plain English!
Examples:
- "I want to test the login form"
- "Find potential bugs on this page"
- "Test the checkout process"

Tips:
- Chrome connects automatically on startup
- Use Ctrl+Enter to send messages
- Press Esc to return to main menu`
		
		return ProcessingCompleteMsg{Message: help}
	}
}

// updateChromeStatus gets current Chrome status
func (v *ChatAdventureView) updateChromeStatus() tea.Cmd {
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

// addMessage adds a message to the chat
func (v *ChatAdventureView) addMessage(role, content string) {
	v.messages = append(v.messages, ChatMessage{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
	})
	v.viewport.SetContent(v.renderMessages())
	v.viewport.GotoBottom()
}

// renderMessages renders all chat messages
func (v *ChatAdventureView) renderMessages() string {
	var lines []string
	
	for _, msg := range v.messages {
		var style lipgloss.Style
		var prefix string
		
		switch msg.Role {
		case "user":
			style = v.userStyle
			prefix = "You: "
		case "assistant":
			style = v.assistantStyle
			prefix = "Tod: "
		case "system":
			style = v.systemStyle
			prefix = "System: "
		}
		
		if msg.IsError {
			style = v.errorStyle
		}
		
		// Format message with timestamp
		timestamp := msg.Timestamp.Format("15:04")
		line := fmt.Sprintf("[%s] %s%s", timestamp, prefix, msg.Content)
		lines = append(lines, style.Render(line))
		lines = append(lines, "") // Add spacing between messages
	}
	
	return strings.Join(lines, "\n")
}

// renderStatusBar renders the status bar
func (v *ChatAdventureView) renderStatusBar() string {
	var status string
	
	if v.isConnected {
		status = fmt.Sprintf("✓ Connected | %s", v.currentURL)
		if v.currentTitle != "" {
			status = fmt.Sprintf("✓ Connected | %s - %s", v.currentTitle, v.currentURL)
		}
	} else {
		status = "⚠ Not connected to Chrome"
	}
	
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Background(lipgloss.Color("235")).
		Width(v.width).
		Padding(0, 1).
		Render(status)
}

// buildAIContext builds context from recent messages
func (v *ChatAdventureView) buildAIContext() string {
	var context []string
	
	// Get last 5 messages for context
	start := len(v.messages) - 5
	if start < 0 {
		start = 0
	}
	
	for _, msg := range v.messages[start:] {
		if msg.Role != "system" {
			context = append(context, fmt.Sprintf("%s: %s", msg.Role, msg.Content))
		}
	}
	
	return strings.Join(context, "\n")
}

// Cleanup cleans up resources
func (v *ChatAdventureView) Cleanup() {
	if v.chromeDPManager != nil {
		browser.CloseGlobalChromeDPManager()
		v.chromeDPManager = nil
	}
}

// Message types for chat adventure
type ProcessingCompleteMsg struct {
	Message         string
	Error           error
	TriggerAnalysis bool
}

type ChromeStatusMsg struct {
	URL   string
	Title string
}

type ChromeErrorMsg struct {
	Error error
}