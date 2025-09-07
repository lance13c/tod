package email

import (
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/ciciliostudio/tod/internal/logging"
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"
)

// IMAPMonitor monitors IMAP server for incoming emails with magic links
type IMAPMonitor struct {
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

// IMAPConfig holds IMAP configuration for reading emails
type IMAPConfig struct {
	Host         string
	Port         string
	Username     string
	Password     string
	UseTLS       bool
	PollInterval time.Duration
}

// GetUsername returns the configured username
func (m *IMAPMonitor) GetUsername() string {
	return m.username
}

// NewIMAPMonitor creates a new IMAP monitor
func NewIMAPMonitor(config *IMAPConfig) (*IMAPMonitor, error) {
	if config.PollInterval == 0 {
		config.PollInterval = 5 * time.Second
	}
	
	monitor := &IMAPMonitor{
		host:         config.Host,
		port:         config.Port,
		username:     config.Username,
		password:     config.Password,
		useTLS:       config.UseTLS,
		pollInterval: config.PollInterval,
	}
	
	return monitor, nil
}

// LoadIMAPConfigFromEnv loads IMAP configuration from environment variables
func LoadIMAPConfigFromEnv() *IMAPConfig {
	return &IMAPConfig{
		Host:         getEnvOrDefault("IMAP_HOST", "imap.fastmail.com"),
		Port:         getEnvOrDefault("IMAP_PORT", "993"), // IMAP SSL port
		Username:     os.Getenv("IMAP_USER"),
		Password:     os.Getenv("IMAP_PASS"),
		UseTLS:       os.Getenv("IMAP_SECURE") != "false", // Default to true
		PollInterval: 5 * time.Second,
	}
}

// LoadIMAPConfigFromFile loads IMAP configuration from config file
func LoadIMAPConfigFromFile(configData map[string]interface{}) *IMAPConfig {
	config := &IMAPConfig{
		PollInterval: 5 * time.Second,
		UseTLS:       true, // Default to secure
	}
	
	if emailConfig, ok := configData["email"].(map[string]interface{}); ok {
		// Try IMAP config first
		if host, ok := emailConfig["imap_host"].(string); ok {
			config.Host = host
		}
		if port, ok := emailConfig["imap_port"].(int); ok {
			config.Port = fmt.Sprintf("%d", port)
		} else if port, ok := emailConfig["imap_port"].(string); ok {
			config.Port = port
		}
		if user, ok := emailConfig["imap_user"].(string); ok {
			config.Username = user
		}
		if pass, ok := emailConfig["imap_pass"].(string); ok {
			config.Password = pass
		}
		if secure, ok := emailConfig["imap_secure"].(bool); ok {
			config.UseTLS = secure
		}
		
		// Fall back to old SMTP config names for compatibility
		if config.Host == "" {
			if host, ok := emailConfig["smtp_host"].(string); ok {
				config.Host = host
			}
		}
		if config.Username == "" {
			if user, ok := emailConfig["smtp_user"].(string); ok {
				config.Username = user
			}
		}
		if config.Password == "" {
			if pass, ok := emailConfig["smtp_pass"].(string); ok {
				config.Password = pass
			}
		}
	}
	
	// Fall back to environment variables if not in config
	if config.Host == "" {
		config.Host = getEnvOrDefault("IMAP_HOST", "imap.fastmail.com")
	}
	if config.Port == "" {
		config.Port = getEnvOrDefault("IMAP_PORT", "993")
	}
	if config.Username == "" {
		config.Username = os.Getenv("IMAP_USER")
	}
	if config.Password == "" {
		config.Password = os.Getenv("IMAP_PASS")
	}
	
	return config
}

// Connect establishes connection to the IMAP server
func (m *IMAPMonitor) Connect() error {
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
	
	// Don't update lastMessageID on initial connect - we want to check recent emails
	// Only set it if it's the very first connection
	if m.lastMessageID == 0 {
		// Check last 10 messages on initial connect to catch recently sent magic links
		status, err := m.client.Status("INBOX", []imap.StatusItem{imap.StatusMessages})
		if err == nil && status.Messages > 0 {
			// Start checking from 10 messages back (or from beginning if less than 10)
			if status.Messages > 10 {
				m.lastMessageID = status.Messages - 10
			} else {
				m.lastMessageID = 0
			}
			logging.Info("[EMAIL MONITOR] Will check last %d messages for magic links", status.Messages - m.lastMessageID)
		}
	} else {
		// On reconnect, keep the existing lastMessageID
		logging.Debug("[EMAIL MONITOR] Reconnected, continuing from message ID: %d", m.lastMessageID)
	}
	
	return nil
}

// Disconnect closes the connection to the IMAP server
func (m *IMAPMonitor) Disconnect() error {
	if m.client != nil {
		return m.client.Logout()
	}
	return nil
}

// StartMonitoring starts monitoring for new emails with magic links (blocking)
func (m *IMAPMonitor) StartMonitoring(onMagicLink func(url string) error) error {
	m.onMagicLink = onMagicLink
	
	if m.client == nil {
		if err := m.Connect(); err != nil {
			return err
		}
	}
	
	logging.Info("Starting email monitoring for magic links...")
	
	ticker := time.NewTicker(m.pollInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			if err := m.checkNewEmails(); err != nil {
				logging.Error("Error checking emails: %v", err)
				// Try to reconnect
				if err := m.Connect(); err != nil {
					logging.Error("Failed to reconnect: %v", err)
				}
			}
		}
	}
}

