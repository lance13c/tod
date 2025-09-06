package config

import (
	"time"
)

// Config represents the complete Tod configuration
type Config struct {
	AI      AIConfig               `yaml:"ai"`
	Testing TestingConfig          `yaml:"testing"`
	Envs    map[string]EnvConfig   `yaml:"environments"`
	Current string                 `yaml:"current_env"`
	Meta    MetaConfig             `yaml:"meta"`
}

// AIConfig holds AI provider configuration
type AIConfig struct {
	Provider string                 `yaml:"provider"` // gemini, openai, grok, claude, openrouter, custom
	APIKey   string                 `yaml:"api_key"`
	Model    string                 `yaml:"model"`
	Endpoint string                 `yaml:"endpoint,omitempty"` // for custom providers
	Settings map[string]interface{} `yaml:"settings,omitempty"`
}

// TestingConfig holds E2E testing framework configuration
type TestingConfig struct {
	Framework string `yaml:"framework"`        // detected or user-specified name
	Version   string `yaml:"version"`          // e.g., "1.40.0"
	Language  string `yaml:"language"`         // typescript, javascript, python
	TestDir   string `yaml:"test_dir"`         // where tests should be generated
	Command   string `yaml:"command"`          // how to run tests (e.g., "npm test")
	Template  string `yaml:"template,omitempty"` // optional: custom test template
	Pattern   string `yaml:"pattern"`          // test file pattern (e.g., "*.spec.ts")
}

// EnvConfig holds environment-specific configuration
type EnvConfig struct {
	Name     string            `yaml:"name"`
	BaseURL  string            `yaml:"base_url"`
	Headers  map[string]string `yaml:"headers,omitempty"`
	Auth     *AuthConfig       `yaml:"auth,omitempty"`
	Cookies  []Cookie          `yaml:"cookies,omitempty"`
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	Type           string            `yaml:"type"`     // none, basic, bearer, oauth, magic_link, username_password
	Username       string            `yaml:"username,omitempty"`
	Password       string            `yaml:"password,omitempty"`
	Token          string            `yaml:"token,omitempty"`
	Headers        map[string]string `yaml:"headers,omitempty"`
	
	// OAuth-specific fields
	Provider      string `yaml:"provider,omitempty"`       // google, github, microsoft, facebook, custom
	ClientID      string `yaml:"client_id,omitempty"`
	ClientSecret  string `yaml:"client_secret,omitempty"`
	LoginURL      string `yaml:"login_url,omitempty"`
	CallbackURL   string `yaml:"callback_url,omitempty"`
	
	// Session storage configuration
	SessionStorage string `yaml:"session_storage,omitempty"` // cookies, localStorage, headers
	
	// Magic link specific
	EmailEndpoint string `yaml:"email_endpoint,omitempty"`
	
	// Username/password form specific
	LoginFormURL  string `yaml:"login_form_url,omitempty"`
	UsernameField string `yaml:"username_field,omitempty"`
	PasswordField string `yaml:"password_field,omitempty"`
	SubmitButton  string `yaml:"submit_button,omitempty"`
}

// Cookie represents a browser cookie
type Cookie struct {
	Name     string `yaml:"name"`
	Value    string `yaml:"value"`
	Domain   string `yaml:"domain,omitempty"`
	Path     string `yaml:"path,omitempty"`
	Secure   bool   `yaml:"secure,omitempty"`
	HTTPOnly bool   `yaml:"http_only,omitempty"`
}

// UsageConfig holds LLM usage tracking and cost data
type UsageConfig struct {
	Session SessionUsage            `json:"session"`
	Daily   map[string]DailyUsage   `json:"daily"`
	Weekly  map[string]WeeklyUsage  `json:"weekly"`
	Monthly map[string]MonthlyUsage `json:"monthly"`
}

