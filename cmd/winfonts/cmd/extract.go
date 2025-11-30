package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var extractCmd = &cobra.Command{
	Use:   "extract <iso-file> <output-directory>",
	Short: "Extract fonts from a Windows ISO file",
	Long: `Extract fonts from a Windows ISO file to a specified output directory.
The command will mount the ISO, locate the fonts directory, and extract all font files.`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		isoFile := args[0]
		outputDir := args[1]

		if _, err := os.Stat(isoFile); os.IsNotExist(err) {
			return fmt.Errorf("ISO file does not exist: %s", isoFile)
		}

		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}

		fmt.Printf("Extracting fonts from %s to %s\n", isoFile, outputDir)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(extractCmd)
}
