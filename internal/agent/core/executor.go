package core

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/lance13c/tod/internal/config"
)

// FlowExecutor handles flow execution using the UI abstraction
type FlowExecutor struct {
	agent     FlowAgent
	context   *FlowContext
}

// NewFlowExecutor creates a new flow executor
func NewFlowExecutor(agent FlowAgent, context *FlowContext) *FlowExecutor {
	return &FlowExecutor{
		agent:   agent,
		context: context,
	}
}

// Execute runs a flow with the provided UI provider
func (e *FlowExecutor) Execute(ctx context.Context, flow *Flow, ui UIProvider) (*ExecutionResult, error) {
	if flow == nil {
		return nil, fmt.Errorf("flow is nil")
	}

	if ui == nil {
		return nil, fmt.Errorf("UI provider is nil")
	}

	startTime := time.Now()
	result := &ExecutionResult{
		Flow:       flow,
		StepsTotal: len(flow.Steps),
		Data:       make(map[string]interface{}),
	}

	ui.ShowMessage(fmt.Sprintf("Starting flow: %s", flow.Name), StyleInfo)
	ui.ShowFlowSummary(flow)

	// Execute each step
	for i, step := range flow.Steps {
		result.StepsRun = i + 1
		ui.ShowProgress(i+1, len(flow.Steps), step.Name)

		stepResult, err := e.executeStep(ctx, &step, ui, result.Data)
		if err != nil {
			result.Error = err
			result.Success = false
			ui.ShowError(fmt.Errorf("step '%s' failed: %w", step.Name, err))
			
			// Ask agent for error handling suggestions
			suggestion, _ := e.agent.HandleError(ctx, &step, err)
			if suggestion != nil && suggestion.CanRetry {
				ui.ShowMessage("Error suggestions:", StyleWarning)
				for _, s := range suggestion.Suggestions {
					ui.ShowMessage("  - "+s, StyleInfo)
				}
				
				// Ask user if they want to retry
				if retry, retryErr := ui.GetConfirmation("Would you like to retry this step?"); retryErr == nil && retry {
					// Retry the step
					ui.ShowMessage("Retrying step...", StyleInfo)
					stepResult, err = e.executeStep(ctx, &step, ui, result.Data)
					if err != nil {
						result.Error = err
						result.Success = false
						break
					}
				} else {
					break
				}
			} else {
				break
			}
		}

		// Store step result data
		if stepResult != nil {
			for k, v := range stepResult {
				result.Data[k] = v
			}
		}

		ui.ShowMessage(fmt.Sprintf("✓ %s completed", step.Name), StyleSuccess)
	}

	result.Duration = time.Since(startTime)
	
	if result.Error == nil {
		result.Success = true
		ui.ShowSuccess(flow.SuccessMessage)
		
		// For signup flows, create test user
		if e.isSignupFlow(flow) {
			testUser, err := e.createTestUserFromFlow(flow, result.Data)
			if err != nil {
				ui.ShowWarning(fmt.Sprintf("Flow succeeded but failed to create test user: %v", err))
			} else {
				result.TestUser = testUser
				ui.ShowMessage(fmt.Sprintf("✓ Test user created: %s", testUser.Name), StyleSuccess)
			}
		}
	} else {
		ui.ShowError(fmt.Errorf(flow.FailureMessage))
	}

	return result, nil
}

// executeStep executes a single step
func (e *FlowExecutor) executeStep(ctx context.Context, step *Step, ui UIProvider, data map[string]interface{}) (map[string]interface{}, error) {
	stepData := make(map[string]interface{})

	switch step.Type {
	case StepTypeInput:
		return e.executeInputStep(ctx, step, ui, data)
	case StepTypeForm:
		return e.executeFormStep(ctx, step, ui, data)
	case StepTypeHTTP:
		return e.executeHTTPStep(ctx, step, ui, data)
	case StepTypeBrowser:
		return e.executeBrowserStep(ctx, step, ui, data)
	case StepTypeEmailWait:
		return e.executeEmailWaitStep(ctx, step, ui, data)
	case StepTypeDelay:
		return e.executeDelayStep(ctx, step, ui, data)
	default:
		return stepData, fmt.Errorf("unsupported step type: %s", step.Type)
	}
}

// executeInputStep handles input collection steps
func (e *FlowExecutor) executeInputStep(ctx context.Context, step *Step, ui UIProvider, data map[string]interface{}) (map[string]interface{}, error) {
	stepData := make(map[string]interface{})

	for _, input := range step.Inputs {
		// Get AI suggestions for this field
		suggestions, err := e.agent.GetFieldSuggestions(ctx, input.Name, input.Type, data)
		if err != nil {
			suggestions = []string{} // Fallback to no suggestions
		}

		var value string
		prompt := input.Label
		if input.Required {
			prompt += " (required)"
		}
		if input.Placeholder != "" {
			prompt += fmt.Sprintf(" [%s]", input.Placeholder)
		}
		prompt += ": "

		// Collect input based on type
		switch input.Type {
		case "password":
			value, err = ui.GetPassword(prompt)
		case "select":
			options := make([]SelectOption, len(input.Options))
			for i, opt := range input.Options {
				options[i] = SelectOption{Value: opt, Label: opt}
			}
			value, err = ui.GetSelection(prompt, options)
		default:
			value, err = ui.GetInput(prompt, suggestions)
		}

		if err != nil {
			return nil, fmt.Errorf("failed to get input for %s: %w", input.Name, err)
		}

		// Validate input
		validation, err := e.agent.ValidateStepInput(ctx, step, value)
		if err != nil {
			return nil, fmt.Errorf("validation error for %s: %w", input.Name, err)
		}

		if !validation.Valid {
			ui.ShowError(fmt.Errorf(validation.Message))
			if len(validation.Suggestions) > 0 {
				ui.ShowMessage("Suggestions:", StyleInfo)
				for _, suggestion := range validation.Suggestions {
					ui.ShowMessage("  - "+suggestion, StyleInfo)
				}
			}
			return nil, fmt.Errorf("invalid input for %s: %s", input.Name, validation.Message)
		}

		stepData[input.Name] = value
	}

	return stepData, nil
}

