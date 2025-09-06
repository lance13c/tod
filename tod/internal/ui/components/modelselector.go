package components

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ciciliostudio/tod/internal/llm"
)

// ModelSelectorOption represents a model option with metadata
type ModelSelectorOption struct {
	ID          string
	Name        string
	Description string
	Pricing     string
	Context     string
	Selected    bool
	Disabled    bool
}

// ModelSelectorModel handles searchable model selection
type ModelSelectorModel struct {
	title           string
	allOptions      []ModelSelectorOption
	filteredOptions []ModelSelectorOption
	searchInput     textinput.Model
	cursor          int
	selectedModel   string
	width           int
	height          int
	finished        bool
	cancelled       bool
	showHelp        bool

	// Styles
	titleStyle       lipgloss.Style
	searchStyle      lipgloss.Style
	optionStyle      lipgloss.Style
	selectedStyle    lipgloss.Style
	cursorStyle      lipgloss.Style
	descriptionStyle lipgloss.Style
	metaStyle        lipgloss.Style
	helpStyle        lipgloss.Style
	borderStyle      lipgloss.Style
}

// NewModelSelectorModel creates a new model selector
func NewModelSelectorModel(title string, models []llm.OpenRouterModel) *ModelSelectorModel {
	// Convert OpenRouter models to options
	options := make([]ModelSelectorOption, len(models))
	for i, model := range models {
		description := model.Description
		if description == "" {
			description = "AI language model"
		}

		// Format pricing info
		pricing := "Free"
		if model.Pricing.Prompt != "" && model.Pricing.Prompt != "0" {
			pricing = "💰 " + model.Pricing.Prompt + "/1K tokens"
		}

		// Format context info
		context := "N/A"
		if model.Context > 0 {
			if model.Context >= 1000000 {
				context = fmt.Sprintf("%.1fM", float64(model.Context)/1000000)
			} else if model.Context >= 1000 {
				context = fmt.Sprintf("%.0fK", float64(model.Context)/1000)
			} else {
				context = fmt.Sprintf("%d", model.Context)
			}
		}

		options[i] = ModelSelectorOption{
			ID:          model.ID,
			Name:        model.Name,
			Description: description,
			Pricing:     pricing,
			Context:     context,
		}
	}

	// Sort options by name
	sort.Slice(options, func(i, j int) bool {
		return strings.ToLower(options[i].Name) < strings.ToLower(options[j].Name)
	})

	// Create search input
	searchInput := textinput.New()
	searchInput.Placeholder = "Search models..."
	searchInput.Focus()
	searchInput.CharLimit = 50
	searchInput.Width = 50

	return &ModelSelectorModel{
		title:           title,
		allOptions:      options,
		filteredOptions: options,
		searchInput:     searchInput,
		width:           100,
		height:          25,

		// Default styles
		titleStyle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7D56F4")).
			MarginBottom(1),

		searchStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7D56F4")).
			Padding(0, 1),

		optionStyle: lipgloss.NewStyle().
			PaddingLeft(2),

		selectedStyle: lipgloss.NewStyle().
			PaddingLeft(2).
			Foreground(lipgloss.Color("#7D56F4")).
			Bold(true),

		cursorStyle: lipgloss.NewStyle().
			PaddingLeft(0).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Bold(true),

		descriptionStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262")).
			PaddingLeft(4),

		metaStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			PaddingLeft(4),

		helpStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262")).
			MarginTop(1),

		borderStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#626262")).
			Padding(1),
	}
}

// SetSize sets the display size
func (m *ModelSelectorModel) SetSize(width, height int) *ModelSelectorModel {
	m.width = width
	m.height = height
	return m
}

// GetSelectedModel returns the ID of the selected model
func (m *ModelSelectorModel) GetSelectedModel() string {
	return m.selectedModel
}

// IsFinished returns true if selection is complete
func (m *ModelSelectorModel) IsFinished() bool {
	return m.finished
}

// IsCancelled returns true if selection was cancelled
func (m *ModelSelectorModel) IsCancelled() bool {
	return m.cancelled
}

// filterOptions filters options based on search query
func (m *ModelSelectorModel) filterOptions() {
	query := strings.ToLower(strings.TrimSpace(m.searchInput.Value()))
	
	if query == "" {
		m.filteredOptions = m.allOptions
		return
	}

	var filtered []ModelSelectorOption
	for _, option := range m.allOptions {
		// Search in name, description, and ID
		searchText := strings.ToLower(option.Name + " " + option.Description + " " + option.ID)
		if strings.Contains(searchText, query) {
			filtered = append(filtered, option)
		}
	}

	m.filteredOptions = filtered
	
	// Reset cursor if it's out of bounds
	if m.cursor >= len(m.filteredOptions) {
		m.cursor = 0
	}
}

