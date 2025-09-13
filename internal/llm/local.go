package llm

import (
	"context"
	"fmt"
	"strings"

	"github.com/lance13c/tod/internal/types"
)

// localClient implements the Client interface for local/offline analysis
type localClient struct {
	// Configuration for local analysis
	options map[string]interface{}
}

// newLocalClient creates a new local client that works without external LLM services
func newLocalClient(options map[string]interface{}) (Client, error) {
	return &localClient{
		options: options,
	}, nil
}

// AnalyzeCode performs local code analysis without LLM
func (c *localClient) AnalyzeCode(ctx context.Context, code, filePath string) (*CodeAnalysis, error) {
	// Perform pattern-based analysis without external LLM
	analysis := &CodeAnalysis{
		Endpoints:    []EndpointInfo{},
		AuthMethods:  []string{},
		Dependencies: []string{},
		Notes:        "Local analysis without LLM",
		Confidence:   0.6, // Lower confidence for pattern matching
	}

	// Simple pattern matching for common frameworks
	endpoints := c.extractEndpointsFromCode(code, filePath)
	analysis.Endpoints = endpoints

	// Extract auth methods
	analysis.AuthMethods = c.extractAuthMethods(code)

	// Extract dependencies
	analysis.Dependencies = c.extractDependencies(code, filePath)

	return analysis, nil
}

// GenerateFlow generates a basic flow using local logic
func (c *localClient) GenerateFlow(ctx context.Context, actions []types.CodeAction) (*FlowSuggestion, error) {
	if len(actions) == 0 {
		return nil, fmt.Errorf("no actions provided for flow generation")
	}

	// Generate a simple flow based on action types
	flow := &FlowSuggestion{
		Name:        "Generated Test Flow",
		Description: "Auto-generated test flow based on discovered actions",
		Steps:       []FlowStep{},
		Personality: "professional",
		Rationale:   "Flow generated using local pattern matching",
	}

	// Sort actions to create logical flow
	sortedActions := c.sortActionsLogically(actions)

	for i, action := range sortedActions {
		step := FlowStep{
			Name:        fmt.Sprintf("Step %d: %s", i+1, action.Description),
			Type:        "http",
			Action:      action.ID,
			Parameters:  c.generateParameters(action),
			Expects:     c.generateExpectations(action),
			Description: fmt.Sprintf("Execute %s %s", action.Implementation.Method, action.Implementation.Endpoint),
		}
		flow.Steps = append(flow.Steps, step)
	}

	return flow, nil
}

// ExtractActions extracts actions using local pattern matching
func (c *localClient) ExtractActions(ctx context.Context, code, framework, language string) ([]types.CodeAction, error) {
	var actions []types.CodeAction

	switch framework {
	case "nextjs":
		actions = c.extractNextJSActions(code)
	case "express":
		actions = c.extractExpressActions(code)
	case "gin":
		actions = c.extractGinActions(code)
	default:
		actions = c.extractGenericActions(code, language)
	}

	return actions, nil
}

// ResearchFramework researches a framework using local pattern matching
func (c *localClient) ResearchFramework(ctx context.Context, frameworkName, version string) (*FrameworkResearch, error) {
	research := &FrameworkResearch{
		Name:        frameworkName,
		Version:     version,
		DisplayName: frameworkName,
		Language:    "javascript",
		Confidence:  0.6,
		Notes:       "Local research using basic pattern matching",
	}

	// Basic framework detection
	switch frameworkName {
	case "playwright":
		research.DisplayName = "Playwright"
		research.Language = "typescript"
		research.RunCommand = "npx playwright test"
		research.ConfigFile = "playwright.config.ts"
		research.TestDir = "tests"
		research.Extensions = []string{".spec.ts", ".test.ts"}
		research.Confidence = 0.9
	case "cypress":
		research.DisplayName = "Cypress"
		research.RunCommand = "npx cypress run"
		research.ConfigFile = "cypress.config.js"
		research.TestDir = "cypress/e2e"
		research.Extensions = []string{".cy.js", ".cy.ts"}
		research.Confidence = 0.9
	default:
		research.RunCommand = "npm test"
		research.TestDir = "tests"
		research.Extensions = []string{".test.js", ".spec.js"}
		research.Confidence = 0.4
		research.Notes = "Unknown framework - using generic defaults without LLM research"
	}

	return research, nil
}

