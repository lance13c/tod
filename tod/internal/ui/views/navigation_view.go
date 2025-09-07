package views

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ciciliostudio/tod/internal/browser"
	"github.com/ciciliostudio/tod/internal/config"
	"github.com/ciciliostudio/tod/internal/llm"
)

// ElementType represents the type of navigable element
type ElementType int

const (
	LinkElement ElementType = iota
	ButtonElement
	FormElement
	ActionElement
)

// InputMode represents the current input mode
type InputMode int

const (
	TypingMode InputMode = iota
	SelectingMode
	CommandMode
)

// SuggestionType represents the type of suggestion
type SuggestionType int

const (
	LinkSuggestion SuggestionType = iota
	ActionSuggestion
	FormSuggestion
	CommandSuggestion
	HistorySuggestion
)

// NavigableElement represents an element that can be navigated to or interacted with
type NavigableElement struct {
	Type        ElementType
	Text        string
	Description string
	URL         string
	Selector    string
	Method      string // click, submit, type, etc.
	JavaScript  string // For complex actions
}

// Suggestion represents an autocomplete suggestion
type Suggestion struct {
	Type       SuggestionType
	Text       string
	Subtitle   string
	Element    *NavigableElement
	Command    *Command
	MatchScore float64
}

// Command represents a natural language command
type Command struct {
	Display     string
	Description string
	Handler     func(*NavigationView) error
}

// NavigationView provides a unified navigation interface
type NavigationView struct {
	// Configuration
	config        *config.Config
	llmClient     llm.Client
	configuredURL string

	// Browser management
	chromeDPManager *browser.ChromeDPManager
	isConnected     bool
	currentURL      string
	currentTitle    string

	// Input state
	input     textinput.Model
	inputMode InputMode

	// Suggestions state
	suggestions     []Suggestion
	selectedIndex   int
	showSuggestions bool
	maxSuggestions  int

	// Page state
	pageElements []NavigableElement
	isAnalyzing  bool

	// Navigation history
	navigationHistory []string
	historyIndex      int

	// Action history for display (Claude Code style)
	history    []string
	maxHistory int

	// UI components
	viewport     viewport.Model
	width        int
	height       int
	isProcessing bool

	// Styles
	titleStyle      lipgloss.Style
	inputStyle      lipgloss.Style
	suggestionStyle lipgloss.Style
	selectedStyle   lipgloss.Style
	subtitleStyle   lipgloss.Style
	borderStyle     lipgloss.Style
	helpStyle       lipgloss.Style
}

// NewNavigationView creates a new navigation view
func NewNavigationView(cfg *config.Config) *NavigationView {
	// Create text input
	ti := textinput.New()
	ti.Placeholder = "Type to navigate or command..."
	ti.CharLimit = 200
	ti.Width = 50
	ti.Focus()

	// Create viewport
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
		}

		if provider != "" {
			options := map[string]interface{}{
				"model": cfg.AI.Model,
			}
			llmClient, _ = llm.NewClient(provider, cfg.AI.APIKey, options)
		}
	}

	env := cfg.GetCurrentEnv()

	return &NavigationView{
		config:         cfg,
		llmClient:      llmClient,
		configuredURL:  env.BaseURL,
		input:          ti,
		viewport:       vp,
		selectedIndex:  -1,
		maxSuggestions: 10,
		maxHistory:     10, // Keep last 10 history messages
		width:          80,
		height:         25,

		// Styles
		titleStyle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			MarginBottom(1),

		inputStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(0, 1),

		suggestionStyle: lipgloss.NewStyle().
			PaddingLeft(2),

		selectedStyle: lipgloss.NewStyle().
			PaddingLeft(2).
			Background(lipgloss.Color("62")).
			Foreground(lipgloss.Color("230")).
			Bold(true),

		subtitleStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Italic(true),

		borderStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")),

		helpStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			MarginTop(1),
	}
}

// Init initializes the navigation view
func (v *NavigationView) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		v.connectToChrome(),
	)
}

