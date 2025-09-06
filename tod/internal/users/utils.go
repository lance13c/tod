package users

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/ciciliostudio/tod/internal/config"
	"gopkg.in/yaml.v3"
)

// UserManager provides high-level operations for managing test users
type UserManager struct {
	factory        *UserFactory
	sessionManager *SessionManager
	loader         *config.TestUserLoader
	config         *config.Config
}

// NewUserManager creates a new user manager
func NewUserManager(cfg *config.Config, projectDir string) *UserManager {
	loader := config.NewTestUserLoader(projectDir)
	sessionManager := NewSessionManager(projectDir)
	
	// Load user config
	userConfig, err := loader.Load()
	if err != nil {
		// Use default config if load fails
		userConfig = loader.DefaultTestUserConfig()
	}

	factory := NewUserFactory(cfg, userConfig)

	return &UserManager{
		factory:        factory,
		sessionManager: sessionManager,
		loader:         loader,
		config:         cfg,
	}
}

// QuickCreateUser creates a test user with minimal input
func (um *UserManager) QuickCreateUser(name, authType, environment string) (*config.TestUser, error) {
	user, err := um.factory.CreateQuickUser(name, authType, environment)
	if err != nil {
		return nil, err
	}

	// Load current config and add user
	userConfig, err := um.loader.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load user config: %w", err)
	}

	userConfig.AddUser(*user)

	// Save updated config
	err = um.loader.Save(userConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to save user config: %w", err)
	}

	return user, nil
}

// CreateFromTemplate creates a user from a template
func (um *UserManager) CreateFromTemplate(templateName, environment string, overrides map[string]interface{}) (*config.TestUser, error) {
	// Load current user config for templates
	userConfig, err := um.loader.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load user config: %w", err)
	}

	// Update factory with current config
	um.factory.userConfig = userConfig

	user, err := um.factory.CreateFromTemplate(templateName, environment, overrides)
	if err != nil {
		return nil, err
	}

	// Add user to config
	userConfig.AddUser(*user)

	// Save updated config
	err = um.loader.Save(userConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to save user config: %w", err)
	}

	return user, nil
}

// GetUser retrieves a user by ID
func (um *UserManager) GetUser(userID string) (*config.TestUser, error) {
	userConfig, err := um.loader.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load user config: %w", err)
	}

	user, exists := userConfig.GetUser(userID)
	if !exists {
		return nil, fmt.Errorf("user not found: %s", userID)
	}

	return &user, nil
}

// ListUsers returns all users, optionally filtered
func (um *UserManager) ListUsers(filters UserFilters) ([]config.TestUser, error) {
	userConfig, err := um.loader.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load user config: %w", err)
	}

	var users []config.TestUser
	for _, user := range userConfig.Users {
		if um.matchesFilters(user, filters) {
			users = append(users, user)
		}
	}

	// Sort by name
	sort.Slice(users, func(i, j int) bool {
		return users[i].Name < users[j].Name
	})

	return users, nil
}

// GetUsersByEnvironment returns users for a specific environment
func (um *UserManager) GetUsersByEnvironment(environment string) ([]config.TestUser, error) {
	return um.ListUsers(UserFilters{Environment: environment})
}

// DeleteUser removes a user
func (um *UserManager) DeleteUser(userID string) error {
	userConfig, err := um.loader.Load()
	if err != nil {
		return fmt.Errorf("failed to load user config: %w", err)
	}

	if !userConfig.RemoveUser(userID) {
		return fmt.Errorf("user not found: %s", userID)
	}

	// End any active session
	um.sessionManager.EndSession(userID)

	// Save updated config
	return um.loader.Save(userConfig)
}

// AuthenticateUser performs authentication for a user
func (um *UserManager) AuthenticateUser(userID string) (*AuthenticationResult, error) {
	user, err := um.GetUser(userID)
	if err != nil {
		return nil, err
	}

	return um.sessionManager.SimulateAuthentication(user), nil
}

