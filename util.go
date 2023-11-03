package main

import (
	"context"
	"log/slog"
	"strings"
	"time"
)

func DownloadTelegramFile(filePath string) ([]byte, error) {
	filePath = strings.Replace(filePath, "/var/lib/telegram-bot-api/"+config.Token+"/", "", -1)
	slog.Info("download telegram file", "filePath", filePath)
	resp, err := Client.R().Get(config.Server + "/file/bot" + config.Token + "/" + filePath)
	if err != nil {
		return nil, err
	}
	return resp.Body(), nil
}
func CreateTimeoutContext(d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), d)
}