// Update handles navigation view updates
func (v *NavigationView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
		v.viewport.Width = msg.Width - 4
		v.viewport.Height = msg.Height - 8
		v.input.Width = msg.Width - 10

	case tea.KeyMsg:
		return v.handleKeyPress(msg)

	case ChromeLaunchedMsg:
		v.isConnected = true
		return v, v.analyzeCurrentPage()

	case ChromeErrorMsg:
		v.isConnected = false
		// Handle Chrome errors similar to chat adventure

	case NavigationCompleteMsg:
		v.isProcessing = false
		if msg.Error == nil {
			v.currentURL = msg.URL
			v.addToHistory(msg.URL)
			// Add history message for successful navigation
			if msg.URL != v.configuredURL {
				v.addHistory(fmt.Sprintf("→ Navigated to %s", msg.URL))
			} else {
				v.addHistory("→ Navigated to homepage")
			}
			// Clear input after successful navigation and regenerate suggestions
			v.input.SetValue("")
			v.showSuggestions = false
			v.selectedIndex = -1
			return v, v.analyzeCurrentPage()
		}

	case PageAnalysisCompleteMsg:
		v.isAnalyzing = false
		if msg.Error == nil {
			v.pageElements = msg.Elements
			// Generate initial suggestions (will show even with empty input)
			v.generateSuggestions()
		}

	case NavigationErrorMsg:
		v.isProcessing = false
		// Could show error in status or as temporary message
	}

	// Update input
	var cmd tea.Cmd
	v.input, cmd = v.input.Update(msg)
	cmds = append(cmds, cmd)

	return v, tea.Batch(cmds...)
}

// View renders the navigation view
func (v *NavigationView) View() string {
	title := v.titleStyle.Render("Navigation Mode")

	// Status bar
	status := v.renderStatusBar()

	// History section (Claude Code style - simple text)
	var historyView string
	if len(v.history) > 0 {
		historyView = strings.Join(v.history, "\n")
	}

	// Input section
	inputView := lipgloss.JoinHorizontal(
		lipgloss.Left,
		"> ",
		v.input.View(),
	)

	// Suggestions section
	suggestionsView := v.renderSuggestions()

	// Help text
	help := v.helpStyle.Render("[Tab: complete] [↑↓: select] [Enter: go] [Esc: clear] [Ctrl+C: quit]")

	// Build the view - include history if it exists
	sections := []string{title, status}

	if historyView != "" {
		sections = append(sections, "", historyView)
	}

	sections = append(sections, "", inputView, "", suggestionsView, "", help)

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// handleKeyPress handles keyboard input
func (v *NavigationView) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		if v.showSuggestions {
			v.showSuggestions = false
			v.selectedIndex = -1
			return v, nil
		} else if v.input.Value() != "" {
			v.input.SetValue("")
			v.suggestions = []Suggestion{}
			return v, nil
		} else {
			return v, func() tea.Msg { return ReturnToMenuMsg{} }
		}

	case tea.KeyCtrlC:
		v.cleanup()
		return v, tea.Quit

	case tea.KeyUp:
		if v.showSuggestions && len(v.suggestions) > 0 {
			if v.selectedIndex > 0 {
				v.selectedIndex--
			} else {
				v.selectedIndex = len(v.suggestions) - 1
			}
		}
		return v, nil

	case tea.KeyDown:
		if !v.showSuggestions {
			v.generateSuggestions()
			v.showSuggestions = true
			v.selectedIndex = 0
		} else if len(v.suggestions) > 0 {
			if v.selectedIndex < len(v.suggestions)-1 {
				v.selectedIndex++
			} else {
				v.selectedIndex = 0
			}
		}
		return v, nil

	case tea.KeyTab:
		if v.showSuggestions && len(v.suggestions) > 0 && v.selectedIndex >= 0 {
			suggestion := v.suggestions[v.selectedIndex]
			v.input.SetValue(suggestion.Text)
			v.input.CursorEnd()
			v.showSuggestions = false
			v.selectedIndex = -1
		}
		return v, nil

	case tea.KeyEnter:
		if v.showSuggestions && len(v.suggestions) > 0 && v.selectedIndex >= 0 {
			// Store the selected suggestion before resetting state
			selectedSuggestion := v.suggestions[v.selectedIndex]
			// Clear input before executing suggestion
			v.input.SetValue("")
			v.showSuggestions = false
			v.selectedIndex = -1
			return v, v.executeSuggestion(selectedSuggestion)
		} else {
			// Clear input before executing direct input
			input := v.input.Value()
			v.input.SetValue("")
			v.showSuggestions = false
			v.selectedIndex = -1
			return v, v.executeInputValue(input)
		}

	case tea.KeyCtrlR:
		return v, v.analyzeCurrentPage()

	case tea.KeyCtrlB:
		return v, v.navigateBack()

	default:
		// Handle regular typing
		var cmd tea.Cmd
		v.input, cmd = v.input.Update(msg)

		// Generate suggestions after typing
		v.generateSuggestions()

		return v, cmd
	}
}

