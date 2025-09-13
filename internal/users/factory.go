package users

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/lance13c/tod/internal/config"
)

// UserFactory provides methods for creating and managing test users
type UserFactory struct {
	config     *config.Config
	userConfig *config.TestUserConfig
}

// NewUserFactory creates a new user factory
func NewUserFactory(config *config.Config, userConfig *config.TestUserConfig) *UserFactory {
	return &UserFactory{
		config:     config,
		userConfig: userConfig,
	}
}

// CreateFromTemplate creates a test user from a template with minimal input
func (f *UserFactory) CreateFromTemplate(templateName, environment string, overrides map[string]interface{}) (*config.TestUser, error) {
	template, exists := f.userConfig.GetTemplate(templateName)
	if !exists {
		return nil, fmt.Errorf("template '%s' not found", templateName)
	}

	// Use current environment if not specified
	if environment == "" {
		environment = f.config.Current
	}

	// Verify environment exists
	if _, exists := f.config.Envs[environment]; !exists {
		return nil, fmt.Errorf("environment '%s' not found", environment)
	}

	// Start with template defaults
	userData := make(map[string]interface{})
	if template.Defaults != nil {
		for k, v := range template.Defaults {
			userData[k] = v
		}
	}

	// Apply overrides
	for k, v := range overrides {
		userData[k] = v
	}

	// Fill in required fields with sensible defaults if missing
	f.fillDefaults(userData, template, environment)

	// Generate user ID
	userName := getStringValue(userData, "name", template.Name+" User")
	userID := f.generateUserID(userName, environment)

	// Create user
	user := &config.TestUser{
		ID:          userID,
		Name:        userName,
		Email:       getStringValue(userData, "email", ""),
		Username:    getStringValue(userData, "username", ""),
		Role:        getStringValue(userData, "role", template.Role),
		Description: getStringValue(userData, "description", template.Description),
		Environment: environment,
		AuthType:    template.AuthType,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Create auth config
	authConfig, err := f.createAuthConfig(template.AuthType, userData)
	if err != nil {
		return nil, fmt.Errorf("failed to create auth config: %w", err)
	}
	user.AuthConfig = authConfig

	return user, nil
}

// CreateQuickUser creates a test user with minimal configuration using smart defaults
func (f *UserFactory) CreateQuickUser(name, authType, environment string) (*config.TestUser, error) {
	if environment == "" {
		environment = f.config.Current
	}

	// Verify environment exists
	if _, exists := f.config.Envs[environment]; !exists {
		return nil, fmt.Errorf("environment '%s' not found", environment)
	}

	userID := f.generateUserID(name, environment)

	user := &config.TestUser{
		ID:          userID,
		Name:        name,
		Email:       f.generateTestEmail(name),
		Username:    f.generateUsername(name),
		Role:        "user",
		Description: fmt.Sprintf("Quick test user for %s", authType),
		Environment: environment,
		AuthType:    authType,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Create auth config with defaults
	authConfig, err := f.createDefaultAuthConfig(authType, user)
	if err != nil {
		return nil, fmt.Errorf("failed to create auth config: %w", err)
	}
	user.AuthConfig = authConfig

	return user, nil
}

// GenerateCredentials generates secure credentials for a user
func (f *UserFactory) GenerateCredentials(authType string) (map[string]string, error) {
	credentials := make(map[string]string)

	switch authType {
	case "none":
		// No credentials needed

	case "basic", "username_password":
		password, err := f.generateSecurePassword()
		if err != nil {
			return nil, err
		}
		credentials["password"] = password

	case "bearer":
		token, err := f.generateBearerToken()
		if err != nil {
			return nil, err
		}
		credentials["token"] = token

	case "oauth":
		// OAuth credentials are typically provided externally
		credentials["client_id"] = "your-oauth-client-id"
		credentials["client_secret"] = "your-oauth-client-secret"

	case "magic_link":
		// Magic link uses email, no additional credentials needed
	}

	return credentials, nil
}

// ValidateUser validates a test user configuration
func (f *UserFactory) ValidateUser(user *config.TestUser) error {
	if user.ID == "" {
		return fmt.Errorf("user ID is required")
	}

	if user.Name == "" {
		return fmt.Errorf("user name is required")
	}

	if user.Environment == "" {
		return fmt.Errorf("environment is required")
	}

	// Verify environment exists
	if _, exists := f.config.Envs[user.Environment]; !exists {
		return fmt.Errorf("environment '%s' not found", user.Environment)
	}

	if user.AuthType == "" {
		return fmt.Errorf("auth type is required")
	}

	// Validate auth-specific requirements
	return f.validateAuthConfig(user.AuthType, user.AuthConfig)
}

// fillDefaults fills in missing required fields with sensible defaults
func (f *UserFactory) fillDefaults(userData map[string]interface{}, template config.TestUserTemplate, environment string) {
	// Ensure name is set
	if getStringValue(userData, "name", "") == "" {
		userData["name"] = template.Name + " User"
	}

	// Generate email if needed
	if getStringValue(userData, "email", "") == "" && f.needsEmail(template.AuthType) {
		name := getStringValue(userData, "name", "test")
		userData["email"] = f.generateTestEmail(name)
	}

	// Generate username if needed
	if getStringValue(userData, "username", "") == "" && f.needsUsername(template.AuthType) {
		name := getStringValue(userData, "name", "test")
		userData["username"] = f.generateUsername(name)
	}

	// Generate password if needed
	if getStringValue(userData, "password", "") == "" && f.needsPassword(template.AuthType) {
		password, _ := f.generateSecurePassword()
		userData["password"] = password
	}

	// Set environment
	userData["environment"] = environment
}

// createAuthConfig creates authentication configuration
func (f *UserFactory) createAuthConfig(authType string, data map[string]interface{}) (*config.TestUserAuthConfig, error) {
	if authType == "none" {
		return nil, nil
	}

	authConfig := &config.TestUserAuthConfig{}

	switch authType {
	case "basic":
		authConfig.Username = getStringValue(data, "username", "")
		authConfig.Password = getStringValue(data, "password", "")

	case "bearer":
		token := getStringValue(data, "token", "")
		if token == "" {
			var err error
			token, err = f.generateBearerToken()
			if err != nil {
				return nil, err
			}
		}
		authConfig.Token = token

	case "oauth":
		authConfig.Provider = getStringValue(data, "provider", "")
		authConfig.ClientID = getStringValue(data, "client_id", "")
		authConfig.ClientSecret = getStringValue(data, "client_secret", "")

	case "magic_link":
		authConfig.EmailEndpoint = getStringValue(data, "email_endpoint", "/auth/magic-link")

	case "username_password":
		authConfig.LoginFormURL = getStringValue(data, "login_form_url", "/login")
		authConfig.UsernameField = getStringValue(data, "username_field", "#email")
		authConfig.PasswordField = getStringValue(data, "password_field", "#password")
		authConfig.SubmitButton = getStringValue(data, "submit_button", "button[type=\"submit\"]")
	}

	return authConfig, nil
}

// createDefaultAuthConfig creates auth config with smart defaults
func (f *UserFactory) createDefaultAuthConfig(authType string, user *config.TestUser) (*config.TestUserAuthConfig, error) {
	if authType == "none" {
		return nil, nil
	}

	authConfig := &config.TestUserAuthConfig{}

	switch authType {
	case "basic":
		authConfig.Username = user.Username
		password, err := f.generateSecurePassword()
		if err != nil {
			return nil, err
		}
		authConfig.Password = password

	case "bearer":
		token, err := f.generateBearerToken()
		if err != nil {
			return nil, err
		}
		authConfig.Token = token

	case "oauth":
		authConfig.Provider = "google" // Default to Google
		authConfig.ClientID = "your-oauth-client-id"
		authConfig.ClientSecret = "your-oauth-client-secret"

	case "magic_link":
		authConfig.EmailEndpoint = "/auth/magic-link"

	case "username_password":
		authConfig.LoginFormURL = "/login"
		authConfig.UsernameField = "#email"
		authConfig.PasswordField = "#password"
		authConfig.SubmitButton = "button[type=\"submit\"]"
	}

	return authConfig, nil
}

// validateAuthConfig validates authentication configuration
func (f *UserFactory) validateAuthConfig(authType string, authConfig *config.TestUserAuthConfig) error {
	if authType == "none" {
		return nil
	}

	if authConfig == nil {
		return fmt.Errorf("auth config is required for auth type: %s", authType)
	}

	switch authType {
	case "basic":
		if authConfig.Username == "" {
			return fmt.Errorf("username is required for basic auth")
		}
		if authConfig.Password == "" {
			return fmt.Errorf("password is required for basic auth")
		}

	case "bearer":
		if authConfig.Token == "" {
			return fmt.Errorf("token is required for bearer auth")
		}

	case "oauth":
		if authConfig.Provider == "" {
			return fmt.Errorf("provider is required for oauth")
		}

	case "username_password":
		if authConfig.LoginFormURL == "" {
			return fmt.Errorf("login form URL is required for username/password auth")
		}
	}

	return nil
}

// generateUserID creates a unique user ID
func (f *UserFactory) generateUserID(name, environment string) string {
	base := strings.ToLower(strings.ReplaceAll(name, " ", "_"))
	base = strings.ReplaceAll(base, "-", "_")
	if environment != "" {
		base = base + "_" + environment
	}
	return base + "_" + fmt.Sprintf("%d", time.Now().Unix())
}

// generateTestEmail creates a test email address
func (f *UserFactory) generateTestEmail(name string) string {
	base := strings.ToLower(strings.ReplaceAll(name, " ", "."))
	base = strings.ReplaceAll(base, "-", ".")
	return base + "+test@example.com"
}

// generateUsername creates a username from a name
func (f *UserFactory) generateUsername(name string) string {
	username := strings.ToLower(strings.ReplaceAll(name, " ", ""))
	username = strings.ReplaceAll(username, "-", "")
	return username
}

// generateSecurePassword generates a secure random password
func (f *UserFactory) generateSecurePassword() (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
	const passwordLength = 12

	password := make([]byte, passwordLength)
	for i := range password {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		password[i] = charset[num.Int64()]
	}

	return string(password), nil
}

// generateBearerToken generates a bearer token
func (f *UserFactory) generateBearerToken() (string, error) {
	const tokenLength = 32
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	token := make([]byte, tokenLength)
	for i := range token {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		token[i] = charset[num.Int64()]
	}

	return "tok_" + string(token), nil
}

// needsEmail returns true if the auth type requires an email
func (f *UserFactory) needsEmail(authType string) bool {
	return authType == "magic_link" || authType == "oauth"
}

// needsUsername returns true if the auth type requires a username
func (f *UserFactory) needsUsername(authType string) bool {
	return authType == "basic" || authType == "username_password"
}

// needsPassword returns true if the auth type requires a password
func (f *UserFactory) needsPassword(authType string) bool {
	return authType == "basic" || authType == "username_password"
}

// Helper function
func getStringValue(data map[string]interface{}, key, defaultValue string) string {
	if value, exists := data[key]; exists {
		if strValue, ok := value.(string); ok {
			return strValue
		}
	}
	return defaultValue
}