package core

import (
	"context"
	"time"

	"github.com/lance13c/tod/internal/config"
	"github.com/lance13c/tod/internal/types"
)

// FlowAgent is the core interface for AI-powered flow operations
type FlowAgent interface {
	// Discovery operations
	DiscoverFlows(ctx context.Context) ([]Flow, error)
	FindSignupFlow(ctx context.Context) (*Flow, error)
	FindFlowByIntent(ctx context.Context, intent string) (*Flow, error)
	
	// Suggestion and assistance
	SuggestNextStep(ctx context.Context, currentStep *Step, context map[string]interface{}) ([]Suggestion, error)
	GetFieldSuggestions(ctx context.Context, fieldName, fieldType string, context map[string]interface{}) ([]string, error)
	ExplainFlow(ctx context.Context, flow *Flow) (string, error)
	
	// Execution support
	ValidateStepInput(ctx context.Context, step *Step, input interface{}) (*ValidationResult, error)
	HandleError(ctx context.Context, step *Step, err error) (*ErrorSuggestion, error)
}

// UIProvider abstracts UI interactions for both CLI and TUI
type UIProvider interface {
	// Input collection
	GetInput(prompt string, suggestions []string) (string, error)
	GetPassword(prompt string) (string, error)
	GetSelection(prompt string, options []SelectOption) (string, error)
	GetConfirmation(prompt string) (bool, error)
	
	// Information display
	ShowMessage(msg string, style MessageStyle)
	ShowProgress(current, total int, message string)
	ShowError(err error)
	ShowSuccess(msg string)
	ShowWarning(msg string)
	
	// Advanced UI
	ShowTable(headers []string, rows [][]string)
	ShowJSON(data interface{})
	ShowFlowSummary(flow *Flow)
}

