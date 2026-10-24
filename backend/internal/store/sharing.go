package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/a-chris/trackete/backend/internal/model"
)

// InviteTTL bounds how long an invitation link stays usable. Owners can resend
// to extend it.
const InviteTTL = 7 * 24 * time.Hour

// ErrAlreadyMember is returned when an address is invited to a tracker it
// already belongs to.
var ErrAlreadyMember = errors.New("already a member")

// normalizeEmail keeps invitation addresses comparable with users.email, which
// is stored lowercased.
func normalizeEmail(email string) string { return strings.ToLower(strings.TrimSpace(email)) }

// TrackerRole returns the accepted role of userID on trackerID, or "" when the
// user has no membership.
func (s *Store) TrackerRole(ctx context.Context, userID, trackerID string) (string, error) {
	var role string
	err := s.db.QueryRowContext(ctx,
		`SELECT role FROM tracker_members WHERE tracker_id = ? AND user_id = ?`, trackerID, userID,
	).Scan(&role)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	return role, nil
}

// AddMember grants an accepted membership. Re-adding an existing member is a
// no-op, so accepting the same invitation twice cannot fail on the primary key.
func (s *Store) AddMember(ctx context.Context, trackerID, userID, role string) error {
	if role != model.RoleOwner && role != model.RoleMember {
		role = model.RoleMember
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO tracker_members (tracker_id, user_id, role, created_at) VALUES (?, ?, ?, ?)
		 ON CONFLICT (tracker_id, user_id) DO NOTHING`,
		trackerID, userID, role, formatTime(now()),
	)
	return err
}

func (s *Store) ListMembers(ctx context.Context, trackerID string) ([]model.TrackerMember, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT m.user_id, u.email, u.name, m.role, m.created_at
		 FROM tracker_members m JOIN users u ON u.id = m.user_id
		 WHERE m.tracker_id = ?
		 ORDER BY CASE m.role WHEN 'owner' THEN 0 ELSE 1 END, m.created_at ASC, u.email ASC`,
		trackerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]model.TrackerMember, 0, 4)
	for rows.Next() {
		var (
			m        model.TrackerMember
			joinedAt string
		)
		if err := rows.Scan(&m.UserID, &m.Email, &m.Name, &m.Role, &joinedAt); err != nil {
			return nil, err
		}
		m.JoinedAt = parseTime(joinedAt)
		out = append(out, m)
	}
	return out, rows.Err()
}

// RemoveMember drops a member's access and retires their invitation rows. The
// owner's own membership is never removable, so a tracker always keeps one.
func (s *Store) RemoveMember(ctx context.Context, trackerID, memberID string) error {
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM tracker_members WHERE tracker_id = ? AND user_id = ? AND role = 'member'`,
		trackerID, memberID,
	)
	if err != nil {
		return err
	}
	if err := expectOne(res); err != nil {
		return err
	}
	// Keep the invitation as history so the owner can offer the address again.
	_, err = s.db.ExecContext(ctx,
		`UPDATE invitations SET status = ?, updated_at = ?, settled_at = ?
		 WHERE tracker_id = ? AND email = (SELECT email FROM users WHERE id = ?)
		   AND status IN ('pending', 'delivered', 'accepted')`,
		model.InviteRemoved, formatTime(now()), formatTime(now()), trackerID, memberID,
	)
	return err
}

// CancelInvitation withdraws an invitation that has not been accepted. It only
// matches an invitation that is still open, so a repeated cancel cannot revive
// or re-settle a terminal row.
func (s *Store) CancelInvitation(ctx context.Context, trackerID, inviteID string) (*model.Invitation, error) {
	ts := formatTime(now())
	res, err := s.db.ExecContext(ctx,
		`UPDATE invitations SET status = ?, updated_at = ?, settled_at = ?
		 WHERE id = ? AND tracker_id = ? AND status IN ('pending', 'delivered')`,
		model.InviteRemoved, ts, ts, inviteID, trackerID,
	)
	if err != nil {
		return nil, err
	}
	if err := expectOne(res); err != nil {
		return nil, err
	}
	return s.InvitationByID(ctx, inviteID)
}

const invitationColumns = `i.id, i.tracker_id, t.name, u.name, i.email, i.status, i.created_at, i.expires_at`

const invitationJoins = `FROM invitations i
	JOIN trackers t ON t.id = i.tracker_id
	JOIN users u ON u.id = i.invited_by`

func scanInvitation(scanner interface{ Scan(...any) error }) (*model.Invitation, error) {
	var (
		inv                  model.Invitation
		createdAt, expiresAt string
	)
	if err := scanner.Scan(
		&inv.ID, &inv.TrackerID, &inv.TrackerName, &inv.InviterName, &inv.Email, &inv.Status,
		&createdAt, &expiresAt,
	); err != nil {
		return nil, err
	}
	inv.CreatedAt = parseTime(createdAt)
	inv.ExpiresAt = parseTime(expiresAt)
	return &inv, nil
}

// CreateInvitation records an invitation and returns it together with whether a
// fresh row was created. Inviting an address that already has a live invitation
// re-issues that one (extending its expiry) instead of piling up duplicates.
func (s *Store) CreateInvitation(ctx context.Context, inviterID, trackerID, email string) (*model.Invitation, bool, error) {
	email = normalizeEmail(email)

	var member int
	if err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(1) FROM tracker_members m JOIN users u ON u.id = m.user_id
		 WHERE m.tracker_id = ? AND u.email = ?`, trackerID, email,
	).Scan(&member); err != nil {
		return nil, false, err
	}
	if member > 0 {
		return nil, false, ErrAlreadyMember
	}

	ts := now()
	expires := ts.Add(InviteTTL)

	var liveID string
	err := s.db.QueryRowContext(ctx,
		`SELECT id FROM invitations WHERE tracker_id = ? AND email = ? AND status IN ('pending', 'delivered')`,
		trackerID, email,
	).Scan(&liveID)
	switch {
	case err == nil:
		if _, err := s.db.ExecContext(ctx,
			`UPDATE invitations SET invited_by = ?, expires_at = ?, updated_at = ? WHERE id = ?`,
			inviterID, formatTime(expires), formatTime(ts), liveID,
		); err != nil {
			return nil, false, err
		}
		inv, err := s.InvitationByID(ctx, liveID)
		return inv, false, err
	case !errors.Is(err, sql.ErrNoRows):
		return nil, false, err
	}

	id := newID()
	if _, err := s.db.ExecContext(ctx,
		`INSERT INTO invitations (id, tracker_id, invited_by, email, status, created_at, updated_at, expires_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		id, trackerID, inviterID, email, model.InvitePending, formatTime(ts), formatTime(ts), formatTime(expires),
	); err != nil {
		return nil, false, err
	}
	inv, err := s.InvitationByID(ctx, id)
	return inv, true, err
}

func (s *Store) InvitationByID(ctx context.Context, id string) (*model.Invitation, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT `+invitationColumns+` `+invitationJoins+` WHERE i.id = ?`, id,
	)
	inv, err := scanInvitation(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return inv, nil
}

