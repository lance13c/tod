package testing

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ciciliostudio/tod/internal/llm"
	"github.com/ciciliostudio/tod/internal/ui/components"
)

// FrameworkSelector provides interactive framework selection
type FrameworkSelector struct {
	projectRoot string
	detector    *FrameworkDetector
	llmClient   llm.Client
}


// Common framework options
var CommonFrameworks = []components.CheckboxOption{
	{
		ID:          "playwright",
		Title:       "Playwright",
		Description: "Microsoft's modern E2E testing framework (JavaScript/TypeScript)",
	},
	{
		ID:          "cypress",
		Title:       "Cypress",
		Description: "Popular E2E testing framework with great developer experience",
	},
	{
		ID:          "selenium",
		Title:       "Selenium WebDriver",
		Description: "Industry standard WebDriver-based testing (Python/Java/JS)",
	},
	{
		ID:          "puppeteer",
		Title:       "Puppeteer",
		Description: "Google's headless Chrome automation library",
	},
	{
		ID:          "webdriverio",
		Title:       "WebdriverIO",
		Description: "Next-gen browser automation framework",
	},
	{
		ID:          "testcafe",
		Title:       "TestCafe",
		Description: "End-to-end testing made simple",
	},
	{
		ID:          "nightwatch",
		Title:       "Nightwatch.js",
		Description: "Simple & powerful testing framework",
	},
	{
		ID:          "custom",
		Title:       "Custom/Other",
		Description: "Define your own testing framework",
	},
}

// NewFrameworkSelector creates a new framework selector
func NewFrameworkSelector(projectRoot string, client llm.Client) *FrameworkSelector {
	return &FrameworkSelector{
		projectRoot: projectRoot,
		detector:    NewFrameworkDetector(projectRoot),
		llmClient:   client,
	}
}

// SelectFramework provides interactive framework selection
func (fs *FrameworkSelector) SelectFramework() (*E2EFramework, error) {
	// Try auto-detection first
	if framework, err := fs.detector.DetectFramework(); err == nil {
		fmt.Printf("✅ Auto-detected: %s\n", framework.DisplayName)
		
		// Ask if user wants to use detected framework
		options := []components.SelectorOption{
			{
				ID:          "use",
				Title:       "Use detected framework",
				Description: "Continue with " + framework.DisplayName,
			},
			{
				ID:          "choose",
				Title:       "Choose different framework",
				Description: "Select from available frameworks",
			},
		}
		
		choice, err := components.RunSelector("✅ Framework detected! What would you like to do?", options)
		if err != nil {
			return nil, err
		}
		
		if choice == "use" {
			// Ensure test directories exist
			err := fs.detector.EnsureTestDirectories(framework, true)
			if err != nil {
				return nil, fmt.Errorf("failed to create test directories: %w", err)
			}
			return framework, nil
		}
	}

	// Framework not detected or user chose to select different one
	fmt.Println("🧪 Select your E2E testing framework:")
	fmt.Println()

	// Run interactive selection using checkbox component
	selectedIDs, err := components.RunCheckboxSelection(
		"🧪 Select your E2E testing framework:",
		CommonFrameworks,
	)
	if err != nil {
		return nil, err
	}

	if len(selectedIDs) == 0 {
		return nil, fmt.Errorf("no framework selected")
	}

	// For now, take the first selected framework (we can support multi later)
	selectedID := selectedIDs[0]

	// Handle custom framework
	if selectedID == "custom" {
		return fs.setupCustomFramework()
	}

	// Setup selected framework
	return fs.setupFrameworkByID(selectedID)
}

