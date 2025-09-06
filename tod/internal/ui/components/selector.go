package components

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SelectorOption represents an option in the selector list
type SelectorOption struct {
	ID          string
	Title       string
	Description string
	Metadata    map[string]string // Additional info like pricing, capabilities, etc.
	Disabled    bool
}

// SelectorModel handles single-select option lists
type SelectorModel struct {
	title       string
	options     []SelectorOption
	cursor      int
	selected    string
	showDetails bool
	finished    bool
	cancelled   bool
	width       int
	height      int
	
	// Styles
	titleStyle       lipgloss.Style
	optionStyle      lipgloss.Style
	selectedStyle    lipgloss.Style
	cursorStyle      lipgloss.Style
	descriptionStyle lipgloss.Style
	metadataStyle    lipgloss.Style
	helpStyle        lipgloss.Style
}

// NewSelectorModel creates a new selector model
func NewSelectorModel(title string, options []SelectorOption) *SelectorModel {
	return &SelectorModel{
		title:   title,
		options: options,
		width:   80,
		height:  20,
		
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
		
		metadataStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFD93D")).
			PaddingLeft(4).
			Italic(true),
		
		helpStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262")).
			MarginTop(1),
	}
}

// SetSize sets the display size
func (m *SelectorModel) SetSize(width, height int) *SelectorModel {
	m.width = width
	m.height = height
	return m
}

// SetDefaultSelection sets the default selected option by ID
func (m *SelectorModel) SetDefaultSelection(id string) *SelectorModel {
	for i, option := range m.options {
		if option.ID == id {
			m.cursor = i
			break
		}
	}
	return m
}

// ShowDetails toggles showing detailed metadata
func (m *SelectorModel) ShowDetails(show bool) *SelectorModel {
	m.showDetails = show
	return m
}

// GetSelected returns the ID of the selected option
func (m *SelectorModel) GetSelected() string {
	return m.selected
}

// GetSelectedOption returns the full selected option
func (m *SelectorModel) GetSelectedOption() *SelectorOption {
	if m.selected == "" {
		return nil
	}
	
	for _, option := range m.options {
		if option.ID == m.selected {
			return &option
		}
	}
	
	return nil
}

// IsFinished returns true if selection is complete
func (m *SelectorModel) IsFinished() bool {
	return m.finished
}

// IsCancelled returns true if selection was cancelled
func (m *SelectorModel) IsCancelled() bool {
	return m.cancelled
}

// Init initializes the selector model
func (m SelectorModel) Init() tea.Cmd {
	return nil
}

// Update handles selector model updates
func (m SelectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			m.cancelled = true
			return m, tea.Quit
			
		case "enter", " ":
			if m.cursor < len(m.options) && !m.options[m.cursor].Disabled {
				m.selected = m.options[m.cursor].ID
				m.finished = true
				return m, tea.Quit
			}
			
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				// Skip disabled options
				for m.cursor >= 0 && m.options[m.cursor].Disabled {
					m.cursor--
				}
				if m.cursor < 0 {
					m.cursor = 0
				}
			}
			
		case "down", "j":
			if m.cursor < len(m.options)-1 {
				m.cursor++
				// Skip disabled options
				for m.cursor < len(m.options) && m.options[m.cursor].Disabled {
					m.cursor++
				}
				if m.cursor >= len(m.options) {
					m.cursor = len(m.options) - 1
				}
			}
			
		case "d":
			// Toggle details view
			m.showDetails = !m.showDetails
		}
		
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	
	return m, nil
}

