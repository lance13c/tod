package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/lance13c/tod/internal/agent/core"
	"github.com/lance13c/tod/internal/services"
	"github.com/lance13c/tod/internal/ui/adapters"
	"github.com/spf13/cobra"
)

// flowCmd represents the flow command
var flowCmd = &cobra.Command{
	Use:   "flow",
	Short: "Discover and run application flows using AI",
	Long: `Use AI to discover, analyze, and execute application flows like signup, login, and onboarding.

Tod's AI agent can understand your codebase and create interactive flows that help you test
real user journeys while automatically creating test users and validating functionality.

Examples:
  tod flow discover               # AI discovers all available flows
  tod flow signup                # AI-guided signup flow
  tod flow signup --quick        # Quick signup with AI defaults
  tod flow run onboarding        # Run a specific flow
  tod flow list                  # List discovered flows
  tod flow explain signup        # AI explains a flow`,
}

// discoverCmd discovers flows using AI
var discoverCmd = &cobra.Command{
	Use:   "discover",
	Short: "AI discovers flows in your application",
	Long: `Use AI to analyze your codebase and discover available user flows.

The AI agent will scan your project files, identify authentication endpoints,
form components, and user journey patterns to create executable flows.`,
	Run: runDiscoverFlows,
}

// signupCmd runs the signup flow
var signupCmd = &cobra.Command{
	Use:   "signup",
	Short: "AI-guided signup flow",
	Long: `Run an AI-guided signup flow that walks you through creating a real user account
in your application. The AI will discover your signup process and guide you through
each step, automatically creating a test user that you can use for further testing.`,
	Run: runSignupFlow,
}

// runFlowCmd runs a specific flow
var runFlowCmd = &cobra.Command{
	Use:   "run [flow-name]",
	Short: "Run a specific discovered flow",
	Long: `Execute a specific flow that was discovered by the AI agent.
	
Available flows can be seen with 'tod flow list'. The AI will guide you through
each step of the selected flow.`,
	Args: cobra.ExactArgs(1),
	Run:  runSpecificFlow,
}

// listFlowsCmd lists discovered flows
var listFlowsCmd = &cobra.Command{
	Use:   "list",
	Short: "List all discovered flows",
	Long:  `Display all flows that have been discovered by the AI agent.`,
	Run:   runListFlows,
}

// explainFlowCmd explains a flow
var explainFlowCmd = &cobra.Command{
	Use:   "explain [flow-name]",
	Short: "AI explains how a flow works",
	Long: `Get an AI explanation of how a specific flow works, what it does,
and what steps are involved.`,
	Args: cobra.ExactArgs(1),
	Run:  runExplainFlow,
}

func init() {
	rootCmd.AddCommand(flowCmd)
	flowCmd.AddCommand(discoverCmd)
	flowCmd.AddCommand(signupCmd)
	flowCmd.AddCommand(runFlowCmd)
	flowCmd.AddCommand(listFlowsCmd)
	flowCmd.AddCommand(explainFlowCmd)

	// discover command flags
	discoverCmd.Flags().Bool("cache", true, "Cache discovered flows")
	discoverCmd.Flags().Bool("verbose", false, "Verbose discovery output")

	// signup command flags
	signupCmd.Flags().Bool("quick", false, "Quick signup with AI defaults")
	signupCmd.Flags().String("save-as", "", "Save created user with specific ID")
	signupCmd.Flags().String("role", "user", "Role for created user")
	signupCmd.Flags().Bool("skip-save", false, "Don't save user after signup")

	// run command flags
	runFlowCmd.Flags().Bool("dry-run", false, "Show what would be executed without running")
	runFlowCmd.Flags().StringToString("vars", nil, "Variables to pass to the flow")

	// list command flags
	listFlowsCmd.Flags().String("category", "", "Filter by category")
	listFlowsCmd.Flags().Bool("json", false, "Output as JSON")
}

