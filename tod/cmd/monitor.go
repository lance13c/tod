package cmd

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ciciliostudio/tod/internal/browser"
	"github.com/ciciliostudio/tod/internal/email"
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
  2. SMTP/IMAP credentials must be configured via environment variables:
     - SMTP_HOST: IMAP server hostname
     - SMTP_PORT: IMAP port (usually 993 for SSL)
     - SMTP_USER: Email username
     - SMTP_PASS: Email password
     - SMTP_SECURE: Set to "true" for SSL/TLS

Example:
  export SMTP_HOST=imap.gmail.com
  export SMTP_PORT=993
  export SMTP_USER=your-email@gmail.com
  export SMTP_PASS=your-app-password
  export SMTP_SECURE=true
  
  # Start Chrome with debugging
  /Applications/Google\ Chrome.app/Contents/MacOS/Google\ Chrome --remote-debugging-port=9222
  
  # Start monitoring
  tod monitor email`,
	Run: runEmailMonitor,
}

func init() {
	rootCmd.AddCommand(monitorCmd)
	monitorCmd.AddCommand(emailMonitorCmd)
	
	// Email monitor flags
	emailMonitorCmd.Flags().String("chrome-port", "9222", "Chrome DevTools debugging port")
	emailMonitorCmd.Flags().String("chrome-host", "localhost", "Chrome DevTools host")
	emailMonitorCmd.Flags().Int("poll-interval", 3, "Email polling interval in seconds")
}

func runEmailMonitor(cmd *cobra.Command, args []string) {
	// Get Chrome connection info
	chromeHost, _ := cmd.Flags().GetString("chrome-host")
	chromePort, _ := cmd.Flags().GetString("chrome-port")
	projectDir, _ := cmd.Flags().GetString("project")
	if projectDir == "" {
		projectDir = "."
	}
	
	// Try to load SMTP configuration from config file first
	var smtpConfig *email.SMTPConfig
	configPath := fmt.Sprintf("%s/.tod/config.yaml", projectDir)
	if _, err := os.Stat(configPath); err == nil {
		// Config file exists, try to load from it
		configData := make(map[string]interface{})
		if data, err := os.ReadFile(configPath); err == nil {
			yaml.Unmarshal(data, &configData)
			smtpConfig = email.LoadSMTPConfigFromFile(configData)
		}
	}
	
	// Fall back to environment variables if not loaded from config
	if smtpConfig == nil || smtpConfig.Username == "" {
		smtpConfig = email.LoadSMTPConfigFromEnv()
	}
	
	// Validate configuration
	if smtpConfig.Username == "" || smtpConfig.Password == "" {
		fmt.Println("❌ SMTP credentials not configured")
		fmt.Println("Please set the following environment variables:")
		fmt.Println("  - SMTP_USER: Your email username")
		fmt.Println("  - SMTP_PASS: Your email password")
		fmt.Println("  - SMTP_HOST: IMAP server (optional, defaults to smtps-proxy.fastmail.com)")
		fmt.Println("  - SMTP_PORT: IMAP port (optional, defaults to 993)")
		os.Exit(1)
	}
	
	// Get Chrome WebSocket URL
	wsURL, err := browser.GetChromeWebSocketURL(chromeHost, chromePort)
	if err != nil {
		fmt.Printf("❌ Failed to connect to Chrome DevTools: %v\n", err)
		fmt.Println("\nMake sure Chrome is running with debugging enabled:")
		fmt.Println("  /Applications/Google\\ Chrome.app/Contents/MacOS/Google\\ Chrome --remote-debugging-port=9222")
		os.Exit(1)
	}
	
	fmt.Printf("✅ Connected to Chrome DevTools at %s:%s\n", chromeHost, chromePort)
	fmt.Printf("📧 Monitoring emails from %s\n", smtpConfig.Username)
	fmt.Println("🔍 Watching for magic links...")
	fmt.Println("\nPress Ctrl+C to stop monitoring")
	
	// Create SMTP monitor
	monitor, err := email.NewSMTPMonitor(smtpConfig)
	if err != nil {
		fmt.Printf("❌ Failed to create email monitor: %v\n", err)
		os.Exit(1)
	}
	
	// Connect to email server
	if err := monitor.Connect(); err != nil {
		fmt.Printf("❌ Failed to connect to email server: %v\n", err)
		fmt.Println("\nCheck your SMTP credentials and server settings")
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
			fmt.Printf("\n🎯 Magic link detected: %s\n", magicLink)
			fmt.Println("🌐 Navigating Chrome to the magic link...")
			
			// Navigate Chrome to the magic link
			if err := browser.NavigateToURLDirect(wsURL, magicLink); err != nil {
				log.Printf("❌ Failed to navigate Chrome: %v", err)
				return err
			}
			
			fmt.Println("✅ Chrome navigated successfully!")
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