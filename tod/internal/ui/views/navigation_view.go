package views

import (
	"context"
	"fmt"
	"log"
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
	"github.com/ciciliostudio/tod/internal/users"
)

// ElementType represents the type of navigable element
type ElementType int

const (
	LinkElement ElementType = iota
	ButtonElement
	FormElement
	ActionElement
	FormFieldElement
)

// InputMode represents the current input mode
type InputMode int

const (
	TypingMode InputMode = iota
	SelectingMode
	CommandMode
	FormInputMode
)

// SuggestionType represents the type of suggestion
type SuggestionType int

const (
	LinkSuggestion SuggestionType = iota
	ActionSuggestion
	FormSuggestion
	FormFieldSuggestion
	CommandSuggestion
	HistorySuggestion
	SectionHeaderSuggestion
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

	// Scrolling state for suggestions viewport
	suggestionScrollOffset   int // Current scroll offset in suggestions list
	visibleSuggestionCount   int // Number of suggestions that fit in viewport
	maxVisibleSuggestions    int // Maximum suggestions to show (calculated from terminal height)

	// Page state
	pageElements []NavigableElement
	isAnalyzing  bool

	// Form handling
	formHandler     *FormHandler
	authConfig      *AuthConfigManager
	inputModal      *InputModal
	currentForm     *LoginForm
	formProcessing  bool
	awaitingInput   bool
	pendingField    *FormField

	// Authentication flow
	authFlow       *users.AuthFlowManager
	isAuthenticating bool

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

	// Initialize auth config manager (needs project directory)
	projectDir := "." // Default to current directory, could be made configurable
	authConfig := NewAuthConfigManager(projectDir)

	// Initialize auth flow manager
	var authFlow *users.AuthFlowManager
	if llmClient != nil {
		authFlow, _ = users.NewAuthFlowManager(projectDir, llmClient)
	}

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

		// Initialize new components
		authConfig: authConfig,
		authFlow:   authFlow,

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
		} else {
			// Clear elements on error
			v.pageElements = []NavigableElement{}
		}

	case NavigationErrorMsg:
		v.isProcessing = false
		// Could show error in status or as temporary message

	case AuthenticationCompleteMsg:
		v.isAuthenticating = false
		if msg.Success {
			v.addHistory("🎉 Authentication completed successfully")
		} else {
			v.addHistory(fmt.Sprintf("❌ Authentication failed: %v", msg.Error))
		}

	case FormInputModalReadyMsg:
		// The modal has been created and shown, just need to trigger UI update
		return v, nil
	}

	// Update input
	var cmd tea.Cmd
	v.input, cmd = v.input.Update(msg)
	cmds = append(cmds, cmd)

	return v, tea.Batch(cmds...)
}

// View renders the navigation view with fixed header and scrollable suggestions
func (v *NavigationView) View() string {
	// Calculate viewport dimensions
	v.calculateViewportDimensions()

	// Fixed header section (always visible)
	title := v.titleStyle.Render("Navigation Mode")
	status := v.renderStatusBar()

	// History section (Claude Code style - simple text)
	var historyView string
	if len(v.history) > 0 {
		historyView = strings.Join(v.history, "\n")
	}

	// Input section (always visible)
	inputView := lipgloss.JoinHorizontal(
		lipgloss.Left,
		"> ",
		v.input.View(),
	)

	// Build fixed header
	headerSections := []string{title, status}
	if historyView != "" {
		headerSections = append(headerSections, "", historyView)
	}
	headerSections = append(headerSections, "", inputView)
	
	fixedHeader := lipgloss.JoinVertical(lipgloss.Left, headerSections...)

	// Scrollable suggestions section (uses available space)
	suggestionsView := v.renderSuggestionsViewport()

	// Help text (always visible at bottom)
	help := v.helpStyle.Render("[Tab: complete] [↑↓: navigate] [Enter: go] [Esc: clear] [Ctrl+C: quit]")

	// Combine fixed header + scrollable suggestions + help
	mainView := lipgloss.JoinVertical(lipgloss.Left, fixedHeader, "", suggestionsView, "", help)

	// If input modal is showing, overlay it
	if v.inputModal != nil && v.inputModal.IsShowing() {
		modalView := v.inputModal.View()
		// Simple overlay - center the modal on screen
		return mainView + "\n" + modalView
	}

	return mainView
}

