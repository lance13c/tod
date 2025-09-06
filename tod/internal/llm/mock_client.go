package llm

import (
	"context"
	"fmt"
	"strings"

	"github.com/ciciliostudio/tod/internal/types"
)

// mockClient is a mock LLM client for testing
type mockClient struct {
	responses map[string]interface{}
}

// newMockClient creates a new mock LLM client
func newMockClient(options map[string]interface{}) (Client, error) {
	return &mockClient{
		responses: make(map[string]interface{}),
	}, nil
}

// AnalyzeCode implements the Client interface with mock responses
func (m *mockClient) AnalyzeCode(ctx context.Context, code, filePath string) (*CodeAnalysis, error) {
	// Simple mock analysis based on code patterns
	analysis := &CodeAnalysis{
		Endpoints:    []EndpointInfo{},
		AuthMethods:  []string{},
		Dependencies: []string{},
		Notes:        "Mock analysis of " + filePath,
		Confidence:   0.8,
	}

	// Look for common API patterns
	if strings.Contains(code, "router.post") || strings.Contains(code, "app.post") {
		endpoint := EndpointInfo{
			Path:        "/api/mock",
			Method:      "POST",
			Parameters:  make(map[string]Param),
			Responses:   make(map[string]interface{}),
			Auth:        "none",
			Description: "Mock endpoint from code analysis",
			LineNumber:  1,
		}
		analysis.Endpoints = append(analysis.Endpoints, endpoint)
	}

	return analysis, nil
}

// GenerateFlow implements the Client interface with mock responses
func (m *mockClient) GenerateFlow(ctx context.Context, actions []types.CodeAction) (*FlowSuggestion, error) {
	flow := &FlowSuggestion{
		Name:        "Mock Test Flow",
		Description: "A mock test flow generated for testing",
		Steps:       []FlowStep{},
		Personality: "friendly",
		Rationale:   "This is a mock flow for testing purposes",
	}

	// Create simple steps from actions
	for i, action := range actions {
		step := FlowStep{
			Name:        fmt.Sprintf("Step %d: %s", i+1, action.ID),
			Type:        "http",
			Action:      action.ID,
			Parameters:  make(map[string]string),
			Expects:     map[string]string{"status": "200"},
			Description: fmt.Sprintf("Execute %s", action.Description),
		}
		flow.Steps = append(flow.Steps, step)
	}

	return flow, nil
}

// ExtractActions implements the Client interface with mock responses
func (m *mockClient) ExtractActions(ctx context.Context, code, framework, language string) ([]types.CodeAction, error) {
	var actions []types.CodeAction

	// Mock action extraction based on code patterns
	if strings.Contains(strings.ToLower(code), "login") || strings.Contains(strings.ToLower(code), "auth") {
		action := types.CodeAction{
			ID:          "sign_in_with_email",
			Name:        "Sign in with email",
			Category:    "Authentication",
			Type:        "form_submit",
			Description: "User signs in with email and password",
			Implementation: types.TechnicalDetails{
				Endpoint:   "/api/auth/login",
				Method:     "POST",
				SourceFile: "mock_analysis",
				Parameters: make(map[string]types.Param),
				Responses:  make(map[string]interface{}),
			},
			Inputs: []types.UserInput{
				{Name: "email", Label: "Email", Type: "email", Required: true, Example: "user@example.com"},
				{Name: "password", Label: "Password", Type: "password", Required: true, Example: "password123"},
			},
			Expects: types.UserExpectation{
				Success:   "You should be redirected to the dashboard",
				Failure:   "An error message appears",
				Validates: []string{"User is authenticated", "Session is created"},
			},
		}
		actions = append(actions, action)
	}

	if strings.Contains(strings.ToLower(code), "magic") || strings.Contains(strings.ToLower(code), "link") {
		action := types.CodeAction{
			ID:          "sign_in_with_magic_link",
			Name:        "Sign in with magic link",
			Category:    "Authentication",
			Type:        "form_submit",
			Description: "User requests a magic link for passwordless sign in",
			Implementation: types.TechnicalDetails{
				Endpoint:   "/api/auth/magic-link",
				Method:     "POST",
				SourceFile: "mock_analysis",
				Parameters: make(map[string]types.Param),
				Responses:  make(map[string]interface{}),
			},
			Inputs: []types.UserInput{
				{Name: "email", Label: "Email", Type: "email", Required: true, Example: "user@example.com"},
			},
			Expects: types.UserExpectation{
				Success:   "A magic link is sent to your email",
				Failure:   "An error message appears",
				Validates: []string{"Email sent", "Magic link generated"},
			},
		}
		actions = append(actions, action)
	}

	return actions, nil
}

