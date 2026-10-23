package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/a-chris/trackete/backend/internal/model"
	"github.com/google/uuid"
)

func newID() string { return uuid.NewString() }

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "UNIQUE constraint failed")
}

func (s *Store) CreateUser(ctx context.Context, email, name, passwordHash string) (*model.User, error) {
	ts := formatTime(now())
	u := &model.User{
		ID:        newID(),
		Email:     strings.ToLower(strings.TrimSpace(email)),
		Name:      strings.TrimSpace(name),
		CreatedAt: now(),
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO users (id, email, name, password_hash, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		u.ID, u.Email, u.Name, passwordHash, ts, ts,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrEmailTaken
		}
		return nil, err
	}
	return u, nil
}

func (s *Store) UserByEmail(ctx context.Context, email string) (*model.User, string, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, email, name, password_hash, created_at, token_version FROM users WHERE email = ?`,
		strings.ToLower(strings.TrimSpace(email)),
	)
	var (
		u       model.User
		hash    string
		created string
	)
	if err := row.Scan(&u.ID, &u.Email, &u.Name, &hash, &created, &u.TokenVersion); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, "", ErrNotFound
		}
		return nil, "", err
	}
	u.CreatedAt = parseTime(created)
	return &u, hash, nil
}

func (s *Store) UserByID(ctx context.Context, id string) (*model.User, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, email, name, created_at, token_version FROM users WHERE id = ?`, id,
	)
	var (
		u       model.User
		created string
	)
	if err := row.Scan(&u.ID, &u.Email, &u.Name, &created, &u.TokenVersion); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	u.CreatedAt = parseTime(created)
	return &u, nil
}

func (s *Store) UpdateUserName(ctx context.Context, userID, name string) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE users SET name = ?, updated_at = ? WHERE id = ?`,
		strings.TrimSpace(name), formatTime(now()), userID,
	)
	if err != nil {
		return err
	}
	return expectOne(res)
}

// UpdateUserPassword stores a new hash and bumps token_version so every
// existing session is rejected. It returns the new version, which the caller
// embeds in the replacement session token.
func (s *Store) UpdateUserPassword(ctx context.Context, userID, passwordHash string) (int, error) {
	res, err := s.db.ExecContext(ctx,
		`UPDATE users SET password_hash = ?, token_version = token_version + 1, updated_at = ? WHERE id = ?`,
		passwordHash, formatTime(now()), userID,
	)
	if err != nil {
		return 0, err
	}
	if err := expectOne(res); err != nil {
		return 0, err
	}
	var version int
	if err := s.db.QueryRowContext(ctx,
		`SELECT token_version FROM users WHERE id = ?`, userID,
	).Scan(&version); err != nil {
		return 0, err
	}
	return version, nil
}

func expectOne(res sql.Result) error {
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
