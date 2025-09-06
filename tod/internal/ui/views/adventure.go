package views

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ciciliostudio/tod/internal/browser"
	"github.com/ciciliostudio/tod/internal/config"
	"github.com/ciciliostudio/tod/internal/llm"
	"github.com/ciciliostudio/tod/internal/types"
)

// AdventureView handles the text adventure interface for navigating web pages
type AdventureView struct {
	config *config.Config
	width  int
	height int

	// Current state
	currentPage PageState
	actions     []types.CodeAction

	// Action list state
	actionListState ActionListState

	// User input
	textInput   textinput.Model
	suggestions []string
	showHelp    bool

	// History display for scrollable conversation (read-only, selectable)
	historyView textarea.Model
	history     []ConversationMessage
	
	// Scroll state tracking
	userScrolledUp bool

	// Session tracking
	sessionID string

	// Feedback
	lastResult string

	// AI capabilities
	llmClient llm.Client

	// Browser automation
	browserClient *browser.Client

	// Styles
	styles *AdventureStyles
}

// PageState represents the current state of the page being tested
type PageState struct {
	URL         string
	Title       string
	Description string
	Elements    []PageElement
}

// PageElement represents an interactive element on the page
type PageElement struct {
	Type        string // button, input, link, etc.
	Label       string
	Selector    string
	Description string
}

// MessageType represents the type of conversation message
type MessageType int

const (
	MessageTypeUser MessageType = iota
	MessageTypeSystem
	MessageTypeSuccess
	MessageTypeError
)

// ConversationMessage represents a single message in the conversation history
type ConversationMessage struct {
	Type      MessageType
	Content   string
	Timestamp time.Time
	PageURL   string
}

// ActionHistory tracks actions taken during the session (kept for compatibility)
type ActionHistory struct {
	Action     string
	Timestamp  string
	Result     string
	PageBefore string
	PageAfter  string
}

// ActionListState manages the filterable action list
type ActionListState struct {
	AllActions      []browser.PageAction `json:"all_actions"`      // All discovered page actions
	FilteredActions []browser.PageAction `json:"filtered_actions"` // Currently filtered actions
	SelectedIndex   int                  `json:"selected_index"`   // Currently selected action index
	FilterQuery     string               `json:"filter_query"`     // Current filter text
	RecentActions   []string             `json:"recent_actions"`   // Recently used actions for smart ranking
	ShowActionList  bool                 `json:"show_action_list"` // Whether to show the action list
	MaxDisplayed    int                  `json:"max_displayed"`    // Maximum actions to display
}

// AdventureStyles contains styling for the adventure interface
type AdventureStyles struct {
	Header         lipgloss.Style
	PageInfo       lipgloss.Style
	ActionBox      lipgloss.Style
	InputPrompt    lipgloss.Style
	Suggestions    lipgloss.Style
	Help           lipgloss.Style
	Error          lipgloss.Style
	Success        lipgloss.Style
	UserMessage    lipgloss.Style
	SystemMessage  lipgloss.Style
	SuccessMessage lipgloss.Style
	ErrorMessage   lipgloss.Style
	ActionList     lipgloss.Style
	ActionItem     lipgloss.Style
	ActionSelected lipgloss.Style
	ActionCategory lipgloss.Style
}

// NewAdventureView creates a new adventure view
func NewAdventureView(cfg *config.Config) *AdventureView {
	return NewAdventureViewWithLLM(cfg, nil)
}

// NewAdventureViewWithLLM creates a new adventure view with LLM client
func NewAdventureViewWithLLM(cfg *config.Config, llmClient llm.Client) *AdventureView {
	// Initialize text input
	ti := textinput.New()
	ti.Placeholder = "Enter your command, brave tester... (type or use ↑/↓)"
	ti.Focus()
	ti.CharLimit = 200
	ti.Width = 60

	// If no LLM client provided, create a default one
	if llmClient == nil && cfg != nil {
		provider := llm.Provider(cfg.AI.Provider)
		client, err := llm.NewClient(provider, cfg.AI.APIKey, map[string]interface{}{
			"model": cfg.AI.Model,
		})
		if err == nil {
			llmClient = client
		}
	}

	// Initialize textarea for conversation history (read-only, selectable)
	historyView := textarea.New()
	historyView.SetValue("Welcome to Tod Adventure Mode!\nType commands to interact with your application.")
	historyView.Focus()
	historyView.Blur() // Start unfocused so input field has focus
	historyView.ShowLineNumbers = false
	historyView.SetWidth(80)
	historyView.SetHeight(20)
	historyView.CharLimit = 0 // No character limit
	
	// Make it read-only by disabling most keybindings
	historyView.KeyMap.InsertNewline.SetEnabled(false)
	historyView.KeyMap.DeleteAfterCursor.SetEnabled(false)
	historyView.KeyMap.DeleteBeforeCursor.SetEnabled(false)
	historyView.KeyMap.DeleteCharacterBackward.SetEnabled(false)
	historyView.KeyMap.DeleteCharacterForward.SetEnabled(false)
	historyView.KeyMap.DeleteWordBackward.SetEnabled(false)
	historyView.KeyMap.DeleteWordForward.SetEnabled(false)
	historyView.KeyMap.InsertNewline.SetEnabled(false)

	return &AdventureView{
		config:         cfg,
		textInput:      ti,
		historyView:    historyView,
		history:        []ConversationMessage{},
		userScrolledUp: false,
		llmClient: llmClient,
		styles:    NewAdventureStyles(),
		currentPage: PageState{
			URL:         "Not connected",
			Title:       "Tod Adventure",
			Description: "Ready to begin your testing journey",
		},
		actionListState: ActionListState{
			AllActions:      []browser.PageAction{},
			FilteredActions: []browser.PageAction{},
			SelectedIndex:   0,
			FilterQuery:     "",
			RecentActions:   []string{},
			ShowActionList:  true,
			MaxDisplayed:    8, // Show max 8 actions at a time
		},
	}
}

// NewAdventureStyles creates the styling for adventure mode
func NewAdventureStyles() *AdventureStyles {
	return &AdventureStyles{
		Header: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1).
			MarginBottom(1),

		PageInfo: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#874BFD")).
			MarginBottom(0),  // Reduce spacing for compact layout

		ActionBox: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#04B575")).
			Padding(1).
			MarginBottom(1),

		InputPrompt: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#04B575")).
			Bold(true).
			MarginTop(1),  // Small gap from history

		Suggestions: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262")).
			Italic(true),

		Help: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262")).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#626262")).
			Padding(0, 1).
			MarginTop(1),

		Error: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF5F87")).
			Bold(true),

		Success: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#04B575")).
			Bold(true),

		UserMessage: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")).
			Bold(true),

		SystemMessage: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#04B575")),

		SuccessMessage: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#04B575")),

		ErrorMessage: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF5F87")),

		ActionList: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#626262")).
			Padding(0, 1).
			MarginTop(1),

		ActionItem: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")).
			PaddingLeft(1),

		ActionSelected: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Bold(true).
			PaddingLeft(1),

		ActionCategory: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#04B575")).
			Bold(true),
	}
}

// renderHistory creates the formatted conversation history content (plain text for textarea)
func (av *AdventureView) renderHistory() string {
	if len(av.history) == 0 {
		return "Welcome to Tod Adventure Mode!\nType commands to interact with your application."
	}

	var content strings.Builder

	for _, msg := range av.history {
		var prefix string

		switch msg.Type {
		case MessageTypeUser:
			prefix = "> "
		case MessageTypeSystem:
			prefix = "  "
		case MessageTypeSuccess:
			prefix = ">> "
		case MessageTypeError:
			prefix = "!! "
		}

		// Use plain text without lipgloss styling for textarea display
		content.WriteString(fmt.Sprintf("%s%s\n", prefix, msg.Content))

		// Add some spacing between user commands and responses
		if msg.Type == MessageTypeUser {
			content.WriteString("\n")
		}
	}

	return content.String()
}

