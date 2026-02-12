package storage

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/hoanghai1803/apricot/internal/models"
)

// CreateSession inserts a new discovery session and returns its ID.
func (s *Store) CreateSession(ctx context.Context, session *models.DiscoverySession) (int64, error) {
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO discovery_sessions
			(preferences_snapshot, blogs_considered, blogs_selected, model_used, input_tokens, output_tokens)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		session.PreferencesSnapshot, session.BlogsConsidered, session.BlogsSelected,
		session.ModelUsed, session.InputTokens, session.OutputTokens,
	)
	if err != nil {
		return 0, fmt.Errorf("creating session: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("getting session id: %w", err)
	}
	return id, nil
}

// GetRecentSessions returns the most recent discovery sessions, ordered by
// created_at DESC and limited to the specified count.
func (s *Store) GetRecentSessions(ctx context.Context, limit int) ([]models.DiscoverySession, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, preferences_snapshot, blogs_considered, blogs_selected,
				model_used, input_tokens, output_tokens, created_at
		 FROM discovery_sessions
		 ORDER BY created_at DESC, id DESC
		 LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("querying recent sessions: %w", err)
	}
	defer rows.Close()

	var sessions []models.DiscoverySession
	for rows.Next() {
		var (
			sess         models.DiscoverySession
			inputTokens  sql.NullInt64
			outputTokens sql.NullInt64
			createdAt    string
		)
		if err := rows.Scan(
			&sess.ID, &sess.PreferencesSnapshot, &sess.BlogsConsidered,
			&sess.BlogsSelected, &sess.ModelUsed, &inputTokens, &outputTokens,
			&createdAt,
		); err != nil {
			return nil, fmt.Errorf("scanning session row: %w", err)
		}
		if inputTokens.Valid {
			v := int(inputTokens.Int64)
			sess.InputTokens = &v
		}
		if outputTokens.Valid {
			v := int(outputTokens.Int64)
			sess.OutputTokens = &v
		}
		sess.CreatedAt = parseTime(createdAt)
		sessions = append(sessions, sess)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating session rows: %w", err)
	}
	return sessions, nil
}
