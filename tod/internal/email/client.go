package email

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

// Client provides email access functionality
type Client struct {
	service    *gmail.Service
	userEmail  string
	projectDir string
}

// Email represents a retrieved email message
type Email struct {
	ID       string    `json:"id"`
	Subject  string    `json:"subject"`
	From     string    `json:"from"`
	To       string    `json:"to"`
	Body     string    `json:"body"`
	Received time.Time `json:"received"`
	Snippet  string    `json:"snippet"`
}

// EmailConfig stores OAuth credentials for email access
type EmailConfig struct {
	Email        string `json:"email"`
	RefreshToken string `json:"refresh_token"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	AccessToken  string `json:"access_token,omitempty"`
	TokenExpiry  int64  `json:"token_expiry,omitempty"`
}

// NewClient creates a new email client with stored credentials
func NewClient(projectDir string) (*Client, error) {
	config, err := loadEmailConfig(projectDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load email config: %w", err)
	}

	service, err := createGmailService(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Gmail service: %w", err)
	}

	return &Client{
		service:    service,
		userEmail:  config.Email,
		projectDir: projectDir,
	}, nil
}

// GetRecentEmails fetches emails from the last specified duration
func (c *Client) GetRecentEmails(since time.Duration) ([]*Email, error) {
	// Calculate the query time
	after := time.Now().Add(-since).Unix()
	
	// Build the Gmail API query
	query := fmt.Sprintf("after:%d", after)
	
	// Get messages
	messages, err := c.service.Users.Messages.List("me").Q(query).MaxResults(10).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list messages: %w", err)
	}

	var emails []*Email
	for _, message := range messages.Messages {
		email, err := c.getEmailDetails(message.Id)
		if err != nil {
			continue // Skip messages we can't read
		}
		emails = append(emails, email)
	}

	return emails, nil
}

// GetEmailsSince fetches emails received since a specific time
func (c *Client) GetEmailsSince(since time.Time) ([]*Email, error) {
	return c.GetRecentEmails(time.Since(since))
}

// getEmailDetails retrieves full details for a specific message
func (c *Client) getEmailDetails(messageID string) (*Email, error) {
	message, err := c.service.Users.Messages.Get("me", messageID).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	email := &Email{
		ID:      message.Id,
		Snippet: message.Snippet,
	}

	// Parse the message timestamp
	email.Received = time.Unix(message.InternalDate/1000, 0)

	// Extract headers and body
	email.extractHeadersAndBody(message.Payload)

	return email, nil
}

// extractHeadersAndBody extracts subject, from, to, and body from message payload
func (e *Email) extractHeadersAndBody(payload *gmail.MessagePart) {
	// Extract headers
	for _, header := range payload.Headers {
		switch header.Name {
		case "Subject":
			e.Subject = header.Value
		case "From":
			e.From = header.Value
		case "To":
			e.To = header.Value
		}
	}

	// Extract body content
	e.Body = e.extractTextBody(payload)
}

// extractTextBody recursively extracts text content from message parts
func (e *Email) extractTextBody(part *gmail.MessagePart) string {
	if part.Body != nil && part.Body.Data != "" {
		// Decode base64url data
		if decoded, err := decodeBase64URL(part.Body.Data); err == nil {
			return decoded
		}
	}

	// Check parts recursively
	var body string
	for _, subPart := range part.Parts {
		if subPart.MimeType == "text/plain" || subPart.MimeType == "text/html" {
			if subContent := e.extractTextBody(subPart); subContent != "" {
				body += subContent + "\n"
			}
		}
	}

	return body
}

// TestConnection verifies that the email client can access Gmail
func (c *Client) TestConnection() error {
	_, err := c.service.Users.Messages.List("me").MaxResults(1).Do()
	if err != nil {
		return fmt.Errorf("failed to test Gmail connection: %w", err)
	}
	return nil
}

// GetUserEmail returns the configured user email
func (c *Client) GetUserEmail() string {
	return c.userEmail
}

// createGmailService creates a Gmail API service from stored config
func createGmailService(config *EmailConfig) (*gmail.Service, error) {
	oauthConfig := &oauth2.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		RedirectURL:  "urn:ietf:wg:oauth:2.0:oob",
		Scopes:       []string{gmail.GmailReadonlyScope},
		Endpoint:     google.Endpoint,
	}

	token := &oauth2.Token{
		RefreshToken: config.RefreshToken,
	}

	// Add access token if available
	if config.AccessToken != "" && config.TokenExpiry > time.Now().Unix() {
		token.AccessToken = config.AccessToken
		token.Expiry = time.Unix(config.TokenExpiry, 0)
	}

	client := oauthConfig.Client(context.Background(), token)
	
	service, err := gmail.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gmail service: %w", err)
	}

	return service, nil
}

// loadEmailConfig loads stored email configuration
func loadEmailConfig(projectDir string) (*EmailConfig, error) {
	configPath := getEmailConfigPath(projectDir)
	
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("email not configured - run 'tod auth setup-email' first")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read email config: %w", err)
	}

	var config EmailConfig
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse email config: %w", err)
	}

	return &config, nil
}

// saveEmailConfig saves email configuration to disk
func saveEmailConfig(projectDir string, config *EmailConfig) error {
	configPath := getEmailConfigPath(projectDir)
	
	// Ensure directory exists
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	return os.WriteFile(configPath, data, 0600) // Restrict permissions for security
}

// getEmailConfigPath returns the path to email configuration file
func getEmailConfigPath(projectDir string) string {
	return filepath.Join(projectDir, ".tod", "credentials", "email.json")
}

// decodeBase64URL decodes Gmail's base64url encoded content
func decodeBase64URL(data string) (string, error) {
	// Gmail uses base64url encoding without padding
	// Convert to standard base64 by replacing chars and adding padding
	data = data + "===" // Add max padding, extras will be ignored
	for i := 0; i < len(data); i++ {
		switch data[i] {
		case '-':
			data = data[:i] + "+" + data[i+1:]
		case '_':
			data = data[:i] + "/" + data[i+1:]
		}
	}
	
	// Now decode as standard base64
	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return "", err
	}
	
	return string(decoded), nil
}