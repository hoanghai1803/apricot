package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
)

// GetPreference retrieves a preference by key and JSON-unmarshals it into dest.
// Returns ErrNotFound if the key does not exist.
func (s *Store) GetPreference(ctx context.Context, key string, dest any) error {
	var raw string
	err := s.db.QueryRowContext(ctx,
		`SELECT value FROM preferences WHERE key = ?`, key,
	).Scan(&raw)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("getting preference %q: %w", key, err)
	}

	if err := json.Unmarshal([]byte(raw), dest); err != nil {
		return fmt.Errorf("unmarshaling preference %q: %w", key, err)
	}
	return nil
}

// SetPreference JSON-marshals value and stores it under the given key. If the
// key already exists, its value and updated_at are overwritten.
func (s *Store) SetPreference(ctx context.Context, key string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshaling preference %q: %w", key, err)
	}

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO preferences (key, value, updated_at)
		 VALUES (?, ?, datetime('now'))
		 ON CONFLICT(key) DO UPDATE SET
			value      = excluded.value,
			updated_at = excluded.updated_at`,
		key, string(data),
	)
	if err != nil {
		return fmt.Errorf("setting preference %q: %w", key, err)
	}
	return nil
}

// GetAllPreferences returns every preference as a map of key to raw JSON value.
func (s *Store) GetAllPreferences(ctx context.Context) (map[string]json.RawMessage, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT key, value FROM preferences`)
	if err != nil {
		return nil, fmt.Errorf("querying all preferences: %w", err)
	}
	defer rows.Close()

	prefs := make(map[string]json.RawMessage)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, fmt.Errorf("scanning preference row: %w", err)
		}
		prefs[key] = json.RawMessage(value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating preference rows: %w", err)
	}
	return prefs, nil
}
