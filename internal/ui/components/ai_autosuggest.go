package components

import (
	"context"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lance13c/tod/internal/agent/core"
)

// AIAutoSuggestModel enhances the existing AutoSuggestModel with AI-powered suggestions
type AIAutoSuggestModel struct {
	*AutoSuggestModel // Embed existing component
	agent             core.FlowAgent
	cache             map[string][]string
	lastInput         string
	debounceTimer     *time.Timer
	context           map[string]interface{}
	fieldName         string
	fieldType         string
}

// NewAIAutoSuggestModel creates a new AI-powered autosuggest model
func NewAIAutoSuggestModel(title, placeholder, defaultValue string, agent core.FlowAgent) *AIAutoSuggestModel {
	base := NewAutoSuggestModel(title, placeholder, defaultValue, []string{})
	
	return &AIAutoSuggestModel{
		AutoSuggestModel: base,
		agent:           agent,
		cache:           make(map[string][]string),
		context:         make(map[string]interface{}),
	}
}

// SetField sets the field metadata for better AI suggestions
func (m *AIAutoSuggestModel) SetField(name, fieldType string) *AIAutoSuggestModel {
	m.fieldName = name
	m.fieldType = fieldType
	return m
}

// SetContext sets context data for AI suggestions
func (m *AIAutoSuggestModel) SetContext(context map[string]interface{}) *AIAutoSuggestModel {
	m.context = context
	return m
}

// Update handles updates and triggers AI suggestions
func (m *AIAutoSuggestModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// First, update the base model
	updatedModel, cmd := m.AutoSuggestModel.Update(msg)
	m.AutoSuggestModel = updatedModel.(*AutoSuggestModel)
	
	// Handle AI-specific updates
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Check if input changed
		currentInput := m.AutoSuggestModel.GetValue()
		if currentInput != m.lastInput {
			m.lastInput = currentInput
			
			// Debounce AI suggestions to avoid too many requests
			if m.debounceTimer != nil {
				m.debounceTimer.Stop()
			}
			
			m.debounceTimer = time.AfterFunc(300*time.Millisecond, func() {
				// Trigger AI suggestion update
				go m.updateAISuggestions(currentInput)
			})
		}
	
	case AISuggestionsMsg:
		// Update suggestions from AI
		m.AutoSuggestModel.SetSuggestions(msg.Suggestions)
		return m, nil
	}
	
	return m, cmd
}

// updateAISuggestions gets AI-powered suggestions for the current input
func (m *AIAutoSuggestModel) updateAISuggestions(input string) {
	if m.agent == nil {
		return
	}
	
	// Check cache first
	cacheKey := m.buildCacheKey(input)
	if cached, exists := m.cache[cacheKey]; exists {
		// Send cached suggestions
		m.sendSuggestions(cached)
		return
	}
	
	// Get AI suggestions
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	suggestions, err := m.agent.GetFieldSuggestions(ctx, m.fieldName, m.fieldType, m.context)
	if err != nil {
		// Fallback to base suggestions on error
		suggestions = m.getBaseSuggestions(input)
	}
	
	// Filter suggestions based on current input
	filtered := m.filterSuggestions(suggestions, input)
	
	// Cache the results
	m.cache[cacheKey] = filtered
	
	// Send suggestions to UI
	m.sendSuggestions(filtered)
}

// sendSuggestions sends suggestions to the UI via message passing
func (m *AIAutoSuggestModel) sendSuggestions(suggestions []string) {
	// This would send a message to update suggestions
	// For now, directly update (in real implementation, this would use tea.Program.Send)
	m.AutoSuggestModel.SetSuggestions(suggestions)
}

// filterSuggestions filters suggestions based on current input
func (m *AIAutoSuggestModel) filterSuggestions(suggestions []string, input string) []string {
	if input == "" {
		return suggestions
	}
	
	inputLower := strings.ToLower(input)
	var filtered []string
	
	for _, suggestion := range suggestions {
		suggestionLower := strings.ToLower(suggestion)
		
		// Include suggestions that:
		// 1. Start with the input
		// 2. Contain the input
		// 3. Are similar to the input
		if strings.HasPrefix(suggestionLower, inputLower) ||
		   strings.Contains(suggestionLower, inputLower) ||
		   m.isSimilar(inputLower, suggestionLower) {
			filtered = append(filtered, suggestion)
		}
	}
	
	// Limit to reasonable number of suggestions
	if len(filtered) > 10 {
		filtered = filtered[:10]
	}
	
	return filtered
}

