package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/ciciliostudio/tod/internal/types"
	"github.com/ciciliostudio/tod/internal/manifest"
	"github.com/spf13/cobra"
)

// actionsCmd represents the actions command
var actionsCmd = &cobra.Command{
	Use:   "actions",
	Short: "Manage discovered actions",
	Long:  `List, filter, and manage actions discovered in your codebase.`,
}

// listActionsCmd lists discovered actions
var listActionsCmd = &cobra.Command{
	Use:   "list",
	Short: "List all discovered actions",
	Long:  `List all actions discovered from your codebase with details.`,
	Run:   runListActions,
}

// showActionCmd shows details for a specific action
var showActionCmd = &cobra.Command{
	Use:   "show [action-id]",
	Short: "Show details for a specific action",
	Args:  cobra.ExactArgs(1),
	Run:   runShowAction,
}

func init() {
	rootCmd.AddCommand(actionsCmd)
	actionsCmd.AddCommand(listActionsCmd)
	actionsCmd.AddCommand(showActionCmd)

	// Flags for list command
	listActionsCmd.Flags().StringP("method", "m", "", "Filter by HTTP method (GET, POST, etc.)")
	listActionsCmd.Flags().StringP("path", "p", "", "Filter by path pattern")
	listActionsCmd.Flags().StringP("auth", "a", "", "Filter by auth type")
	listActionsCmd.Flags().BoolP("json", "j", false, "Output as JSON")
	listActionsCmd.Flags().BoolP("summary", "s", false, "Show summary only")
}

func runListActions(cmd *cobra.Command, args []string) {
	projectDir, _ := rootCmd.PersistentFlags().GetString("project")
	manager := manifest.NewManager(projectDir)

	// Load manifest
	manifestData, err := manager.LoadManifest()
	if err != nil {
		fmt.Printf("❌ Error loading manifest: %v\n", err)
		fmt.Println("💡 Run 'tif init' to initialize the project")
		os.Exit(1)
	}

	// Get filters
	methodFilter, _ := cmd.Flags().GetString("method")
	pathFilter, _ := cmd.Flags().GetString("path")
	authFilter, _ := cmd.Flags().GetString("auth")
	showJSON, _ := cmd.Flags().GetBool("json")
	showSummary, _ := cmd.Flags().GetBool("summary")

	// Filter actions
	actions := manifestData.Actions
	if methodFilter != "" {
		var filtered []types.CodeAction
		for _, action := range actions {
			if strings.EqualFold(action.Implementation.Method, methodFilter) {
				filtered = append(filtered, action)
			}
		}
		actions = filtered
	}

	if pathFilter != "" {
		var filtered []types.CodeAction
		for _, action := range actions {
			if strings.Contains(action.Implementation.Endpoint, pathFilter) {
				filtered = append(filtered, action)
			}
		}
		actions = filtered
	}

	if authFilter != "" {
		var filtered []types.CodeAction
		for _, action := range actions {
			// Auth is no longer a direct field - would need to check Implementation if needed
			// For now, skip auth filtering until we decide on the structure
			filtered = append(filtered, action)
		}
		actions = filtered
	}

	// Sort actions by category, then name
	sort.Slice(actions, func(i, j int) bool {
		if actions[i].Category != actions[j].Category {
			return actions[i].Category < actions[j].Category
		}
		return actions[i].Name < actions[j].Name
	})

	// Output
	if showJSON {
		outputJSON(actions)
		return
	}

	if showSummary {
		outputSummary(actions, manifestData.Project.Framework, manifestData.Project.Language)
		return
	}

	outputTable(actions)
}

