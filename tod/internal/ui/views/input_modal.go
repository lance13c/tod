package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ciciliostudio/tod/internal/config"
)

// InputModalMode represents the current mode of the input modal
type InputModalMode int

const (
	InputModeTyping InputModalMode = iota
	InputModeSelecting
)

// SavedUserOption represents a saved user option in the modal
type SavedUserOption struct {
	User     config.TestUser
	Display  string  // e.g., "dominic@ciciliostudio.com (Dom)"
	Selected bool
	Index    int
}

// InputModal handles interactive form field input with saved user options
type InputModal struct {
	// Field information
	fieldType   FormFieldType
	fieldLabel  string
	placeholder string
	domain      string

	// Input state
	textInput    textinput.Model
	mode         InputModalMode
	showModal    bool

	// Saved user options
	savedUsers     []SavedUserOption
	selectedIndex  int
	maxOptions     int

	// UI dimensions
	width  int
	height int

	// Styles
	modalStyle       lipgloss.Style
	headerStyle      lipgloss.Style
	inputStyle       lipgloss.Style
	optionStyle      lipgloss.Style
	selectedStyle    lipgloss.Style
	helpStyle        lipgloss.Style
	borderStyle      lipgloss.Style

	// Result
	completed bool
	cancelled bool
	result    InputResult
}

// InputResult represents the result of the input modal
type InputResult struct {
	Value       string
	SelectedUser *config.TestUser
	NewEntry     bool
	Cancelled    bool
}

// NewInputModal creates a new input modal
func NewInputModal(fieldType FormFieldType, fieldLabel, placeholder, domain string, savedUsers []config.TestUser) *InputModal {
	// Create text input
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.CharLimit = 200
	ti.Width = 50
	ti.Focus()

	// Convert saved users to options
	var options []SavedUserOption
	for i, user := range savedUsers {
		display := user.Email
		if user.Name != "" {
			display = fmt.Sprintf("%s (%s)", user.Email, user.Name)
		}
		options = append(options, SavedUserOption{
			User:    user,
			Display: display,
			Index:   i,
		})
	}

	return &InputModal{
		fieldType:   fieldType,
		fieldLabel:  fieldLabel,
		placeholder: placeholder,
		domain:      domain,
		textInput:   ti,
		mode:        InputModeTyping,
		showModal:   true,
		savedUsers:  options,
		selectedIndex: -1,
		maxOptions:  5,
		width:       80,
		height:      25,

		// Styles
		modalStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1, 2).
			Background(lipgloss.Color("235")).
			Width(60).
			Height(20),

		headerStyle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			MarginBottom(1),

		inputStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(0, 1).
			MarginBottom(1),

		optionStyle: lipgloss.NewStyle().
			PaddingLeft(2).
			Foreground(lipgloss.Color("248")),

		selectedStyle: lipgloss.NewStyle().
			PaddingLeft(2).
			Background(lipgloss.Color("62")).
			Foreground(lipgloss.Color("230")).
			Bold(true),

		helpStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			MarginTop(1),

		borderStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")),
	}
}

