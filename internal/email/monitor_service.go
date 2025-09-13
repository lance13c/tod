package email

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/lance13c/tod/internal/browser"
	"github.com/lance13c/tod/internal/logging"
	"gopkg.in/yaml.v3"
)

// MonitorService manages background email monitoring
type MonitorService struct {
	monitor      *IMAPMonitor
	stopChan     chan struct{}
	wsURL        string
	mu           sync.Mutex
	isRunning    bool
	projectDir   string
	onMagicLink  func(string) error
	linkDetected chan string
}

var (
	// Global monitor service instance
	globalMonitor     *MonitorService
	globalMonitorOnce sync.Once
)

// GetMonitorService returns the global monitor service instance
func GetMonitorService(projectDir string) *MonitorService {
	globalMonitorOnce.Do(func() {
		globalMonitor = &MonitorService{
			projectDir:   projectDir,
			linkDetected: make(chan string, 10),
		}
	})
	return globalMonitor
}

// SetOnMagicLink sets a global callback for when magic links are detected
func (m *MonitorService) SetOnMagicLink(callback func(string) error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onMagicLink = callback
}

// GetLinkDetectedChannel returns the channel for detected links
func (m *MonitorService) GetLinkDetectedChannel() <-chan string {
	return m.linkDetected
}

// StartBackgroundMonitoring starts email monitoring in the background
func (m *MonitorService) StartBackgroundMonitoring() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.isRunning {
		return fmt.Errorf("email monitoring is already running")
	}
	
	// Load configuration from file
	configPath := filepath.Join(m.projectDir, ".tod", "config.yaml")
	configData, err := loadConfigFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	
	// Load IMAP config from file
	imapConfig := LoadIMAPConfigFromFile(configData)
	
	// Validate configuration
	if imapConfig.Username == "" || imapConfig.Password == "" {
		return fmt.Errorf("Email (IMAP) credentials not configured in .tod/config.yaml")
	}
	
	// Try to connect to Chrome
	wsURL, err := browser.GetChromeWebSocketURL("localhost", "9222")
	if err != nil {
		logging.Warn("Chrome DevTools not available. Magic links will be logged but not navigated: %v", err)
		m.wsURL = ""
	} else {
		m.wsURL = wsURL
		logging.Info("Connected to Chrome DevTools")
	}
	
	// Create IMAP monitor
	monitor, err := NewIMAPMonitor(imapConfig)
	if err != nil {
		return fmt.Errorf("failed to create email monitor: %w", err)
	}
	
	// Start monitoring in background
	stopChan, err := monitor.StartMonitoringBackground(func(magicLink string) error {
		logging.Info("[MONITOR SERVICE] Magic link detected: %s", magicLink)
		
		// Send to channel if anyone is listening
		select {
		case m.linkDetected <- magicLink:
		default:
			// Channel full or no listeners
		}
		
		// Call global callback if set
		if m.onMagicLink != nil {
			if err := m.onMagicLink(magicLink); err != nil {
				logging.Error("[MONITOR SERVICE] Callback error: %v", err)
			}
		}
		
		// Navigate Chrome if available
		if m.wsURL != "" {
			logging.Info("[MONITOR SERVICE] Auto-navigating Chrome to magic link...")
			if err := browser.NavigateToURLDirect(m.wsURL, magicLink); err != nil {
				logging.Error("[MONITOR SERVICE] Failed to navigate Chrome: %v", err)
				// Try to reconnect to Chrome
				if newWSURL, err := browser.GetChromeWebSocketURL("localhost", "9222"); err == nil {
					m.wsURL = newWSURL
					// Retry navigation
					if err := browser.NavigateToURLDirect(m.wsURL, magicLink); err == nil {
						logging.Info("[MONITOR SERVICE] Chrome navigated successfully after reconnection")
					}
				}
			} else {
				logging.Info("[MONITOR SERVICE] Chrome navigated to magic link successfully")
			}
		}
		
		return nil
	})
	
	if err != nil {
		return fmt.Errorf("failed to start monitoring: %w", err)
	}
	
	m.monitor = monitor
	m.stopChan = stopChan
	m.isRunning = true
	
	logging.Info("[MONITOR SERVICE] Email monitoring started in background (user: %s)", imapConfig.Username)
	return nil
}

// StopMonitoring stops the background email monitoring
func (m *MonitorService) StopMonitoring() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if !m.isRunning {
		return
	}
	
	if m.stopChan != nil {
		close(m.stopChan)
	}
	
	m.isRunning = false
	logging.Info("Email monitoring stopped")
}

// IsRunning returns whether monitoring is currently active
func (m *MonitorService) IsRunning() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.isRunning
}

// loadConfigFile loads configuration from YAML file
func loadConfigFile(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	
	var config map[string]interface{}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	
	return config, nil
}

// AutoStartMonitoring automatically starts monitoring if configured
func AutoStartMonitoring(projectDir string) {
	// Check if config exists
	configPath := filepath.Join(projectDir, ".tod", "config.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return
	}
	
	// Load config
	configData, err := loadConfigFile(configPath)
	if err != nil {
		return
	}
	
	// Check if email is configured
	if emailConfig, ok := configData["email"].(map[string]interface{}); ok {
		// Check for IMAP config first, then fall back to SMTP names
		hasConfig := false
		if user, ok := emailConfig["imap_user"].(string); ok && user != "" {
			hasConfig = true
		} else if user, ok := emailConfig["smtp_user"].(string); ok && user != "" {
			hasConfig = true
		}
		
		if hasConfig {
			// Start monitoring
			monitor := GetMonitorService(projectDir)
			if err := monitor.StartBackgroundMonitoring(); err != nil {
				logging.Warn("Failed to auto-start email monitoring: %v", err)
			}
		}
	}
}