// extractEndpointsFromCode uses pattern matching to find endpoints
func (c *localClient) extractEndpointsFromCode(code, filePath string) []EndpointInfo {
	var endpoints []EndpointInfo

	// Common HTTP methods
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}

	for _, method := range methods {
		// Look for route definitions
		patterns := []string{
			fmt.Sprintf(`app.%s`, method),
			fmt.Sprintf(`router.%s`, method),
			fmt.Sprintf(`@%sMapping`, method),
			fmt.Sprintf(`%s("`, method),
		}

		for _, pattern := range patterns {
			if strings.Contains(code, pattern) {
				endpoint := EndpointInfo{
					Method:      method,
					Path:        c.extractPathFromPattern(code, pattern),
					Parameters:  make(map[string]Param),
					Responses:   make(map[string]interface{}),
					Auth:        c.detectAuthFromCode(code),
					Description: fmt.Sprintf("%s endpoint", method),
					LineNumber:  1, // TODO: Extract actual line number
				}
				endpoints = append(endpoints, endpoint)
			}
		}
	}

	return endpoints
}

// extractAuthMethods identifies authentication methods in code
func (c *localClient) extractAuthMethods(code string) []string {
	var methods []string

	authPatterns := map[string]string{
		"magic_link":   "magic",
		"bearer_token": "bearer",
		"session":      "session",
		"oauth":        "oauth",
		"basic_auth":   "basic",
	}

	for method, pattern := range authPatterns {
		if strings.Contains(code, pattern) {
			methods = append(methods, method)
		}
	}

	return methods
}

// extractDependencies finds framework dependencies
func (c *localClient) extractDependencies(code, filePath string) []string {
	var deps []string

	// Common framework imports
	frameworks := map[string]string{
		"express": "express",
		"fastify": "fastify",
		"next":    "next",
		"gin":     "gin",
		"flask":   "flask",
		"fastapi": "fastapi",
	}

	for dep, pattern := range frameworks {
		if strings.Contains(code, pattern) {
			deps = append(deps, dep)
		}
	}

	return deps
}

// sortActionsLogically arranges actions in a logical test order
func (c *localClient) sortActionsLogically(actions []types.CodeAction) []types.CodeAction {
	// Simple sorting: auth endpoints first, then others
	var authActions []types.CodeAction
	var otherActions []types.CodeAction

	for _, action := range actions {
		if strings.Contains(action.Implementation.Endpoint, "auth") || strings.Contains(action.Implementation.Endpoint, "login") {
			authActions = append(authActions, action)
		} else {
			otherActions = append(otherActions, action)
		}
	}

	// Combine auth actions first, then others
	result := append(authActions, otherActions...)
	return result
}

// generateParameters creates parameters for an action
func (c *localClient) generateParameters(action types.CodeAction) map[string]string {
	params := make(map[string]string)

	// Generate common parameters based on action type
	if strings.Contains(action.Implementation.Endpoint, "auth") {
		if action.Implementation.Method == "POST" {
			params["email"] = "${TEST_EMAIL}"
			if strings.Contains(action.Implementation.Endpoint, "login") {
				params["password"] = "${TEST_PASSWORD}"
			}
		}
	}

	// Add parameters from action definition
	for name := range action.Implementation.Parameters {
		if _, exists := params[name]; !exists {
			params[name] = fmt.Sprintf("${%s}", name)
		}
	}

	return params
}

// generateExpectations creates expectations for an action
func (c *localClient) generateExpectations(action types.CodeAction) map[string]string {
	expects := make(map[string]string)

	// Default expectation
	expects["status"] = "200"

	// Add specific expectations based on action type
	if action.Implementation.Method == "POST" && strings.Contains(action.Implementation.Endpoint, "auth") {
		expects["body_contains"] = "success"
	}

	return expects
}

// Framework-specific extraction methods

func (c *localClient) extractNextJSActions(code string) []types.CodeAction {
	var actions []types.CodeAction

	// Look for Next.js API routes
	if strings.Contains(code, "export async function") {
		methods := []string{"GET", "POST", "PUT", "DELETE"}
		for _, method := range methods {
			if strings.Contains(code, fmt.Sprintf("export async function %s", method)) {
				action := types.CodeAction{
					ID:          fmt.Sprintf("%s_nextjs_route", method),
					Name:        fmt.Sprintf("NextJS %s Route", method),
					Category:    "API",
					Type:        "api_request",
					Description: fmt.Sprintf("Execute %s request to NextJS route", method),
					Implementation: types.TechnicalDetails{
						Method:     method,
						Endpoint:   "/api/route", // TODO: Extract actual path
						SourceFile: "detected",
					},
				}
				actions = append(actions, action)
			}
		}
	}

	return actions
}