func runDiscoverFlows(cmd *cobra.Command, args []string) {
	// Check if project is initialized
	if todConfig == nil {
		fmt.Println("Error: Tod is not initialized in this project!")
		fmt.Println("Run 'tod init' to get started.")
		os.Exit(1)
	}

	projectDir, _ := cmd.Flags().GetString("project")
	if projectDir == "" {
		projectDir = "."
	}

	verbose, _ := cmd.Flags().GetBool("verbose")
	useCache, _ := cmd.Flags().GetBool("cache")

	// Create services
	flowService, err := services.NewFlowService(todConfig, projectDir)
	if err != nil {
		fmt.Printf("Error creating flow service: %v\n", err)
		os.Exit(1)
	}

	// Create CLI adapter
	ui := adapters.NewCLIAdapter()

	ui.ShowMessage("AI Agent analyzing your application...", core.StyleInfo)

	ctx := context.Background()
	startTime := time.Now()

	// Discover flows
	result, err := flowService.DiscoverAndCache(ctx, useCache)
	if err != nil {
		ui.ShowError(fmt.Errorf("discovery failed: %w", err))
		os.Exit(1)
	}

	duration := time.Since(startTime)

	// Display results
	ui.ShowSuccess(fmt.Sprintf("Discovery completed in %v", duration))
	fmt.Printf("\nFound %d flows:\n\n", len(result.Flows))

	if len(result.Flows) == 0 {
		ui.ShowWarning("No flows discovered. This could mean:")
		fmt.Println("  • No authentication endpoints found")
		fmt.Println("  • Application uses unsupported patterns")
		fmt.Println("  • Try running with --verbose for more details")
		return
	}

	// Display flows
	headers := []string{"Name", "Category", "Steps", "Confidence", "Description"}
	rows := make([][]string, len(result.Flows))

	for i, flow := range result.Flows {
		confidence := fmt.Sprintf("%.0f%%", flow.Confidence*100)
		steps := fmt.Sprintf("%d", len(flow.Steps))
		
		desc := flow.Description
		if len(desc) > 50 {
			desc = desc[:47] + "..."
		}

		rows[i] = []string{flow.Name, flow.Category, steps, confidence, desc}
	}

	ui.ShowTable(headers, rows)

	// Show next steps
	fmt.Printf(`
Next steps:
• tod flow signup           # Run signup flow
• tod flow run <name>       # Run specific flow  
• tod flow explain <name>   # Get AI explanation
• tod                       # Use interactive mode

`)

	if verbose {
		fmt.Printf("\nAnalyzed sources:\n")
		for _, source := range result.Sources {
			fmt.Printf("  • %s\n", source)
		}
	}
}

func runSignupFlow(cmd *cobra.Command, args []string) {
	// Check if project is initialized
	if todConfig == nil {
		fmt.Println("Error: Tod is not initialized in this project!")
		fmt.Println("Run 'tod init' to get started.")
		os.Exit(1)
	}

	projectDir, _ := cmd.Flags().GetString("project")
	if projectDir == "" {
		projectDir = "."
	}

	quick, _ := cmd.Flags().GetBool("quick")
	saveAs, _ := cmd.Flags().GetString("save-as")
	role, _ := cmd.Flags().GetString("role")
	skipSave, _ := cmd.Flags().GetBool("skip-save")

	// Create services
	flowService, err := services.NewFlowService(todConfig, projectDir)
	if err != nil {
		fmt.Printf("Error creating flow service: %v\n", err)
		os.Exit(1)
	}

	// Create CLI adapter
	ui := adapters.NewCLIAdapter()

	ui.ShowMessage("AI Agent searching for signup flow...", core.StyleInfo)

	ctx := context.Background()

	// Find signup flow
	flow, err := flowService.GetSignupFlow(ctx)
	if err != nil {
		ui.ShowError(fmt.Errorf("signup flow not found: %w", err))
		fmt.Println("\nTry running 'tod flow discover' first to find available flows.")
		os.Exit(1)
	}

	ui.ShowMessage(fmt.Sprintf("Found signup flow: %s", flow.Name), core.StyleSuccess)

	if quick {
		ui.ShowMessage("Running in quick mode with AI defaults...", core.StyleInfo)
	}

	// Execute flow
	result, err := flowService.ExecuteFlow(ctx, flow.ID, ui)
	if err != nil {
		ui.ShowError(fmt.Errorf("flow execution failed: %w", err))
		os.Exit(1)
	}

	if !result.Success {
		ui.ShowError(fmt.Errorf("signup flow failed"))
		if result.Error != nil {
			ui.ShowError(result.Error)
		}
		os.Exit(1)
	}

	// Handle user creation
	if !skipSave && result.TestUser != nil {
		if saveAs != "" {
			result.TestUser.ID = saveAs
		}
		if role != "" {
			result.TestUser.Role = role
		}

		ui.ShowSuccess(fmt.Sprintf("Test user created: %s", result.TestUser.Name))
		fmt.Printf("  • ID: %s\n", result.TestUser.ID)
		fmt.Printf("  • Email: %s\n", result.TestUser.Email)
		fmt.Printf("  • Environment: %s\n", result.TestUser.Environment)

		fmt.Println("\nNext steps:")
		fmt.Printf("• tod users list              # View your test user\n")
		fmt.Printf("• tod flow run login         # Test login with new user\n")
		fmt.Printf("• tod                        # Start interactive testing\n")
	}
}

