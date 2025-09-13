package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/lance13c/tod/internal/browser"
	"github.com/lance13c/tod/internal/email"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// monitorCmd represents the monitor command
var monitorCmd = &cobra.Command{
	Use:   "monitor",
	Short: "Monitor for events and take automated actions",
	Long: `Monitor various sources for events and automatically respond to them.
	
Currently supports:
  - Email monitoring for magic links (auto-navigates Chrome)`,
}

// emailMonitorCmd monitors emails for magic links
var emailMonitorCmd = &cobra.Command{
	Use:   "email",
	Short: "Monitor emails for magic links and auto-navigate Chrome",
	Long: `Monitors your email inbox for incoming messages containing magic links
and automatically navigates the connected Chrome DevTools browser to those links.

This is useful for testing authentication flows that use magic links.

Prerequisites:
  1. Chrome must be running with debugging enabled (port 9222)
  2. IMAP credentials must be configured in .tod/config.yaml or environment variables:
     
     Config file (.tod/config.yaml):
       email:
         imap_host: imap.fastmail.com
         imap_port: 993
         imap_user: user@example.com
         imap_pass: your-app-password
         imap_secure: true
     
     OR Environment variables:
       - IMAP_HOST: IMAP server hostname
       - IMAP_PORT: IMAP port (usually 993 for SSL)
       - IMAP_USER: Email username
       - IMAP_PASS: Email password
       - IMAP_SECURE: Set to "false" to disable SSL (default: true)

Example:
  # Using config file
  tod monitor email
  
  # Using environment variables
  export IMAP_HOST=imap.gmail.com
  export IMAP_PORT=993
  export IMAP_USER=your-email@gmail.com
  export IMAP_PASS=your-app-password
  tod monitor email`,
	Run: runEmailMonitor,
}

func init() {
	rootCmd.AddCommand(monitorCmd)
	monitorCmd.AddCommand(emailMonitorCmd)
	
	// Email monitor flags
	emailMonitorCmd.Flags().String("chrome-port", "9222", "Chrome DevTools debugging port")
	emailMonitorCmd.Flags().String("chrome-host", "localhost", "Chrome DevTools host")
	emailMonitorCmd.Flags().Int("poll-interval", 5, "Email polling interval in seconds")
	emailMonitorCmd.Flags().Bool("auto-nav", true, "Automatically navigate Chrome to detected magic links")
}