// connectToChrome establishes Chrome connection
func (v *NavigationView) connectToChrome() tea.Cmd {
	return func() tea.Msg {
		if v.chromeDPManager != nil {
			return ChromeLaunchedMsg{}
		}

		manager, err := browser.GetGlobalChromeDPManager(v.configuredURL, true) // headless=true
		if err != nil {
			return ChromeErrorMsg{Error: err}
		}

		v.chromeDPManager = manager
		return ChromeLaunchedMsg{}
	}
}

// analyzeCurrentPage analyzes the current page for navigable elements
func (v *NavigationView) analyzeCurrentPage() tea.Cmd {
	return func() tea.Msg {
		if v.chromeDPManager == nil {
			return PageAnalysisCompleteMsg{Error: fmt.Errorf("Chrome not connected")}
		}

		v.isAnalyzing = true

		// Get page info
		url, title, _ := v.chromeDPManager.GetPageInfo()
		v.currentURL = url
		v.currentTitle = title

		// Extract interactive elements
		interactiveElements, err := v.chromeDPManager.ExtractInteractiveElements()
		if err != nil {
			return PageAnalysisCompleteMsg{Error: err}
		}

		// Convert to NavigableElements
		var elements []NavigableElement
		for _, elem := range interactiveElements {
			navElement := NavigableElement{
				Text:        elem.Text,
				Description: elem.Text,
				Selector:    elem.Selector,
			}

			// Prioritize navigation elements first
			if elem.IsNavigation {
				navElement.Type = LinkElement
				navElement.URL = elem.FullUrl // Use full resolved URL
				navElement.Method = "navigate"
			} else if elem.IsButton {
				navElement.Type = ButtonElement
				navElement.Method = "click"
			} else {
				switch elem.Tag {
				case "a":
					navElement.Type = LinkElement
					navElement.URL = elem.FullUrl
					navElement.Method = "navigate"
				case "button":
					navElement.Type = ButtonElement
					navElement.Method = "click"
				case "input":
					if elem.Type == "submit" {
						navElement.Type = FormElement
						navElement.Method = "click"
					} else {
						navElement.Type = FormElement
						navElement.Method = "type"
					}
				default:
					navElement.Type = ActionElement
					navElement.Method = "click"
				}
			}

			elements = append(elements, navElement)
		}

		return PageAnalysisCompleteMsg{Elements: elements}
	}
}

