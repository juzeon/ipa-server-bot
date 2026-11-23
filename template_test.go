package main

import (
	"strings"
	"testing"
	"time"
)

func TestUntrustedMetadataIsEscaped(t *testing.T) {
	app := Application{
		UUID:          "11111111-1111-1111-1111-111111111111",
		Name:          `</b><script>alert(1)</script>`,
		Package:       `com.example.app</string><key>url</key>`,
		Version:       `1.0<img src=x onerror=alert(1)>`,
		CertOrg:       "Evil\u202ETeam",
		IPA:           "https://cdn.example/app.ipa",
		Plist:         "https://cdn.example/manifest.plist",
		InstallPage:   "https://cdn.example/install.html",
		CertCreatedAt: time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC),
		CertExpiredAt: time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC),
		CreatedAt:     time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
	}

	info := BuildAppInfoTemplate(app)
	if strings.Contains(info, "<script>") || strings.Contains(info, "</b><script>") || strings.Contains(info, "\u202E") {
		t.Fatalf("info template was not escaped: %s", info)
	}
	if !strings.Contains(info, "&lt;script&gt;") {
		t.Fatalf("expected escaped script in info template: %s", info)
	}
	if !strings.Contains(info, `href="https://cdn.example/app.ipa"`) {
		t.Fatalf("expected download link: %s", info)
	}

	list, err := BuildAppListTemplate([]Application{app}, "bot_user", 1)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(list, "<script>") || strings.Contains(list, "\u202E") {
		t.Fatalf("list template was not escaped: %s", list)
	}

	plist, err := app.BuildPlistContent()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(plist, "</string><key>url</key>") {
		t.Fatalf("plist injection was not escaped: %s", plist)
	}
	if strings.Count(plist, "<key>url</key>") != 1 {
		t.Fatalf("plist url key count = %d\n%s", strings.Count(plist, "<key>url</key>"), plist)
	}
	if !strings.Contains(plist, "https://cdn.example/app.ipa") {
		t.Fatalf("plist lost the package url: %s", plist)
	}

	page, err := app.BuildInstallPageContent()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(page, "<script>") {
		t.Fatalf("install page was not escaped: %s", page)
	}
	if !strings.Contains(page, "script-src 'none'") {
		t.Fatalf("install page is missing CSP: %s", page)
	}
	if !strings.Contains(page, "itms-services://?action=download-manifest&amp;url=https%3A%2F%2Fcdn.example%2Fmanifest.plist") {
		t.Fatalf("install url was not encoded: %s", page)
	}
}

func TestUnsafeInstallURLsAreRejected(t *testing.T) {
	app := Application{
		Name:    "Demo",
		Package: "com.example.app",
		Version: "1.0",
		IPA:     "javascript:alert(1)",
		Plist:   "https://cdn.example/manifest.plist",
	}
	if _, err := app.BuildPlistContent(); err == nil {
		t.Fatal("expected javascript package url to be rejected")
	}
	info := BuildAppInfoTemplate(app)
	if strings.Contains(info, "javascript:") {
		t.Fatalf("javascript url was rendered: %s", info)
	}
	app.IPA = "https://cdn.example/app.ipa"
	app.Plist = "https://user:pass@cdn.example/manifest.plist"
	if _, err := app.BuildInstallPageContent(); err == nil {
		t.Fatal("expected credentialed manifest url to be rejected")
	}
}