func runShowAction(cmd *cobra.Command, args []string) {
	projectDir, _ := rootCmd.PersistentFlags().GetString("project")
	manager := manifest.NewManager(projectDir)

	actionID := args[0]

	// Get the specific action
	action, err := manager.GetActionByID(actionID)
	if err != nil {
		fmt.Printf("❌ Action not found: %s\n", actionID)
		fmt.Println("💡 Use 'tif actions list' to see available actions")
		os.Exit(1)
	}

	// Display detailed action information
	fmt.Printf("🎯 Action: %s\n\n", action.ID)
	fmt.Printf("  Name: %s\n", action.Name)
	fmt.Printf("  Category: %s\n", action.Category)
	fmt.Printf("  Type: %s\n", action.Type)
	fmt.Printf("  Source: %s\n", action.Implementation.SourceFile)
	
	if action.Description != "" {
		fmt.Printf("  Description: %s\n", action.Description)
	}

	if action.Implementation.Method != "" {
		fmt.Printf("  HTTP Method: %s\n", action.Implementation.Method)
	}

	if action.Implementation.Endpoint != "" {
		fmt.Printf("  Endpoint: %s\n", action.Implementation.Endpoint)
	}

	if len(action.Inputs) > 0 {
		fmt.Printf("\n📝 User Inputs:\n")
		for _, input := range action.Inputs {
			required := "optional"
			if input.Required {
				required = "required"
			}
			fmt.Printf("  • %s (%s, %s)", input.Name, input.Type, required)
			if input.Example != "" {
				fmt.Printf(" - e.g., %s", input.Example)
			}
			fmt.Println()
		}
	}

	fmt.Printf("\n📋 Expected Results:\n")
	if action.Expects.Success != "" {
		fmt.Printf("  ✅ Success: %s\n", action.Expects.Success)
	}
	if action.Expects.Failure != "" {
		fmt.Printf("  ❌ Failure: %s\n", action.Expects.Failure)
	}
	if len(action.Expects.Validates) > 0 {
		fmt.Printf("  🔍 Validates:\n")
		for _, validation := range action.Expects.Validates {
			fmt.Printf("    • %s\n", validation)
		}
	}


	fmt.Printf("\n🕒 Last Modified: %s\n", action.LastModified.Format("2006-01-02 15:04:05"))
}

func outputTable(actions []types.CodeAction) {
	if len(actions) == 0 {
		fmt.Println("📭 No actions found matching the criteria")
		return
	}

	fmt.Printf("🎯 Found %d action(s):\n\n", len(actions))

	// Create table writer
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', tabwriter.TabIndent)
	
	// Header
	fmt.Fprintln(w, "ID\tNAME\tCATEGORY\tTYPE\tSOURCE")
	fmt.Fprintln(w, "──\t────\t────────\t────\t──────")

	// Rows
	for _, action := range actions {
		source := action.Implementation.SourceFile
		if len(source) > 40 {
			source = "..." + source[len(source)-37:]
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			action.ID,
			action.Name,
			action.Category,
			action.Type,
			source,
		)
	}

	w.Flush()

	fmt.Printf("\n💡 Use 'tif actions show <id>' for detailed information\n")
}

func outputSummary(actions []types.CodeAction, framework, language string) {
	fmt.Printf("📊 Actions Summary\n\n")
	fmt.Printf("Project: %s (%s)\n", framework, language)
	fmt.Printf("Total Actions: %d\n\n", len(actions))

	if len(actions) == 0 {
		return
	}

	// Count by method
	methodCounts := make(map[string]int)
	authCounts := make(map[string]int)
	pathCounts := make(map[string]int)

	for _, action := range actions {
		if action.Implementation.Method != "" {
			methodCounts[action.Implementation.Method]++
		}
		authCounts["N/A"]++ // Auth info not available in new structure
		
		// Group by endpoint prefix
		pathPrefix := getPathPrefix(action.Implementation.Endpoint)
		pathCounts[pathPrefix]++
	}

	// Display method breakdown
	fmt.Println("HTTP Methods:")
	for method, count := range methodCounts {
		fmt.Printf("  %s: %d\n", method, count)
	}

	fmt.Println("\nAuthentication:")
	for auth, count := range authCounts {
		fmt.Printf("  %s: %d\n", auth, count)
	}

	fmt.Println("\nPath Prefixes:")
	for path, count := range pathCounts {
		fmt.Printf("  %s: %d\n", path, count)
	}
}

func outputJSON(actions []types.CodeAction) {
	// This would output JSON - implementing basic version
	fmt.Println("[")
	for i, action := range actions {
		fmt.Printf("  {\n")
		fmt.Printf("    \"id\": \"%s\",\n", action.ID)
		fmt.Printf("    \"name\": \"%s\",\n", action.Name)
		fmt.Printf("    \"category\": \"%s\",\n", action.Category)
		fmt.Printf("    \"type\": \"%s\",\n", action.Type)
		fmt.Printf("    \"description\": \"%s\",\n", action.Description)
		fmt.Printf("    \"source\": \"%s\"\n", action.Implementation.SourceFile)
		if i < len(actions)-1 {
			fmt.Printf("  },\n")
		} else {
			fmt.Printf("  }\n")
		}
	}
	fmt.Println("]")
}

func getPathPrefix(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) >= 3 {
		return "/" + parts[1] + "/" + parts[2]
	} else if len(parts) >= 2 {
		return "/" + parts[1]
	}
	return path
}