// Init initializes the model selector
func (m ModelSelectorModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles model selector updates
func (m ModelSelectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.cancelled = true
			return m, tea.Quit

		case "enter":
			if len(m.filteredOptions) > 0 && m.cursor < len(m.filteredOptions) {
				m.selectedModel = m.filteredOptions[m.cursor].ID
				m.finished = true
				return m, tea.Quit
			}

		case "up", "k":
			if len(m.filteredOptions) > 0 {
				if m.cursor > 0 {
					m.cursor--
				} else {
					m.cursor = len(m.filteredOptions) - 1
				}
			}

		case "down", "j":
			if len(m.filteredOptions) > 0 {
				if m.cursor < len(m.filteredOptions)-1 {
					m.cursor++
				} else {
					m.cursor = 0
				}
			}

		case "pgup":
			if len(m.filteredOptions) > 0 {
				m.cursor = max(0, m.cursor-10)
			}

		case "pgdown":
			if len(m.filteredOptions) > 0 {
				m.cursor = min(len(m.filteredOptions)-1, m.cursor+10)
			}

		case "home":
			m.cursor = 0

		case "end":
			if len(m.filteredOptions) > 0 {
				m.cursor = len(m.filteredOptions) - 1
			}

		case "?":
			m.showHelp = !m.showHelp

		default:
			// Handle search input
			oldValue := m.searchInput.Value()
			m.searchInput, cmd = m.searchInput.Update(msg)
			
			// If search changed, re-filter
			if m.searchInput.Value() != oldValue {
				m.filterOptions()
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.searchInput.Width = min(50, m.width-10)
	}

	return m, cmd
}

// View renders the model selector
func (m ModelSelectorModel) View() string {
	if m.finished || m.cancelled {
		return ""
	}

	var b strings.Builder

	// Title
	b.WriteString(m.titleStyle.Render(m.title))
	b.WriteString("\n\n")

	// Search input
	b.WriteString("🔍 " + m.searchStyle.Render(m.searchInput.View()))
	b.WriteString("\n\n")

	// Results count
	if len(m.filteredOptions) == 0 {
		b.WriteString(m.helpStyle.Render("No models found matching your search."))
		b.WriteString("\n\n")
		b.WriteString(m.helpStyle.Render("[esc] Cancel • [?] Help"))
		return m.borderStyle.Width(m.width - 4).Render(b.String())
	}

	count := len(m.filteredOptions)
	total := len(m.allOptions)
	if count != total {
		b.WriteString(m.helpStyle.Render(fmt.Sprintf("Showing %d of %d models", count, total)))
	} else {
		b.WriteString(m.helpStyle.Render(fmt.Sprintf("%d models available", total)))
	}
	b.WriteString("\n\n")

	// Calculate visible range for scrolling
	maxVisible := min(m.height-10, 15) // Reserve space for header/footer
	start := 0
	end := len(m.filteredOptions)

	if len(m.filteredOptions) > maxVisible {
		// Center cursor in visible area
		start = max(0, m.cursor-maxVisible/2)
		end = min(len(m.filteredOptions), start+maxVisible)
		
		// Adjust if we're at the end
		if end == len(m.filteredOptions) && end-start < maxVisible {
			start = max(0, end-maxVisible)
		}
	}

	// Show scroll indicator if needed
	if start > 0 {
		b.WriteString(m.metaStyle.Render("    ↑ " + fmt.Sprintf("%d more above", start)))
		b.WriteString("\n")
	}

	// Model options
	for i := start; i < end; i++ {
		option := m.filteredOptions[i]
		
		// Model name line
		var line strings.Builder
		
		if i == m.cursor {
			line.WriteString(m.cursorStyle.Render("→ " + option.Name))
		} else {
			line.WriteString(m.optionStyle.Render("  " + option.Name))
		}
		
		b.WriteString(line.String())
		b.WriteString("\n")
		
		// Model metadata
		meta := fmt.Sprintf("ID: %s • Context: %s • %s", 
			option.ID, option.Context, option.Pricing)
		b.WriteString(m.metaStyle.Render(meta))
		b.WriteString("\n")
		
		// Description if available
		if option.Description != "" && option.Description != "AI language model" {
			desc := option.Description
			if len(desc) > 80 {
				desc = desc[:77] + "..."
			}
			b.WriteString(m.descriptionStyle.Render(desc))
			b.WriteString("\n")
		}
		
		b.WriteString("\n")
	}

	// Show scroll indicator if needed
	if end < len(m.filteredOptions) {
		remaining := len(m.filteredOptions) - end
		b.WriteString(m.metaStyle.Render("    ↓ " + fmt.Sprintf("%d more below", remaining)))
		b.WriteString("\n")
	}

	// Help text
	if m.showHelp {
		help := []string{
			"Navigation:",
			"  ↑↓/jk: move cursor",
			"  PgUp/PgDn: jump 10 models",
			"  Home/End: first/last model",
			"  Type to search models",
			"",
			"Actions:",
			"  Enter: select model",
			"  Esc: cancel selection",
			"  ?: toggle this help",
		}
		b.WriteString("\n")
		b.WriteString(m.helpStyle.Render(strings.Join(help, "\n")))
	} else {
		help := "[↑↓/jk] Navigate • [Enter] Select • [Esc] Cancel • [?] Help"
		b.WriteString(m.helpStyle.Render(help))
	}

	return m.borderStyle.Width(m.width - 4).Render(b.String())
}

// RunModelSelection runs model selection and returns the selected model ID
func RunModelSelection(title string, models []llm.OpenRouterModel) (string, error) {
	model := NewModelSelectorModel(title, models)

	program := tea.NewProgram(model, tea.WithAltScreen())
	result, err := program.Run()
	if err != nil {
		return "", err
	}

	finalModel := result.(ModelSelectorModel)
	if finalModel.IsCancelled() {
		return "", ErrSelectionCancelled{}
	}

	return finalModel.GetSelectedModel(), nil
}

// Helper functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}