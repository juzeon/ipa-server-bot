package main

import (
	"context"
	"errors"
	"io"
	"os"
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
	if config.APIHash != "" {
		s = strings.ReplaceAll(s, config.APIHash, "[redacted]")
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

type byteLimitWriter struct {
	w     io.Writer
	n     int64
	limit int64
}

func (l *byteLimitWriter) Write(p []byte) (int, error) {
	if l.limit < 0 || l.n >= l.limit || int64(len(p)) > l.limit-l.n {
		return 0, errTooLarge
	}
	n, err := l.w.Write(p)
	l.n += int64(n)
	return n, err
}

// writeLimitedTemp streams into a new temporary file and stops once limit bytes
// would be exceeded. On failure the temporary file is removed.
func writeLimitedTemp(limit int64, write func(io.Writer) (int64, error)) (path string, size int64, err error) {
	f, err := os.CreateTemp("", "ipa-*")
	if err != nil {
		return "", 0, err
	}
	path = f.Name()
	closed := false
	defer func() {
		if !closed {
			_ = f.Close()
		}
		if err != nil {
			_ = os.Remove(path)
			path = ""
			size = 0
		}
	}()
	n, werr := write(&byteLimitWriter{w: f, limit: limit})
	if werr != nil {
		if errors.Is(werr, errTooLarge) {
			err = errTooLarge
			return
		}
		err = werr
		return
	}
	if n > limit {
		err = errTooLarge
		return
	}
	if cerr := f.Close(); cerr != nil {
		closed = true
		err = cerr
		return
	}
	closed = true
	size = n
	return path, size, nil
}

func CreateTimeoutContext(d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), d)
}
