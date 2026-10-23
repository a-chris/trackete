package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/a-chris/trackete/backend/internal/model"
)

const widgetColumns = `id, dashboard_id, COALESCE(tracker_id, ''), title, chart_type, config, pos_x, pos_y, width, height, created_at, updated_at`

func (s *Store) ensureDefaultDashboard(ctx context.Context, userID string) (*model.Dashboard, error) {
	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM dashboards WHERE user_id = ?`, userID).Scan(&count); err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, nil
	}
	return s.CreateDashboard(ctx, userID, "My dashboard")
}

// EnsureDefaultDashboard creates the first dashboard for a user, if any.
func (s *Store) EnsureDefaultDashboard(ctx context.Context, userID string) error {
	_, err := s.ensureDefaultDashboard(ctx, userID)
	return err
}

func (s *Store) ListDashboards(ctx context.Context, userID string) ([]model.Dashboard, error) {
	if err := s.EnsureDefaultDashboard(ctx, userID); err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, is_default, created_at, updated_at FROM dashboards
		 WHERE user_id = ? ORDER BY is_default DESC, created_at ASC`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]model.Dashboard, 0, 4)
	for rows.Next() {
		var (
			d         model.Dashboard
			isDefault int
			createdAt string
			updatedAt string
		)
		if err := rows.Scan(&d.ID, &d.Name, &isDefault, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		d.IsDefault = isDefault == 1
		d.CreatedAt = parseTime(createdAt)
		d.UpdatedAt = parseTime(updatedAt)
		out = append(out, d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range out {
		widgets, err := s.ListWidgets(ctx, out[i].ID)
		if err != nil {
			return nil, err
		}
		out[i].Widgets = widgets
	}
	return out, nil
}

func (s *Store) GetDashboard(ctx context.Context, userID, id string) (*model.Dashboard, error) {
	var (
		d         model.Dashboard
		isDefault int
		createdAt string
		updatedAt string
	)
	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, is_default, created_at, updated_at FROM dashboards WHERE id = ? AND user_id = ?`,
		id, userID,
	).Scan(&d.ID, &d.Name, &isDefault, &createdAt, &updatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	d.IsDefault = isDefault == 1
	d.CreatedAt = parseTime(createdAt)
	d.UpdatedAt = parseTime(updatedAt)
	widgets, err := s.ListWidgets(ctx, d.ID)
	if err != nil {
		return nil, err
	}
	d.Widgets = widgets
	return &d, nil
}

// CreateDashboard inserts a dashboard, making it the default when it is the
// user's first one. The decision is part of the INSERT itself, so there is no
// count-then-insert window in which two creates could both claim is_default = 1.
func (s *Store) CreateDashboard(ctx context.Context, userID, name string) (*model.Dashboard, error) {
	var lastErr error
	for attempt := 0; attempt < 4; attempt++ {
		id, err := s.insertDashboard(ctx, userID, name, false)
		if err == nil {
			return s.GetDashboard(ctx, userID, id)
		}
		lastErr = err

		switch {
		case isUniqueViolation(err):
			// A concurrent create already claimed the default; add this one as a
			// normal dashboard so the caller still succeeds.
			id, err = s.insertDashboard(ctx, userID, name, true)
			if err == nil {
				return s.GetDashboard(ctx, userID, id)
			}
			lastErr = err
			if !isLockedError(err) {
				return nil, err
			}
		case isLockedError(err):
			// Another writer briefly held SQLite. Retry the deciding INSERT rather
			// than forcing a non-default row, so a lock alone can never leave a
			// user without a default dashboard.
			time.Sleep(time.Duration(attempt+1) * 25 * time.Millisecond)
		default:
			return nil, err
		}
	}
	return nil, lastErr
}

// isRetryableDashboardInsert reports whether a failed dashboard insert is worth
// one retry as a non-default row: a unique violation means a concurrent create
// already claimed the default, and a locked database deserves a second attempt.
func isLockedError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "database is locked") || strings.Contains(msg, "database table is locked")
}

// insertDashboard writes one dashboard row. When forceNotDefault is false the
// default flag is computed in-statement from the user's existing dashboards.
func (s *Store) insertDashboard(ctx context.Context, userID, name string, forceNotDefault bool) (string, error) {
	ts := formatTime(now())
	id := newID()

	query := `INSERT INTO dashboards (id, user_id, name, is_default, created_at, updated_at)
	          VALUES (?, ?, ?, CASE WHEN EXISTS (SELECT 1 FROM dashboards WHERE user_id = ?) THEN 0 ELSE 1 END, ?, ?)`
	args := []any{id, userID, strings.TrimSpace(name), userID, ts, ts}
	if forceNotDefault {
		query = `INSERT INTO dashboards (id, user_id, name, is_default, created_at, updated_at)
		         VALUES (?, ?, ?, 0, ?, ?)`
		args = []any{id, userID, strings.TrimSpace(name), ts, ts}
	}

	if _, err := s.db.ExecContext(ctx, query, args...); err != nil {
		return "", err
	}
	return id, nil
}

func (s *Store) RenameDashboard(ctx context.Context, userID, id, name string) (*model.Dashboard, error) {
	res, err := s.db.ExecContext(ctx,
		`UPDATE dashboards SET name = ?, updated_at = ? WHERE id = ? AND user_id = ?`,
		strings.TrimSpace(name), formatTime(now()), id, userID,
	)
	if err != nil {
		return nil, err
	}
	if err := expectOne(res); err != nil {
		return nil, err
	}
	return s.GetDashboard(ctx, userID, id)
}

// DeleteDashboard removes a dashboard and, when it was the default, promotes the
// oldest remaining one. Both steps run in one transaction and the promotion
// error is returned so a user is never left without a default dashboard.
func (s *Store) DeleteDashboard(ctx context.Context, userID, id string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var isDefault int
	err = tx.QueryRowContext(ctx,
		`SELECT is_default FROM dashboards WHERE id = ? AND user_id = ?`, id, userID,
	).Scan(&isDefault)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM dashboards WHERE id = ? AND user_id = ?`, id, userID); err != nil {
		return err
	}
	if isDefault == 1 {
		res, err := tx.ExecContext(ctx,
			`UPDATE dashboards SET is_default = 1, updated_at = ? WHERE id = (
			   SELECT id FROM dashboards WHERE user_id = ? ORDER BY created_at ASC LIMIT 1
			 )`, formatTime(now()), userID,
		)
		if err != nil {
			return err
		}
		// No rows means this was the last dashboard; the next list/create call
		// recreates a default one (see EnsureDefaultDashboard).
		if _, err := res.RowsAffected(); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func scanWidget(scanner interface{ Scan(...any) error }) (*model.Widget, error) {
	var (
		w                    model.Widget
		rawConfig            string
		createdAt, updatedAt string
	)
	if err := scanner.Scan(
		&w.ID, &w.DashboardID, &w.TrackerID, &w.Title, &w.ChartType, &rawConfig,
		&w.Layout.X, &w.Layout.Y, &w.Layout.W, &w.Layout.H, &createdAt, &updatedAt,
	); err != nil {
		return nil, err
	}
	cfg := model.DefaultWidgetConfig()
	if err := json.Unmarshal([]byte(rawConfig), &cfg); err == nil {
		w.Config = cfg
	} else {
		w.Config = model.DefaultWidgetConfig()
	}
	w.Layout.ID = w.ID
	w.CreatedAt = parseTime(createdAt)
	w.UpdatedAt = parseTime(updatedAt)
	return &w, nil
}

func (s *Store) ListWidgets(ctx context.Context, dashboardID string) ([]model.Widget, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+widgetColumns+` FROM widgets WHERE dashboard_id = ? ORDER BY pos_y ASC, pos_x ASC`, dashboardID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]model.Widget, 0, 8)
	for rows.Next() {
		w, err := scanWidget(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *w)
	}
	return out, rows.Err()
}

func (s *Store) GetWidget(ctx context.Context, userID, widgetID string) (*model.Widget, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT `+widgetColumns+` FROM widgets
		 WHERE id = ? AND dashboard_id IN (SELECT id FROM dashboards WHERE user_id = ?)`,
		widgetID, userID,
	)
	w, err := scanWidget(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return w, nil
}

func (s *Store) ownsDashboard(ctx context.Context, userID, dashboardID string) error {
	var count int
	if err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(1) FROM dashboards WHERE id = ? AND user_id = ?`, dashboardID, userID,
	).Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) CreateWidget(ctx context.Context, userID string, w *model.Widget) (*model.Widget, error) {
	if err := s.ownsDashboard(ctx, userID, w.DashboardID); err != nil {
		return nil, err
	}
	cfg, err := json.Marshal(w.Config)
	if err != nil {
		return nil, err
	}
	ts := formatTime(now())
	w.ID = newID()
	if _, err := s.db.ExecContext(ctx,
		`INSERT INTO widgets (id, dashboard_id, tracker_id, title, chart_type, config, pos_x, pos_y, width, height, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		w.ID, w.DashboardID, w.TrackerID, w.Title, w.ChartType, string(cfg),
		w.Layout.X, w.Layout.Y, w.Layout.W, w.Layout.H, ts, ts,
	); err != nil {
		return nil, err
	}
	return s.GetWidget(ctx, userID, w.ID)
}

func (s *Store) UpdateWidget(ctx context.Context, userID string, w *model.Widget) (*model.Widget, error) {
	cfg, err := json.Marshal(w.Config)
	if err != nil {
		return nil, err
	}
	res, err := s.db.ExecContext(ctx,
		`UPDATE widgets SET tracker_id = ?, title = ?, chart_type = ?, config = ?, pos_x = ?, pos_y = ?, width = ?, height = ?, updated_at = ?
		 WHERE id = ? AND dashboard_id IN (SELECT id FROM dashboards WHERE user_id = ?)`,
		w.TrackerID, w.Title, w.ChartType, string(cfg),
		w.Layout.X, w.Layout.Y, w.Layout.W, w.Layout.H, formatTime(now()),
		w.ID, userID,
	)
	if err != nil {
		return nil, err
	}
	if err := expectOne(res); err != nil {
		return nil, err
	}
	return s.GetWidget(ctx, userID, w.ID)
}

func (s *Store) DeleteWidget(ctx context.Context, userID, widgetID string) error {
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM widgets WHERE id = ? AND dashboard_id IN (SELECT id FROM dashboards WHERE user_id = ?)`,
		widgetID, userID,
	)
	if err != nil {
		return err
	}
	return expectOne(res)
}

func (s *Store) UpdateLayouts(ctx context.Context, userID, dashboardID string, items []model.LayoutItem) error {
	if err := s.ownsDashboard(ctx, userID, dashboardID); err != nil {
		return err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	ts := formatTime(now())
	for _, item := range items {
		if _, err := tx.Exec(
			`UPDATE widgets SET pos_x = ?, pos_y = ?, width = ?, height = ?, updated_at = ?
			 WHERE id = ? AND dashboard_id = ?`,
			item.X, item.Y, item.W, item.H, ts, item.ID, dashboardID,
		); err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}