// StartMonitoringBackground starts monitoring for new emails in the background
func (m *IMAPMonitor) StartMonitoringBackground(onMagicLink func(url string) error) (chan struct{}, error) {
	m.onMagicLink = onMagicLink
	
	if m.client == nil {
		if err := m.Connect(); err != nil {
			return nil, err
		}
	}
	
	logging.Info("[EMAIL MONITOR] Starting background email monitoring for magic links...")
	
	stopChan := make(chan struct{})
	
	go func() {
		ticker := time.NewTicker(m.pollInterval)
		defer ticker.Stop()
		defer m.Disconnect()
		
		checkCount := 0
		for {
			select {
			case <-stopChan:
				logging.Info("[EMAIL MONITOR] Stopping email monitoring...")
				return
			case <-ticker.C:
				checkCount++
				// Log heartbeat every 10 checks
				if checkCount%10 == 0 {
					logging.Debug("[EMAIL MONITOR] Heartbeat: Still monitoring (check #%d)", checkCount)
				}
				
				if err := m.checkNewEmails(); err != nil {
					logging.Error("[EMAIL MONITOR] Error checking emails: %v", err)
					// Try to reconnect
					if err := m.Connect(); err != nil {
						logging.Error("[EMAIL MONITOR] Failed to reconnect: %v", err)
					}
				}
			}
		}
	}()
	
	return stopChan, nil
}

// checkNewEmails checks for new emails and extracts magic links
func (m *IMAPMonitor) checkNewEmails() error {
	logging.Debug("[EMAIL CHECK] Starting email check...")
	
	// Get current mailbox status
	status, err := m.client.Status("INBOX", []imap.StatusItem{imap.StatusMessages})
	if err != nil {
		return err
	}
	
	logging.Debug("[EMAIL CHECK] Inbox has %d total messages", status.Messages)
	
	if status.Messages == 0 {
		logging.Debug("[EMAIL CHECK] Inbox is empty, nothing to check")
		return nil
	}
	
	// Check for new messages OR recent messages if we just started
	from := m.lastMessageID + 1
	to := status.Messages
	
	// If this is our first check (lastMessageID could be 0 or set to messages-10)
	// make sure we check recent messages
	if from == 1 || m.lastMessageID == 0 {
		// Check last 10 messages on first scan
		if status.Messages > 10 {
			from = status.Messages - 9  // -9 because we want 10 messages total
		} else {
			from = 1
		}
		logging.Info("[EMAIL CHECK] First scan - checking messages from %d to %d", from, to)
	} else {
		logging.Debug("[EMAIL CHECK] Checking messages from %d to %d (last checked: %d)", 
			from, to, m.lastMessageID)
	}
	
	if from > to {
		logging.Debug("[EMAIL CHECK] No new messages since last check")
		return nil
	}
	
	newMessageCount := to - from + 1
	logging.Debug("[EMAIL CHECK] Found %d new message(s) to process", newMessageCount)
	
	seqset := new(imap.SeqSet)
	seqset.AddRange(from, to)
	
	// Fetch the messages - CRITICAL: Use BODY[] not just BODY
	messages := make(chan *imap.Message, 10)
	done := make(chan error, 1)
	
	go func() {
		// This is the critical fix - fetch BODY[] for actual content
		section := &imap.BodySectionName{}
		items := []imap.FetchItem{
			imap.FetchEnvelope,
			imap.FetchFlags,
			imap.FetchInternalDate,
			section.FetchItem(),  // This fetches BODY[] - the full content
		}
		done <- m.client.Fetch(seqset, items, messages)
	}()
	
	// Process messages
	processed := 0
	for msg := range messages {
		if msg == nil {
			continue
		}
		
		processed++
		
		// Process the message body
		if err := m.processMessage(msg); err != nil {
			logging.Error("[EMAIL PARSE] Error processing message: %v", err)
		}
		
		// Update last message ID
		if msg.SeqNum > m.lastMessageID {
			m.lastMessageID = msg.SeqNum
		}
	}
	
	if err := <-done; err != nil {
		return err
	}
	
	logging.Debug("[EMAIL CHECK] Processed %d messages", processed)
	return nil
}