// addMessage adds a new message to the conversation history
func (av *AdventureView) addMessage(msgType MessageType, content string) {
	message := ConversationMessage{
		Type:      msgType,
		Content:   content,
		Timestamp: time.Now(),
		PageURL:   av.currentPage.URL,
	}

	av.history = append(av.history, message)

	// Update textarea content
	historyContent := av.renderHistory()
	av.historyView.SetValue(historyContent)
	
	// Smart scroll: only auto-scroll if user hasn't manually scrolled up
	if !av.userScrolledUp {
		av.historyView.CursorEnd()
	}
}

// Init implements tea.Model
func (av *AdventureView) Init() tea.Cmd {
	return textinput.Blink
}

// Update implements tea.Model
func (av *AdventureView) Update(msg tea.Msg) (*AdventureView, tea.Cmd) {
	var cmd tea.Cmd
	var historyCmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		av.width = msg.Width
		av.height = msg.Height
		av.textInput.Width = av.width - 4
		
		// Update history view dimensions
		av.updateHistoryViewDimensions()

	case tea.KeyMsg:
		// Check if we should handle history view scrolling
		if av.shouldHandleHistoryScrolling(msg) {
			return av.handleHistoryScrolling(msg)
		}
		
		switch msg.String() {
		case "ctrl+c", "esc":
			return av, tea.Quit
		case "enter":
			return av.handleUserInputOrActionSelection()
		case "ctrl+h":
			av.showHelp = !av.showHelp
			return av, nil
		case "ctrl+a":
			// Select all text in history
			av.historyView.CursorStart()
			av.historyView.SetCursor(len(av.historyView.Value()))
			return av, nil
		case "ctrl+shift+c":
			// Copy entire history as markdown
			historyText := av.exportHistoryAsMarkdown()
			clipboard.WriteAll(historyText)
			av.addMessage(MessageTypeSystem, "History copied to clipboard")
			return av, nil
		case "up", "down":
			return av.handleArrowKeys(msg.String() == "up")
		case "1", "2", "3", "4", "5", "6", "7", "8", "9":
			return av.handleNumberSelection(msg.String())
		case "tab":
			return av.handleTabNavigation()
		}
	}

	// Update history view (for scrolling and text selection)
	av.historyView, historyCmd = av.historyView.Update(msg)

	// Update text input
	av.textInput, cmd = av.textInput.Update(msg)

	// Clear last result when user starts typing new input
	if len(av.textInput.Value()) > 0 && av.lastResult != "" {
		av.lastResult = ""
	}

	// Update action filter and suggestions based on current input
	av.updateActionFilter()
	av.updateSuggestions()

	return av, tea.Batch(cmd, historyCmd)
}

// View implements tea.Model
func (av *AdventureView) View() string {
	// Update history view dimensions based on current screen size
	av.updateHistoryViewDimensions()
	
	// Fixed header at top
	header := av.styles.Header.Render(fmt.Sprintf("[*] Tod Adventure Mode - %s", av.config.Current))

	// Dynamic history section (selectable textarea)
	historySection := av.historyView.View()

	// Build bottom sections dynamically
	var bottomSections []string

	// Current page info (compact)
	pageInfo := av.renderCompactPageInfo()
	bottomSections = append(bottomSections, pageInfo)

	// Input prompt
	inputSection := av.renderInputSection()
	bottomSections = append(bottomSections, inputSection)

	// Action list, help, or suggestions
	if av.showHelp {
		help := av.renderHelp()
		bottomSections = append(bottomSections, help)
	} else {
		// Show action list if available, otherwise show suggestions
		actionList := av.renderActionList()
		if actionList != "" {
			bottomSections = append(bottomSections, actionList)
		} else if len(av.suggestions) > 0 {
			suggestions := av.renderSuggestions()
			bottomSections = append(bottomSections, suggestions)
		}
	}

	bottom := strings.Join(bottomSections, "\n")

	// Combine all sections with dynamic spacing
	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		historySection,
		bottom,
	)
}

func (av *AdventureView) renderCompactPageInfo() string {
	if av.currentPage.URL == "" || av.currentPage.URL == "Not connected" {
		return av.styles.PageInfo.Render("[>] Not connected to any page")
	}
	
	content := fmt.Sprintf("[>] %s", av.currentPage.Title)
	if av.currentPage.Title == "" {
		content = fmt.Sprintf("[>] %s", av.currentPage.URL)
	}
	
	return av.styles.PageInfo.Render(content)
}

func (av *AdventureView) renderPageInfo() string {
	content := fmt.Sprintf("[>] You are at: %s\n", av.currentPage.URL)

	if av.currentPage.Title != "" {
		content += fmt.Sprintf("[Page] %s\n", av.currentPage.Title)
	}

	if av.currentPage.Description != "" {
		content += fmt.Sprintf("* %s", av.currentPage.Description)
	}

	return av.styles.PageInfo.Render(content)
}

func (av *AdventureView) renderAvailableActions() string {
	content := ">> Available Quests <<\n\n"

	for i, action := range av.actions {
		if i >= 10 { // Show max 10 actions
			content += fmt.Sprintf("   ... and %d more (type to filter)\n", len(av.actions)-10)
			break
		}

		content += fmt.Sprintf("  [%d] %s - %s\n", i+1, action.Name, action.Description)
	}

	return av.styles.ActionBox.Render(content)
}

func (av *AdventureView) renderInputSection() string {
	prompt := av.styles.InputPrompt.Render("Your move> ")
	input := av.textInput.View()

	return prompt + input
}

func (av *AdventureView) renderSuggestions() string {
	if len(av.suggestions) == 0 {
		return ""
	}

	content := "* Hints: "
	for i, suggestion := range av.suggestions {
		if i > 0 {
			content += " • "
		}
		content += suggestion
		if i >= 2 { // Show max 3 suggestions
			break
		}
	}

	return av.styles.Suggestions.Render(content)
}

func (av *AdventureView) renderFeedback() string {
	return av.styles.Success.Render(">> " + av.lastResult)
}

func (av *AdventureView) renderHelp() string {
	help := `[*] Tod Adventure Help

Navigation Commands:
  • Type naturally: "click login button", "fill email field"
  • Use numbers: "1" to select first action
  • Navigation: "go to /dashboard", "visit home page"

Input Commands:  
  • "fill <field> with <value>" - Fill form fields
  • "click <element>" - Click buttons, links
  • "select <option>" - Select from dropdowns

Special Commands:
  • "help" - Show/hide this help (or press Ctrl+H)
  • "actions" - List all available actions
  • "history" - Show session history
  • "back" - Go back to previous page
  • "refresh" - Reload current page

Keyboard Shortcuts:
  • Ctrl+A - Select all history text
  • Ctrl+Shift+C - Copy entire history as markdown
  • PgUp/PgDn - Scroll through history
  • Home/End - Jump to top/bottom of history

Tips:
  • Use TAB for autocomplete
  • Press ↑/↓ to cycle through suggestions or scroll history
  • Type part of an action name to filter
  • Mouse selection works in history area
  • All actions are recorded for test generation`

	return av.styles.Help.Render(help)
}

