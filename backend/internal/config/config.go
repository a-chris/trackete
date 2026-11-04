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

	c.TrustedProxies = parseTrustedProxies(os.Getenv("TRUSTED_PROXIES"))
	if len(c.TrustedProxies) == 0 {
		log.Print("trusted proxies: none, so forwarded client IPs are ignored")
	} else {
		log.Printf("trusted proxies: %s", strings.Join(c.TrustedProxies, ", "))
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

// defaultTrustedProxies mirrors ActionDispatch::RemoteIp::TRUSTED_PROXIES in
// Rails: private addresses are hops, not clients. A self-hosted instance always
// sits behind one of them (Docker's bridge gateway, a Caddy on the same host),
// so forwarded headers work without configuration.
//
// Trusting these ranges means a client that can already reach the app from a
// private address could spoof X-Forwarded-For. That is the same trade every
// Rails deployment makes; list the proxy explicitly in TRUSTED_PROXIES to
// narrow it, or set it to "none" to trust nothing at all.
var defaultTrustedProxies = []string{
	"127.0.0.0/8",    // loopback IPv4
	"::1",            // loopback IPv6
	"fc00::/7",       // private IPv6
	"10.0.0.0/8",     // private IPv4
	"172.16.0.0/12",  // private IPv4
	"192.168.0.0/16", // private IPv4
	"169.254.0.0/16", // link-local IPv4
	"fe80::/10",      // link-local IPv6
}

// parseTrustedProxies resolves the TRUSTED_PROXIES value. Empty (unset, or set
// to an empty value by an env file) means the private-range defaults; "none"
// means trust no proxy, which leaves every client identified by its socket
// address.
func parseTrustedProxies(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return append([]string(nil), defaultTrustedProxies...)
	}
	if strings.EqualFold(raw, "none") {
		return nil
	}

	proxies := make([]string, 0, len(defaultTrustedProxies))
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
		proxies = append(proxies, proxy)
	}
	return proxies
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
