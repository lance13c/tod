package testing

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/lance13c/tod/internal/types"
	"github.com/lance13c/tod/internal/llm"
)

// TestGenerator generates E2E tests using LLM for any framework
type TestGenerator struct {
	projectRoot string
	llmClient   llm.Client
}

// NewTestGenerator creates a new test generator
func NewTestGenerator(projectRoot string, client llm.Client) *TestGenerator {
	return &TestGenerator{
		projectRoot: projectRoot,
		llmClient:   client,
	}
}

// GenerationOptions configures test generation
type GenerationOptions struct {
	Framework     *E2EFramework
	Actions       []types.CodeAction
	FlowName      string
	OutputDir     string
	TestStyle     string // "adventure", "standard", "minimal"
	IncludeSetup  bool
	IncludeAuth   bool
}

// GeneratedTest represents a generated test file
type GeneratedTest struct {
	Framework    string    `json:"framework"`
	FileName     string    `json:"file_name"`
	FilePath     string    `json:"file_path"`
	Content      string    `json:"content"`
	Actions      []string  `json:"actions"`      // Action IDs included
	FlowName     string    `json:"flow_name"`
	Language     string    `json:"language"`
	GeneratedAt  time.Time `json:"generated_at"`
}

// TestGenerationResult contains all generated test files
type TestGenerationResult struct {
	Tests       []GeneratedTest `json:"tests"`
	SetupFile   *GeneratedTest  `json:"setup_file,omitempty"`
	ConfigFile  *GeneratedTest  `json:"config_file,omitempty"`
	TotalFiles  int             `json:"total_files"`
	Framework   string          `json:"framework"`
	GeneratedAt time.Time       `json:"generated_at"`
}

// GenerateTests creates E2E tests for the given actions and framework
func (g *TestGenerator) GenerateTests(ctx context.Context, options GenerationOptions) (*TestGenerationResult, error) {
	fmt.Printf("🧪 Generating %s tests for %d actions...\n", options.Framework.DisplayName, len(options.Actions))

	// Create generation context
	genCtx := &testGenerationContext{
		framework:   options.Framework,
		actions:     options.Actions,
		projectRoot: g.projectRoot,
		outputDir:   options.OutputDir,
		style:       options.TestStyle,
	}

	result := &TestGenerationResult{
		Tests:       []GeneratedTest{},
		Framework:   options.Framework.Name,
		GeneratedAt: time.Now(),
	}

	// Generate main test file
	mainTest, err := g.generateMainTestFile(ctx, genCtx, options.FlowName)
	if err != nil {
		return nil, fmt.Errorf("failed to generate main test: %w", err)
	}
	result.Tests = append(result.Tests, *mainTest)

	// Generate setup file if needed
	if options.IncludeSetup {
		setupFile, err := g.generateSetupFile(ctx, genCtx)
		if err != nil {
			fmt.Printf("⚠️ Warning: failed to generate setup file: %v\n", err)
		} else {
			result.SetupFile = setupFile
		}
	}

	// Generate config file if needed
	if options.Framework.ConfigFile != "" {
		configFile, err := g.generateConfigFile(ctx, genCtx)
		if err != nil {
			fmt.Printf("⚠️ Warning: failed to generate config file: %v\n", err)
		} else {
			result.ConfigFile = configFile
		}
	}

	result.TotalFiles = len(result.Tests)
	if result.SetupFile != nil {
		result.TotalFiles++
	}
	if result.ConfigFile != nil {
		result.TotalFiles++
	}

	fmt.Printf("✅ Generated %d test files for %s\n", result.TotalFiles, options.Framework.DisplayName)
	return result, nil
}

// testGenerationContext holds context for test generation
type testGenerationContext struct {
	framework   *E2EFramework
	actions     []types.CodeAction
	projectRoot string
	outputDir   string
	style       string
}