// handleKeyPress handles keyboard input
func (v *NavigationView) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// If input modal is showing, handle modal input first
	if v.inputModal != nil && v.inputModal.IsShowing() {
		var cmd tea.Cmd
		v.inputModal, cmd = v.inputModal.Update(msg)
		
		// Check if modal completed
		if v.inputModal.IsComplete() {
			return v.handleModalComplete()
		}
		
		return v, cmd
	}

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
			v.selectedIndex = v.findNextSelectableIndex(v.selectedIndex, -1)
			// Ensure selected item is visible in viewport
			v.scrollToShowSelected()
			// Populate input field with selected suggestion
			if v.selectedIndex >= 0 && v.selectedIndex < len(v.suggestions) {
				suggestion := v.suggestions[v.selectedIndex]
				if suggestion.Type != SectionHeaderSuggestion {
					v.input.SetValue(suggestion.Text)
					v.input.CursorEnd()
				}
			}
		}
		return v, nil

	case tea.KeyDown:
		if !v.showSuggestions {
			v.generateSuggestions()
			v.showSuggestions = true
			v.selectedIndex = v.findFirstSelectableIndex()
			v.scrollToShowSelected()
		} else if len(v.suggestions) > 0 {
			v.selectedIndex = v.findNextSelectableIndex(v.selectedIndex, 1)
			// Ensure selected item is visible in viewport
			v.scrollToShowSelected()
		}
		// Populate input field with selected suggestion
		if v.showSuggestions && v.selectedIndex >= 0 && v.selectedIndex < len(v.suggestions) {
			suggestion := v.suggestions[v.selectedIndex]
			if suggestion.Type != SectionHeaderSuggestion {
				v.input.SetValue(suggestion.Text)
				v.input.CursorEnd()
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
		
		// Initialize form handler with the chrome manager
		v.formHandler = NewFormHandler(manager)
		
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

		// Wait for page to be fully loaded before analyzing
		if err := v.chromeDPManager.WaitForPageLoad(5 * time.Second); err != nil {
			fmt.Printf("Warning: page load wait failed: %v\n", err)
		}

		// Get page info
		url, title, _ := v.chromeDPManager.GetPageInfo()
		v.currentURL = url
		v.currentTitle = title

		fmt.Printf("🔍 Analyzing page: %s (title: %s)\n", url, title)

		// Extract interactive elements
		interactiveElements, err := v.chromeDPManager.ExtractInteractiveElements()
		if err != nil {
			fmt.Printf("❌ Failed to extract interactive elements: %v\n", err)
			return PageAnalysisCompleteMsg{Error: err}
		}

		fmt.Printf("Found %d interactive elements\n", len(interactiveElements))

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
					} else if elem.Type == "checkbox" || elem.Type == "radio" {
						navElement.Type = ActionElement
						navElement.Method = "click"
						navElement.Description = fmt.Sprintf("Toggle %s", elem.Text)
					} else {
						navElement.Type = FormElement
						navElement.Method = "type"
					}
				case "select":
					navElement.Type = ActionElement
					navElement.Method = "click" 
					navElement.Description = fmt.Sprintf("Select from %s", elem.Text)
				case "textarea":
					navElement.Type = FormElement
					navElement.Method = "type"
				default:
					// Handle other interactive elements
					navElement.Type = ActionElement
					navElement.Method = "click"
				}
			}

			elements = append(elements, navElement)
		}

		// Also detect forms if form handler is available
		if v.formHandler != nil {
			fmt.Printf("Form handler available, detecting forms on %s...\n", url)
			if form, err := v.formHandler.DetectLoginForm(); err == nil && form != nil {
				fmt.Printf("✅ Form detected: Domain=%s, EmailField=%v, PasswordField=%v, IsMagicLink=%v, IsComplete=%v\n", 
					form.Domain, form.EmailField != nil, form.PasswordField != nil, form.IsMagicLink, form.IsComplete)
				v.currentForm = form
				v.addFormFieldsToElements(&elements, form)
				fmt.Printf("Added %d form field actions to elements\n", countFormFields(form))
			} else {
				if err != nil {
					fmt.Printf("❌ Form detection failed: %v\n", err)
				} else {
					fmt.Printf("⚠️ No forms detected on page\n")
				}
			}
		} else {
			fmt.Printf("⚠️ Form handler not available\n")
		}

		fmt.Printf("✅ Page analysis complete: %d total navigable elements\n", len(elements))
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

		// Add ALL page elements with smart categorization and prioritization
		// Separate elements by type for better organization
		var formFields []NavigableElement
		var formSubmits []NavigableElement
		var buttons []NavigableElement
		var navigationLinks []NavigableElement
		var otherElements []NavigableElement
		
		for _, elem := range v.pageElements {
			switch elem.Type {
			case FormFieldElement:
				formFields = append(formFields, elem)
			case FormElement:
				if elem.Method == "form_submit" {
					formSubmits = append(formSubmits, elem)
				} else {
					otherElements = append(otherElements, elem)
				}
			case ButtonElement:
				buttons = append(buttons, elem)
			case LinkElement:
				navigationLinks = append(navigationLinks, elem)
			case ActionElement:
				otherElements = append(otherElements, elem)
			default:
				otherElements = append(otherElements, elem)
			}
		}
		
		// Create prioritized list based on context
		prioritizedElements := []NavigableElement{}
		
		if v.currentForm != nil {
			// Form context: prioritize form actions first
			prioritizedElements = append(prioritizedElements, formFields...)
			prioritizedElements = append(prioritizedElements, formSubmits...)
			prioritizedElements = append(prioritizedElements, buttons...)
			prioritizedElements = append(prioritizedElements, navigationLinks...)
			prioritizedElements = append(prioritizedElements, otherElements...)
		} else {
			// Regular context: prioritize interactive elements
			prioritizedElements = append(prioritizedElements, buttons...)
			prioritizedElements = append(prioritizedElements, formFields...)
			prioritizedElements = append(prioritizedElements, formSubmits...)
			prioritizedElements = append(prioritizedElements, navigationLinks...)
			prioritizedElements = append(prioritizedElements, otherElements...)
		}

		// Add ALL elements as suggestions with section headers
		v.addSuggestionsWithHeaders(formFields, formSubmits, buttons, navigationLinks, otherElements)

		v.showSuggestions = true
		v.selectedIndex = v.findFirstSelectableIndex()
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
			case FormFieldElement:
				suggestion.Subtitle = "📝 input"
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

