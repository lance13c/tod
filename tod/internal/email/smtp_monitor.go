package email

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"
)

// SMTPMonitor monitors IMAP server for incoming emails with magic links
type SMTPMonitor struct {
	// IMAP configuration
	host     string
	port     string
	username string
	password string
	useTLS   bool
	
	// Monitoring state
	client        *client.Client
	lastMessageID uint32
	pollInterval  time.Duration
	
	// Callbacks
	onMagicLink func(url string) error
}

// SMTPConfig holds SMTP/IMAP configuration
type SMTPConfig struct {
	Host         string
	Port         string
	Username     string
	Password     string
	UseTLS       bool
	PollInterval time.Duration
}

// GetUsername returns the configured username
func (s *SMTPMonitor) GetUsername() string {
	return s.username
}

// NewSMTPMonitor creates a new SMTP/IMAP monitor
func NewSMTPMonitor(config *SMTPConfig) (*SMTPMonitor, error) {
	if config.PollInterval == 0 {
		config.PollInterval = 3 * time.Second
	}
	
	monitor := &SMTPMonitor{
		host:         config.Host,
		port:         config.Port,
		username:     config.Username,
		password:     config.Password,
		useTLS:       config.UseTLS,
		pollInterval: config.PollInterval,
	}
	
	return monitor, nil
}

// LoadSMTPConfigFromEnv loads SMTP configuration from environment variables
func LoadSMTPConfigFromEnv() *SMTPConfig {
	return &SMTPConfig{
		Host:         getEnvOrDefault("SMTP_HOST", "smtps-proxy.fastmail.com"),
		Port:         getEnvOrDefault("SMTP_PORT", "993"), // IMAP SSL port
		Username:     os.Getenv("SMTP_USER"),
		Password:     os.Getenv("SMTP_PASS"),
		UseTLS:       os.Getenv("SMTP_SECURE") == "true",
		PollInterval: 3 * time.Second,
	}
}

// LoadSMTPConfigFromFile loads SMTP configuration from config file
func LoadSMTPConfigFromFile(configData map[string]interface{}) *SMTPConfig {
	config := &SMTPConfig{
		PollInterval: 3 * time.Second,
	}
	
	if emailConfig, ok := configData["email"].(map[string]interface{}); ok {
		if host, ok := emailConfig["smtp_host"].(string); ok {
			config.Host = host
		}
		if port, ok := emailConfig["smtp_port"].(int); ok {
			config.Port = fmt.Sprintf("%d", port)
		} else if port, ok := emailConfig["smtp_port"].(string); ok {
			config.Port = port
		}
		if user, ok := emailConfig["smtp_user"].(string); ok {
			config.Username = user
		}
		if pass, ok := emailConfig["smtp_pass"].(string); ok {
			config.Password = pass
		}
		if secure, ok := emailConfig["smtp_secure"].(bool); ok {
			config.UseTLS = secure
		}
	}
	
	// Fall back to environment variables if not in config
	if config.Host == "" {
		config.Host = getEnvOrDefault("SMTP_HOST", "smtps-proxy.fastmail.com")
	}
	if config.Port == "" {
		config.Port = getEnvOrDefault("SMTP_PORT", "993")
	}
	if config.Username == "" {
		config.Username = os.Getenv("SMTP_USER")
	}
	if config.Password == "" {
		config.Password = os.Getenv("SMTP_PASS")
	}
	
	return config
}

// Connect establishes connection to the IMAP server
func (m *SMTPMonitor) Connect() error {
	address := fmt.Sprintf("%s:%s", m.host, m.port)
	
	var c *client.Client
	var err error
	
	if m.useTLS || m.port == "993" {
		// Connect with TLS
		c, err = client.DialTLS(address, &tls.Config{
			ServerName: m.host,
		})
	} else {
		// Connect without TLS
		c, err = client.Dial(address)
	}
	
	if err != nil {
		return fmt.Errorf("failed to connect to IMAP server: %w", err)
	}
	
	// Login
	if err := c.Login(m.username, m.password); err != nil {
		c.Logout()
		return fmt.Errorf("failed to login: %w", err)
	}
	
	m.client = c
	
	// Select INBOX
	_, err = c.Select("INBOX", false)
	if err != nil {
		return fmt.Errorf("failed to select INBOX: %w", err)
	}
	
	// Get the latest message ID to start monitoring from
	m.updateLastMessageID()
	
	return nil
}

// Disconnect closes the connection to the IMAP server
func (m *SMTPMonitor) Disconnect() error {
	if m.client != nil {
		return m.client.Logout()
	}
	return nil
}

// StartMonitoring starts monitoring for new emails with magic links (blocking)
func (m *SMTPMonitor) StartMonitoring(onMagicLink func(url string) error) error {
	m.onMagicLink = onMagicLink
	
	if m.client == nil {
		if err := m.Connect(); err != nil {
			return err
		}
	}
	
	log.Println("Starting email monitoring for magic links...")
	
	ticker := time.NewTicker(m.pollInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			if err := m.checkNewEmails(); err != nil {
				log.Printf("Error checking emails: %v", err)
				// Try to reconnect
				if err := m.Connect(); err != nil {
					log.Printf("Failed to reconnect: %v", err)
				}
			}
		}
	}
}