// View renders the selector list
func (m SelectorModel) View() string {
	if m.finished || m.cancelled {
		return ""
	}

	var b strings.Builder
	
	// Title
	b.WriteString(m.titleStyle.Render(m.title))
	b.WriteString("\n")
	
	// Options
	for i, option := range m.options {
		var line strings.Builder
		
		// Option text with arrow for current selection
		optionText := option.Title
		
		// Apply styles based on cursor position
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
			} else {
				line.WriteString(m.optionStyle.Render("  " + optionText))
			}
		}
		
		b.WriteString(line.String())
		
		// Description
		if option.Description != "" {
			b.WriteString("\n")
			style := m.descriptionStyle
			if option.Disabled {
				style = style.Copy().Foreground(lipgloss.Color("#4A4A4A"))
			}
			b.WriteString(style.Render(option.Description))
		}
		
		// Metadata (if showing details)
		if m.showDetails && len(option.Metadata) > 0 {
			for key, value := range option.Metadata {
				b.WriteString("\n")
				style := m.metadataStyle
				if option.Disabled {
					style = style.Copy().Foreground(lipgloss.Color("#4A4A4A"))
				}
				b.WriteString(style.Render(key + ": " + value))
			}
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
func (m SelectorModel) buildHelpText() string {
	var parts []string
	
	// Navigation help
	parts = append(parts, "↑↓/jk: navigate")
	parts = append(parts, "enter/space: select")
	
	// Details toggle
	if m.hasMetadata() {
		if m.showDetails {
			parts = append(parts, "d: hide details")
		} else {
			parts = append(parts, "d: show details")
		}
	}
	
	parts = append(parts, "q/esc: cancel")
	
	return strings.Join(parts, " • ")
}

// hasMetadata checks if any option has metadata
func (m SelectorModel) hasMetadata() bool {
	for _, option := range m.options {
		if len(option.Metadata) > 0 {
			return true
		}
	}
	return false
}

// AI Provider options with metadata
var AIProviderOptions = []SelectorOption{
	{
		ID:          "openai",
		Title:       "OpenAI (GPT-4)",
		Description: "Best for code generation and complex reasoning",
		Metadata: map[string]string{
			"Models":      "GPT-4, GPT-4 Turbo, GPT-3.5 Turbo",
			"Pricing":     "Pay-per-use, ~$0.03/1K tokens",
			"Speed":       "Fast",
			"Strengths":   "Code generation, general purpose",
		},
	},
	{
		ID:          "claude",
		Title:       "Claude (Anthropic)",
		Description: "Strong reasoning and analysis capabilities",
		Metadata: map[string]string{
			"Models":      "Claude 3.5 Sonnet, Claude 3 Haiku",
			"Pricing":     "Pay-per-use, ~$0.015/1K tokens",
			"Speed":       "Fast",
			"Strengths":   "Reasoning, analysis, safety",
		},
	},
	{
		ID:          "gemini",
		Title:       "Gemini (Google)",
		Description: "Multi-modal capabilities and Google integration",
		Metadata: map[string]string{
			"Models":      "Gemini Pro, Gemini Flash",
			"Pricing":     "Free tier available, competitive pricing",
			"Speed":       "Very fast",
			"Strengths":   "Multi-modal, large context",
		},
	},
	{
		ID:          "grok",
		Title:       "Grok (xAI)",
		Description: "Fast responses with real-time information",
		Metadata: map[string]string{
			"Models":      "Grok Beta",
			"Pricing":     "Subscription-based",
			"Speed":       "Very fast",
			"Strengths":   "Real-time data, fast responses",
		},
	},
	{
		ID:          "openrouter",
		Title:       "OpenRouter (400+ Models)",
		Description: "Access to 400+ AI models through one API",
		Metadata: map[string]string{
			"Models":      "400+ models from various providers",
			"Pricing":     "Pay-per-use, varies by model",
			"Speed":       "Varies by model",
			"Strengths":   "Model diversity, competitive pricing",
		},
	},
	{
		ID:          "custom",
		Title:       "Custom (OpenAI-compatible)",
		Description: "Use any OpenAI-compatible API endpoint",
		Metadata: map[string]string{
			"Models":      "Depends on provider",
			"Pricing":     "Varies by provider",
			"Speed":       "Varies",
			"Strengths":   "Flexibility, local models",
		},
	},
}

// OAuth Provider options
var OAuthProviderOptions = []SelectorOption{
	{
		ID:          "google",
		Title:       "Google",
		Description: "Sign in with Google account",
		Metadata: map[string]string{
			"Authorization URL": "https://accounts.google.com/oauth/authorize",
			"Token URL":         "https://oauth2.googleapis.com/token",
			"Scopes":           "openid profile email",
			"Setup Required":   "Client ID and Secret from Google Console",
		},
	},
	{
		ID:          "github",
		Title:       "GitHub",
		Description: "Sign in with GitHub account",
		Metadata: map[string]string{
			"Authorization URL": "https://github.com/login/oauth/authorize",
			"Token URL":         "https://github.com/login/oauth/access_token",
			"Scopes":           "user:email",
			"Setup Required":   "OAuth App registration on GitHub",
		},
	},
	{
		ID:          "microsoft",
		Title:       "Microsoft/Azure AD",
		Description: "Sign in with Microsoft or Azure AD account",
		Metadata: map[string]string{
			"Authorization URL": "https://login.microsoftonline.com/common/oauth2/v2.0/authorize",
			"Token URL":         "https://login.microsoftonline.com/common/oauth2/v2.0/token",
			"Scopes":           "openid profile email",
			"Setup Required":   "App registration in Azure portal",
		},
	},
	{
		ID:          "facebook",
		Title:       "Facebook",
		Description: "Sign in with Facebook account",
		Metadata: map[string]string{
			"Authorization URL": "https://www.facebook.com/v18.0/dialog/oauth",
			"Token URL":         "https://graph.facebook.com/v18.0/oauth/access_token",
			"Scopes":           "email public_profile",
			"Setup Required":   "Facebook App with OAuth configured",
		},
	},
	{
		ID:          "custom",
		Title:       "Custom OAuth Provider",
		Description: "Configure a custom OAuth 2.0 provider",
		Metadata: map[string]string{
			"Authorization URL": "Custom URL required",
			"Token URL":         "Custom URL required",
			"Scopes":           "Provider-specific",
			"Setup Required":   "Provider documentation and credentials",
		},
	},
}

// Auth Type options
var AuthTypeOptions = []SelectorOption{
	{
		ID:          "none",
		Title:       "No Authentication",
		Description: "Skip authentication for this environment",
	},
	{
		ID:          "basic",
		Title:       "Basic Auth (API)",
		Description: "Username/password for API authentication",
	},
	{
		ID:          "bearer",
		Title:       "Bearer Token (API)",
		Description: "Token-based API authentication",
	},
	{
		ID:          "oauth",
		Title:       "OAuth Login",
		Description: "OAuth provider authentication (Google, GitHub, etc.)",
	},
	{
		ID:          "magic_link",
		Title:       "Magic Link",
		Description: "Email-based passwordless authentication",
	},
	{
		ID:          "username_password",
		Title:       "Username/Password Form",
		Description: "Traditional web form login",
	},
}

// RunSelector runs a selector and returns the selected option ID
func RunSelector(title string, options []SelectorOption) (string, error) {
	model := NewSelectorModel(title, options)
	
	program := tea.NewProgram(model, tea.WithAltScreen())
	result, err := program.Run()
	if err != nil {
		return "", err
	}
	
	finalModel := result.(SelectorModel)
	if finalModel.IsCancelled() {
		return "", ErrSelectionCancelled{}
	}
	
	return finalModel.GetSelected(), nil
}

// RunSelectorWithDetails runs a selector with details enabled
func RunSelectorWithDetails(title string, options []SelectorOption) (*SelectorOption, error) {
	model := NewSelectorModel(title, options).ShowDetails(true)
	
	program := tea.NewProgram(model, tea.WithAltScreen())
	result, err := program.Run()
	if err != nil {
		return nil, err
	}
	
	finalModel := result.(SelectorModel)
	if finalModel.IsCancelled() {
		return nil, ErrSelectionCancelled{}
	}
	
	return finalModel.GetSelectedOption(), nil
}

