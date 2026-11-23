package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
)

func TestWriteLimitedTempRemovesOversizedFile(t *testing.T) {
	path, size, err := writeLimitedTemp(4, func(w io.Writer) (int64, error) {
		n, werr := io.Copy(w, bytes.NewReader([]byte("12345")))
		return n, werr
	})
	if !errors.Is(err, errTooLarge) {
		t.Fatalf("err = %v", err)
	}
	if path != "" || size != 0 {
		t.Fatalf("path = %q size = %d", path, size)
	}
}

func TestWriteLimitedTempKeepsFileUntilRemoved(t *testing.T) {
	path, size, err := writeLimitedTemp(8, func(w io.Writer) (int64, error) {
		n, werr := w.Write([]byte("ipa"))
		return int64(n), werr
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(path) })
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "ipa" || size != 3 {
		t.Fatalf("data = %q size = %d", data, size)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("temp file was left on disk")
	}
}

func TestRedactAPIHash(t *testing.T) {
	config = &Config{Token: "123:ABC", APIHash: "hash-secret-value"}
	t.Cleanup(func() { config = nil })
	got := redactSecrets("token 123:ABC hash hash-secret-value")
	if strings.Contains(got, "123:ABC") || strings.Contains(got, "hash-secret-value") {
		t.Fatalf("secret leaked: %s", got)
	}
}
