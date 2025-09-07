package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ciciliostudio/tod/internal/config"
	"github.com/ciciliostudio/tod/internal/discovery"
	"github.com/ciciliostudio/tod/internal/manifest"
	"github.com/ciciliostudio/tod/internal/testing"
	"github.com/ciciliostudio/tod/internal/ui"
	"github.com/ciciliostudio/tod/internal/ui/components"
	"github.com/spf13/cobra"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize Tod in current project",
	Long: `Initialize Tod in the current project with interactive setup:
- Detect or configure E2E testing framework
- Set up AI provider (OpenAI, Claude, Gemini, Grok, OpenRouter, Custom)  
- Configure environments and settings
- Scan codebase for actions and create manifest
- Generate configuration files`,
	Run: runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)

	// Add flags
	initCmd.Flags().BoolP("force", "", false, "Force initialization even if .tod already exists")
	initCmd.Flags().BoolP("skip-llm", "", false, "Skip LLM analysis for complex code")
	initCmd.Flags().BoolP("non-interactive", "", false, "Skip interactive prompts and use defaults")

	// Non-interactive configuration flags
	initCmd.Flags().StringP("model", "", "gpt-5", "AI model (e.g., gpt-5, claude-4-opus, gemini-2.5-pro, or custom:provider:model-name)")
	initCmd.Flags().StringP("ai-key", "", "", "AI API key (uses TOD_AI_API_KEY env var if not provided)")
	initCmd.Flags().StringP("base-url", "", "http://localhost:3000", "Base URL for development environment")
	initCmd.Flags().StringP("framework", "", "", "E2E testing framework (auto-detect if not provided)")
	initCmd.Flags().StringP("language", "", "typescript", "Programming language")
	initCmd.Flags().StringP("test-dir", "", "tests/e2e", "Directory for test files")
}

func printTodBanner() {
	blueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#4A9EFF"))

	fmt.Println()
	fmt.Println(blueStyle.Render("████████╗██╗  ██╗███████╗         ████████╗ ██████╗ ██████╗ "))
	fmt.Println(blueStyle.Render("╚══██╔══╝██║  ██║██╔════╝         ╚══██╔══╝██╔═══██╗██╔══██╗"))
	fmt.Println(blueStyle.Render("   ██║   ███████║█████╗              ██║   ██║   ██║██║  ██║"))
	fmt.Println(blueStyle.Render("   ██║   ██╔══██║██╔══╝              ██║   ██║   ██║██║  ██║"))
	fmt.Println(blueStyle.Render("   ██║   ██║  ██║███████╗            ██║   ╚██████╔╝██████╔╝"))
	fmt.Println(blueStyle.Render("   ╚═╝   ╚═╝  ╚═╝╚══════╝            ╚═╝    ╚═════╝ ╚═════╝ "))
	fmt.Println()
	fmt.Println("            THE TOD KNOWS HE'S SUPREME")
	fmt.Println("            GIGACHAD OF E2E TESTING")
	fmt.Println("    ═══════════════════════════════════════════════")
	fmt.Println()
}

