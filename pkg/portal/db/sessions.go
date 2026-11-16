package db

import (
	"database/sql"
	"errors"
	"time"
)

var ErrSessionNotFound = errors.New("session not found")

type Session struct {
	ID        string
	UserID    string
	ExpiresAt time.Time
}

func (s *Store) CreateSession(id, userID string, expiresAt time.Time) error {
	_, err := s.DB.Exec(
		`INSERT INTO sessions (id, user_id, expires_at) VALUES (?, ?, ?)`,
		id, userID, expiresAt,
	)
	return err
}

func (s *Store) GetSession(id string) (*Session, error) {
	sess := &Session{}
	err := s.DB.QueryRow(
		`SELECT id, user_id, expires_at FROM sessions WHERE id = ?`, id,
	).Scan(&sess.ID, &sess.UserID, &sess.ExpiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrSessionNotFound
	}
	if err != nil {
		return nil, err
	}
	return sess, nil
}

func (s *Store) ExtendSession(id string, newExpiresAt time.Time) error {
	_, err := s.DB.Exec(
		`UPDATE sessions SET expires_at = ? WHERE id = ?`, newExpiresAt, id,
	)
	return err
}

func (s *Store) DeleteSession(id string) error {
	_, err := s.DB.Exec(`DELETE FROM sessions WHERE id = ?`, id)
	return err
}

func (s *Store) PruneExpiredSessions(now time.Time) error {
	_, err := s.DB.Exec(`DELETE FROM sessions WHERE expires_at < ?`, now)
	return err
}
