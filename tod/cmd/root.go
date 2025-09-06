package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/ciciliostudio/tod/internal/config"
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
	loader := config.NewLoader(projectDir)
	
	if verbose {
		fmt.Fprintf(os.Stderr, "[DEBUG] Config loader created in %v\n", time.Since(startTime))
	}
	
	// Load configuration if available
	if loader.IsInitialized() {
		loadStart := time.Now()
		var err error
		todConfig, err = loader.Load()
		if err != nil {
			if verbose {
				fmt.Fprintf(os.Stderr, "Warning: Failed to load config: %v\n", err)
			}
		} else {
			if verbose {
				fmt.Fprintf(os.Stderr, "[DEBUG] Config loaded in %v\n", time.Since(loadStart))
			}
			
			// Apply environment override from flag
			if env, _ := rootCmd.PersistentFlags().GetString("env"); env != "" {
				if _, exists := todConfig.Envs[env]; exists {
					todConfig.Current = env
				}
			}
			
			if verbose {
				fmt.Fprintf(os.Stderr, "Using config with environment: %s\n", todConfig.Current)
			}
		}
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "[DEBUG] Total config init time: %v\n", time.Since(startTime))
	}
}

// runTUI launches the main TUI interface
func runTUI(cmd *cobra.Command, args []string) {
	startTime := time.Now()
	verbose, _ := cmd.Flags().GetBool("verbose")
	
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

	// Launch TUI with lazy loading
	tuiStart := time.Now()
	if err := launchTUI(todConfig); err != nil {
		fmt.Printf("Alas! There was an error: %v\n", err)
		os.Exit(1)
	}
	
	if verbose {
		fmt.Fprintf(os.Stderr, "[DEBUG] TUI execution time: %v\n", time.Since(tuiStart))
		fmt.Fprintf(os.Stderr, "[DEBUG] Total runtime: %v\n", time.Since(startTime))
	}
}

