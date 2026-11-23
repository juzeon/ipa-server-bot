package main

import (
	"testing"

	"github.com/gotd/botapi"
)

func TestIsPrivateChat(t *testing.T) {
	private := &botapi.Update{Message: &botapi.Message{Chat: botapi.Chat{Type: botapi.ChatTypePrivate}}}
	if !isPrivateChat(private) {
		t.Fatal("expected a private chat")
	}
	group := &botapi.Update{Message: &botapi.Message{Chat: botapi.Chat{Type: botapi.ChatTypeGroup}}}
	if isPrivateChat(group) {
		t.Fatal("group chat should be rejected")
	}
	callback := &botapi.Update{CallbackQuery: &botapi.CallbackQuery{
		Message: &botapi.Message{Chat: botapi.Chat{Type: botapi.ChatTypeSupergroup}},
	}}
	if isPrivateChat(callback) {
		t.Fatal("supergroup callback should be rejected")
	}
	if isPrivateChat(nil) || isPrivateChat(&botapi.Update{}) {
		t.Fatal("missing chat should be rejected")
	}
}

func TestParseConfigPath(t *testing.T) {
	got, err := parseConfigPath(nil)
	if err != nil || got != "config.yml" {
		t.Fatalf("default path = %q err = %v", got, err)
	}
	got, err = parseConfigPath([]string{"-c", "/etc/ipa-server-bot/config.yml"})
	if err != nil || got != "/etc/ipa-server-bot/config.yml" {
		t.Fatalf("flag path = %q err = %v", got, err)
	}
	if _, err := parseConfigPath([]string{"-c", ""}); err == nil {
		t.Fatal("empty config path should be rejected")
	}
}