// generateSuggestions creates suggestions based on current input
func (v *NavigationView) generateSuggestions() {
	input := strings.TrimSpace(v.input.Value())
	v.suggestions = []Suggestion{}

	// If input is empty, show top page elements and commands
	if input == "" {
		// Add common commands first
		commands := []Command{
			{Display: "go back", Description: "Navigate back in history"},
			{Display: "go to home", Description: "Navigate to homepage"},
			{Display: "refresh", Description: "Refresh current page"},
		}

		for _, cmd := range commands {
			v.suggestions = append(v.suggestions, Suggestion{
				Type:       CommandSuggestion,
				Text:       cmd.Display,
				Subtitle:   cmd.Description + " ⌘",
				Command:    &cmd,
				MatchScore: 1.0,
			})
		}

		// Add top page elements (prioritize links and buttons)
		elemCount := 0
		maxInitialElements := 7 // Show up to 7 elements when no input

		for _, elem := range v.pageElements {
			if elemCount >= maxInitialElements {
				break
			}

			suggestion := Suggestion{
				Type:       v.elementTypeToSuggestionType(elem.Type),
				Text:       elem.Text,
				Element:    &elem,
				MatchScore: 0.8,
			}

			switch elem.Type {
			case LinkElement:
				suggestion.Subtitle = "→ " + elem.URL
			case ButtonElement:
				suggestion.Subtitle = "button"
			case FormElement:
				suggestion.Subtitle = "📝 form"
			case ActionElement:
				suggestion.Subtitle = "action"
			}

			v.suggestions = append(v.suggestions, suggestion)
			elemCount++
		}

		v.showSuggestions = true
		v.selectedIndex = 0
		return
	}

	// Input provided - filter suggestions
	// Check for commands first
	if command := v.matchCommand(input); command != nil {
		v.suggestions = append(v.suggestions, Suggestion{
			Type:       CommandSuggestion,
			Text:       command.Display,
			Subtitle:   command.Description,
			Command:    command,
			MatchScore: 1.0,
		})
	}

	// Match page elements
	for _, elem := range v.pageElements {
		if score := v.fuzzyMatch(input, elem.Text); score > 0.3 {
			suggestion := Suggestion{
				Type:       v.elementTypeToSuggestionType(elem.Type),
				Text:       elem.Text,
				Element:    &elem,
				MatchScore: score,
			}

			switch elem.Type {
			case LinkElement:
				suggestion.Subtitle = "→ " + elem.URL
			case ButtonElement:
				suggestion.Subtitle = "button"
			case FormElement:
				suggestion.Subtitle = "📝 form"
			case ActionElement:
				suggestion.Subtitle = "action"
			}

			v.suggestions = append(v.suggestions, suggestion)
		}
	}

	// Add history matches
	for _, hist := range v.navigationHistory {
		if score := v.fuzzyMatch(input, hist); score > 0.5 {
			v.suggestions = append(v.suggestions, Suggestion{
				Type:       HistorySuggestion,
				Text:       hist,
				Subtitle:   "📜 history",
				MatchScore: score,
			})
		}
	}

	// Sort by relevance
	sort.Slice(v.suggestions, func(i, j int) bool {
		return v.suggestions[i].MatchScore > v.suggestions[j].MatchScore
	})

	// Limit suggestions
	if len(v.suggestions) > v.maxSuggestions {
		v.suggestions = v.suggestions[:v.maxSuggestions]
	}

	v.showSuggestions = len(v.suggestions) > 0
	if v.showSuggestions {
		v.selectedIndex = 0
	}
}

// renderStatusBar renders the status bar
func (v *NavigationView) renderStatusBar() string {
	var parts []string

	if v.isConnected {
		parts = append(parts, "Connected")
	} else {
		parts = append(parts, "Not connected")
	}

	if v.currentTitle != "" {
		parts = append(parts, fmt.Sprintf("%s", v.currentTitle))
	}

	if len(v.pageElements) > 0 {
		parts = append(parts, fmt.Sprintf("[%d elements]", len(v.pageElements)))
	}

	if v.isAnalyzing {
		parts = append(parts, "Analyzing...")
	}

	status := strings.Join(parts, " | ")
	return v.subtitleStyle.Render(status)
}

