package adapters

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ciciliostudio/tod/internal/agent/core"
	"github.com/ciciliostudio/tod/internal/ui/components"
)

// TUIAdapter implements the UIProvider interface for TUI interactions
type TUIAdapter struct {
	model            tea.Model
	program          *tea.Program
	inputChan        chan string
	resultChan       chan interface{}
	progressChan     chan ProgressUpdate
	confirmationChan chan bool
	mu               sync.Mutex
	styles           TUIStyles
}

// TUIStyles holds styling for the TUI adapter
type TUIStyles struct {
	Success lipgloss.Style
	Error   lipgloss.Style
	Warning lipgloss.Style
	Info    lipgloss.Style
	Debug   lipgloss.Style
	Header  lipgloss.Style
	Border  lipgloss.Style
}

// ProgressUpdate represents a progress update
type ProgressUpdate struct {
	Current int
	Total   int
	Message string
}

// NewTUIAdapter creates a new TUI adapter
func NewTUIAdapter() *TUIAdapter {
	styles := TUIStyles{
		Success: lipgloss.NewStyle().
			Foreground(lipgloss.Color("46")).
			Bold(true),
		Error: lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true),
		Warning: lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")).
			Bold(true),
		Info: lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")).
			Bold(true),
		Debug: lipgloss.NewStyle().
			Foreground(lipgloss.Color("242")).
			Italic(true),
		Header: lipgloss.NewStyle().
			Foreground(lipgloss.Color("212")).
			Bold(true).
			BorderStyle(lipgloss.RoundedBorder()).
			Padding(0, 1),
		Border: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			Padding(1),
	}

	return &TUIAdapter{
		inputChan:        make(chan string, 1),
		resultChan:       make(chan interface{}, 1),
		progressChan:     make(chan ProgressUpdate, 10),
		confirmationChan: make(chan bool, 1),
		styles:           styles,
	}
}

// SetProgram sets the tea program for the adapter
func (t *TUIAdapter) SetProgram(program *tea.Program) {
	t.program = program
}

// GetInput gets input from the user with optional suggestions
func (t *TUIAdapter) GetInput(prompt string, suggestions []string) (string, error) {
	if t.program == nil {
		// Fallback to simple prompt
		return t.getSimpleInput(prompt)
	}

	// Send input request to TUI
	// TODO: This would need integration with the main TUI program to use inputModel
	// inputModel := components.NewAutoSuggestModel(prompt, "", "", suggestions)
	
	// For now, return a placeholder
	return t.getSimpleInput(prompt)
}

// GetPassword gets password input (hidden)
func (t *TUIAdapter) GetPassword(prompt string) (string, error) {
	// Password input in TUI would use a special component
	return t.getSimpleInput(prompt + " (hidden)")
}

// GetSelection gets selection from options
func (t *TUIAdapter) GetSelection(prompt string, options []core.SelectOption) (string, error) {
	// Convert to selector options
	selectorOptions := make([]components.SelectorOption, len(options))
	for i, opt := range options {
		selectorOptions[i] = components.SelectorOption{
			ID:          opt.Value,
			Title:       opt.Label,
			Description: opt.Description,
			Disabled:    opt.Disabled,
		}
	}

	// This would integrate with the TUI selector
	if len(options) > 0 {
		return options[0].Value, nil // Placeholder
	}
	return "", fmt.Errorf("no options provided")
}

// GetConfirmation gets yes/no confirmation
func (t *TUIAdapter) GetConfirmation(prompt string) (bool, error) {
	if t.program != nil {
		// Send confirmation request to the TUI
		t.program.Send(ConfirmationRequestMsg{
			Prompt:   prompt,
			Response: t.confirmationChan,
		})
		
		// Wait for response
		select {
		case response := <-t.confirmationChan:
			return response, nil
		case <-time.After(30 * time.Second):
			return false, fmt.Errorf("confirmation timeout")
		}
	}
	
	// Fallback to console prompt
	fmt.Printf("%s (y/n): ", prompt)
	var input string
	_, err := fmt.Scanln(&input)
	if err != nil {
		return false, err
	}
	return strings.ToLower(input) == "y" || strings.ToLower(input) == "yes", nil
}

// ShowMessage displays a message with styling
func (t *TUIAdapter) ShowMessage(msg string, style core.MessageStyle) {
	styledMsg := t.styleMessage(msg, style)
	
	if t.program != nil {
		// Send message to TUI program
		t.program.Send(MessageDisplayMsg{Message: styledMsg, Style: style})
	} else {
		// Fallback to console output
		fmt.Println(styledMsg)
	}
}

// ShowProgress displays progress information
func (t *TUIAdapter) ShowProgress(current, total int, message string) {
	update := ProgressUpdate{
		Current: current,
		Total:   total,
		Message: message,
	}
	
	if t.program != nil {
		t.program.Send(ProgressUpdateMsg(update))
	} else {
		// Fallback progress bar
		percentage := float64(current) / float64(total) * 100
		bar := t.buildProgressBar(int(percentage), 30)
		fmt.Printf("\r[%s] %d/%d - %s", bar, current, total, message)
		if current == total {
			fmt.Println()
		}
	}
}

// ShowError displays an error message
func (t *TUIAdapter) ShowError(err error) {
	t.ShowMessage(err.Error(), core.StyleError)
}

