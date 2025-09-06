package views

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ciciliostudio/tod/internal/agent/core"
	"github.com/ciciliostudio/tod/internal/config"
	"github.com/ciciliostudio/tod/internal/services"
	"github.com/ciciliostudio/tod/internal/ui/adapters"
)

// FlowViewState represents the current state of the flow view
type FlowViewState int

const (
	FlowStateLoading FlowViewState = iota
	FlowStateDiscovery
	FlowStateSelection
	FlowStateExecution
	FlowStateResults
	FlowStateError
)

// FlowView handles the AI flow discovery and execution interface
type FlowView struct {
	state       FlowViewState
	config      *config.Config
	flowService *services.FlowService
	adapter     *adapters.TUIAdapter
	
	// UI components
	flowList    list.Model
	width       int
	height      int
	
	// Data
	flows       []core.Flow
	currentFlow *core.Flow
	result      *core.ExecutionResult
	error       error
	
	// Loading state
	loadingMsg  string
	loadingDots int
	
	// Styles
	styles FlowViewStyles
}

// FlowViewStyles holds styling for the flow view
type FlowViewStyles struct {
	Title       lipgloss.Style
	Subtitle    lipgloss.Style
	Loading     lipgloss.Style
	Error       lipgloss.Style
	Success     lipgloss.Style
	FlowItem    lipgloss.Style
	Selected    lipgloss.Style
	Border      lipgloss.Style
	Progress    lipgloss.Style
	Help        lipgloss.Style
}

// FlowItem implements list.Item for flows
type FlowItem struct {
	flow core.Flow
}

func (f FlowItem) Title() string       { return f.flow.Name }
func (f FlowItem) Description() string { 
	confidence := fmt.Sprintf("%.0f%%", f.flow.Confidence*100)
	steps := fmt.Sprintf("%d steps", len(f.flow.Steps))
	return fmt.Sprintf("%s | %s | %s", f.flow.Category, steps, confidence)
}
func (f FlowItem) FilterValue() string { return f.flow.Name + " " + f.flow.Description }

// NewFlowView creates a new flow view
func NewFlowView(cfg *config.Config, projectRoot string) (*FlowView, error) {
	// Create flow service
	flowService, err := services.NewFlowService(cfg, projectRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to create flow service: %w", err)
	}

	// Create TUI adapter
	adapter := adapters.NewTUIAdapter()

	// Create list model
	flowList := list.New([]list.Item{}, NewFlowDelegate(), 0, 0)
	flowList.Title = ""
	flowList.SetFilteringEnabled(true)
	flowList.SetShowHelp(false)

	// Create styles
	styles := FlowViewStyles{
		Title: lipgloss.NewStyle().
			Foreground(lipgloss.Color("212")).
			Bold(true).
			MarginBottom(1),
		Subtitle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("246")).
			MarginBottom(1),
		Loading: lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")).
			Italic(true),
		Error: lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true),
		Success: lipgloss.NewStyle().
			Foreground(lipgloss.Color("46")).
			Bold(true),
		FlowItem: lipgloss.NewStyle().
			Padding(1, 2).
			Margin(0, 1),
		Selected: lipgloss.NewStyle().
			Background(lipgloss.Color("62")).
			Foreground(lipgloss.Color("230")).
			Padding(1, 2).
			Margin(0, 1).
			Bold(true),
		Border: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(1),
		Progress: lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")),
		Help: lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			MarginTop(1),
	}

	return &FlowView{
		state:       FlowStateLoading,
		config:      cfg,
		flowService: flowService,
		adapter:     adapter,
		flowList:    flowList,
		styles:      styles,
		loadingMsg:  "AI Agent analyzing your application",
	}, nil
}

// Init initializes the flow view
func (v *FlowView) Init() tea.Cmd {
	return tea.Batch(
		v.discoverFlows(),
		v.loadingAnimation(),
	)
}