// renderActionList renders the filterable action list
func (av *AdventureView) renderActionList() string {
	if !av.actionListState.ShowActionList || len(av.actionListState.FilteredActions) == 0 {
		return ""
	}

	var content strings.Builder
	
	// Header
	filterInfo := ""
	if av.actionListState.FilterQuery != "" {
		filterInfo = fmt.Sprintf(" (filtered by \"%s\")", av.actionListState.FilterQuery)
	}
	content.WriteString(fmt.Sprintf("Available Actions (%d found)%s:\n\n", 
		len(av.actionListState.FilteredActions), filterInfo))

	// Group actions by category
	actionsByCategory := make(map[string][]browser.PageAction)
	categoryOrder := []string{"Authentication", "Navigation", "Form", "Interactive"}
	
	for _, action := range av.actionListState.FilteredActions {
		category := action.Category
		if _, exists := actionsByCategory[category]; !exists {
			actionsByCategory[category] = []browser.PageAction{}
		}
		actionsByCategory[category] = append(actionsByCategory[category], action)
	}

	displayedCount := 0
	maxDisplayed := av.actionListState.MaxDisplayed

	// Render actions by category
	for _, category := range categoryOrder {
		actions, exists := actionsByCategory[category]
		if !exists || len(actions) == 0 {
			continue
		}

		// Category header
		content.WriteString(av.styles.ActionCategory.Render(fmt.Sprintf("%s:\n", category)))

		for _, action := range actions {
			if displayedCount >= maxDisplayed {
				remaining := len(av.actionListState.FilteredActions) - displayedCount
				content.WriteString(av.styles.ActionItem.Render(fmt.Sprintf("   ... and %d more (type to filter)\n", remaining)))
				break
			}

			actionIndex := av.findActionIndex(action)
			isSelected := actionIndex == av.actionListState.SelectedIndex

			// Format action line
			var line string
			if displayedCount < 9 { // Show numbers 1-9 for quick selection
				line = fmt.Sprintf("  [%d] %s", displayedCount+1, action.Description)
			} else {
				line = fmt.Sprintf("  %s", action.Description)
			}

			// Apply styling based on selection
			if isSelected {
				content.WriteString(av.styles.ActionSelected.Render(line) + "\n")
			} else {
				content.WriteString(av.styles.ActionItem.Render(line) + "\n")
			}

			displayedCount++
			if displayedCount >= maxDisplayed {
				break
			}
		}

		content.WriteString("\n")
		if displayedCount >= maxDisplayed {
			break
		}
	}

	return av.styles.ActionList.Render(content.String())
}

// findActionIndex finds the index of an action in the filtered list
func (av *AdventureView) findActionIndex(target browser.PageAction) int {
	for i, action := range av.actionListState.FilteredActions {
		if action.ID == target.ID {
			return i
		}
	}
	return -1
}

// filterActions filters the actions based on the current query
func (av *AdventureView) filterActions() {
	query := strings.ToLower(av.actionListState.FilterQuery)
	
	if query == "" {
		// No filter - show all actions, sorted by priority
		av.actionListState.FilteredActions = make([]browser.PageAction, len(av.actionListState.AllActions))
		copy(av.actionListState.FilteredActions, av.actionListState.AllActions)
	} else {
		// Filter actions using fuzzy matching
		av.actionListState.FilteredActions = []browser.PageAction{}
		
		for _, action := range av.actionListState.AllActions {
			if av.matchesFilter(action, query) {
				av.actionListState.FilteredActions = append(av.actionListState.FilteredActions, action)
			}
		}

		// Sort filtered results by relevance
		av.sortActionsByRelevance(query)
	}

	// Reset selection if out of bounds
	if av.actionListState.SelectedIndex >= len(av.actionListState.FilteredActions) {
		av.actionListState.SelectedIndex = 0
	}
}

// matchesFilter checks if an action matches the filter query
func (av *AdventureView) matchesFilter(action browser.PageAction, query string) bool {
	// Check description (main match)
	if strings.Contains(strings.ToLower(action.Description), query) {
		return true
	}

	// Check text content
	if strings.Contains(strings.ToLower(action.Text), query) {
		return true
	}

	// Check category
	if strings.Contains(strings.ToLower(action.Category), query) {
		return true
	}

	// Check type
	if strings.Contains(strings.ToLower(action.Type), query) {
		return true
	}

	// Fuzzy matching on description (simple version)
	return av.fuzzyMatch(strings.ToLower(action.Description), query)
}

// fuzzyMatch performs simple fuzzy matching
func (av *AdventureView) fuzzyMatch(text, query string) bool {
	if len(query) == 0 {
		return true
	}

	textPos := 0
	for _, char := range query {
		found := false
		for textPos < len(text) {
			if rune(text[textPos]) == char {
				found = true
				textPos++
				break
			}
			textPos++
		}
		if !found {
			return false
		}
	}
	return true
}

// sortActionsByRelevance sorts actions by relevance to the query
func (av *AdventureView) sortActionsByRelevance(query string) {
	sort.SliceStable(av.actionListState.FilteredActions, func(i, j int) bool {
		actionA := av.actionListState.FilteredActions[i]
		actionB := av.actionListState.FilteredActions[j]

		scoreA := av.calculateRelevanceScore(actionA, query)
		scoreB := av.calculateRelevanceScore(actionB, query)

		return scoreA > scoreB
	})
}

// calculateRelevanceScore calculates how relevant an action is to the query
func (av *AdventureView) calculateRelevanceScore(action browser.PageAction, query string) int {
	score := action.Priority // Start with base priority

	desc := strings.ToLower(action.Description)
	text := strings.ToLower(action.Text)

	// Exact matches get high score
	if strings.Contains(desc, query) {
		score += 50
	}
	if strings.Contains(text, query) {
		score += 30
	}

	// Prefix matches get medium score
	if strings.HasPrefix(desc, query) {
		score += 25
	}

	// Recent actions get bonus
	for _, recent := range av.actionListState.RecentActions {
		if action.ID == recent {
			score += 20
			break
		}
	}

	return score
}

// updateActionFilter updates the filter based on current input
func (av *AdventureView) updateActionFilter() {
	av.actionListState.FilterQuery = av.textInput.Value()
	av.filterActions()
}

// discoverPageActions discovers actions on the current page
func (av *AdventureView) discoverPageActions() {
	if av.browserClient == nil {
		av.actionListState.AllActions = []browser.PageAction{}
		av.actionListState.FilteredActions = []browser.PageAction{}
		return
	}

	actions, err := av.browserClient.GetPageActions()
	if err != nil {
		// Log error for debugging but don't fail completely
		av.addMessage(MessageTypeSystem, fmt.Sprintf("Action discovery failed: %s", err.Error()))
		av.actionListState.AllActions = []browser.PageAction{}
		av.actionListState.FilteredActions = []browser.PageAction{}
		return
	}

	// Log action discovery results for debugging
	if len(actions) > 0 {
		av.addMessage(MessageTypeSystem, fmt.Sprintf("Discovered %d actions on page", len(actions)))
	}

	av.actionListState.AllActions = actions
	av.filterActions() // Apply current filter to new actions
}

// handleUserInputOrActionSelection handles both user input and action selection
func (av *AdventureView) handleUserInputOrActionSelection() (*AdventureView, tea.Cmd) {
	input := strings.TrimSpace(av.textInput.Value())
	
	// If there's a selected action and no input, execute the selected action
	if input == "" && len(av.actionListState.FilteredActions) > 0 && av.actionListState.SelectedIndex >= 0 {
		if av.actionListState.SelectedIndex < len(av.actionListState.FilteredActions) {
			selectedAction := av.actionListState.FilteredActions[av.actionListState.SelectedIndex]
			return av.executePageAction(selectedAction)
		}
		return av, nil
	}

	// If no input, do nothing
	if input == "" {
		return av, nil
	}

	// Check if input is a number for quick selection
	if num, err := strconv.Atoi(input); err == nil && num >= 1 && num <= 9 {
		if num-1 < len(av.actionListState.FilteredActions) {
			selectedAction := av.actionListState.FilteredActions[num-1]
			av.textInput.SetValue("")
			return av.executePageAction(selectedAction)
		}
	}

	// Add user message to history
	av.addMessage(MessageTypeUser, input)

	// Clear the input
	av.textInput.SetValue("")

	// Process the command as before
	result := av.processCommand(input)
	
	// Determine message type based on result content
	msgType := MessageTypeSystem
	if strings.Contains(result, "Successfully") || strings.Contains(result, "Success") {
		msgType = MessageTypeSuccess
	} else if strings.Contains(result, "Failed") || strings.Contains(result, "Error") || strings.Contains(result, "failed") {
		msgType = MessageTypeError
	}

	// Add system response to history
	av.addMessage(msgType, result)

	// Clear old lastResult (keeping for compatibility)
	av.lastResult = ""

	// Discover new actions after executing command
	av.discoverPageActions()

	return av, nil
}

