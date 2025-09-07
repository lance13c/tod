package views

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/ciciliostudio/tod/internal/config"
)

// AuthConfigManager manages saved user credentials for authentication
type AuthConfigManager struct {
	testUserLoader *config.TestUserLoader
	projectDir     string
}

// NewAuthConfigManager creates a new auth config manager
func NewAuthConfigManager(projectDir string) *AuthConfigManager {
	return &AuthConfigManager{
		testUserLoader: config.NewTestUserLoader(projectDir),
		projectDir:     projectDir,
	}
}

// GetSavedUsersForDomain retrieves all saved users for a specific domain
func (a *AuthConfigManager) GetSavedUsersForDomain(domain string) ([]config.TestUser, error) {
	// Load test user configuration
	testConfig, err := a.testUserLoader.Load()
	if err != nil {
		log.Printf("Warning: failed to load test user config: %v", err)
		return []config.TestUser{}, nil // Return empty instead of error
	}

	var domainUsers []config.TestUser

	// Find users that match this domain
	for _, user := range testConfig.Users {
		if a.matchesDomain(user, domain) {
			domainUsers = append(domainUsers, user)
		}
	}

	return domainUsers, nil
}

// SaveUserForDomain saves or updates a user configuration for a domain
func (a *AuthConfigManager) SaveUserForDomain(domain, email, password, name string) error {
	// Load existing configuration
	testConfig, err := a.testUserLoader.Load()
	if err != nil {
		// If no config exists, create default
		testConfig = a.testUserLoader.DefaultTestUserConfig()
	}

	// Generate user ID based on domain and email
	userID := a.generateUserID(domain, email)

	// Check if user already exists
	existingUser, exists := testConfig.GetUser(userID)
	
	now := time.Now()
	
	if exists {
		// Update existing user
		existingUser.Email = email
		existingUser.Password = password
		existingUser.Name = name
		existingUser.UpdatedAt = now
		
		// Update auth config if it exists
		if existingUser.AuthConfig == nil {
			existingUser.AuthConfig = &config.TestUserAuthConfig{}
		}
		existingUser.AuthConfig.Username = email
		existingUser.AuthConfig.Password = password
		
		testConfig.Users[userID] = existingUser
		
		log.Printf("Updated saved user for %s: %s", domain, email)
	} else {
		// Create new user
		newUser := config.TestUser{
			ID:          userID,
			Name:        name,
			Email:       email,
			Username:    email, // Use email as username by default
			Password:    password,
			Environment: "production", // Default to production
			AuthType:    "username_password",
			AuthConfig: &config.TestUserAuthConfig{
				Username: email,
				Password: password,
			},
			Metadata: map[string]interface{}{
				"domain":     domain,
				"created_by": "navigation_mode",
			},
			CreatedAt: now,
			UpdatedAt: now,
		}

		testConfig.AddUser(newUser)
		
		log.Printf("Created new saved user for %s: %s", domain, email)
	}

	// Save the configuration
	return a.testUserLoader.Save(testConfig)
}

// SaveMagicLinkUserForDomain saves a magic link user (email only)
func (a *AuthConfigManager) SaveMagicLinkUserForDomain(domain, email, name string) error {
	// Load existing configuration
	testConfig, err := a.testUserLoader.Load()
	if err != nil {
		testConfig = a.testUserLoader.DefaultTestUserConfig()
	}

	// Generate user ID
	userID := a.generateUserID(domain, email)

	// Check if user already exists
	existingUser, exists := testConfig.GetUser(userID)
	
	now := time.Now()
	
	if exists {
		// Update existing user to magic link type
		existingUser.Email = email
		existingUser.Name = name
		existingUser.AuthType = "magic_link"
		existingUser.UpdatedAt = now
		
		// Update auth config for magic link
		if existingUser.AuthConfig == nil {
			existingUser.AuthConfig = &config.TestUserAuthConfig{}
		}
		existingUser.AuthConfig.EmailCheckEnabled = true
		existingUser.AuthConfig.EmailTimeout = 30
		
		testConfig.Users[userID] = existingUser
		
		log.Printf("Updated magic link user for %s: %s", domain, email)
	} else {
		// Create new magic link user
		newUser := config.TestUser{
			ID:          userID,
			Name:        name,
			Email:       email,
			Environment: "production",
			AuthType:    "magic_link",
			AuthConfig: &config.TestUserAuthConfig{
				EmailCheckEnabled: true,
				EmailTimeout:      30,
			},
			Metadata: map[string]interface{}{
				"domain":     domain,
				"created_by": "navigation_mode",
				"auth_method": "magic_link",
			},
			CreatedAt: now,
			UpdatedAt: now,
		}

		testConfig.AddUser(newUser)
		
		log.Printf("Created new magic link user for %s: %s", domain, email)
	}

	return a.testUserLoader.Save(testConfig)
}

// GetUserByEmailAndDomain retrieves a specific user by email and domain
func (a *AuthConfigManager) GetUserByEmailAndDomain(domain, email string) (*config.TestUser, error) {
	users, err := a.GetSavedUsersForDomain(domain)
	if err != nil {
		return nil, err
	}

	normalizedEmail := strings.ToLower(strings.TrimSpace(email))
	
	for _, user := range users {
		if strings.ToLower(strings.TrimSpace(user.Email)) == normalizedEmail {
			return &user, nil
		}
	}

	return nil, fmt.Errorf("user not found for email %s on domain %s", email, domain)
}

