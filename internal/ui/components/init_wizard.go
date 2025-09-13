package components

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lance13c/tod/internal/config"
)

// InitStep represents a step in the initialization wizard
type InitStep int

const (
	StepFramework InitStep = iota
	StepFrameworkVersion
	StepLanguage
	StepTestDir
	StepCommand
	StepPattern
	StepAIModel
	StepAPIKey
	StepEnvironment
	StepBaseURL
	StepComplete
)

// InitWizardModel handles the unified initialization wizard
type InitWizardModel struct {
	currentStep InitStep
	steps       []InitStep
	stepTitles  map[InitStep]string
	width       int
	height      int
	finished    bool
	cancelled   bool

	// Current input
	textInput    textinput.Model
	suggestions  []string
	filteredSugs []string
	selectedSug  int
	showSugs     bool
	placeholder  string
	defaultValue string

	// History tracking per field type
	history    map[InitStep][]string
	maxHistory int

	// Configuration being built
	config        *config.Config
	testingConfig config.TestingConfig
	aiConfig      config.AIConfig
	envConfig     config.EnvConfig

	// Detection results - simplified to avoid import cycles
	detectedFramework map[string]string // Store simple key-value pairs

	// Context
	cwd string

	// Styles
	titleStyle       lipgloss.Style
	stepStyle        lipgloss.Style
	inputStyle       lipgloss.Style
	suggestStyle     lipgloss.Style
	selectedSugStyle lipgloss.Style
	historyStyle     lipgloss.Style
	helpStyle        lipgloss.Style
}