// renderSuggestions renders the suggestions list
func (v *NavigationView) renderSuggestions() string {
	// Always show suggestions if we have page elements (even with empty input)
	if len(v.suggestions) == 0 && len(v.pageElements) > 0 && !v.showSuggestions {
		v.generateSuggestions() // Generate initial suggestions
	}

	if len(v.suggestions) == 0 {
		return v.subtitleStyle.Render("[Analyzing page for navigation options...]")
	}

	// First pass: calculate the maximum width needed for alignment
	maxMainTextWidth := 0
	for _, suggestion := range v.suggestions {
		displayText := truncateText(suggestion.Text, 120)
		// Add 3 for the "├─ " prefix
		mainTextWidth := len(displayText) + 3
		if mainTextWidth > maxMainTextWidth {
			maxMainTextWidth = mainTextWidth
		}
	}

	var lines []string
	for i, suggestion := range v.suggestions {
		var line string
		var style lipgloss.Style

		// Truncate the main suggestion text
		displayText := truncateText(suggestion.Text, 120)

		if i == v.selectedIndex {
			style = v.selectedStyle
			line = fmt.Sprintf("┌─ %s", displayText)
		} else {
			style = v.suggestionStyle
			line = fmt.Sprintf("├─ %s", displayText)
		}

		// Calculate padding needed to align subtitles
		currentWidth := len(displayText) + 3                 // Add 3 for prefix
		paddingNeeded := maxMainTextWidth - currentWidth + 5 // Add 5 for spacing

		// Add subtitle with proper alignment
		if suggestion.Subtitle != "" {
			subtitle := truncateText(suggestion.Subtitle, 40)
			padding := strings.Repeat(" ", paddingNeeded)
			line += fmt.Sprintf("%s%s", padding, subtitle)
		}

		// Add type indicator
		switch suggestion.Type {
		case CommandSuggestion:
			line += " ⌘"
		}

		lines = append(lines, style.Render(line))
	}

	return strings.Join(lines, "\n")
}

// Helper functions and message types will be added in the next part...

// Message types
type PageAnalysisCompleteMsg struct {
	Elements []NavigableElement
	Error    error
}

// Cleanup cleans up resources
func (v *NavigationView) Cleanup() {
	v.cleanup()
}

func (v *NavigationView) cleanup() {
	if v.chromeDPManager != nil {
		browser.CloseGlobalChromeDPManager()
		v.chromeDPManager = nil
	}
}

// Placeholder methods to be implemented
func (v *NavigationView) executeSuggestion(suggestion Suggestion) tea.Cmd {
	return func() tea.Msg {
		v.isProcessing = true

		switch suggestion.Type {
		case CommandSuggestion:
			if suggestion.Command != nil && suggestion.Command.Handler != nil {
				// Add history for command execution
				v.addHistory(fmt.Sprintf("→ Executed: %s", suggestion.Command.Display))

				if err := suggestion.Command.Handler(v); err != nil {
					return NavigationErrorMsg{Error: err}
				}
				return NavigationCompleteMsg{
					URL:     v.currentURL,
					Success: true,
				}
			}

		case LinkSuggestion, ActionSuggestion, FormSuggestion:
			if suggestion.Element != nil {
				return v.executeElement(*suggestion.Element)()
			}

		case HistorySuggestion:
			if err := v.navigateToURL(suggestion.Text); err != nil {
				return NavigationErrorMsg{Error: err}
			}
			return NavigationCompleteMsg{
				URL:     suggestion.Text,
				Success: true,
			}
		}

		return NavigationErrorMsg{Error: fmt.Errorf("unknown suggestion type")}
	}
}

func (v *NavigationView) executeInput() tea.Cmd {
	return v.executeInputValue(v.input.Value())
}

func (v *NavigationView) executeInputValue(inputValue string) tea.Cmd {
	input := strings.TrimSpace(inputValue)
	if input == "" {
		return nil
	}

	return func() tea.Msg {
		v.isProcessing = true

		// First try to match as a command
		if command := v.matchCommand(input); command != nil {
			if command.Handler != nil {
				if err := command.Handler(v); err != nil {
					return NavigationErrorMsg{Error: err}
				}
				return NavigationCompleteMsg{
					URL:     v.currentURL,
					Success: true,
				}
			}
		}

		// Then try to match against page elements
		var bestMatch *NavigableElement
		bestScore := 0.0

		for _, elem := range v.pageElements {
			score := v.fuzzyMatch(input, elem.Text)
			if score > bestScore && score > 0.3 {
				bestScore = score
				bestMatch = &elem
			}
		}

		if bestMatch != nil {
			return v.executeElement(*bestMatch)()
		}

		// If no matches found, try interpreting as URL or search
		if strings.HasPrefix(input, "http") || strings.Contains(input, ".") {
			// Looks like a URL
			if err := v.navigateToURL(input); err != nil {
				return NavigationErrorMsg{Error: err}
			}
			return NavigationCompleteMsg{
				URL:     input,
				Success: true,
			}
		}

		return NavigationErrorMsg{Error: fmt.Errorf("no matches found for: %s", input)}
	}
}