// handleArrowKeys handles up/down arrow navigation
func (av *AdventureView) handleArrowKeys(up bool) (*AdventureView, tea.Cmd) {
	input := av.textInput.Value()
	
	// If no input and no actions, scroll history view
	if input == "" && len(av.actionListState.FilteredActions) == 0 {
		// Create the appropriate key message
		var keyMsg tea.KeyMsg
		if up {
			keyMsg = tea.KeyMsg{Type: tea.KeyUp}
			av.userScrolledUp = true
		} else {
			keyMsg = tea.KeyMsg{Type: tea.KeyDown}
			// Check if we're scrolling to bottom (approximate)
			totalLines := len(strings.Split(av.historyView.Value(), "\n"))
			if totalLines <= av.historyView.Height() {
				av.userScrolledUp = false
			}
		}
		
		var historyCmd tea.Cmd
		av.historyView, historyCmd = av.historyView.Update(keyMsg)
		return av, historyCmd
	}

	// Navigate through filtered actions
	if len(av.actionListState.FilteredActions) > 0 {
		if up {
			av.actionListState.SelectedIndex--
			if av.actionListState.SelectedIndex < 0 {
				av.actionListState.SelectedIndex = len(av.actionListState.FilteredActions) - 1
			}
		} else {
			av.actionListState.SelectedIndex++
			if av.actionListState.SelectedIndex >= len(av.actionListState.FilteredActions) {
				av.actionListState.SelectedIndex = 0
			}
		}
	}

	return av, nil
}

// handleNumberSelection handles number key quick selection
func (av *AdventureView) handleNumberSelection(numStr string) (*AdventureView, tea.Cmd) {
	num, err := strconv.Atoi(numStr)
	if err != nil || num < 1 || num > 9 {
		return av, nil
	}

	// Select action by number if available
	if num-1 < len(av.actionListState.FilteredActions) {
		selectedAction := av.actionListState.FilteredActions[num-1]
		return av.executePageAction(selectedAction)
	}

	// Otherwise, let the input handle it normally
	av.textInput.SetValue(av.textInput.Value() + numStr)
	av.updateActionFilter()
	return av, nil
}

// handleTabNavigation handles tab key for cycling through actions
func (av *AdventureView) handleTabNavigation() (*AdventureView, tea.Cmd) {
	if len(av.actionListState.FilteredActions) > 0 {
		av.actionListState.SelectedIndex++
		if av.actionListState.SelectedIndex >= len(av.actionListState.FilteredActions) {
			av.actionListState.SelectedIndex = 0
		}
	}
	return av, nil
}

// executePageAction executes a page action using the browser client
func (av *AdventureView) executePageAction(action browser.PageAction) (*AdventureView, tea.Cmd) {
	if av.browserClient == nil {
		av.addMessage(MessageTypeError, "Browser client not connected")
		return av, nil
	}

	// Add action to recent actions
	av.actionListState.RecentActions = append([]string{action.ID}, av.actionListState.RecentActions...)
	if len(av.actionListState.RecentActions) > 10 {
		av.actionListState.RecentActions = av.actionListState.RecentActions[:10]
	}

	// Add user message showing the action taken
	av.addMessage(MessageTypeUser, fmt.Sprintf("Execute: %s", action.Description))

	var result string
	var msgType MessageType

	// Execute the action based on its type
	switch action.Type {
	case "link":
		if href, exists := action.Attributes["href"]; exists {
			pageInfo, err := av.browserClient.NavigateToURL(href)
			if err != nil {
				result = fmt.Sprintf("Failed to navigate: %s", err.Error())
				msgType = MessageTypeError
			} else {
				result = fmt.Sprintf("Successfully navigated to %s", pageInfo.Title)
				msgType = MessageTypeSuccess
				av.currentPage = PageState{
					URL:         pageInfo.URL,
					Title:       pageInfo.Title,
					Description: pageInfo.Description,
				}
				// Discover new actions after navigation
				av.discoverPageActions()
			}
		}
	
	case "button":
		err := av.browserClient.ClickElement(action.Selector)
		if err != nil {
			result = fmt.Sprintf("Failed to click button: %s", err.Error())
			msgType = MessageTypeError
		} else {
			result = fmt.Sprintf("Successfully clicked: %s", action.Description)
			msgType = MessageTypeSuccess
			// Discover new actions after clicking
			av.discoverPageActions()
		}
	
	case "input":
		// For inputs, we need to prompt for the value or use a default
		inputType := action.Attributes["type"]
		if inputType == "checkbox" || inputType == "radio" {
			// For checkboxes and radios, just click them
			err := av.browserClient.ClickElement(action.Selector)
			if err != nil {
				result = fmt.Sprintf("Failed to toggle %s: %s", inputType, err.Error())
				msgType = MessageTypeError
			} else {
				result = fmt.Sprintf("Successfully toggled: %s", action.Description)
				msgType = MessageTypeSuccess
			}
		} else {
			// For text inputs, we should ideally prompt for value, but for now use placeholder
			placeholder := action.Attributes["placeholder"]
			if placeholder != "" {
				err := av.browserClient.FillField(action.Selector, placeholder)
				if err != nil {
					result = fmt.Sprintf("Failed to fill field: %s", err.Error())
					msgType = MessageTypeError
				} else {
					result = fmt.Sprintf("Successfully filled field with placeholder text")
					msgType = MessageTypeSuccess
				}
			} else {
				result = fmt.Sprintf("Ready to fill %s (would need value input)", action.Description)
				msgType = MessageTypeSystem
			}
		}
	
	case "clickable":
		err := av.browserClient.ClickElement(action.Selector)
		if err != nil {
			result = fmt.Sprintf("Failed to click element: %s", err.Error())
			msgType = MessageTypeError
		} else {
			result = fmt.Sprintf("Successfully clicked: %s", action.Description)
			msgType = MessageTypeSuccess
			// Discover new actions after clicking
			av.discoverPageActions()
		}
	
	default:
		result = fmt.Sprintf("Action type '%s' not yet supported", action.Type)
		msgType = MessageTypeError
	}

	// Add result message
	av.addMessage(msgType, result)
	
	// Clear input and reset selection
	av.textInput.SetValue("")
	av.actionListState.FilterQuery = ""
	av.actionListState.SelectedIndex = 0
	av.filterActions()

	return av, nil
}

func (av *AdventureView) processCommand(input string) string {
	input = strings.ToLower(input)

	// Handle special commands
	switch {
	case input == "help" || input == "?":
		av.showHelp = !av.showHelp
		return "Help toggled"

	case input == "actions":
		return fmt.Sprintf("Found %d available actions", len(av.actions))

	case input == "history":
		return fmt.Sprintf("Session has %d recorded actions", len(av.history))

	case input == "back":
		return "Navigation back not implemented yet"

	case input == "refresh":
		return "Page refresh not implemented yet"

	case strings.HasPrefix(input, "fill "):
		return av.processFillCommand(input)

	case strings.HasPrefix(input, "click "):
		return av.processClickCommand(input)

	case strings.HasPrefix(input, "go to ") || strings.HasPrefix(input, "visit ") || strings.HasPrefix(input, "navigate to ") || strings.HasPrefix(input, "open "):
		return av.processNavigateCommand(input)

	default:
		// Try to match against available actions or use LLM interpretation
		return av.processActionMatch(input)
	}
}

