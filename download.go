package winfonts

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/google/uuid"
)

const (
	orgID     = "y6jn8c31"
	profileID = "606624d44113"

	sessionEndpoint  = "https://vlscppe.microsoft.com/tags"
	skuEndpoint      = "https://www.microsoft.com/software-download-connector/api/getskuinformationbyproductedition"
	downloadEndpoint = "https://www.microsoft.com/software-download-connector/api/GetProductDownloadLinksBySku"
)

type WindowsVersion string

const (
	Windows11 WindowsVersion = "windows11"
	Windows10 WindowsVersion = "windows10"
)

type WindowsEdition string

const (
	EditionHome        WindowsEdition = "home"
	EditionPro         WindowsEdition = "pro"
	EditionEnterprise  WindowsEdition = "enterprise"
	EditionEducation   WindowsEdition = "education"
)

type Architecture string

const (
	ArchX64   Architecture = "x64"
	ArchX86   Architecture = "x86"
	ArchARM64 Architecture = "ARM64"
)

type Language string

const (
	LanguageEnglishUS Language = "en-US"
	LanguagePtBR      Language = "pt-BR"
)

type WindowsDownloader struct {
	client    *http.Client
	sessionID string
	version   WindowsVersion
	edition   WindowsEdition
	arch      Architecture
	language  Language
}

type SKUInfo struct {
	ID           string `json:"id"`
	Language     string `json:"language"`
	LanguageCode string `json:"languageCode"`
}

type DownloadOption struct {
	Architecture     string `json:"architecture"`
	DownloadType     string `json:"downloadType"`
	URI              string `json:"uri"`
	IsoSha256        string `json:"isoSha256"`
}

type DownloadResponse struct {
	ProductDownloadOptions []DownloadOption `json:"ProductDownloadOptions"`
}

func NewWindowsDownloader(version WindowsVersion, edition WindowsEdition, arch Architecture, language Language) *WindowsDownloader {
	return &WindowsDownloader{
		client:    &http.Client{},
		sessionID: uuid.New().String(),
		version:   version,
		edition:   edition,
		arch:      arch,
		language:  language,
	}
}

func (w *WindowsDownloader) registerSession() error {
	sessionURL := fmt.Sprintf("%s?org_id=%s&session_id=%s", sessionEndpoint, orgID, w.sessionID)

	req, err := http.NewRequest("GET", sessionURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create session request: %w", err)
	}

	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to register session: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("session registration failed with status: %d", resp.StatusCode)
	}

	return nil
}

func (w *WindowsDownloader) getSKUInformation(productEditionID string) ([]SKUInfo, error) {
	params := url.Values{}
	params.Add("profile", profileID)
	params.Add("productEditionId", productEditionID)
	params.Add("SKU", "undefined")
	params.Add("friendlyFileName", "undefined")
	params.Add("Locale", string(w.language))
	params.Add("sessionID", w.sessionID)

	skuURL := fmt.Sprintf("%s?%s", skuEndpoint, params.Encode())

	req, err := http.NewRequest("GET", skuURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create SKU request: %w", err)
	}

	resp, err := w.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get SKU information: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("SKU request failed with status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read SKU response: %w", err)
	}

	var skus []SKUInfo
	if err := json.Unmarshal(body, &skus); err != nil {
		return nil, fmt.Errorf("failed to parse SKU response: %w", err)
	}

	return skus, nil
}

func (w *WindowsDownloader) getDownloadLink(skuID string) (string, error) {
	params := url.Values{}
	params.Add("profile", profileID)
	params.Add("productEditionId", "undefined")
	params.Add("SKU", skuID)
	params.Add("friendlyFileName", "undefined")
	params.Add("Locale", string(w.language))
	params.Add("sessionID", w.sessionID)

	downloadURL := fmt.Sprintf("%s?%s", downloadEndpoint, params.Encode())

	req, err := http.NewRequest("GET", downloadURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create download request: %w", err)
	}

	req.Header.Set("Referer", fmt.Sprintf("https://www.microsoft.com/software-download/%s", w.version))

	resp, err := w.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get download link: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download link request failed with status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read download response: %w", err)
	}

	var downloadResp DownloadResponse
	if err := json.Unmarshal(body, &downloadResp); err != nil {
		return "", fmt.Errorf("failed to parse download response: %w", err)
	}

	for _, option := range downloadResp.ProductDownloadOptions {
		if option.Architecture == string(w.arch) {
			return option.URI, nil
		}
	}

	return "", fmt.Errorf("no download link found for architecture %s", w.arch)
}

func (w *WindowsDownloader) GetDownloadURL(productEditionID string) (string, error) {
	if err := w.registerSession(); err != nil {
		return "", fmt.Errorf("failed to register session: %w", err)
	}

	skus, err := w.getSKUInformation(productEditionID)
	if err != nil {
		return "", fmt.Errorf("failed to get SKU information: %w", err)
	}

	if len(skus) == 0 {
		return "", fmt.Errorf("no SKUs found for edition")
	}

	downloadURL, err := w.getDownloadLink(skus[0].ID)
	if err != nil {
		return "", fmt.Errorf("failed to get download link: %w", err)
	}

	return downloadURL, nil
}
