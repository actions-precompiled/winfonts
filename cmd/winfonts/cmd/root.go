package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "winfonts",
	Short: "A tool for managing Windows fonts",
	Long: `winfonts is a CLI tool for managing Windows fonts.
It provides commands to install, list, and manage fonts from Windows.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Use 'winfonts --help' for more information")
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().BoolP("version", "v", false, "Print version information")
}