// setupFrameworkByID configures a framework by its ID
func (fs *FrameworkSelector) setupFrameworkByID(id string) (*E2EFramework, error) {
	// Find the framework option by ID
	var displayName string
	for _, fw := range CommonFrameworks {
		if fw.ID == id {
			displayName = fw.Title
			break
		}
	}
	
	if displayName == "" {
		return nil, fmt.Errorf("unknown framework ID: %s", id)
	}

	framework := &E2EFramework{
		Name:        id,
		DisplayName: displayName,
		Language:    fs.detector.detectLanguage(),
	}

	// Set defaults based on framework
	switch id {
	case "playwright":
		framework.RunCommand = "npx playwright test"
		framework.ConfigFile = "playwright.config.ts"
		framework.TestDir = "tests"
		framework.Extensions = []string{".spec.ts", ".test.ts"}

	case "cypress":
		framework.RunCommand = "npx cypress run"
		framework.ConfigFile = "cypress.config.js"
		framework.TestDir = "cypress/e2e"
		framework.Extensions = []string{".cy.ts", ".cy.js"}

	case "selenium":
		framework.RunCommand = "npm test"
		framework.TestDir = "test"
		framework.Extensions = []string{".test.js", ".spec.js"}

	case "puppeteer":
		framework.RunCommand = "npm test"
		framework.TestDir = "test"
		framework.Extensions = []string{".test.js", ".spec.js"}

	case "webdriverio":
		framework.RunCommand = "npx wdio run"
		framework.ConfigFile = "wdio.conf.js"
		framework.TestDir = "test/specs"
		framework.Extensions = []string{".e2e.js", ".spec.js"}

	case "testcafe":
		framework.RunCommand = "npx testcafe"
		framework.TestDir = "tests"
		framework.Extensions = []string{".js", ".ts"}

	case "nightwatch":
		framework.RunCommand = "npx nightwatch"
		framework.ConfigFile = "nightwatch.conf.js"
		framework.TestDir = "tests"
		framework.Extensions = []string{".js"}
	}

	// Prompt for customization and ensure directories
	framework, err := fs.customizeFramework(framework)
	if err != nil {
		return nil, err
	}
	
	// Ensure test directories exist
	err = fs.detector.EnsureTestDirectories(framework, true)
	if err != nil {
		return nil, fmt.Errorf("failed to create test directories: %w", err)
	}
	
	return framework, nil
}

// setupCustomFramework handles custom framework configuration using LLM research
func (fs *FrameworkSelector) setupCustomFramework() (*E2EFramework, error) {
	fmt.Println("🛠️  Setting up custom framework...")

	// Get framework name from user using auto-suggest
	commonFrameworks := []string{
		"selenium-webdriver", "puppeteer-core", "webdriverio", "testcafe", 
		"nightwatch", "protractor", "codeceptjs", "detox", "appium",
	}
	frameworkName, err := components.RunAutoSuggestInput(
		"Framework name:",
		"Enter framework name (e.g., selenium, puppeteer)",
		"",
		commonFrameworks,
	)
	if err != nil || frameworkName == "" {
		return nil, fmt.Errorf("framework name is required")
	}

	// Get version (optional)
	versionSuggestions := []string{"latest", "1.0.0", "^1.0.0", "~1.0.0"}
	version, err := components.RunAutoSuggestInput(
		"Framework version:",
		"Enter version or 'latest'",
		"latest",
		versionSuggestions,
	)
	if err != nil {
		version = "latest"
	}

	// Research the framework using LLM
	fmt.Printf("🔍 Researching %s framework...\n", frameworkName)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	research, err := fs.llmClient.ResearchFramework(ctx, frameworkName, version)
	if err != nil {
		fmt.Printf("❌ Failed to research framework: %v\n", err)
		return fs.setupManualCustomFramework(frameworkName, version)
	}

	if research.Confidence < 0.5 {
		fmt.Printf("⚠️  Low confidence in framework research (%.1f). Please verify the configuration.\n", research.Confidence)
		if research.Notes != "" {
			fmt.Printf("📝 Notes: %s\n", research.Notes)
		}
	} else {
		fmt.Printf("✅ Successfully researched %s framework (confidence: %.1f)\n", research.DisplayName, research.Confidence)
	}

	// Convert research to framework
	framework := &E2EFramework{
		Name:         research.Name,
		DisplayName:  research.DisplayName,
		Version:      research.Version,
		Language:     research.Language,
		RunCommand:   research.RunCommand,
		ConfigFile:   research.ConfigFile,
		TestDir:      research.TestDir,
		Extensions:   research.Extensions,
		Custom:       true,
		CustomConfig: map[string]string{
			"install_steps":  strings.Join(research.InstallSteps, "\n"),
			"example_test":   research.ExampleTest,
			"documentation":  research.Documentation,
			"notes":          research.Notes,
		},
	}

	// Show research results and ask for confirmation
	fmt.Printf("\n📋 Framework Configuration:\n")
	fmt.Printf("  Name: %s\n", framework.DisplayName)
	fmt.Printf("  Language: %s\n", framework.Language)
	fmt.Printf("  Run Command: %s\n", framework.RunCommand)
	fmt.Printf("  Test Directory: %s\n", framework.TestDir)
	fmt.Printf("  Config File: %s\n", framework.ConfigFile)
	fmt.Printf("  Extensions: %v\n", framework.Extensions)
	
	if len(research.InstallSteps) > 0 {
		fmt.Printf("\n📦 Installation Steps:\n")
		for i, step := range research.InstallSteps {
			fmt.Printf("  %d. %s\n", i+1, step)
		}
	}

	// Ask if user wants to customize using selector
	options := []components.SelectorOption{
		{
			ID:          "use",
			Title:       "Use this configuration",
			Description: "Continue with the researched framework settings",
		},
		{
			ID:          "customize",
			Title:       "Customize settings",
			Description: "Modify the framework configuration",
		},
	}
	
	choice, err := components.RunSelector("🤔 What would you like to do?", options)
	if err != nil {
		return nil, err
	}
	
	if choice == "customize" {
		return fs.customizeFramework(framework)
	}

	// Ensure test directories exist
	err = fs.detector.EnsureTestDirectories(framework, true)
	if err != nil {
		return nil, fmt.Errorf("failed to create test directories: %w", err)
	}

	return framework, nil
}

