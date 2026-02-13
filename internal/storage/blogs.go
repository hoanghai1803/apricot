package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/hoanghai1803/apricot/internal/models"
)

// UpsertBlog inserts a blog post or updates it if a row with the same URL
// already exists. On conflict the full_content, content_hash, and fetched_at
// fields are updated. The row ID is returned.
func (s *Store) UpsertBlog(ctx context.Context, blog *models.Blog) (int64, error) {
	var publishedAt *string
	if blog.PublishedAt != nil {
		v := blog.PublishedAt.Format("2006-01-02 15:04:05")
		publishedAt = &v
	}

	fetchedAt := blog.FetchedAt.Format("2006-01-02 15:04:05")

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO blogs (source_id, title, url, description, full_content, published_at, fetched_at, content_hash)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(url) DO UPDATE SET
			full_content = excluded.full_content,
			content_hash = excluded.content_hash,
			fetched_at   = excluded.fetched_at`,
		blog.SourceID, blog.Title, blog.URL, nullableString(blog.Description),
		nullableString(blog.FullContent), publishedAt, fetchedAt,
		nullableString(blog.ContentHash),
	)
	if err != nil {
		return 0, fmt.Errorf("upserting blog: %w", err)
	}

	// Retrieve the ID of the upserted row. SQLite's last_insert_rowid()
	// may not reflect the correct ID on an UPDATE path, so we query by URL.
	var id int64
	if err := s.db.QueryRowContext(ctx, `SELECT id FROM blogs WHERE url = ?`, blog.URL).Scan(&id); err != nil {
		return 0, fmt.Errorf("getting upserted blog id: %w", err)
	}
	return id, nil
}

// GetBlogByURL returns the blog post with the given URL.
// Returns nil, ErrNotFound if no matching row exists.
func (s *Store) GetBlogByURL(ctx context.Context, url string) (*models.Blog, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT b.id, b.source_id, COALESCE(b.custom_source, bs.name, '') AS source, b.title, b.url,
				b.description, b.full_content, b.published_at, b.fetched_at,
				b.content_hash, b.created_at
		 FROM blogs b
		 LEFT JOIN blog_sources bs ON bs.id = b.source_id
		 WHERE b.url = ?`, url)

	blog, err := scanBlog(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("getting blog by url: %w", err)
	}
	return blog, nil
}

// GetBlogByID returns the blog post with the given ID.
// Returns nil, ErrNotFound if no matching row exists.
func (s *Store) GetBlogByID(ctx context.Context, id int64) (*models.Blog, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT b.id, b.source_id, COALESCE(b.custom_source, bs.name, '') AS source, b.title, b.url,
				b.description, b.full_content, b.published_at, b.fetched_at,
				b.content_hash, b.created_at
		 FROM blogs b
		 LEFT JOIN blog_sources bs ON bs.id = b.source_id
		 WHERE b.id = ?`, id)

	blog, err := scanBlog(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("getting blog by id: %w", err)
	}
	return blog, nil
}

// GetCustomSourceID returns the ID of the sentinel "custom://user-added" source.
func (s *Store) GetCustomSourceID(ctx context.Context) (int64, error) {
	var id int64
	err := s.db.QueryRowContext(ctx,
		`SELECT id FROM blog_sources WHERE feed_url = 'custom://user-added'`,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("getting custom source id: %w", err)
	}
	return id, nil
}

// CreateCustomBlog inserts a user-added blog post linked to the sentinel
// "Custom" source. The customSource field overrides the display source name.
func (s *Store) CreateCustomBlog(ctx context.Context, url, title, description, fullContent, customSource string) (int64, error) {
	sourceID, err := s.GetCustomSourceID(ctx)
	if err != nil {
		return 0, err
	}

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO blogs (source_id, title, url, description, full_content, fetched_at, custom_source)
		 VALUES (?, ?, ?, ?, ?, datetime('now'), ?)`,
		sourceID, title, url, nullableString(description),
		nullableString(fullContent), nullableString(customSource),
	)
	if err != nil {
		return 0, fmt.Errorf("creating custom blog: %w", err)
	}

	var id int64
	if err := s.db.QueryRowContext(ctx, `SELECT id FROM blogs WHERE url = ?`, url).Scan(&id); err != nil {
		return 0, fmt.Errorf("getting custom blog id: %w", err)
	}
	return id, nil
}

// SaveBlogs batch-upserts multiple blog posts inside a single transaction.
func (s *Store) SaveBlogs(ctx context.Context, blogs []models.Blog) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // rollback after commit is a no-op

	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO blogs (source_id, title, url, description, full_content, published_at, fetched_at, content_hash)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(url) DO UPDATE SET
			full_content = excluded.full_content,
			content_hash = excluded.content_hash,
			fetched_at   = excluded.fetched_at`)
	if err != nil {
		return fmt.Errorf("preparing statement: %w", err)
	}
	defer stmt.Close()

	for i := range blogs {
		b := &blogs[i]
		var publishedAt *string
		if b.PublishedAt != nil {
			v := b.PublishedAt.Format("2006-01-02 15:04:05")
			publishedAt = &v
		}
		fetchedAt := b.FetchedAt.Format("2006-01-02 15:04:05")

		if _, err := stmt.ExecContext(ctx,
			b.SourceID, b.Title, b.URL, nullableString(b.Description),
			nullableString(b.FullContent), publishedAt, fetchedAt,
			nullableString(b.ContentHash),
		); err != nil {
			return fmt.Errorf("upserting blog %q: %w", b.URL, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}
	return nil
}

// scanner is a minimal interface satisfied by both *sql.Row and *sql.Rows.
type scanner interface {
	Scan(dest ...any) error
}

// scanBlog scans a single blog row into a models.Blog.
func scanBlog(row scanner) (*models.Blog, error) {
	var (
		blog        models.Blog
		description sql.NullString
		fullContent sql.NullString
		publishedAt sql.NullString
		fetchedAt   string
		contentHash sql.NullString
		createdAt   string
	)

	if err := row.Scan(
		&blog.ID, &blog.SourceID, &blog.Source, &blog.Title, &blog.URL,
		&description, &fullContent, &publishedAt, &fetchedAt,
		&contentHash, &createdAt,
	); err != nil {
		return nil, err
	}

	blog.Description = description.String
	blog.FullContent = fullContent.String
	blog.ContentHash = contentHash.String
	blog.PublishedAt = parseTimePtr(nullStringToPtr(publishedAt))
	blog.FetchedAt = parseTime(fetchedAt)
	blog.CreatedAt = parseTime(createdAt)

	return &blog, nil
}

// nullableString converts an empty string to nil for nullable TEXT columns.
func nullableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// nullStringToPtr converts a sql.NullString to a *string.
func nullStringToPtr(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	return &ns.String
}