// addSuggestionsWithHeaders adds suggestions organized by category with section headers
func (v *NavigationView) addSuggestionsWithHeaders(formFields, formSubmits, buttons, navigationLinks, otherElements []NavigableElement) {
	// Helper function to add section header
	addHeader := func(title string) {
		if len(v.suggestions) > 0 { // Don't add header as first item
			v.suggestions = append(v.suggestions, Suggestion{
				Type:       SectionHeaderSuggestion,
				Text:       title,
				Subtitle:   "",
				MatchScore: 0.0,
			})
		}
	}

	// Helper function to add elements of a specific type
	addElementsOfType := func(elements []NavigableElement, header string) {
		if len(elements) > 0 {
			addHeader(header)
			for _, elem := range elements {
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
					suggestion.Subtitle = "🎯 button"
				case FormElement:
					suggestion.Subtitle = "📝 form submit"
				case FormFieldElement:
					suggestion.Subtitle = "📝 input field"
				case ActionElement:
					suggestion.Subtitle = "⚡ action"
				}

				v.suggestions = append(v.suggestions, suggestion)
			}
		}
	}

	// Add sections in priority order based on context
	if v.currentForm != nil {
		// Form context: prioritize form actions
		addElementsOfType(formFields, "📝 Form Fields")
		addElementsOfType(formSubmits, "📝 Form Actions")
		addElementsOfType(buttons, "🎯 Buttons")
		addElementsOfType(navigationLinks, "🔗 Navigation")
		addElementsOfType(otherElements, "⚡ Other Actions")
	} else {
		// Regular context: prioritize interactive elements
		addElementsOfType(buttons, "🎯 Buttons")
		addElementsOfType(formFields, "📝 Form Fields")
		addElementsOfType(formSubmits, "📝 Form Actions")
		addElementsOfType(navigationLinks, "🔗 Navigation")
		addElementsOfType(otherElements, "⚡ Other Actions")
	}
}

// findNextSelectableIndex finds the next selectable suggestion index (skipping headers)
func (v *NavigationView) findNextSelectableIndex(currentIndex int, direction int) int {
	if len(v.suggestions) == 0 {
		return -1
	}
	
	startIndex := currentIndex
	for {
		currentIndex += direction
		
		// Wrap around
		if currentIndex >= len(v.suggestions) {
			currentIndex = 0
		} else if currentIndex < 0 {
			currentIndex = len(v.suggestions) - 1
		}
		
		// Found a selectable suggestion
		if v.suggestions[currentIndex].Type != SectionHeaderSuggestion {
			return currentIndex
		}
		
		// Prevent infinite loop if only headers exist (shouldn't happen)
		if currentIndex == startIndex {
			return -1
		}
	}
}

