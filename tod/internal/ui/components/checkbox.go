package components

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// CheckboxOption represents an option in the checkbox list
type CheckboxOption struct {
	ID          string
	Title       string
	Description string
	Selected    bool
	Disabled    bool
}

// CheckboxModel handles multi-select checkbox lists
type CheckboxModel struct {
	title       string
	options     []CheckboxOption
	cursor      int
	selected    map[string]bool
	minSelect   int
	maxSelect   int
	width       int
	height      int
	finished    bool
	cancelled   bool
	
	// Styles
	titleStyle       lipgloss.Style
	optionStyle      lipgloss.Style
	selectedStyle    lipgloss.Style
	cursorStyle      lipgloss.Style
	descriptionStyle lipgloss.Style
	helpStyle        lipgloss.Style
}

// NewCheckboxModel creates a new checkbox model
func NewCheckboxModel(title string, options []CheckboxOption) *CheckboxModel {
	selected := make(map[string]bool)
	for _, opt := range options {
		if opt.Selected {
			selected[opt.ID] = true
		}
	}

	return &CheckboxModel{
		title:    title,
		options:  options,
		selected: selected,
		width:    80,
		height:   20,
		minSelect: 0,
		maxSelect: len(options),
		
		// Default styles
		titleStyle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7D56F4")).
			MarginBottom(1),
		
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
		
		helpStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262")).
			MarginTop(1),
	}
}

// SetConstraints sets selection constraints
func (m *CheckboxModel) SetConstraints(min, max int) *CheckboxModel {
	m.minSelect = min
	m.maxSelect = max
	return m
}

// SetSize sets the display size
func (m *CheckboxModel) SetSize(width, height int) *CheckboxModel {
	m.width = width
	m.height = height
	return m
}

// GetSelected returns the IDs of selected options
func (m *CheckboxModel) GetSelected() []string {
	var selected []string
	for id := range m.selected {
		selected = append(selected, id)
	}
	return selected
}

// IsFinished returns true if selection is complete
func (m *CheckboxModel) IsFinished() bool {
	return m.finished
}

// IsCancelled returns true if selection was cancelled
func (m *CheckboxModel) IsCancelled() bool {
	return m.cancelled
}

// Init initializes the checkbox model
func (m CheckboxModel) Init() tea.Cmd {
	return nil
}

// Update handles checkbox model updates
func (m CheckboxModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			m.cancelled = true
			return m, tea.Quit
			
		case "enter":
			selectedCount := len(m.selected)
			if selectedCount >= m.minSelect && selectedCount <= m.maxSelect {
				m.finished = true
				return m, tea.Quit
			}
			
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
			
		case "down", "j":
			if m.cursor < len(m.options)-1 {
				m.cursor++
			}
			
		case " ", "x":
			if m.cursor < len(m.options) {
				option := m.options[m.cursor]
				if !option.Disabled {
					if m.selected[option.ID] {
						delete(m.selected, option.ID)
					} else {
						if len(m.selected) < m.maxSelect {
							m.selected[option.ID] = true
						}
					}
				}
			}
		}
		
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	
	return m, nil
}

// View renders the checkbox list
func (m CheckboxModel) View() string {
	if m.finished || m.cancelled {
		return ""
	}

	var b strings.Builder
	
	// Title
	b.WriteString(m.titleStyle.Render(m.title))
	b.WriteString("\n\n")
	
	// Options
	for i, option := range m.options {
		var line strings.Builder
		
		// Checkbox symbol
		var checkbox string
		if m.selected[option.ID] {
			checkbox = "☑"
		} else {
			checkbox = "☐"
		}
		
		// Option text
		optionText := checkbox + " " + option.Title
		
		// Apply styles based on cursor position and selection
		if i == m.cursor {
			if option.Disabled {
				line.WriteString(m.cursorStyle.Copy().
					Foreground(lipgloss.Color("#626262")).
					Render("→ " + optionText))
			} else {
				line.WriteString(m.cursorStyle.Render("→ " + optionText))
			}
		} else {
			if option.Disabled {
				line.WriteString(m.optionStyle.Copy().
					Foreground(lipgloss.Color("#626262")).
					Render("  " + optionText))
			} else if m.selected[option.ID] {
				line.WriteString(m.selectedStyle.Render("  " + optionText))
			} else {
				line.WriteString(m.optionStyle.Render("  " + optionText))
			}
		}
		
		b.WriteString(line.String())
		
		// Description
		if option.Description != "" {
			b.WriteString("\n")
			b.WriteString(m.descriptionStyle.Render(option.Description))
		}
		
		b.WriteString("\n")
	}
	
	// Help text
	help := m.buildHelpText()
	b.WriteString("\n")
	b.WriteString(m.helpStyle.Render(help))
	
	return b.String()
}

// buildHelpText creates context-appropriate help text
func (m CheckboxModel) buildHelpText() string {
	selectedCount := len(m.selected)
	
	var parts []string
	
	// Navigation help
	parts = append(parts, "↑↓/jk: navigate")
	parts = append(parts, "space/x: toggle")
	
	// Selection constraints
	if m.minSelect > 0 || m.maxSelect < len(m.options) {
		if selectedCount < m.minSelect {
			parts = append(parts, 
				lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6B6B")).
					Render("need "+pluralize(m.minSelect-selectedCount, "more selection")))
		} else if selectedCount >= m.minSelect {
			parts = append(parts, "enter: confirm")
		}
		
		if selectedCount >= m.maxSelect {
			parts = append(parts, 
				lipgloss.NewStyle().Foreground(lipgloss.Color("#FFD93D")).
					Render("max selections reached"))
		}
	} else {
		parts = append(parts, "enter: confirm")
	}
	
	parts = append(parts, "q/esc: cancel")
	
	// Selection count
	if m.maxSelect > 1 {
		countText := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4")).
			Render("(" + pluralize(selectedCount, "selection") + ")")
		parts = append(parts, countText)
	}
	
	return strings.Join(parts, " • ")
}

// pluralize returns singular or plural form based on count
func pluralize(count int, word string) string {
	if count == 1 {
		return "1 " + word
	}
	return lipgloss.NewStyle().Bold(true).Render(string(rune('0'+count))) + " " + word + "s"
}

// RunCheckboxSelection runs a checkbox selection and returns the selected IDs
func RunCheckboxSelection(title string, options []CheckboxOption) ([]string, error) {
	model := NewCheckboxModel(title, options)
	
	program := tea.NewProgram(model, tea.WithAltScreen())
	result, err := program.Run()
	if err != nil {
		return nil, err
	}
	
	finalModel := result.(CheckboxModel)
	if finalModel.IsCancelled() {
		return nil, ErrSelectionCancelled{}
	}
	
	return finalModel.GetSelected(), nil
}

// ErrSelectionCancelled is returned when user cancels selection
type ErrSelectionCancelled struct{}

func (e ErrSelectionCancelled) Error() string {
	return "selection cancelled"
}