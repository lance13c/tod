package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var appVersion = "dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version information",
	Long:  "Print the version information for Tod",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Tod version %s\n", appVersion)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

func SetVersion(version string) {
	appVersion = version
}