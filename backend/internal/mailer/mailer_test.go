package mailer

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewResolvesTransport(t *testing.T) {
	cases := []struct {
		name string
		cfg  Config
		want string
	}{
		{"explicit file", Config{Transport: "file"}, TransportFile},
		{"explicit log wins over host", Config{Transport: "log", Host: "smtp.example.com"}, TransportLog},
		{"smtp host implies smtp", Config{Host: "smtp.example.com"}, TransportSMTP},
		{"file path implies file", Config{FilePath: "/tmp/outbox.log"}, TransportFile},
		{"nothing configured logs", Config{}, TransportLog},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := New(tc.cfg).Transport(); got != tc.want {
				t.Fatalf("expected transport %q, got %q", tc.want, got)
			}
		})
	}
}

// The local file transport is the development fallback: it must create the
// directory, append rather than truncate, and keep the invitation link readable.
func TestFileTransportAppendsMessages(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "outbox.log")
	m := New(Config{Transport: TransportFile, FilePath: path, From: "Trackete <no-reply@trackete.local>"})

	msg := Invitation("friend@example.com", "Owner", "Beers", "http://localhost:8080/invite/token-value", time.Date(2026, time.February, 3, 4, 5, 0, 0, time.UTC))
	if err := m.Send(context.Background(), msg); err != nil {
		t.Fatalf("send: %v", err)
	}
	if err := m.Send(context.Background(), msg); err != nil {
		t.Fatalf("send again: %v", err)
	}

	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read outbox: %v", err)
	}
	text := string(body)
	for _, want := range []string{
		"To: friend@example.com",
		"Subject: Owner invited you to \"Beers\" on Trackete",
		"http://localhost:8080/invite/token-value",
		"3 February 2026 04:05 UTC",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("outbox is missing %q:\n%s", want, text)
		}
	}
	if got := strings.Count(text, "To: friend@example.com"); got != 2 {
		t.Fatalf("expected two appended messages, found %d", got)
	}
}

func TestLogTransportWritesNoFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "outbox.log")
	m := New(Config{Transport: TransportLog, FilePath: path})

	if err := m.Send(context.Background(), Message{To: "friend@example.com", Subject: "Hi", Body: "Body"}); err != nil {
		t.Fatalf("send: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("log transport must not write the outbox file, stat err = %v", err)
	}
}

func TestSendRejectsEmptyRecipient(t *testing.T) {
	m := New(Config{Transport: TransportLog})
	if err := m.Send(context.Background(), Message{Subject: "Hi"}); err == nil {
		t.Fatal("expected an error for a message without a recipient")
	}
}

// A misconfigured SMTP transport must fail loudly: the invitation stays pending
// so the owner can retry, instead of being silently dropped.
func TestSMTPTransportWithoutHostFails(t *testing.T) {
	m := New(Config{Transport: TransportSMTP})
	err := m.Send(context.Background(), Message{To: "friend@example.com", Subject: "Hi", Body: "Body"})
	if err == nil || !strings.Contains(err.Error(), "SMTP_HOST") {
		t.Fatalf("expected an SMTP_HOST configuration error, got %v", err)
	}
}

func TestInvitationFallsBackToGenericInviter(t *testing.T) {
	msg := Invitation("friend@example.com", "  ", "Beers", "http://localhost:8080/invite/t", time.Now())
	if !strings.Contains(msg.Body, "Someone invited you") {
		t.Fatalf("expected a generic inviter in the body, got %q", msg.Body)
	}
}
