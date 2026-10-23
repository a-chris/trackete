package config

import (
	"crypto/rand"
	"encoding/hex"
	"log"
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
	Version           string
}

func Load() Config {
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
			if origin = strings.TrimSpace(origin); origin != "" {
				c.CORSOrigins = append(c.CORSOrigins, origin)
			}
		}
	}

	if c.JWTSecret == "" {
		c.JWTSecret = randomSecret()
		log.Print("WARN: JWT_SECRET is not set, using an ephemeral secret. " +
			"Everyone will be logged out when the container restarts. " +
			"Set JWT_SECRET to keep sessions across restarts.")
	}

	return c
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

func randomSecret() string {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		log.Fatalf("generate secret: %v", err)
	}
	return hex.EncodeToString(buf)
}
