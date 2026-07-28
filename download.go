package winfonts

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	orgID      = "y6jn8c31"
	profileID  = "606624d44113"
	// instanceID is Microsoft's constant ov-df customer id (same as Fido/Rufus).
	instanceID = "560dc9f3-1aa5-4a2f-b63c-9e18f8d0e175"

	sessionEndpoint  = "https://vlscppe.microsoft.com/tags"
	mdtEndpoint      = "https://ov-df.microsoft.com/mdt.js"
	ovdfEndpoint     = "https://ov-df.microsoft.com/"
	skuEndpoint      = "https://www.microsoft.com/software-download-connector/api/getskuinformationbyproductedition"
	downloadEndpoint = "https://www.microsoft.com/software-download-connector/api/GetProductDownloadLinksBySku"
)

// Default product edition IDs used by Microsoft's software-download pages.
// Windows 11 Home/Pro/Edu multi-edition ISO: 3321 (x64), 3324 (ARM64).
// Windows 10 Home/Pro/Edu: 2618.
const (
	DefaultProductEditionWin11X64   = "3321"
	DefaultProductEditionWin11ARM64 = "3324"
	DefaultProductEditionWin10      = "2618"
)

type WindowsVersion string

const (
	Windows11 WindowsVersion = "windows11"
	Windows10 WindowsVersion = "windows10"
)

type WindowsEdition string

const (
	EditionHome       WindowsEdition = "home"
	EditionPro        WindowsEdition = "pro"
	EditionEnterprise WindowsEdition = "enterprise"
	EditionEducation  WindowsEdition = "education"
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

// languageNames maps CLI language codes to Microsoft SKU language names.
var languageNames = map[Language][]string{
	LanguageEnglishUS: {"English", "English (United States)"},
	LanguagePtBR:      {"Brazilian Portuguese", "Portuguese (Brazil)"},
}

type WindowsDownloader struct {
	client    *http.Client
	sessionID string
	version   WindowsVersion
	edition   WindowsEdition
	arch      Architecture
	language  Language
}

type SKUInfo struct {
	ID                          string      `json:"Id"`
	Description                 string      `json:"Description"`
	ProductDisplayName          string      `json:"ProductDisplayName"`
	Language                    string      `json:"Language"`
	LocalizedLanguage           string      `json:"LocalizedLanguage"`
	LocalizedProductDisplayName string      `json:"LocalizedProductDisplayName"`
	ProductEditionName          interface{} `json:"ProductEditionName"`
	FriendlyFileNames           []string    `json:"FriendlyFileNames"`
}

type SKUResponse struct {
	Skus                []SKUInfo              `json:"Skus"`
	ValidationContainer map[string]interface{} `json:"ValidationContainer"`
	Tickets             interface{}            `json:"Tickets"`
	Errors              []APIError             `json:"Errors"`
}

type DownloadOption struct {
	DownloadType                int    `json:"DownloadType"`
	Name                        string `json:"Name"`
	Uri                         string `json:"Uri"`
	ProductDisplayName          string `json:"ProductDisplayName"`
	Language                    string `json:"Language"`
	LocalizedProductDisplayName string `json:"LocalizedProductDisplayName"`
	LocalizedLanguage           string `json:"LocalizedLanguage"`
}

type APIError struct {
	Key   string `json:"Key"`
	Value string `json:"Value"`
	Type  int    `json:"Type"`
}

type DownloadResponse struct {
	ProductDownloadOptions     []DownloadOption       `json:"ProductDownloadOptions"`
	ProductDownload            interface{}            `json:"ProductDownload"`
	ValidationContainer        map[string]interface{} `json:"ValidationContainer"`
	DownloadExpirationDatetime string                 `json:"DownloadExpirationDatetime"`
	Errors                     []APIError             `json:"Errors"`
}

func NewWindowsDownloader(version WindowsVersion, edition WindowsEdition, arch Architecture, language Language) *WindowsDownloader {
	jar, _ := cookiejar.New(nil)
	return &WindowsDownloader{
		client:    &http.Client{Jar: jar, Timeout: 60 * time.Second},
		sessionID: uuid.New().String(),
		version:   version,
		edition:   edition,
		arch:      arch,
		language:  language,
	}
}

// DefaultProductEditionID returns Microsoft's product edition id for the given version/arch.
func DefaultProductEditionID(version WindowsVersion, arch Architecture) string {
	switch version {
	case Windows10:
		return DefaultProductEditionWin10
	case Windows11:
		if arch == ArchARM64 {
			return DefaultProductEditionWin11ARM64
		}
		return DefaultProductEditionWin11X64
	default:
		return DefaultProductEditionWin11X64
	}
}

func (w *WindowsDownloader) softwareDownloadPath() string {
	if w.version == Windows10 {
		// Microsoft's Windows 10 ISO page uses this path.
		return "windows10ISO"
	}
	return string(w.version)
}

func (w *WindowsDownloader) getArchDownloadType() int {
	switch w.arch {
	case ArchX86:
		return 0
	case ArchX64:
		return 1
	case ArchARM64:
		return 2
	default:
		return 1
	}
}

func (w *WindowsDownloader) addBrowserHeaders(req *http.Request) {
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("DNT", "1")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Referer", fmt.Sprintf("https://www.microsoft.com/en-us/software-download/%s", w.softwareDownloadPath()))
	req.Header.Set("Origin", "https://www.microsoft.com")
}

func (w *WindowsDownloader) validateLocale() error {
	localeURL := fmt.Sprintf("https://www.microsoft.com/en-US/software-download/%s", w.softwareDownloadPath())

	req, err := http.NewRequest("GET", localeURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create locale request: %w", err)
	}

	w.addBrowserHeaders(req)

	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to validate locale: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("locale validation failed with status: %d", resp.StatusCode)
	}

	return nil
}

