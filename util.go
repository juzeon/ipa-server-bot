package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"path"
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

func sanitizeTelegramFilePath(filePath string) (string, error) {
	if config == nil || config.Token == "" || filePath == "" {
		return "", errors.New("invalid file path")
	}
	prefix := "/var/lib/telegram-bot-api/" + config.Token + "/"
	if strings.HasPrefix(filePath, "/") && !strings.HasPrefix(filePath, prefix) {
		return "", errors.New("invalid file path")
	}
	filePath = strings.TrimPrefix(filePath, prefix)
	filePath = strings.TrimPrefix(filePath, "/")
	if filePath == "" || strings.Contains(filePath, "\\") || strings.ContainsAny(filePath, "\r\n\t?#") || strings.Contains(filePath, "://") {
		return "", errors.New("invalid file path")
	}
	for _, part := range strings.Split(filePath, "/") {
		if part == "" || part == "." || part == ".." || strings.Contains(part, ":") {
			return "", errors.New("invalid file path")
		}
	}
	cleaned := path.Clean("/" + filePath)
	rel := strings.TrimPrefix(cleaned, "/")
	if rel == "" || rel == "." || rel == ".." || strings.HasPrefix(rel, "../") || strings.Contains(rel, "..") {
		return "", errors.New("invalid file path")
	}
	return rel, nil
}

func telegramFileURL(filePath string) (string, error) {
	rel, err := sanitizeTelegramFilePath(filePath)
	if err != nil {
		return "", err
	}
	if config == nil {
		return "", errors.New("invalid server url")
	}
	base, err := url.Parse(config.Server)
	if err != nil || base.Hostname() == "" || (base.Scheme != "https" && base.Scheme != "http") || base.User != nil {
		return "", errors.New("invalid server url")
	}
	segments := strings.Split(rel, "/")
	escaped := make([]string, len(segments))
	for i, seg := range segments {
		escaped[i] = url.PathEscape(seg)
	}
	basePath := strings.TrimSuffix(base.EscapedPath(), "/")
	return base.Scheme + "://" + base.Host + basePath + "/file/bot" + config.Token + "/" + strings.Join(escaped, "/"), nil
}

func sameOriginRedirect(req *http.Request, via []*http.Request) error {
	if req == nil || req.URL == nil || len(via) == 0 || len(via) >= 3 || via[0] == nil || via[0].URL == nil {
		return errors.New("redirect is not allowed")
	}
	orig := via[0].URL
	if req.URL.Scheme != orig.Scheme || !strings.EqualFold(req.URL.Hostname(), orig.Hostname()) || req.URL.Port() != orig.Port() {
		return errors.New("redirect to another origin is not allowed")
	}
	return nil
}

func DownloadTelegramFile(filePath string, dst io.Writer) error {
	rawURL, err := telegramFileURL(filePath)
	if err != nil {
		slog.Error("reject telegram file path")
		return errors.New("telegram file download failed")
	}
	slog.Info("download telegram file")
	resp, err := Client.R().SetDoNotParseResponse(true).Get(rawURL)
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
