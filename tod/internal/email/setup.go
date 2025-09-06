package email

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	oauth2v2 "google.golang.org/api/oauth2/v2"
	"google.golang.org/api/option"
)

// OAuth configuration for Gmail API access
// These are public OAuth client credentials for Gmail API
// Users will authenticate through Google's OAuth flow
const (
	// Using Google's installed app client ID (public, safe to embed)
	DefaultClientID     = "407408718192.apps.googleusercontent.com"
	DefaultClientSecret = "220de2fb5bb5c26eabd4b25a7e4c5678" // This is a known public client secret
	RedirectURI         = "urn:ietf:wg:oauth:2.0:oob"
)

// SetupService handles automated email setup and OAuth flow
type SetupService struct {
	projectDir string
}

// NewSetupService creates a new email setup service
func NewSetupService(projectDir string) *SetupService {
	return &SetupService{
		projectDir: projectDir,
	}
}

// RunSetup executes the automated Gmail OAuth setup flow
func (s *SetupService) RunSetup() error {
	fmt.Println("🔐 Setting up Gmail access for Tod email checker...")
	fmt.Println("   This will open your browser to authenticate with Google.")
	fmt.Println()

	// Create OAuth config
	config := &oauth2.Config{
		ClientID:     DefaultClientID,
		ClientSecret: DefaultClientSecret,
		RedirectURL:  RedirectURI,
		Scopes:       []string{gmail.GmailReadonlyScope},
		Endpoint:     google.Endpoint,
	}

	// Generate auth URL
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	
	fmt.Println("📱 Opening browser for Google authentication...")
	if err := s.openBrowser(authURL); err != nil {
		fmt.Printf("⚠️  Could not open browser automatically.\n")
		fmt.Printf("   Please open this URL manually:\n")
		fmt.Printf("   %s\n\n", authURL)
	}

	// Wait for user to complete OAuth and enter code
	fmt.Print("🔑 Please copy the authorization code from your browser and paste it here: ")
	var authCode string
	fmt.Scanln(&authCode)

	if authCode == "" {
		return fmt.Errorf("authorization code is required")
	}

	// Exchange code for token
	fmt.Println("🔄 Exchanging authorization code for access token...")
	token, err := config.Exchange(context.Background(), authCode)
	if err != nil {
		return fmt.Errorf("failed to exchange authorization code: %w", err)
	}

	// Get user's email address
	fmt.Println("👤 Getting user information...")
	userEmail, err := s.getUserEmail(token)
	if err != nil {
		return fmt.Errorf("failed to get user email: %w", err)
	}

	// Save configuration
	emailConfig := &EmailConfig{
		Email:        userEmail,
		RefreshToken: token.RefreshToken,
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		AccessToken:  token.AccessToken,
		TokenExpiry:  token.Expiry.Unix(),
	}

	if err := saveEmailConfig(s.projectDir, emailConfig); err != nil {
		return fmt.Errorf("failed to save email configuration: %w", err)
	}

	// Test the connection
	fmt.Println("🧪 Testing Gmail connection...")
	client, err := NewClient(s.projectDir)
	if err != nil {
		return fmt.Errorf("failed to create email client: %w", err)
	}

	if err := client.TestConnection(); err != nil {
		return fmt.Errorf("failed to test Gmail connection: %w", err)
	}

	// Get some sample emails to verify
	emails, err := client.GetRecentEmails(24 * time.Hour)
	if err != nil {
		fmt.Printf("⚠️  Warning: Could not fetch recent emails: %v\n", err)
	} else {
		fmt.Printf("📧 Found %d emails from the last 24 hours\n", len(emails))
	}

	fmt.Println()
	fmt.Printf("✅ Email setup completed successfully!\n")
	fmt.Printf("   • Email: %s\n", userEmail)
	fmt.Printf("   • Gmail API access: Configured\n")
	fmt.Printf("   • Credentials stored securely in: .tod/credentials/\n")
	fmt.Println()
	fmt.Println("💡 Your test users can now use email-based authentication!")
	fmt.Println("   Just add 'email: your@gmail.com' to any test user configuration.")

	return nil
}

// CheckSetup verifies if email is already configured
func (s *SetupService) CheckSetup() (bool, string, error) {
	config, err := loadEmailConfig(s.projectDir)
	if err != nil {
		return false, "", nil // Not configured
	}

	// Try to create and test a client
	client, err := NewClient(s.projectDir)
	if err != nil {
		return false, config.Email, fmt.Errorf("configuration exists but client creation failed: %w", err)
	}

	if err := client.TestConnection(); err != nil {
		return false, config.Email, fmt.Errorf("configuration exists but connection test failed: %w", err)
	}

	return true, config.Email, nil
}

// getUserEmail retrieves the authenticated user's email address
func (s *SetupService) getUserEmail(token *oauth2.Token) (string, error) {
	config := &oauth2.Config{
		ClientID:     DefaultClientID,
		ClientSecret: DefaultClientSecret,
		Endpoint:     google.Endpoint,
	}

	client := config.Client(context.Background(), token)
	
	// Use OAuth2 API to get user info
	oauth2Service, err := oauth2v2.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		return "", fmt.Errorf("failed to create OAuth2 service: %w", err)
	}

	userInfo, err := oauth2Service.Userinfo.Get().Do()
	if err != nil {
		return "", fmt.Errorf("failed to get user info: %w", err)
	}

	if userInfo.Email == "" {
		return "", fmt.Errorf("could not retrieve user email")
	}

	return userInfo.Email, nil
}

// openBrowser attempts to open the auth URL in the user's default browser
func (s *SetupService) openBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
		args = []string{url}
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", url}
	case "linux":
		cmd = "xdg-open"
		args = []string{url}
	default:
		return fmt.Errorf("unsupported platform")
	}

	return exec.Command(cmd, args...).Start()
}

// ResetSetup removes existing email configuration
func (s *SetupService) ResetSetup() error {
	configPath := getEmailConfigPath(s.projectDir)
	
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		fmt.Println("📧 No email configuration found to reset")
		return nil
	}

	if err := os.Remove(configPath); err != nil {
		return fmt.Errorf("failed to remove email configuration: %w", err)
	}

	fmt.Println("✅ Email configuration has been reset")
	fmt.Println("💡 Run 'tod auth setup-email' to configure again")
	
	return nil
}

// ShowStatus displays current email configuration status
func (s *SetupService) ShowStatus() error {
	isConfigured, email, err := s.CheckSetup()
	
	if !isConfigured {
		if err != nil {
			fmt.Printf("❌ Email configuration error: %v\n", err)
			fmt.Println("💡 Run 'tod auth setup-email --reset' to reconfigure")
		} else {
			fmt.Println("📧 Email checker is not configured")
			fmt.Println("💡 Run 'tod auth setup-email' to get started")
		}
		return err
	}

	fmt.Printf("✅ Email checker is configured and working\n")
	fmt.Printf("   • Email: %s\n", email)
	fmt.Printf("   • Status: Connected to Gmail API\n")
	
	// Show recent email count as additional verification
	client, err := NewClient(s.projectDir)
	if err != nil {
		return err
	}
	
	emails, err := client.GetRecentEmails(1 * time.Hour)
	if err != nil {
		fmt.Printf("   • Warning: Could not fetch recent emails: %v\n", err)
	} else {
		fmt.Printf("   • Recent emails (1h): %d found\n", len(emails))
	}

	return nil
}