// findFirstSelectableIndex finds the first selectable suggestion (not a header)
func (v *NavigationView) findFirstSelectableIndex() int {
	for i, suggestion := range v.suggestions {
		if suggestion.Type != SectionHeaderSuggestion {
			return i
		}
	}
	return -1
}

// scrollToShowSelected ensures the selected item is visible in the viewport
func (v *NavigationView) scrollToShowSelected() {
	if v.selectedIndex < 0 || v.selectedIndex >= len(v.suggestions) {
		return
	}
	
	// Calculate current viewport bounds
	viewportStart := v.suggestionScrollOffset
	viewportEnd := viewportStart + v.maxVisibleSuggestions
	
	// If selected item is above the viewport, scroll up
	if v.selectedIndex < viewportStart {
		v.suggestionScrollOffset = v.selectedIndex
	}
	
	// If selected item is below the viewport, scroll down
	if v.selectedIndex >= viewportEnd {
		v.suggestionScrollOffset = v.selectedIndex - v.maxVisibleSuggestions + 1
	}
	
	// Ensure scroll offset stays within valid bounds
	if v.suggestionScrollOffset < 0 {
		v.suggestionScrollOffset = 0
	}
	
	maxOffset := max(0, len(v.suggestions) - v.maxVisibleSuggestions)
	if v.suggestionScrollOffset > maxOffset {
		v.suggestionScrollOffset = maxOffset
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

	if v.isAnalyzing {
		parts = append(parts, "Analyzing...")
	} else if len(v.pageElements) > 0 {
		parts = append(parts, fmt.Sprintf("[%d elements]", len(v.pageElements)))
	} else {
		parts = append(parts, "[0 elements]")
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

		// Handle section headers differently
		if suggestion.Type == SectionHeaderSuggestion {
			// Section headers use special styling
			style = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("208")). // Orange color for headers
				PaddingTop(1).
				PaddingBottom(0)
			
			line = fmt.Sprintf("▶ %s", suggestion.Text)
			// Headers are not selectable, so skip selection styling
		} else {
			// Regular suggestions
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
		}

		lines = append(lines, style.Render(line))
	}

	return strings.Join(lines, "\n")
}

// calculateViewportDimensions calculates how many suggestions can fit in available space
func (v *NavigationView) calculateViewportDimensions() {
	// Calculate fixed header height
	// Title: 1 line + 1 margin = 2
	// Status: 1 line = 1
	// History: variable (max 3 lines to keep header reasonable) + 1 margin
	// Input: 1 line + 1 margin = 2
	// Help: 1 line + 2 margins = 3
	// Additional spacing: 2-3 lines
	
	fixedHeaderLines := 2 + 1 // Title + status
	
	// Add history (limit to 3 recent items to prevent header from being too tall)
	historyLines := 0
	if len(v.history) > 0 {
		historyLines = min(3, len(v.history)) + 1 // +1 for spacing
	}
	
	fixedHeaderLines += historyLines + 2 + 3 + 3 // input + spacing + help + margins
	
	// Calculate available lines for suggestions
	availableHeight := v.height - fixedHeaderLines
	if availableHeight < 5 {
		availableHeight = 5 // Minimum viewport size
	}
	
	v.maxVisibleSuggestions = availableHeight
	v.visibleSuggestionCount = min(len(v.suggestions), v.maxVisibleSuggestions)
	
	// Ensure scroll offset is valid
	if v.suggestionScrollOffset < 0 {
		v.suggestionScrollOffset = 0
	}
	maxOffset := max(0, len(v.suggestions)-v.maxVisibleSuggestions)
	if v.suggestionScrollOffset > maxOffset {
		v.suggestionScrollOffset = maxOffset
	}
}