// Update handles modal updates
func (m *InputModal) Update(msg tea.Msg) (*InputModal, tea.Cmd) {
	if !m.showModal {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.modalStyle = m.modalStyle.Width(min(msg.Width-4, 60)).Height(min(msg.Height-4, 20))

	case tea.KeyMsg:
		return m.handleKeyPress(msg)
	}

	// Update text input if in typing mode
	if m.mode == InputModeTyping {
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

// handleKeyPress handles keyboard input for the modal
func (m *InputModal) handleKeyPress(msg tea.KeyMsg) (*InputModal, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.cancelled = true
		m.completed = true
		m.showModal = false
		m.result = InputResult{Cancelled: true}
		return m, nil

	case tea.KeyEnter:
		return m.handleEnterKey()

	case tea.KeyDown:
		if len(m.savedUsers) > 0 {
			if m.mode == InputModeTyping {
				// Switch to selecting mode
				m.mode = InputModeSelecting
				m.selectedIndex = 0
			} else {
				// Move down in options
				if m.selectedIndex < len(m.savedUsers)-1 {
					m.selectedIndex++
				} else {
					// Wrap to typing mode or stay at bottom
					m.selectedIndex = len(m.savedUsers) - 1
				}
			}
		}
		return m, nil

	case tea.KeyUp:
		if m.mode == InputModeSelecting {
			if m.selectedIndex > 0 {
				m.selectedIndex--
			} else {
				// Switch back to typing mode
				m.mode = InputModeTyping
				m.selectedIndex = -1
			}
		}
		return m, nil

	case tea.KeyTab:
		// Toggle between typing and selecting modes
		if len(m.savedUsers) > 0 {
			if m.mode == InputModeTyping {
				m.mode = InputModeSelecting
				if m.selectedIndex == -1 {
					m.selectedIndex = 0
				}
			} else {
				m.mode = InputModeTyping
				m.selectedIndex = -1
			}
		}
		return m, nil

	default:
		// Handle regular typing in typing mode
		if m.mode == InputModeTyping {
			var cmd tea.Cmd
			m.textInput, cmd = m.textInput.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

// handleEnterKey handles the Enter key press
func (m *InputModal) handleEnterKey() (*InputModal, tea.Cmd) {
	if m.mode == InputModeSelecting && m.selectedIndex >= 0 && m.selectedIndex < len(m.savedUsers) {
		// User selected a saved user
		selectedUser := m.savedUsers[m.selectedIndex].User
		m.result = InputResult{
			Value:        m.getValueForUser(selectedUser),
			SelectedUser: &selectedUser,
			NewEntry:     false,
			Cancelled:    false,
		}
	} else {
		// User entered new text
		inputValue := strings.TrimSpace(m.textInput.Value())
		if inputValue == "" {
			// Don't complete if empty input
			return m, nil
		}

		m.result = InputResult{
			Value:        inputValue,
			SelectedUser: nil,
			NewEntry:     true,
			Cancelled:    false,
		}
	}

	m.completed = true
	m.showModal = false
	return m, nil
}

// getValueForUser returns the appropriate value for the selected user based on field type
func (m *InputModal) getValueForUser(user config.TestUser) string {
	switch m.fieldType {
	case EmailField:
		return user.Email
	case UsernameField:
		if user.Username != "" {
			return user.Username
		}
		return user.Email // Fallback to email
	case PasswordField:
		return user.Password
	default:
		return user.Email // Default fallback
	}
}

// View renders the input modal
func (m *InputModal) View() string {
	if !m.showModal {
		return ""
	}

	// Header
	header := m.headerStyle.Render(fmt.Sprintf("Enter %s", m.fieldLabel))

	// Input section
	var inputSection string
	if m.mode == InputModeTyping {
		inputSection = m.inputStyle.Render(m.textInput.View())
	} else {
		// Show input in non-focused state
		unfocusedInput := m.textInput
		unfocusedInput.Blur()
		inputSection = m.inputStyle.Render(unfocusedInput.View())
	}

	// Saved users section
	var optionsSection string
	if len(m.savedUsers) > 0 {
		optionsLines := []string{}
		
		// Add header for saved users
		optionsLines = append(optionsLines, m.helpStyle.Render("Saved users for "+m.domain+":"))

		// Show up to maxOptions
		displayCount := min(len(m.savedUsers), m.maxOptions)
		for i := 0; i < displayCount; i++ {
			option := m.savedUsers[i]
			line := fmt.Sprintf("□ %s", option.Display)

			if m.mode == InputModeSelecting && i == m.selectedIndex {
				line = fmt.Sprintf("■ %s", option.Display)
				optionsLines = append(optionsLines, m.selectedStyle.Render(line))
			} else {
				optionsLines = append(optionsLines, m.optionStyle.Render(line))
			}
		}

		if len(m.savedUsers) > m.maxOptions {
			optionsLines = append(optionsLines, m.helpStyle.Render(fmt.Sprintf("... and %d more", len(m.savedUsers)-m.maxOptions)))
		}

		optionsSection = strings.Join(optionsLines, "\n")
	}

	// Help section
	helpLines := []string{}
	helpLines = append(helpLines, "[Enter: confirm]")
	
	if len(m.savedUsers) > 0 {
		helpLines = append(helpLines, "[↑↓: select saved user]")
		helpLines = append(helpLines, "[Tab: toggle input/select]")
	}
	
	helpLines = append(helpLines, "[Esc: cancel]")
	helpText := m.helpStyle.Render(strings.Join(helpLines, " "))

	// Build modal content
	sections := []string{header, inputSection}
	
	if optionsSection != "" {
		sections = append(sections, "", optionsSection)
	}
	
	sections = append(sections, "", helpText)

	modalContent := strings.Join(sections, "\n")
	
	// Wrap in modal style
	return m.centerModal(m.modalStyle.Render(modalContent))
}

// centerModal centers the modal on the screen
func (m *InputModal) centerModal(content string) string {
	if m.width == 0 || m.height == 0 {
		return content
	}

	// Simple centering - this could be enhanced
	padding := strings.Repeat("\n", max(0, (m.height-10)/2))
	return padding + content
}

// IsComplete returns whether the modal interaction is complete
func (m *InputModal) IsComplete() bool {
	return m.completed
}

// IsCancelled returns whether the modal was cancelled
func (m *InputModal) IsCancelled() bool {
	return m.cancelled
}

// GetResult returns the modal result
func (m *InputModal) GetResult() InputResult {
	return m.result
}

// IsShowing returns whether the modal is currently showing
func (m *InputModal) IsShowing() bool {
	return m.showModal
}

// Show displays the modal
func (m *InputModal) Show() {
	m.showModal = true
	m.completed = false
	m.cancelled = false
	m.result = InputResult{}
	m.textInput.Focus()
}

// Hide hides the modal
func (m *InputModal) Hide() {
	m.showModal = false
}

// Reset resets the modal state
func (m *InputModal) Reset() {
	m.textInput.SetValue("")
	m.mode = InputModeTyping
	m.selectedIndex = -1
	m.completed = false
	m.cancelled = false
	m.result = InputResult{}
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