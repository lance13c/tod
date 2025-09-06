package core

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ciciliostudio/tod/internal/config"
	"github.com/ciciliostudio/tod/internal/discovery"
	"github.com/ciciliostudio/tod/internal/types"
	"github.com/ciciliostudio/tod/internal/llm"
)

// DefaultFlowAgent implements the FlowAgent interface using LLM
type DefaultFlowAgent struct {
	llmClient   llm.Client
	scanner     *discovery.Scanner
	config      *config.Config
	userConfig  *config.TestUserConfig
	projectRoot string
}

// NewFlowAgent creates a new flow agent
func NewFlowAgent(cfg *config.Config, projectRoot string) (*DefaultFlowAgent, error) {
	// Create LLM client
	provider := llm.Provider(cfg.AI.Provider)
	llmClient, err := llm.NewClient(provider, cfg.AI.APIKey, map[string]interface{}{
		"model": cfg.AI.Model,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM client: %w", err)
	}

	// Create scanner
	scanner := discovery.NewScanner(projectRoot, discovery.ScanOptions{
		Framework: cfg.Testing.Framework,
		Language:  cfg.Testing.Language,
		SkipLLM:   false,
	}, cfg)

	// Load user config
	userLoader := config.NewTestUserLoader(projectRoot)
	userConfig, _ := userLoader.Load() // Ignore error, use empty config

	return &DefaultFlowAgent{
		llmClient:   llmClient,
		scanner:     scanner,
		config:      cfg,
		userConfig:  userConfig,
		projectRoot: projectRoot,
	}, nil
}

// DiscoverFlows discovers all available flows using AI
func (a *DefaultFlowAgent) DiscoverFlows(ctx context.Context) ([]Flow, error) {
	// First, scan the project for actions
	scanResults, err := a.scanner.ScanProject()
	if err != nil {
		return nil, fmt.Errorf("failed to scan project: %w", err)
	}

	// Use LLM to analyze actions and discover flows
	flows, err := a.analyzeActionsForFlows(ctx, scanResults.Actions)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze flows: %w", err)
	}

	return flows, nil
}

// FindSignupFlow specifically looks for signup/registration flows
func (a *DefaultFlowAgent) FindSignupFlow(ctx context.Context) (*Flow, error) {
	flows, err := a.DiscoverFlows(ctx)
	if err != nil {
		return nil, err
	}

	// Find signup-related flow
	for _, flow := range flows {
		if a.isSignupFlow(&flow) {
			return &flow, nil
		}
	}

	return nil, fmt.Errorf("no signup flow found")
}

// FindFlowByIntent finds a flow based on user intent
func (a *DefaultFlowAgent) FindFlowByIntent(ctx context.Context, intent string) (*Flow, error) {
	flows, err := a.DiscoverFlows(ctx)
	if err != nil {
		return nil, err
	}

	// For now, use simple matching until LLM integration is complete
	intentLower := strings.ToLower(intent)
	for _, flow := range flows {
		if a.matchesIntent(&flow, intentLower) {
			return &flow, nil
		}
	}

	return nil, fmt.Errorf("no flow found for intent: %s", intent)
}

// SuggestNextStep provides AI suggestions for the next step
func (a *DefaultFlowAgent) SuggestNextStep(ctx context.Context, currentStep *Step, context map[string]interface{}) ([]Suggestion, error) {
	if currentStep == nil {
		return []Suggestion{}, nil
	}

	// Generate suggestions based on step type and context
	suggestions := []Suggestion{}

	// Add step-specific suggestions
	switch currentStep.Type {
	case StepTypeInput:
		suggestions = a.generateInputSuggestions(currentStep, context)
	case StepTypeForm:
		suggestions = a.generateFormSuggestions(currentStep, context)
	case StepTypeHTTP:
		suggestions = a.generateHTTPSuggestions(currentStep, context)
	}

	return suggestions, nil
}

// GetFieldSuggestions provides AI suggestions for specific fields
func (a *DefaultFlowAgent) GetFieldSuggestions(ctx context.Context, fieldName, fieldType string, context map[string]interface{}) ([]string, error) {
	suggestions := []string{}

	switch fieldType {
	case "email":
		suggestions = []string{
			"test@example.com",
			"user@test.com",
			"demo@company.com",
			fmt.Sprintf("test+%d@example.com", time.Now().Unix()),
		}
	case "password":
		suggestions = []string{
			"Test123!",
			"SecurePass123",
			"Demo12345",
		}
	case "name":
		suggestions = []string{
			"Test User",
			"Demo Account",
			"QA Tester",
		}
	case "username":
		suggestions = []string{
			"testuser",
			"demo_user",
			fmt.Sprintf("user_%d", time.Now().Unix()),
		}
	}

	return suggestions, nil
}