func (av *AdventureView) processFillCommand(input string) string {
	// Example: "fill email with user@example.com"
	parts := strings.Split(input, " with ")
	if len(parts) != 2 {
		return "Invalid fill command. Use: fill <field> with <value>"
	}

	field := strings.TrimPrefix(parts[0], "fill ")
	value := parts[1]

	// Execute actual fill if browser client is available
	if av.browserClient != nil {
		// Convert natural language to selector
		selector := av.convertToSelector(field)
		err := av.browserClient.FillField(selector, value)
		if err != nil {
			return fmt.Sprintf("❌ Failed to fill '%s': %s", field, err.Error())
		}

		return fmt.Sprintf("✅ Successfully filled '%s' with '%s'", field, value)
	}

	return fmt.Sprintf("Would fill '%s' with '%s' (browser client not connected)", field, value)
}

func (av *AdventureView) processClickCommand(input string) string {
	element := strings.TrimPrefix(input, "click ")
	
	// Execute actual click if browser client is available
	if av.browserClient != nil {
		// Convert natural language to selector
		selector := av.convertToSelector(element)
		err := av.browserClient.ClickElement(selector)
		if err != nil {
			return fmt.Sprintf("❌ Failed to click '%s': %s", element, err.Error())
		}
		
		// Update page state after click
		pageInfo, err := av.browserClient.GetCurrentPage()
		if err == nil {
			av.currentPage = PageState{
				URL:         pageInfo.URL,
				Title:       pageInfo.Title,
				Description: pageInfo.Description,
			}
		}

		// Discover new actions after click
		av.discoverPageActions()

		return fmt.Sprintf("✅ Successfully clicked '%s'", element)
	}

	return fmt.Sprintf("Would click '%s' (browser client not connected)", element)
}

func (av *AdventureView) processNavigateCommand(input string) string {
	var url string
	switch {
	case strings.HasPrefix(input, "go to "):
		url = strings.TrimPrefix(input, "go to ")
	case strings.HasPrefix(input, "visit "):
		url = strings.TrimPrefix(input, "visit ")
	case strings.HasPrefix(input, "navigate to "):
		url = strings.TrimPrefix(input, "navigate to ")
	case strings.HasPrefix(input, "open "):
		url = strings.TrimPrefix(input, "open ")
	}

	// Handle common page aliases
	url = av.resolvePageAlias(url)

	// Execute actual navigation if browser client is available
	if av.browserClient != nil {
		pageInfo, err := av.browserClient.NavigateToURL(url)
		if err != nil {
			return fmt.Sprintf("❌ Failed to navigate to '%s': %s", url, err.Error())
		}

		// Update the current page state
		av.currentPage = PageState{
			URL:         pageInfo.URL,
			Title:       pageInfo.Title,
			Description: pageInfo.Description,
		}

		// Discover new actions after navigation
		av.discoverPageActions()

		return fmt.Sprintf("✅ Successfully navigated to '%s' (%s)", pageInfo.URL, pageInfo.Title)
	}

	return fmt.Sprintf("Would navigate to '%s' (browser client not connected)", url)
}

func (av *AdventureView) processActionMatch(input string) string {
	// First priority: Check discovered page actions with smart matching
	if match := av.findBestPageActionMatch(input); match != nil {
		_, _ = av.executePageAction(*match)
		return fmt.Sprintf(">> Executed: %s", match.Description)
	}

	// Second priority: Process common action aliases and intents
	if result := av.processCommonActionAliases(input); result != "" {
		return result
	}

	// Third priority: Legacy action matching for backward compatibility
	for _, action := range av.actions {
		if strings.Contains(strings.ToLower(action.Name), input) ||
			strings.Contains(strings.ToLower(action.Description), input) {
			return fmt.Sprintf("Would execute action: %s (not implemented yet)", action.Name)
		}
	}

	// Fourth priority: If we have an LLM client, try intelligent interpretation with conversation context
	if av.llmClient != nil {
		ctx := context.Background()
		conversation := av.convertToLLMConversationContext()
		interpretation, err := av.llmClient.InterpretCommandWithContext(ctx, input, av.actions, conversation)
		if err == nil && interpretation != nil {
			return av.processLLMInterpretation(interpretation, input)
		}
	}

	// Fallback: Generate smart suggestions based on available actions
	return av.generateSmartSuggestions(input)
}

// processLLMInterpretation handles the result from LLM command interpretation
func (av *AdventureView) processLLMInterpretation(interp *llm.CommandInterpretation, originalInput string) string {
	switch interp.CommandType {
	case "navigation":
		if page, exists := interp.Parameters["page"]; exists {
			resolvedPage := av.resolvePageAlias(page)
			return fmt.Sprintf(">> Interpreted navigation command → Would navigate to '%s' (not implemented yet)", resolvedPage)
		}
		return fmt.Sprintf(">> Interpreted as navigation command (confidence: %.0f%%)", interp.Confidence*100)

	case "authentication":
		if interp.ActionID != "" {
			return fmt.Sprintf(">> Found matching authentication action → Would execute: %s (not implemented yet)", interp.ActionID)
		}
		return fmt.Sprintf(">> Interpreted as authentication command (confidence: %.0f%%)", interp.Confidence*100)

	case "interaction":
		if element, exists := interp.Parameters["element"]; exists {
			return fmt.Sprintf(">> Would interact with element '%s' (not implemented yet)", element)
		}
		return fmt.Sprintf(">> Interpreted as interaction command (confidence: %.0f%%)", interp.Confidence*100)

	case "form_input":
		if field, exists := interp.Parameters["field"]; exists {
			if value, hasValue := interp.Parameters["value"]; hasValue {
				return fmt.Sprintf(">> Would fill field '%s' with '%s' (not implemented yet)", field, value)
			}
			return fmt.Sprintf(">> Would fill field '%s' (not implemented yet)", field)
		}
		return fmt.Sprintf(">> Interpreted as form input command (confidence: %.0f%%)", interp.Confidence*100)

	case "action_match":
		if interp.ActionID != "" {
			// Find the matching action
			for _, action := range av.actions {
				if action.ID == interp.ActionID {
					return fmt.Sprintf(">> Found similar action → Would execute: %s (not implemented yet)", action.Name)
				}
			}
		}
		return fmt.Sprintf(">> Found potentially matching action (confidence: %.0f%%)", interp.Confidence*100)

	default:
		// Include suggestions from LLM if available
		if len(interp.Suggestions) > 0 {
			suggestions := strings.Join(interp.Suggestions, ", ")
			return fmt.Sprintf("!! Command not recognized. Try: %s", suggestions)
		}
		return av.generateUnknownCommandResponse(originalInput)
	}
}

// generateUnknownCommandResponse creates a helpful response for unrecognized commands
func (av *AdventureView) generateUnknownCommandResponse(input string) string {
	// Provide helpful suggestions based on context
	suggestions := []string{
		"navigate to homepage",
		"sign in",
		"click button",
		"fill field with value",
	}

	// If we have actions available, suggest some of them
	if len(av.actions) > 0 {
		suggestions = append(suggestions, fmt.Sprintf("try action: %s", av.actions[0].Name))
	}

	suggestionText := strings.Join(suggestions, " | ")
	return fmt.Sprintf("!! Unknown command: '%s'. Try: %s", input, suggestionText)
}

