package config

import (
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Config holds every runtime knob. Everything is optional; the defaults are
// meant to make `docker run` work with zero configuration.
type Config struct {
	Port              string
	DBPath            string
	JWTSecret         string
	TokenTTL          time.Duration
	AllowRegistration bool
	StaticDir         string
	CORSOrigins       []string
	TrustedProxies    []string
	Version           string

	// AppURL is the externally reachable origin used in invitation links.
	AppURL string
	// Mail settings pick how invitation emails leave the process: smtp, a local
	// file that accumulates the messages, or the server log.
	MailTransport string
	SMTPHost      string
	SMTPPort      string
	SMTPUsername  string
	SMTPPassword  string
	MailFrom      string
	MailFile      string
}

func Load() (Config, error) {
	c := Config{
		Port:              env("PORT", "8080"),
		DBPath:            env("DB_PATH", "./data/trackete.db"),
		JWTSecret:         strings.TrimSpace(os.Getenv("JWT_SECRET")),
		TokenTTL:          30 * 24 * time.Hour,
		AllowRegistration: envBool("ALLOW_REGISTRATION", true),
		StaticDir:         strings.TrimSpace(os.Getenv("STATIC_DIR")),
		Version:           env("APP_VERSION", "dev"),
		AppURL:            strings.TrimRight(env("APP_URL", "http://localhost:8080"), "/"),
		MailTransport:     strings.ToLower(env("MAIL_TRANSPORT", "")),
		SMTPHost:          env("SMTP_HOST", ""),
		SMTPPort:          env("SMTP_PORT", "587"),
		SMTPUsername:      env("SMTP_USERNAME", ""),
		SMTPPassword:      os.Getenv("SMTP_PASSWORD"),
		MailFrom:          env("MAIL_FROM", "Trackete <no-reply@trackete.local>"),
		MailFile:          env("MAIL_FILE_PATH", ""),
	}

	// Invitation mail must go somewhere. An SMTP host means real delivery; a
	// file next to the database persists the messages for development, so a
	// fresh install still produces working invitation links.
	if c.MailFile == "" {
		if dir := filepath.Dir(c.DBPath); dir != "" && dir != "." {
			c.MailFile = filepath.Join(dir, "outbox.log")
		}
	}
	switch c.MailTransport {
	case "", "smtp", "file", "log":
	default:
		log.Printf("WARN: unknown MAIL_TRANSPORT %q, using the local file fallback", c.MailTransport)
		c.MailTransport = "file"
	}
	if c.MailTransport == "" {
		if c.SMTPHost != "" {
			c.MailTransport = "smtp"
		} else if c.MailFile != "" {
			c.MailTransport = "file"
		} else {
			c.MailTransport = "log"
		}
	}
	if c.MailTransport == "smtp" && c.SMTPHost == "" {
		log.Printf("WARN: MAIL_TRANSPORT=smtp without SMTP_HOST: every invitation stays pending")
	}

	if raw := env("TOKEN_TTL", ""); raw != "" {
		if d, err := time.ParseDuration(raw); err == nil && d > time.Minute {
			c.TokenTTL = d
		} else {
			log.Printf("WARN: invalid TOKEN_TTL %q, using %s", raw, c.TokenTTL)
		}
	}

	if raw := env("CORS_ORIGINS", ""); raw != "" {
		for _, origin := range strings.Split(raw, ",") {
			if origin = strings.TrimSpace(origin); origin == "" {
				continue
			}
			// A wildcard would reflect every Origin while still allowing
			// credentials, so it is ignored: list the origins explicitly.
			if origin == "*" {
				log.Print("WARN: CORS_ORIGINS contains \"*\", ignoring it. " +
					"Cookies are sent with credentials, so every origin must be listed explicitly.")
				continue
			}
			c.CORSOrigins = append(c.CORSOrigins, origin)
		}
	}

	if raw := env("TRUSTED_PROXIES", ""); raw != "" {
		for _, proxy := range strings.Split(raw, ",") {
			if proxy = strings.TrimSpace(proxy); proxy == "" {
				continue
			}
			if _, _, err := net.ParseCIDR(proxy); err != nil {
				if net.ParseIP(proxy) == nil {
					log.Printf("WARN: invalid TRUSTED_PROXIES entry %q, ignoring it (use an IP or CIDR)", proxy)
					continue
				}
			}
			c.TrustedProxies = append(c.TrustedProxies, proxy)
		}
	}

	if c.JWTSecret == "" {
		secret, err := resolveSecret(c.DBPath)
		if err != nil {
			return Config{}, fmt.Errorf("session secret: %w (fix the file, or set JWT_SECRET to provide one explicitly)", err)
		}
		c.JWTSecret = secret.value
	}

	return c, nil
}

func env(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	v, err := strconv.ParseBool(raw)
	if err != nil {
		return fallback
	}
	return v
}
