package main

import (
	"testing"

	"github.com/go-telegram/bot/models"
)

func TestIsPrivateChat(t *testing.T) {
	private := &models.Update{Message: &models.Message{Chat: models.Chat{Type: "private"}}}
	if !isPrivateChat(private) {
		t.Fatal("expected a private chat")
	}
	group := &models.Update{Message: &models.Message{Chat: models.Chat{Type: "group"}}}
	if isPrivateChat(group) {
		t.Fatal("group chat should be rejected")
	}
	callback := &models.Update{CallbackQuery: &models.CallbackQuery{
		Message: &models.Message{Chat: models.Chat{Type: "supergroup"}},
	}}
	if isPrivateChat(callback) {
		t.Fatal("supergroup callback should be rejected")
	}
	if isPrivateChat(nil) || isPrivateChat(&models.Update{}) {
		t.Fatal("missing chat should be rejected")
	}
}