// resolvePageAlias converts common page aliases to proper URLs or page names
func (av *AdventureView) resolvePageAlias(page string) string {
	page = strings.ToLower(strings.TrimSpace(page))

	aliases := map[string]string{
		"home":          "/",
		"homepage":      "/",
		"main":          "/",
		"index":         "/",
		"dashboard":     "/dashboard",
		"dash":          "/dashboard",
		"profile":       "/profile",
		"account":       "/account",
		"settings":      "/settings",
		"config":        "/settings",
		"login":         "/login",
		"signin":        "/login",
		"sign-in":       "/login",
		"signup":        "/register",
		"register":      "/register",
		"sign-up":       "/register",
		"logout":        "/logout",
		"signout":       "/logout",
		"sign-out":      "/logout",
		"about":         "/about",
		"contact":       "/contact",
		"help":          "/help",
		"docs":          "/docs",
		"documentation": "/docs",
		"admin":         "/admin",
		"cart":          "/cart",
		"checkout":      "/checkout",
		"orders":        "/orders",
		"billing":       "/billing",
	}

	if resolved, exists := aliases[page]; exists {
		return resolved
	}

	return page
}

func (av *AdventureView) updateSuggestions() {
	input := strings.ToLower(av.textInput.Value())

	if input == "" {
		// Show contextual suggestions based on available actions
		av.suggestions = av.getContextualSuggestions()
		return
	}

	var suggestions []string

	// 1. Try to get intelligent suggestions from LLM if available with conversation context
	if av.llmClient != nil && len(input) > 2 {
		ctx := context.Background()
		conversation := av.convertToLLMConversationContext()
		interpretation, err := av.llmClient.InterpretCommandWithContext(ctx, input, av.actions, conversation)
		if err == nil && interpretation != nil && len(interpretation.Suggestions) > 0 {
			// Use LLM suggestions first (they're usually most relevant)
			for _, suggestion := range interpretation.Suggestions {
				if len(suggestions) < 3 {
					suggestions = append(suggestions, suggestion)
				}
			}
		}
	}

	// 2. Match against available action names and descriptions
	for _, action := range av.actions {
		actionName := strings.ToLower(action.Name)
		actionDesc := strings.ToLower(action.Description)

		if strings.Contains(actionName, input) || strings.Contains(actionDesc, input) {
			if !av.contains(suggestions, action.Name) && len(suggestions) < 5 {
				suggestions = append(suggestions, action.Name)
			}
		}
	}

	// 3. Add command completions based on partial input
	suggestions = av.addCommandCompletions(input, suggestions)

	// 4. Add navigation suggestions for common patterns
	suggestions = av.addNavigationSuggestions(input, suggestions)

	// 5. Fall back to common commands if no specific matches
	if len(suggestions) == 0 {
		suggestions = av.getFallbackSuggestions(input)
	}

	av.suggestions = suggestions
}

// getContextualSuggestions returns suggestions based on current context
func (av *AdventureView) getContextualSuggestions() []string {
	var suggestions []string

	// If we have available actions, suggest the most common ones
	if len(av.actions) > 0 {
		authActions := av.getActionsByCategory("Authentication")
		navActions := av.getActionsByCategory("Navigation")

		if len(authActions) > 0 {
			suggestions = append(suggestions, authActions[0].Name)
		}
		if len(navActions) > 0 {
			suggestions = append(suggestions, "navigate to homepage")
		}

		// Add first few actions if available
		for i, action := range av.actions {
			if i >= 2 || len(suggestions) >= 4 {
				break
			}
			if !av.contains(suggestions, action.Name) {
				suggestions = append(suggestions, action.Name)
			}
		}
	}

	// Always include basic commands
	basicCommands := []string{"help", "actions", "navigate to homepage"}
	for _, cmd := range basicCommands {
		if !av.contains(suggestions, cmd) && len(suggestions) < 5 {
			suggestions = append(suggestions, cmd)
		}
	}

	return suggestions
}

// addCommandCompletions adds command completions based on partial input
func (av *AdventureView) addCommandCompletions(input string, existing []string) []string {
	commandMappings := map[string][]string{
		"nav":   {"navigate to homepage", "navigate to login", "navigate to dashboard"},
		"go":    {"go to homepage", "go to login", "go to dashboard"},
		"visit": {"visit homepage", "visit login", "visit dashboard"},
		"open":  {"open homepage", "open login", "open dashboard"},
		"sign":  {"sign in", "sign up", "sign out"},
		"log":   {"login", "logout"},
		"click": {"click button", "click link", "click sign in"},
		"fill":  {"fill email with user@example.com", "fill password", "fill form"},
		"type":  {"type in field", "type username", "type password"},
	}

	for prefix, completions := range commandMappings {
		if strings.HasPrefix(prefix, input) || strings.HasPrefix(input, prefix) {
			for _, completion := range completions {
				if !av.contains(existing, completion) && len(existing) < 5 {
					existing = append(existing, completion)
				}
			}
		}
	}

	return existing
}

// addNavigationSuggestions adds navigation-specific suggestions
func (av *AdventureView) addNavigationSuggestions(input string, existing []string) []string {
	if strings.Contains(input, "nav") || strings.Contains(input, "go") || strings.Contains(input, "visit") {
		navSuggestions := []string{
			"navigate to homepage",
			"navigate to login",
			"navigate to dashboard",
			"navigate to profile",
		}

		for _, suggestion := range navSuggestions {
			if strings.Contains(suggestion, input) && !av.contains(existing, suggestion) && len(existing) < 5 {
				existing = append(existing, suggestion)
			}
		}
	}

	return existing
}

// getFallbackSuggestions returns fallback suggestions when no matches found
func (av *AdventureView) getFallbackSuggestions(input string) []string {
	return []string{
		"help",
		"navigate to homepage",
		"actions",
		"sign in",
		"click button",
	}
}

// getActionsByCategory returns actions filtered by category
func (av *AdventureView) getActionsByCategory(category string) []types.CodeAction {
	var filtered []types.CodeAction
	for _, action := range av.actions {
		if action.Category == category {
			filtered = append(filtered, action)
		}
	}
	return filtered
}

// contains checks if a string slice contains a specific string
func (av *AdventureView) contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func (av *AdventureView) cycleSuggestions(up bool) (*AdventureView, tea.Cmd) {
	if len(av.suggestions) == 0 {
		return av, nil
	}

	// For now, just set the first suggestion
	// In a full implementation, we'd cycle through them
	av.textInput.SetValue(av.suggestions[0])
	av.textInput.CursorEnd()

	return av, nil
}


// SetActions updates the available actions for the current page
func (av *AdventureView) SetActions(actions []types.CodeAction) {
	av.actions = actions
}

// SetPageState updates the current page state
func (av *AdventureView) SetPageState(state PageState) {
	av.currentPage = state
}

// SetBrowserClient sets the browser automation client
func (av *AdventureView) SetBrowserClient(client *browser.Client) {
	av.browserClient = client
	// Don't trigger action discovery here - wait for navigation first
}

// updateHistoryViewDimensions calculates and updates the history view dimensions
func (av *AdventureView) updateHistoryViewDimensions() {
	// Calculate dimensions for dynamic layout
	headerHeight := 2  // Header + spacing
	inputHeight := 3   // Page info + input + spacing
	minHistoryHeight := 5
	maxActionsHeight := 10  // Max space for action list/help
	
	// Calculate available height for history
	availableHeight := av.height - headerHeight - inputHeight
	
	// Reserve space for actions/suggestions if they exist
	if av.showHelp {
		availableHeight -= 15 // Help text is larger
	} else if len(av.actionListState.FilteredActions) > 0 {
		actionCount := len(av.actionListState.FilteredActions)
		if actionCount > av.actionListState.MaxDisplayed {
			actionCount = av.actionListState.MaxDisplayed
		}
		actionsHeight := actionCount + 3 // Actions + header + spacing
		if actionsHeight > maxActionsHeight {
			actionsHeight = maxActionsHeight
		}
		availableHeight -= actionsHeight
	} else if len(av.suggestions) > 0 {
		availableHeight -= 2 // Suggestions height
	}
	
	// Ensure minimum height
	historyHeight := availableHeight
	if historyHeight < minHistoryHeight {
		historyHeight = minHistoryHeight
	}
	
	// Update textarea dimensions
	av.historyView.SetWidth(av.width - 2) // Account for borders/padding
	av.historyView.SetHeight(historyHeight)
}

