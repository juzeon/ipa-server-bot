package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"
)

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

func (o *Application) BuildPlistContent() string {
	plist := plistTemplate
	plist = strings.ReplaceAll(plist, "__IPA__", o.IPA)
	plist = strings.ReplaceAll(plist, "__NAME__", o.Name)
	plist = strings.ReplaceAll(plist, "__PACKAGE__", o.Package)
	plist = strings.ReplaceAll(plist, "__VERSION__", o.Version)
	return plist
}
func (o *Application) BuildInstallPageContent() string {
	return strings.ReplaceAll(installPageTemplate, "__URL__",
		"itms-services://?action=download-manifest&url="+o.Plist)
}

type Session struct {
	ChatID       int64         `json:"chat_id"`
	State        int           `json:"state"`
	Applications []Application `json:"applications"`
}

func NewSession(chatID int64) *Session {
	if _, err := os.Stat("sessions"); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			err = os.Mkdir("sessions", 0755)
			if err != nil {
				panic(err)
			}
		} else {
			panic(err)
		}
	}
	filename := "sessions/" + fmt.Sprintf("%d", chatID) + ".json"
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
		v, err := os.ReadFile(filename)
		if err != nil {
			panic(err)
		}
		var session Session
		err = json.Unmarshal(v, &session)
		if err != nil {
			panic(err)
		}
		return &session
	}
}
func (o *Session) Save() {
	slog.Info("save session", "chat_id", o.ChatID)
	v, err := json.MarshalIndent(o, "", "  ")
	if err != nil {
		panic(err)
	}
	err = os.WriteFile("sessions/"+fmt.Sprintf("%d", o.ChatID)+".json", v, 0644)
	if err != nil {
		panic(err)
	}
}
