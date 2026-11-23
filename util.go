package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
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

var errTooLarge = errors.New("file is too large")

func copyLimited(dst io.Writer, src io.Reader, limit int64) error {
	n, err := io.Copy(dst, io.LimitReader(src, limit+1))
	if err != nil {
		return err
	}
	if n > limit {
		return errTooLarge
	}
	return nil
}

func readLimited(r io.Reader, limit int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, errTooLarge
	}
	return data, nil
}

func DownloadTelegramFile(filePath string, dst io.Writer) error {
	filePath = strings.Replace(filePath, "/var/lib/telegram-bot-api/"+config.Token+"/", "", -1)
	slog.Info("download telegram file", "filePath", redactSecrets(filePath))
	resp, err := Client.R().SetDoNotParseResponse(true).Get(config.Server + "/file/bot" + config.Token + "/" + filePath)
	if err != nil || resp == nil || resp.RawBody() == nil {
		if err != nil {
			slog.Error("telegram download failed", "err", redactSecrets(err.Error()))
		}
		return errors.New("telegram file download failed")
	}
	defer resp.RawBody().Close()
	if resp.StatusCode() != http.StatusOK {
		slog.Error("telegram download failed", "status", resp.StatusCode())
		return errors.New("telegram file download failed")
	}
	if err := copyLimited(dst, resp.RawBody(), maxIPABytes); err != nil {
		if errors.Is(err, errTooLarge) {
			return userErr("The IPA file is too large.")
		}
		slog.Error("telegram download failed", "err", redactSecrets(err.Error()))
		return errors.New("telegram file download failed")
	}
	return nil
}
func CreateTimeoutContext(d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), d)
}