// renderSuggestionsViewport renders only the visible portion of suggestions with scroll indicators
func (v *NavigationView) renderSuggestionsViewport() string {
	if len(v.suggestions) == 0 {
		return v.subtitleStyle.Render("[Analyzing page for navigation options...]")
	}
	
	// Calculate visible range
	startIdx := v.suggestionScrollOffset
	endIdx := min(startIdx+v.maxVisibleSuggestions, len(v.suggestions))
	
	if startIdx >= len(v.suggestions) {
		startIdx = max(0, len(v.suggestions)-v.maxVisibleSuggestions)
		endIdx = len(v.suggestions)
		v.suggestionScrollOffset = startIdx
	}
	
	var lines []string
	
	// Add "more above" indicator if scrolled down
	if startIdx > 0 {
		moreAbove := fmt.Sprintf("▲ %d more above", startIdx)
		lines = append(lines, v.subtitleStyle.Render(moreAbove))
	}
	
	// Render visible suggestions using the same logic as original renderSuggestions
	visibleSuggestions := v.suggestions[startIdx:endIdx]
	
	// First pass: calculate the maximum width needed for alignment
	maxMainTextWidth := 0
	for _, suggestion := range visibleSuggestions {
		if suggestion.Type != SectionHeaderSuggestion {
			displayText := truncateText(suggestion.Text, 120)
			mainTextWidth := len(displayText) + 3
			if mainTextWidth > maxMainTextWidth {
				maxMainTextWidth = mainTextWidth
			}
		}
	}
	
	// Render each visible suggestion
	for i, suggestion := range visibleSuggestions {
		globalIndex := startIdx + i
		var line string
		var style lipgloss.Style
		
		// Handle section headers differently
		if suggestion.Type == SectionHeaderSuggestion {
			style = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("208")).
				PaddingTop(1).
				PaddingBottom(0)
			
			line = fmt.Sprintf("▶ %s", suggestion.Text)
		} else {
			displayText := truncateText(suggestion.Text, 120)
			
			if globalIndex == v.selectedIndex {
				style = v.selectedStyle
				line = fmt.Sprintf("┌─ %s", displayText)
			} else {
				style = v.suggestionStyle
				line = fmt.Sprintf("├─ %s", displayText)
			}
			
			// Calculate padding needed to align subtitles
			currentWidth := len(displayText) + 3
			paddingNeeded := maxMainTextWidth - currentWidth + 5
			
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
		}
		
		lines = append(lines, style.Render(line))
	}
	
	// Add "more below" indicator if there are items below viewport
	if endIdx < len(v.suggestions) {
		moreBelow := fmt.Sprintf("▼ %d more below", len(v.suggestions)-endIdx)
		lines = append(lines, v.subtitleStyle.Render(moreBelow))
	}
	
	return strings.Join(lines, "\n")
}

