package users

import (
	"context"
	"fmt"
	"time"

	"github.com/ciciliostudio/tod/internal/config"
	"github.com/ciciliostudio/tod/internal/email"
	"github.com/ciciliostudio/tod/internal/llm"
)

// AuthFlowManager handles enhanced authentication flows with email checking
type AuthFlowManager struct {
	sessionManager *SessionManager
	emailClient    *email.Client
	extractor      *email.ExtractorService
	llmClient      llm.Client
	projectDir     string
}

// NewAuthFlowManager creates a new enhanced auth flow manager
func NewAuthFlowManager(projectDir string, llmClient llm.Client) (*AuthFlowManager, error) {
	sessionManager := NewSessionManager(projectDir)
	
	// Try to create email client (optional)
	emailClient, err := email.NewClient(projectDir)
	if err != nil {
		// Email not configured - that's OK, we'll just skip email checking
		emailClient = nil
	}
	
	var extractor *email.ExtractorService
	if emailClient != nil {
		extractor = email.NewExtractorService(llmClient)
	}

	return &AuthFlowManager{
		sessionManager: sessionManager,
		emailClient:    emailClient,
		extractor:      extractor,
		llmClient:      llmClient,
		projectDir:     projectDir,
	}, nil
}

// AuthenticateWithEmailSupport performs authentication with automatic email checking
func (a *AuthFlowManager) AuthenticateWithEmailSupport(user *config.TestUser) *AuthenticationResult {
	// Start with basic authentication simulation
	result := a.sessionManager.SimulateAuthentication(user)
	
	// If email is configured and user has email, enhance with email checking
	if a.emailClient != nil && user.Email != "" && a.needsEmailCheck(user.AuthType) {
		fmt.Printf("📧 Email checking enabled for %s (%s)\n", user.Name, user.Email)
		
		// Perform email-enhanced authentication
		emailResult := a.performEmailAuthentication(user, result)
		if emailResult != nil {
			return emailResult
		}
	}
	
	return result
}

// needsEmailCheck determines if an auth type requires email checking
func (a *AuthFlowManager) needsEmailCheck(authType string) bool {
	switch authType {
	case "magic_link", "email_verification", "2fa", "sms":
		return true
	default:
		return false
	}
}

// performEmailAuthentication handles authentication flows that require email checking
func (a *AuthFlowManager) performEmailAuthentication(user *config.TestUser, baseResult *AuthenticationResult) *AuthenticationResult {
	switch user.AuthType {
	case "magic_link":
		return a.handleMagicLinkAuth(user, baseResult)
	case "email_verification":
		return a.handleEmailVerification(user, baseResult)
	case "2fa", "sms":
		return a.handle2FAAuth(user, baseResult)
	default:
		return baseResult
	}
}

// handleMagicLinkAuth handles magic link authentication with email checking
func (a *AuthFlowManager) handleMagicLinkAuth(user *config.TestUser, baseResult *AuthenticationResult) *AuthenticationResult {
	fmt.Printf("🔗 Waiting for magic link email for %s...\n", user.Email)
	
	// Wait for magic link email
	context := fmt.Sprintf("User '%s' just clicked 'Send Magic Link' button. Looking for magic link email.", user.Name)
	
	extractResult, err := a.extractor.WaitForAuthEmail(
		a.emailClient,
		email.AuthTypeMagicLink,
		context,
		30*time.Second, // Wait up to 30 seconds
	)
	
	if err != nil || !extractResult.Success {
		result := &AuthenticationResult{
			Success: false,
			Message: fmt.Sprintf("Magic link email not found: %s", extractResult.Error),
			Error:   fmt.Errorf("magic link email timeout"),
		}
		return result
	}
	
	fmt.Printf("✅ Found magic link: %s\n", extractResult.Value)
	
	// Update the result with magic link
	enhancedResult := *baseResult
	enhancedResult.Success = true
	enhancedResult.Message = "Magic link authentication ready"
	enhancedResult.RedirectURL = extractResult.Value
	
	// Store the magic link in session data
	if enhancedResult.Session != nil {
		enhancedResult.Session.SessionData["magic_link"] = extractResult.Value
		enhancedResult.Session.SessionData["auth_method"] = "email_verified"
		a.sessionManager.UpdateSession(enhancedResult.Session)
	}
	
	return &enhancedResult
}

