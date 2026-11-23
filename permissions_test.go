package main

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/google/uuid"
)

func TestSessionAndConfigAreOwnerOnly(t *testing.T) {
	dir := t.TempDir()
	oldSessions := sessionDir
	oldConfig := configPath
	sessionDir = filepath.Join(dir, "sessions")
	configPath = filepath.Join(dir, "config.yml")
	t.Cleanup(func() {
		sessionDir = oldSessions
		configPath = oldConfig
	})

	if err := os.WriteFile(configPath, []byte("server: https://example.com\ntoken: t\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := NewConfig()
	if cfg.Token != "t" {
		t.Fatalf("token = %q", cfg.Token)
	}
	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("config mode = %o", info.Mode().Perm())
	}

	(&Session{ChatID: 42}).Save()
	info, err = os.Stat(sessionPath(42))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("session mode = %o", info.Mode().Perm())
	}
	dirInfo, err := os.Stat(sessionDir)
	if err != nil {
		t.Fatal(err)
	}
	if dirInfo.Mode().Perm()&0o077 != 0 {
		t.Fatalf("session dir mode = %o", dirInfo.Mode().Perm())
	}
}

func TestSessionChatIDIsPinned(t *testing.T) {
	old := sessionDir
	sessionDir = t.TempDir()
	t.Cleanup(func() { sessionDir = old })

	if err := os.WriteFile(sessionPath(7), []byte(`{"chat_id":99,"applications":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	session := NewSession(7)
	if session.ChatID != 7 {
		t.Fatalf("chat id = %d", session.ChatID)
	}
	session.Save()
	if _, err := os.Stat(sessionPath(99)); !os.IsNotExist(err) {
		t.Fatal("save followed the chat id stored in the file")
	}
	if _, err := os.Stat(sessionPath(7) + ".tmp"); !os.IsNotExist(err) {
		t.Fatal("temp session file was left behind")
	}
}

func TestSessionUpdatesDoNotDropWrites(t *testing.T) {
	old := sessionDir
	sessionDir = t.TempDir()
	t.Cleanup(func() { sessionDir = old })

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			withSession(5, func(session *Session) {
				session.Applications = append(session.Applications, Application{UUID: uuid.NewString()})
				session.Save()
			})
		}()
	}
	wg.Wait()
	if got := len(NewSession(5).Applications); got != 20 {
		t.Fatalf("saved %d applications", got)
	}
}