// GetActiveSession returns the active session for a user
func (um *UserManager) GetActiveSession(userID string) (*UserSession, error) {
	return um.sessionManager.GetSession(userID)
}

// ListActiveSessions returns all active sessions
func (um *UserManager) ListActiveSessions() ([]*UserSession, error) {
	return um.sessionManager.ListActiveSessions()
}

// ExportUsers exports users to a YAML file
func (um *UserManager) ExportUsers(filePath string, filters UserFilters) error {
	users, err := um.ListUsers(filters)
	if err != nil {
		return err
	}

	exportData := struct {
		Users     []config.TestUser `yaml:"users"`
		ExportedAt time.Time         `yaml:"exported_at"`
		Filters   UserFilters       `yaml:"filters,omitempty"`
	}{
		Users:      users,
		ExportedAt: time.Now(),
		Filters:    filters,
	}

	data, err := yaml.Marshal(exportData)
	if err != nil {
		return fmt.Errorf("failed to marshal export data: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	return os.WriteFile(filePath, data, 0644)
}

// ImportUsers imports users from a YAML file
func (um *UserManager) ImportUsers(filePath string, overwriteExisting bool) (int, int, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to read import file: %w", err)
	}

	var importData struct {
		Users []config.TestUser `yaml:"users"`
	}

	err = yaml.Unmarshal(data, &importData)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to parse import file: %w", err)
	}

	userConfig, err := um.loader.Load()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to load user config: %w", err)
	}

	imported := 0
	skipped := 0

	for _, user := range importData.Users {
		// Check if user already exists
		if _, exists := userConfig.GetUser(user.ID); exists && !overwriteExisting {
			skipped++
			continue
		}

		// Validate user
		err := um.factory.ValidateUser(&user)
		if err != nil {
			fmt.Printf("⚠️  Skipping invalid user '%s': %v\n", user.Name, err)
			skipped++
			continue
		}

		userConfig.AddUser(user)
		imported++
	}

	// Save updated config
	err = um.loader.Save(userConfig)
	if err != nil {
		return imported, skipped, fmt.Errorf("failed to save user config: %w", err)
	}

	return imported, skipped, nil
}

// GetAvailableTemplates returns all available user templates
func (um *UserManager) GetAvailableTemplates() ([]config.TestUserTemplate, error) {
	userConfig, err := um.loader.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load user config: %w", err)
	}

	return userConfig.ListTemplates(), nil
}

// ValidateUserConfig validates all users in the configuration
func (um *UserManager) ValidateUserConfig() ([]ValidationError, error) {
	userConfig, err := um.loader.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load user config: %w", err)
	}

	var validationErrors []ValidationError

	for userID, user := range userConfig.Users {
		err := um.factory.ValidateUser(&user)
		if err != nil {
			validationErrors = append(validationErrors, ValidationError{
				UserID:  userID,
				Field:   "",
				Message: err.Error(),
			})
		}
	}

	return validationErrors, nil
}

// CleanupExpiredSessions removes expired session files
func (um *UserManager) CleanupExpiredSessions() error {
	return um.sessionManager.CleanupExpiredSessions()
}

// UserFilters defines filters for listing users
type UserFilters struct {
	Environment string
	AuthType    string
	Role        string
	Active      *bool // nil = any, true = active only, false = inactive only
}

// ValidationError represents a user validation error
type ValidationError struct {
	UserID  string
	Field   string
	Message string
}

// matchesFilters checks if a user matches the provided filters
func (um *UserManager) matchesFilters(user config.TestUser, filters UserFilters) bool {
	if filters.Environment != "" && user.Environment != filters.Environment {
		return false
	}

	if filters.AuthType != "" && user.AuthType != filters.AuthType {
		return false
	}

	if filters.Role != "" && user.Role != filters.Role {
		return false
	}

	// Add more filter logic as needed

	return true
}