// processMessage extracts and processes a single email message
func (m *IMAPMonitor) processMessage(msg *imap.Message) error {
	// Log message details
	if msg.Envelope != nil {
		logging.Debug("[EMAIL PARSE] Processing message from: %v, subject: %s", 
			msg.Envelope.From, msg.Envelope.Subject)
	}
	
	// Get the email body - with fallback approaches
	var body io.Reader
	
	// Try to get the full body first
	body = msg.GetBody(&imap.BodySectionName{})
	if body == nil {
		logging.Debug("[EMAIL PARSE] GetBody returned nil, trying alternative approach")
		// Try to get specific parts
		for name, literal := range msg.Body {
			logging.Debug("[EMAIL PARSE] Found body section: %v", name)
			if literal != nil {
				body = literal
				break
			}
		}
	}
	
	if body == nil {
		return fmt.Errorf("no body sections available")
	}
	
	// Parse the message
	mr, err := mail.CreateReader(body)
	if err != nil {
		return fmt.Errorf("failed to parse message: %w", err)
	}
	
	// Extract text from all parts
	var emailContent strings.Builder
	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			logging.Debug("[EMAIL PARSE] Failed to read part: %v", err)
			continue
		}
		
		switch h := p.Header.(type) {
		case *mail.InlineHeader:
			// Read the body
			b, _ := io.ReadAll(p.Body)
			emailContent.Write(b)
			logging.Debug("[EMAIL PARSE] Read inline part (%d bytes)", len(b))
		case *mail.AttachmentHeader:
			// Skip attachments
			logging.Debug("[EMAIL PARSE] Skipping attachment")
			_ = h
		}
	}
	
	// Extract magic link from email content
	content := emailContent.String()
	logging.Debug("[EMAIL PARSE] Email content length: %d bytes", len(content))
	
	// Log first 500 chars of content for debugging (sanitized)
	if len(content) > 0 {
		preview := content
		if len(preview) > 500 {
			preview = preview[:500] + "..."
		}
		logging.Debug("[EMAIL PARSE] Content preview: %s", preview)
	}
	
	link := extractMagicLinkFromContent(content)
	if link != "" {
		logging.Info("[EMAIL PARSE] Found magic link: %s", link)
		if m.onMagicLink != nil {
			if err := m.onMagicLink(link); err != nil {
				logging.Error("[EMAIL PARSE] Error in callback: %v", err)
			}
		}
	} else {
		logging.Debug("[EMAIL PARSE] No magic link found in this message")
	}
	
	return nil
}

// updateLastMessageID updates the last message ID to the current latest
func (m *IMAPMonitor) updateLastMessageID() {
	status, err := m.client.Status("INBOX", []imap.StatusItem{imap.StatusMessages})
	if err == nil && status.Messages > 0 {
		m.lastMessageID = status.Messages
		logging.Debug("[EMAIL MONITOR] Updated last message ID to: %d", m.lastMessageID)
	}
}