func (c *localClient) extractExpressActions(code string) []types.CodeAction {
	var actions []types.CodeAction

	// Look for Express route definitions
	methods := []string{"get", "post", "put", "delete", "patch"}
	for _, method := range methods {
		pattern := fmt.Sprintf("app.%s(", method)
		if strings.Contains(code, pattern) {
			action := types.CodeAction{
				ID:          fmt.Sprintf("%s_express_route", method),
				Name:        fmt.Sprintf("Express %s Route", method),
				Category:    "API",
				Type:        "api_request",
				Description: fmt.Sprintf("Execute %s request to Express route", method),
				Implementation: types.TechnicalDetails{
					Method:     method,
					Endpoint:   c.extractPathFromPattern(code, pattern),
					SourceFile: "detected",
				},
			}
			actions = append(actions, action)
		}
	}

	return actions
}

func (c *localClient) extractGinActions(code string) []types.CodeAction {
	var actions []types.CodeAction

	// Look for Gin route definitions
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}
	for _, method := range methods {
		pattern := fmt.Sprintf("router.%s(", method)
		if strings.Contains(code, pattern) {
			action := types.CodeAction{
				ID:          fmt.Sprintf("%s_gin_route", method),
				Name:        fmt.Sprintf("Gin %s Route", method),
				Category:    "API",
				Type:        "api_request",
				Description: fmt.Sprintf("Execute %s request to Gin route", method),
				Implementation: types.TechnicalDetails{
					Method:     method,
					Endpoint:   c.extractPathFromPattern(code, pattern),
					SourceFile: "detected",
				},
			}
			actions = append(actions, action)
		}
	}

	return actions
}

func (c *localClient) extractGenericActions(code, language string) []types.CodeAction {
	var actions []types.CodeAction

	// Generic pattern matching based on language
	switch language {
	case "typescript", "javascript":
		// Look for HTTP method calls
		if strings.Contains(code, "fetch(") || strings.Contains(code, "axios") {
			action := types.CodeAction{
				ID:          "generic_http_call",
				Name:        "Generic HTTP Call",
				Category:    "API",
				Type:        "api_request",
				Description: "Execute generic HTTP request",
				Implementation: types.TechnicalDetails{
					Method:     "GET", // Default
					Endpoint:   "/api/generic",
					SourceFile: "detected",
				},
			}
			actions = append(actions, action)
		}
	case "go":
		// Look for HTTP handler functions
		if strings.Contains(code, "http.HandleFunc") {
			action := types.CodeAction{
				ID:          "go_http_handler",
				Name:        "Go HTTP Handler",
				Category:    "API",
				Type:        "api_request",
				Description: "Execute Go HTTP handler request",
				Implementation: types.TechnicalDetails{
					Method:     "GET", // Default
					Endpoint:   "/api/handler",
					SourceFile: "detected",
				},
			}
			actions = append(actions, action)
		}
	case "python":
		// Look for Flask/FastAPI decorators
		if strings.Contains(code, "@app.route") || strings.Contains(code, "@router.") {
			action := types.CodeAction{
				ID:          "python_route",
				Name:        "Python Route",
				Category:    "API",
				Type:        "api_request",
				Description: "Execute Python route request",
				Implementation: types.TechnicalDetails{
					Method:     "GET", // Default
					Endpoint:   "/api/route",
					SourceFile: "detected",
				},
			}
			actions = append(actions, action)
		}
	}

	return actions
}

// Helper methods

func (c *localClient) extractPathFromPattern(code, pattern string) string {
	// Simple path extraction - would need more sophisticated parsing in practice
	return "/api/extracted"
}

func (c *localClient) detectAuthFromCode(code string) string {
	if strings.Contains(code, "magic") {
		return "magic_link"
	}
	if strings.Contains(code, "bearer") || strings.Contains(code, "jwt") {
		return "bearer"
	}
	if strings.Contains(code, "session") {
		return "session"
	}
	return "none"
}