// shouldHandleHistoryScrolling determines if we should let the history view handle the key
func (av *AdventureView) shouldHandleHistoryScrolling(msg tea.KeyMsg) bool {
	// Let history handle scrolling keys when input is empty or when explicitly scrolling
	inputEmpty := strings.TrimSpace(av.textInput.Value()) == ""
	scrollKeys := []string{"pgup", "pgdown", "home", "end", "ctrl+home", "ctrl+end"}
	
	for _, key := range scrollKeys {
		if msg.String() == key {
			return true
		}
	}
	
	// Also handle mouse events for history view
	return inputEmpty && (msg.String() == "up" || msg.String() == "down")
}

// handleHistoryScrolling handles scrolling within the history view
func (av *AdventureView) handleHistoryScrolling(msg tea.KeyMsg) (*AdventureView, tea.Cmd) {
	// Mark that user has scrolled up (to disable auto-scroll)
	if msg.String() == "pgup" || msg.String() == "up" || msg.String() == "home" {
		av.userScrolledUp = true
	} else if msg.String() == "pgdown" || msg.String() == "down" || msg.String() == "end" {
		// Check if we're at the bottom after this scroll
		// If so, re-enable auto-scroll
		if msg.String() == "end" || msg.String() == "pgdown" {
			av.userScrolledUp = false
		}
	}
	
	// Let the history view handle the scrolling
	var cmd tea.Cmd
	av.historyView, cmd = av.historyView.Update(msg)
	return av, cmd
}

// exportHistoryAsMarkdown exports the conversation history as markdown
func (av *AdventureView) exportHistoryAsMarkdown() string {
	var content strings.Builder
	
	content.WriteString("# Tod Adventure Session\n\n")
	content.WriteString(fmt.Sprintf("**Environment:** %s\n", av.config.Current))
	if av.currentPage.URL != "Not connected" {
		content.WriteString(fmt.Sprintf("**Current Page:** %s\n", av.currentPage.URL))
	}
	content.WriteString(fmt.Sprintf("**Exported:** %s\n\n", time.Now().Format("2006-01-02 15:04:05")))
	content.WriteString("---\n\n")
	
	for _, msg := range av.history {
		var msgType string
		switch msg.Type {
		case MessageTypeUser:
			msgType = "**User**"
		case MessageTypeSystem:
			msgType = "*System*"
		case MessageTypeSuccess:
			msgType = "**✅ Success**"
		case MessageTypeError:
			msgType = "**❌ Error**"
		}
		
		timestamp := msg.Timestamp.Format("15:04:05")
		content.WriteString(fmt.Sprintf("### %s `%s`\n", msgType, timestamp))
		
		// Format the content with proper code blocks for commands
		if msg.Type == MessageTypeUser {
			content.WriteString(fmt.Sprintf("```\n%s\n```\n\n", msg.Content))
		} else {
			content.WriteString(fmt.Sprintf("%s\n\n", msg.Content))
		}
	}
	
	return content.String()
}

func (av *AdventureView) convertToSelector(element string) string {
	element = strings.ToLower(strings.TrimSpace(element))
	
	// Common element mappings
	selectorMappings := map[string]string{
		"email":                "[type='email'], input[name*='email'], #email",
		"email field":          "[type='email'], input[name*='email'], #email",
		"password":             "[type='password'], input[name*='password'], #password",
		"password field":       "[type='password'], input[name*='password'], #password",
		"login button":         "button[type='submit'], input[type='submit'], button[name*='login'], button[class*='login']",
		"sign in button":       "button[type='submit'], input[type='submit'], button[name*='signin'], button[class*='signin']",
		"submit button":        "button[type='submit'], input[type='submit']",
		"username":             "input[name*='username'], input[name*='user'], #username",
		"username field":       "input[name*='username'], input[name*='user'], #username",
		"search":               "input[type='search'], input[name*='search'], #search",
		"search field":         "input[type='search'], input[name*='search'], #search",
		"button":               "button",
		"link":                 "a",
		"home link":            "a[href='/'], a[href='#/'], nav a[href='/']",
		"homepage link":        "a[href='/'], a[href='#/'], nav a[href='/']",
	}
	
	// Check for exact matches first
	if selector, exists := selectorMappings[element]; exists {
		return selector
	}
	
	// Check for partial matches
	for key, selector := range selectorMappings {
		if strings.Contains(element, key) || strings.Contains(key, element) {
			return selector
		}
	}
	
	// Fallback strategies
	switch {
	case strings.Contains(element, "button"):
		if strings.Contains(element, "login") || strings.Contains(element, "sign in") {
			return "button[type='submit'], button[name*='login'], button[class*='login']"
		}
		return "button"
		
	case strings.Contains(element, "field") || strings.Contains(element, "input"):
		fieldName := strings.Replace(strings.Replace(element, "field", "", -1), "input", "", -1)
		fieldName = strings.TrimSpace(fieldName)
		return fmt.Sprintf("input[name*='%s'], input[placeholder*='%s'], #%s", fieldName, fieldName, fieldName)
		
	case strings.Contains(element, "link"):
		linkText := strings.Replace(element, "link", "", -1)
		linkText = strings.TrimSpace(linkText)
		if linkText != "" {
			return fmt.Sprintf("a[href*='%s'], a[title*='%s']", linkText, linkText)
		}
		return "a"
		
	default:
		// Last resort: try to find by attribute values
		return fmt.Sprintf("[name*='%s'], [id*='%s'], [class*='%s']", element, element, element)
	}
}

// findBestPageActionMatch finds the best matching page action for natural language input
func (av *AdventureView) findBestPageActionMatch(input string) *browser.PageAction {
	if len(av.actionListState.AllActions) == 0 {
		return nil
	}

	input = strings.ToLower(strings.TrimSpace(input))
	var bestMatch *browser.PageAction
	bestScore := 0

	for _, action := range av.actionListState.AllActions {
		score := av.calculateNaturalLanguageScore(action, input)
		if score > bestScore && score >= 50 { // Minimum confidence threshold
			bestScore = score
			actionCopy := action
			bestMatch = &actionCopy
		}
	}

	return bestMatch
}

// calculateNaturalLanguageScore calculates how well an action matches natural language input
func (av *AdventureView) calculateNaturalLanguageScore(action browser.PageAction, input string) int {
	score := 0
	desc := strings.ToLower(action.Description)
	text := strings.ToLower(action.Text)

	// Exact matches get highest score
	if desc == input || text == input {
		return 100
	}

	// Check for exact word matches in description
	if strings.Contains(desc, input) {
		score += 80
	}

	// Check for exact word matches in text content
	if strings.Contains(text, input) {
		score += 70
	}

	// Check for word boundary matches (better than substring)
	inputWords := strings.Fields(input)
	descWords := strings.Fields(desc)
	textWords := strings.Fields(text)

	matchingWords := 0
	for _, inputWord := range inputWords {
		for _, descWord := range descWords {
			if inputWord == descWord {
				matchingWords++
				score += 15
				break
			}
		}
		for _, textWord := range textWords {
			if inputWord == textWord {
				matchingWords++
				score += 10
				break
			}
		}
	}

	// Bonus for high word match ratio
	if len(inputWords) > 0 {
		wordRatio := float64(matchingWords) / float64(len(inputWords))
		if wordRatio > 0.5 {
			score += int(wordRatio * 20)
		}
	}

	// Fuzzy matching bonus
	if av.fuzzyMatch(desc, input) {
		score += 20
	}

	// Priority bonus for interactive elements
	switch action.Type {
	case "button":
		score += 10
	case "link":
		score += 8
	case "input":
		score += 6
	}

	return score
}