// DeleteUserForDomain removes a user configuration
func (a *AuthConfigManager) DeleteUserForDomain(domain, email string) error {
	testConfig, err := a.testUserLoader.Load()
	if err != nil {
		return err
	}

	userID := a.generateUserID(domain, email)
	
	if testConfig.RemoveUser(userID) {
		log.Printf("Deleted saved user for %s: %s", domain, email)
		return a.testUserLoader.Save(testConfig)
	}

	return fmt.Errorf("user not found")
}

// UpdateUserLastUsed updates the last used timestamp for a user
func (a *AuthConfigManager) UpdateUserLastUsed(domain, email string) error {
	testConfig, err := a.testUserLoader.Load()
	if err != nil {
		return err
	}

	userID := a.generateUserID(domain, email)
	
	if user, exists := testConfig.GetUser(userID); exists {
		user.UpdatedAt = time.Now()
		
		// Update metadata
		if user.Metadata == nil {
			user.Metadata = make(map[string]interface{})
		}
		user.Metadata["last_used"] = time.Now().Unix()
		
		testConfig.Users[userID] = user
		return a.testUserLoader.Save(testConfig)
	}

	return fmt.Errorf("user not found")
}

// GetRecentUsersForDomain returns recently used users for a domain, sorted by last use
func (a *AuthConfigManager) GetRecentUsersForDomain(domain string, limit int) ([]config.TestUser, error) {
	users, err := a.GetSavedUsersForDomain(domain)
	if err != nil {
		return nil, err
	}

	// Sort by UpdatedAt (most recent first)
	for i := 0; i < len(users)-1; i++ {
		for j := i + 1; j < len(users); j++ {
			if users[i].UpdatedAt.Before(users[j].UpdatedAt) {
				users[i], users[j] = users[j], users[i]
			}
		}
	}

	// Limit results
	if limit > 0 && len(users) > limit {
		users = users[:limit]
	}

	return users, nil
}

// GetDomainStats returns statistics about saved users for a domain
func (a *AuthConfigManager) GetDomainStats(domain string) (*DomainStats, error) {
	users, err := a.GetSavedUsersForDomain(domain)
	if err != nil {
		return nil, err
	}

	stats := &DomainStats{
		Domain:      domain,
		TotalUsers:  len(users),
		AuthTypes:   make(map[string]int),
		LastUpdated: time.Time{},
	}

	for _, user := range users {
		// Count auth types
		stats.AuthTypes[user.AuthType]++
		
		// Track most recent update
		if user.UpdatedAt.After(stats.LastUpdated) {
			stats.LastUpdated = user.UpdatedAt
		}
	}

	return stats, nil
}

// matchesDomain checks if a user matches a given domain
func (a *AuthConfigManager) matchesDomain(user config.TestUser, domain string) bool {
	// Check metadata first
	if user.Metadata != nil {
		if userDomain, exists := user.Metadata["domain"]; exists {
			if domainStr, ok := userDomain.(string); ok {
				return strings.EqualFold(domainStr, domain)
			}
		}
	}

	// Fallback: extract domain from email
	if user.Email != "" {
		emailParts := strings.Split(user.Email, "@")
		if len(emailParts) == 2 {
			emailDomain := emailParts[1]
			// Check if domains are related (exact match or subdomain)
			return strings.EqualFold(emailDomain, domain) || 
				   strings.HasSuffix(domain, "."+emailDomain) ||
				   strings.HasSuffix(emailDomain, "."+domain)
		}
	}

	return false
}

// generateUserID creates a unique user ID based on domain and email
func (a *AuthConfigManager) generateUserID(domain, email string) string {
	// Normalize domain and email
	normalizedDomain := strings.ToLower(strings.TrimSpace(domain))
	normalizedEmail := strings.ToLower(strings.TrimSpace(email))
	
	// Remove common prefixes from domain
	normalizedDomain = strings.TrimPrefix(normalizedDomain, "www.")
	normalizedDomain = strings.TrimPrefix(normalizedDomain, "app.")
	normalizedDomain = strings.TrimPrefix(normalizedDomain, "api.")
	
	// Create ID by combining domain and email
	return fmt.Sprintf("%s_%s", 
		strings.ReplaceAll(normalizedDomain, ".", "_"),
		strings.ReplaceAll(normalizedEmail, "@", "_at_"))
}

// DomainStats represents statistics about saved users for a domain
type DomainStats struct {
	Domain      string
	TotalUsers  int
	AuthTypes   map[string]int
	LastUpdated time.Time
}

// HasUsersForAuthType checks if there are any users for a specific auth type
func (s *DomainStats) HasUsersForAuthType(authType string) bool {
	count, exists := s.AuthTypes[authType]
	return exists && count > 0
}

// GetAuthTypeCount returns the count of users for a specific auth type
func (s *DomainStats) GetAuthTypeCount(authType string) int {
	if count, exists := s.AuthTypes[authType]; exists {
		return count
	}
	return 0
}