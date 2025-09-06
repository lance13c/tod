package views

import (
	"context"
	"fmt"
	"strings"

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

	// User input
	textInput   textinput.Model
	suggestions []string
	showHelp    bool

	// Session tracking
	sessionID string
	history   []ActionHistory

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

// ActionHistory tracks actions taken during the session
type ActionHistory struct {
	Action     string
	Timestamp  string
	Result     string
	PageBefore string
	PageAfter  string
}

// AdventureStyles contains styling for the adventure interface
type AdventureStyles struct {
	Header      lipgloss.Style
	PageInfo    lipgloss.Style
	ActionBox   lipgloss.Style
	InputPrompt lipgloss.Style
	Suggestions lipgloss.Style
	Help        lipgloss.Style
	Error       lipgloss.Style
	Success     lipgloss.Style
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

	return &AdventureView{
		config:    cfg,
		textInput: ti,
		llmClient: llmClient,
		styles:    NewAdventureStyles(),
		currentPage: PageState{
			URL:         "Not connected",
			Title:       "Tod Adventure",
			Description: "Ready to begin your testing journey",
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
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#874BFD")).
			Padding(1).
			MarginBottom(1),

		ActionBox: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#04B575")).
			Padding(1).
			MarginBottom(1),

		InputPrompt: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#04B575")).
			Bold(true),

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
	}
}

// Init implements tea.Model
func (av *AdventureView) Init() tea.Cmd {
	return textinput.Blink
}

// Update implements tea.Model
func (av *AdventureView) Update(msg tea.Msg) (*AdventureView, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		av.width = msg.Width
		av.height = msg.Height
		av.textInput.Width = av.width - 4

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return av, tea.Quit
		case "enter":
			return av.handleUserInput()
		case "ctrl+h", "?":
			av.showHelp = !av.showHelp
			return av, nil
		case "up", "down":
			return av.cycleSuggestions(msg.String() == "up")
		}
	}

	// Update text input
	av.textInput, cmd = av.textInput.Update(msg)

	// Clear last result when user starts typing new input
	if len(av.textInput.Value()) > 0 && av.lastResult != "" {
		av.lastResult = ""
	}

	// Update suggestions based on current input
	av.updateSuggestions()

	return av, cmd
}

