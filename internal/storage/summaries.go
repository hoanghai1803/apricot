package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/hoanghai1803/apricot/internal/models"
)

// UpsertSummary inserts a blog summary or updates it if a row with the same
// blog_id already exists.
func (s *Store) UpsertSummary(ctx context.Context, summary *models.BlogSummary) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO blog_summaries (blog_id, summary, model_used)
		 VALUES (?, ?, ?)
		 ON CONFLICT(blog_id) DO UPDATE SET
			summary    = excluded.summary,
			model_used = excluded.model_used,
			created_at = datetime('now')`,
		summary.BlogID, summary.Summary, summary.ModelUsed,
	)
	if err != nil {
		return fmt.Errorf("upserting summary: %w", err)
	}
	return nil
}

// GetSummaryByBlogID returns the summary for the given blog ID.
// Returns nil, ErrNotFound if no matching row exists.
func (s *Store) GetSummaryByBlogID(ctx context.Context, blogID int64) (*models.BlogSummary, error) {
	var (
		summary   models.BlogSummary
		createdAt string
	)
	err := s.db.QueryRowContext(ctx,
		`SELECT id, blog_id, summary, model_used, created_at
		 FROM blog_summaries WHERE blog_id = ?`, blogID,
	).Scan(&summary.ID, &summary.BlogID, &summary.Summary, &summary.ModelUsed, &createdAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("getting summary by blog id: %w", err)
	}
	summary.CreatedAt = parseTime(createdAt)
	return &summary, nil
}

// HasSummary returns true if a summary exists for the given blog ID.
func (s *Store) HasSummary(ctx context.Context, blogID int64) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM blog_summaries WHERE blog_id = ?)`, blogID,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("checking summary existence: %w", err)
	}
	return exists, nil
}
