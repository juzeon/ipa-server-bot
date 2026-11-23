package main

import (
	"errors"
	"strings"
	"testing"
)

func TestRedactSecrets(t *testing.T) {
	config = &Config{
		Token: "123456:SECRETTOKEN",
		S3:    S3{SecretAccessKey: "supersecretvalue"},
	}
	t.Cleanup(func() { config = nil })

	got := redactSecrets("token 123456:SECRETTOKEN key supersecretvalue")
	if strings.Contains(got, "SECRETTOKEN") || strings.Contains(got, "supersecretvalue") {
		t.Fatalf("secrets leaked: %s", got)
	}
	if !strings.Contains(got, "[redacted]") {
		t.Fatalf("expected redaction, got %s", got)
	}
}

func TestHandlerFailureMessageHidesInternalErrors(t *testing.T) {
	config = &Config{Token: "123456:SECRETTOKEN"}
	t.Cleanup(func() { config = nil })

	reply, logDetail := handlerFailureMessage(errors.New("Get https://example/bot123456:SECRETTOKEN/file: refused"))
	if reply != "Something went wrong. Please try again." {
		t.Fatalf("reply = %q", reply)
	}
	if strings.Contains(reply, "SECRETTOKEN") || strings.Contains(logDetail, "SECRETTOKEN") {
		t.Fatalf("token leaked reply=%q log=%q", reply, logDetail)
	}
	if logDetail == "" {
		t.Fatal("expected a server log line")
	}

	reply, logDetail = handlerFailureMessage(userErr("Application not found. token 123456:SECRETTOKEN"))
	if strings.Contains(reply, "SECRETTOKEN") {
		t.Fatalf("user reply leaked token: %q", reply)
	}
	if !strings.Contains(reply, "Application not found.") {
		t.Fatalf("reply = %q", reply)
	}
	if logDetail != "" {
		t.Fatalf("user error should not be logged as internal: %q", logDetail)
	}
}
