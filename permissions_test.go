package main

import (
	"os"
	"path/filepath"
	"testing"
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