// processCommonActionAliases handles common natural language patterns and aliases
func (av *AdventureView) processCommonActionAliases(input string) string {
	input = strings.ToLower(strings.TrimSpace(input))
	
	// Common authentication aliases
	authAliases := map[string][]string{
		"sign in":    {"login", "log in", "sign in", "signin", "authenticate"},
		"sign up":    {"register", "sign up", "signup", "create account", "join"},
		"sign out":   {"logout", "log out", "sign out", "signout"},
	}

	// Navigation aliases
	navAliases := map[string][]string{
		"home":       {"home", "homepage", "main page", "go home"},
		"back":       {"back", "go back", "previous"},
		"forward":    {"forward", "go forward", "next"},
	}

	// Check authentication patterns
	for action, aliases := range authAliases {
		for _, alias := range aliases {
			if input == alias || strings.Contains(input, alias) {
				if match := av.findActionByIntent(action); match != nil {
					_, _ = av.executePageAction(*match)
					return fmt.Sprintf(">> Successfully executed %s", action)
				}
			}
		}
	}

	// Check navigation patterns
	for action, aliases := range navAliases {
		for _, alias := range aliases {
			if input == alias || strings.Contains(input, alias) {
				if match := av.findActionByIntent(action); match != nil {
					_, _ = av.executePageAction(*match)
					return fmt.Sprintf(">> Successfully executed %s", action)
				}
			}
		}
	}

	// Check for form field filling patterns like "fill email" or "enter password"
	fillPatterns := []string{"fill", "enter", "type", "input"}
	for _, pattern := range fillPatterns {
		if strings.HasPrefix(input, pattern+" ") {
			fieldName := strings.TrimPrefix(input, pattern+" ")
			if match := av.findInputFieldByName(fieldName); match != nil {
				// For now, just indicate we found a match
				return fmt.Sprintf(">> Found field to %s: %s (value input needed)", pattern, match.Description)
			}
		}
	}

	// Check for click patterns like "click login" or "press submit"
	clickPatterns := []string{"click", "press", "tap", "select"}
	for _, pattern := range clickPatterns {
		if strings.HasPrefix(input, pattern+" ") {
			elementName := strings.TrimPrefix(input, pattern+" ")
			if match := av.findClickableByName(elementName); match != nil {
				_, _ = av.executePageAction(*match)
				return fmt.Sprintf(">> Successfully %sed %s", pattern, elementName)
			}
		}
	}

	return "" // No alias match found
}

// findActionByIntent finds an action based on intent (like "sign in", "home", etc.)
func (av *AdventureView) findActionByIntent(intent string) *browser.PageAction {
	intentLower := strings.ToLower(intent)
	
	for _, action := range av.actionListState.AllActions {
		desc := strings.ToLower(action.Description)
		text := strings.ToLower(action.Text)
		
		if strings.Contains(desc, intentLower) || strings.Contains(text, intentLower) {
			return &action
		}
	}
	return nil
}

// findInputFieldByName finds an input field by name or related text
func (av *AdventureView) findInputFieldByName(fieldName string) *browser.PageAction {
	fieldLower := strings.ToLower(fieldName)
	
	for _, action := range av.actionListState.AllActions {
		if action.Type != "input" {
			continue
		}
		
		desc := strings.ToLower(action.Description)
		text := strings.ToLower(action.Text)
		
		if strings.Contains(desc, fieldLower) || strings.Contains(text, fieldLower) {
			return &action
		}
	}
	return nil
}

// findClickableByName finds a clickable element by name or related text
func (av *AdventureView) findClickableByName(elementName string) *browser.PageAction {
	nameLower := strings.ToLower(elementName)
	
	for _, action := range av.actionListState.AllActions {
		if action.Type != "button" && action.Type != "link" && action.Type != "clickable" {
			continue
		}
		
		desc := strings.ToLower(action.Description)
		text := strings.ToLower(action.Text)
		
		if strings.Contains(desc, nameLower) || strings.Contains(text, nameLower) {
			return &action
		}
	}
	return nil
}

// generateSmartSuggestions generates intelligent suggestions based on available actions and input
func (av *AdventureView) generateSmartSuggestions(input string) string {
	if len(av.actionListState.AllActions) == 0 {
		return fmt.Sprintf("!! Unknown command '%s'. No page actions available. Try navigating to a page first.", input)
	}

	suggestions := []string{}
	inputLower := strings.ToLower(input)

	// Find partial matches in available actions
	for _, action := range av.actionListState.AllActions {
		desc := strings.ToLower(action.Description)
		text := strings.ToLower(action.Text)
		
		if strings.Contains(desc, inputLower) || strings.Contains(text, inputLower) {
			suggestions = append(suggestions, fmt.Sprintf("'%s'", action.Description))
			if len(suggestions) >= 3 {
				break
			}
		}
	}

	// If no partial matches, suggest common actions
	if len(suggestions) == 0 {
		commonTypes := map[string][]string{}
		for _, action := range av.actionListState.AllActions {
			commonTypes[action.Type] = append(commonTypes[action.Type], action.Description)
		}

		if buttons, ok := commonTypes["button"]; ok && len(buttons) > 0 {
			suggestions = append(suggestions, fmt.Sprintf("Try: click %s", strings.ToLower(buttons[0])))
		}
		if links, ok := commonTypes["link"]; ok && len(links) > 0 {
			suggestions = append(suggestions, fmt.Sprintf("Try: %s", strings.ToLower(links[0])))
		}
		if inputs, ok := commonTypes["input"]; ok && len(inputs) > 0 {
			suggestions = append(suggestions, fmt.Sprintf("Try: fill %s", strings.ToLower(inputs[0])))
		}
	}

	if len(suggestions) == 0 {
		return fmt.Sprintf("!! Unknown command '%s'. Type 'help' for available commands or try describing what you want to do.", input)
	}

	return fmt.Sprintf("!! Unknown command '%s'. Did you mean: %s", input, strings.Join(suggestions, ", "))
}

// convertToLLMConversationContext converts internal conversation history to LLM format
func (av *AdventureView) convertToLLMConversationContext() *llm.ConversationContext {
	if len(av.history) == 0 {
		return &llm.ConversationContext{
			SessionID: av.sessionID,
			Messages:  []llm.ConversationMessage{},
		}
	}

	// Convert recent messages (last 10 to keep token usage reasonable)
	recentHistory := av.history
	if len(recentHistory) > 10 {
		recentHistory = recentHistory[len(recentHistory)-10:]
	}

	llmMessages := make([]llm.ConversationMessage, 0, len(recentHistory))
	
	for _, msg := range recentHistory {
		llmMsg := llm.ConversationMessage{
			Role:    av.convertMessageTypeToRole(msg.Type),
			Content: msg.Content,
		}
		llmMessages = append(llmMessages, llmMsg)
	}

	return &llm.ConversationContext{
		SessionID: av.sessionID,
		Messages:  llmMessages,
		MaxTokens: 4000, // Reasonable limit for conversation context
	}
}

// convertMessageTypeToRole converts internal MessageType to LLM role format
func (av *AdventureView) convertMessageTypeToRole(msgType MessageType) string {
	switch msgType {
	case MessageTypeUser:
		return "user"
	case MessageTypeSystem, MessageTypeSuccess, MessageTypeError:
		return "assistant"
	default:
		return "assistant"
	}
}
