package config

import (
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// TestUser represents a test user with authentication credentials
type TestUser struct {
	ID          string                 `yaml:"id"`
	Name        string                 `yaml:"name"`
	Email       string                 `yaml:"email,omitempty"`
	Username    string                 `yaml:"username,omitempty"`
	Password    string                 `yaml:"password,omitempty"`
	Role        string                 `yaml:"role,omitempty"`
	Description string                 `yaml:"description,omitempty"`
	Environment string                 `yaml:"environment"`
	AuthType    string                 `yaml:"auth_type"`
	AuthConfig  *TestUserAuthConfig    `yaml:"auth_config,omitempty"`
	Metadata    map[string]interface{} `yaml:"metadata,omitempty"`
	CreatedAt   time.Time              `yaml:"created_at"`
	UpdatedAt   time.Time              `yaml:"updated_at"`
}

// TestUserAuthConfig holds authentication-specific configuration for test users
type TestUserAuthConfig struct {
	// Basic auth
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`

	// Bearer token
	Token   string            `yaml:"token,omitempty"`
	Headers map[string]string `yaml:"headers,omitempty"`

	// OAuth
	Provider     string `yaml:"provider,omitempty"`
	ClientID     string `yaml:"client_id,omitempty"`
	ClientSecret string `yaml:"client_secret,omitempty"`
	AccessToken  string `yaml:"access_token,omitempty"`
	RefreshToken string `yaml:"refresh_token,omitempty"`
	ExpiresAt    *time.Time `yaml:"expires_at,omitempty"`

	// Magic link
	EmailEndpoint string `yaml:"email_endpoint,omitempty"`
	LastMagicLink string `yaml:"last_magic_link,omitempty"`

	// Email checking (automatic verification code/magic link extraction)
	EmailCheckEnabled bool   `yaml:"email_check_enabled,omitempty"`
	EmailTimeout      int    `yaml:"email_timeout,omitempty"` // Timeout in seconds (default: 30)

	// Username/password form
	LoginFormURL  string `yaml:"login_form_url,omitempty"`
	UsernameField string `yaml:"username_field,omitempty"`
	PasswordField string `yaml:"password_field,omitempty"`
	SubmitButton  string `yaml:"submit_button,omitempty"`

	// Session storage
	SessionStorage string                 `yaml:"session_storage,omitempty"`
	Cookies        []Cookie               `yaml:"cookies,omitempty"`
	StoredData     map[string]interface{} `yaml:"stored_data,omitempty"`
}

// TestUserTemplate represents a template for creating test users
type TestUserTemplate struct {
	Name        string                 `yaml:"name"`
	Description string                 `yaml:"description"`
	AuthType    string                 `yaml:"auth_type"`
	Role        string                 `yaml:"role,omitempty"`
	Fields      []TestUserField        `yaml:"fields"`
	Defaults    map[string]interface{} `yaml:"defaults,omitempty"`
}

// TestUserField represents a field that needs to be filled when creating a user from template
type TestUserField struct {
	Name        string   `yaml:"name"`
	Label       string   `yaml:"label"`
	Type        string   `yaml:"type"` // text, password, email, select, boolean
	Required    bool     `yaml:"required"`
	Default     string   `yaml:"default,omitempty"`
	Options     []string `yaml:"options,omitempty"` // for select type
	Placeholder string   `yaml:"placeholder,omitempty"`
	Description string   `yaml:"description,omitempty"`
}

// TestUserConfig holds all test user configurations
type TestUserConfig struct {
	Users     map[string]TestUser     `yaml:"users"`
	Templates map[string]TestUserTemplate `yaml:"templates"`
	Meta      struct {
		Version   string    `yaml:"version"`
		CreatedAt time.Time `yaml:"created_at"`
		UpdatedAt time.Time `yaml:"updated_at"`
	} `yaml:"meta"`
}

// TestUserLoader handles loading and saving test user configurations
type TestUserLoader struct {
	projectDir string
}

// NewTestUserLoader creates a new test user loader
func NewTestUserLoader(projectDir string) *TestUserLoader {
	return &TestUserLoader{
		projectDir: projectDir,
	}
}

// GetTestUserConfigPath returns the path to the test users config file
func (l *TestUserLoader) GetTestUserConfigPath() string {
	return filepath.Join(l.projectDir, ".tod", "test_users.yaml")
}

// Load loads the test user configuration
func (l *TestUserLoader) Load() (*TestUserConfig, error) {
	configPath := l.GetTestUserConfigPath()
	
	// Return empty config if file doesn't exist
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return l.DefaultTestUserConfig(), nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config TestUserConfig
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

// Save saves the test user configuration
func (l *TestUserLoader) Save(config *TestUserConfig) error {
	configPath := l.GetTestUserConfigPath()
	
	// Ensure directory exists
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Update metadata
	now := time.Now()
	if config.Meta.CreatedAt.IsZero() {
		config.Meta.CreatedAt = now
	}
	config.Meta.UpdatedAt = now
	config.Meta.Version = "1.0.0"

	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

// DefaultTestUserConfig returns a default test user configuration with templates
func (l *TestUserLoader) DefaultTestUserConfig() *TestUserConfig {
	now := time.Now()
	return &TestUserConfig{
		Users:     make(map[string]TestUser),
		Templates: l.getDefaultTemplates(),
		Meta: struct {
			Version   string    `yaml:"version"`
			CreatedAt time.Time `yaml:"created_at"`
			UpdatedAt time.Time `yaml:"updated_at"`
		}{
			Version:   "1.0.0",
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
}

// getDefaultTemplates returns default test user templates
func (l *TestUserLoader) getDefaultTemplates() map[string]TestUserTemplate {
	templates := make(map[string]TestUserTemplate)

	// Admin user template
	templates["admin"] = TestUserTemplate{
		Name:        "Administrator",
		Description: "Admin user with full permissions",
		AuthType:    "username_password",
		Role:        "admin",
		Fields: []TestUserField{
			{Name: "name", Label: "Full Name", Type: "text", Required: true, Default: "Admin User"},
			{Name: "email", Label: "Email", Type: "email", Required: true, Placeholder: "admin@example.com"},
			{Name: "username", Label: "Username", Type: "text", Required: true, Default: "admin"},
			{Name: "password", Label: "Password", Type: "password", Required: true, Default: "admin123"},
		},
	}

	// Regular user template
	templates["user"] = TestUserTemplate{
		Name:        "Regular User",
		Description: "Standard user with basic permissions",
		AuthType:    "username_password",
		Role:        "user",
		Fields: []TestUserField{
			{Name: "name", Label: "Full Name", Type: "text", Required: true, Default: "Test User"},
			{Name: "email", Label: "Email", Type: "email", Required: true, Placeholder: "user@example.com"},
			{Name: "username", Label: "Username", Type: "text", Required: true, Default: "testuser"},
			{Name: "password", Label: "Password", Type: "password", Required: true, Default: "test123"},
		},
	}

	// API user template (bearer token)
	templates["api_user"] = TestUserTemplate{
		Name:        "API User",
		Description: "User for API testing with bearer token",
		AuthType:    "bearer",
		Role:        "api",
		Fields: []TestUserField{
			{Name: "name", Label: "API User Name", Type: "text", Required: true, Default: "API Test User"},
			{Name: "token", Label: "Bearer Token", Type: "password", Required: true, Placeholder: "your-api-token"},
		},
	}

	// OAuth user template
	templates["oauth_user"] = TestUserTemplate{
		Name:        "OAuth User",
		Description: "User for OAuth-based authentication testing",
		AuthType:    "oauth",
		Role:        "user",
		Fields: []TestUserField{
			{Name: "name", Label: "Full Name", Type: "text", Required: true, Default: "OAuth Test User"},
			{Name: "email", Label: "Email", Type: "email", Required: true, Placeholder: "oauth@example.com"},
			{Name: "provider", Label: "OAuth Provider", Type: "select", Required: true, Options: []string{"google", "github", "microsoft", "facebook"}},
		},
	}

	// Magic link user template  
	templates["magic_link_user"] = TestUserTemplate{
		Name:        "Magic Link User",
		Description: "User for magic link authentication testing with automatic email checking",
		AuthType:    "magic_link",
		Role:        "user",
		Fields: []TestUserField{
			{Name: "name", Label: "Full Name", Type: "text", Required: true, Default: "Magic Link User"},
			{Name: "email", Label: "Email", Type: "email", Required: true, Placeholder: "magic@example.com"},
			{Name: "email_check_enabled", Label: "Enable Email Checking", Type: "boolean", Required: false, Default: "true"},
		},
		Defaults: map[string]interface{}{
			"email_check_enabled": true,
			"email_timeout":      30,
		},
	}

	return templates
}

// AddUser adds a new test user
func (c *TestUserConfig) AddUser(user TestUser) {
	if c.Users == nil {
		c.Users = make(map[string]TestUser)
	}
	
	now := time.Now()
	if user.CreatedAt.IsZero() {
		user.CreatedAt = now
	}
	user.UpdatedAt = now
	
	c.Users[user.ID] = user
}

// GetUser returns a test user by ID
func (c *TestUserConfig) GetUser(id string) (TestUser, bool) {
	user, exists := c.Users[id]
	return user, exists
}

// GetUsersByEnvironment returns all users for a specific environment
func (c *TestUserConfig) GetUsersByEnvironment(env string) []TestUser {
	var users []TestUser
	for _, user := range c.Users {
		if user.Environment == env {
			users = append(users, user)
		}
	}
	return users
}

// GetUsersByAuthType returns all users with a specific auth type
func (c *TestUserConfig) GetUsersByAuthType(authType string) []TestUser {
	var users []TestUser
	for _, user := range c.Users {
		if user.AuthType == authType {
			users = append(users, user)
		}
	}
	return users
}

// RemoveUser removes a test user by ID
func (c *TestUserConfig) RemoveUser(id string) bool {
	if _, exists := c.Users[id]; exists {
		delete(c.Users, id)
		return true
	}
	return false
}

// GetTemplate returns a template by name
func (c *TestUserConfig) GetTemplate(name string) (TestUserTemplate, bool) {
	template, exists := c.Templates[name]
	return template, exists
}

// ListTemplates returns all available templates
func (c *TestUserConfig) ListTemplates() []TestUserTemplate {
	var templates []TestUserTemplate
	for _, template := range c.Templates {
		templates = append(templates, template)
	}
	return templates
}