// SessionUsage tracks current session usage
type SessionUsage struct {
	StartTime    time.Time     `json:"start_time"`
	TotalTokens  int64         `json:"total_tokens"`
	InputTokens  int64         `json:"input_tokens"`
	OutputTokens int64         `json:"output_tokens"`
	TotalCost    float64       `json:"total_cost"`
	RequestCount int           `json:"request_count"`
	Providers    map[string]ProviderUsage `json:"providers"`
}

// DailyUsage tracks usage per day
type DailyUsage struct {
	Date         string        `json:"date"`
	TotalTokens  int64         `json:"total_tokens"`
	InputTokens  int64         `json:"input_tokens"`
	OutputTokens int64         `json:"output_tokens"`
	TotalCost    float64       `json:"total_cost"`
	RequestCount int           `json:"request_count"`
	Providers    map[string]ProviderUsage `json:"providers"`
}

// WeeklyUsage tracks usage per week
type WeeklyUsage struct {
	Week         string        `json:"week"`
	TotalTokens  int64         `json:"total_tokens"`
	InputTokens  int64         `json:"input_tokens"`
	OutputTokens int64         `json:"output_tokens"`
	TotalCost    float64       `json:"total_cost"`
	RequestCount int           `json:"request_count"`
	Providers    map[string]ProviderUsage `json:"providers"`
}

// MonthlyUsage tracks usage per month
type MonthlyUsage struct {
	Month        string        `json:"month"`
	TotalTokens  int64         `json:"total_tokens"`
	InputTokens  int64         `json:"input_tokens"`
	OutputTokens int64         `json:"output_tokens"`
	TotalCost    float64       `json:"total_cost"`
	RequestCount int           `json:"request_count"`
	Providers    map[string]ProviderUsage `json:"providers"`
}

// ProviderUsage tracks usage per LLM provider
type ProviderUsage struct {
	Model        string  `json:"model"`
	TotalTokens  int64   `json:"total_tokens"`
	InputTokens  int64   `json:"input_tokens"`
	OutputTokens int64   `json:"output_tokens"`
	TotalCost    float64 `json:"total_cost"`
	RequestCount int     `json:"request_count"`
}

// MetaConfig holds metadata about the configuration
type MetaConfig struct {
	Version   string    `yaml:"version"`
	CreatedAt time.Time `yaml:"created_at"`
	UpdatedAt time.Time `yaml:"updated_at"`
}

// DefaultConfig returns a new config with sensible defaults
func DefaultConfig() *Config {
	now := time.Now()
	return &Config{
		AI: AIConfig{
			Provider: "openai",
			Model:    "gpt-4-turbo",
		},
		Testing: TestingConfig{
			Language: "typescript",
			TestDir:  "tests/e2e",
			Command:  "npm test",
			Pattern:  "*.spec.ts",
		},
		Envs: map[string]EnvConfig{
			"development": {
				Name:    "development",
				BaseURL: "http://localhost:3000",
			},
		},
		Current: "development",
		Meta: MetaConfig{
			Version:   "1.0.0",
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.AI.Provider == "" {
		return NewValidationError("ai.provider is required")
	}
	
	if c.AI.APIKey == "" && c.AI.Provider != "custom" {
		return NewValidationError("ai.api_key is required for provider: " + c.AI.Provider)
	}
	
	if c.Testing.Framework == "" {
		return NewValidationError("testing.framework is required")
	}
	
	if c.Current != "" {
		if _, exists := c.Envs[c.Current]; !exists {
			return NewValidationError("current_env references non-existent environment: " + c.Current)
		}
	}
	
	return nil
}

// GetCurrentEnv returns the configuration for the current environment
func (c *Config) GetCurrentEnv() *EnvConfig {
	if c.Current == "" {
		return nil
	}
	
	env, exists := c.Envs[c.Current]
	if !exists {
		return nil
	}
	
	return &env
}

// ValidationError represents a configuration validation error
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return "config validation error: " + e.Message
}

// NewValidationError creates a new validation error
func NewValidationError(message string) error {
	return &ValidationError{Message: message}
}