package main

import (
	"context"
	"log/slog"
	"strings"
	"time"
)

func redactSecrets(s string) string {
	if config == nil || s == "" {
		return s
	}
	if config.Token != "" {
		s = strings.ReplaceAll(s, config.Token, "[redacted]")
	}
	if len(config.S3.SecretAccessKey) >= 8 {
		s = strings.ReplaceAll(s, config.S3.SecretAccessKey, "[redacted]")
	}
	return s
}

func DownloadTelegramFile(filePath string) ([]byte, error) {
	filePath = strings.Replace(filePath, "/var/lib/telegram-bot-api/"+config.Token+"/", "", -1)
	slog.Info("download telegram file", "filePath", redactSecrets(filePath))
	resp, err := Client.R().Get(config.Server + "/file/bot" + config.Token + "/" + filePath)
	if err != nil {
		return nil, err
	}
	return resp.Body(), nil
}
func CreateTimeoutContext(d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), d)
}
