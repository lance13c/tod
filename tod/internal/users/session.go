package users

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ciciliostudio/tod/internal/config"
)

// SessionManager handles user session management and authentication state
type SessionManager struct {
	projectDir string
}

// NewSessionManager creates a new session manager
func NewSessionManager(projectDir string) *SessionManager {
	return &SessionManager{
		projectDir: projectDir,
	}
}

// UserSession represents an active user session
type UserSession struct {
	UserID      string                 `json:"user_id"`
	Environment string                 `json:"environment"`
	AuthType    string                 `json:"auth_type"`
	StartedAt   time.Time              `json:"started_at"`
	LastUsedAt  time.Time              `json:"last_used_at"`
	ExpiresAt   *time.Time             `json:"expires_at,omitempty"`
	Cookies     []config.Cookie        `json:"cookies,omitempty"`
	Headers     map[string]string      `json:"headers,omitempty"`
	LocalStorage map[string]string     `json:"local_storage,omitempty"`
	SessionData map[string]interface{} `json:"session_data,omitempty"`
	IsActive    bool                   `json:"is_active"`
}

// AuthenticationResult represents the result of an authentication attempt
type AuthenticationResult struct {
	Success     bool              `json:"success"`
	Message     string            `json:"message"`
	Session     *UserSession      `json:"session,omitempty"`
	Cookies     []config.Cookie   `json:"cookies,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	RedirectURL string            `json:"redirect_url,omitempty"`
	Error       error             `json:"error,omitempty"`
}

// StartSession creates a new session for a user
func (sm *SessionManager) StartSession(user *config.TestUser) (*UserSession, error) {
	session := &UserSession{
		UserID:      user.ID,
		Environment: user.Environment,
		AuthType:    user.AuthType,
		StartedAt:   time.Now(),
		LastUsedAt:  time.Now(),
		Headers:     make(map[string]string),
		LocalStorage: make(map[string]string),
		SessionData: make(map[string]interface{}),
		IsActive:    true,
	}

	// Set session expiration based on auth type
	switch user.AuthType {
	case "bearer":
		// Bearer tokens typically have shorter lifespans
		expiresAt := time.Now().Add(24 * time.Hour)
		session.ExpiresAt = &expiresAt
	case "oauth":
		// OAuth tokens can have varying lifespans
		if user.AuthConfig != nil && user.AuthConfig.ExpiresAt != nil {
			session.ExpiresAt = user.AuthConfig.ExpiresAt
		} else {
			// Default OAuth session to 1 hour
			expiresAt := time.Now().Add(1 * time.Hour)
			session.ExpiresAt = &expiresAt
		}
	default:
		// Most sessions expire in 24 hours
		expiresAt := time.Now().Add(24 * time.Hour)
		session.ExpiresAt = &expiresAt
	}

	// Save session
	err := sm.saveSession(session)
	if err != nil {
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	return session, nil
}

// GetSession retrieves a session by user ID
func (sm *SessionManager) GetSession(userID string) (*UserSession, error) {
	sessionPath := sm.getSessionPath(userID)
	
	if _, err := os.Stat(sessionPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("session not found for user: %s", userID)
	}

	data, err := os.ReadFile(sessionPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read session: %w", err)
	}

	var session UserSession
	err = json.Unmarshal(data, &session)
	if err != nil {
		return nil, fmt.Errorf("failed to parse session: %w", err)
	}

	// Check if session is expired
	if session.ExpiresAt != nil && session.ExpiresAt.Before(time.Now()) {
		session.IsActive = false
		sm.saveSession(&session) // Update the saved session
		return &session, fmt.Errorf("session expired")
	}

	return &session, nil
}

// UpdateSession updates an existing session
func (sm *SessionManager) UpdateSession(session *UserSession) error {
	session.LastUsedAt = time.Now()
	return sm.saveSession(session)
}

// EndSession terminates a user session
func (sm *SessionManager) EndSession(userID string) error {
	session, err := sm.GetSession(userID)
	if err != nil {
		return err // Session might already be ended or not exist
	}

	session.IsActive = false
	return sm.saveSession(session)
}

// CleanupExpiredSessions removes expired session files
func (sm *SessionManager) CleanupExpiredSessions() error {
	sessionsDir := sm.getSessionsDir()
	
	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Sessions directory doesn't exist yet
		}
		return fmt.Errorf("failed to read sessions directory: %w", err)
	}

	now := time.Now()
	cleaned := 0

	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			sessionPath := filepath.Join(sessionsDir, entry.Name())
			
			// Try to read and check expiration
			data, err := os.ReadFile(sessionPath)
			if err != nil {
				continue // Skip files we can't read
			}

			var session UserSession
			err = json.Unmarshal(data, &session)
			if err != nil {
				continue // Skip invalid session files
			}

			// Remove if expired
			if session.ExpiresAt != nil && session.ExpiresAt.Before(now) {
				os.Remove(sessionPath)
				cleaned++
			}
		}
	}

	if cleaned > 0 {
		fmt.Printf("🧹 Cleaned up %d expired sessions\n", cleaned)
	}

	return nil
}

// ListActiveSessions returns all active sessions
func (sm *SessionManager) ListActiveSessions() ([]*UserSession, error) {
	sessionsDir := sm.getSessionsDir()
	
	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*UserSession{}, nil // No sessions yet
		}
		return nil, fmt.Errorf("failed to read sessions directory: %w", err)
	}

	var activeSessions []*UserSession
	now := time.Now()

	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			sessionPath := filepath.Join(sessionsDir, entry.Name())
			
			data, err := os.ReadFile(sessionPath)
			if err != nil {
				continue // Skip files we can't read
			}

			var session UserSession
			err = json.Unmarshal(data, &session)
			if err != nil {
				continue // Skip invalid session files
			}

			// Only include active, non-expired sessions
			if session.IsActive && (session.ExpiresAt == nil || session.ExpiresAt.After(now)) {
				activeSessions = append(activeSessions, &session)
			}
		}
	}

	return activeSessions, nil
}

// SimulateAuthentication simulates authentication for a test user
func (sm *SessionManager) SimulateAuthentication(user *config.TestUser) *AuthenticationResult {
	result := &AuthenticationResult{
		Headers: make(map[string]string),
	}

	switch user.AuthType {
	case "none":
		result.Success = true
		result.Message = "No authentication required"

	case "basic":
		if user.AuthConfig != nil && user.AuthConfig.Username != "" && user.AuthConfig.Password != "" {
			result.Success = true
			result.Message = "Basic authentication simulated"
			// Add basic auth header
			result.Headers["Authorization"] = fmt.Sprintf("Basic %s", encodeBasicAuth(user.AuthConfig.Username, user.AuthConfig.Password))
		} else {
			result.Success = false
			result.Message = "Missing basic auth credentials"
		}

	case "bearer":
		if user.AuthConfig != nil && user.AuthConfig.Token != "" {
			result.Success = true
			result.Message = "Bearer token authentication simulated"
			result.Headers["Authorization"] = fmt.Sprintf("Bearer %s", user.AuthConfig.Token)
		} else {
			result.Success = false
			result.Message = "Missing bearer token"
		}

	case "oauth":
		// OAuth simulation - would typically involve redirect flow
		result.Success = true
		result.Message = "OAuth authentication would be initiated"
		result.RedirectURL = fmt.Sprintf("/auth/%s/callback", user.AuthConfig.Provider)

	case "magic_link":
		// Magic link simulation
		result.Success = true
		result.Message = "Magic link would be sent to " + user.Email

	case "username_password":
		// Username/password form simulation
		result.Success = true
		result.Message = "Form login simulated"
		// Add session cookie
		result.Cookies = []config.Cookie{
			{
				Name:  "session_id",
				Value: fmt.Sprintf("sess_%d", time.Now().Unix()),
				Path:  "/",
			},
		}

	default:
		result.Success = false
		result.Message = fmt.Sprintf("Unknown auth type: %s", user.AuthType)
	}

	if result.Success {
		// Create session
		session, err := sm.StartSession(user)
		if err != nil {
			result.Success = false
			result.Message = "Failed to create session: " + err.Error()
			result.Error = err
		} else {
			result.Session = session
			// Copy cookies and headers to session
			if len(result.Cookies) > 0 {
				session.Cookies = result.Cookies
			}
			if len(result.Headers) > 0 {
				session.Headers = result.Headers
			}
			sm.UpdateSession(session)
		}
	}

	return result
}

// saveSession saves a session to disk
func (sm *SessionManager) saveSession(session *UserSession) error {
	sessionsDir := sm.getSessionsDir()
	err := os.MkdirAll(sessionsDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create sessions directory: %w", err)
	}

	sessionPath := sm.getSessionPath(session.UserID)
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	return os.WriteFile(sessionPath, data, 0644)
}

// getSessionsDir returns the sessions directory path
func (sm *SessionManager) getSessionsDir() string {
	return filepath.Join(sm.projectDir, ".tod", "sessions")
}

// getSessionPath returns the path for a specific user's session file
func (sm *SessionManager) getSessionPath(userID string) string {
	return filepath.Join(sm.getSessionsDir(), fmt.Sprintf("%s.json", userID))
}

// encodeBasicAuth encodes username and password for basic auth (simplified)
func encodeBasicAuth(username, password string) string {
	// In a real implementation, this would use base64 encoding
	// For simulation purposes, we'll just return a placeholder
	return fmt.Sprintf("encoded_%s:%s", username, password)
}