// getBaseSuggestions provides fallback suggestions when AI is unavailable
func (m *AIAutoSuggestModel) getBaseSuggestions(input string) []string {
	suggestions := []string{}
	
	switch m.fieldType {
	case "email":
		suggestions = []string{
			"test@example.com",
			"user@test.com",
			"demo@company.com",
		}
		if input != "" && !strings.Contains(input, "@") {
			suggestions = append([]string{input + "@example.com", input + "@test.com"}, suggestions...)
		}
	
	case "password":
		suggestions = []string{
			"Test123!",
			"SecurePass123",
			"Demo12345",
		}
	
	case "name":
		suggestions = []string{
			"Test User",
			"Demo Account",
			"QA Tester",
		}
		if input != "" {
			suggestions = append([]string{input + " User", "Test " + input}, suggestions...)
		}
	
	case "username":
		suggestions = []string{
			"testuser",
			"demo_user",
		}
		if input != "" {
			suggestions = append([]string{
				strings.ToLower(input),
				strings.ToLower(input) + "_test",
			}, suggestions...)
		}
	
	case "url":
		suggestions = []string{
			"https://example.com",
			"http://localhost:3000",
			"https://test.company.com",
		}
	
	case "phone":
		suggestions = []string{
			"+1 (555) 123-4567",
			"555-0123",
			"+44 20 7123 4567",
		}
	
	default:
		// Generic suggestions based on field name
		if strings.Contains(strings.ToLower(m.fieldName), "company") {
			suggestions = []string{"Acme Corp", "Test Company", "Demo Inc"}
		} else if strings.Contains(strings.ToLower(m.fieldName), "city") {
			suggestions = []string{"New York", "London", "San Francisco"}
		}
	}
	
	return suggestions
}

// buildCacheKey creates a cache key for suggestions
func (m *AIAutoSuggestModel) buildCacheKey(input string) string {
	return strings.ToLower(m.fieldType + ":" + m.fieldName + ":" + input)
}

// isSimilar checks if two strings are similar (basic similarity)
func (m *AIAutoSuggestModel) isSimilar(a, b string) bool {
	if len(a) == 0 || len(b) == 0 {
		return false
	}
	
	// Simple similarity check: if they share a significant portion
	shorter, longer := a, b
	if len(b) < len(a) {
		shorter, longer = b, a
	}
	
	// Check if shorter string is mostly contained in longer
	matches := 0
	for i := 0; i < len(shorter); i++ {
		if i < len(longer) && shorter[i] == longer[i] {
			matches++
		}
	}
	
	similarity := float64(matches) / float64(len(shorter))
	return similarity > 0.6 // 60% similarity threshold
}

// ClearCache clears the suggestion cache
func (m *AIAutoSuggestModel) ClearCache() {
	m.cache = make(map[string][]string)
}

// GetCacheStats returns cache statistics
func (m *AIAutoSuggestModel) GetCacheStats() (int, int) {
	totalEntries := len(m.cache)
	totalSuggestions := 0
	for _, suggestions := range m.cache {
		totalSuggestions += len(suggestions)
	}
	return totalEntries, totalSuggestions
}

// Message types for AI suggestions

// AISuggestionsMsg carries AI-generated suggestions
type AISuggestionsMsg struct {
	Suggestions []string
}

// AIErrorMsg carries errors from AI suggestion requests
type AIErrorMsg struct {
	Error error
}

// Enhanced helper functions

// RunAIAutoSuggestInput runs an AI-powered autosuggest input
func RunAIAutoSuggestInput(title, placeholder, defaultValue string, agent core.FlowAgent, fieldName, fieldType string) (string, error) {
	model := NewAIAutoSuggestModel(title, placeholder, defaultValue, agent)
	model.SetField(fieldName, fieldType)
	
	program := tea.NewProgram(model, tea.WithAltScreen())
	result, err := program.Run()
	if err != nil {
		return "", err
	}
	
	finalModel := result.(*AIAutoSuggestModel)
	if finalModel.AutoSuggestModel.IsCancelled() {
		return "", ErrSelectionCancelled{}
	}
	
	return finalModel.AutoSuggestModel.GetValue(), nil
}

// RunAIAutoSuggestInputWithContext runs AI autosuggest with additional context
func RunAIAutoSuggestInputWithContext(
	title, placeholder, defaultValue string,
	agent core.FlowAgent,
	fieldName, fieldType string,
	context map[string]interface{},
) (string, error) {
	model := NewAIAutoSuggestModel(title, placeholder, defaultValue, agent)
	model.SetField(fieldName, fieldType).SetContext(context)
	
	program := tea.NewProgram(model, tea.WithAltScreen())
	result, err := program.Run()
	if err != nil {
		return "", err
	}
	
	finalModel := result.(*AIAutoSuggestModel)
	if finalModel.AutoSuggestModel.IsCancelled() {
		return "", ErrSelectionCancelled{}
	}
	
	return finalModel.AutoSuggestModel.GetValue(), nil
}