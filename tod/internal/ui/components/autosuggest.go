package components

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// AutoSuggestModel handles text input with auto-suggestions
type AutoSuggestModel struct {
	textInput     textinput.Model
	suggestions   []string
	filteredSugs  []string
	selectedSug   int
	showSugs      bool
	prompt        string
	placeholder   string
	defaultValue  string
	finished      bool
	cancelled     bool
	width         int
	maxSugs       int
	
	// Command history
	history       []string
	maxHistory    int
	showHistory   bool
	
	// Styles
	promptStyle     lipgloss.Style
	inputStyle      lipgloss.Style
	suggestStyle    lipgloss.Style
	selectedSugStyle lipgloss.Style
	helpStyle       lipgloss.Style
	historyStyle    lipgloss.Style
}

// NewAutoSuggestModel creates a new auto-suggest input model
func NewAutoSuggestModel(prompt, placeholder, defaultValue string, suggestions []string) *AutoSuggestModel {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.SetValue(defaultValue)
	ti.Focus()
	ti.CharLimit = 200
	ti.Width = 60

	return &AutoSuggestModel{
		textInput:    ti,
		suggestions:  suggestions,
		prompt:       prompt,
		placeholder:  placeholder,
		defaultValue: defaultValue,
		width:        80,
		maxSugs:      5,
		maxHistory:   10,
		
		// Default styles
		promptStyle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7D56F4")).
			MarginBottom(1),
		
		inputStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7D56F4")).
			Padding(0, 1),
		
		suggestStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262")).
			PaddingLeft(2),
		
		selectedSugStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			PaddingLeft(1).
			PaddingRight(1),
		
		historyStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#4A4A4A")).
			Italic(true).
			PaddingLeft(2),
		
		helpStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262")).
			MarginTop(1),
	}
}

// SetWidth sets the display width
func (m *AutoSuggestModel) SetWidth(width int) *AutoSuggestModel {
	m.width = width
	m.textInput.Width = width - 4 // Account for border padding
	return m
}

// SetMaxSuggestions sets the maximum number of suggestions to show
func (m *AutoSuggestModel) SetMaxSuggestions(max int) *AutoSuggestModel {
	m.maxSugs = max
	return m
}

// SetSuggestions updates the available suggestions
func (m *AutoSuggestModel) SetSuggestions(suggestions []string) *AutoSuggestModel {
	m.suggestions = suggestions
	m.updateSuggestions() // Refresh filtered suggestions based on current input
	return m
}

// SetHistory updates the command history
func (m *AutoSuggestModel) SetHistory(history []string) *AutoSuggestModel {
	m.history = history
	m.showHistory = len(history) > 0
	return m
}

// AddToHistory adds a value to the command history
func (m *AutoSuggestModel) AddToHistory(value string) {
	if value == "" || value == m.defaultValue {
		return
	}
	
	// Remove if already exists to avoid duplicates
	for i, h := range m.history {
		if h == value {
			m.history = append(m.history[:i], m.history[i+1:]...)
			break
		}
	}
	
	// Add to front
	m.history = append([]string{value}, m.history...)
	
	// Limit history size
	if len(m.history) > m.maxHistory {
		m.history = m.history[:m.maxHistory]
	}
	
	m.showHistory = len(m.history) > 0
}

// GetValue returns the current input value
func (m *AutoSuggestModel) GetValue() string {
	value := strings.TrimSpace(m.textInput.Value())
	if value == "" {
		return m.defaultValue
	}
	return value
}

// IsFinished returns true if input is complete
func (m *AutoSuggestModel) IsFinished() bool {
	return m.finished
}

// IsCancelled returns true if input was cancelled
func (m *AutoSuggestModel) IsCancelled() bool {
	return m.cancelled
}

// Init initializes the auto-suggest model
func (m AutoSuggestModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles auto-suggest model updates
func (m AutoSuggestModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.cancelled = true
			return m, tea.Quit
			
		case "enter":
			// If a suggestion is selected, use it
			if m.showSugs && len(m.filteredSugs) > 0 && m.selectedSug >= 0 && m.selectedSug < len(m.filteredSugs) {
				m.textInput.SetValue(m.filteredSugs[m.selectedSug])
				m.showSugs = false
				m.selectedSug = 0
			} else {
				// Finish input
				m.finished = true
				return m, tea.Quit
			}
			
		case "tab":
			// Autocomplete with first suggestion
			if m.showSugs && len(m.filteredSugs) > 0 {
				m.textInput.SetValue(m.filteredSugs[0])
				m.showSugs = false
				m.selectedSug = 0
			}
			
		case "up":
			if m.showSugs && len(m.filteredSugs) > 0 {
				if m.selectedSug > 0 {
					m.selectedSug--
				}
			}
			
		case "down":
			if m.showSugs && len(m.filteredSugs) > 0 {
				if m.selectedSug < len(m.filteredSugs)-1 {
					m.selectedSug++
				}
			}
			
		default:
			// Update text input and filter suggestions
			m.textInput, cmd = m.textInput.Update(msg)
			m.updateSuggestions()
		}
		
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.textInput.Width = msg.Width - 4
		
	default:
		m.textInput, cmd = m.textInput.Update(msg)
	}
	
	return m, cmd
}