// ResearchFramework implements the Client interface with mock responses
func (m *mockClient) ResearchFramework(ctx context.Context, frameworkName, version string) (*FrameworkResearch, error) {
	// Provide realistic mock data for common frameworks
	research := &FrameworkResearch{
		Name:        frameworkName,
		Version:     version,
		DisplayName: strings.Title(frameworkName),
		Language:    "javascript",
		Confidence:  0.9,
	}

	// Mock framework-specific configurations
	switch strings.ToLower(frameworkName) {
	case "playwright":
		research.DisplayName = "Playwright"
		research.Language = "typescript"
		research.RunCommand = "npx playwright test"
		research.ConfigFile = "playwright.config.ts"
		research.TestDir = "tests"
		research.Extensions = []string{".spec.ts", ".spec.js", ".test.ts", ".test.js"}
		research.InstallSteps = []string{
			"npm install @playwright/test --save-dev",
			"npx playwright install",
			"npx playwright init",
		}
		research.ExampleTest = `import { test, expect } from '@playwright/test';

test('basic test', async ({ page }) => {
  await page.goto('https://playwright.dev/');
  await expect(page).toHaveTitle(/Playwright/);
});`
		research.Documentation = "Playwright is a framework for Web Testing and Automation. It allows testing Chromium, Firefox and WebKit with a single API."

	case "cypress":
		research.DisplayName = "Cypress"
		research.Language = "javascript"
		research.RunCommand = "npx cypress run"
		research.ConfigFile = "cypress.config.js"
		research.TestDir = "cypress/e2e"
		research.Extensions = []string{".cy.js", ".cy.ts", ".spec.js", ".spec.ts"}
		research.InstallSteps = []string{
			"npm install cypress --save-dev",
			"npx cypress open",
		}
		research.ExampleTest = `describe('My First Test', () => {
  it('Does not do much!', () => {
    cy.visit('https://example.cypress.io');
    cy.contains('type').click();
    cy.url().should('include', '/commands/actions');
  });
});`
		research.Documentation = "Cypress is a next generation front end testing tool built for the modern web."

	default:
		// Generic framework template
		research.RunCommand = fmt.Sprintf("npx %s test", frameworkName)
		research.ConfigFile = fmt.Sprintf("%s.config.js", frameworkName)
		research.TestDir = "tests"
		research.Extensions = []string{".test.js", ".spec.js"}
		research.InstallSteps = []string{
			fmt.Sprintf("npm install %s --save-dev", frameworkName),
			fmt.Sprintf("npx %s init", frameworkName),
		}
		research.ExampleTest = fmt.Sprintf(`// Example %s test
test('sample test', async () => {
  // Add your test logic here
});`, frameworkName)
		research.Documentation = fmt.Sprintf("Mock documentation for %s framework", frameworkName)
		research.Confidence = 0.5
		research.Notes = "This is mock data for an unknown framework. Please verify the configuration."
	}

	return research, nil
}

// InterpretCommand implements the Client interface with mock command interpretation
func (m *mockClient) InterpretCommand(ctx context.Context, command string, availableActions []types.CodeAction) (*CommandInterpretation, error) {
	command = strings.ToLower(strings.TrimSpace(command))

	interpretation := &CommandInterpretation{
		Intent:      command,
		Confidence:  0.8,
		Parameters:  make(map[string]string),
		Suggestions: []string{},
	}

	// Simple pattern matching for mock interpretation
	switch {
	case strings.Contains(command, "navigate") || strings.Contains(command, "go to") || strings.Contains(command, "visit"):
		interpretation.CommandType = "navigation"
		if strings.Contains(command, "homepage") || strings.Contains(command, "home") {
			interpretation.Parameters["page"] = "/"
			interpretation.Suggestions = []string{"go to /", "visit homepage", "navigate to main page"}
		} else if strings.Contains(command, "login") {
			interpretation.Parameters["page"] = "/login"
			interpretation.Suggestions = []string{"go to login", "navigate to sign in"}
		} else if strings.Contains(command, "dashboard") {
			interpretation.Parameters["page"] = "/dashboard"
			interpretation.Suggestions = []string{"go to dashboard", "navigate to main dashboard"}
		}

	case strings.Contains(command, "sign in") || strings.Contains(command, "login") || strings.Contains(command, "authenticate"):
		interpretation.CommandType = "authentication"
		interpretation.Parameters["action"] = "sign_in"
		interpretation.Suggestions = []string{"click sign in button", "fill login form", "enter credentials"}

		// Try to match with available actions
		for _, action := range availableActions {
			if strings.Contains(strings.ToLower(action.Name), "sign in") || strings.Contains(strings.ToLower(action.Name), "login") {
				interpretation.ActionID = action.ID
				break
			}
		}

	case strings.Contains(command, "click") || strings.Contains(command, "press") || strings.Contains(command, "tap"):
		interpretation.CommandType = "interaction"
		interpretation.Parameters["action"] = "click"
		interpretation.Suggestions = []string{"click button", "click link", "press enter"}

	case strings.Contains(command, "fill") || strings.Contains(command, "type") || strings.Contains(command, "enter"):
		interpretation.CommandType = "form_input"
		interpretation.Parameters["action"] = "fill"
		interpretation.Suggestions = []string{"fill form field", "type in input", "enter text"}

	default:
		interpretation.CommandType = "unknown"
		interpretation.Confidence = 0.3
		interpretation.Suggestions = []string{"try 'navigate to homepage'", "try 'sign in'", "try 'click button'"}
	}

	return interpretation, nil
}

