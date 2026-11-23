package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestApplicationExpired(t *testing.T) {
	now := time.Date(2026, 11, 24, 0, 0, 0, 0, time.UTC)
	if applicationExpired(time.Time{}, now) {
		t.Fatal("missing creation time should be kept")
	}
	if applicationExpired(now.Add(-applicationTTL+time.Second), now) {
		t.Fatal("application inside the retention window should be kept")
	}
	if !applicationExpired(now.Add(-applicationTTL), now) {
		t.Fatal("application at the retention limit should expire")
	}
	if !applicationExpired(now.Add(-applicationTTL-time.Hour), now) {
		t.Fatal("old application should expire")
	}
}

func TestExpireStoredApplications(t *testing.T) {
	old := sessionDir
	sessionDir = t.TempDir()
	t.Cleanup(func() { sessionDir = old })

	now := time.Date(2026, 11, 24, 0, 0, 0, 0, time.UTC)
	fresh := Application{UUID: "11111111-1111-1111-1111-111111111111", Name: "fresh", CreatedAt: now.Add(-time.Hour)}
	stale := Application{UUID: "22222222-2222-2222-2222-222222222222", Name: "stale", CreatedAt: now.Add(-8 * 24 * time.Hour)}
	stuck := Application{UUID: "33333333-3333-3333-3333-333333333333", Name: "stuck", CreatedAt: now.Add(-9 * 24 * time.Hour)}
	(&Session{ChatID: 7, Applications: []Application{fresh, stale, stuck}}).Save()
	if err := os.WriteFile(filepath.Join(sessionDir, "notes.txt"), []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}

	deleted := map[string]bool{}
	kept, changed := retainApplications(NewSession(7).Applications, now, func(id string) error {
		if id == stuck.UUID {
			return errors.New("storage unavailable")
		}
		deleted[id] = true
		return nil
	})
	if !changed || !deleted[stale.UUID] || deleted[stuck.UUID] {
		t.Fatalf("deleted = %#v changed = %v", deleted, changed)
	}
	if len(kept) != 2 || kept[0].UUID != fresh.UUID || kept[1].UUID != stuck.UUID {
		t.Fatalf("kept = %#v", kept)
	}

	withSession(7, func(session *Session) {
		session.Applications = kept
		session.Save()
	})
	got := NewSession(7).Applications
	if len(got) != 2 || got[0].Name != "fresh" || got[1].Name != "stuck" {
		t.Fatalf("stored = %#v", got)
	}
	if _, err := os.Stat(filepath.Join(sessionDir, "notes.txt")); err != nil {
		t.Fatal(err)
	}
	ids, err := listSessionChatIDs()
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 1 || ids[0] != 7 {
		t.Fatalf("session ids = %#v", ids)
	}
}