func runInit(cmd *cobra.Command, args []string) {
	// Display the epic Tod banner
	printTodBanner()

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting current directory: %v\n", err)
		os.Exit(1)
	}

	loader := config.NewLoader(cwd)

	// Load existing config if present (for --force mode)
	var existingConfig *config.Config
	isInitialized := loader.IsInitialized()
	if isInitialized {
		force, _ := cmd.Flags().GetBool("force")
		if !force {
			fmt.Println("Tod is already initialized in this project!")
			fmt.Println("Use --force to reinitialize or run 'tod' to start.")
			os.Exit(1)
		}

		// Load existing config to use as defaults
		existingConfig, err = loader.Load()
		if err != nil {
			fmt.Printf("Could not load existing config (will use defaults): %v\n", err)
			existingConfig = nil
		} else {
			fmt.Println("Using existing config values as defaults...")
		}
	}

	nonInteractive, _ := cmd.Flags().GetBool("non-interactive")

	var todConfig *config.Config
	if nonInteractive {
		todConfig = createNonInteractiveConfig(cmd, cwd, existingConfig)
	} else {
		// Use the new unified wizard instead of separate prompts
		var err error
		todConfig, err = components.RunInitWizard(cwd, existingConfig)
		if err != nil {
			fmt.Printf("Error running initialization wizard: %v\n", err)
			os.Exit(1)
		}
	}

	// Create .tod directory structure
	configPath := loader.GetConfigPath()
	err = createTodDirectory(filepath.Dir(configPath))
	if err != nil {
		fmt.Printf("Error creating .tod directory: %v\n", err)
		os.Exit(1)
	}

	// Add .tod to project's .gitignore
	err = updateProjectGitignore(cwd)
	if err != nil {
		fmt.Printf("Warning: Could not update .gitignore: %v\n", err)
		// Don't exit - this is not a critical failure
	}

	// Save configuration
	err = loader.Save(todConfig, configPath)
	if err != nil {
		fmt.Printf("Error saving configuration: %v\n", err)
		os.Exit(1)
	}

	// Interactive analysis selection (only in interactive mode)
	var results *discovery.ScanResults
	if nonInteractive {
		// In non-interactive mode, skip analysis by default
		fmt.Println("\nScanning project for actions (basic scan)...")
		var err error
		results, err = scanProjectActions(cwd, todConfig, nil)
		if err != nil {
			fmt.Printf("Error scanning project: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Skip analysis selection and go directly to navigation mode
		// Use "skip" type to bypass AI analysis
		analysisChoice := &components.AnalysisChoice{
			Type:        "skip",
			Directories: []string{},
		}

		results, err = scanProjectActions(cwd, todConfig, analysisChoice)
		if err != nil {
			fmt.Printf("Error scanning project: %v\n", err)
			os.Exit(1)
		}
	}

	// Save manifest
	manifestPath := filepath.Join(filepath.Dir(configPath), "manifest.json")
	err = manifest.SaveManifest(manifestPath, results)
	if err != nil {
		fmt.Printf("Error saving manifest: %v\n", err)
		os.Exit(1)
	}

	// Print summary
	printInitSummary(todConfig, results)

	fmt.Println("\nTod initialization complete!")
	fmt.Println("\nLaunching Tod Adventure Mode...")

	// Auto-launch TUI with Tod Adventure Mode
	time.Sleep(1 * time.Second) // Brief pause to let user see the message
	
	// Launch TUI directly to Tod Adventure Mode
	if err := launchTUIWithTodAdventure(todConfig); err != nil {
		fmt.Printf("Error launching Tod: %v\n", err)
		// Fall back to showing next steps
		printNextSteps(todConfig)
	}
}

// runInteractiveSetup is now replaced by the unified init wizard

// Helper functions still needed by other files (users.go, etc.)
func askString(reader *bufio.Reader, prompt, defaultVal string) string {
	fmt.Printf("%s", prompt)
	if defaultVal != "" {
		fmt.Printf("[%s] ", defaultVal)
	}

	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" {
		return defaultVal
	}

	return input
}

func askYesNo(reader *bufio.Reader, prompt string, defaultVal bool) bool {
	fmt.Printf("%s", prompt)

	input, _ := reader.ReadString('\n')
	input = strings.ToLower(strings.TrimSpace(input))

	if input == "" {
		return defaultVal
	}

	return input == "y" || input == "yes"
}

func askChoice(reader *bufio.Reader, prompt string, min, max int) int {
	for {
		fmt.Printf("%s", prompt)

		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		var choice int
		if _, err := fmt.Sscanf(input, "%d", &choice); err == nil {
			if choice >= min && choice <= max {
				return choice
			}
		}

		fmt.Printf("Please enter a number between %d and %d\n", min, max)
	}
}

