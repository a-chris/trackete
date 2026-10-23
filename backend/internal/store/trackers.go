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
	query := `SELECT id, name, description, icon, color, kind, unit, goal, archived, created_at, updated_at
		FROM trackers WHERE user_id = ?`
	if !includeArchived {
		query += ` AND archived = 0`
	}
	query += ` ORDER BY archived ASC, created_at ASC`

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
		&goal, &archived, &createdAt, &updatedAt,
	); err != nil {
		return nil, err
	}
	t.Goal = decodeGoal(goal)
	t.Archived = archived == 1
	t.CreatedAt = parseTime(createdAt)
	t.UpdatedAt = parseTime(updatedAt)
	return &t, nil
}

func (s *Store) GetTracker(ctx context.Context, userID, id string) (*model.Tracker, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, name, description, icon, color, kind, unit, goal, archived, created_at, updated_at
		 FROM trackers WHERE id = ? AND user_id = ?`, id, userID,
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
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO trackers (id, user_id, name, description, icon, color, kind, unit, goal, archived, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 0, ?, ?)`,
		t.ID, userID, t.Name, t.Description, t.Icon, t.Color, t.Kind, t.Unit, encodeGoal(t.Goal), ts, ts,
	)
	if err != nil {
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