// updateSuggestions filters suggestions based on current input
func (m *AutoSuggestModel) updateSuggestions() {
	input := strings.ToLower(strings.TrimSpace(m.textInput.Value()))
	
	if input == "" {
		m.showSugs = false
		m.filteredSugs = nil
		m.selectedSug = 0
		return
	}
	
	// Filter suggestions
	var filtered []string
	for _, suggestion := range m.suggestions {
		if strings.Contains(strings.ToLower(suggestion), input) {
			filtered = append(filtered, suggestion)
			if len(filtered) >= m.maxSugs {
				break
			}
		}
	}
	
	m.filteredSugs = filtered
	m.showSugs = len(filtered) > 0 && input != ""
	m.selectedSug = 0
}

// View renders the auto-suggest input
func (m AutoSuggestModel) View() string {
	if m.finished || m.cancelled {
		return ""
	}
	
	var b strings.Builder
	
	// Show history at the top (like Claude Code)
	if m.showHistory && len(m.history) > 0 {
		for _, historyItem := range m.history {
			b.WriteString(m.historyStyle.Render("  > " + historyItem))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	
	// Prompt
	b.WriteString(m.promptStyle.Render(m.prompt))
	b.WriteString("\n")
	
	// Input field
	inputView := m.inputStyle.Render(m.textInput.View())
	b.WriteString(inputView)
	b.WriteString("\n")
	
	// Suggestions
	if m.showSugs && len(m.filteredSugs) > 0 {
		b.WriteString("\n")
		for i, suggestion := range m.filteredSugs {
			if i == m.selectedSug {
				b.WriteString(m.selectedSugStyle.Render("→ " + suggestion))
			} else {
				b.WriteString(m.suggestStyle.Render("  " + suggestion))
			}
			b.WriteString("\n")
		}
	}
	
	// Help text
	help := m.buildHelpText()
	b.WriteString("\n")
	b.WriteString(m.helpStyle.Render(help))
	
	return b.String()
}

// buildHelpText creates context-appropriate help text
func (m AutoSuggestModel) buildHelpText() string {
	var parts []string
	
	if m.showSugs && len(m.filteredSugs) > 0 {
		parts = append(parts, "↑↓: navigate suggestions")
		parts = append(parts, "tab: autocomplete")
		parts = append(parts, "enter: select/confirm")
	} else {
		parts = append(parts, "enter: confirm")
	}
	
	if m.defaultValue != "" {
		parts = append(parts, "empty for default: "+m.defaultValue)
	}
	
	parts = append(parts, "esc: cancel")
	
	return strings.Join(parts, " • ")
}

// Common suggestion sets for different use cases
var (
	TestCommandSuggestions = []string{
		"npm test",
		"npm run test",
		"npm run test:e2e",
		"pnpm test",
		"pnpm run test",
		"pnpm run test:e2e",
		"yarn test",
		"yarn run test",
		"yarn run test:e2e",
		"npx playwright test",
		"npx cypress run",
		"npx jest",
		"npm run cypress",
		"pnpm run cypress",
		"yarn run cypress",
		"npm run playwright",
		"pnpm run playwright",
		"yarn run playwright",
		"make test",
		"go test",
		"python -m pytest",
		"python -m unittest",
		"mvn test",
		"gradle test",
		"deno test",
		"bun test",
	}
	
	TestDirectorySuggestions = []string{
		"tests",
		"test",
		"e2e",
		"tests/e2e",
		"test/e2e",
		"cypress/e2e",
		"__tests__",
		"spec",
		"specs",
		"integration",
		"tests/integration",
		"src/__tests__",
		"src/tests",
	}
	
	FileExtensionSuggestions = []string{
		"*.spec.ts",
		"*.test.ts",
		"*.e2e.ts",
		"*.spec.js",
		"*.test.js",
		"*.e2e.js",
		"*.cy.ts",
		"*.cy.js",
		"**/*.spec.ts",
		"**/*.test.ts",
		"**/*.e2e.ts",
		"**/*.spec.js",
		"**/*.test.js",
		"**/*.e2e.js",
	}
	
	LanguageSuggestions = []string{
		"typescript",
		"javascript",
		"python",
		"java",
		"csharp",
		"go",
		"ruby",
		"php",
	}
)

// RunAutoSuggestInput runs an auto-suggest input and returns the value
func RunAutoSuggestInput(prompt, placeholder, defaultValue string, suggestions []string) (string, error) {
	return RunAutoSuggestInputWithHistory(prompt, placeholder, defaultValue, suggestions, nil)
}

// RunAutoSuggestInputWithHistory runs an auto-suggest input with command history
func RunAutoSuggestInputWithHistory(prompt, placeholder, defaultValue string, suggestions []string, history []string) (string, error) {
	model := NewAutoSuggestModel(prompt, placeholder, defaultValue, suggestions).SetHistory(history)
	
	program := tea.NewProgram(model, tea.WithAltScreen())
	result, err := program.Run()
	if err != nil {
		return "", err
	}
	
	finalModel := result.(AutoSuggestModel)
	if finalModel.IsCancelled() {
		return "", ErrInputCancelled{}
	}
	
	return finalModel.GetValue(), nil
}

// ErrInputCancelled is returned when user cancels input
type ErrInputCancelled struct{}

func (e ErrInputCancelled) Error() string {
	return "input cancelled"
}