// Helper functions are available from input_modal.go

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

		case LinkSuggestion, ActionSuggestion, FormSuggestion, FormFieldSuggestion:
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
				
				// Wait a moment for navigation to start
				time.Sleep(200 * time.Millisecond)
				
				// Get the actual URL after navigation (handles redirects)
				actualURL, _, err := v.chromeDPManager.GetPageInfo()
				if err != nil {
					// Fallback to requested URL if we can't get the actual URL
					actualURL = element.URL
				}
				
				return NavigationCompleteMsg{
					URL:     actualURL,
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

		case "form_input":
			// Handle form input using the input modal
			return v.handleFormInput(element)()

		case "form_submit":
			// Submit the form directly
			if v.formHandler != nil {
				if err := v.formHandler.SubmitForm(); err != nil {
					return NavigationErrorMsg{Error: err}
				}
				return NavigationCompleteMsg{
					URL:     v.currentURL,
					Success: true,
				}
			}
			return NavigationErrorMsg{Error: fmt.Errorf("form handler not available")}

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
	case FormFieldElement:
		return FormFieldSuggestion
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

// countFormFields counts the number of form fields that will be added as elements
func countFormFields(form *LoginForm) int {
	count := 0
	if form.EmailField != nil {
		count++
	}
	if form.PasswordField != nil {
		count++
	}
	if form.UsernameField != nil {
		count++
	}
	if form.SubmitButton != nil && form.IsComplete {
		count++
	}
	return count
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
	// Convert NavigableElements to NavigationElements for LLM
	var llmElements []llm.NavigationElement
	var clickableElements []NavigableElement
	
	for _, elem := range v.pageElements {
		// Include all clickable elements, not just links
		if elem.Type == LinkElement || elem.Type == ButtonElement || elem.Type == ActionElement {
			clickableElements = append(clickableElements, elem)
			
			llmElements = append(llmElements, llm.NavigationElement{
				Text:        elem.Text,
				Selector:    elem.Selector,
				URL:         elem.URL,
				Type:        v.elementTypeToString(elem.Type),
				Description: elem.Description,
			})
		}
	}

	if len(llmElements) == 0 {
		// No clickable elements found, try URL navigation
		if strings.HasPrefix(target, "http") || strings.HasPrefix(target, "/") || strings.Contains(target, ".") {
			return v.navigateToURL(target)
		}
		return fmt.Errorf("no navigable elements found on page")
	}

	// Try LLM ranking if available
	if v.llmClient != nil {
		ctx := context.Background()
		ranking, err := v.llmClient.RankNavigationElements(ctx, target, llmElements)
		
		if err == nil && len(ranking.Elements) > 0 {
			// Try elements in ranked order using SmartClick
			for _, rankedElem := range ranking.Elements {
				if rankedElem.Confidence > 0.3 { // Only try elements with reasonable confidence
					// Find the corresponding NavigableElement
					var targetElement *NavigableElement
					for _, elem := range clickableElements {
						if elem.Text == rankedElem.Text && elem.Selector == rankedElem.Selector {
							targetElement = &elem
							break
						}
					}
					
					if targetElement != nil {
						// Try SmartClick with the LLM-recommended strategy
						success, err := v.trySmartClick(*targetElement, rankedElem)
						if success {
							elementText := truncateText(rankedElem.Text, 30)
							v.addHistory(fmt.Sprintf("→ Successfully navigated via \"%s\"", elementText))
							return nil
						}
						
						// Log attempt but continue to next option
						if err != nil {
							log.Printf("SmartClick failed for %s: %v", rankedElem.Text, err)
						}
					}
				}
			}
			
			// If we tried LLM ranking but nothing worked, show the top suggestions
			if len(ranking.Elements) > 0 {
				topSuggestions := []string{}
				for i, elem := range ranking.Elements {
					if i >= 3 { break } // Show top 3
					if elem.Confidence > 0.1 {
						topSuggestions = append(topSuggestions, fmt.Sprintf("\"%s\" (%.0f%% match)", 
							elem.Text, elem.Confidence*100))
					}
				}
				if len(topSuggestions) > 0 {
					return fmt.Errorf("could not navigate to \"%s\". Top matches were: %s", 
						target, strings.Join(topSuggestions, ", "))
				}
			}
		}
	}

	// Fallback to fuzzy matching if LLM unavailable or failed
	return v.fallbackNavigateToTarget(target, clickableElements)
}

// trySmartClick attempts to click an element using SmartClick with LLM-suggested strategy
func (v *NavigationView) trySmartClick(element NavigableElement, rankedElem llm.RankedNavigationElement) (bool, error) {
	if v.chromeDPManager == nil {
		return false, fmt.Errorf("Chrome not connected")
	}

	// For navigation links, prefer direct URL navigation if available
	if element.Type == LinkElement && element.URL != "" {
		if err := v.navigateToURL(element.URL); err == nil {
			return true, nil
		}
	}

	// Use SmartClick for all other cases or as fallback
	return v.chromeDPManager.SmartClick(element.Selector, element.Text)
}

// fallbackNavigateToTarget provides the original fuzzy matching logic
func (v *NavigationView) fallbackNavigateToTarget(target string, elements []NavigableElement) error {
	var potentialMatches []NavigableElement

	for _, elem := range elements {
		score := v.fuzzyMatch(target, elem.Text)
		if score > 0.3 {
			potentialMatches = append(potentialMatches, elem)
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

		// Try SmartClick on the best match
		success, err := v.chromeDPManager.SmartClick(bestMatch.Selector, bestMatch.Text)
		if success {
			elementText := truncateText(bestMatch.Text, 30)
			v.addHistory(fmt.Sprintf("→ Successfully navigated via \"%s\"", elementText))
			return nil
		}
		
		if err != nil {
			return fmt.Errorf("failed to click \"%s\": %w", bestMatch.Text, err)
		}
	}

	// Only try URL navigation if it looks like a URL or path
	if strings.HasPrefix(target, "http") || strings.HasPrefix(target, "/") || strings.Contains(target, ".") {
		return v.navigateToURL(target)
	}

	return fmt.Errorf("no navigation element found matching: %s", target)
}

// elementTypeToString converts ElementType to string for LLM
func (v *NavigationView) elementTypeToString(elemType ElementType) string {
	switch elemType {
	case LinkElement:
		return "link"
	case ButtonElement:
		return "button" 
	case FormElement:
		return "form"
	case ActionElement:
		return "action"
	case FormFieldElement:
		return "input"
	default:
		return "unknown"
	}
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


// Additional message types
type NavigationErrorMsg struct {
	Error error
}

type FormDetectedMsg struct {
	Form *LoginForm
}

type FormFieldReadyMsg struct {
	Field *FormField
}

type AuthenticationCompleteMsg struct {
	Success bool
	User    *config.TestUser
	Error   error
}

type FormInputModalReadyMsg struct {
	Field *FormField
}

// addFormFieldsToElements adds form field actions to the elements list
func (v *NavigationView) addFormFieldsToElements(elements *[]NavigableElement, form *LoginForm) {
	if form == nil {
		return
	}

	// Add email field action
	if form.EmailField != nil {
		*elements = append(*elements, NavigableElement{
			Type:        FormFieldElement,
			Text:        fmt.Sprintf("Enter %s", form.EmailField.Label),
			Description: fmt.Sprintf("Fill in the %s field", form.EmailField.Label),
			Selector:    form.EmailField.Selector,
			Method:      "form_input",
		})
	}

	// Add password field action
	if form.PasswordField != nil {
		*elements = append(*elements, NavigableElement{
			Type:        FormFieldElement,
			Text:        fmt.Sprintf("Enter %s", form.PasswordField.Label),
			Description: fmt.Sprintf("Fill in the %s field", form.PasswordField.Label),
			Selector:    form.PasswordField.Selector,
			Method:      "form_input",
		})
	}

	// Add username field action
	if form.UsernameField != nil {
		*elements = append(*elements, NavigableElement{
			Type:        FormFieldElement,
			Text:        fmt.Sprintf("Enter %s", form.UsernameField.Label),
			Description: fmt.Sprintf("Fill in the %s field", form.UsernameField.Label),
			Selector:    form.UsernameField.Selector,
			Method:      "form_input",
		})
	}

	// Add submit form action
	if form.SubmitButton != nil && form.IsComplete {
		*elements = append(*elements, NavigableElement{
			Type:        FormElement,
			Text:        "Submit Form",
			Description: "Submit the login form",
			Selector:    form.SubmitButton.Selector,
			Method:      "form_submit",
		})
	}
}

// handleModalComplete handles completion of the input modal
func (v *NavigationView) handleModalComplete() (tea.Model, tea.Cmd) {
	if v.inputModal == nil || v.pendingField == nil {
		return v, nil
	}

	result := v.inputModal.GetResult()
	
	if result.Cancelled {
		v.awaitingInput = false
		v.pendingField = nil
		return v, nil
	}

	// Fill the form field
	return v, v.fillFormField(result)
}

// fillFormField fills a form field with the provided value
func (v *NavigationView) fillFormField(result InputResult) tea.Cmd {
	return func() tea.Msg {
		if v.formHandler == nil || v.pendingField == nil {
			return NavigationErrorMsg{Error: fmt.Errorf("form handler or field not available")}
		}

		// Fill the field
		if err := v.formHandler.FillField(v.pendingField, result.Value); err != nil {
			return NavigationErrorMsg{Error: fmt.Errorf("failed to fill field: %w", err)}
		}

		// If user selected a saved user, save their last used time
		if result.SelectedUser != nil && v.authConfig != nil {
			domain := v.formHandler.GetDomain()
			v.authConfig.UpdateUserLastUsed(domain, result.SelectedUser.Email)
		}

		// If this was a new entry, potentially save it
		if result.NewEntry && v.pendingField.Type == EmailField {
			// TODO: Implement saving new user after collecting all required fields
		}

		// Reset modal state
		v.awaitingInput = false
		v.pendingField = nil

		// Check if we can submit the form now
		if v.canSubmitForm() {
			return v.submitForm()()
		}

		// Continue with form filling
		v.addHistory(fmt.Sprintf("→ Filled field: %s", v.pendingField.Label))
		return NavigationCompleteMsg{
			URL:     v.currentURL,
			Success: true,
		}
	}
}

// canSubmitForm checks if all required form fields are filled
func (v *NavigationView) canSubmitForm() bool {
	if v.currentForm == nil {
		return false
	}

	// For magic link forms (email only), can submit after email is filled
	if v.currentForm.IsMagicLink {
		return v.currentForm.EmailField != nil && v.currentForm.EmailField.Value != ""
	}

	// For regular forms, need both email/username and password
	hasIdentifier := (v.currentForm.EmailField != nil && v.currentForm.EmailField.Value != "") ||
					  (v.currentForm.UsernameField != nil && v.currentForm.UsernameField.Value != "")
	hasPassword := v.currentForm.PasswordField != nil && v.currentForm.PasswordField.Value != ""

	return hasIdentifier && hasPassword
}

// submitForm submits the current form
func (v *NavigationView) submitForm() tea.Cmd {
	return func() tea.Msg {
		if v.formHandler == nil {
			return NavigationErrorMsg{Error: fmt.Errorf("form handler not available")}
		}

		v.addHistory("→ Submitting form...")
		
		// Submit the form
		if err := v.formHandler.SubmitForm(); err != nil {
			return NavigationErrorMsg{Error: fmt.Errorf("failed to submit form: %w", err)}
		}

		// Wait for page changes
		result, err := v.formHandler.WaitForPageChange(10 * time.Second)
		if err != nil {
			return NavigationErrorMsg{Error: fmt.Errorf("failed to wait for page change: %w", err)}
		}

		// Handle the result
		if result.MagicLinkSent {
			v.addHistory("📧 " + result.Message)
			// Trigger email checking for magic link
			return v.handleMagicLinkSent()()
		}

		if result.NavigationOccurred {
			v.addHistory("→ " + result.Message)
			return NavigationCompleteMsg{
				URL:     result.FinalURL,
				Success: true,
			}
		}

		if result.ErrorDetected {
			v.addHistory("❌ " + result.Message)
			return NavigationErrorMsg{Error: fmt.Errorf(result.Message)}
		}

		return NavigationCompleteMsg{
			URL:     result.FinalURL,
			Success: true,
		}
	}
}

// handleFormInput handles form input by showing the input modal
func (v *NavigationView) handleFormInput(element NavigableElement) tea.Cmd {
	return func() tea.Msg {
		if v.currentForm == nil {
			return NavigationErrorMsg{Error: fmt.Errorf("no form detected")}
		}

		// Find the corresponding form field
		var field *FormField
		if v.currentForm.EmailField != nil && v.currentForm.EmailField.Selector == element.Selector {
			field = v.currentForm.EmailField
		} else if v.currentForm.PasswordField != nil && v.currentForm.PasswordField.Selector == element.Selector {
			field = v.currentForm.PasswordField
		} else if v.currentForm.UsernameField != nil && v.currentForm.UsernameField.Selector == element.Selector {
			field = v.currentForm.UsernameField
		}

		if field == nil {
			return NavigationErrorMsg{Error: fmt.Errorf("form field not found")}
		}

		// Get saved users for this domain
		var savedUsers []config.TestUser
		if v.authConfig != nil {
			domain := v.formHandler.GetDomain()
			savedUsers, _ = v.authConfig.GetRecentUsersForDomain(domain, 5)
		}

		// Create and show input modal
		v.inputModal = NewInputModal(field.Type, field.Label, field.Placeholder, v.formHandler.GetDomain(), savedUsers)
		v.inputModal.Show()
		v.awaitingInput = true
		v.pendingField = field

		v.addHistory(fmt.Sprintf("→ Opening input for: %s", field.Label))
		
		return FormInputModalReadyMsg{
			Field: field,
		}
	}
}

// handleMagicLinkSent handles magic link detection and email checking
func (v *NavigationView) handleMagicLinkSent() tea.Cmd {
	return func() tea.Msg {
		// Check if we have email checking capability
		if v.authFlow == nil {
			v.addHistory("⚠️ Email checking not configured")
			return NavigationCompleteMsg{
				URL:     v.currentURL,
				Success: true,
			}
		}

		// Get the current user's email from the form
		var userEmail string
		if v.currentForm != nil && v.currentForm.EmailField != nil {
			userEmail = v.currentForm.EmailField.Value
		}

		if userEmail == "" {
			v.addHistory("⚠️ No email address found for magic link checking")
			return NavigationCompleteMsg{
				URL:     v.currentURL,
				Success: true,
			}
		}

		v.addHistory("📧 Checking email for magic link...")

		// Create a test user for the magic link authentication
		testUser := &config.TestUser{
			Name:     "Magic Link User",
			Email:    userEmail,
			AuthType: "magic_link",
			AuthConfig: &config.TestUserAuthConfig{
				EmailCheckEnabled: true,
				EmailTimeout:      30,
			},
		}

		// Use the auth flow manager to authenticate with email support
		authResult := v.authFlow.AuthenticateWithEmailSupport(testUser)

		if authResult.Success && authResult.RedirectURL != "" {
			v.addHistory(fmt.Sprintf("✅ Found magic link: %s", authResult.RedirectURL[:50]+"..."))
			
			// Navigate to the magic link
			if err := v.chromeDPManager.Navigate(authResult.RedirectURL); err != nil {
				v.addHistory(fmt.Sprintf("❌ Failed to navigate to magic link: %v", err))
				return NavigationErrorMsg{Error: err}
			}

			// Save this user for future use
			if v.authConfig != nil {
				domain := v.formHandler.GetDomain()
				v.authConfig.SaveMagicLinkUserForDomain(domain, userEmail, testUser.Name)
			}

			v.addHistory("🎉 Successfully authenticated with magic link")
			return NavigationCompleteMsg{
				URL:     authResult.RedirectURL,
				Success: true,
			}
		} else {
			v.addHistory(fmt.Sprintf("❌ Magic link not found: %s", authResult.Message))
			return NavigationErrorMsg{Error: authResult.Error}
		}
	}
}
