package llm

import (
	"context"
	"fmt"
	"strings"

	"github.com/ciciliostudio/tod/internal/types"
)

// Provider represents different LLM providers
type Provider string

const (
	OpenAI     Provider = "openai"
	Anthropic  Provider = "anthropic"
	OpenRouter Provider = "openrouter"
	Google     Provider = "google"
	Local      Provider = "local"
	Mock       Provider = "mock"
)

// Client interface for LLM operations
type Client interface {
	AnalyzeCode(ctx context.Context, code, filePath string) (*CodeAnalysis, error)
	GenerateFlow(ctx context.Context, actions []types.CodeAction) (*FlowSuggestion, error)
	ExtractActions(ctx context.Context, code, framework, language string) ([]types.CodeAction, error)
	ResearchFramework(ctx context.Context, frameworkName, version string) (*FrameworkResearch, error)
	InterpretCommand(ctx context.Context, command string, availableActions []types.CodeAction) (*CommandInterpretation, error)
	InterpretCommandWithContext(ctx context.Context, command string, availableActions []types.CodeAction, conversation *ConversationContext) (*CommandInterpretation, error)
	AnalyzeScreenshot(ctx context.Context, screenshot []byte, prompt string) (*ScreenshotAnalysis, error)
	RankNavigationElements(ctx context.Context, userInput string, elements []NavigationElement) (*NavigationRanking, error)
	GetLastUsage() *UsageStats
	EstimateCost(operation string, inputSize int) *UsageStats
}

// CodeAnalysis represents the result of LLM code analysis
type CodeAnalysis struct {
	Endpoints    []EndpointInfo `json:"endpoints"`
	AuthMethods  []string       `json:"auth_methods"`
	Dependencies []string       `json:"dependencies"`
	Notes        string         `json:"notes"`
	Confidence   float64        `json:"confidence"`
	Usage        *UsageStats    `json:"usage,omitempty"`
}

// EndpointInfo represents an analyzed endpoint
type EndpointInfo struct {
	Path        string                 `json:"path"`
	Method      string                 `json:"method"`
	Parameters  map[string]Param       `json:"parameters"`
	Responses   map[string]interface{} `json:"responses"`
	Auth        string                 `json:"auth"`
	Description string                 `json:"description"`
	LineNumber  int                    `json:"line_number"`
}

// Param represents a parameter with its metadata
type Param struct {
	Type        string      `json:"type"`
	Required    bool        `json:"required"`
	Default     interface{} `json:"default,omitempty"`
	Validation  string      `json:"validation,omitempty"`
	Description string      `json:"description,omitempty"`
}

// FlowSuggestion represents a suggested test flow
type FlowSuggestion struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Steps       []FlowStep  `json:"steps"`
	Personality string      `json:"personality"`
	Rationale   string      `json:"rationale"`
	Usage       *UsageStats `json:"usage,omitempty"`
}

// FlowStep represents a step in a test flow
type FlowStep struct {
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	Action      string            `json:"action"`
	Parameters  map[string]string `json:"parameters"`
	Expects     map[string]string `json:"expects"`
	Description string            `json:"description"`
}

// FrameworkResearch represents LLM research results for a custom framework
type FrameworkResearch struct {
	Name          string      `json:"name"`
	Version       string      `json:"version"`
	DisplayName   string      `json:"display_name"`
	Language      string      `json:"language"`
	RunCommand    string      `json:"run_command"`
	ConfigFile    string      `json:"config_file"`
	TestDir       string      `json:"test_dir"`
	Extensions    []string    `json:"extensions"`
	InstallSteps  []string    `json:"install_steps"`
	ExampleTest   string      `json:"example_test"`
	Documentation string      `json:"documentation"`
	Confidence    float64     `json:"confidence"`
	Notes         string      `json:"notes"`
	Usage         *UsageStats `json:"usage,omitempty"`
}

// CommandInterpretation represents the result of interpreting a natural language command
type CommandInterpretation struct {
	Intent      string            `json:"intent"`
	CommandType string            `json:"command_type"`
	Parameters  map[string]string `json:"parameters"`
	Suggestions []string          `json:"suggestions"`
	Confidence  float64           `json:"confidence"`
	ActionID    string            `json:"action_id,omitempty"`
	Usage       *UsageStats       `json:"usage,omitempty"`
}

// MessageContent represents different types of content for multimodal messages
type MessageContent struct {
	Type      string `json:"type"`      // "text" or "image"
	Text      string `json:"text,omitempty"`
	ImageData string `json:"image_data,omitempty"` // base64 encoded image
	MediaType string `json:"media_type,omitempty"` // "image/jpeg", "image/png", etc.
}

// ConversationMessage represents a message in a conversation thread
type ConversationMessage struct {
	Role    string `json:"role"`    // "user", "assistant", or "system"
	Content string `json:"content"` // The message content
}

// ConversationContext contains conversation history and metadata for LLM calls
type ConversationContext struct {
	SessionID string                `json:"session_id"`          // Unique session identifier
	Messages  []ConversationMessage `json:"messages"`            // Conversation history
	MaxTokens int                   `json:"max_tokens,omitempty"` // Optional token limit
}