// NewInitWizardModel creates a new initialization wizard model
func NewInitWizardModel(cwd string, existingConfig *config.Config) *InitWizardModel {
	ti := textinput.New()
	ti.Focus()
	ti.CharLimit = 200
	ti.Width = 60

	steps := []InitStep{
		StepFramework, StepFrameworkVersion, StepLanguage, StepTestDir,
		StepCommand, StepPattern, StepAIModel, StepAPIKey,
		StepEnvironment, StepBaseURL, StepComplete,
	}

	stepTitles := map[InitStep]string{
		StepFramework:        "E2E Testing Framework",
		StepFrameworkVersion: "Framework Version",
		StepLanguage:         "Programming Language",
		StepTestDir:          "Test Directory",
		StepCommand:          "Run Command",
		StepPattern:          "File Pattern",
		StepAIModel:          "AI Model",
		StepAPIKey:           "API Key",
		StepEnvironment:      "Environment Name",
		StepBaseURL:          "Base URL",
		StepComplete:         "Complete",
	}

	// Initialize config structs with existing values or defaults
	var testingConfig config.TestingConfig
	var aiConfig config.AIConfig
	var envConfig config.EnvConfig

	if existingConfig != nil {
		testingConfig = existingConfig.Testing
		aiConfig = existingConfig.AI
		currentEnv := existingConfig.GetCurrentEnv()
		if currentEnv != nil {
			envConfig = *currentEnv
		} else {
			envConfig = config.EnvConfig{Name: "development", BaseURL: "http://localhost:3000"}
		}
	} else {
		testingConfig = config.TestingConfig{}
		aiConfig = config.AIConfig{}
		envConfig = config.EnvConfig{Name: "development", BaseURL: "http://localhost:3000"}
	}

	return &InitWizardModel{
		currentStep: StepFramework,
		steps:       steps,
		stepTitles:  stepTitles,
		textInput:   ti,
		width:       80,
		maxHistory:  10,
		history:     make(map[InitStep][]string),
		cwd:         cwd,

		// Initialize configs with existing or default values
		testingConfig: testingConfig,
		aiConfig:      aiConfig,
		envConfig:     envConfig,

		// Styles
		titleStyle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7D56F4")).
			MarginBottom(1).
			Align(lipgloss.Center),

		stepStyle: lipgloss.NewStyle().
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

// Init initializes the wizard model
func (m *InitWizardModel) Init() tea.Cmd {
	m.setupStepInput()
	return textinput.Blink
}

// Update handles wizard model updates
func (m *InitWizardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.cancelled = true
			return m, tea.Quit

		case "enter":
			if m.showSugs && len(m.filteredSugs) > 0 && m.selectedSug >= 0 && m.selectedSug < len(m.filteredSugs) {
				m.textInput.SetValue(m.filteredSugs[m.selectedSug])
				m.showSugs = false
				m.selectedSug = 0
			} else {
				// Save current input and move to next step
				value := m.getCurrentValue()
				m.saveCurrentStep(value)
				m.addToHistory(m.currentStep, value)

				// Save config to disk after each step (except Complete)
				if m.currentStep != StepComplete {
					if err := m.upsertConfigStep(); err != nil {
						// Log error but don't halt the wizard
						fmt.Printf("⚠️ Warning: Could not save configuration: %v\n", err)
					}
				}

				if m.currentStep == StepComplete {
					m.finished = true
					return m, tea.Quit
				}

				m.nextStep()
				m.setupStepInput()
			}

		case "tab":
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
			m.textInput, cmd = m.textInput.Update(msg)
			m.updateSuggestions()
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.textInput.Width = msg.Width - 4

	default:
		m.textInput, cmd = m.textInput.Update(msg)
	}

	return m, cmd
}

// View renders the initialization wizard
func (m *InitWizardModel) View() string {
	if m.finished || m.cancelled {
		return ""
	}

	var b strings.Builder

	// Title
	title := "🗿 THE TOD INITIALIZATION WIZARD 🗿"
	b.WriteString(m.titleStyle.Render(title))
	b.WriteString("\n\n")

	// Progress indicator
	progress := fmt.Sprintf("Step %d of %d", int(m.currentStep)+1, len(m.steps))
	b.WriteString(m.stepStyle.Render(progress))
	b.WriteString("\n")

	// Special handling for complete step
	if m.currentStep == StepComplete {
		b.WriteString("\n")
		completeStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#4A9EFF")).
			MarginTop(2).
			MarginBottom(2).
			Align(lipgloss.Center)
		
		b.WriteString(completeStyle.Render("✓ Configuration Complete!"))
		b.WriteString("\n\n")
		b.WriteString(m.helpStyle.Render("Press enter to finish initialization..."))
		return b.String()
	}

	// Show history for current step
	if history, exists := m.history[m.currentStep]; exists && len(history) > 0 {
		for _, historyItem := range history {
			b.WriteString(m.historyStyle.Render("  > " + historyItem))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	// Current step prompt
	stepTitle := m.stepTitles[m.currentStep]
	b.WriteString(m.stepStyle.Render(stepTitle + ":"))
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

// setupStepInput configures input for the current step
func (m *InitWizardModel) setupStepInput() {
	switch m.currentStep {
	case StepFramework:
		m.suggestions = []string{
			"playwright", "cypress", "selenium", "puppeteer", "webdriverio",
			"testcafe", "nightwatch", "jest-puppeteer",
		}
		m.defaultValue = getValueOrFallback(m.testingConfig.Framework, "playwright")
		m.placeholder = "Enter framework name"

	case StepFrameworkVersion:
		m.suggestions = []string{"latest", "1.40.0", "^1.40.0", "~1.40.0"}
		m.defaultValue = getValueOrFallback(m.testingConfig.Version, "1.40.0")
		m.placeholder = "Enter framework version"

	case StepLanguage:
		m.suggestions = LanguageSuggestions
		m.defaultValue = getValueOrFallback(m.testingConfig.Language, "typescript")
		m.placeholder = "Programming language for tests"

	case StepTestDir:
		// Filter test directory suggestions to only show existing paths
		var existingPaths []string
		for _, path := range TestDirectorySuggestions {
			fullPath := filepath.Join(m.cwd, path)
			if info, err := os.Stat(fullPath); err == nil && info.IsDir() {
				existingPaths = append(existingPaths, path)
			}
		}
		m.suggestions = existingPaths
		m.defaultValue = getValueOrFallback(m.testingConfig.TestDir, "tests/e2e")
		m.placeholder = "Directory containing test files"

	case StepCommand:
		m.suggestions = TestCommandSuggestions
		m.defaultValue = getValueOrFallback(m.testingConfig.Command, "npm test")
		m.placeholder = "Command to execute tests"

	case StepPattern:
		m.suggestions = FileExtensionSuggestions
		m.defaultValue = getValueOrFallback(m.testingConfig.Pattern, "*.spec.ts")
		m.placeholder = "File pattern for test files"

	case StepAIModel:
		modelSuggestions := config.GetModelSuggestions()
		modelIDs := config.GetModelIDs()
		m.suggestions = append(modelSuggestions, modelIDs...)

		// Convert existing AI config to display name or use default
		if m.aiConfig.Provider != "" && m.aiConfig.Model != "" {
			// Try to find matching display name in registry
			for _, model := range config.ModelRegistry {
				if model.Provider == m.aiConfig.Provider && model.ModelName == m.aiConfig.Model {
					m.defaultValue = model.DisplayName
					break
				}
			}
			// If no match found, create a display string
			if m.defaultValue == "" {
				m.defaultValue = m.aiConfig.Provider + ":" + m.aiConfig.Model
			}
		} else {
			m.defaultValue = "gpt-5 (OpenAI)"
		}
		m.placeholder = "Select a model or enter custom:provider:model-name"

	case StepAPIKey:
		m.suggestions = []string{"sk-...", "sk-ant-...", "AIza...", "${TOD_AI_API_KEY}"}
		m.defaultValue = m.aiConfig.APIKey // Use existing API key or empty
		m.placeholder = "Enter your API key (leave empty to use TOD_AI_API_KEY)"

	case StepEnvironment:
		m.suggestions = []string{"development", "staging", "production", "local"}
		m.defaultValue = getValueOrFallback(m.envConfig.Name, "development")
		m.placeholder = "Environment name"

	case StepBaseURL:
		m.suggestions = []string{
			"http://localhost:3000", "http://localhost:8080", "http://localhost:4200",
			"https://staging.example.com", "https://example.com",
		}
		m.defaultValue = getValueOrFallback(m.envConfig.BaseURL, "http://localhost:3000")
		m.placeholder = "Base URL for the environment"
	
	case StepComplete:
		// Clear input for complete step
		m.suggestions = nil
		m.defaultValue = ""
		m.placeholder = ""
		m.textInput.SetValue("")
		m.textInput.Blur()
		return
	}

	m.textInput.SetValue(m.defaultValue)
	m.textInput.Placeholder = m.placeholder
	m.updateSuggestions()
}

// Helper function to get a value or fallback
func getValueOrFallback(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}

// updateSuggestions filters suggestions based on current input
func (m *InitWizardModel) updateSuggestions() {
	input := strings.ToLower(strings.TrimSpace(m.textInput.Value()))

	if input == "" {
		m.showSugs = false
		m.filteredSugs = nil
		m.selectedSug = 0
		return
	}

	var filtered []string
	maxSugs := 5
	for _, suggestion := range m.suggestions {
		if strings.Contains(strings.ToLower(suggestion), input) {
			filtered = append(filtered, suggestion)
			if len(filtered) >= maxSugs {
				break
			}
		}
	}

	m.filteredSugs = filtered
	m.showSugs = len(filtered) > 0 && input != ""
	m.selectedSug = 0
}

// getCurrentValue returns the current input value or default
func (m *InitWizardModel) getCurrentValue() string {
	value := strings.TrimSpace(m.textInput.Value())
	if value == "" {
		return m.defaultValue
	}
	return value
}

// saveCurrentStep saves the current step's input to the appropriate config
func (m *InitWizardModel) saveCurrentStep(value string) {
	switch m.currentStep {
	case StepFramework:
		m.testingConfig.Framework = value
	case StepFrameworkVersion:
		m.testingConfig.Version = value
	case StepLanguage:
		m.testingConfig.Language = value
	case StepTestDir:
		m.testingConfig.TestDir = value
	case StepCommand:
		m.testingConfig.Command = value
	case StepPattern:
		m.testingConfig.Pattern = value
	case StepAIModel:
		// Parse model selection - convert display name to ID first
		modelID := parseModelDisplayName(value)
		selectedModel, err := config.ParseModelSelection(modelID)
		if err != nil {
			selectedModel = &config.ModelInfo{
				ID:        "gpt-5",
				Provider:  "openai",
				ModelName: "gpt-5",
			}
		}
		m.aiConfig.Provider = selectedModel.Provider
		m.aiConfig.Model = selectedModel.ModelName
	case StepAPIKey:
		if value == "" {
			value = os.Getenv("TOD_AI_API_KEY")
		}
		m.aiConfig.APIKey = value
	case StepEnvironment:
		m.envConfig.Name = value
	case StepBaseURL:
		m.envConfig.BaseURL = value
	}
}

// addToHistory adds a value to the step's history
func (m *InitWizardModel) addToHistory(step InitStep, value string) {
	if value == "" || value == m.defaultValue {
		return
	}

	if m.history[step] == nil {
		m.history[step] = make([]string, 0)
	}

	history := m.history[step]

	// Remove if already exists
	for i, h := range history {
		if h == value {
			history = append(history[:i], history[i+1:]...)
			break
		}
	}

	// Add to front
	history = append([]string{value}, history...)

	// Limit size
	if len(history) > m.maxHistory {
		history = history[:m.maxHistory]
	}

	m.history[step] = history
}

// nextStep moves to the next step
func (m *InitWizardModel) nextStep() {
	if int(m.currentStep) < len(m.steps)-1 {
		m.currentStep = InitStep(int(m.currentStep) + 1)
	}
}

// buildHelpText creates help text for the current step
func (m *InitWizardModel) buildHelpText() string {
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

// IsFinished returns true if wizard is complete
func (m *InitWizardModel) IsFinished() bool {
	return m.finished
}

// IsCancelled returns true if wizard was cancelled
func (m *InitWizardModel) IsCancelled() bool {
	return m.cancelled
}

// GetConfig returns the built configuration
func (m *InitWizardModel) GetConfig() *config.Config {
	now := time.Now()
	return &config.Config{
		AI:      m.aiConfig,
		Testing: m.testingConfig,
		Envs: map[string]config.EnvConfig{
			m.envConfig.Name: m.envConfig,
		},
		Current: m.envConfig.Name,
		Meta: config.MetaConfig{
			Version:   "1.0.0",
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
}

// parseModelDisplayName converts a display name back to model ID for parsing
func parseModelDisplayName(displayName string) string {
	// Check if it's already a model ID (no parentheses)
	if !strings.Contains(displayName, "(") {
		return displayName
	}

	// Try to find a model with matching display name
	for _, model := range config.ModelRegistry {
		if model.DisplayName == displayName {
			return model.ID
		}
	}

	// Fallback: try to extract a reasonable model ID from display name
	// e.g., "Gemini 2.5 Flash (Google)" -> try "gemini-2.5-flash"
	lower := strings.ToLower(displayName)
	if strings.Contains(lower, "gemini") && strings.Contains(lower, "2.5") && strings.Contains(lower, "flash") {
		return "gemini-2.5-flash"
	}
	if strings.Contains(lower, "gemini") && strings.Contains(lower, "2.5") && strings.Contains(lower, "pro") {
		return "gemini-2.5-pro"
	}
	if strings.Contains(lower, "claude") && strings.Contains(lower, "4") && strings.Contains(lower, "opus") {
		return "claude-4-opus"
	}
	if strings.Contains(lower, "gpt-5") {
		return "gpt-5"
	}
	if strings.Contains(lower, "grok-4") {
		return "grok-4"
	}

	// Default fallback
	return displayName
}

// upsertConfigStep saves the current step's configuration to disk
func (m *InitWizardModel) upsertConfigStep() error {
	// Create .tod directory if it doesn't exist
	configDir := filepath.Join(m.cwd, ".tod")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create .tod directory: %w", err)
	}

	configPath := filepath.Join(configDir, "config.yaml")

	// Load existing config or create default
	loader := config.NewLoader(m.cwd)
	var existingConfig *config.Config
	if loader.IsInitialized() {
		var err error
		existingConfig, err = loader.Load()
		if err != nil {
			// If we can't load existing, start with default
			existingConfig = config.DefaultConfig()
		}
	} else {
		existingConfig = config.DefaultConfig()
	}

	// Update existing config with current values
	if m.testingConfig.Framework != "" {
		existingConfig.Testing.Framework = m.testingConfig.Framework
	}
	if m.testingConfig.Version != "" {
		existingConfig.Testing.Version = m.testingConfig.Version
	}
	if m.testingConfig.Language != "" {
		existingConfig.Testing.Language = m.testingConfig.Language
	}
	if m.testingConfig.TestDir != "" {
		existingConfig.Testing.TestDir = m.testingConfig.TestDir
	}
	if m.testingConfig.Command != "" {
		existingConfig.Testing.Command = m.testingConfig.Command
	}
	if m.testingConfig.Pattern != "" {
		existingConfig.Testing.Pattern = m.testingConfig.Pattern
	}

	if m.aiConfig.Provider != "" {
		existingConfig.AI.Provider = m.aiConfig.Provider
	}
	if m.aiConfig.Model != "" {
		existingConfig.AI.Model = m.aiConfig.Model
	}
	if m.aiConfig.APIKey != "" {
		existingConfig.AI.APIKey = m.aiConfig.APIKey
	}

	if m.envConfig.Name != "" {
		if existingConfig.Envs == nil {
			existingConfig.Envs = make(map[string]config.EnvConfig)
		}
		existingConfig.Envs[m.envConfig.Name] = m.envConfig
		existingConfig.Current = m.envConfig.Name
	}

	// Save the updated config
	return loader.Save(existingConfig, configPath)
}

// RunInitWizard runs the initialization wizard
func RunInitWizard(cwd string, existingConfig *config.Config) (*config.Config, error) {
	model := NewInitWizardModel(cwd, existingConfig)

	program := tea.NewProgram(model, tea.WithAltScreen())
	result, err := program.Run()
	if err != nil {
		return nil, err
	}

	finalModel := result.(*InitWizardModel)
	if finalModel.IsCancelled() {
		return nil, ErrInputCancelled{}
	}

	return finalModel.GetConfig(), nil
}