func (w *WindowsDownloader) registerSession() error {
	sessionURL := fmt.Sprintf("%s?org_id=%s&session_id=%s", sessionEndpoint, orgID, w.sessionID)

	req, err := http.NewRequest("GET", sessionURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create session request: %w", err)
	}

	w.addBrowserHeaders(req)

	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to register session: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("session registration failed with status: %d", resp.StatusCode)
	}

	return nil
}

// registerDeviceFingerprint completes Microsoft's ov-df anti-bot handshake required
// before GetProductDownloadLinksBySku will return ISO links (otherwise SentinelReject).
func (w *WindowsDownloader) registerDeviceFingerprint() error {
	mdtURL := fmt.Sprintf("%s?instanceId=%s&PageId=si&session_id=%s", mdtEndpoint, instanceID, w.sessionID)

	req, err := http.NewRequest("GET", mdtURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create mdt.js request: %w", err)
	}
	w.addBrowserHeaders(req)
	req.Header.Set("Accept", "*/*")

	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch mdt.js: %w", err)
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return fmt.Errorf("failed to read mdt.js: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("mdt.js request failed with status: %d", resp.StatusCode)
	}

	wMatch := regexp.MustCompile(`[?&]w=([A-F0-9]+)`).FindSubmatch(body)
	rtMatch := regexp.MustCompile(`rticks\s*=\s*"?\+?(\d+)`).FindSubmatch(body)
	if wMatch == nil || rtMatch == nil {
		return fmt.Errorf("failed to parse ov-df parameters from mdt.js")
	}

	replyURL := fmt.Sprintf(
		"%s?session_id=%s&CustomerId=%s&PageId=si&w=%s&mdt=%d&rticks=%s",
		ovdfEndpoint, w.sessionID, instanceID, string(wMatch[1]), time.Now().UnixMilli(), string(rtMatch[1]),
	)

	req, err = http.NewRequest("GET", replyURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create ov-df reply request: %w", err)
	}
	w.addBrowserHeaders(req)
	req.Header.Set("Accept", "*/*")

	resp, err = w.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send ov-df reply: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ov-df reply failed with status: %d", resp.StatusCode)
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

	w.addBrowserHeaders(req)

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

	var skuResponse SKUResponse
	if err := json.Unmarshal(body, &skuResponse); err != nil {
		return nil, fmt.Errorf("failed to parse SKU response: %w", err)
	}
	if len(skuResponse.Errors) > 0 {
		return nil, fmt.Errorf("SKU API error: %s", skuResponse.Errors[0].Value)
	}

	return skuResponse.Skus, nil
}

