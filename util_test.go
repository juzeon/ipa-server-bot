package main

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestTelegramFileURLRejectsTraversal(t *testing.T) {
	config = &Config{Server: "https://bot.example", Token: "123456:SECRET"}
	t.Cleanup(func() { config = nil })

	got, err := telegramFileURL("/var/lib/telegram-bot-api/123456:SECRET/documents/file_0.ipa")
	if err != nil {
		t.Fatal(err)
	}
	want := "https://bot.example/file/bot123456:SECRET/documents/file_0.ipa"
	if got != want {
		t.Fatalf("got %s", got)
	}
	if strings.Contains(got, "%3A") {
		t.Fatalf("token colon was escaped: %s", got)
	}

	for _, p := range []string{
		"../../etc/passwd",
		"/etc/passwd",
		"documents/../../secret",
		"https://evil.example/a.ipa",
		"//evil.example/a.ipa",
		"documents/file?.ipa",
		`documents\file.ipa`,
		"",
	} {
		if _, err := telegramFileURL(p); err == nil {
			t.Fatalf("accepted %q", p)
		}
	}
}

func TestSameOriginRedirect(t *testing.T) {
	orig, _ := url.Parse("https://bot.example/file/bot123:ABC/documents/a.ipa")
	first := &http.Request{URL: orig}
	same, _ := url.Parse("https://bot.example/file/bot123:ABC/documents/a.ipa")
	if err := sameOriginRedirect(&http.Request{URL: same}, []*http.Request{first}); err != nil {
		t.Fatal(err)
	}
	other, _ := url.Parse("https://evil.example/a")
	if err := sameOriginRedirect(&http.Request{URL: other}, []*http.Request{first}); err == nil {
		t.Fatal("expected cross-origin redirect to be rejected")
	}
	downgrade, _ := url.Parse("http://bot.example/a")
	if err := sameOriginRedirect(&http.Request{URL: downgrade}, []*http.Request{first}); err == nil {
		t.Fatal("expected scheme change to be rejected")
	}
}
