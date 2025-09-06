package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ciciliostudio/tod/internal/config"
	"github.com/ciciliostudio/tod/internal/ui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().StringP("env", "e", "", "environment to use")
	rootCmd.PersistentFlags().StringP("project", "p", ".", "project directory")
}

// initConfig reads in config file and ENV variables.
func initConfig() {
	projectDir, _ := rootCmd.PersistentFlags().GetString("project")
	loader := config.NewLoader(projectDir)
	
	// Load configuration if available
	if loader.IsInitialized() {
		var err error
		todConfig, err = loader.Load()
		if err != nil {
			if verbose, _ := rootCmd.PersistentFlags().GetBool("verbose"); verbose {
				fmt.Fprintf(os.Stderr, "Warning: Failed to load config: %v\n", err)
			}
		} else {
			// Apply environment override from flag
			if env, _ := rootCmd.PersistentFlags().GetString("env"); env != "" {
				if _, exists := todConfig.Envs[env]; exists {
					todConfig.Current = env
				}
			}
			
			if verbose, _ := rootCmd.PersistentFlags().GetBool("verbose"); verbose {
				fmt.Fprintf(os.Stderr, "Using config with environment: %s\n", todConfig.Current)
			}
		}
	}

	// Also keep viper for backward compatibility
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath(".tod")
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		if verbose, _ := rootCmd.PersistentFlags().GetBool("verbose"); verbose {
			fmt.Fprintln(os.Stderr, "Using viper config file:", viper.ConfigFileUsed())
		}
	}
}

// runTUI launches the main TUI interface
func runTUI(cmd *cobra.Command, args []string) {
	// Check if project is initialized
	if todConfig == nil {
		fmt.Println("🚨 Tod is not initialized in this project!")
		fmt.Println("Run 'tod init' to get started.")
		os.Exit(1)
	}

	// Initialize the main model with configuration
	model := ui.NewModel(todConfig)

	// Create the program with some options
	program := tea.NewProgram(
		model,
		tea.WithAltScreen(),       // Use alternate screen buffer
		tea.WithMouseCellMotion(), // Enable mouse support
	)

	// Run the program
	if _, err := program.Run(); err != nil {
		fmt.Printf("Alas! There was an error: %v\n", err)
		os.Exit(1)
	}
}