// CheckRecentEmails checks emails from the last N minutes for magic links
func (m *IMAPMonitor) CheckRecentEmails(minutes int) (string, error) {
	if m.client == nil {
		if err := m.Connect(); err != nil {
			return "", err
		}
	}
	
	logging.Info("[EMAIL CHECK] Checking emails from last %d minutes for magic links...", minutes)
	
	// Get current mailbox status
	status, err := m.client.Status("INBOX", []imap.StatusItem{imap.StatusMessages})
	if err != nil {
		return "", err
	}
	
	if status.Messages == 0 {
		logging.Debug("[EMAIL CHECK] Inbox is empty")
		return "", nil
	}
	
	// Check last 20 messages (should cover recent emails)
	from := uint32(1)
	to := status.Messages
	
	if status.Messages > 20 {
		from = status.Messages - 19
	}
	
	logging.Debug("[EMAIL CHECK] Scanning messages %d to %d for recent magic links", from, to)
	
	seqset := new(imap.SeqSet)
	seqset.AddRange(from, to)
	
	messages := make(chan *imap.Message, 10)
	done := make(chan error, 1)
	
	go func() {
		section := &imap.BodySectionName{}
		items := []imap.FetchItem{
			imap.FetchEnvelope,
			imap.FetchInternalDate,
			section.FetchItem(),
		}
		done <- m.client.Fetch(seqset, items, messages)
	}()
	
	// Check messages from most recent first
	cutoffTime := time.Now().Add(-time.Duration(minutes) * time.Minute)
	var foundLink string
	
	for msg := range messages {
		// Check if message is recent enough
		if msg.InternalDate.Before(cutoffTime) {
			continue
		}
		
		logging.Debug("[EMAIL CHECK] Processing message from %s (Subject: %s)", 
			msg.InternalDate.Format("15:04:05"), msg.Envelope.Subject)
		
		// Extract body
		section := &imap.BodySectionName{}
		body := msg.GetBody(section)
		if body != nil {
			content, err := ioutil.ReadAll(body)
			if err == nil {
				// Look for magic links
				link := extractMagicLinkFromContent(string(content))
				if link != "" {
					logging.Info("[EMAIL CHECK] Found magic link in email from %s", msg.InternalDate.Format("15:04:05"))
					foundLink = link
					break
				}
			}
		}
	}
	
	if err := <-done; err != nil {
		return "", err
	}
	
	return foundLink, nil
}

// extractMagicLinkFromContent extracts magic link URLs from email content
func extractMagicLinkFromContent(content string) string {
	logging.Debug("[REGEX EXTRACT] Starting regex extraction on %d bytes", len(content))
	
	// First try: Look for URLs with auth-related keywords
	authURLRegex := regexp.MustCompile(
		`https?://[^\s<>"']+(?:verify|auth|login|confirm|activate|magic|token|signin|sign-in)[^\s<>"']*`,
	)
	matches := authURLRegex.FindAllString(content, -1)
	logging.Debug("[REGEX EXTRACT] Auth regex found %d matches", len(matches))
	
	if len(matches) > 0 {
		// Log all matches for debugging
		for i, match := range matches {
			logging.Debug("[REGEX EXTRACT] Auth match %d: %s", i+1, match)
		}
		// Clean up the URL (remove any trailing punctuation)
		cleanURL := strings.TrimRight(matches[0], ".,;:!?")
		return cleanURL
	}
	
	// Second try: Look for any HTTP/HTTPS URL that's not unsubscribe/privacy/terms
	generalURLRegex := regexp.MustCompile(`https?://[^\s<>"']+`)
	matches = generalURLRegex.FindAllString(content, -1)
	logging.Debug("[REGEX EXTRACT] General regex found %d URLs", len(matches))
	
	for _, url := range matches {
		lowerURL := strings.ToLower(url)
		
		// Skip common non-auth links
		if !strings.Contains(lowerURL, "unsubscribe") &&
		   !strings.Contains(lowerURL, "privacy") &&
		   !strings.Contains(lowerURL, "terms") &&
		   !strings.Contains(lowerURL, "preferences") &&
		   !strings.Contains(lowerURL, "email-settings") &&
		   !strings.Contains(lowerURL, "support") &&
		   !strings.Contains(lowerURL, "help") {
			// Clean up the URL
			cleanURL := strings.TrimRight(url, ".,;:!?")
			logging.Debug("[REGEX EXTRACT] Selected URL: %s", cleanURL)
			return cleanURL
		}
	}
	
	logging.Debug("[REGEX EXTRACT] No suitable URL found")
	return ""
}

// getEnvOrDefault gets environment variable or returns default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}