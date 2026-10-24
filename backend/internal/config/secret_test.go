package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveSecretCreatesSecretFile(t *testing.T) {
	t.Setenv("JWT_SECRET_FILE", "")
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "nested", "trackete.db")

	source, err := resolveSecret(dbPath)
	if err != nil {
		t.Fatalf("resolveSecret: %v", err)
	}
	if !source.created {
		t.Fatal("expected a freshly created secret")
	}
	if len(source.value) != 64 {
		t.Fatalf("secret = %q, want 64 hex characters", source.value)
	}

	info, err := os.Stat(source.path)
	if err != nil {
		t.Fatalf("stat secret: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("secret file mode = %o, want 600", perm)
	}
}

func TestResolveSecretReusesExistingFile(t *testing.T) {
	t.Setenv("JWT_SECRET_FILE", "")
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "trackete.db")

	first, err := resolveSecret(dbPath)
	if err != nil {
		t.Fatalf("first resolveSecret: %v", err)
	}

	// A second boot must reuse the stored secret, not replace it.
	second, err := resolveSecret(dbPath)
	if err != nil {
		t.Fatalf("second resolveSecret: %v", err)
	}
	if second.created {
		t.Fatal("second boot reported creating a new secret")
	}
	if second.value != first.value {
		t.Fatalf("secret changed between boots: %q != %q", second.value, first.value)
	}
	if second.path != first.path {
		t.Fatalf("secret path changed: %q != %q", second.path, first.path)
	}
}

func TestResolveSecretKeepsHandWrittenSecret(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".jwt-secret")
	if err := os.WriteFile(path, []byte("  custom-secret-value\n"), 0o600); err != nil {
		t.Fatalf("write secret: %v", err)
	}
	t.Setenv("JWT_SECRET_FILE", path)

	source, err := resolveSecret(filepath.Join(dir, "trackete.db"))
	if err != nil {
		t.Fatalf("resolveSecret: %v", err)
	}
	if source.value != "custom-secret-value" {
		t.Fatalf("secret = %q, want the file content with surrounding space trimmed", source.value)
	}
	if source.created {
		t.Fatal("existing secret reported as created")
	}
}

func TestResolveSecretFailsOnEmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".jwt-secret")
	if err := os.WriteFile(path, []byte("   \n"), 0o600); err != nil {
		t.Fatalf("write secret: %v", err)
	}
	t.Setenv("JWT_SECRET_FILE", path)

	source, err := resolveSecret(filepath.Join(dir, "trackete.db"))
	if err == nil {
		t.Fatalf("resolveSecret returned %q, want an error for an empty file", source.value)
	}
	if !strings.Contains(err.Error(), "is empty") {
		t.Fatalf("error = %v, want it to point at the empty file", err)
	}
}

func TestResolveSecretFailsWhenItCannotWrite(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })
	t.Setenv("JWT_SECRET_FILE", "")

	if _, err := resolveSecret(filepath.Join(dir, "trackete.db")); err == nil {
		t.Fatal("resolveSecret succeeded on a read-only directory, want a fatal error")
	}
}

func TestLoadUsesEnvSecretAndSkipsFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("JWT_SECRET", "from-the-environment")
	t.Setenv("JWT_SECRET_FILE", "")
	t.Setenv("DB_PATH", filepath.Join(dir, "trackete.db"))

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.JWTSecret != "from-the-environment" {
		t.Fatalf("JWTSecret = %q, want the environment value", cfg.JWTSecret)
	}
	if _, err := os.Stat(filepath.Join(dir, secretFileName)); !os.IsNotExist(err) {
		t.Fatalf("secret file was written even though JWT_SECRET is set (stat err = %v)", err)
	}
}

func TestLoadStoresAndReusesSecretForDBPath(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("JWT_SECRET", "")
	t.Setenv("JWT_SECRET_FILE", "")
	t.Setenv("DB_PATH", filepath.Join(dir, "trackete.db"))

	first, err := Load()
	if err != nil {
		t.Fatalf("first Load: %v", err)
	}
	second, err := Load()
	if err != nil {
		t.Fatalf("second Load: %v", err)
	}
	if first.JWTSecret == "" || first.JWTSecret != second.JWTSecret {
		t.Fatalf("secret not stable across boots: %q then %q", first.JWTSecret, second.JWTSecret)
	}
}