// StartMonitoringBackground starts monitoring for new emails in the background
func (m *SMTPMonitor) StartMonitoringBackground(onMagicLink func(url string) error) (chan struct{}, error) {
	m.onMagicLink = onMagicLink
	
	if m.client == nil {
		if err := m.Connect(); err != nil {
			return nil, err
		}
	}
	
	log.Println("Starting background email monitoring for magic links...")
	
	stopChan := make(chan struct{})
	
	go func() {
		ticker := time.NewTicker(m.pollInterval)
		defer ticker.Stop()
		defer m.Disconnect()
		
		for {
			select {
			case <-stopChan:
				log.Println("Stopping email monitoring...")
				return
			case <-ticker.C:
				if err := m.checkNewEmails(); err != nil {
					log.Printf("Error checking emails: %v", err)
					// Try to reconnect
					if err := m.Connect(); err != nil {
						log.Printf("Failed to reconnect: %v", err)
					}
				}
			}
		}
	}()
	
	return stopChan, nil
}

// checkNewEmails checks for new emails and extracts magic links
func (m *SMTPMonitor) checkNewEmails() error {
	// Get current mailbox status
	status, err := m.client.Status("INBOX", []imap.StatusItem{imap.StatusMessages})
	if err != nil {
		return err
	}
	
	if status.Messages == 0 {
		return nil
	}
	
	// Check for new messages
	from := m.lastMessageID + 1
	to := status.Messages
	
	if from > to {
		// No new messages
		return nil
	}
	
	seqset := new(imap.SeqSet)
	seqset.AddRange(from, to)
	
	// Fetch the messages
	messages := make(chan *imap.Message, 10)
	done := make(chan error, 1)
	
	go func() {
		done <- m.client.Fetch(seqset, []imap.FetchItem{imap.FetchEnvelope, imap.FetchBody}, messages)
	}()
	
	// Process messages
	for msg := range messages {
		if msg == nil {
			continue
		}
		
		// Get the email body
		body := msg.GetBody(&imap.BodySectionName{})
		if body == nil {
			continue
		}
		
		// Parse the message
		mr, err := mail.CreateReader(body)
		if err != nil {
			log.Printf("Failed to parse message: %v", err)
			continue
		}
		
		// Extract text from all parts
		var emailContent strings.Builder
		for {
			p, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Printf("Failed to read part: %v", err)
				continue
			}
			
			switch h := p.Header.(type) {
			case *mail.InlineHeader:
				// Read the body
				b, _ := io.ReadAll(p.Body)
				emailContent.Write(b)
			case *mail.AttachmentHeader:
				// Skip attachments
				_ = h
			}
		}
		
		// Extract magic link from email content
		content := emailContent.String()
		if link := extractMagicLinkFromContent(content); link != "" {
			log.Printf("Found magic link: %s", link)
			if m.onMagicLink != nil {
				if err := m.onMagicLink(link); err != nil {
					log.Printf("Error handling magic link: %v", err)
				}
			}
		}
		
		// Update last message ID
		if msg.SeqNum > m.lastMessageID {
			m.lastMessageID = msg.SeqNum
		}
	}
	
	if err := <-done; err != nil {
		return err
	}
	
	return nil
}

// updateLastMessageID updates the last message ID to the current latest
func (m *SMTPMonitor) updateLastMessageID() {
	status, err := m.client.Status("INBOX", []imap.StatusItem{imap.StatusMessages})
	if err == nil && status.Messages > 0 {
		m.lastMessageID = status.Messages
	}
}

// extractMagicLinkFromContent extracts magic link URLs from email content
func extractMagicLinkFromContent(content string) string {
	// First try: Look for URLs with auth-related keywords
	authURLRegex := regexp.MustCompile(`https?://[^\s<>"']+(?:verify|auth|login|confirm|activate|magic|token)[^\s<>"']*`)
	if matches := authURLRegex.FindAllString(content, -1); len(matches) > 0 {
		return matches[0]
	}
	
	// Second try: Look for any HTTPS URL that's not unsubscribe/privacy/terms
	generalURLRegex := regexp.MustCompile(`https://[^\s<>"']+`)
	matches := generalURLRegex.FindAllString(content, -1)
	
	for _, url := range matches {
		lowerURL := strings.ToLower(url)
		if !strings.Contains(lowerURL, "unsubscribe") &&
		   !strings.Contains(lowerURL, "privacy") &&
		   !strings.Contains(lowerURL, "terms") &&
		   !strings.Contains(lowerURL, "preferences") &&
		   !strings.Contains(lowerURL, "email-settings") {
			return url
		}
	}
	
	return ""
}

// getEnvOrDefault gets environment variable or returns default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}