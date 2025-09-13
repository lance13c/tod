package llm

import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/lance13c/tod/internal/types"
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
	// Log the API call
	logFile, err := os.OpenFile(".tod/api_calls.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err == nil {
		defer logFile.Close()
		logger := log.New(logFile, "[MOCK_CLIENT] ", log.LstdFlags|log.Lmicroseconds)
		logger.Printf("AnalyzeCode called with filePath: %s", filePath)
		logger.Printf("Code/Prompt length: %d characters", len(code))
		
		// Log first 500 chars of the prompt
		if len(code) > 500 {
			logger.Printf("First 500 chars of prompt: %s...", code[:500])
		} else {
			logger.Printf("Full prompt: %s", code)
		}
	}

	// For test generation, return actual test code
	if filePath == "test-generation.txt" {
		testCode := `// Generated Test Code
describe('User Actions Test Suite', () => {
  test('should perform user action 1', async () => {
    // Test implementation here
    await page.click('[data-testid="button1"]');
    await expect(page).toHaveURL('/success');
  });

  test('should perform user action 2', async () => {
    // Test implementation here
    await page.fill('input[name="email"]', 'test@example.com');
    await page.click('button[type="submit"]');
    await expect(page.locator('.success-message')).toBeVisible();
  });
});`

		if logFile != nil {
			logger := log.New(logFile, "[MOCK_CLIENT] ", log.LstdFlags|log.Lmicroseconds)
			logger.Printf("Returning test generation response, length: %d", len(testCode))
		}

		return &CodeAnalysis{
			Notes:      testCode,
			Confidence: 0.9,
		}, nil
	}

	// Handle action discovery from HTML analysis
	if filePath == "page.html" || strings.Contains(code, "Analyze the page and identify important user actions") {
		// Return mock discovered actions in the expected format
		mockActions := `Click the Start button | high
Navigate to the sign in page | high
Fill out the contact form | medium
View the pricing information | medium
Click the Get Started button | low`

		if logFile != nil {
			logger := log.New(logFile, "[MOCK_CLIENT] ", log.LstdFlags|log.Lmicroseconds)
			logger.Printf("Returning action discovery response with %d mock actions", 5)
		}

		return &CodeAnalysis{
			Notes:      mockActions,
			Confidence: 0.8,
		}, nil
	}

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

	if logFile != nil {
		logger := log.New(logFile, "[MOCK_CLIENT] ", log.LstdFlags|log.Lmicroseconds)
		logger.Printf("Returning standard analysis response")
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

	// Enhanced pattern matching for mock interpretation
	switch {
	case strings.Contains(command, "navigate") || strings.Contains(command, "go to") || strings.Contains(command, "visit") || 
		 strings.Contains(command, "go home") || strings.Contains(command, "homepage") || 
		 (strings.Contains(command, "go") && strings.Contains(command, "home")):
		interpretation.CommandType = "navigation"
		interpretation.Confidence = 0.9
		if strings.Contains(command, "homepage") || strings.Contains(command, "home") || strings.Contains(command, "main") {
			interpretation.Parameters["page"] = "/"
			interpretation.Parameters["target"] = "homepage"
			interpretation.Suggestions = []string{"navigate to homepage", "go to main page", "visit home"}
		} else if strings.Contains(command, "login") || strings.Contains(command, "sign") {
			interpretation.Parameters["page"] = "/login"
			interpretation.Parameters["target"] = "login"
			interpretation.Suggestions = []string{"go to login", "navigate to sign in"}
		} else if strings.Contains(command, "dashboard") {
			interpretation.Parameters["page"] = "/dashboard"
			interpretation.Parameters["target"] = "dashboard"
			interpretation.Suggestions = []string{"go to dashboard", "navigate to main dashboard"}
		} else {
			// Generic navigation
			interpretation.Parameters["target"] = "unknown"
			interpretation.Suggestions = []string{"try 'go to homepage'", "try 'navigate to login'"}
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

// InterpretCommandWithContext implements the Client interface with conversation-aware mock command interpretation
func (m *mockClient) InterpretCommandWithContext(ctx context.Context, command string, availableActions []types.CodeAction, conversation *ConversationContext) (*CommandInterpretation, error) {
	// For the mock client, we enhance the basic interpretation with conversation context
	interpretation, err := m.InterpretCommand(ctx, command, availableActions)
	if err != nil {
		return nil, err
	}

	// Enhanced mock behavior based on conversation history
	if conversation != nil && len(conversation.Messages) > 0 {
		// Look at recent conversation for context
		recentMessages := conversation.Messages
		if len(recentMessages) > 5 {
			recentMessages = recentMessages[len(recentMessages)-5:] // Last 5 messages
		}

		// Enhanced interpretation based on conversation context
		for _, msg := range recentMessages {
			if msg.Role == "user" {
				msgLower := strings.ToLower(msg.Content)
				
				// If user previously tried authentication commands
				if strings.Contains(msgLower, "login") || strings.Contains(msgLower, "sign in") {
					if interpretation.CommandType == "unknown" && (strings.Contains(command, "email") || strings.Contains(command, "password")) {
						interpretation.CommandType = "form_input"
						interpretation.Parameters["context"] = "continuing_authentication"
						interpretation.Confidence = 0.9
						interpretation.Suggestions = []string{"fill email field", "enter password", "click login button"}
					}
				}

				// If user was navigating previously
				if strings.Contains(msgLower, "navigate") || strings.Contains(msgLower, "go to") {
					if interpretation.CommandType == "unknown" && strings.Contains(command, "click") {
						interpretation.CommandType = "interaction" 
						interpretation.Parameters["context"] = "continuing_navigation"
						interpretation.Confidence = 0.85
					}
				}
			}
		}

		// Boost confidence for conversational commands
		if interpretation.CommandType != "unknown" {
			interpretation.Confidence = min(1.0, interpretation.Confidence+0.1) // Slight confidence boost
		}

		// Add contextual suggestions
		interpretation.Suggestions = append(interpretation.Suggestions, "Continue with current flow", "Try different approach")
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

// RankNavigationElements implements the Client interface with mock navigation ranking
func (m *mockClient) RankNavigationElements(ctx context.Context, userInput string, elements []NavigationElement) (*NavigationRanking, error) {
	userInputLower := strings.ToLower(userInput)
	rankedElements := []RankedNavigationElement{}

	// Simple matching logic for mock
	for _, element := range elements {
		confidence := 0.0
		strategy := "standard"
		reasoning := ""

		elementTextLower := strings.ToLower(element.Text)

		// Direct match gets highest confidence
		if elementTextLower == userInputLower {
			confidence = 0.95
			reasoning = "Exact text match"
		} else if strings.Contains(elementTextLower, userInputLower) {
			confidence = 0.85
			reasoning = "Text contains user input"
		} else if strings.Contains(userInputLower, elementTextLower) && elementTextLower != "" {
			confidence = 0.75
			reasoning = "User input contains element text"
		} else {
			// Fuzzy matching for common patterns
			if (strings.Contains(userInputLower, "sign") && strings.Contains(elementTextLower, "sign")) ||
			   (strings.Contains(userInputLower, "login") && strings.Contains(elementTextLower, "log")) ||
			   (strings.Contains(userInputLower, "auth") && strings.Contains(elementTextLower, "auth")) {
				confidence = 0.6
				reasoning = "Related authentication terms"
			} else if strings.Contains(userInputLower, "home") && strings.Contains(elementTextLower, "home") {
				confidence = 0.7
				reasoning = "Home navigation match"
			} else {
				// Calculate basic similarity
				similarity := calculateBasicSimilarity(userInputLower, elementTextLower)
				if similarity > 0.3 {
					confidence = similarity * 0.5
					reasoning = "Basic text similarity"
				}
			}
		}

		// Suggest strategy based on element type and text patterns
		if strings.Contains(elementTextLower, "react") || strings.Contains(elementTextLower, "vue") {
			strategy = "javascript"
		} else if element.Type == "button" && strings.Contains(elementTextLower, "submit") {
			strategy = "event"
		}

		if confidence > 0.1 {
			rankedElements = append(rankedElements, RankedNavigationElement{
				NavigationElement: element,
				Confidence:        confidence,
				Reasoning:         reasoning,
				Strategy:          strategy,
			})
		}
	}

	// Sort by confidence descending
	sort.Slice(rankedElements, func(i, j int) bool {
		return rankedElements[i].Confidence > rankedElements[j].Confidence
	})

	return &NavigationRanking{
		Elements: rankedElements,
		Usage: &UsageStats{
			Provider:     "mock",
			Model:        "mock-model",
			InputTokens:  20,
			OutputTokens: 50,
			TotalTokens:  70,
			InputCost:    0.0,
			OutputCost:   0.0,
			TotalCost:    0.0,
		},
	}, nil
}

// calculateBasicSimilarity calculates a simple text similarity score
func calculateBasicSimilarity(a, b string) float64 {
	if a == "" || b == "" {
		return 0.0
	}

	// Count common words
	wordsA := strings.Fields(a)
	wordsB := strings.Fields(b)
	
	commonWords := 0
	for _, wordA := range wordsA {
		for _, wordB := range wordsB {
			if wordA == wordB {
				commonWords++
				break
			}
		}
	}

	totalWords := len(wordsA) + len(wordsB)
	if totalWords == 0 {
		return 0.0
	}

	return float64(commonWords*2) / float64(totalWords)
}