// InterpretCommand implements the Client interface using local pattern matching
func (c *localClient) InterpretCommand(ctx context.Context, command string, availableActions []types.CodeAction) (*CommandInterpretation, error) {
	command = strings.ToLower(strings.TrimSpace(command))

	interpretation := &CommandInterpretation{
		Intent:      command,
		Confidence:  0.7, // Local has decent confidence for pattern matching
		Parameters:  make(map[string]string),
		Suggestions: []string{},
	}

	// Enhanced pattern matching for local client
	switch {
	case strings.Contains(command, "navigate") || strings.Contains(command, "go to") || strings.Contains(command, "visit") || strings.Contains(command, "open"):
		interpretation.CommandType = "navigation"

		// Extract page from command
		for _, word := range []string{"homepage", "home", "main", "index"} {
			if strings.Contains(command, word) {
				interpretation.Parameters["page"] = "/"
				interpretation.Suggestions = []string{"go to /", "visit homepage", "navigate to main page"}
				break
			}
		}

		for _, word := range []string{"login", "signin", "sign-in"} {
			if strings.Contains(command, word) {
				interpretation.Parameters["page"] = "/login"
				interpretation.Suggestions = []string{"go to login", "navigate to sign in page"}
				break
			}
		}

		for _, word := range []string{"dashboard", "dash"} {
			if strings.Contains(command, word) {
				interpretation.Parameters["page"] = "/dashboard"
				interpretation.Suggestions = []string{"go to dashboard", "navigate to main dashboard"}
				break
			}
		}

	case strings.Contains(command, "sign in") || strings.Contains(command, "login") || strings.Contains(command, "authenticate"):
		interpretation.CommandType = "authentication"
		interpretation.Parameters["action"] = "sign_in"
		interpretation.Suggestions = []string{"click sign in button", "fill login form", "enter credentials"}

		// Try to match with available actions
		for _, action := range availableActions {
			actionName := strings.ToLower(action.Name)
			if strings.Contains(actionName, "sign in") || strings.Contains(actionName, "login") {
				interpretation.ActionID = action.ID
				interpretation.Confidence = 0.9
				break
			}
		}

	case strings.Contains(command, "click") || strings.Contains(command, "press") || strings.Contains(command, "tap"):
		interpretation.CommandType = "interaction"
		interpretation.Parameters["action"] = "click"
		interpretation.Suggestions = []string{"click button", "click link", "press enter"}

		// Extract element to click
		parts := strings.Fields(command)
		for i, part := range parts {
			if part == "click" && i+1 < len(parts) {
				interpretation.Parameters["element"] = parts[i+1]
				break
			}
		}

	case strings.Contains(command, "fill") || strings.Contains(command, "type") || strings.Contains(command, "enter"):
		interpretation.CommandType = "form_input"
		interpretation.Parameters["action"] = "fill"
		interpretation.Suggestions = []string{"fill form field", "type in input", "enter text"}

		// Extract field and value if present
		if strings.Contains(command, " with ") {
			parts := strings.Split(command, " with ")
			if len(parts) == 2 {
				interpretation.Parameters["field"] = strings.TrimSpace(parts[0])
				interpretation.Parameters["value"] = strings.TrimSpace(parts[1])
			}
		}

	default:
		interpretation.CommandType = "unknown"
		interpretation.Confidence = 0.4
		interpretation.Suggestions = []string{
			"try 'navigate to homepage'",
			"try 'sign in'",
			"try 'click button'",
			"try 'fill field with value'",
		}

		// Try fuzzy matching against available actions
		for _, action := range availableActions {
			if c.fuzzyMatch(command, action.Name) || c.fuzzyMatch(command, action.Description) {
				interpretation.CommandType = "action_match"
				interpretation.ActionID = action.ID
				interpretation.Confidence = 0.6
				interpretation.Suggestions = []string{fmt.Sprintf("Did you mean: %s?", action.Name)}
				break
			}
		}
	}

	return interpretation, nil
}

// fuzzyMatch performs simple fuzzy matching between command and text
func (c *localClient) fuzzyMatch(command, text string) bool {
	command = strings.ToLower(command)
	text = strings.ToLower(text)

	// Simple word overlap matching
	commandWords := strings.Fields(command)
	textWords := strings.Fields(text)

	if len(commandWords) == 0 || len(textWords) == 0 {
		return false
	}

	matches := 0
	for _, cmdWord := range commandWords {
		for _, textWord := range textWords {
			if strings.Contains(textWord, cmdWord) || strings.Contains(cmdWord, textWord) {
				matches++
				break
			}
		}
	}

	// If more than half the words match, consider it a fuzzy match
	return float64(matches)/float64(len(commandWords)) > 0.5
}

// GetLastUsage implements the Client interface
func (c *localClient) GetLastUsage() *UsageStats {
	// Local analysis has no cost
	return &UsageStats{
		Provider:     "local",
		Model:        "local-analysis",
		InputTokens:  0,
		OutputTokens: 0,
		TotalTokens:  0,
		InputCost:    0.0,
		OutputCost:   0.0,
		TotalCost:    0.0,
	}
}