func (v *NavigationView) executeElement(element NavigableElement) tea.Cmd {
	return func() tea.Msg {
		if v.chromeDPManager == nil {
			return NavigationErrorMsg{Error: fmt.Errorf("Chrome not connected")}
		}

		switch element.Method {
		case "navigate":
			if element.URL != "" {
				if err := v.chromeDPManager.Navigate(element.URL); err != nil {
					return NavigationErrorMsg{Error: err}
				}
				return NavigationCompleteMsg{
					URL:     element.URL,
					Success: true,
				}
			}

		case "click":
			if element.Selector != "" {
				// Wait for element and click
				if err := v.chromeDPManager.WaitForElement(element.Selector); err != nil {
					return NavigationErrorMsg{Error: fmt.Errorf("element not found: %w", err)}
				}

				if err := v.chromeDPManager.Click(element.Selector); err != nil {
					return NavigationErrorMsg{Error: err}
				}

				// Wait a bit for any navigation or changes
				time.Sleep(500 * time.Millisecond)

				// Get updated page info
				url, _, _ := v.chromeDPManager.GetPageInfo()

				// Add history message for the click action
				elementText := truncateText(element.Text, 30)
				if url != v.currentURL {
					// Navigation occurred after click
					return NavigationCompleteMsg{
						URL:     url,
						Success: true,
					}
				} else {
					// No navigation, just clicked element
					v.addHistory(fmt.Sprintf("→ Clicked \"%s\"", elementText))
					return NavigationCompleteMsg{
						URL:     url,
						Success: true,
					}
				}
			}

		case "type":
			// For form fields, focus and wait for user input
			return NavigationErrorMsg{Error: fmt.Errorf("form input not yet supported")}

		case "submit":
			if element.Selector != "" {
				if err := v.chromeDPManager.Click(element.Selector); err != nil {
					return NavigationErrorMsg{Error: err}
				}

				time.Sleep(1 * time.Second) // Wait for form submission

				url, _, _ := v.chromeDPManager.GetPageInfo()
				return NavigationCompleteMsg{
					URL:     url,
					Success: true,
				}
			}
		}

		return NavigationErrorMsg{Error: fmt.Errorf("unsupported element method: %s", element.Method)}
	}
}

func (v *NavigationView) matchCommand(input string) *Command {
	inputLower := strings.ToLower(strings.TrimSpace(input))

	commands := []Command{
		{
			Display:     "go to home",
			Description: "Navigate to homepage",
			Handler: func(v *NavigationView) error {
				return v.navigateToURL(v.configuredURL)
			},
		},
		{
			Display:     "go back",
			Description: "Navigate back in history",
			Handler: func(v *NavigationView) error {
				return v.goBack()
			},
		},
		{
			Display:     "refresh",
			Description: "Refresh current page analysis",
			Handler: func(v *NavigationView) error {
				return v.refreshPage()
			},
		},
		{
			Display:     "connect",
			Description: "Connect to Chrome browser",
			Handler: func(v *NavigationView) error {
				return v.reconnectChrome()
			},
		},
	}

	// Check for prefix matches with common command patterns
	commandPatterns := map[string]string{
		"go":       "go to home",
		"back":     "go back",
		"refresh":  "refresh",
		"reload":   "refresh",
		"connect":  "connect",
		"home":     "go to home",
		"homepage": "go to home",
	}

	// Direct pattern matching
	if cmd, exists := commandPatterns[inputLower]; exists {
		for _, command := range commands {
			if command.Display == cmd {
				return &command
			}
		}
	}

	// Fuzzy matching against command displays
	var bestMatch *Command
	bestScore := 0.0

	for _, command := range commands {
		score := v.fuzzyMatch(inputLower, command.Display)
		if score > 0.7 && score > bestScore {
			bestScore = score
			bestMatch = &command
		}
	}

	// Check for "go to [page]" pattern
	if strings.HasPrefix(inputLower, "go to ") {
		target := strings.TrimPrefix(inputLower, "go to ")
		if target != "" {
			return &Command{
				Display:     fmt.Sprintf("go to %s", target),
				Description: fmt.Sprintf("Navigate to %s", target),
				Handler: func(v *NavigationView) error {
					return v.navigateToTarget(target)
				},
			}
		}
	}

	// Check for "click [element]" pattern
	if strings.HasPrefix(inputLower, "click ") {
		target := strings.TrimPrefix(inputLower, "click ")
		if target != "" {
			return &Command{
				Display:     fmt.Sprintf("click %s", target),
				Description: fmt.Sprintf("Click on %s", target),
				Handler: func(v *NavigationView) error {
					return v.clickTarget(target)
				},
			}
		}
	}

	return bestMatch
}