func runSpecificFlow(cmd *cobra.Command, args []string) {
	flowName := args[0]
	
	// Check if project is initialized
	if todConfig == nil {
		fmt.Println("Error: Tod is not initialized in this project!")
		fmt.Println("Run 'tod init' to get started.")
		os.Exit(1)
	}

	projectDir, _ := cmd.Flags().GetString("project")
	if projectDir == "" {
		projectDir = "."
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	vars, _ := cmd.Flags().GetStringToString("vars")

	// Create services
	flowService, err := services.NewFlowService(todConfig, projectDir)
	if err != nil {
		fmt.Printf("Error creating flow service: %v\n", err)
		os.Exit(1)
	}

	// Create CLI adapter
	ui := adapters.NewCLIAdapter()

	ctx := context.Background()

	// Find flow by name or intent
	flow, err := flowService.FindFlowByIntent(ctx, flowName)
	if err != nil {
		ui.ShowError(fmt.Errorf("flow '%s' not found: %w", flowName, err))
		fmt.Println("\nAvailable flows:")
		runListFlows(cmd, []string{}) // Show available flows
		os.Exit(1)
	}

	if dryRun {
		ui.ShowMessage("Dry run mode - showing what would be executed:", core.StyleInfo)
		ui.ShowFlowSummary(flow)
		return
	}

	// Set variables if provided
	flowContext := &core.FlowContext{
		Environment: todConfig.Current,
		BaseURL:     todConfig.GetCurrentEnv().BaseURL,
		Config:      todConfig,
		Variables:   vars,
	}

	ui.ShowMessage(fmt.Sprintf("Running flow: %s", flow.Name), core.StyleInfo)

	// Execute flow
	result, err := flowService.ExecuteFlowWithContext(ctx, flow.ID, ui, flowContext)
	if err != nil {
		ui.ShowError(fmt.Errorf("flow execution failed: %w", err))
		os.Exit(1)
	}

	if result.Success {
		ui.ShowSuccess("Flow completed successfully!")
		fmt.Printf("Duration: %v\n", result.Duration)
		fmt.Printf("Steps executed: %d/%d\n", result.StepsRun, result.StepsTotal)
	} else {
		ui.ShowError(fmt.Errorf("flow failed"))
		if result.Error != nil {
			ui.ShowError(result.Error)
		}
	}
}

func runListFlows(cmd *cobra.Command, args []string) {
	// Check if project is initialized
	if todConfig == nil {
		fmt.Println("Error: Tod is not initialized in this project!")
		fmt.Println("Run 'tod init' to get started.")
		os.Exit(1)
	}

	projectDir, _ := cmd.Flags().GetString("project")
	if projectDir == "" {
		projectDir = "."
	}

	category, _ := cmd.Flags().GetString("category")
	jsonOutput, _ := cmd.Flags().GetBool("json")

	// Create services
	flowService, err := services.NewFlowService(todConfig, projectDir)
	if err != nil {
		fmt.Printf("Error creating flow service: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()

	// Get flows
	flows, err := flowService.GetFlows(ctx, category)
	if err != nil {
		fmt.Printf("Error getting flows: %v\n", err)
		os.Exit(1)
	}

	if len(flows) == 0 {
		fmt.Println("No flows found. Run 'tod flow discover' to discover flows.")
		return
	}

	if jsonOutput {
		ui := adapters.NewCLIAdapter()
		ui.ShowJSON(flows)
		return
	}

	// Display as table
	ui := adapters.NewCLIAdapter()
	headers := []string{"Name", "Category", "Steps", "Confidence", "Last Updated"}
	rows := make([][]string, len(flows))

	for i, flow := range flows {
		confidence := fmt.Sprintf("%.0f%%", flow.Confidence*100)
		steps := fmt.Sprintf("%d", len(flow.Steps))
		updated := flow.LastUpdated.Format("2006-01-02")

		rows[i] = []string{flow.Name, flow.Category, steps, confidence, updated}
	}

	ui.ShowTable(headers, rows)

	fmt.Printf("\nTotal: %d flows\n", len(flows))
	fmt.Println("\nUse 'tod flow run <name>' to execute a flow")
	fmt.Println("Use 'tod flow explain <name>' for details")
}

func runExplainFlow(cmd *cobra.Command, args []string) {
	flowName := args[0]
	
	// Check if project is initialized
	if todConfig == nil {
		fmt.Println("Error: Tod is not initialized in this project!")
		fmt.Println("Run 'tod init' to get started.")
		os.Exit(1)
	}

	projectDir, _ := cmd.Flags().GetString("project")
	if projectDir == "" {
		projectDir = "."
	}

	// Create services
	flowService, err := services.NewFlowService(todConfig, projectDir)
	if err != nil {
		fmt.Printf("Error creating flow service: %v\n", err)
		os.Exit(1)
	}

	ui := adapters.NewCLIAdapter()
	ctx := context.Background()

	// Find flow
	flow, err := flowService.FindFlowByIntent(ctx, flowName)
	if err != nil {
		ui.ShowError(fmt.Errorf("flow '%s' not found: %w", flowName, err))
		os.Exit(1)
	}

	// Get AI explanation
	explanation, err := flowService.ExplainFlow(ctx, flow)
	if err != nil {
		ui.ShowError(fmt.Errorf("failed to get explanation: %w", err))
		os.Exit(1)
	}

	// Display flow details
	ui.ShowFlowSummary(flow)
	fmt.Println("AI Explanation:")
	fmt.Println(strings.TrimSpace(explanation))
}