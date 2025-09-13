package adapters

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/lance13c/tod/internal/agent/core"
	"github.com/lance13c/tod/internal/ui/components"
	"golang.org/x/term"
)

// CLIAdapter implements the UIProvider interface for CLI interactions
type CLIAdapter struct {
	reader *bufio.Reader
}

// NewCLIAdapter creates a new CLI adapter
func NewCLIAdapter() *CLIAdapter {
	return &CLIAdapter{
		reader: bufio.NewReader(os.Stdin),
	}
}

// GetInput gets input from the user with optional suggestions
func (c *CLIAdapter) GetInput(prompt string, suggestions []string) (string, error) {
	if len(suggestions) > 0 {
		// Use existing autosuggest component
		return components.RunAutoSuggestInput(
			prompt,
			"Enter value",
			"",
			suggestions,
		)
	}
	return c.askString(prompt, "")
}

// GetPassword gets password input (hidden)
func (c *CLIAdapter) GetPassword(prompt string) (string, error) {
	fmt.Print(prompt)
	
	// Hide password input
	bytePassword, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return "", err
	}
	
	fmt.Println() // Print newline after hidden input
	return string(bytePassword), nil
}

// GetSelection gets selection from options
func (c *CLIAdapter) GetSelection(prompt string, options []core.SelectOption) (string, error) {
	// Convert to components.SelectorOption
	selectorOptions := make([]components.SelectorOption, len(options))
	for i, opt := range options {
		selectorOptions[i] = components.SelectorOption{
			ID:          opt.Value,
			Title:       opt.Label,
			Description: opt.Description,
			Disabled:    opt.Disabled,
		}
	}

	selectedID, err := components.RunSelector(prompt, selectorOptions)
	if err != nil {
		return "", err
	}

	return selectedID, nil
}

// GetConfirmation gets yes/no confirmation
func (c *CLIAdapter) GetConfirmation(prompt string) (bool, error) {
	return c.askYesNo(prompt, false)
}

// ShowMessage displays a message with styling
func (c *CLIAdapter) ShowMessage(msg string, style core.MessageStyle) {
	switch style {
	case core.StyleSuccess:
		fmt.Printf("✓ %s\n", msg)
	case core.StyleError:
		fmt.Printf("✗ %s\n", msg)
	case core.StyleWarning:
		fmt.Printf("⚠ %s\n", msg)
	case core.StyleInfo:
		fmt.Printf("ℹ %s\n", msg)
	case core.StyleDebug:
		fmt.Printf("🔍 %s\n", msg)
	default:
		fmt.Println(msg)
	}
}

// ShowProgress displays progress information
func (c *CLIAdapter) ShowProgress(current, total int, message string) {
	percentage := float64(current) / float64(total) * 100
	progressBar := c.buildProgressBar(int(percentage), 30)
	fmt.Printf("\r[%s] %d/%d - %s", progressBar, current, total, message)
	
	if current == total {
		fmt.Println() // New line when complete
	}
}

// ShowError displays an error message
func (c *CLIAdapter) ShowError(err error) {
	c.ShowMessage(err.Error(), core.StyleError)
}

// ShowSuccess displays a success message
func (c *CLIAdapter) ShowSuccess(msg string) {
	c.ShowMessage(msg, core.StyleSuccess)
}

// ShowWarning displays a warning message
func (c *CLIAdapter) ShowWarning(msg string) {
	c.ShowMessage(msg, core.StyleWarning)
}

// ShowTable displays a table
func (c *CLIAdapter) ShowTable(headers []string, rows [][]string) {
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

	// Print header
	c.printTableRow(headers, widths)
	c.printTableSeparator(widths)
	
	// Print rows
	for _, row := range rows {
		c.printTableRow(row, widths)
	}
}

// ShowJSON displays JSON data
func (c *CLIAdapter) ShowJSON(data interface{}) {
	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		c.ShowError(fmt.Errorf("failed to marshal JSON: %w", err))
		return
	}
	
	fmt.Println(string(jsonBytes))
}

// ShowFlowSummary displays a flow summary
func (c *CLIAdapter) ShowFlowSummary(flow *core.Flow) {
	fmt.Printf(`
╭─────────────────────────────────────╮
│ Flow: %s
│ %s
╰─────────────────────────────────────╯

Description: %s
Category: %s
Steps: %d
Confidence: %.0f%%

`, 
		flow.Name,
		strings.Repeat(" ", max(0, 35-len(flow.Name))),
		flow.Description,
		flow.Category,
		len(flow.Steps),
		flow.Confidence*100,
	)

	if len(flow.Steps) > 0 {
		fmt.Println("Steps:")
		for i, step := range flow.Steps {
			fmt.Printf("  %d. %s - %s\n", i+1, step.Name, step.Description)
		}
		fmt.Println()
	}
}

// Helper methods

func (c *CLIAdapter) askString(prompt, defaultVal string) (string, error) {
	if defaultVal != "" {
		fmt.Printf("%s [%s]: ", prompt, defaultVal)
	} else {
		fmt.Printf("%s: ", prompt)
	}
	
	input, err := c.reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultVal, nil
	}
	
	return input, nil
}

func (c *CLIAdapter) askYesNo(prompt string, defaultVal bool) (bool, error) {
	defaultStr := "y/N"
	if defaultVal {
		defaultStr = "Y/n"
	}
	
	fmt.Printf("%s [%s]: ", prompt, defaultStr)
	
	input, err := c.reader.ReadString('\n')
	if err != nil {
		return false, err
	}
	
	input = strings.ToLower(strings.TrimSpace(input))
	
	if input == "" {
		return defaultVal, nil
	}
	
	return input == "y" || input == "yes", nil
}

func (c *CLIAdapter) buildProgressBar(percentage, width int) string {
	filled := percentage * width / 100
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	return fmt.Sprintf("%s %d%%", bar, percentage)
}

func (c *CLIAdapter) printTableRow(row []string, widths []int) {
	for i, cell := range row {
		if i < len(widths) {
			fmt.Printf("%-*s  ", widths[i], cell)
		}
	}
	fmt.Println()
}

func (c *CLIAdapter) printTableSeparator(widths []int) {
	for i, width := range widths {
		fmt.Print(strings.Repeat("-", width))
		if i < len(widths)-1 {
			fmt.Print("  ")
		}
	}
	fmt.Println()
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}