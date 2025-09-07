package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ciciliostudio/tod/internal/browser"
	"github.com/ciciliostudio/tod/internal/config"
	"github.com/ciciliostudio/tod/internal/email"
	"github.com/ciciliostudio/tod/internal/logging"
	"github.com/spf13/cobra"
)

var cfgFile string
var todConfig *config.Config

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "tod",
	Short: "Tod - Text-adventure Testing",
	Long: `Tod is a delightful CLI testing tool that presents E2E testing as 
interactive text-adventure journeys, with AI-powered assistance and support
for any testing framework.

When run without arguments, Tod launches the interactive adventure interface.
Use subcommands for specific operations like 'init', 'actions', or 'generate'.`,
	Run: runTUI,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is .tod/config.yaml)")
	rootCmd.PersistentFlags().BoolP("verbose", "V", false, "verbose output")
	rootCmd.PersistentFlags().StringP("env", "e", "", "environment to use")
	rootCmd.PersistentFlags().StringP("project", "p", ".", "project directory")
	rootCmd.Flags().BoolP("version", "v", false, "show version information")
}

// initConfig reads in config file and ENV variables.
func initConfig() {
	startTime := time.Now()
	verbose, _ := rootCmd.PersistentFlags().GetBool("verbose")
	
	projectDir, _ := rootCmd.PersistentFlags().GetString("project")
	
	// Initialize logging first
	if err := logging.Initialize(projectDir); err != nil {
		// Fall back to stderr if logging fails to initialize
		fmt.Fprintf(os.Stderr, "Warning: Failed to initialize logging: %v\n", err)
	} else {
		// Redirect standard log to our file logger
		logging.RedirectStandardLog()
	}
	
	// Set log level based on verbose flag
	if verbose {
		logging.GetLogger().SetLevel(logging.DEBUG)
	}
	
	loader := config.NewLoader(projectDir)
	
	if verbose {
		logging.Debug("Config loader created in %v", time.Since(startTime))
	}
	
	// Load configuration if available
	if loader.IsInitialized() {
		loadStart := time.Now()
		var err error
		todConfig, err = loader.Load()
		if err != nil {
			logging.Warn("Failed to load config: %v", err)
		} else {
			if verbose {
				logging.Debug("Config loaded in %v", time.Since(loadStart))
			}
			
			// Apply environment override from flag
			if env, _ := rootCmd.PersistentFlags().GetString("env"); env != "" {
				if _, exists := todConfig.Envs[env]; exists {
					todConfig.Current = env
				}
			}
			
			// Auto-start email monitoring if configured
			email.AutoStartMonitoring(projectDir)
			
			logging.Info("Using config with environment: %s", todConfig.Current)
		}
	}

	if verbose {
		logging.Debug("Total config init time: %v", time.Since(startTime))
	}
}

// runTUI launches the main TUI interface
func runTUI(cmd *cobra.Command, args []string) {
	startTime := time.Now()
	verbose, _ := cmd.Flags().GetBool("verbose")
	
	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	
	// Handle cleanup in a goroutine
	go func() {
		<-sigChan
		// Clean up Chrome when receiving interrupt signal
		browser.CloseGlobalChromeDPManager()
		os.Exit(0)
	}()
	
	// Check for version flag
	if showVersion, _ := cmd.Flags().GetBool("version"); showVersion {
		fmt.Printf("Tod version %s\n", appVersion)
		return
	}

	// Check if project is initialized
	if todConfig == nil {
		fmt.Println("🚨 Tod is not initialized in this project!")
		fmt.Println("Run 'tod init' to get started.")
		os.Exit(1)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "[DEBUG] Pre-TUI checks completed in %v\n", time.Since(startTime))
	}

	// Launch TUI with lazy loading (directly into Tod Adventure Mode)
	tuiStart := time.Now()
	err := launchTUI(todConfig)
	
	// Check if user requested reconfiguration
	if err == ErrRestartConfig {
		// User requested reconfiguration from within Tod Adventure Mode
		fmt.Println("\nRestarting configuration...")
		// Run init command directly
		initCmd.Run(initCmd, args)
		return
	} else if err != nil {
		fmt.Printf("Alas! There was an error: %v\n", err)
		// Clean up before exit
		browser.CloseGlobalChromeDPManager()
		os.Exit(1)
	}
	
	// Clean up Chrome on normal exit
	browser.CloseGlobalChromeDPManager()
	
	if verbose {
		fmt.Fprintf(os.Stderr, "[DEBUG] TUI execution time: %v\n", time.Since(tuiStart))
		fmt.Fprintf(os.Stderr, "[DEBUG] Total runtime: %v\n", time.Since(startTime))
	}
}

