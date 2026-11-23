package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

const applicationTTL = 7 * 24 * time.Hour

func applicationExpired(created, now time.Time) bool {
	if created.IsZero() || now.Before(created) {
		return false
	}
	return now.Sub(created) >= applicationTTL
}

func retainApplications(apps []Application, now time.Time, deleteFn func(string) error) ([]Application, bool) {
	kept := make([]Application, 0, len(apps))
	changed := false
	for _, app := range apps {
		if !applicationExpired(app.CreatedAt, now) {
			kept = append(kept, app)
			continue
		}
		if err := deleteFn(app.UUID); err != nil {
			slog.Error("expire application", "uuid", app.UUID, "err", redactSecrets(err.Error()))
			kept = append(kept, app)
			continue
		}
		slog.Info("expired application", "uuid", app.UUID)
		changed = true
	}
	return kept, changed
}

func expireChat(chatID int64, now time.Time) {
	withSession(chatID, func(session *Session) {
		kept, changed := retainApplications(session.Applications, now, DeleteApplicationObjects)
		if !changed {
			return
		}
		session.Applications = kept
		session.Save()
	})
}

func digitsOnly(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func listSessionChatIDs() ([]int64, error) {
	entries, err := os.ReadDir(sessionDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var ids []int64
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		idPart, ok := strings.CutSuffix(name, ".json")
		if !ok || !digitsOnly(idPart) {
			continue
		}
		id, err := strconv.ParseInt(idPart, 10, 64)
		if err != nil {
			continue
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func expireStoredApplications(now time.Time) {
	ids, err := listSessionChatIDs()
	if err != nil {
		slog.Error("list sessions", "err", err)
		return
	}
	for _, id := range ids {
		func() {
			defer func() {
				if rec := recover(); rec != nil {
					slog.Error("expire session", "chat_id", id, "err", redactSecrets(fmt.Sprint(rec)))
				}
			}()
			expireChat(id, now)
		}()
	}
}

func runExpiryLoop(ctx context.Context) {
	go func() {
		expireStoredApplications(time.Now())
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				expireStoredApplications(now)
			}
		}
	}()
}
