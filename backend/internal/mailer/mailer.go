// Package mailer delivers invitation email. Transports are chosen once from
// configuration so the API only ever has to ask "did this message go out?".
package mailer

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/mail"
	"net/smtp"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Transport names accepted by MAIL_TRANSPORT.
const (
	TransportSMTP = "smtp"
	TransportFile = "file"
	TransportLog  = "log"
)

// Config is the transport configuration. Empty fields fall back to a local,
// no-network transport so a fresh install still produces usable invitation
// links.
type Config struct {
	Transport string
	Host      string
	Port      string
	Username  string
	Password  string
	From      string
	FilePath  string
}

// Message is a plain-text email.
type Message struct {
	To      string
	Subject string
	Body    string
}

// Mailer sends messages through one fixed transport.
type Mailer struct {
	cfg       Config
	transport string
	now       func() time.Time
}

func New(cfg Config) *Mailer {
	m := &Mailer{cfg: cfg, transport: resolveTransport(cfg), now: time.Now}
	if strings.TrimSpace(m.cfg.From) == "" {
		m.cfg.From = "no-reply@trackete.local"
	}
	if strings.TrimSpace(m.cfg.Port) == "" {
		m.cfg.Port = "587"
	}
	return m
}

// resolveTransport honours an explicit choice and otherwise infers one: an SMTP
// host means real delivery, a file path means the local outbox, and anything
// else falls back to the server log.
func resolveTransport(cfg Config) string {
	switch choice := strings.ToLower(strings.TrimSpace(cfg.Transport)); choice {
	case TransportSMTP, TransportFile, TransportLog:
		return choice
	}
	switch {
	case strings.TrimSpace(cfg.Host) != "":
		return TransportSMTP
	case strings.TrimSpace(cfg.FilePath) != "":
		return TransportFile
	default:
		return TransportLog
	}
}

// Transport reports the transport in use, for startup logging and tests.
func (m *Mailer) Transport() string { return m.transport }

// Send hands the message to the transport. A nil error means the message was
// accepted by it; callers keep the invitation pending otherwise.
func (m *Mailer) Send(ctx context.Context, msg Message) error {
	if strings.TrimSpace(msg.To) == "" {
		return fmt.Errorf("mail: missing recipient")
	}
	switch m.transport {
	case TransportSMTP:
		return m.sendSMTP(ctx, msg)
	case TransportFile:
		return m.appendFile(msg)
	default:
		log.Printf("mail (log transport) to=%s subject=%q\n%s", msg.To, msg.Subject, msg.Body)
		return nil
	}
}

func (m *Mailer) render(msg Message) string {
	var b strings.Builder
	b.WriteString("To: " + msg.To + "\r\n")
	b.WriteString("From: " + m.cfg.From + "\r\n")
	b.WriteString("Subject: " + msg.Subject + "\r\n")
	b.WriteString("Date: " + m.now().UTC().Format(time.RFC1123Z) + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
	b.WriteString("\r\n")
	b.WriteString(strings.ReplaceAll(strings.TrimRight(msg.Body, "\n"), "\n", "\r\n"))
	b.WriteString("\r\n")
	return b.String()
}

func (m *Mailer) appendFile(msg Message) error {
	path := strings.TrimSpace(m.cfg.FilePath)
	if path == "" {
		return fmt.Errorf("mail: file transport without MAIL_FILE_PATH")
	}
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := file.WriteString(m.render(msg) + "---\r\n"); err != nil {
		return err
	}
	return file.Sync()
}

func (m *Mailer) sendSMTP(ctx context.Context, msg Message) error {
	host := strings.TrimSpace(m.cfg.Host)
	if host == "" {
		return fmt.Errorf("mail: smtp transport without SMTP_HOST")
	}

	dialer := &net.Dialer{Timeout: 10 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(host, m.cfg.Port))
	if err != nil {
		return fmt.Errorf("mail: dial %s: %w", host, err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("mail: smtp handshake: %w", err)
	}
	defer client.Close()

	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(&tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}); err != nil {
			return fmt.Errorf("mail: starttls: %w", err)
		}
	}
	if m.cfg.Username != "" {
		if ok, _ := client.Extension("AUTH"); ok {
			if err := client.Auth(smtp.PlainAuth("", m.cfg.Username, m.cfg.Password, host)); err != nil {
				return fmt.Errorf("mail: auth: %w", err)
			}
		}
	}
	if err := client.Mail(envelopeFrom(m.cfg.From)); err != nil {
		return fmt.Errorf("mail: envelope from: %w", err)
	}
	if err := client.Rcpt(msg.To); err != nil {
		return fmt.Errorf("mail: recipient %s: %w", msg.To, err)
	}
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("mail: data: %w", err)
	}
	if _, err := writer.Write([]byte(m.render(msg))); err != nil {
		return fmt.Errorf("mail: write body: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("mail: close body: %w", err)
	}
	return client.Quit()
}

// envelopeFrom strips the display name from a From header value.
func envelopeFrom(from string) string {
	if addr, err := mail.ParseAddress(from); err == nil {
		return addr.Address
	}
	return from
}

// Invitation renders the message that carries an acceptance link. The link
// opens a page that can accept or decline, so one URL is enough.
func Invitation(to, inviterName, trackerName, inviteURL string, expires time.Time) Message {
	if strings.TrimSpace(inviterName) == "" {
		inviterName = "Someone"
	}
	body := strings.Join([]string{
		fmt.Sprintf("%s invited you to share the tracker %q on Trackete.", inviterName, trackerName),
		"",
		"Open the invitation:",
		inviteURL,
		"",
		"From there you can accept and start logging, or decline.",
		fmt.Sprintf("The link stops working after %s.", expires.UTC().Format("2 January 2006 15:04 UTC")),
		"Ignoring this email leaves the invitation unanswered.",
	}, "\n")

	return Message{
		To:      to,
		Subject: fmt.Sprintf("%s invited you to %q on Trackete", inviterName, trackerName),
		Body:    body,
	}
}
