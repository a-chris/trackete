package config

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// secretFileName is the file that holds the signing secret when neither
// JWT_SECRET nor JWT_SECRET_FILE is set. It lives next to the database, so it is
// inside the /data volume and is restored together with the data.
const secretFileName = ".jwt-secret"

type secretSource struct {
	value   string
	path    string
	created bool
}

// resolveSecret loads the secret that signs session cookies, creating it on the
// first run.
//
// Every failure is returned: a secret that cannot be persisted must stop the
// process instead of degrading to a throwaway value, because that would silently
// sign every user out on the next restart and quietly break sessions across
// restarts forever.
func resolveSecret(dbPath string) (secretSource, error) {
	path := strings.TrimSpace(os.Getenv("JWT_SECRET_FILE"))
	if path == "" {
		path = filepath.Join(filepath.Dir(dbPath), secretFileName)
	}

	secret, err := readSecret(path)
	switch {
	case err == nil:
		return secretSource{value: secret, path: path}, nil
	case !errors.Is(err, fs.ErrNotExist):
		return secretSource{}, err
	}

	return createSecret(path)
}

func readSecret(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	secret := strings.TrimSpace(string(raw))
	if secret == "" {
		return "", fmt.Errorf("%s is empty: delete it so a new secret is generated", path)
	}
	return secret, nil
}

func createSecret(path string) (secretSource, error) {
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return secretSource{}, fmt.Errorf("create directory for %s: %w", path, err)
		}
	}

	secret, err := randomSecret()
	if err != nil {
		return secretSource{}, err
	}

	// Create the file with a temporary name and hard-link it into place: the real
	// path either does not exist or already holds a complete secret, so a crash
	// cannot leave a half-written file behind, and a second process booting at the
	// same moment cannot install a different secret.
	tmp, err := os.CreateTemp(dir, secretFileName+"-*")
	if err != nil {
		return secretSource{}, fmt.Errorf("create secret file in %s: %w", dir, err)
	}
	defer os.Remove(tmp.Name())

	_, writeErr := tmp.WriteString(secret + "\n")
	if writeErr == nil {
		writeErr = tmp.Sync()
	}
	if closeErr := tmp.Close(); writeErr == nil {
		writeErr = closeErr
	}
	if writeErr != nil {
		return secretSource{}, fmt.Errorf("write %s: %w", tmp.Name(), writeErr)
	}

	if err := os.Link(tmp.Name(), path); err != nil {
		if errors.Is(err, fs.ErrExist) {
			// Another process created the secret first: use its value.
			existing, readErr := readSecret(path)
			if readErr != nil {
				return secretSource{}, readErr
			}
			return secretSource{value: existing, path: path}, nil
		}
		return secretSource{}, fmt.Errorf("install %s: %w", path, err)
	}

	log.Printf("generated a new session secret in %s", path)
	return secretSource{value: secret, path: path, created: true}, nil
}

// randomSecret returns 32 random bytes as hex, the recommended size for an
// HMAC-SHA256 key.
func randomSecret() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate secret: %w", err)
	}
	return hex.EncodeToString(buf), nil
}