// Update handles updates
func (v *FlowView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
		v.flowList.SetSize(msg.Width-4, msg.Height-8)

	case tea.KeyMsg:
		return v.handleKeyMsg(msg)

	case FlowsDiscoveredMsg:
		v.flows = msg.Flows
		v.state = FlowStateSelection
		v.updateFlowList()
		return v, nil

	case FlowExecutionStartedMsg:
		v.currentFlow = msg.Flow
		v.state = FlowStateExecution
		return v, v.executeFlow(msg.Flow)

	case FlowExecutionCompletedMsg:
		v.result = msg.Result
		v.state = FlowStateResults
		return v, nil

	case FlowErrorMsg:
		v.error = msg.Error
		v.state = FlowStateError
		return v, nil

	case LoadingTickMsg:
		v.loadingDots = (v.loadingDots + 1) % 4
		if v.state == FlowStateLoading {
			cmds = append(cmds, v.loadingAnimation())
		}

	}

	// Update list model
	if v.state == FlowStateSelection {
		var cmd tea.Cmd
		v.flowList, cmd = v.flowList.Update(msg)
		cmds = append(cmds, cmd)
	}

	return v, tea.Batch(cmds...)
}

// View renders the flow view
func (v *FlowView) View() string {
	switch v.state {
	case FlowStateLoading:
		return v.renderLoading()
	case FlowStateDiscovery:
		return v.renderDiscovery()
	case FlowStateSelection:
		return v.renderSelection()
	case FlowStateExecution:
		return v.renderExecution()
	case FlowStateResults:
		return v.renderResults()
	case FlowStateError:
		return v.renderError()
	default:
		return "Unknown state"
	}
}

// Render methods

func (v *FlowView) renderLoading() string {
	dots := strings.Repeat(".", v.loadingDots)
	spaces := strings.Repeat(" ", 3-v.loadingDots)
	
	content := fmt.Sprintf(`
%s

%s%s%s

Discovering flows in your application...
• Analyzing authentication endpoints
• Scanning form components  
• Identifying user journeys
• Building executable flows

This may take a moment...
`,
		v.styles.Title.Render("AI Flow Discovery"),
		v.styles.Loading.Render(v.loadingMsg),
		dots,
		spaces,
	)
	
	return v.styles.Border.Render(content)
}

func (v *FlowView) renderDiscovery() string {
	return v.styles.Border.Render("Discovery in progress...")
}

func (v *FlowView) renderSelection() string {
	if len(v.flows) == 0 {
		return v.renderNoFlows()
	}

	header := v.styles.Title.Render("Discovered Flows")
	subtitle := v.styles.Subtitle.Render(fmt.Sprintf("Found %d flows", len(v.flows)))
	
	content := header + "\n" + subtitle + "\n\n" + v.flowList.View()
	
	help := v.styles.Help.Render("↑↓: navigate • enter: select flow • /: search • q: back")
	
	return content + "\n" + help
}

func (v *FlowView) renderNoFlows() string {
	content := fmt.Sprintf(`
%s

%s

No flows were discovered in your application.

This could mean:
• No authentication endpoints found
• Application uses patterns not yet supported
• Code is not in expected locations

Suggestions:
• Check if your app is running
• Verify authentication routes exist
• Try running 'tod init' again

%s
`,
		v.styles.Title.Render("No Flows Found"),
		v.styles.Error.Render("AI analysis complete, but no flows detected"),
		v.styles.Help.Render("Press 'q' to go back"),
	)
	
	return v.styles.Border.Render(content)
}

func (v *FlowView) renderExecution() string {
	if v.currentFlow == nil {
		return "Execution error: no flow selected"
	}

	content := fmt.Sprintf(`
%s

%s

Executing: %s

Steps:
`,
		v.styles.Title.Render("Flow Execution"),
		v.styles.Progress.Render("AI Agent is running the flow..."),
		v.currentFlow.Name,
	)

	// Show steps with status
	for i, step := range v.currentFlow.Steps {
		status := "⏸"  // Pending
		if i == 0 {
			status = "⏳" // Currently running
		}
		content += fmt.Sprintf("  %s %d. %s\n", status, i+1, step.Name)
	}

	content += "\n" + v.styles.Help.Render("Flow is running... please wait")

	return v.styles.Border.Render(content)
}