// handleEmailVerification handles email verification code authentication
func (a *AuthFlowManager) handleEmailVerification(user *config.TestUser, baseResult *AuthenticationResult) *AuthenticationResult {
	fmt.Printf("📨 Waiting for verification code email for %s...\n", user.Email)
	
	context := fmt.Sprintf("User '%s' just requested an email verification code. Looking for verification code email.", user.Name)
	
	extractResult, err := a.extractor.WaitForAuthEmail(
		a.emailClient,
		email.AuthTypeVerificationCode,
		context,
		30*time.Second,
	)
	
	if err != nil || !extractResult.Success {
		result := &AuthenticationResult{
			Success: false,
			Message: fmt.Sprintf("Verification code email not found: %s", extractResult.Error),
			Error:   fmt.Errorf("verification code email timeout"),
		}
		return result
	}
	
	fmt.Printf("✅ Found verification code: %s\n", extractResult.Value)
	
	// Update the result with verification code
	enhancedResult := *baseResult
	enhancedResult.Success = true
	enhancedResult.Message = "Email verification code retrieved"
	
	// Store the verification code in session data
	if enhancedResult.Session != nil {
		enhancedResult.Session.SessionData["verification_code"] = extractResult.Value
		enhancedResult.Session.SessionData["auth_method"] = "email_verified"
		a.sessionManager.UpdateSession(enhancedResult.Session)
	}
	
	return &enhancedResult
}

// handle2FAAuth handles 2FA/SMS code authentication
func (a *AuthFlowManager) handle2FAAuth(user *config.TestUser, baseResult *AuthenticationResult) *AuthenticationResult {
	fmt.Printf("🔐 Waiting for 2FA code email for %s...\n", user.Email)
	
	context := fmt.Sprintf("User '%s' just triggered 2FA authentication. Looking for 2FA code email or SMS forwarded to email.", user.Name)
	
	extractResult, err := a.extractor.WaitForAuthEmail(
		a.emailClient,
		email.AuthType2FA,
		context,
		30*time.Second,
	)
	
	if err != nil || !extractResult.Success {
		result := &AuthenticationResult{
			Success: false,
			Message: fmt.Sprintf("2FA code email not found: %s", extractResult.Error),
			Error:   fmt.Errorf("2FA code email timeout"),
		}
		return result
	}
	
	fmt.Printf("✅ Found 2FA code: %s\n", extractResult.Value)
	
	// Update the result with 2FA code
	enhancedResult := *baseResult
	enhancedResult.Success = true
	enhancedResult.Message = "2FA code retrieved from email"
	
	// Store the 2FA code in session data
	if enhancedResult.Session != nil {
		enhancedResult.Session.SessionData["2fa_code"] = extractResult.Value
		enhancedResult.Session.SessionData["auth_method"] = "email_2fa"
		a.sessionManager.UpdateSession(enhancedResult.Session)
	}
	
	return &enhancedResult
}

// GetSessionManager returns the underlying session manager
func (a *AuthFlowManager) GetSessionManager() *SessionManager {
	return a.sessionManager
}

// IsEmailConfigured returns true if email checking is available
func (a *AuthFlowManager) IsEmailConfigured() bool {
	return a.emailClient != nil
}

// GetConfiguredEmail returns the configured email address
func (a *AuthFlowManager) GetConfiguredEmail() string {
	if a.emailClient != nil {
<<<<<<< HEAD
		return a.emailClient.GetUsername()
=======
		return a.emailClient.userEmail
>>>>>>> origin/main
	}
	return ""
}

// TestEmailAccess tests if email access is working
func (a *AuthFlowManager) TestEmailAccess() error {
	if a.emailClient == nil {
		return fmt.Errorf("email not configured")
	}
	
	return a.emailClient.TestConnection()
}