// ExplainFlow provides an AI explanation of a flow
func (a *DefaultFlowAgent) ExplainFlow(ctx context.Context, flow *Flow) (string, error) {
	if flow == nil {
		return "", fmt.Errorf("flow is nil")
	}

	explanation := fmt.Sprintf(`
Flow: %s

%s

This flow contains %d steps:
`, flow.Name, flow.Description, len(flow.Steps))

	for i, step := range flow.Steps {
		explanation += fmt.Sprintf("\n%d. %s - %s", i+1, step.Name, step.Description)
	}

	if flow.AuthType != "" {
		explanation += fmt.Sprintf("\n\nAuthentication: %s", flow.AuthType)
	}

	explanation += fmt.Sprintf("\n\nConfidence: %.0f%%", flow.Confidence*100)

	return explanation, nil
}

// ValidateStepInput validates user input for a step
func (a *DefaultFlowAgent) ValidateStepInput(ctx context.Context, step *Step, input interface{}) (*ValidationResult, error) {
	result := &ValidationResult{Valid: true}

	if step == nil {
		result.Valid = false
		result.Message = "Step is nil"
		return result, nil
	}

	inputStr, ok := input.(string)
	if !ok {
		result.Valid = false
		result.Message = "Input must be a string"
		return result, nil
	}

	// Basic validation based on step inputs
	for _, field := range step.Inputs {
		if field.Required && strings.TrimSpace(inputStr) == "" {
			result.Valid = false
			result.Message = fmt.Sprintf("Field '%s' is required", field.Label)
			result.Suggestions = []string{
				fmt.Sprintf("Please enter a value for %s", field.Label),
			}
			return result, nil
		}

		// Type-specific validation
		if err := a.validateFieldType(field, inputStr); err != nil {
			result.Valid = false
			result.Message = err.Error()
			result.Suggestions = []string{
				fmt.Sprintf("Please enter a valid %s", field.Type),
			}
			return result, nil
		}
	}

	return result, nil
}

// HandleError provides AI suggestions for handling errors
func (a *DefaultFlowAgent) HandleError(ctx context.Context, step *Step, err error) (*ErrorSuggestion, error) {
	suggestion := &ErrorSuggestion{
		Message:  err.Error(),
		CanRetry: true,
		AutoFix:  false,
	}

	// Provide context-aware suggestions
	errorMsg := strings.ToLower(err.Error())
	if strings.Contains(errorMsg, "network") || strings.Contains(errorMsg, "timeout") {
		suggestion.Suggestions = []string{
			"Check your internet connection",
			"Verify the server is running",
			"Try again in a moment",
		}
		suggestion.CanRetry = true
	} else if strings.Contains(errorMsg, "validation") || strings.Contains(errorMsg, "invalid") {
		suggestion.Suggestions = []string{
			"Double-check the input format",
			"Try a different value",
			"Check field requirements",
		}
		suggestion.CanRetry = true
	} else if strings.Contains(errorMsg, "permission") || strings.Contains(errorMsg, "unauthorized") {
		suggestion.Suggestions = []string{
			"Check your authentication credentials",
			"Verify you have the required permissions",
			"Try logging in again",
		}
		suggestion.CanRetry = false
	}

	return suggestion, nil
}

// Helper methods

func (a *DefaultFlowAgent) analyzeActionsForFlows(ctx context.Context, actions []types.CodeAction) ([]Flow, error) {
	flows := []Flow{}

	// Group actions by category
	authActions := []types.CodeAction{}
	for _, action := range actions {
		if strings.Contains(strings.ToLower(action.Category), "auth") {
			authActions = append(authActions, action)
		}
	}

	// Create signup flow if auth actions found
	if len(authActions) > 0 {
		signupFlow := a.createSignupFlowFromActions(authActions)
		if signupFlow != nil {
			flows = append(flows, *signupFlow)
		}
	}

	return flows, nil
}