func runEmailMonitor(cmd *cobra.Command, args []string) {
	// Get Chrome connection info
	chromeHost, _ := cmd.Flags().GetString("chrome-host")
	chromePort, _ := cmd.Flags().GetString("chrome-port")
	projectDir, _ := cmd.Flags().GetString("project")
	if projectDir == "" {
		projectDir = "."
	}
	
	// Try to load IMAP configuration from config file first
	var imapConfig *email.IMAPConfig
	configPath := fmt.Sprintf("%s/.tod/config.yaml", projectDir)
	if _, err := os.Stat(configPath); err == nil {
		// Config file exists, try to load from it
		configData := make(map[string]interface{})
		if data, err := os.ReadFile(configPath); err == nil {
			yaml.Unmarshal(data, &configData)
			imapConfig = email.LoadIMAPConfigFromFile(configData)
		}
	}
	
	// Fall back to environment variables if not loaded from config
	if imapConfig == nil || imapConfig.Username == "" {
		imapConfig = email.LoadIMAPConfigFromEnv()
	}
	
	// Validate configuration
	if imapConfig.Username == "" || imapConfig.Password == "" {
		fmt.Println("❌ IMAP credentials not configured")
		fmt.Println("\nPlease configure in .tod/config.yaml:")
		fmt.Println("  email:")
		fmt.Println("    imap_host: imap.fastmail.com")
		fmt.Println("    imap_port: 993")
		fmt.Println("    imap_user: your-email@example.com")
		fmt.Println("    imap_pass: your-app-password")
		fmt.Println("    imap_secure: true")
		fmt.Println("\nOr set environment variables:")
		fmt.Println("  - IMAP_USER: Your email username")
		fmt.Println("  - IMAP_PASS: Your email password")
		fmt.Println("  - IMAP_HOST: IMAP server (optional, defaults to imap.fastmail.com)")
		fmt.Println("  - IMAP_PORT: IMAP port (optional, defaults to 993)")
		os.Exit(1)
	}
	
	// Check if auto-navigation is enabled
	autoNav, _ := cmd.Flags().GetBool("auto-nav")
	var wsURL string
	
	if autoNav {
		// Try to get Chrome WebSocket URL
		var err error
		wsURL, err = browser.GetChromeWebSocketURL(chromeHost, chromePort)
		if err != nil {
			fmt.Printf("⚠️ Chrome DevTools not available. Magic links will be logged but not navigated: %v\n", err)
			fmt.Println("\nTo enable auto-navigation, start Chrome with debugging:")
			fmt.Println("  /Applications/Google\\ Chrome.app/Contents/MacOS/Google\\ Chrome --remote-debugging-port=9222")
			autoNav = false
		} else {
			fmt.Printf("✅ Connected to Chrome DevTools at %s:%s\n", chromeHost, chromePort)
		}
	}
	
	fmt.Printf("📧 Monitoring emails from %s\n", imapConfig.Username)
	fmt.Println("🔍 Watching for magic links...")
	if autoNav {
		fmt.Println("🚀 Auto-navigation enabled")
	} else {
		fmt.Println("📝 Auto-navigation disabled (links will be logged only)")
	}
	fmt.Println("\nPress Ctrl+C to stop monitoring")
	
	// Create IMAP monitor
	monitor, err := email.NewIMAPMonitor(imapConfig)
	if err != nil {
		fmt.Printf("❌ Failed to create email monitor: %v\n", err)
		os.Exit(1)
	}
	
	// Connect to email server
	if err := monitor.Connect(); err != nil {
		fmt.Printf("❌ Failed to connect to email server: %v\n", err)
		fmt.Println("\nCheck your IMAP credentials and server settings")
		fmt.Println("Common IMAP servers:")
		fmt.Println("  - Gmail: imap.gmail.com:993")
		fmt.Println("  - Fastmail: imap.fastmail.com:993")
		fmt.Println("  - Outlook: outlook.office365.com:993")
		os.Exit(1)
	}
	defer monitor.Disconnect()
	
	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	
	// Start monitoring in a goroutine
	errChan := make(chan error, 1)
	go func() {
		err := monitor.StartMonitoring(func(magicLink string) error {
			fmt.Printf("\n🎉 Magic link detected: %s\n", magicLink)
			
			if autoNav && wsURL != "" {
				fmt.Println("🌐 Navigating Chrome to the magic link...")
				
				// Navigate Chrome to the magic link
				if err := browser.NavigateToURLDirect(wsURL, magicLink); err != nil {
					fmt.Printf("❌ Failed to navigate Chrome: %v\n", err)
					// Try to reconnect to Chrome
					if newWSURL, err := browser.GetChromeWebSocketURL(chromeHost, chromePort); err == nil {
						wsURL = newWSURL
						// Retry navigation
						if err := browser.NavigateToURLDirect(wsURL, magicLink); err == nil {
							fmt.Println("✅ Chrome navigated successfully after reconnection!")
						}
					}
				} else {
					fmt.Println("✅ Chrome navigated successfully!")
				}
			} else {
				fmt.Println("📝 Link detected (auto-navigation disabled)")
			}
			
			return nil
		})
		errChan <- err
	}()
	
	// Wait for interrupt or error
	select {
	case <-sigChan:
		fmt.Println("\n\n👋 Stopping email monitor...")
	case err := <-errChan:
		if err != nil {
			fmt.Printf("\n❌ Monitor error: %v\n", err)
			os.Exit(1)
		}
	}
}