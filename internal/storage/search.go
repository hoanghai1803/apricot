package storage

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/hoanghai1803/apricot/internal/models"
)

// SearchBlogs performs a full-text search on blogs using FTS5.
// Returns matching blogs with source names joined, limited to the given count.
func (s *Store) SearchBlogs(ctx context.Context, query string, limit int) ([]models.Blog, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return []models.Blog{}, nil
	}
	if limit <= 0 {
		limit = 20
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT b.id, b.source_id, COALESCE(b.custom_source, bs.name, '') AS source,
				b.title, b.url, b.description, b.full_content,
				b.published_at, b.fetched_at, b.content_hash, b.reading_time_minutes, b.created_at
		 FROM blogs_fts fts
		 JOIN blogs b ON b.id = fts.rowid
		 LEFT JOIN blog_sources bs ON bs.id = b.source_id
		 WHERE blogs_fts MATCH ?
		 ORDER BY rank
		 LIMIT ?`,
		query, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("searching blogs: %w", err)
	}
	defer rows.Close()

	var blogs []models.Blog
	for rows.Next() {
		var (
			blog           models.Blog
			description    sql.NullString
			fullContent    sql.NullString
			publishedAt    sql.NullString
			fetchedAt      string
			contentHash    sql.NullString
			readingTimeMin sql.NullInt64
			createdAt      string
		)

		if err := rows.Scan(
			&blog.ID, &blog.SourceID, &blog.Source,
			&blog.Title, &blog.URL,
			&description, &fullContent,
			&publishedAt, &fetchedAt,
			&contentHash, &readingTimeMin, &createdAt,
		); err != nil {
			return nil, fmt.Errorf("scanning search result: %w", err)
		}

		blog.Description = description.String
		blog.FullContent = fullContent.String
		blog.ContentHash = contentHash.String
		if readingTimeMin.Valid {
			v := int(readingTimeMin.Int64)
			blog.ReadingTimeMinutes = &v
		}
		blog.PublishedAt = parseTimePtr(nullStringToPtr(publishedAt))
		blog.FetchedAt = parseTime(fetchedAt)
		blog.CreatedAt = parseTime(createdAt)

		blogs = append(blogs, blog)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating search results: %w", err)
	}

	if blogs == nil {
		blogs = []models.Blog{}
	}
	return blogs, nil
}