// RefreshInvitationExpiry extends a live invitation's expiry by InviteTTL from
// now, so resending a lapsed invitation emails a usable link. Settled
// invitations are left alone.
func (s *Store) RefreshInvitationExpiry(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE invitations SET expires_at = ?, updated_at = ?
		 WHERE id = ? AND status IN ('pending', 'delivered')`,
		formatTime(now().Add(InviteTTL)), formatTime(now()), id,
	)
	if err != nil {
		return err
	}
	return expectOne(res)
}

func (s *Store) SetInvitationStatus(ctx context.Context, id, status string) error {
	settled := any(nil)
	switch status {
	case model.InviteAccepted, model.InviteRejected, model.InviteRemoved:
		settled = formatTime(now())
	}
	res, err := s.db.ExecContext(ctx,
		`UPDATE invitations SET status = ?, updated_at = ?, settled_at = ? WHERE id = ?`,
		status, formatTime(now()), settled, id,
	)
	if err != nil {
		return err
	}
	return expectOne(res)
}

func (s *Store) ListInvitations(ctx context.Context, trackerID string) ([]model.Invitation, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+invitationColumns+` `+invitationJoins+`
		 WHERE i.tracker_id = ? ORDER BY i.created_at DESC`,
		trackerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]model.Invitation, 0, 8)
	for rows.Next() {
		inv, err := scanInvitation(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *inv)
	}
	return out, rows.Err()
}

// SharedWithEmails returns, for each tracker ID, the addresses the tracker is
// shared with: accepted members other than the owner, plus invitations that are
// still open. Trackers shared with nobody are absent from the map.
func (s *Store) SharedWithEmails(ctx context.Context, trackerIDs []string) (map[string][]string, error) {
	out := make(map[string][]string, len(trackerIDs))
	if len(trackerIDs) == 0 {
		return out, nil
	}
	args := make([]any, len(trackerIDs))
	for i, id := range trackerIDs {
		args[i] = id
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(trackerIDs)), ",")

	add := func(trackerID, email string) {
		for _, seen := range out[trackerID] {
			if seen == email {
				return
			}
		}
		out[trackerID] = append(out[trackerID], email)
	}

	memberRows, err := s.db.QueryContext(ctx,
		`SELECT m.tracker_id, u.email FROM tracker_members m JOIN users u ON u.id = m.user_id
		 WHERE m.role = ? AND m.tracker_id IN (`+placeholders+`) ORDER BY u.email`,
		append([]any{model.RoleMember}, args...)...)
	if err != nil {
		return nil, err
	}
	defer memberRows.Close()
	for memberRows.Next() {
		var trackerID, email string
		if err := memberRows.Scan(&trackerID, &email); err != nil {
			return nil, err
		}
		add(trackerID, email)
	}
	if err := memberRows.Err(); err != nil {
		return nil, err
	}

	inviteRows, err := s.db.QueryContext(ctx,
		`SELECT tracker_id, email FROM invitations
		 WHERE status IN ('pending', 'delivered') AND tracker_id IN (`+placeholders+`) ORDER BY email`,
		args...)
	if err != nil {
		return nil, err
	}
	defer inviteRows.Close()
	for inviteRows.Next() {
		var trackerID, email string
		if err := inviteRows.Scan(&trackerID, &email); err != nil {
			return nil, err
		}
		add(trackerID, email)
	}
	return out, inviteRows.Err()
}

// InviteSuggestions lists addresses the user invited before, newest first, so
// the share field can autocomplete instead of making them retype an address.
func (s *Store) InviteSuggestions(ctx context.Context, userID string, limit int) ([]model.InviteSuggestion, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT i.email, COALESCE(MAX(u.name), ''), MAX(i.created_at)
		 FROM invitations i LEFT JOIN users u ON u.email = i.email
		 WHERE i.invited_by = ?
		 GROUP BY i.email
		 ORDER BY 3 DESC
		 LIMIT ?`,
		userID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]model.InviteSuggestion, 0, limit)
	for rows.Next() {
		var (
			suggestion model.InviteSuggestion
			createdAt  string
		)
		if err := rows.Scan(&suggestion.Email, &suggestion.Name, &createdAt); err != nil {
			return nil, err
		}
		out = append(out, suggestion)
	}
	return out, rows.Err()
}

// LastInviteSentAt isn't tracked separately: the invitation row's updated_at is
// the last delivery attempt, which resend refreshes along with the expiry.