// generateMainTestFile creates the main test file with all actions
func (g *TestGenerator) generateMainTestFile(ctx context.Context, genCtx *testGenerationContext, flowName string) (*GeneratedTest, error) {
	// Build prompt for LLM test generation
	prompt := g.buildTestGenerationPrompt(genCtx, flowName)
	
	// Use LLM to generate the actual test code
	// For now, create a mock response since we don't have LLM test generation implemented
	testContent, err := g.generateTestContent(ctx, prompt, genCtx)
	if err != nil {
		return nil, err
	}

	// Determine file name and path
	fileName := g.generateFileName(flowName, genCtx.framework)
	filePath := filepath.Join(genCtx.outputDir, fileName)

	// Extract action IDs
	var actionIDs []string
	for _, action := range genCtx.actions {
		actionIDs = append(actionIDs, action.ID)
	}

	test := &GeneratedTest{
		Framework:   genCtx.framework.Name,
		FileName:    fileName,
		FilePath:    filePath,
		Content:     testContent,
		Actions:     actionIDs,
		FlowName:    flowName,
		Language:    genCtx.framework.Language,
		GeneratedAt: time.Now(),
	}

	return test, nil
}

// generateSetupFile creates framework-specific setup/helper code
func (g *TestGenerator) generateSetupFile(ctx context.Context, genCtx *testGenerationContext) (*GeneratedTest, error) {
	setupContent := g.generateSetupContent(genCtx)
	
	fileName := g.generateSetupFileName(genCtx.framework)
	filePath := filepath.Join(genCtx.outputDir, fileName)

	setup := &GeneratedTest{
		Framework:   genCtx.framework.Name,
		FileName:    fileName,
		FilePath:    filePath,
		Content:     setupContent,
		Actions:     []string{"setup"},
		FlowName:    "setup",
		Language:    genCtx.framework.Language,
		GeneratedAt: time.Now(),
	}

	return setup, nil
}

// generateConfigFile creates framework configuration
func (g *TestGenerator) generateConfigFile(ctx context.Context, genCtx *testGenerationContext) (*GeneratedTest, error) {
	configContent := g.generateConfigContent(genCtx)
	
	fileName := genCtx.framework.ConfigFile
	filePath := filepath.Join(g.projectRoot, fileName)

	config := &GeneratedTest{
		Framework:   genCtx.framework.Name,
		FileName:    fileName,
		FilePath:    filePath,
		Content:     configContent,
		Actions:     []string{"config"},
		FlowName:    "config",
		Language:    "javascript", // Config files are usually JS/TS
		GeneratedAt: time.Now(),
	}

	return config, nil
}

// buildTestGenerationPrompt creates the LLM prompt for test generation
func (g *TestGenerator) buildTestGenerationPrompt(genCtx *testGenerationContext, flowName string) string {
	var actionsDesc strings.Builder
	for _, action := range genCtx.actions {
		actionsDesc.WriteString(fmt.Sprintf("- **%s**: %s\n", action.Name, action.Description))
		actionsDesc.WriteString(fmt.Sprintf("  - Type: %s\n", action.Type))
		if len(action.Inputs) > 0 {
			actionsDesc.WriteString("  - Inputs: ")
			for i, input := range action.Inputs {
				if i > 0 {
					actionsDesc.WriteString(", ")
				}
				actionsDesc.WriteString(fmt.Sprintf("%s (%s)", input.Label, input.Type))
			}
			actionsDesc.WriteString("\n")
		}
		actionsDesc.WriteString(fmt.Sprintf("  - Success: %s\n", action.Expects.Success))
		actionsDesc.WriteString(fmt.Sprintf("  - Implementation: %s %s\n\n", 
			action.Implementation.Method, action.Implementation.Endpoint))
	}

	return fmt.Sprintf(`Generate a complete E2E test for the %s framework that tests the following user journey:

**Flow Name**: %s
**Framework**: %s (%s)
**Language**: %s
**Style**: %s

**Actions to Test**:
%s

Please generate a complete, working E2E test that:
1. **Follows %s best practices and conventions**
2. **Uses proper syntax for %s framework**
3. **Includes setup and teardown if needed**
4. **Has descriptive test names and comments**
5. **Handles async operations properly**
6. **Includes proper assertions and error handling**
7. **Uses appropriate selectors and waits**

For the test style "%s":
- "adventure": Use creative, story-like descriptions and engaging language
- "standard": Use clear, professional test descriptions
- "minimal": Use concise, minimal descriptions

Make sure the test is:
- **Production ready** with proper error handling
- **Maintainable** with clear structure
- **Reliable** with appropriate waits and retries
- **Complete** with all necessary imports and setup

Return ONLY the complete test file content, ready to save and run.`, 
		genCtx.framework.DisplayName,
		flowName,
		genCtx.framework.DisplayName,
		genCtx.framework.Version,
		genCtx.framework.Language,
		genCtx.style,
		actionsDesc.String(),
		genCtx.framework.DisplayName,
		genCtx.framework.DisplayName,
		genCtx.style)
}