// ScreenshotAnalysis represents the result of analyzing a screenshot
type ScreenshotAnalysis struct {
	Description string      `json:"description"`
	Elements    []UIElement `json:"elements"`
	Suggestions []string    `json:"suggestions"`
	Errors      []UIError   `json:"errors"`
	Confidence  float64     `json:"confidence"`
	Usage       *UsageStats `json:"usage,omitempty"`
}

// UIElement represents a UI element found in a screenshot
type UIElement struct {
	ID          string            `json:"id"`
	Type        string            `json:"type"`        // "button", "input", "link", etc.
	Text        string            `json:"text"`
	Selector    string            `json:"selector"`
	Attributes  map[string]string `json:"attributes"`
	Properties  map[string]string `json:"properties"`  // Alias for compatibility
	Bounds      Bounds            `json:"bounds"`
	Clickable   bool              `json:"clickable"`
	Description string            `json:"description"`
}

// UIError represents an error or issue found in the UI
type UIError struct {
	Type        string  `json:"type"`        // "accessibility", "broken_link", "missing_alt", etc.
	Severity    string  `json:"severity"`    // "low", "medium", "high", "critical"
	Description string  `json:"description"`
	Element     string  `json:"element"`     // selector or description of problematic element
	Suggestion  string  `json:"suggestion"`
}

// Bounds represents the position and size of a UI element
type Bounds struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

// NewClient creates a new LLM client based on provider
func NewClient(provider Provider, apiKey string, options map[string]interface{}) (Client, error) {
	switch provider {
	case OpenAI:
		return newOpenAIClient(apiKey, options)
	case Anthropic:
		return newAnthropicClient(apiKey, options)
	case OpenRouter:
		return newOpenRouterClient(apiKey, options)
	case Google:
		return newGoogleClient(apiKey, options)
	case Local:
		return newLocalClient(options)
	case Mock:
		return newMockClient(options)
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", provider)
	}
}

// AnalyzeCodeWithLLM is a convenience function for code analysis
func AnalyzeCodeWithLLM(ctx context.Context, client Client, code, filePath, framework, language string) (*types.CodeAction, error) {
	analysis, err := client.AnalyzeCode(ctx, code, filePath)
	if err != nil {
		return nil, err
	}

	if len(analysis.Endpoints) == 0 {
		return nil, fmt.Errorf("no endpoints found in code")
	}

	// Convert first endpoint to TestAction
	endpoint := analysis.Endpoints[0]
	action := &types.CodeAction{
		ID:          generateActionID(endpoint.Path, endpoint.Method),
		Name:        endpoint.Description,
		Category:    "API",
		Type:        "api_request",
		Description: endpoint.Description,
		Implementation: types.TechnicalDetails{
			Endpoint:   endpoint.Path,
			Method:     endpoint.Method,
			SourceFile: fmt.Sprintf("%s:%d", filePath, endpoint.LineNumber),
			Parameters: convertParameters(endpoint.Parameters),
			Responses:  endpoint.Responses,
		},
	}

	return action, nil
}

// generateActionID creates a unique ID for an action
func generateActionID(path, method string) string {
	// Convert path to safe identifier
	id := strings.ReplaceAll(path, "/", "_")
	id = strings.ReplaceAll(id, "-", "_")
	id = strings.ReplaceAll(id, "{", "")
	id = strings.ReplaceAll(id, "}", "")
	id = strings.ToLower(id)

	// Remove leading underscores
	id = strings.TrimLeft(id, "_")

	return fmt.Sprintf("%s_%s", strings.ToLower(method), id)
}

// convertParameters converts LLM parameters to discovery parameters
func convertParameters(llmParams map[string]Param) map[string]types.Param {
	params := make(map[string]types.Param)
	for name, param := range llmParams {
		params[name] = types.Param{
			Type:       param.Type,
			Required:   param.Required,
			Default:    param.Default,
			Validation: param.Validation,
		}
	}
	return params
}

// Smart action discovery prompt for user-friendly actions
const smartActionDiscoveryPrompt = `Analyze this code and determine what USER ACTIONS it implements for E2E testing.

Transform technical code into user-friendly test actions that describe what a real user would DO:

GOOD Examples:
- Code: router.post('/api/auth/login', validateEmail, checkPassword) 
  Action: "Sign in with email" - User enters email and password to access their account

- Code: router.post('/api/auth/magic-link', sendMagicLinkEmail)
  Action: "Sign in with magic link" - User requests a passwordless sign-in link via email

- Code: router.post('/api/subscription/trial', createTrialSubscription)
  Action: "Start free trial" - User begins a trial period of the product

- Code: <form onSubmit={handleCheckout}> with credit card fields
  Action: "Complete purchase" - User enters payment details to buy the product

- Code: router.get('/dashboard', requireAuth, renderDashboard)
  Action: "Go to dashboard" - User navigates to their main application screen

Framework: %s
File: %s

Code:
%s

For each user action you identify, return:
{
  "id": "sign_in_with_email",
  "name": "Sign in with email", 
  "category": "Authentication",
  "type": "form_submit",
  "description": "User enters credentials to sign into their account",
  "inputs": [
    {"name": "email", "label": "Email address", "type": "email", "required": true, "example": "user@example.com"},
    {"name": "password", "label": "Password", "type": "password", "required": true, "example": "********"}
  ],
  "expects": {
    "success": "User is redirected to dashboard",
    "failure": "Invalid credentials error message appears", 
    "validates": ["User is authenticated", "Session is created"]
  }
}

Focus on the USER'S PERSPECTIVE:
1. What is the user trying to accomplish?
2. What would a non-technical person call this action?
3. What inputs does the user need to provide?
4. What does success look like to the user?
5. How would this appear in an E2E test scenario?

Analyze the PURPOSE and INTENT of the code, not just its technical structure.`

