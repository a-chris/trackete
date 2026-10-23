package store

import (
	"embed"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

type migration struct {
	version int
	name    string
	sql     string
}

func loadMigrations() ([]migration, error) {
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	out := make([]migration, 0, len(names))
	for _, name := range names {
		prefix, _, ok := strings.Cut(name, "_")
		if !ok {
			return nil, fmt.Errorf("migration %q must start with a numeric prefix", name)
		}
		version, err := strconv.Atoi(prefix)
		if err != nil {
			return nil, fmt.Errorf("migration %q: invalid version: %w", name, err)
		}
		body, err := migrationsFS.ReadFile("migrations/" + name)
		if err != nil {
			return nil, err
		}
		out = append(out, migration{version: version, name: name, sql: string(body)})
	}
	return out, nil
}

func (s *Store) migrate() error {
	if _, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		applied_at TEXT NOT NULL
	)`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	migrations, err := loadMigrations()
	if err != nil {
		return err
	}

	for _, m := range migrations {
		var exists int
		err := s.db.QueryRow(`SELECT COUNT(1) FROM schema_migrations WHERE version = ?`, m.version).Scan(&exists)
		if err != nil {
			return fmt.Errorf("check migration %s: %w", m.name, err)
		}
		if exists > 0 {
			continue
		}

		tx, err := s.db.Begin()
		if err != nil {
			return err
		}
		for _, stmt := range splitStatements(m.sql) {
			if _, err := tx.Exec(stmt); err != nil {
				tx.Rollback()
				return fmt.Errorf("apply migration %s: %w", m.name, err)
			}
		}
		if _, err := tx.Exec(
			`INSERT INTO schema_migrations (version, name, applied_at) VALUES (?, ?, ?)`,
			m.version, m.name, formatTime(now()),
		); err != nil {
			tx.Rollback()
			return fmt.Errorf("record migration %s: %w", m.name, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", m.name, err)
		}
	}
	return nil
}

// splitStatements is only used for our own migration files, which contain plain
// DDL without semicolons inside string literals or triggers.
func splitStatements(sqlText string) []string {
	parts := strings.Split(sqlText, ";")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
