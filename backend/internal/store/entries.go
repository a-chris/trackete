package store

import (
	"context"
	"database/sql"
	"errors"

	"github.com/a-chris/trackete/backend/internal/model"
)

// UpsertEntry stores exactly one value per tracker, member and day. Logging
// twice on the same day overwrites that member's previous value; other members'
// entries for the same day are untouched.
func (s *Store) UpsertEntry(ctx context.Context, userID, trackerID, day string, value float64, note string) (*model.Entry, error) {
	ts := formatTime(now())
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO entries (tracker_id, user_id, day, value, note, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT (tracker_id, user_id, day) DO UPDATE SET
		   value = excluded.value,
		   note = excluded.note,
		   updated_at = excluded.updated_at`,
		trackerID, userID, day, value, note, ts, ts,
	)
	if err != nil {
		return nil, err
	}
	return &model.Entry{
		TrackerID: trackerID,
		UserID:    userID,
		Day:       day,
		Value:     value,
		Note:      note,
		UpdatedAt: parseTime(ts),
	}, nil
}

// ListEntries returns every member's entries for a tracker in a date range,
// attributed to the member who logged them.
func (s *Store) ListEntries(ctx context.Context, trackerID, from, to string) ([]model.Entry, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT e.tracker_id, e.user_id, u.name, e.day, e.value, e.note, e.updated_at
		 FROM entries e JOIN users u ON u.id = e.user_id
		 WHERE e.tracker_id = ? AND e.day >= ? AND e.day <= ?
		 ORDER BY e.day ASC, u.name ASC`,
		trackerID, from, to,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]model.Entry, 0, 32)
	for rows.Next() {
		var (
			e         model.Entry
			updatedAt string
		)
		if err := rows.Scan(&e.TrackerID, &e.UserID, &e.UserName, &e.Day, &e.Value, &e.Note, &updatedAt); err != nil {
			return nil, err
		}
		e.UpdatedAt = parseTime(updatedAt)
		out = append(out, e)
	}
	return out, rows.Err()
}

// DeleteEntry removes one member's entry. A member can only reach their own
// rows: another member's entry for the same day yields ErrNotFound.
func (s *Store) DeleteEntry(ctx context.Context, userID, trackerID, day string) error {
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM entries WHERE tracker_id = ? AND user_id = ? AND day = ?`, trackerID, userID, day,
	)
	if err != nil {
		return err
	}
	return expectOne(res)
}

// EntriesBetween returns every entry of a user in a date range, grouped by
// tracker. Used by the overview screen so it can render all trackers in a
// single round-trip; shared trackers contribute only the caller's own values.
func (s *Store) EntriesBetween(ctx context.Context, userID, from, to string) (map[string][]model.Entry, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT tracker_id, day, value, note, updated_at FROM entries
		 WHERE user_id = ? AND day >= ? AND day <= ? ORDER BY day ASC`,
		userID, from, to,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[string][]model.Entry)
	for rows.Next() {
		var (
			e         model.Entry
			updatedAt string
		)
		if err := rows.Scan(&e.TrackerID, &e.Day, &e.Value, &e.Note, &updatedAt); err != nil {
			return nil, err
		}
		e.UserID = userID
		e.UpdatedAt = parseTime(updatedAt)
		out[e.TrackerID] = append(out[e.TrackerID], e)
	}
	return out, rows.Err()
}

// EarliestEntryDay is the first day anyone logged on the tracker. It bounds the
// "all time" range, which covers the shared history rather than one member's.
func (s *Store) EarliestEntryDay(ctx context.Context, trackerID string) (string, error) {
	var day sql.NullString
	err := s.db.QueryRowContext(ctx,
		`SELECT MIN(day) FROM entries WHERE tracker_id = ?`, trackerID,
	).Scan(&day)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	return day.String, nil
}
