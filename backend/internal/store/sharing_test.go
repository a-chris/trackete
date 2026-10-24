package store

import (
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/a-chris/trackete/backend/internal/model"
)

// 0004 has to move the entry key from (tracker_id, day) to
// (tracker_id, user_id, day) without dropping a single row, and every tracker
// that predates sharing needs its creator recorded as the owner member.
func TestTrackerSharingMigrationPreservesLegacyData(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sharing.db")

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
	if _, err := db.Exec(
		`INSERT INTO trackers (id, user_id, name, kind, created_at, updated_at)
		 VALUES ('t1', 'u1', 'Legacy tracker', 'numeric', ?, ?)`, ts, ts,
	); err != nil {
		t.Fatalf("insert tracker: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO entries (tracker_id, user_id, day, value, note, created_at, updated_at)
		 VALUES ('t1', 'u1', '2026-01-02', 4, 'kept', ?, ?)`, ts, ts,
	); err != nil {
		t.Fatalf("insert legacy entry: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close raw sqlite: %v", err)
	}

	st, err := Open(path)
	if err != nil {
		t.Fatalf("open store (running migrations): %v", err)
	}
	defer st.Close()

	// The owner membership is backfilled, so the migration does not hide the
	// tracker from the user who created it.
	tracker, err := st.GetTracker(t.Context(), "u1", "t1")
	if err != nil {
		t.Fatalf("get migrated tracker: %v", err)
	}
	if tracker.Role != model.RoleOwner {
		t.Fatalf("expected backfilled owner role, got %q", tracker.Role)
	}

	entries, err := st.ListEntries(t.Context(), "t1", "2026-01-01", "2026-01-31")
	if err != nil {
		t.Fatalf("list migrated entries: %v", err)
	}
	if len(entries) != 1 || entries[0].Value != 4 || entries[0].Note != "kept" || entries[0].UserID != "u1" {
		t.Fatalf("legacy entry not preserved: %+v", entries)
	}

	// Two members may log the same day on the same tracker.
	member, err := st.CreateUser(t.Context(), "member@example.com", "Member", "x")
	if err != nil {
		t.Fatalf("create member: %v", err)
	}
	if err := st.AddMember(t.Context(), "t1", member.ID, model.RoleMember); err != nil {
		t.Fatalf("add member: %v", err)
	}
	if _, err := st.UpsertEntry(t.Context(), member.ID, "t1", "2026-01-02", 7, ""); err != nil {
		t.Fatalf("member entry on an already logged day: %v", err)
	}
	entries, err = st.ListEntries(t.Context(), "t1", "2026-01-02", "2026-01-02")
	if err != nil {
		t.Fatalf("list entries: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected one entry per member on the same day, got %d", len(entries))
	}

	var applied int
	if err := st.db.QueryRow(`SELECT COUNT(1) FROM schema_migrations WHERE version = 4`).Scan(&applied); err != nil {
		t.Fatalf("count applied migrations: %v", err)
	}
	if applied != 1 {
		t.Fatalf("expected migration 4 recorded once, got %d", applied)
	}
}

// Access follows accepted membership: a member reads and logs, loses access the
// moment they are removed, and can only ever touch their own entries.
func TestMembershipControlsAccess(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "members.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()

	ctx := t.Context()
	owner, err := st.CreateUser(ctx, "owner@example.com", "Owner", "x")
	if err != nil {
		t.Fatalf("create owner: %v", err)
	}
	member, err := st.CreateUser(ctx, "member@example.com", "Member", "x")
	if err != nil {
		t.Fatalf("create member: %v", err)
	}
	tracker, err := st.CreateTracker(ctx, owner.ID, &model.Tracker{Name: "Shared", Kind: model.KindNumeric})
	if err != nil {
		t.Fatalf("create tracker: %v", err)
	}
	if tracker.Role != model.RoleOwner {
		t.Fatalf("creator must be the owner, got %q", tracker.Role)
	}

	// Before joining, the tracker is invisible.
	if _, err := st.GetTracker(ctx, member.ID, tracker.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("non-member read: expected ErrNotFound, got %v", err)
	}
	visible, err := st.ListTrackers(ctx, member.ID, true)
	if err != nil {
		t.Fatalf("list trackers: %v", err)
	}
	if len(visible) != 0 {
		t.Fatalf("non-member should see no trackers, got %d", len(visible))
	}

	if err := st.AddMember(ctx, tracker.ID, member.ID, model.RoleMember); err != nil {
		t.Fatalf("add member: %v", err)
	}
	asMember, err := st.GetTracker(ctx, member.ID, tracker.ID)
	if err != nil {
		t.Fatalf("member read: %v", err)
	}
	if asMember.Role != model.RoleMember {
		t.Fatalf("expected member role, got %q", asMember.Role)
	}
	visible, err = st.ListTrackers(ctx, member.ID, true)
	if err != nil {
		t.Fatalf("list trackers as member: %v", err)
	}
	if len(visible) != 1 || visible[0].ID != tracker.ID {
		t.Fatalf("member should see the shared tracker, got %+v", visible)
	}

	// Membership does not grant metadata edits: those stay keyed to the owner.
	asMember.Name = "Renamed by member"
	if _, err := st.UpdateTracker(ctx, member.ID, asMember); !errors.Is(err, ErrNotFound) {
		t.Fatalf("member update: expected ErrNotFound, got %v", err)
	}

	if _, err := st.UpsertEntry(ctx, owner.ID, tracker.ID, "2026-02-01", 3, ""); err != nil {
		t.Fatalf("owner entry: %v", err)
	}
	if _, err := st.UpsertEntry(ctx, member.ID, tracker.ID, "2026-02-01", 5, ""); err != nil {
		t.Fatalf("member entry: %v", err)
	}
	// The member cannot delete the owner's row for the same day.
	if err := st.DeleteEntry(ctx, member.ID, tracker.ID, "2026-02-02"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("delete unlogged day: expected ErrNotFound, got %v", err)
	}

	if err := st.RemoveMember(ctx, tracker.ID, member.ID); err != nil {
		t.Fatalf("remove member: %v", err)
	}
	if _, err := st.GetTracker(ctx, member.ID, tracker.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("removed member read: expected ErrNotFound, got %v", err)
	}
	// The owner's own membership is not removable.
	if err := st.RemoveMember(ctx, tracker.ID, owner.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("owner removal: expected ErrNotFound, got %v", err)
	}
	if _, err := st.GetTracker(ctx, owner.ID, tracker.ID); err != nil {
		t.Fatalf("owner keeps access: %v", err)
	}
}

func TestInvitationLifecycleAndSuggestions(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "invites.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()

	ctx := t.Context()
	owner, err := st.CreateUser(ctx, "owner@example.com", "Owner", "x")
	if err != nil {
		t.Fatalf("create owner: %v", err)
	}
	tracker, err := st.CreateTracker(ctx, owner.ID, &model.Tracker{Name: "Shared", Kind: model.KindNumeric})
	if err != nil {
		t.Fatalf("create tracker: %v", err)
	}

	inv, created, err := st.CreateInvitation(ctx, owner.ID, tracker.ID, " Friend@Example.com ")
	if err != nil {
		t.Fatalf("create invitation: %v", err)
	}
	if !created {
		t.Fatal("expected a new invitation row")
	}
	if inv.Email != "friend@example.com" || inv.Status != model.InvitePending {
		t.Fatalf("unexpected invitation: %+v", inv)
	}
	if inv.TrackerName != "Shared" || inv.InviterName != "Owner" {
		t.Fatalf("invitation should carry tracker and inviter names: %+v", inv)
	}

	// Inviting the same address again re-issues the live invitation.
	again, created, err := st.CreateInvitation(ctx, owner.ID, tracker.ID, "friend@example.com")
	if err != nil {
		t.Fatalf("re-invite: %v", err)
	}
	if created || again.ID != inv.ID {
		t.Fatalf("expected the live invitation reused, got %+v (created=%v)", again, created)
	}

	if err := st.SetInvitationStatus(ctx, inv.ID, model.InviteDelivered); err != nil {
		t.Fatalf("mark delivered: %v", err)
	}

	// A joined address cannot be invited again, and the suggestion list keeps
	// the address the owner already used.
	friend, err := st.CreateUser(ctx, "friend@example.com", "Friend", "x")
	if err != nil {
		t.Fatalf("create friend: %v", err)
	}
	if err := st.AddMember(ctx, tracker.ID, friend.ID, model.RoleMember); err != nil {
		t.Fatalf("add friend: %v", err)
	}
	if _, _, err := st.CreateInvitation(ctx, owner.ID, tracker.ID, "friend@example.com"); !errors.Is(err, ErrAlreadyMember) {
		t.Fatalf("invite existing member: expected ErrAlreadyMember, got %v", err)
	}
	if err := st.SetInvitationStatus(ctx, inv.ID, model.InviteAccepted); err != nil {
		t.Fatalf("mark accepted: %v", err)
	}
	if err := st.RemoveMember(ctx, tracker.ID, friend.ID); err != nil {
		t.Fatalf("remove friend: %v", err)
	}
	history, err := st.ListInvitations(ctx, tracker.ID)
	if err != nil {
		t.Fatalf("list invitations: %v", err)
	}
	if len(history) != 1 || history[0].Status != model.InviteRemoved {
		t.Fatalf("expected the invitation kept as removed history, got %+v", history)
	}

	suggestions, err := st.InviteSuggestions(ctx, owner.ID, 10)
	if err != nil {
		t.Fatalf("suggestions: %v", err)
	}
	if len(suggestions) != 1 || suggestions[0].Email != "friend@example.com" || suggestions[0].Name != "Friend" {
		t.Fatalf("unexpected suggestions: %+v", suggestions)
	}
}

// Resending relies on the refresh: a lapsed live invitation gets a fresh
// InviteTTL, while settled history is left alone.
func TestRefreshInvitationExpiry(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "refresh.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()

	ctx := t.Context()
	owner, err := st.CreateUser(ctx, "owner@example.com", "Owner", "x")
	if err != nil {
		t.Fatalf("create owner: %v", err)
	}
	tracker, err := st.CreateTracker(ctx, owner.ID, &model.Tracker{Name: "Shared", Kind: model.KindNumeric})
	if err != nil {
		t.Fatalf("create tracker: %v", err)
	}
	inv, _, err := st.CreateInvitation(ctx, owner.ID, tracker.ID, "friend@example.com")
	if err != nil {
		t.Fatalf("create invitation: %v", err)
	}

	// Simulate a lapsed invitation: the stored expiry is in the past.
	past := formatTime(now().Add(-time.Hour))
	if _, err := st.db.ExecContext(ctx, `UPDATE invitations SET expires_at = ? WHERE id = ?`, past, inv.ID); err != nil {
		t.Fatalf("expire invitation: %v", err)
	}

	if err := st.RefreshInvitationExpiry(ctx, inv.ID); err != nil {
		t.Fatalf("refresh expiry: %v", err)
	}
	refreshed, err := st.InvitationByID(ctx, inv.ID)
	if err != nil {
		t.Fatalf("read invitation: %v", err)
	}
	remaining := refreshed.ExpiresAt.Sub(now())
	if !refreshed.ExpiresAt.After(time.Time(parseTime(past))) || remaining < InviteTTL-time.Minute || remaining > InviteTTL+time.Minute {
		t.Fatalf("expected the expiry refreshed to about InviteTTL from now, got %v remaining", remaining)
	}

	// Settled invitations are not touched.
	if err := st.SetInvitationStatus(ctx, inv.ID, model.InviteAccepted); err != nil {
		t.Fatalf("mark accepted: %v", err)
	}
	if err := st.RefreshInvitationExpiry(ctx, inv.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("settled invitation refresh: expected ErrNotFound, got %v", err)
	}
}
