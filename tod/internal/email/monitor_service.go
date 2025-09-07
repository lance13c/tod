package email

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/ciciliostudio/tod/internal/browser"
	"gopkg.in/yaml.v3"
)

// MonitorService manages background email monitoring
type MonitorService struct {
	monitor    *SMTPMonitor
	stopChan   chan struct{}
	wsURL      string
	mu         sync.Mutex
	isRunning  bool
	projectDir string
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
			projectDir: projectDir,
		}
	})
	return globalMonitor
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
	
	// Load SMTP config from file
	smtpConfig := LoadSMTPConfigFromFile(configData)
	
	// Validate configuration
	if smtpConfig.Username == "" || smtpConfig.Password == "" {
		return fmt.Errorf("SMTP credentials not configured in .tod/config.yaml")
	}
	
	// Try to connect to Chrome
	wsURL, err := browser.GetChromeWebSocketURL("localhost", "9222")
	if err != nil {
		log.Printf("Warning: Chrome DevTools not available. Magic links will be logged but not navigated: %v", err)
		m.wsURL = ""
	} else {
		m.wsURL = wsURL
		log.Printf("Connected to Chrome DevTools")
	}
	
	// Create SMTP monitor
	monitor, err := NewSMTPMonitor(smtpConfig)
	if err != nil {
		return fmt.Errorf("failed to create email monitor: %w", err)
	}
	
	// Start monitoring in background
	stopChan, err := monitor.StartMonitoringBackground(func(magicLink string) error {
		log.Printf("Magic link detected: %s", magicLink)
		
		// Navigate Chrome if available
		if m.wsURL != "" {
			if err := browser.NavigateToURLDirect(m.wsURL, magicLink); err != nil {
				log.Printf("Failed to navigate Chrome: %v", err)
				// Try to reconnect to Chrome
				if newWSURL, err := browser.GetChromeWebSocketURL("localhost", "9222"); err == nil {
					m.wsURL = newWSURL
					// Retry navigation
					browser.NavigateToURLDirect(m.wsURL, magicLink)
				}
			} else {
				log.Printf("Chrome navigated to magic link successfully")
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
	
	log.Printf("Email monitoring started in background (user: %s)", smtpConfig.Username)
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
	log.Println("Email monitoring stopped")
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
		if user, ok := emailConfig["smtp_user"].(string); ok && user != "" {
			// Start monitoring
			monitor := GetMonitorService(projectDir)
			if err := monitor.StartBackgroundMonitoring(); err != nil {
				log.Printf("Failed to auto-start email monitoring: %v", err)
			}
		}
	}
}