package email

import (
	"fmt"
	"time"

	"github.com/lance13c/tod/internal/logging"
)

// NewIMAPClientWrapper creates a Client that uses IMAP instead of Gmail API
func NewIMAPClientWrapper(projectDir string) (*Client, error) {
	config := LoadIMAPConfig(projectDir)
	if config.Host == "" || config.Username == "" {
		return nil, fmt.Errorf("IMAP not configured")
	}
	
	// Create an IMAP monitor to use for fetching
	monitor, err := NewIMAPMonitor(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create IMAP monitor: %w", err)
	}
	
	if err := monitor.Connect(); err != nil {
		return nil, fmt.Errorf("failed to connect to IMAP: %w", err)
	}
	
	// Create a wrapper client
	return &Client{
		service:    nil, // No Gmail service
		userEmail:  config.Username,
		projectDir: projectDir,
		imapMonitor: monitor, // Store the IMAP monitor
	}, nil
}

// GetRecentEmailsViaIMAP fetches recent emails using IMAP
func (c *Client) GetRecentEmailsViaIMAP(since time.Duration) ([]*Email, error) {
	if c.imapMonitor == nil {
		return nil, fmt.Errorf("IMAP not configured")
	}
	
	minutes := int(since.Minutes())
	if minutes < 1 {
		minutes = 1
	}
	
	logging.Info("[IMAP CLIENT] Checking emails from last %d minutes", minutes)
	
	// Use the CheckRecentEmails method to look for magic links
	magicLink, err := c.imapMonitor.CheckRecentEmails(minutes)
	if err != nil {
		return nil, err
	}
	
	// If we found a magic link, create a pseudo-email with it
	if magicLink != "" {
		email := &Email{
			ID:       "imap-magic-link",
			From:     "noreply",
			To:       c.userEmail,
			Subject:  "Magic Link",
			Body:     magicLink,
			Snippet:  magicLink,
			Received: time.Now(),
		}
		return []*Email{email}, nil
	}
	
	// No magic link found
	return []*Email{}, nil
}