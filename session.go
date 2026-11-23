package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var sessionLocks sync.Map

func sessionLock(chatID int64) *sync.Mutex {
	v, _ := sessionLocks.LoadOrStore(chatID, &sync.Mutex{})
	return v.(*sync.Mutex)
}

func withSession(chatID int64, fn func(*Session)) {
	mu := sessionLock(chatID)
	mu.Lock()
	defer mu.Unlock()
	fn(NewSession(chatID))
}

var sessionDir = "sessions"

func sessionPath(chatID int64) string {
	name := fmt.Sprintf("%d.json", chatID)
	if name != filepath.Base(name) {
		panic(errors.New("invalid session"))
	}
	return filepath.Join(sessionDir, name)
}

const (
	SessionStateDefault = iota
)

type Application struct {
	UUID          string    `json:"uuid"`
	Name          string    `json:"name"`
	Package       string    `json:"package"`
	Version       string    `json:"version"`
	CertCreatedAt time.Time `json:"cert_created_at"`
	CertExpiredAt time.Time `json:"cert_expired_at"`
	CertOrg       string    `json:"cert_org"`
	IPA           string    `json:"ipa"`
	Plist         string    `json:"plist"`
	InstallPage   string    `json:"install_page"`
	CreatedAt     time.Time `json:"created_at"`
}

func (o *Application) BuildPlistContent() (string, error) {
	ipa, err := sanitizeHTTPURL(o.IPA)
	if err != nil {
		return "", err
	}
	return strings.NewReplacer(
		"__IPA__", html.EscapeString(ipa),
		"__NAME__", htmlText(o.Name),
		"__PACKAGE__", htmlText(o.Package),
		"__VERSION__", htmlText(o.Version),
	).Replace(plistTemplate), nil
}
func (o *Application) BuildInstallPageContent() (string, error) {
	manifest, err := sanitizeHTTPURL(o.Plist)
	if err != nil {
		return "", err
	}
	installURL := "itms-services://?action=download-manifest&url=" + url.QueryEscape(manifest)
	return strings.NewReplacer(
		"__URL__", html.EscapeString(installURL),
		"__NAME__", htmlText(o.Name),
	).Replace(installPageTemplate), nil
}

type Session struct {
	ChatID       int64         `json:"chat_id"`
	State        int           `json:"state"`
	Applications []Application `json:"applications"`
}

func ensureSessionDir() {
	if err := os.MkdirAll(sessionDir, 0o700); err != nil {
		panic(err)
	}
	tightenOwnerOnly(sessionDir, 0o700)
}

func NewSession(chatID int64) *Session {
	ensureSessionDir()
	filename := sessionPath(chatID)
	if _, err := os.Stat(filename); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			slog.Info("create session", "chat_id", chatID)
			return &Session{
				ChatID: chatID,
				State:  SessionStateDefault,
			}
		} else {
			panic(err)
		}
	} else {
		slog.Info("read session", "chat_id", chatID)
		tightenOwnerOnly(filename, 0o600)
		v, err := os.ReadFile(filename)
		if err != nil {
			panic(err)
		}
		var session Session
		err = json.Unmarshal(v, &session)
		if err != nil {
			panic(err)
		}
		session.ChatID = chatID
		return &session
	}
}
func (o *Session) Save() {
	ensureSessionDir()
	slog.Info("save session", "chat_id", o.ChatID)
	v, err := json.MarshalIndent(o, "", "  ")
	if err != nil {
		panic(err)
	}
	path := sessionPath(o.ChatID)
	tmp := path + ".tmp"
	if err = os.WriteFile(tmp, v, 0o600); err != nil {
		panic(err)
	}
	if err = os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		panic(err)
	}
	tightenOwnerOnly(path, 0o600)
}
