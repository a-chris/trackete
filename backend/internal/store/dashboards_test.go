package store

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
	"time"
)

// migrate() must repair databases where 0002 left users with dashboards but no
// flagged default: 0003 promotes exactly one row per such user.
func TestDefaultDashboardBackfillMigration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "backfill.db")

	db, err := sql.Open("sqlite", fmt.Sprintf(
		"file:%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)", path,
	))
	if err != nil {
		t.Fatalf("open raw sqlite: %v", err)
	}

	initSQL, err := migrationsFS.ReadFile("migrations/0001_init.sql")
	if err != nil {
		t.Fatalf("read 0001 migration: %v", err)
	}
	for _, stmt := range splitStatements(string(initSQL)) {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("apply 0001 by hand: %v", err)
		}
	}

	// Pretend 0001 is the only migration that has ever run.
	if _, err := db.Exec(`CREATE TABLE schema_migrations (
		version INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		applied_at TEXT NOT NULL
	)`); err != nil {
		t.Fatalf("create schema_migrations: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO schema_migrations (version, name, applied_at) VALUES (1, '0001_init.sql', ?)`,
		formatTime(now()),
	); err != nil {
		t.Fatalf("record 0001: %v", err)
	}

	ts := formatTime(time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC))
	if _, err := db.Exec(
		`INSERT INTO users (id, email, name, password_hash, created_at, updated_at)
		 VALUES ('u1', 'legacy@example.com', 'Legacy', 'x', ?, ?)`, ts, ts,
	); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	// Two dashboards, neither flagged: the state 0002 allowed to persist.
	for i, id := range []string{"d-old", "d-new"} {
		created := formatTime(time.Date(2026, 1, 3+i, 0, 0, 0, 0, time.UTC))
		if _, err := db.Exec(
			`INSERT INTO dashboards (id, user_id, name, is_default, created_at, updated_at)
			 VALUES (?, 'u1', ?, 0, ?, ?)`, id, "Dashboard "+id, created, created,
		); err != nil {
			t.Fatalf("insert dashboard %s: %v", id, err)
		}
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close raw sqlite: %v", err)
	}

	// Opening the store applies 0002 and 0003 on top of the hand-built database.
	st, err := Open(path)
	if err != nil {
		t.Fatalf("open store (running migrations): %v", err)
	}
	defer st.Close()

	var (
		defaults  int
		defaultID string
	)
	if err := st.db.QueryRow(
		`SELECT COUNT(1) FROM dashboards WHERE user_id = 'u1' AND is_default = 1`,
	).Scan(&defaults); err != nil {
		t.Fatalf("count defaults: %v", err)
	}
	if defaults != 1 {
		t.Fatalf("expected exactly one backfilled default dashboard, got %d", defaults)
	}
	if err := st.db.QueryRow(
		`SELECT id FROM dashboards WHERE user_id = 'u1' AND is_default = 1`,
	).Scan(&defaultID); err != nil {
		t.Fatalf("select default dashboard: %v", err)
	}
	// The oldest dashboard wins, so sorting stays stable with ListDashboards.
	if defaultID != "d-old" {
		t.Fatalf("expected the oldest dashboard (d-old) promoted, got %q", defaultID)
	}

	// Both migrations are recorded, so a restart does not replay them.
	var applied int
	if err := st.db.QueryRow(
		`SELECT COUNT(1) FROM schema_migrations WHERE version IN (2, 3)`,
	).Scan(&applied); err != nil {
		t.Fatalf("count applied migrations: %v", err)
	}
	if applied != 2 {
		t.Fatalf("expected migrations 2 and 3 recorded, got %d rows", applied)
	}
}

// Eight concurrent creates for one user must all succeed and leave exactly one
// default dashboard, even though every insert starts with no dashboards.
func TestCreateDashboardConcurrentFirstInsert(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "concurrent.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()

	user, err := st.CreateUser(t.Context(), "concurrent@example.com", "Concurrent", "x")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	const workers = 8
	errs := make(chan error, workers)
	start := make(chan struct{})
	for i := 0; i < workers; i++ {
		go func(i int) {
			<-start
			_, err := st.CreateDashboard(t.Context(), user.ID, fmt.Sprintf("Dashboard %d", i))
			errs <- err
		}(i)
	}
	close(start)

	for i := 0; i < workers; i++ {
		if err := <-errs; err != nil {
			t.Fatalf("concurrent create failed: %v", err)
		}
	}

	var total, defaults int
	if err := st.db.QueryRow(
		`SELECT COUNT(1), COALESCE(SUM(is_default), 0) FROM dashboards WHERE user_id = ?`, user.ID,
	).Scan(&total, &defaults); err != nil {
		t.Fatalf("count dashboards: %v", err)
	}
	if total != workers {
		t.Fatalf("expected %d dashboards, got %d", workers, total)
	}
	if defaults != 1 {
		t.Fatalf("expected exactly one default dashboard, got %d", defaults)
	}
}
