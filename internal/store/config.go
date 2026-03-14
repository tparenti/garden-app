package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

func (s *SQLiteStore) GetConfig(ctx context.Context, key string) (string, error) {
	var value string
	err := s.db.QueryRowContext(ctx, "SELECT value FROM config WHERE key = ?", key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("config key %q not set", key)
	}
	return value, err
}

func (s *SQLiteStore) SetConfig(ctx context.Context, key, value string) error {
	_, err := s.db.ExecContext(ctx,
		"INSERT INTO config (key, value) VALUES (?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value",
		key, value)
	return err
}