// setupManualCustomFramework handles manual setup when LLM research fails
func (fs *FrameworkSelector) setupManualCustomFramework(frameworkName, version string) (*E2EFramework, error) {
	fmt.Println("⚙️  Setting up framework manually...")

	framework := &E2EFramework{
		Name:         strings.ToLower(frameworkName),
		DisplayName:  frameworkName,
		Version:      version,
		Language:     fs.detector.detectLanguage(),
		Custom:       true,
		CustomConfig: make(map[string]string),
	}

	// Prompt for framework details manually using auto-suggest
	var err error
	
	framework.RunCommand, err = components.RunAutoSuggestInput(
		"Test run command:",
		"Command to run tests",
		"npm test",
		components.TestCommandSuggestions,
	)
	if err != nil {
		return nil, err
	}

	framework.TestDir, err = components.RunAutoSuggestInput(
		"Test directory:",
		"Directory containing test files",
		"tests",
		components.TestDirectorySuggestions,
	)
	if err != nil {
		return nil, err
	}

	extensions, err := components.RunAutoSuggestInput(
		"Test file extensions:",
		"File patterns for test files (comma-separated)",
		".test.js,.spec.js",
		components.FileExtensionSuggestions,
	)
	if err != nil {
		return nil, err
	}
	framework.Extensions = strings.Split(strings.ReplaceAll(extensions, " ", ""), ",")

	configFile, err := components.RunAutoSuggestInput(
		"Config file (optional):",
		"Framework configuration file",
		"",
		[]string{frameworkName + ".config.js", frameworkName + ".config.ts", "config.json", "config.yaml"},
	)
	if err != nil {
		return nil, err
	}
	framework.ConfigFile = configFile

	// Ensure test directories exist
	err = fs.detector.EnsureTestDirectories(framework, true)
	if err != nil {
		return nil, fmt.Errorf("failed to create test directories: %w", err)
	}

	return framework, nil
}

// customizeFramework allows user to customize detected/selected framework
func (fs *FrameworkSelector) customizeFramework(framework *E2EFramework) (*E2EFramework, error) {
	fmt.Printf("\n⚙️  Configuring %s...\n", framework.DisplayName)

	// Show current settings and allow customization
	fmt.Printf("Current settings:\n")
	fmt.Printf("  Run command: %s\n", framework.RunCommand)
	fmt.Printf("  Test directory: %s\n", framework.TestDir)
	if framework.ConfigFile != "" {
		fmt.Printf("  Config file: %s\n", framework.ConfigFile)
	}

	// Ask if user wants to customize using selector
	options := []components.SelectorOption{
		{
			ID:          "keep",
			Title:       "Keep current settings",
			Description: "Use the current configuration as-is",
		},
		{
			ID:          "customize",
			Title:       "Customize settings",
			Description: "Modify the run command, test directory, etc.",
		},
	}
	
	choice, err := components.RunSelector("🤔 What would you like to do?", options)
	if err != nil {
		return framework, nil // Continue with defaults on error
	}

	if choice == "keep" {
		return framework, nil
	}

	// Customize run command
	newRunCommand, err := components.RunAutoSuggestInput(
		"Test run command:",
		"Command to run tests",
		framework.RunCommand,
		components.TestCommandSuggestions,
	)
	if err == nil {
		framework.RunCommand = newRunCommand
	}

	// Customize test directory
	newTestDir, err := components.RunAutoSuggestInput(
		"Test directory:",
		"Directory containing test files",
		framework.TestDir,
		components.TestDirectorySuggestions,
	)
	if err == nil {
		framework.TestDir = newTestDir
	}

	// Customize config file
	configPrompt := "Config file:"
	configDefault := framework.ConfigFile
	if configDefault == "" {
		configPrompt = "Config file (optional):"
	}
	
	configSuggestions := []string{
		framework.Name + ".config.js",
		framework.Name + ".config.ts", 
		"config.json",
		"config.yaml",
	}
	
	newConfigFile, err := components.RunAutoSuggestInput(
		configPrompt,
		"Framework configuration file",
		configDefault,
		configSuggestions,
	)
	if err == nil {
		framework.ConfigFile = newConfigFile
	}

	return framework, nil
}


