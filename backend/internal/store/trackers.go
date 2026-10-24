package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/a-chris/trackete/backend/internal/model"
)

func encodeGoal(goal *model.Goal) any {
	if goal == nil {
		return nil
	}
	body, err := json.Marshal(goal)
	if err != nil {
		return nil
	}
	return string(body)
}

func decodeGoal(raw sql.NullString) *model.Goal {
	if !raw.Valid || raw.String == "" {
		return nil
	}
	var g model.Goal
	if err := json.Unmarshal([]byte(raw.String), &g); err != nil {
		return nil
	}
	return &g
}

func (s *Store) ListTrackers(ctx context.Context, userID string, includeArchived bool) ([]model.Tracker, error) {
	query := `SELECT ` + trackerColumns + `
		FROM trackers t JOIN tracker_members m ON m.tracker_id = t.id
		WHERE m.user_id = ?`
	if !includeArchived {
		query += ` AND t.archived = 0`
	}
	query += ` ORDER BY t.archived ASC, t.created_at ASC`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]model.Tracker, 0, 8)
	for rows.Next() {
		t, err := scanTracker(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	return out, rows.Err()
}

func scanTracker(scanner interface{ Scan(...any) error }) (*model.Tracker, error) {
	var (
		t                    model.Tracker
		goal                 sql.NullString
		archived             int
		createdAt, updatedAt string
	)
	if err := scanner.Scan(
		&t.ID, &t.Name, &t.Description, &t.Icon, &t.Color, &t.Kind, &t.Unit,
		&goal, &archived, &t.Role, &createdAt, &updatedAt,
	); err != nil {
		return nil, err
	}
	t.Goal = decodeGoal(goal)
	t.Archived = archived == 1
	t.CreatedAt = parseTime(createdAt)
	t.UpdatedAt = parseTime(updatedAt)
	return &t, nil
}

// GetTracker returns a tracker the user is an accepted member of. Non-members
// get ErrNotFound, so an unshared tracker is indistinguishable from a missing
// one.
func (s *Store) GetTracker(ctx context.Context, userID, id string) (*model.Tracker, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT `+trackerColumns+`
		 FROM trackers t JOIN tracker_members m ON m.tracker_id = t.id
		 WHERE t.id = ? AND m.user_id = ?`, id, userID,
	)
	t, err := scanTracker(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return t, nil
}

func (s *Store) CreateTracker(ctx context.Context, userID string, t *model.Tracker) (*model.Tracker, error) {
	ts := formatTime(now())
	t.ID = newID()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO trackers (id, user_id, name, description, icon, color, kind, unit, goal, archived, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 0, ?, ?)`,
		t.ID, userID, t.Name, t.Description, t.Icon, t.Color, t.Kind, t.Unit, encodeGoal(t.Goal), ts, ts,
	); err != nil {
		return nil, err
	}

	// The creator is the owner and starts as an accepted member, which is what
	// every read path checks.
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO tracker_members (tracker_id, user_id, role, created_at) VALUES (?, ?, ?, ?)`,
		t.ID, userID, model.RoleOwner, ts,
	); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.GetTracker(ctx, userID, t.ID)
}

func (s *Store) UpdateTracker(ctx context.Context, userID string, t *model.Tracker) (*model.Tracker, error) {
	res, err := s.db.ExecContext(ctx,
		`UPDATE trackers SET name = ?, description = ?, icon = ?, color = ?, kind = ?, unit = ?, goal = ?, archived = ?, updated_at = ?
		 WHERE id = ? AND user_id = ?`,
		t.Name, t.Description, t.Icon, t.Color, t.Kind, t.Unit, encodeGoal(t.Goal), boolToInt(t.Archived), formatTime(now()),
		t.ID, userID,
	)
	if err != nil {
		return nil, err
	}
	if err := expectOne(res); err != nil {
		return nil, err
	}
	return s.GetTracker(ctx, userID, t.ID)
}

func (s *Store) DeleteTracker(ctx context.Context, userID, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM trackers WHERE id = ? AND user_id = ?`, id, userID)
	if err != nil {
		return err
	}
	return expectOne(res)
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

// trackerColumns is every column scanTracker reads. The membership role comes
// from the join, so callers always learn how they may act on the tracker.
const trackerColumns = `t.id, t.name, t.description, t.icon, t.color, t.kind, t.unit,
	t.goal, t.archived, m.role, t.created_at, t.updated_at`
