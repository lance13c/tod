package cmd

import (
	"fmt"
	"os"

	"github.com/ciciliostudio/tod/internal/email"
	"github.com/spf13/cobra"
)

// authCmd represents the auth command
var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication settings",
	Long: `Manage authentication settings for Tod testing, including email setup
for automated verification code and magic link extraction.

Tod can automatically check your Gmail for authentication emails and extract
verification codes, magic links, and 2FA codes during testing.

Examples:
  tod auth setup-email     # Configure Gmail access (one-time setup)
  tod auth status          # Check current authentication status
  tod auth reset-email     # Reset email configuration`,
}

// setupEmailCmd configures Gmail access for email checking
var setupEmailCmd = &cobra.Command{
	Use:   "setup-email",
	Short: "Setup Gmail access for email checking",
	Long: `Configure Gmail access so Tod can automatically check for authentication emails.

This command will:
  1. Open your browser for Google OAuth authentication
  2. Request read-only access to your Gmail
  3. Store encrypted credentials securely
  4. Test the connection to verify it works

The setup is one-time only. Your credentials are stored locally and never shared.`,
	Run: runSetupEmail,
}

// statusCmd shows current authentication status
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show authentication status",
	Long:  `Display the current status of Tod authentication settings, including email configuration.`,
	Run:   runAuthStatus,
}

// resetEmailCmd resets email configuration
var resetEmailCmd = &cobra.Command{
	Use:   "reset-email",
	Short: "Reset email configuration",
	Long:  `Remove existing email configuration and credentials. You'll need to run setup-email again to reconfigure.`,
	Run:   runResetEmail,
}

func init() {
	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(setupEmailCmd)
	authCmd.AddCommand(statusCmd)
	authCmd.AddCommand(resetEmailCmd)

	// setup-email command flags
	setupEmailCmd.Flags().Bool("reset", false, "Reset existing configuration before setup")
	setupEmailCmd.Flags().Bool("force", false, "Force setup even if already configured")
}

func runSetupEmail(cmd *cobra.Command, args []string) {
	projectDir, _ := cmd.Flags().GetString("project")
	if projectDir == "" {
		projectDir = "."
	}

	reset, _ := cmd.Flags().GetBool("reset")
	force, _ := cmd.Flags().GetBool("force")

	setupService := email.NewSetupService(projectDir)

	// Check if already configured
	if !reset && !force {
		isConfigured, email, err := setupService.CheckSetup()
		if isConfigured {
			fmt.Printf("✅ Email checker is already configured for: %s\n", email)
			fmt.Println("💡 Use --reset to reconfigure or --force to setup anyway")
			return
		}
		if err != nil {
			fmt.Printf("⚠️  Found existing configuration but it has issues: %v\n", err)
			fmt.Println("🔧 Proceeding with setup to fix the configuration...")
		}
	}

	// Reset if requested
	if reset {
		fmt.Println("🔄 Resetting existing email configuration...")
		if err := setupService.ResetSetup(); err != nil {
			fmt.Printf("❌ Failed to reset configuration: %v\n", err)
			os.Exit(1)
		}
		fmt.Println()
	}

	// Run the setup process
	if err := setupService.RunSetup(); err != nil {
		fmt.Printf("❌ Email setup failed: %v\n", err)
		fmt.Println()
		fmt.Println("💡 Troubleshooting tips:")
		fmt.Println("   • Make sure you have internet connection")
		fmt.Println("   • Check that you copied the full authorization code")
		fmt.Println("   • Try running 'tod auth setup-email --reset' to start over")
		os.Exit(1)
	}
}

func runAuthStatus(cmd *cobra.Command, args []string) {
	projectDir, _ := cmd.Flags().GetString("project")
	if projectDir == "" {
		projectDir = "."
	}

	fmt.Println("🔐 Tod Authentication Status")
	fmt.Println()

	// Check email setup
	setupService := email.NewSetupService(projectDir)
	if err := setupService.ShowStatus(); err != nil {
		// Error details are already printed by ShowStatus
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("💡 Tips:")
	fmt.Println("   • Add 'email: your@gmail.com' to test users for automatic auth")
	fmt.Println("   • Tod will automatically check for verification codes and magic links")
	fmt.Println("   • Run 'tod auth setup-email' if you need to reconfigure")
}

func runResetEmail(cmd *cobra.Command, args []string) {
	projectDir, _ := cmd.Flags().GetString("project")
	if projectDir == "" {
		projectDir = "."
	}

	setupService := email.NewSetupService(projectDir)
	
	if err := setupService.ResetSetup(); err != nil {
		fmt.Printf("❌ Failed to reset email configuration: %v\n", err)
		os.Exit(1)
	}
}