const flowGenerationPrompt = `Given these API actions, create a logical test flow that would test the complete user journey.

Actions:
%s

Create a test flow that:
1. Tests the actions in a logical sequence
2. Handles authentication properly
3. Includes magic link flows if present
4. Has appropriate error handling
5. Uses a friendly, adventure-themed personality

Return JSON in this format:
{
  "name": "Magic Link Authentication Journey",
  "description": "Complete magic link authentication flow",
  "steps": [
    {
      "name": "Request Magic Link",
      "type": "http",
      "action": "post_api_auth_magic_link",
      "parameters": {"email": "${TEST_EMAIL}"},
      "expects": {"status": "200"},
      "description": "The brave adventurer requests a magical portal key..."
    }
  ],
  "personality": "friendly",
  "rationale": "This flow tests the complete authentication journey..."
}`

// Framework research prompt for custom frameworks
const frameworkResearchPrompt = `Research the E2E testing framework "%s" version "%s" and provide detailed information for setting it up and using it.

I need comprehensive information to configure and use this framework for end-to-end testing. Research the latest documentation, GitHub repositories, and best practices.

Please provide detailed information in this JSON format:

{
  "name": "%s",
  "version": "%s",
  "display_name": "Framework Display Name",
  "language": "javascript|typescript|python|java|csharp|go|ruby|php",
  "run_command": "command to run tests (e.g., 'npm test', 'npx framework test')",
  "config_file": "typical configuration file name (e.g., 'framework.config.js')",
  "test_dir": "default test directory (e.g., 'tests', 'e2e', 'spec')",
  "extensions": [".test.js", ".spec.js", ".e2e.js"],
  "install_steps": [
    "npm install framework --save-dev",
    "npx framework init",
    "configure framework.config.js"
  ],
  "example_test": "// Example test code showing basic usage\ntest('example test', async () => {\n  // framework-specific test code\n});",
  "documentation": "Brief overview of key concepts and features",
  "confidence": 0.95,
  "notes": "Additional important information, gotchas, or tips"
}

Focus on:
1. **Installation**: How to install and set up the framework
2. **Configuration**: Default config files and setup
3. **Test Structure**: How tests are organized and written  
4. **Commands**: How to run tests
5. **Best Practices**: Common patterns and conventions
6. **File Extensions**: What file patterns to look for
7. **Language Support**: Primary language(s) used

If the framework doesn't exist or you're not confident, set confidence < 0.5 and explain in notes.
Be specific and practical - this will be used to automatically configure the framework.`

// Navigation ranking types

// NavigationElement represents a navigable element for LLM ranking
type NavigationElement struct {
	Text        string `json:"text"`
	Selector    string `json:"selector"`
	URL         string `json:"url,omitempty"`
	Type        string `json:"type"` // "link", "button", "form", etc.
	Description string `json:"description,omitempty"`
}

// RankedNavigationElement represents an element with LLM-assigned ranking
type RankedNavigationElement struct {
	NavigationElement
	Confidence float64 `json:"confidence"`
	Reasoning  string  `json:"reasoning,omitempty"`
	Strategy   string  `json:"strategy"` // "standard", "javascript", "event", "focus_enter", "text"
}

// NavigationRanking represents LLM ranking results
type NavigationRanking struct {
	Elements []RankedNavigationElement `json:"elements"`
	Usage    *UsageStats               `json:"usage,omitempty"`
}

// Navigation ranking prompt
const navigationRankingPrompt = `You are helping a user navigate a website. The user wants to: "%s"

Available navigation elements:
%s

Your task is to rank these elements by how likely they are to fulfill the user's intent. For each element, consider:
1. Text similarity to user intent
2. Common web patterns (e.g., "Sign In" for signin)
3. Element type and context

Also suggest the best clicking strategy for each element:
- "standard": Regular CSS selector click (default)
- "javascript": JavaScript element.click() for React/Vue components
- "event": Dispatch click event for complex event handlers
- "focus_enter": Focus element and press Enter
- "text": Find element by text content if selector fails

Return JSON array of ranked elements (highest confidence first):
[
  {
    "text": "element text",
    "selector": "css selector",
    "url": "url if applicable",
    "type": "element type",
    "confidence": 0.95,
    "reasoning": "why this element matches user intent",
    "strategy": "recommended clicking strategy"
  }
]

Only include elements with confidence > 0.1. Sort by confidence descending.`