func createNonInteractiveConfig(cmd *cobra.Command, cwd string, existingConfig *config.Config) *config.Config {
	fmt.Println("Using non-interactive mode with provided flags...")

	// Start with existing config or defaults
	var baseConfig *config.Config
	if existingConfig != nil {
		baseConfig = existingConfig
		fmt.Println("Using existing configuration as base...")
	} else {
		baseConfig = config.DefaultConfig()
	}

	// Get flag values
	modelFlag, _ := cmd.Flags().GetString("model")
	aiKey, _ := cmd.Flags().GetString("ai-key")
	baseURL, _ := cmd.Flags().GetString("base-url")
	framework, _ := cmd.Flags().GetString("framework")
	language, _ := cmd.Flags().GetString("language")
	testDir, _ := cmd.Flags().GetString("test-dir")

	// Use existing values as defaults for unspecified flags
	if modelFlag == "gpt-5" && existingConfig != nil { // Default flag value
		modelFlag = existingConfig.AI.Provider + ":" + existingConfig.AI.Model
	}
	if aiKey == "" && existingConfig != nil {
		aiKey = existingConfig.AI.APIKey
	}
	if baseURL == "http://localhost:3000" && existingConfig != nil { // Default flag value
		currentEnv := existingConfig.GetCurrentEnv()
		if currentEnv != nil {
			baseURL = currentEnv.BaseURL
		}
	}
	if framework == "" && existingConfig != nil {
		framework = existingConfig.Testing.Framework
	}
	if language == "typescript" && existingConfig != nil { // Default flag value
		language = existingConfig.Testing.Language
	}
	if testDir == "tests/e2e" && existingConfig != nil { // Default flag value
		testDir = existingConfig.Testing.TestDir
	}

	// Parse the model selection
	selectedModel, err := config.ParseModelSelection(modelFlag)
	if err != nil {
		fmt.Printf("Could not parse model '%s': %v\n", modelFlag, err)
		if existingConfig != nil {
			fmt.Printf("Using existing model: %s\n", existingConfig.AI.Model)
			selectedModel = &config.ModelInfo{
				Provider:  existingConfig.AI.Provider,
				ModelName: existingConfig.AI.Model,
			}
		} else {
			fmt.Println("Using default: gpt-5")
			selectedModel = &config.ModelInfo{
				ID:        "gpt-5",
				Provider:  "openai",
				ModelName: "gpt-5",
			}
		}
	}

	fmt.Printf("Selected model: %s\n", selectedModel.DisplayName)

	// Use TOD_AI_API_KEY env var if no key provided
	if aiKey == "" {
		aiKey = os.Getenv("TOD_AI_API_KEY")
	}

	// Auto-detect framework if not provided and not in existing config
	if framework == "" {
		fmt.Println("Auto-detecting E2E framework...")
		detector := testing.NewFrameworkDetector(cwd)
		detectedFramework, err := detector.DetectFramework()
		if err == nil && detectedFramework != nil {
			fmt.Printf("Detected: %s v%s (%s)\n", detectedFramework.DisplayName, detectedFramework.Version, detectedFramework.Language)
			framework = detectedFramework.Name
			if language == "typescript" { // Only override if still default
				language = detectedFramework.Language
			}
		} else {
			fmt.Println("No framework detected, using playwright default")
			framework = "playwright"
		}
	}

	// Build new configuration, preserving existing values
	now := time.Now()
	newConfig := &config.Config{
		AI: config.AIConfig{
			Provider: selectedModel.Provider,
			APIKey:   aiKey,
			Model:    selectedModel.ModelName,
		},
		Testing: config.TestingConfig{
			Framework: framework,
			Version:   getStringOrDefault(baseConfig.Testing.Version, "latest"),
			Language:  language,
			TestDir:   testDir,
			Command:   getStringOrDefault(baseConfig.Testing.Command, "npm test"),
			Pattern:   getStringOrDefault(baseConfig.Testing.Pattern, "*.spec.ts"),
			Template:  baseConfig.Testing.Template,
		},
		Envs:    make(map[string]config.EnvConfig),
		Current: "development",
		Meta: config.MetaConfig{
			Version:   "1.0.0",
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	// Preserve all existing environments and add/update development
	if existingConfig != nil {
		for name, env := range existingConfig.Envs {
			newConfig.Envs[name] = env
		}
		newConfig.Current = existingConfig.Current
		// Copy AI settings that weren't overridden
		if newConfig.AI.Endpoint == "" {
			newConfig.AI.Endpoint = existingConfig.AI.Endpoint
		}
		if newConfig.AI.Settings == nil {
			newConfig.AI.Settings = existingConfig.AI.Settings
		}
	}

	// Update or add development environment
	currentEnvName := newConfig.Current
	if currentEnvName == "" {
		currentEnvName = "development"
		newConfig.Current = currentEnvName
	}

	currentEnv := config.EnvConfig{
		Name:    currentEnvName,
		BaseURL: baseURL,
	}

	// Preserve existing environment settings if updating the current env
	if existingEnv, exists := newConfig.Envs[currentEnvName]; exists {
		currentEnv.Headers = existingEnv.Headers
		currentEnv.Auth = existingEnv.Auth
		currentEnv.Cookies = existingEnv.Cookies
	}

	newConfig.Envs[currentEnvName] = currentEnv

	return newConfig
}

// Helper function to get string value or default
func getStringOrDefault(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}

// launchTUIWithFlowDiscovery launches the TUI directly into Navigation mode
func launchTUIWithTodAdventure(todConfig *config.Config) error {
	// Create the main model with Navigation Mode as initial view
	model := ui.NewModelWithInitialView(todConfig, ui.ViewNavigation)

	// Create the program with some options
	program := tea.NewProgram(
		model,
		tea.WithAltScreen(),       // Use alternate screen buffer
		tea.WithMouseCellMotion(), // Enable mouse support
	)

	// Run the program
	_, err := program.Run()
	return err
}

func scanProjectActions(cwd string, todConfig *config.Config, analysisChoice *components.AnalysisChoice) (*discovery.ScanResults, error) {
	// Determine if we should skip LLM analysis
	skipLLM := analysisChoice == nil || analysisChoice.Type == "skip"

	scanner := discovery.NewScanner(cwd, discovery.ScanOptions{
		Framework: todConfig.Testing.Framework,
		Language:  todConfig.Testing.Language,
		SkipLLM:   skipLLM,
	}, todConfig)

	// If we have analysis choices, configure the scanner accordingly
	if analysisChoice != nil && analysisChoice.Type != "skip" {
		return scanner.ScanProjectWithDirectories(analysisChoice.Directories)
	}

	return scanner.ScanProject()
}

func createTodDirectory(todDir string) error {
	// Create directory structure
	dirs := []string{
		todDir,
		filepath.Join(todDir, "cache"),
		filepath.Join(todDir, "sessions"),
		filepath.Join(todDir, "generated"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	// Create .gitignore for .tod directory
	gitignore := filepath.Join(todDir, ".gitignore")
	gitignoreContent := `# Tod cache and temporary files
cache/
sessions/
*.log
*.tmp

# Keep important files
!manifest.json
!config.yaml
!generated/
`
	return os.WriteFile(gitignore, []byte(gitignoreContent), 0644)
}

func printInitSummary(todConfig *config.Config, results *discovery.ScanResults) {
	fmt.Println("\nConfiguration Summary:")
	fmt.Printf("   • AI Provider: %s (%s)\n", todConfig.AI.Provider, todConfig.AI.Model)
	fmt.Printf("   • Test Framework: %s v%s\n", todConfig.Testing.Framework, todConfig.Testing.Version)
	fmt.Printf("   • Language: %s\n", todConfig.Testing.Language)
	fmt.Printf("   • Current Environment: %s\n", todConfig.Current)

	if env := todConfig.GetCurrentEnv(); env != nil {
		fmt.Printf("   • Base URL: %s\n", env.BaseURL)
	}

	fmt.Println("\nDiscovery Summary:")
	fmt.Printf("   • Framework: %s\n", results.Project.Framework)
	fmt.Printf("   • Language: %s\n", results.Project.Language)
	fmt.Printf("   • Actions: %d\n", len(results.Actions))
	fmt.Printf("   • Files scanned: %d\n", len(results.Files))

	if len(results.Actions) > 0 {
		fmt.Println("\nKey Actions Found:")
		for i, action := range results.Actions {
			if i >= 5 { // Show max 5 actions
				fmt.Printf("   ... and %d more\n", len(results.Actions)-5)
				break
			}
			fmt.Printf("   • %s: %s\n", action.Name, action.Description)
		}
	}

	if len(results.Errors) > 0 {
		fmt.Printf("\n%d warnings during scan - check logs for details\n", len(results.Errors))
	}
}

func printNextSteps(todConfig *config.Config) {
	fmt.Println("\nNext Steps:")

	// Check if authentication is configured
	currentEnv := todConfig.GetCurrentEnv()
	hasAuth := currentEnv != nil && currentEnv.Auth != nil && currentEnv.Auth.Type != "none"

	// Always suggest AI flow discovery first
	fmt.Println("   AI-Powered Flow Discovery:")
	fmt.Println("      • tod flow discover         # AI finds flows in your app")
	fmt.Println("      • tod flow signup           # AI-guided signup flow")
	fmt.Println("      • tod                       # Interactive mode with AI flows")

	if hasAuth {
		fmt.Printf("\n   Authentication detected (%s):\n", currentEnv.Auth.Type)
		fmt.Println("      • tod flow signup           # Create real test user via signup")
		fmt.Println("      • tod users create          # Manual user creation")
		fmt.Println("      • tod users create --template admin")

		fmt.Printf("\n   AI will help with %s authentication:\n", currentEnv.Auth.Type)
		switch currentEnv.Auth.Type {
		case "username_password":
			fmt.Println("      - Smart form field suggestions")
			fmt.Println("      - Auto-generated test credentials")
			fmt.Println("      - Form validation guidance")
		case "bearer":
			fmt.Println("      - API token management")
			fmt.Println("      - Authentication header setup")
		case "oauth":
			fmt.Println("      - OAuth flow navigation")
			fmt.Println("      - Provider-specific guidance")
		case "magic_link":
			fmt.Println("      - Email verification simulation")
			fmt.Println("      - Link extraction and navigation")
		case "basic":
			fmt.Println("      - HTTP basic auth setup")
			fmt.Println("      - Credential encoding")
		}

		fmt.Println("\n   Recommended flow:")
		fmt.Println("      1. tod flow discover        # Let AI find your flows")
		fmt.Println("      2. tod flow signup          # Create real test user")
		fmt.Println("      3. tod                      # Start interactive testing")

	} else {
		fmt.Println("\n   No authentication detected:")
		fmt.Println("      • AI can still discover navigation flows")
		fmt.Println("      • Create test users manually if needed")
		fmt.Println("      • Add authentication later")

		fmt.Println("\n   Recommended flow:")
		fmt.Println("      1. tod flow discover        # AI finds available flows")
		fmt.Println("      2. tod actions list         # View discovered actions")
		fmt.Println("      3. tod                      # Start interactive testing")
	}

	fmt.Println("\n   Additional commands:")
	fmt.Println("      • tod users list            # View configured users")
	fmt.Println("      • tod flow list             # View discovered flows")
	fmt.Println("      • tod flow explain <name>   # AI explains a flow")

	fmt.Printf("\nEnvironment: %s (%s)\n", todConfig.Current, currentEnv.BaseURL)
	fmt.Println("Your AI testing assistant is ready!")
}

// updateProjectGitignore adds .tod to the project's .gitignore file if not already present
func updateProjectGitignore(projectPath string) error {
	gitignorePath := filepath.Join(projectPath, ".gitignore")

	// Check if .gitignore exists
	content, err := os.ReadFile(gitignorePath)
	if err != nil {
		if os.IsNotExist(err) {
			// Create new .gitignore with just .tod
			newContent := "# Tod testing configuration\n.tod/\n"
			return os.WriteFile(gitignorePath, []byte(newContent), 0644)
		}
		return err
	}

	contentStr := string(content)

	// Check if .tod is already in gitignore (various patterns)
	todPatterns := []string{".tod", ".tod/", "/.tod", "/.tod/"}
	for _, pattern := range todPatterns {
		if strings.Contains(contentStr, pattern) {
			// Already present, no need to add
			return nil
		}
	}

	// Add .tod to the end of the file
	newEntry := "\n# Tod testing configuration\n.tod/\n"

	// Ensure we don't add extra newlines if file already ends with newline
	if !strings.HasSuffix(contentStr, "\n") {
		newEntry = "\n" + newEntry
	}

	updatedContent := contentStr + newEntry
	return os.WriteFile(gitignorePath, []byte(updatedContent), 0644)
}
