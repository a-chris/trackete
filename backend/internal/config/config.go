package config

import (
	"fmt"
	"log"
	"net"
	"os"
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