// GetLastUsage implements the Client interface for mock
func (m *mockClient) GetLastUsage() *UsageStats {
	// Return mock usage stats
	return &UsageStats{
		Provider:     "mock",
		Model:        "mock-model",
		InputTokens:  100,
		OutputTokens: 50,
		TotalTokens:  150,
		InputCost:    0.0001,
		OutputCost:   0.0002,
		TotalCost:    0.0003,
	}
}

// AnalyzeScreenshot implements the Client interface with mock screenshot analysis
func (m *mockClient) AnalyzeScreenshot(ctx context.Context, screenshot []byte, prompt string) (*ScreenshotAnalysis, error) {
	// Mock screenshot analysis
	analysis := &ScreenshotAnalysis{
		Description: "Mock analysis of screenshot. This is a simulated response for testing purposes.",
		Elements: []UIElement{
			{
				Type:        "button",
				Text:        "Login",
				Selector:    "button[type='submit']",
				Properties:  map[string]string{"visible": "true", "enabled": "true"},
				Description: "Primary login button",
			},
			{
				Type:        "input",
				Text:        "",
				Selector:    "input[type='email']",
				Properties:  map[string]string{"placeholder": "Email", "required": "true"},
				Description: "Email input field",
			},
			{
				Type:        "input",
				Text:        "",
				Selector:    "input[type='password']",
				Properties:  map[string]string{"placeholder": "Password", "required": "true"},
				Description: "Password input field",
			},
		},
		Suggestions: []string{
			"Consider adding visual focus indicators for better accessibility",
			"The login form appears well-structured and user-friendly",
			"Consider adding password visibility toggle for better UX",
		},
		Errors: []UIError{
			{
				Type:        "accessibility",
				Description: "Missing alt text on some images",
				Severity:    "medium",
				Suggestion:  "Add descriptive alt attributes to improve screen reader compatibility",
			},
		},
		Confidence: 0.85,
		Usage: &UsageStats{
			Provider:     "mock",
			Model:        "mock-vision-model",
			InputTokens:  250, // Higher for image processing
			OutputTokens: 150,
			TotalTokens:  400,
			InputCost:    0.0,
			OutputCost:   0.0,
			TotalCost:    0.0,
		},
	}

	return analysis, nil
}

// EstimateCost implements the Client interface for mock
func (m *mockClient) EstimateCost(operation string, inputSize int) *UsageStats {
	// Return mock cost estimates
	var inputTokens, outputTokens int64

	switch operation {
	case "analyze_code":
		inputTokens = int64(inputSize) / 4
		outputTokens = 200
	case "generate_flow":
		inputTokens = int64(inputSize) / 4
		outputTokens = 300
	case "research_framework":
		inputTokens = 100
		outputTokens = 500
	case "analyze_screenshot":
		inputTokens = 500 // Higher cost for image processing
		outputTokens = 300
	default:
		inputTokens = int64(inputSize) / 4
		outputTokens = 250
	}

	return &UsageStats{
		Provider:     "mock",
		Model:        "mock-model",
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		TotalTokens:  inputTokens + outputTokens,
		InputCost:    0.0, // Mock = free
		OutputCost:   0.0,
		TotalCost:    0.0,
	}
}