// generateTestContent creates the actual test content (mock for now)
func (g *TestGenerator) generateTestContent(ctx context.Context, prompt string, genCtx *testGenerationContext) (string, error) {
	// TODO: This should use the LLM client to generate actual test content
	// For now, provide framework-specific templates
	
	switch genCtx.framework.Name {
	case "playwright":
		return g.generatePlaywrightTest(genCtx), nil
	case "cypress":
		return g.generateCypressTest(genCtx), nil
	default:
		return g.generateGenericTest(genCtx), nil
	}
}

// generatePlaywrightTest creates a Playwright-specific test template
func (g *TestGenerator) generatePlaywrightTest(genCtx *testGenerationContext) string {
	return fmt.Sprintf(`import { test, expect } from '@playwright/test';

test.describe('%s Journey', () => {
  test('complete user journey', async ({ page }) => {
    // Generated test for %d actions
    console.log('🚀 Starting adventure: %s');
    
%s
    
    console.log('✅ Adventure completed successfully!');
  });
});`,
		strings.Title(genCtx.outputDir),
		len(genCtx.actions),
		genCtx.outputDir,
		g.generateActionSteps(genCtx, "playwright"))
}

// generateCypressTest creates a Cypress-specific test template
func (g *TestGenerator) generateCypressTest(genCtx *testGenerationContext) string {
	return fmt.Sprintf(`describe('%s Journey', () => {
  it('should complete the user journey', () => {
    // Generated test for %d actions
    cy.log('🚀 Starting adventure: %s');
    
%s
    
    cy.log('✅ Adventure completed successfully!');
  });
});`,
		strings.Title(genCtx.outputDir),
		len(genCtx.actions),
		genCtx.outputDir,
		g.generateActionSteps(genCtx, "cypress"))
}

// generateGenericTest creates a generic test template
func (g *TestGenerator) generateGenericTest(genCtx *testGenerationContext) string {
	return fmt.Sprintf(`// Generated E2E test for %s framework
// Test: %s Journey (%d actions)

test('%s user journey', async () => {
  console.log('🚀 Starting test: %s');
  
%s
  
  console.log('✅ Test completed successfully!');
});`,
		genCtx.framework.DisplayName,
		strings.Title(genCtx.outputDir),
		len(genCtx.actions),
		genCtx.outputDir,
		genCtx.outputDir,
		g.generateActionSteps(genCtx, "generic"))
}

// generateActionSteps creates test steps for each action
func (g *TestGenerator) generateActionSteps(genCtx *testGenerationContext, framework string) string {
	var steps strings.Builder
	
	for i, action := range genCtx.actions {
		steps.WriteString(fmt.Sprintf("    // Step %d: %s\n", i+1, action.Name))
		steps.WriteString(fmt.Sprintf("    // %s\n", action.Description))
		
		switch framework {
		case "playwright":
			steps.WriteString(g.generatePlaywrightStep(action))
		case "cypress":
			steps.WriteString(g.generateCypressStep(action))
		default:
			steps.WriteString(g.generateGenericStep(action))
		}
		
		steps.WriteString("\n")
	}
	
	return steps.String()
}

// generatePlaywrightStep creates a Playwright-specific test step
func (g *TestGenerator) generatePlaywrightStep(action types.CodeAction) string {
	switch action.Type {
	case "page_visit":
		return fmt.Sprintf("    await page.goto('%s');\n    await expect(page).toHaveTitle(/%s/);",
			action.Implementation.Endpoint, action.Name)
	case "form_submit":
		var step strings.Builder
		for _, input := range action.Inputs {
			step.WriteString(fmt.Sprintf("    await page.fill('[name=\"%s\"]', '%s');\n", 
				input.Name, input.Example))
		}
		step.WriteString("    await page.click('button[type=\"submit\"]');")
		return step.String()
	default:
		return fmt.Sprintf("    // TODO: Implement %s action", action.Type)
	}
}

