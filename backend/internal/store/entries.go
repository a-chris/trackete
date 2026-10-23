package store

import (
	"context"
	"database/sql"
	"errors"

	"github.com/a-chris/trackete/backend/internal/model"
)

// UpsertEntry stores exactly one value per tracker and day. Logging twice on
// the same day overwrites the previous value.
func (s *Store) UpsertEntry(ctx context.Context, userID, trackerID, day string, value float64, note string) (*model.Entry, error) {
	ts := formatTime(now())
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO entries (tracker_id, user_id, day, value, note, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT (tracker_id, day) DO UPDATE SET
		   value = excluded.value,
		   note = excluded.note,
		   user_id = excluded.user_id,
		   updated_at = excluded.updated_at`,
		trackerID, userID, day, value, note, ts, ts,
	)
	if err != nil {
		return nil, err
	}
	return &model.Entry{
		TrackerID: trackerID,
		Day:       day,
		Value:     value,
		Note:      note,
		UpdatedAt: parseTime(ts),
	}, nil
}

func (s *Store) ListEntries(ctx context.Context, trackerID, from, to string) ([]model.Entry, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT tracker_id, day, value, note, updated_at FROM entries
		 WHERE tracker_id = ? AND day >= ? AND day <= ? ORDER BY day ASC`,
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
		if err := rows.Scan(&e.TrackerID, &e.Day, &e.Value, &e.Note, &updatedAt); err != nil {
			return nil, err
		}
		e.UpdatedAt = parseTime(updatedAt)
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *Store) DeleteEntry(ctx context.Context, trackerID, day string) error {
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM entries WHERE tracker_id = ? AND day = ?`, trackerID, day,
	)
	if err != nil {
		return err
	}
	if err := expectOne(res); err != nil {
		return err
	}
	return nil
}

// EntriesBetween returns every entry of a user in a date range, grouped by
// tracker. Used by the overview screen so it can render all trackers in a
// single round-trip.
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
		e.UpdatedAt = parseTime(updatedAt)
		out[e.TrackerID] = append(out[e.TrackerID], e)
	}
	return out, rows.Err()
}

func (s *Store) EarliestEntryDay(ctx context.Context, userID, trackerID string) (string, error) {
	var day sql.NullString
	err := s.db.QueryRowContext(ctx,
		`SELECT MIN(day) FROM entries WHERE user_id = ? AND tracker_id = ?`, userID, trackerID,
	).Scan(&day)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	return day.String, nil
}