// Flow represents a discovered or defined testing flow
type Flow struct {
	ID          string                 `json:"id" yaml:"id"`
	Name        string                 `json:"name" yaml:"name"`
	Description string                 `json:"description" yaml:"description"`
	Category    string                 `json:"category" yaml:"category"` // "authentication", "onboarding", "checkout", etc.
	AuthType    string                 `json:"auth_type,omitempty" yaml:"auth_type,omitempty"`
	Environment string                 `json:"environment,omitempty" yaml:"environment,omitempty"`
	Steps       []Step                 `json:"steps" yaml:"steps"`
	
	// Discovery metadata
	DiscoveredFrom []string           `json:"discovered_from,omitempty" yaml:"discovered_from,omitempty"` // Source files
	Confidence     float64            `json:"confidence" yaml:"confidence"`
	LastUpdated    time.Time          `json:"last_updated" yaml:"last_updated"`
	
	// Execution metadata
	Metadata       map[string]interface{} `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	
	// UI properties
	Personality    string             `json:"personality,omitempty" yaml:"personality,omitempty"`
	SuccessMessage string             `json:"success_message,omitempty" yaml:"success_message,omitempty"`
	FailureMessage string             `json:"failure_message,omitempty" yaml:"failure_message,omitempty"`
}

// Step represents a single step in a flow
type Step struct {
	ID          string                 `json:"id" yaml:"id"`
	Name        string                 `json:"name" yaml:"name"`
	Type        StepType               `json:"type" yaml:"type"`
	Description string                 `json:"description" yaml:"description"`
	
	// Step configuration
	Action      types.CodeAction   `json:"action,omitempty" yaml:"action,omitempty"`
	Inputs      []InputField           `json:"inputs,omitempty" yaml:"inputs,omitempty"`
	Expects     ExpectedResult         `json:"expects" yaml:"expects"`
	
	// Execution properties
	Prompt      string                 `json:"prompt,omitempty" yaml:"prompt,omitempty"`
	Optional    bool                   `json:"optional,omitempty" yaml:"optional,omitempty"`
	Retryable   bool                   `json:"retryable,omitempty" yaml:"retryable,omitempty"`
	
	// AI assistance
	HelpText    string                 `json:"help_text,omitempty" yaml:"help_text,omitempty"`
	Examples    []string               `json:"examples,omitempty" yaml:"examples,omitempty"`
}

// StepType defines the type of step
type StepType string

const (
	StepTypeHTTP         StepType = "http"
	StepTypeForm         StepType = "form"
	StepTypeBrowser      StepType = "browser"
	StepTypeEmailWait    StepType = "email_wait"
	StepTypeEmailExtract StepType = "email_extract"
	StepTypeDelay        StepType = "delay"
	StepTypeInput        StepType = "input"
	StepTypeValidation   StepType = "validation"
)

// InputField defines an input field for a step
type InputField struct {
	Name        string   `json:"name" yaml:"name"`
	Label       string   `json:"label" yaml:"label"`
	Type        string   `json:"type" yaml:"type"` // "text", "email", "password", "select", etc.
	Required    bool     `json:"required" yaml:"required"`
	Placeholder string   `json:"placeholder,omitempty" yaml:"placeholder,omitempty"`
	Default     string   `json:"default,omitempty" yaml:"default,omitempty"`
	Options     []string `json:"options,omitempty" yaml:"options,omitempty"` // for select type
	Validation  string   `json:"validation,omitempty" yaml:"validation,omitempty"`
	HelpText    string   `json:"help_text,omitempty" yaml:"help_text,omitempty"`
}

// ExpectedResult defines what to expect after a step
type ExpectedResult struct {
	Success   string            `json:"success" yaml:"success"`
	Failure   string            `json:"failure,omitempty" yaml:"failure,omitempty"`
	Status    int               `json:"status,omitempty" yaml:"status,omitempty"`
	Contains  []string          `json:"contains,omitempty" yaml:"contains,omitempty"`
	NotContains []string        `json:"not_contains,omitempty" yaml:"not_contains,omitempty"`
	Headers   map[string]string `json:"headers,omitempty" yaml:"headers,omitempty"`
	Cookies   []string          `json:"cookies,omitempty" yaml:"cookies,omitempty"`
}

// Suggestion represents an AI suggestion
type Suggestion struct {
	Value       string  `json:"value"`
	Label       string  `json:"label,omitempty"`
	Description string  `json:"description,omitempty"`
	Confidence  float64 `json:"confidence"`
	Category    string  `json:"category,omitempty"`
}

// ValidationResult represents the result of input validation
type ValidationResult struct {
	Valid   bool     `json:"valid"`
	Message string   `json:"message,omitempty"`
	Errors  []string `json:"errors,omitempty"`
	Suggestions []string `json:"suggestions,omitempty"`
}

// ErrorSuggestion provides AI suggestions for handling errors
type ErrorSuggestion struct {
	Message     string   `json:"message"`
	Suggestions []string `json:"suggestions"`
	CanRetry    bool     `json:"can_retry"`
	AutoFix     bool     `json:"auto_fix"`
}

// SelectOption represents an option in a selection UI
type SelectOption struct {
	Value       string `json:"value"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
	Disabled    bool   `json:"disabled,omitempty"`
}

// MessageStyle defines the style for UI messages
type MessageStyle string

const (
	StyleInfo    MessageStyle = "info"
	StyleSuccess MessageStyle = "success"
	StyleWarning MessageStyle = "warning"
	StyleError   MessageStyle = "error"
	StyleDebug   MessageStyle = "debug"
)

// ExecutionResult represents the result of flow execution
type ExecutionResult struct {
	Success     bool                   `json:"success"`
	Flow        *Flow                  `json:"flow"`
	StepsRun    int                    `json:"steps_run"`
	StepsTotal  int                    `json:"steps_total"`
	Duration    time.Duration          `json:"duration"`
	Error       error                  `json:"error,omitempty"`
	Data        map[string]interface{} `json:"data,omitempty"`
	
	// For user creation flows
	TestUser    *config.TestUser       `json:"test_user,omitempty"`
	Credentials map[string]string      `json:"credentials,omitempty"`
}

// FlowDiscoveryResult represents discovered flows with metadata
type FlowDiscoveryResult struct {
	Flows       []Flow    `json:"flows"`
	TotalFound  int       `json:"total_found"`
	Duration    time.Duration `json:"duration"`
	Confidence  float64   `json:"confidence"`
	Sources     []string  `json:"sources"` // Files that were analyzed
}

// FlowContext provides context for flow execution
type FlowContext struct {
	Environment   string                 `json:"environment"`
	BaseURL       string                 `json:"base_url"`
	Variables     map[string]string      `json:"variables"`
	Config        *config.Config         `json:"config"`
	UserConfig    *config.TestUserConfig `json:"user_config,omitempty"`
	SessionData   map[string]interface{} `json:"session_data,omitempty"`
}