func (w *WindowsDownloader) matchSKU(skus []SKUInfo) (SKUInfo, error) {
	wantNames := languageNames[w.language]
	if len(wantNames) == 0 {
		// Fall back to treating the language code / raw value as a display name.
		wantNames = []string{string(w.language)}
	}

	for _, sku := range skus {
		for _, want := range wantNames {
			if strings.EqualFold(sku.Language, want) || strings.EqualFold(sku.LocalizedLanguage, want) {
				return sku, nil
			}
		}
	}

	// Secondary match: localized language contains the locale tag (e.g. "en-US").
	localeHint := string(w.language)
	for _, sku := range skus {
		if strings.Contains(strings.ToLower(sku.LocalizedLanguage), strings.ToLower(localeHint)) {
			return sku, nil
		}
	}

	available := make([]string, 0, len(skus))
	for _, sku := range skus {
		available = append(available, sku.Language)
	}
	return SKUInfo{}, fmt.Errorf("no SKU found for language %s (available: %s)", w.language, strings.Join(available, ", "))
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

	w.addBrowserHeaders(req)

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

	if len(downloadResp.Errors) > 0 {
		err0 := downloadResp.Errors[0]
		if err0.Type == 9 || strings.Contains(err0.Key, "715") || strings.Contains(err0.Value, "715-123130") {
			return "", fmt.Errorf("Microsoft rate-limited this IP for ISO downloads (code 715-123130); try again later")
		}
		if strings.Contains(err0.Key, "Sentinel") || strings.Contains(err0.Value, "Sentinel") {
			return "", fmt.Errorf("Microsoft anti-bot rejected the download request (SentinelReject); session handshake may have failed")
		}
		return "", fmt.Errorf("download API error: %s", err0.Value)
	}

	archType := w.getArchDownloadType()
	var availableTypes []string
	for _, option := range downloadResp.ProductDownloadOptions {
		availableTypes = append(availableTypes, fmt.Sprintf("%d", option.DownloadType))
		if option.DownloadType == archType {
			return option.Uri, nil
		}
	}

	// If only one arch is offered (common for Win11 multi-edition x64/ARM64 split pages), use it.
	if len(downloadResp.ProductDownloadOptions) == 1 {
		return downloadResp.ProductDownloadOptions[0].Uri, nil
	}

	return "", fmt.Errorf("no download link found for architecture %s (download type %d; available types: %s)",
		w.arch, archType, strings.Join(availableTypes, ", "))
}

// GetDownloadURL obtains a Microsoft ISO download URL via the consumer
// software-download connector (ov-df handshake + language-matched SKU).
func (w *WindowsDownloader) GetDownloadURL(productEditionID string) (string, error) {
	if productEditionID == "" {
		productEditionID = DefaultProductEditionID(w.version, w.arch)
	}

	if err := w.validateLocale(); err != nil {
		return "", fmt.Errorf("failed to validate locale: %w", err)
	}

	if err := w.registerSession(); err != nil {
		return "", fmt.Errorf("failed to register session: %w", err)
	}

	if err := w.registerDeviceFingerprint(); err != nil {
		return "", fmt.Errorf("failed to complete device fingerprint handshake: %w", err)
	}

	skus, err := w.getSKUInformation(productEditionID)
	if err != nil {
		return "", fmt.Errorf("failed to get SKU information: %w", err)
	}

	if len(skus) == 0 {
		return "", fmt.Errorf("no SKUs found for product edition %s", productEditionID)
	}

	sku, err := w.matchSKU(skus)
	if err != nil {
		return "", err
	}

	downloadURL, err := w.getDownloadLink(sku.ID)
	if err != nil {
		return "", fmt.Errorf("failed to get download link: %w", err)
	}

	return downloadURL, nil
}