func (v *FlowView) renderResults() string {
	if v.result == nil {
		return "No results available"
	}

	title := "Flow Results"
	if v.result.Success {
		title = v.styles.Success.Render("✓ Flow Completed Successfully")
	} else {
		title = v.styles.Error.Render("✗ Flow Failed")
	}

	content := fmt.Sprintf(`
%s

Flow: %s
Duration: %v
Steps: %d/%d completed

`,
		title,
		v.result.Flow.Name,
		v.result.Duration,
		v.result.StepsRun,
		v.result.StepsTotal,
	)

	if v.result.TestUser != nil {
		content += fmt.Sprintf(`
Test User Created:
• Name: %s
• Email: %s
• Environment: %s

`,
			v.result.TestUser.Name,
			v.result.TestUser.Email,
			v.result.TestUser.Environment,
		)
	}

	if v.result.Error != nil {
		content += v.styles.Error.Render(fmt.Sprintf("Error: %v", v.result.Error)) + "\n"
	}

	content += v.styles.Help.Render("Press 'q' to go back • 'r' to run again")

	return v.styles.Border.Render(content)
}

func (v *FlowView) renderError() string {
	content := fmt.Sprintf(`
%s

%s

%s
`,
		v.styles.Title.Render("Flow Error"),
		v.styles.Error.Render("An error occurred during flow discovery or execution"),
		v.error.Error(),
	)

	if v.error != nil {
		content += "\n" + v.styles.Help.Render("Press 'q' to go back • 'r' to retry")
	}

	return v.styles.Border.Render(content)
}

// Event handlers

func (v *FlowView) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch v.state {
	case FlowStateSelection:
		switch msg.String() {
		case "enter":
			if selectedItem := v.flowList.SelectedItem(); selectedItem != nil {
				flowItem := selectedItem.(FlowItem)
				return v, func() tea.Msg {
					return FlowExecutionStartedMsg{Flow: &flowItem.flow}
				}
			}
		case "q", "esc":
			return v, tea.Quit
		}

	case FlowStateResults, FlowStateError:
		switch msg.String() {
		case "q", "esc":
			return v, tea.Quit
		case "r":
			// Retry/restart
			v.state = FlowStateLoading
			return v, v.discoverFlows()
		}

	default:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return v, tea.Quit
		}
	}

	return v, nil
}

// Commands

func (v *FlowView) discoverFlows() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		result, err := v.flowService.DiscoverAndCache(ctx, true)
		if err != nil {
			return FlowErrorMsg{Error: err}
		}

		return FlowsDiscoveredMsg{Flows: result.Flows}
	}
}

func (v *FlowView) executeFlow(flow *core.Flow) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		result, err := v.flowService.ExecuteFlow(ctx, flow.ID, v.adapter)
		if err != nil {
			return FlowErrorMsg{Error: err}
		}

		return FlowExecutionCompletedMsg{Result: result}
	}
}

func (v *FlowView) loadingAnimation() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
		return LoadingTickMsg{}
	})
}

// Helper methods

func (v *FlowView) updateFlowList() {
	items := make([]list.Item, len(v.flows))
	for i, flow := range v.flows {
		items[i] = FlowItem{flow: flow}
	}
	v.flowList.SetItems(items)
}

// Message types

type FlowsDiscoveredMsg struct {
	Flows []core.Flow
}

type FlowExecutionStartedMsg struct {
	Flow *core.Flow
}

type FlowExecutionCompletedMsg struct {
	Result *core.ExecutionResult
}

type FlowErrorMsg struct {
	Error error
}

type LoadingTickMsg struct{}

// Flow delegate for list styling
func NewFlowDelegate() list.DefaultDelegate {
	d := list.NewDefaultDelegate()
	d.SetHeight(3)
	d.SetSpacing(1)
	return d
}