// generateCypressStep creates a Cypress-specific test step
func (g *TestGenerator) generateCypressStep(action types.CodeAction) string {
	switch action.Type {
	case "page_visit":
		return fmt.Sprintf("    cy.visit('%s');\n    cy.title().should('contain', '%s');",
			action.Implementation.Endpoint, action.Name)
	case "form_submit":
		var step strings.Builder
		for _, input := range action.Inputs {
			step.WriteString(fmt.Sprintf("    cy.get('[name=\"%s\"]').type('%s');\n", 
				input.Name, input.Example))
		}
		step.WriteString("    cy.get('button[type=\"submit\"]').click();")
		return step.String()
	default:
		return fmt.Sprintf("    // TODO: Implement %s action", action.Type)
	}
}

// generateGenericStep creates a generic test step
func (g *TestGenerator) generateGenericStep(action types.CodeAction) string {
	return fmt.Sprintf("  // TODO: Test %s (%s)", action.Name, action.Type)
}

// generateSetupContent creates setup/helper code
func (g *TestGenerator) generateSetupContent(genCtx *testGenerationContext) string {
	switch genCtx.framework.Name {
	case "playwright":
		return `import { defineConfig } from '@playwright/test';

// Playwright test setup
export const config = defineConfig({
  testDir: './tests',
  timeout: 30000,
  retries: 2,
  use: {
    baseURL: process.env.BASE_URL || 'http://localhost:3000',
    headless: process.env.HEADLESS !== 'false',
  },
});`
	case "cypress":
		return `// Cypress support file
import './commands';

// Setup for all tests
beforeEach(() => {
  cy.log('🚀 Setting up test environment');
});`
	default:
		return fmt.Sprintf(`// Setup file for %s
// Add your test setup and helper functions here`, genCtx.framework.DisplayName)
	}
}

// generateConfigContent creates framework configuration
func (g *TestGenerator) generateConfigContent(genCtx *testGenerationContext) string {
	// Use custom config from framework research if available
	if customExample, exists := genCtx.framework.CustomConfig["example_test"]; exists {
		return fmt.Sprintf(`// %s configuration file
// Generated by TIF

%s`, genCtx.framework.DisplayName, customExample)
	}

	// Default config templates
	switch genCtx.framework.Name {
	case "playwright":
		return `import { defineConfig, devices } from '@playwright/test';

export default defineConfig({
  testDir: './tests',
  timeout: 30 * 1000,
  expect: {
    timeout: 5000
  },
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: 'html',
  use: {
    baseURL: 'http://localhost:3000',
    trace: 'on-first-retry',
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
});`
	case "cypress":
		return `import { defineConfig } from 'cypress'

export default defineConfig({
  e2e: {
    baseUrl: 'http://localhost:3000',
    setupNodeEvents(on, config) {
      // implement node event listeners here
    },
  },
})`
	default:
		return fmt.Sprintf(`// %s configuration
module.exports = {
  // Add your configuration here
};`, genCtx.framework.DisplayName)
	}
}

// generateFileName creates an appropriate test file name
func (g *TestGenerator) generateFileName(flowName string, framework *E2EFramework) string {
	// Clean flow name for file
	cleanName := strings.ToLower(strings.ReplaceAll(flowName, " ", "-"))
	
	// Use appropriate extension
	if len(framework.Extensions) > 0 {
		ext := framework.Extensions[0]
		return cleanName + ext
	}
	
	// Default extension based on language
	switch framework.Language {
	case "typescript":
		return cleanName + ".spec.ts"
	case "javascript":
		return cleanName + ".spec.js"
	case "python":
		return cleanName + "_test.py"
	default:
		return cleanName + ".test.js"
	}
}

// generateSetupFileName creates setup file name
func (g *TestGenerator) generateSetupFileName(framework *E2EFramework) string {
	switch framework.Name {
	case "playwright":
		return "playwright.setup.ts"
	case "cypress":
		return "cypress/support/e2e.js"
	default:
		return "test-setup.js"
	}
}