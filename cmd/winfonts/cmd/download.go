package cmd

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/actions-precompiled/winfonts"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
)

var (
	windowsVersion   string
	windowsEdition   string
	windowsArch      string
	windowsLanguage  string
	productEditionID string
	outputFile       string
)

var downloadCmd = &cobra.Command{
	Use:   "download",
	Short: "Download Windows ISO from Microsoft servers",
	Long: `Download Windows ISO files directly from Microsoft's official servers.
This command uses the same API endpoints as Microsoft's official download tool.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		version := winfonts.WindowsVersion(windowsVersion)
		edition := winfonts.WindowsEdition(windowsEdition)
		arch := winfonts.Architecture(windowsArch)
		language := winfonts.Language(windowsLanguage)

		if productEditionID == "" {
			productEditionID = getDefaultProductEditionID(version, edition)
		}

		fmt.Printf("Downloading Windows ISO...\n")
		fmt.Printf("  Version: %s\n", version)
		fmt.Printf("  Edition: %s\n", edition)
		fmt.Printf("  Architecture: %s\n", arch)
		fmt.Printf("  Language: %s\n", language)

		downloader := winfonts.NewWindowsDownloader(version, edition, arch, language)

		fmt.Println("\nObtaining download URL from Microsoft...")
		downloadURL, err := downloader.GetDownloadURL(productEditionID)
		if err != nil {
			return fmt.Errorf("failed to get download URL: %w", err)
		}

		fmt.Printf("Download URL obtained: %s\n", downloadURL)

		if outputFile == "" {
			outputFile = fmt.Sprintf("windows_%s_%s_%s.iso", version, edition, arch)
		}

		fmt.Printf("\nDownloading to: %s\n", outputFile)
		if err := downloadFile(downloadURL, outputFile); err != nil {
			return fmt.Errorf("failed to download ISO: %w", err)
		}

		fmt.Printf("\nDownload completed successfully: %s\n", outputFile)
		return nil
	},
}

func downloadFile(url, filepath string) error {
	out, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer out.Close()

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	bar := progressbar.DefaultBytes(
		resp.ContentLength,
		"Downloading",
	)

	_, err = io.Copy(io.MultiWriter(out, bar), resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	fmt.Println()
	return nil
}

func getDefaultProductEditionID(version winfonts.WindowsVersion, edition winfonts.WindowsEdition) string {
	editionMap := map[winfonts.WindowsVersion]map[winfonts.WindowsEdition]string{
		winfonts.Windows11: {
			winfonts.EditionHome:       "2618",
			winfonts.EditionPro:        "2618",
			winfonts.EditionEnterprise: "2618",
			winfonts.EditionEducation:  "2618",
		},
		winfonts.Windows10: {
			winfonts.EditionHome:       "2935",
			winfonts.EditionPro:        "2935",
			winfonts.EditionEnterprise: "2935",
			winfonts.EditionEducation:  "2935",
		},
	}

	if versionMap, ok := editionMap[version]; ok {
		if id, ok := versionMap[edition]; ok {
			return id
		}
	}

	return "2618"
}

func init() {
	rootCmd.AddCommand(downloadCmd)

	downloadCmd.Flags().StringVarP(&windowsVersion, "version", "v", "windows11", "Windows version (windows11, windows10)")
	downloadCmd.Flags().StringVarP(&windowsEdition, "edition", "e", "pro", "Windows edition (home, pro, enterprise, education)")
	downloadCmd.Flags().StringVarP(&windowsArch, "arch", "a", "x64", "Architecture (x64, x86, ARM64)")
	downloadCmd.Flags().StringVarP(&windowsLanguage, "language", "l", "en-US", "Language (en-US, pt-BR)")
	downloadCmd.Flags().StringVarP(&productEditionID, "product-id", "p", "", "Product edition ID (optional, uses defaults if not specified)")
	downloadCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file path (default: windows_{version}_{edition}_{arch}.iso)")
}