// executeFormStep handles form submission steps
func (e *FlowExecutor) executeFormStep(ctx context.Context, step *Step, ui UIProvider, data map[string]interface{}) (map[string]interface{}, error) {
	ui.ShowMessage("Submitting form data...", StyleInfo)
	
	// For now, simulate form submission
	// In a real implementation, this would interact with the browser or make HTTP requests
	time.Sleep(500 * time.Millisecond) // Simulate processing time
	
	return map[string]interface{}{
		"form_submitted": true,
		"timestamp": time.Now(),
	}, nil
}

// executeHTTPStep handles HTTP request steps
func (e *FlowExecutor) executeHTTPStep(ctx context.Context, step *Step, ui UIProvider, data map[string]interface{}) (map[string]interface{}, error) {
	ui.ShowMessage(fmt.Sprintf("Making HTTP request to %s", step.Action.Implementation.Endpoint), StyleInfo)
	
	// For now, simulate HTTP request
	// In a real implementation, this would make actual HTTP requests
	time.Sleep(300 * time.Millisecond) // Simulate network delay
	
	return map[string]interface{}{
		"http_status": 200,
		"timestamp": time.Now(),
		"endpoint": step.Action.Implementation.Endpoint,
	}, nil
}

// executeBrowserStep handles browser interaction steps
func (e *FlowExecutor) executeBrowserStep(ctx context.Context, step *Step, ui UIProvider, data map[string]interface{}) (map[string]interface{}, error) {
	ui.ShowMessage("Performing browser interaction...", StyleInfo)
	
	// Simulate browser interaction
	time.Sleep(1 * time.Second)
	
	return map[string]interface{}{
		"browser_action": true,
		"timestamp": time.Now(),
	}, nil
}

// executeEmailWaitStep handles waiting for email steps
func (e *FlowExecutor) executeEmailWaitStep(ctx context.Context, step *Step, ui UIProvider, data map[string]interface{}) (map[string]interface{}, error) {
	ui.ShowMessage("Waiting for email... (this is simulated)", StyleInfo)
	
	// Simulate email waiting
	for i := 0; i < 5; i++ {
		ui.ShowProgress(i+1, 5, "Checking email...")
		time.Sleep(500 * time.Millisecond)
	}
	
	return map[string]interface{}{
		"email_received": true,
		"verification_link": "https://example.com/verify?token=abc123",
		"timestamp": time.Now(),
	}, nil
}

// executeDelayStep handles delay steps
func (e *FlowExecutor) executeDelayStep(ctx context.Context, step *Step, ui UIProvider, data map[string]interface{}) (map[string]interface{}, error) {
	duration := 2 * time.Second // Default delay
	ui.ShowMessage(fmt.Sprintf("Waiting %v...", duration), StyleInfo)
	time.Sleep(duration)
	
	return map[string]interface{}{
		"delay_completed": true,
		"timestamp": time.Now(),
	}, nil
}

// Helper methods

func (e *FlowExecutor) isSignupFlow(flow *Flow) bool {
	flowName := strings.ToLower(flow.Name)
	return strings.Contains(flowName, "signup") ||
		   strings.Contains(flowName, "register") ||
		   strings.Contains(flowName, "sign up") ||
		   flow.Category == "authentication"
}

func (e *FlowExecutor) createTestUserFromFlow(flow *Flow, data map[string]interface{}) (*config.TestUser, error) {
	if e.context == nil || e.context.Config == nil {
		return nil, fmt.Errorf("missing context or config")
	}

	// Extract user information from flow data
	email, _ := data["email"].(string)
	name, _ := data["name"].(string)
	password, _ := data["password"].(string)
	username, _ := data["username"].(string)

	if email == "" {
		return nil, fmt.Errorf("no email found in flow data")
	}

	if name == "" {
		name = "Flow Test User"
	}

	// Generate user ID
	userID := fmt.Sprintf("flow_user_%d", time.Now().Unix())

	// Create test user
	user := &config.TestUser{
		ID:          userID,
		Name:        name,
		Email:       email,
		Username:    username,
		Role:        "user",
		Description: fmt.Sprintf("Created from flow: %s", flow.Name),
		Environment: e.context.Environment,
		AuthType:    flow.AuthType,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		AuthConfig: &config.TestUserAuthConfig{
			Username: username,
			Password: password,
		},
	}

	// Save to user config if available
	if e.context.UserConfig != nil {
		e.context.UserConfig.AddUser(*user)
		
		// Save config to disk - use current directory as fallback
		projectRoot := "."
		loader := config.NewTestUserLoader(projectRoot)
		if err := loader.Save(e.context.UserConfig); err != nil {
			return user, fmt.Errorf("user created but failed to save: %w", err)
		}
	}

	return user, nil
}