func (v *NavigationView) fuzzyMatch(input, text string) float64 {
	if input == "" {
		return 0.0
	}

	inputLower := strings.ToLower(strings.TrimSpace(input))
	textLower := strings.ToLower(strings.TrimSpace(text))

	// Exact match gets highest score
	if inputLower == textLower {
		return 1.0
	}

	// Prefix match gets high score
	if strings.HasPrefix(textLower, inputLower) {
		return 0.9
	}

	// Contains match gets medium score
	if strings.Contains(textLower, inputLower) {
		// Score based on position and length ratio
		pos := strings.Index(textLower, inputLower)
		posScore := 1.0 - float64(pos)/float64(len(textLower))
		lengthScore := float64(len(inputLower)) / float64(len(textLower))
		return 0.6 + (posScore+lengthScore)/5.0
	}

	// Word boundary matching
	textWords := strings.Fields(textLower)
	inputWords := strings.Fields(inputLower)

	matchCount := 0
	for _, inputWord := range inputWords {
		for _, textWord := range textWords {
			if strings.HasPrefix(textWord, inputWord) || strings.Contains(textWord, inputWord) {
				matchCount++
				break
			}
		}
	}

	if matchCount > 0 {
		wordScore := float64(matchCount) / float64(len(textWords))
		return 0.3 + wordScore*0.3
	}

	// Character-based fuzzy matching for typos
	if len(inputLower) > 2 {
		charMatches := 0
		for _, char := range inputLower {
			if strings.ContainsRune(textLower, char) {
				charMatches++
			}
		}

		if charMatches >= len(inputLower)/2 {
			return float64(charMatches) / float64(len(inputLower)) * 0.2
		}
	}

	return 0.0
}

func (v *NavigationView) elementTypeToSuggestionType(elemType ElementType) SuggestionType {
	switch elemType {
	case LinkElement:
		return LinkSuggestion
	case ButtonElement:
		return ActionSuggestion
	case FormElement:
		return FormSuggestion
	default:
		return ActionSuggestion
	}
}

func (v *NavigationView) addToHistory(url string) {
	v.navigationHistory = append(v.navigationHistory, url)
	if len(v.navigationHistory) > 20 {
		v.navigationHistory = v.navigationHistory[1:]
	}
}

func (v *NavigationView) navigateBack() tea.Cmd {
	return func() tea.Msg {
		if v.historyIndex > 0 {
			v.historyIndex--
			url := v.navigationHistory[v.historyIndex]
			if err := v.navigateToURL(url); err != nil {
				return NavigationErrorMsg{Error: err}
			}
			return NavigationCompleteMsg{URL: url, Success: true}
		}
		return NavigationErrorMsg{Error: fmt.Errorf("no back history available")}
	}
}

// addHistory adds a simple text message to the history display
func (v *NavigationView) addHistory(message string) {
	v.history = append(v.history, message)
	if len(v.history) > v.maxHistory {
		v.history = v.history[1:] // Remove oldest message
	}
}

// truncateText truncates text to maxLen and adds "..." if needed
func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen-3] + "..."
}

