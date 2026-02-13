package storage

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/hoanghai1803/apricot/internal/models"
)

// validStatuses is the set of allowed reading list statuses.
var validStatuses = map[string]bool{
	"unread":  true,
	"reading": true,
	"read":    true,
}

// AddToReadingList adds a blog post to the reading list with status "unread".
// Returns a descriptive error if the blog_id does not exist (foreign key) or
// the blog is already on the list (unique constraint).
func (s *Store) AddToReadingList(ctx context.Context, blogID int64) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO reading_list (blog_id, status) VALUES (?, 'unread')`,
		blogID,
	)
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "UNIQUE constraint failed") {
			return fmt.Errorf("blog %d is already on the reading list", blogID)
		}
		if strings.Contains(errMsg, "FOREIGN KEY constraint failed") {
			return fmt.Errorf("blog %d does not exist", blogID)
		}
		return fmt.Errorf("adding to reading list: %w", err)
	}
	return nil
}

// GetReadingList returns reading list items with associated blog data and
// summaries. If status is empty, all items are returned. Results are ordered
// by added_at DESC.
func (s *Store) GetReadingList(ctx context.Context, status string) ([]models.ReadingListItem, error) {
	query := `
		SELECT rl.id, rl.blog_id, rl.status, rl.notes, rl.added_at, rl.read_at,
			   b.id, b.source_id, COALESCE(b.custom_source, bs.name, '') AS source, b.title, b.url,
			   b.description, b.full_content, b.published_at, b.fetched_at,
			   b.content_hash, b.created_at,
			   s.summary
		FROM reading_list rl
		JOIN blogs b ON b.id = rl.blog_id
		LEFT JOIN blog_sources bs ON bs.id = b.source_id
		LEFT JOIN blog_summaries s ON s.blog_id = rl.blog_id`

	var args []any
	if status != "" {
		query += " WHERE rl.status = ?"
		args = append(args, status)
	}
	query += " ORDER BY rl.added_at DESC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying reading list: %w", err)
	}
	defer rows.Close()

	var items []models.ReadingListItem
	for rows.Next() {
		var (
			item        models.ReadingListItem
			notes       sql.NullString
			addedAt     string
			readAt      sql.NullString
			blog        models.Blog
			description sql.NullString
			fullContent sql.NullString
			publishedAt sql.NullString
			fetchedAt   string
			contentHash sql.NullString
			blogCreated string
			summary     sql.NullString
		)

		if err := rows.Scan(
			&item.ID, &item.BlogID, &item.Status, &notes, &addedAt, &readAt,
			&blog.ID, &blog.SourceID, &blog.Source, &blog.Title, &blog.URL,
			&description, &fullContent, &publishedAt, &fetchedAt,
			&contentHash, &blogCreated,
			&summary,
		); err != nil {
			return nil, fmt.Errorf("scanning reading list row: %w", err)
		}

		if notes.Valid {
			item.Notes = &notes.String
		}
		item.AddedAt = parseTime(addedAt)
		item.ReadAt = parseTimePtr(nullStringToPtr(readAt))

		blog.Description = description.String
		blog.FullContent = fullContent.String
		blog.ContentHash = contentHash.String
		blog.PublishedAt = parseTimePtr(nullStringToPtr(publishedAt))
		blog.FetchedAt = parseTime(fetchedAt)
		blog.CreatedAt = parseTime(blogCreated)
		item.Blog = &blog

		if summary.Valid {
			item.Summary = &summary.String
		}

		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating reading list rows: %w", err)
	}

	// Initialize empty Tags slices and load tags from join table.
	for i := range items {
		items[i].Tags = []string{}
	}
	if err := s.loadTagsForItems(ctx, items); err != nil {
		return nil, fmt.Errorf("loading tags: %w", err)
	}

	return items, nil
}

// UpdateReadingListStatus updates the status of a reading list item. The
// status must be one of "unread", "reading", or "read". When the status
// becomes "read", read_at is set to the current time; otherwise it is cleared.
func (s *Store) UpdateReadingListStatus(ctx context.Context, id int64, status string) error {
	if !validStatuses[status] {
		return fmt.Errorf("invalid reading list status %q: must be one of unread, reading, read", status)
	}

	var query string
	if status == "read" {
		query = `UPDATE reading_list SET status = ?, read_at = datetime('now') WHERE id = ?`
	} else {
		query = `UPDATE reading_list SET status = ?, read_at = NULL WHERE id = ?`
	}

	res, err := s.db.ExecContext(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("updating reading list status: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// UpdateReadingListNotes updates the notes field of a reading list item.
func (s *Store) UpdateReadingListNotes(ctx context.Context, id int64, notes string) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE reading_list SET notes = ? WHERE id = ?`,
		nullableString(notes), id,
	)
	if err != nil {
		return fmt.Errorf("updating reading list notes: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// RemoveFromReadingList deletes a reading list item by ID.
func (s *Store) RemoveFromReadingList(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM reading_list WHERE id = ?`, id,
	)
	if err != nil {
		return fmt.Errorf("removing from reading list: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
