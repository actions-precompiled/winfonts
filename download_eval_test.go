package winfonts

import (
	"strings"
	"testing"
)

func TestEvaluationDownloadURL(t *testing.T) {
	d := NewWindowsDownloader(Windows11, EditionEnterprise, ArchX64, LanguageEnglishUS)
	url, err := d.getEvaluationDownloadURL()
	if err != nil {
		t.Fatalf("getEvaluationDownloadURL: %v", err)
	}
	t.Logf("url=%s", url)
	if !strings.Contains(strings.ToLower(url), ".iso") {
		t.Fatalf("expected .iso URL, got %s", url)
	}
	if strings.Contains(strings.ToUpper(url), "LTSC") {
		t.Fatalf("did not expect LTSC ISO: %s", url)
	}
}

func TestEvaluationDownloadURLPtBR(t *testing.T) {
	d := NewWindowsDownloader(Windows11, EditionEnterprise, ArchX64, LanguagePtBR)
	url, err := d.getEvaluationDownloadURL()
	if err != nil {
		t.Fatalf("getEvaluationDownloadURL pt-BR: %v", err)
	}
	t.Logf("url=%s", url)
	if !strings.Contains(strings.ToLower(url), "pt-br") && !strings.Contains(strings.ToLower(url), "brazilian") {
		// File names usually contain pt-br locale.
		t.Logf("warning: locale not obvious in URL: %s", url)
	}
}