// Helper methods for command handlers
func (v *NavigationView) navigateToURL(url string) error {
	if v.chromeDPManager == nil {
		return fmt.Errorf("Chrome not connected")
	}

	// Ensure URL has protocol
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		if strings.Contains(url, ".") {
			url = "https://" + url
		} else {
			// Try relative to current domain
			if v.currentURL != "" {
				url = v.currentURL + "/" + strings.TrimPrefix(url, "/")
			} else {
				url = v.configuredURL + "/" + strings.TrimPrefix(url, "/")
			}
		}
	}

	return v.chromeDPManager.Navigate(url)
}

func (v *NavigationView) goBack() error {
	if v.historyIndex > 0 {
		v.historyIndex--
		return v.navigateToURL(v.navigationHistory[v.historyIndex])
	}
	return fmt.Errorf("no back history available")
}

func (v *NavigationView) refreshPage() error {
	if v.currentURL != "" {
		return v.navigateToURL(v.currentURL)
	}
	return fmt.Errorf("no current page to refresh")
}

func (v *NavigationView) reconnectChrome() error {
	if v.chromeDPManager != nil {
		browser.CloseGlobalChromeDPManager()
		v.chromeDPManager = nil
		v.isConnected = false
	}

	manager, err := browser.GetGlobalChromeDPManager(v.configuredURL, true)
	if err != nil {
		return err
	}

	v.chromeDPManager = manager
	v.isConnected = true
	return nil
}

func (v *NavigationView) navigateToTarget(target string) error {
	// First, collect all navigation links
	var navigationLinks []NavigableElement
	var potentialMatches []NavigableElement

	for _, elem := range v.pageElements {
		if elem.Type == LinkElement && elem.URL != "" {
			navigationLinks = append(navigationLinks, elem)

			// Check for fuzzy matches
			score := v.fuzzyMatch(target, elem.Text)
			if score > 0.3 {
				potentialMatches = append(potentialMatches, elem)
			}
		}
	}

	// If we have potential matches, use the best one
	if len(potentialMatches) > 0 {
		bestMatch := potentialMatches[0]
		bestScore := v.fuzzyMatch(target, bestMatch.Text)

		for _, match := range potentialMatches[1:] {
			score := v.fuzzyMatch(target, match.Text)
			if score > bestScore {
				bestMatch = match
				bestScore = score
			}
		}

		// Add feedback about what we're navigating to
		v.addHistory(fmt.Sprintf("→ Found match: \"%s\" (score: %.1f)", bestMatch.Text, bestScore))
		return v.navigateToURL(bestMatch.URL)
	}

	// If no good matches found, show available navigation options
	if len(navigationLinks) > 0 {
		availableOptions := []string{}
		for _, link := range navigationLinks[:min(5, len(navigationLinks))] {
			if link.Text != "" {
				availableOptions = append(availableOptions, "\""+link.Text+"\"")
			}
		}

		if len(availableOptions) > 0 {
			optionsStr := strings.Join(availableOptions, ", ")
			v.addHistory(fmt.Sprintf("→ No match for \"%s\". Available options: %s", target, optionsStr))
			return fmt.Errorf("no navigation match found for \"%s\". Available options: %s", target, optionsStr)
		}
	}

	// Only try URL navigation if it looks like a URL or path
	if strings.HasPrefix(target, "http") || strings.HasPrefix(target, "/") || strings.Contains(target, ".") {
		return v.navigateToURL(target)
	}

	return fmt.Errorf("no navigation element found matching: %s", target)
}

func (v *NavigationView) clickTarget(target string) error {
	if v.chromeDPManager == nil {
		return fmt.Errorf("Chrome not connected")
	}

	// Find matching clickable element
	for _, elem := range v.pageElements {
		if v.fuzzyMatch(target, elem.Text) > 0.5 {
			if elem.Method == "click" && elem.Selector != "" {
				if err := v.chromeDPManager.WaitForElement(elem.Selector); err != nil {
					return fmt.Errorf("element not found: %w", err)
				}
				return v.chromeDPManager.Click(elem.Selector)
			}
		}
	}

	return fmt.Errorf("no clickable element found matching: %s", target)
}

// Helper function to get minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Additional message types
type NavigationErrorMsg struct {
	Error error
}