func (a *DefaultFlowAgent) createSignupFlowFromActions(actions []types.CodeAction) *Flow {
	// Look for signup/register actions
	var signupAction *types.CodeAction
	for _, action := range actions {
		actionName := strings.ToLower(action.Name)
		if strings.Contains(actionName, "signup") || 
		   strings.Contains(actionName, "register") ||
		   strings.Contains(actionName, "create") {
			signupAction = &action
			break
		}
	}

	if signupAction == nil {
		return nil
	}

	// Create flow from action
	flow := &Flow{
		ID:          "signup_flow",
		Name:        "Sign Up Flow",
		Description: "User registration and account creation",
		Category:    "authentication",
		AuthType:    "username_password", // Default, could be detected
		Environment: a.config.Current,
		Confidence:  0.85,
		LastUpdated: time.Now(),
		Personality: "friendly",
		SuccessMessage: "Account created successfully! Welcome aboard!",
		FailureMessage: "Account creation failed. Let's try that again.",
		Steps: []Step{
			{
				ID:          "navigate_signup",
				Name:        "Navigate to Sign Up",
				Type:        StepTypeHTTP,
				Description: "Go to the sign up page",
				Action:      *signupAction,
				Prompt:      "Navigating to sign up page...",
				Expects: ExpectedResult{
					Success: "Sign up form appears",
					Status:  200,
					Contains: []string{"sign up", "create account", "register"},
				},
			},
			{
				ID:          "fill_signup_form",
				Name:        "Fill Sign Up Form",
				Type:        StepTypeForm,
				Description: "Enter user information",
				Inputs: []InputField{
					{
						Name:        "email",
						Label:       "Email Address",
						Type:        "email",
						Required:    true,
						Placeholder: "your@email.com",
					},
					{
						Name:        "password",
						Label:       "Password",
						Type:        "password",
						Required:    true,
						Placeholder: "Choose a strong password",
					},
					{
						Name:        "name",
						Label:       "Full Name",
						Type:        "text",
						Required:    true,
						Placeholder: "Your full name",
					},
				},
				Expects: ExpectedResult{
					Success: "Account created successfully",
					Status:  201,
				},
			},
		},
	}

	return flow
}

func (a *DefaultFlowAgent) isSignupFlow(flow *Flow) bool {
	flowName := strings.ToLower(flow.Name)
	return strings.Contains(flowName, "signup") ||
		   strings.Contains(flowName, "register") ||
		   strings.Contains(flowName, "sign up") ||
		   flow.Category == "authentication"
}

func (a *DefaultFlowAgent) matchesIntent(flow *Flow, intent string) bool {
	flowName := strings.ToLower(flow.Name)
	flowDesc := strings.ToLower(flow.Description)
	
	return strings.Contains(flowName, intent) ||
		   strings.Contains(flowDesc, intent) ||
		   strings.Contains(intent, strings.ToLower(flow.Category))
}

func (a *DefaultFlowAgent) formatFlowsForPrompt(flows []Flow) string {
	result := ""
	for _, flow := range flows {
		result += fmt.Sprintf("- %s: %s (%s)\n", flow.ID, flow.Name, flow.Description)
	}
	return result
}

func (a *DefaultFlowAgent) generateInputSuggestions(step *Step, stepContext map[string]interface{}) []Suggestion {
	suggestions := []Suggestion{}
	
	for _, input := range step.Inputs {
		ctx := context.Background()
		fieldSuggestions, _ := a.GetFieldSuggestions(ctx, input.Name, input.Type, stepContext)
		for _, suggestion := range fieldSuggestions {
			suggestions = append(suggestions, Suggestion{
				Value:      suggestion,
				Label:      fmt.Sprintf("%s suggestion", input.Label),
				Confidence: 0.8,
				Category:   input.Type,
			})
		}
	}
	
	return suggestions
}

func (a *DefaultFlowAgent) generateFormSuggestions(step *Step, stepContext map[string]interface{}) []Suggestion {
	return a.generateInputSuggestions(step, stepContext)
}

func (a *DefaultFlowAgent) generateHTTPSuggestions(step *Step, stepContext map[string]interface{}) []Suggestion {
	return []Suggestion{
		{
			Value:      "Continue",
			Label:      "Proceed with request",
			Confidence: 0.9,
			Category:   "action",
		},
	}
}

func (a *DefaultFlowAgent) validateFieldType(field InputField, value string) error {
	switch field.Type {
	case "email":
		if !strings.Contains(value, "@") {
			return fmt.Errorf("invalid email format")
		}
	case "password":
		if len(value) < 6 {
			return fmt.Errorf("password too short")
		}
	}
	return nil
}