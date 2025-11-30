package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/actions-precompiled/winfonts"
	"github.com/spf13/cobra"
)

var (
	fetchVersion  string
	fetchEdition  string
	fetchArch     string
	fetchLanguage string
	fetchProductID string
	fetchOutputDir string
	keepISO       bool
)

var fetchCmd = &cobra.Command{
	Use:   "fetch",
	Short: "Download Windows ISO and extract fonts",
	Long: `Download a Windows ISO from Microsoft's servers and automatically extract fonts.
This command combines the download and extract operations into a single step.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if fetchOutputDir == "" {
			return fmt.Errorf("output directory is required (use -o or --output)")
		}

		version := winfonts.WindowsVersion(fetchVersion)
		edition := winfonts.WindowsEdition(fetchEdition)
		arch := winfonts.Architecture(fetchArch)
		language := winfonts.Language(fetchLanguage)

		if fetchProductID == "" {
			fetchProductID = getDefaultProductEditionID(version, edition)
		}

		if err := os.MkdirAll(fetchOutputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}

		tempISO := filepath.Join(os.TempDir(), fmt.Sprintf("winfonts_%s_%s_%s.iso", version, edition, arch))
		if keepISO {
			tempISO = filepath.Join(fetchOutputDir, fmt.Sprintf("windows_%s_%s_%s.iso", version, edition, arch))
		}

		fmt.Printf("Fetching Windows fonts...\n")
		fmt.Printf("  Version: %s\n", version)
		fmt.Printf("  Edition: %s\n", edition)
		fmt.Printf("  Architecture: %s\n", arch)
		fmt.Printf("  Language: %s\n", language)
		fmt.Printf("  Output: %s\n", fetchOutputDir)

		downloader := winfonts.NewWindowsDownloader(version, edition, arch, language)

		fmt.Println("\nObtaining download URL from Microsoft...")
		downloadURL, err := downloader.GetDownloadURL(fetchProductID)
		if err != nil {
			return fmt.Errorf("failed to get download URL: %w", err)
		}

		fmt.Printf("Download URL obtained\n")

		fmt.Printf("\nDownloading ISO to: %s\n", tempISO)
		if err := downloadFile(downloadURL, tempISO); err != nil {
			return fmt.Errorf("failed to download ISO: %w", err)
		}

		fmt.Println("\nExtracting fonts from ISO...")
		isoFile, err := os.Open(tempISO)
		if err != nil {
			return fmt.Errorf("failed to open ISO file: %w", err)
		}
		defer isoFile.Close()

		extractor, err := winfonts.NewFontExtractor(isoFile, fetchOutputDir)
		if err != nil {
			return fmt.Errorf("failed to create font extractor: %w", err)
		}

		if err := extractor.Run(cmd.Context()); err != nil {
			return fmt.Errorf("failed to extract fonts: %w", err)
		}

		if !keepISO {
			fmt.Printf("\nCleaning up temporary ISO file...\n")
			os.Remove(tempISO)
		}

		fmt.Printf("\nFonts successfully extracted to: %s\n", fetchOutputDir)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(fetchCmd)

	fetchCmd.Flags().StringVarP(&fetchOutputDir, "output", "o", "", "Output directory for extracted fonts (required)")
	fetchCmd.Flags().StringVarP(&fetchVersion, "version", "v", "windows11", "Windows version (windows11, windows10)")
	fetchCmd.Flags().StringVarP(&fetchEdition, "edition", "e", "pro", "Windows edition (home, pro, enterprise, education)")
	fetchCmd.Flags().StringVarP(&fetchArch, "arch", "a", "x64", "Architecture (x64, x86, ARM64)")
	fetchCmd.Flags().StringVarP(&fetchLanguage, "language", "l", "en-US", "Language (en-US, pt-BR)")
	fetchCmd.Flags().StringVarP(&fetchProductID, "product-id", "p", "", "Product edition ID (optional)")
	fetchCmd.Flags().BoolVarP(&keepISO, "keep-iso", "k", false, "Keep the downloaded ISO file after extraction")

	fetchCmd.MarkFlagRequired("output")
}
