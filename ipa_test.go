package main

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsIPAFileName(t *testing.T) {
	if !isIPAFileName("My App.IPA") {
		t.Fatal("expected uppercase extension to be accepted")
	}
	for _, name := range []string{"", "../a.ipa", `a\b.ipa`, "a.ipa\x00.jpg", "notes.txt"} {
		if isIPAFileName(name) {
			t.Fatalf("accepted unsafe name %q", name)
		}
	}
}

func TestParseIPARejectsBombsAndTraversal(t *testing.T) {
	origMeta := maxMetadataBytes
	origEntries := maxZipEntries
	t.Cleanup(func() {
		maxMetadataBytes = origMeta
		maxZipEntries = origEntries
	})

	valid := testIPA(t, map[string]string{
		"Payload/Demo.app/Info.plist":            infoPlistXML("com.example.demo", "Hello &amp; Co", "1.2.3"),
		"Payload/../Evil.app/Info.plist":         infoPlistXML("com.evil.app", "Evil", "9"),
		"Payload/Demo.app/Frameworks/Info.plist": infoPlistXML("com.framework", "Framework", "0"),
	})
	app, err := parseIPA(valid)
	if err != nil {
		t.Fatal(err)
	}
	if app.Package != "com.example.demo" || app.Name != "Hello & Co" || app.Version != "1.2.3" {
		t.Fatalf("parsed %+v", app)
	}

	maxMetadataBytes = 128
	bomb := testIPA(t, map[string]string{
		"Payload/Bomb.app/Info.plist": strings.Repeat("A", 200),
	})
	if _, err := parseIPA(bomb); err == nil || !strings.Contains(err.Error(), "too large") {
		t.Fatalf("expected metadata limit, got %v", err)
	}

	maxMetadataBytes = origMeta
	maxZipEntries = 3
	tooMany := testIPA(t, map[string]string{
		"Payload/A.app/Info.plist": infoPlistXML("com.example.a", "A", "1"),
		"a":                        "1",
		"b":                        "2",
		"c":                        "3",
	})
	if _, err := parseIPA(tooMany); err == nil || !strings.Contains(err.Error(), "too many") {
		t.Fatalf("expected entry limit, got %v", err)
	}
}

func TestXMLNestingLimit(t *testing.T) {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0"?><plist version="1.0">`)
	for i := 0; i < 40; i++ {
		b.WriteString("<array>")
	}
	for i := 0; i < 40; i++ {
		b.WriteString("</array>")
	}
	b.WriteString("</plist>")
	if !xmlNestingTooDeep([]byte(b.String()), 32) {
		t.Fatal("expected deep plist to be rejected")
	}
	if xmlNestingTooDeep([]byte("bplist00"), 32) {
		t.Fatal("binary plist should skip the xml depth check")
	}
}

func TestCopyLimited(t *testing.T) {
	var dst bytes.Buffer
	if err := copyLimited(&dst, strings.NewReader("abcdef"), 3); err != errTooLarge {
		t.Fatalf("got %v", err)
	}
	dst.Reset()
	if err := copyLimited(&dst, strings.NewReader("abc"), 3); err != nil || dst.String() != "abc" {
		t.Fatalf("got %v %q", err, dst.String())
	}
}

func TestValidObjectKey(t *testing.T) {
	if !validObjectKey("11111111-1111-1111-1111-111111111111/0.ipa") {
		t.Fatal("expected uuid key")
	}
	for _, key := range []string{"", "../x", "/abs", `a\b`, "a//b", ".", "a/../b"} {
		if validObjectKey(key) {
			t.Fatalf("accepted %q", key)
		}
	}
}

func infoPlistXML(bundle, name, version string) string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0"><dict>
<key>CFBundleIdentifier</key><string>` + bundle + `</string>
<key>CFBundleDisplayName</key><string>` + name + `</string>
<key>CFBundleShortVersionString</key><string>` + version + `</string>
</dict></plist>`
}

func testIPA(t *testing.T, files map[string]string) string {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "app.ipa")
	if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