// View implements tea.Model
func (av *AdventureView) View() string {
	var sections []string

	// Header
	header := av.styles.Header.Render("[*] Tod Adventure Mode - " + av.config.Current)
	sections = append(sections, header)

	// Current page info
	pageInfo := av.renderPageInfo()
	sections = append(sections, pageInfo)

	// Available actions
	if len(av.actions) > 0 {
		actionsInfo := av.renderAvailableActions()
		sections = append(sections, actionsInfo)
	}

	// Show last command result
	if av.lastResult != "" && !av.showHelp {
		feedback := av.renderFeedback()
		sections = append(sections, feedback)
	}

	// Input prompt
	inputSection := av.renderInputSection()
	sections = append(sections, inputSection)

	// Suggestions
	if len(av.suggestions) > 0 && !av.showHelp {
		suggestions := av.renderSuggestions()
		sections = append(sections, suggestions)
	}

	// Help
	if av.showHelp {
		help := av.renderHelp()
		sections = append(sections, help)
	}

	return strings.Join(sections, "\n")
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
  • "help" or "?" - Show/hide this help
  • "actions" - List all available actions
  • "history" - Show session history
  • "back" - Go back to previous page
  • "refresh" - Reload current page

Tips:
  • Use TAB for autocomplete
  • Press ↑/↓ to cycle through suggestions
  • Type part of an action name to filter
  • All actions are recorded for test generation`

	return av.styles.Help.Render(help)
}

func (av *AdventureView) handleUserInput() (*AdventureView, tea.Cmd) {
	input := strings.TrimSpace(av.textInput.Value())
	if input == "" {
		return av, nil
	}

	// Clear the input
	av.textInput.SetValue("")

	// Process the command
	result := av.processCommand(input)

	// Store the result for display
	av.lastResult = result

	// Add to history
	av.addToHistory(input, result)

	// TODO: Execute the actual action and update page state
	// This will be implemented when we add browser automation

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

		return fmt.Sprintf("✅ Successfully navigated to '%s' (%s)", pageInfo.URL, pageInfo.Title)
	}

	return fmt.Sprintf("Would navigate to '%s' (browser client not connected)", url)
}

func (av *AdventureView) processActionMatch(input string) string {
	// First, try to find a matching action by name or description
	for _, action := range av.actions {
		if strings.Contains(strings.ToLower(action.Name), input) ||
			strings.Contains(strings.ToLower(action.Description), input) {
			return fmt.Sprintf("Would execute action: %s (not implemented yet)", action.Name)
		}
	}

	// If no direct match and we have an LLM client, try intelligent interpretation
	if av.llmClient != nil {
		ctx := context.Background()
		interpretation, err := av.llmClient.InterpretCommand(ctx, input, av.actions)
		if err == nil && interpretation != nil {
			return av.processLLMInterpretation(interpretation, input)
		}
	}

	// Fallback to unknown command with helpful suggestions
	return av.generateUnknownCommandResponse(input)
}

// processLLMInterpretation handles the result from LLM command interpretation
func (av *AdventureView) processLLMInterpretation(interp *llm.CommandInterpretation, originalInput string) string {
	switch interp.CommandType {
	case "navigation":
		if page, exists := interp.Parameters["page"]; exists {
			resolvedPage := av.resolvePageAlias(page)
			return fmt.Sprintf("🧭 Interpreted navigation command → Would navigate to '%s' (not implemented yet)", resolvedPage)
		}
		return fmt.Sprintf("🧭 Interpreted as navigation command (confidence: %.0f%%)", interp.Confidence*100)

	case "authentication":
		if interp.ActionID != "" {
			return fmt.Sprintf("🔐 Found matching authentication action → Would execute: %s (not implemented yet)", interp.ActionID)
		}
		return fmt.Sprintf("🔐 Interpreted as authentication command (confidence: %.0f%%)", interp.Confidence*100)

	case "interaction":
		if element, exists := interp.Parameters["element"]; exists {
			return fmt.Sprintf("👆 Would interact with element '%s' (not implemented yet)", element)
		}
		return fmt.Sprintf("👆 Interpreted as interaction command (confidence: %.0f%%)", interp.Confidence*100)

	case "form_input":
		if field, exists := interp.Parameters["field"]; exists {
			if value, hasValue := interp.Parameters["value"]; hasValue {
				return fmt.Sprintf("✏️ Would fill field '%s' with '%s' (not implemented yet)", field, value)
			}
			return fmt.Sprintf("✏️ Would fill field '%s' (not implemented yet)", field)
		}
		return fmt.Sprintf("✏️ Interpreted as form input command (confidence: %.0f%%)", interp.Confidence*100)

	case "action_match":
		if interp.ActionID != "" {
			// Find the matching action
			for _, action := range av.actions {
				if action.ID == interp.ActionID {
					return fmt.Sprintf("🎯 Found similar action → Would execute: %s (not implemented yet)", action.Name)
				}
			}
		}
		return fmt.Sprintf("🎯 Found potentially matching action (confidence: %.0f%%)", interp.Confidence*100)

	default:
		// Include suggestions from LLM if available
		if len(interp.Suggestions) > 0 {
			suggestions := strings.Join(interp.Suggestions, ", ")
			return fmt.Sprintf("❓ Command not recognized. Try: %s", suggestions)
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
	return fmt.Sprintf("❓ Unknown command: '%s'. Try: %s", input, suggestionText)
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

	// 1. Try to get intelligent suggestions from LLM if available
	if av.llmClient != nil && len(input) > 2 {
		ctx := context.Background()
		interpretation, err := av.llmClient.InterpretCommand(ctx, input, av.actions)
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

func (av *AdventureView) addToHistory(command, result string) {
	history := ActionHistory{
		Action:     command,
		Timestamp:  fmt.Sprintf("%d", len(av.history)+1), // Simple counter for now
		Result:     result,
		PageBefore: av.currentPage.URL,
		PageAfter:  av.currentPage.URL, // Will be different after real navigation
	}

	av.history = append(av.history, history)
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
}

// convertToSelector converts natural language element descriptions to CSS selectors
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