// AnalyzeScreenshot implements the Client interface (local implementation)
func (c *localClient) AnalyzeScreenshot(ctx context.Context, screenshot []byte, prompt string) (*ScreenshotAnalysis, error) {
	// Local screenshot analysis - simplified implementation
	return &ScreenshotAnalysis{
		Description: "Local screenshot analysis - this is a simplified implementation for testing without external APIs",
		Elements: []UIElement{
			{
				Type:        "generic",
				Text:        "Detected UI elements",
				Selector:    "body",
				Properties:  map[string]string{"analyzed": "locally"},
				Description: "Local analysis detected generic UI elements",
			},
		},
		Suggestions: []string{
			"Local analysis complete",
			"Consider using a full vision LLM for detailed analysis",
		},
		Errors:     []UIError{},
		Confidence: 0.6, // Lower confidence for local analysis
		Usage: &UsageStats{
			Provider:     "local",
			Model:        "local-vision",
			InputTokens:  0,
			OutputTokens: 0,
			TotalTokens:  0,
			InputCost:    0.0,
			OutputCost:   0.0,
			TotalCost:    0.0,
		},
	}, nil
}

// InterpretCommandWithContext implements the Client interface
func (c *localClient) InterpretCommandWithContext(ctx context.Context, command string, availableActions []types.CodeAction, conversation *ConversationContext) (*CommandInterpretation, error) {
	// Delegate to regular interpret command - local client doesn't need conversation context
	return c.InterpretCommand(ctx, command, availableActions)
}

// RankNavigationElements implements the Client interface using local ranking
func (c *localClient) RankNavigationElements(ctx context.Context, userInput string, elements []NavigationElement) (*NavigationRanking, error) {
	// Simple local ranking based on text similarity
	var rankedElements []RankedNavigationElement
	
	userInputLower := strings.ToLower(userInput)
	
	for _, element := range elements {
		score := c.calculateSimpleScore(userInputLower, element.Text, element.Type)
		
		rankedElements = append(rankedElements, RankedNavigationElement{
			NavigationElement: element,
			Confidence:        score, // Use calculated similarity score
			Reasoning:         fmt.Sprintf("Local text similarity matching for '%s'", element.Text),
			Strategy:          "standard",
		})
	}
	
	// Sort by confidence (highest first)  
	for i := 0; i < len(rankedElements)-1; i++ {
		for j := i + 1; j < len(rankedElements); j++ {
			if rankedElements[i].Confidence < rankedElements[j].Confidence {
				rankedElements[i], rankedElements[j] = rankedElements[j], rankedElements[i]
			}
		}
	}
	
	return &NavigationRanking{
		Elements: rankedElements,
		Usage: &UsageStats{
			Provider:     "local",
			Model:        "local-ranking",
			InputTokens:  0,
			OutputTokens: 0,
			TotalTokens:  0,
			InputCost:    0.0,
			OutputCost:   0.0,
			TotalCost:    0.0,
		},
	}, nil
}

// calculateSimpleScore calculates a simple similarity score between user input and element
func (c *localClient) calculateSimpleScore(userInput, elementText, elementType string) float64 {
	elementTextLower := strings.ToLower(elementText)
	
	// Exact match
	if userInput == elementTextLower {
		return 1.0
	}
	
	// Prefix match
	if strings.HasPrefix(elementTextLower, userInput) {
		return 0.9
	}
	
	// Contains match
	if strings.Contains(elementTextLower, userInput) {
		return 0.7
	}
	
	// Word overlap
	userWords := strings.Fields(userInput)
	elementWords := strings.Fields(elementTextLower)
	
	matches := 0
	for _, userWord := range userWords {
		for _, elementWord := range elementWords {
			if strings.Contains(elementWord, userWord) || strings.Contains(userWord, elementWord) {
				matches++
				break
			}
		}
	}
	
	if len(userWords) > 0 {
		wordScore := float64(matches) / float64(len(userWords))
		if wordScore > 0 {
			return 0.3 + (wordScore * 0.4) // Scale to 0.3-0.7 range
		}
	}
	
	// Type-based boost for common actions
	switch elementType {
	case "button", "link":
		if strings.Contains(userInput, "click") || strings.Contains(userInput, "go") {
			return 0.2
		}
	case "input":
		if strings.Contains(userInput, "type") || strings.Contains(userInput, "fill") {
			return 0.2
		}
	}
	
	return 0.0
}

// EstimateCost implements the Client interface
func (c *localClient) EstimateCost(operation string, inputSize int) *UsageStats {
	// Local analysis is free
	return &UsageStats{
		Provider:     "local",
		Model:        "local-analysis",
		InputTokens:  0,
		OutputTokens: 0,
		TotalTokens:  0,
		InputCost:    0.0,
		OutputCost:   0.0,
		TotalCost:    0.0,
	}
}