// ShowSuccess displays a success message
func (t *TUIAdapter) ShowSuccess(msg string) {
	t.ShowMessage(msg, core.StyleSuccess)
}

// ShowWarning displays a warning message
func (t *TUIAdapter) ShowWarning(msg string) {
	t.ShowMessage(msg, core.StyleWarning)
}

// ShowTable displays a table
func (t *TUIAdapter) ShowTable(headers []string, rows [][]string) {
	tableStr := t.renderTable(headers, rows)
	
	if t.program != nil {
		t.program.Send(TableDisplayMsg{Content: tableStr})
	} else {
		fmt.Println(tableStr)
	}
}

// ShowJSON displays JSON data
func (t *TUIAdapter) ShowJSON(data interface{}) {
	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		t.ShowError(fmt.Errorf("failed to marshal JSON: %w", err))
		return
	}
	
	jsonStr := string(jsonBytes)
	if t.program != nil {
		t.program.Send(JSONDisplayMsg{Content: jsonStr})
	} else {
		fmt.Println(jsonStr)
	}
}

// ShowFlowSummary displays a flow summary
func (t *TUIAdapter) ShowFlowSummary(flow *core.Flow) {
	summary := t.renderFlowSummary(flow)
	
	if t.program != nil {
		t.program.Send(FlowSummaryMsg{Flow: flow, Content: summary})
	} else {
		fmt.Println(summary)
	}
}

// Helper methods

func (t *TUIAdapter) getSimpleInput(prompt string) (string, error) {
	// Fallback input method when TUI is not available
	fmt.Printf("%s: ", prompt)
	var input string
	_, err := fmt.Scanln(&input)
	return input, err
}

func (t *TUIAdapter) styleMessage(msg string, style core.MessageStyle) string {
	switch style {
	case core.StyleSuccess:
		return t.styles.Success.Render("✓ " + msg)
	case core.StyleError:
		return t.styles.Error.Render("✗ " + msg)
	case core.StyleWarning:
		return t.styles.Warning.Render("⚠ " + msg)
	case core.StyleInfo:
		return t.styles.Info.Render("ℹ " + msg)
	case core.StyleDebug:
		return t.styles.Debug.Render("🔍 " + msg)
	default:
		return msg
	}
}

func (t *TUIAdapter) buildProgressBar(percentage, width int) string {
	filled := percentage * width / 100
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	return fmt.Sprintf("%s %d%%", bar, percentage)
}

func (t *TUIAdapter) renderTable(headers []string, rows [][]string) string {
	// Calculate column widths
	widths := make([]int, len(headers))
	for i, header := range headers {
		widths[i] = len(header)
	}
	
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	var result strings.Builder
	
	// Header
	result.WriteString(t.styles.Header.Render("Table"))
	result.WriteString("\n\n")
	
	// Column headers
	for i, header := range headers {
		if i < len(widths) {
			result.WriteString(fmt.Sprintf("%-*s  ", widths[i], header))
		}
	}
	result.WriteString("\n")
	
	// Separator
	for i, width := range widths {
		result.WriteString(strings.Repeat("-", width))
		if i < len(widths)-1 {
			result.WriteString("  ")
		}
	}
	result.WriteString("\n")
	
	// Rows
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) {
				result.WriteString(fmt.Sprintf("%-*s  ", widths[i], cell))
			}
		}
		result.WriteString("\n")
	}
	
	return t.styles.Border.Render(result.String())
}

func (t *TUIAdapter) renderFlowSummary(flow *core.Flow) string {
	var content strings.Builder
	
	content.WriteString(t.styles.Header.Render(fmt.Sprintf("Flow: %s", flow.Name)))
	content.WriteString("\n\n")
	
	content.WriteString(fmt.Sprintf("Description: %s\n", flow.Description))
	content.WriteString(fmt.Sprintf("Category: %s\n", flow.Category))
	content.WriteString(fmt.Sprintf("Steps: %d\n", len(flow.Steps)))
	content.WriteString(fmt.Sprintf("Confidence: %.0f%%\n", flow.Confidence*100))
	
	if flow.AuthType != "" {
		content.WriteString(fmt.Sprintf("Auth Type: %s\n", flow.AuthType))
	}
	
	if len(flow.Steps) > 0 {
		content.WriteString("\nSteps:\n")
		for i, step := range flow.Steps {
			content.WriteString(fmt.Sprintf("  %d. %s - %s\n", i+1, step.Name, step.Description))
		}
	}
	
	return t.styles.Border.Render(content.String())
}

// TUI Message types for integration with the main program
type MessageDisplayMsg struct {
	Message string
	Style   core.MessageStyle
}

type ProgressUpdateMsg ProgressUpdate

type TableDisplayMsg struct {
	Content string
}

type JSONDisplayMsg struct {
	Content string
}

type FlowSummaryMsg struct {
	Flow    *core.Flow
	Content string
}

type InputRequestMsg struct {
	Prompt      string
	Suggestions []string
	Response    chan string
}

type SelectionRequestMsg struct {
	Prompt   string
	Options  []core.SelectOption
	Response chan string
}

// SendConfirmation sends a confirmation response
func (t *TUIAdapter) SendConfirmation(response bool) {
	select {
	case t.confirmationChan <- response:
		// Sent successfully
	default:
		// Channel full or not waiting for confirmation
	}
}

type ConfirmationRequestMsg struct {
	Prompt   string
	Response chan bool
}