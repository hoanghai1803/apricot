package storage

import (
	"context"
	"fmt"

	"github.com/hoanghai1803/apricot/internal/models"
)

// defaultSources defines the 20 engineering blogs seeded into a new database.
var defaultSources = []models.BlogSource{
	{Name: "Netflix Tech Blog", Company: "Netflix", FeedURL: "https://netflixtechblog.com/feed", SiteURL: "https://netflixtechblog.com", IsActive: true},
	{Name: "Engineering at Meta", Company: "Meta", FeedURL: "https://engineering.fb.com/feed/", SiteURL: "https://engineering.fb.com", IsActive: true},
	{Name: "Uber Engineering", Company: "Uber", FeedURL: "https://www.uber.com/blog/engineering/rss/", SiteURL: "https://www.uber.com/blog/engineering", IsActive: true},
	{Name: "AWS Architecture Blog", Company: "AWS", FeedURL: "https://aws.amazon.com/blogs/architecture/feed/", SiteURL: "https://aws.amazon.com/blogs/architecture", IsActive: true},
	{Name: "Google Research Blog", Company: "Google", FeedURL: "https://blog.research.google/feeds/posts/default?alt=rss", SiteURL: "https://blog.research.google", IsActive: true},
	{Name: "Google Cloud Blog", Company: "Google", FeedURL: "https://cloudblog.withgoogle.com/rss/", SiteURL: "https://cloud.google.com/blog", IsActive: true},
	{Name: "Spotify Engineering", Company: "Spotify", FeedURL: "https://engineering.atspotify.com/feed/", SiteURL: "https://engineering.atspotify.com", IsActive: true},
	{Name: "Figma Blog", Company: "Figma", FeedURL: "https://www.figma.com/blog/feed/atom.xml", SiteURL: "https://www.figma.com/blog", IsActive: true},
	{Name: "Datadog Engineering", Company: "Datadog", FeedURL: "https://www.datadoghq.com/blog/engineering/index.xml", SiteURL: "https://www.datadoghq.com/blog/engineering", IsActive: true},
	{Name: "Stripe Engineering", Company: "Stripe", FeedURL: "https://stripe.com/blog/feed.rss", SiteURL: "https://stripe.com/blog", IsActive: true},
	{Name: "Airbnb Tech Blog", Company: "Airbnb", FeedURL: "https://medium.com/feed/airbnb-engineering", SiteURL: "https://medium.com/airbnb-engineering", IsActive: true},
	{Name: "Grab Engineering", Company: "Grab", FeedURL: "https://engineering.grab.com/feed.xml", SiteURL: "https://engineering.grab.com", IsActive: true},
	{Name: "Cloudflare Blog", Company: "Cloudflare", FeedURL: "https://blog.cloudflare.com/rss/", SiteURL: "https://blog.cloudflare.com", IsActive: true},
	{Name: "Slack Engineering", Company: "Slack", FeedURL: "https://slack.engineering/feed/", SiteURL: "https://slack.engineering", IsActive: true},
	{Name: "GitHub Engineering", Company: "GitHub", FeedURL: "https://github.blog/engineering/feed/", SiteURL: "https://github.blog/engineering", IsActive: true},
	{Name: "Vercel Blog", Company: "Vercel", FeedURL: "https://vercel.com/atom", SiteURL: "https://vercel.com/blog", IsActive: true},
	{Name: "Dropbox Tech Blog", Company: "Dropbox", FeedURL: "https://dropbox.tech/feed", SiteURL: "https://dropbox.tech", IsActive: true},
	{Name: "Instacart Tech", Company: "Instacart", FeedURL: "https://tech.instacart.com/feed", SiteURL: "https://tech.instacart.com", IsActive: true},
	{Name: "Pinterest Engineering", Company: "Pinterest", FeedURL: "https://medium.com/feed/pinterest-engineering", SiteURL: "https://medium.com/pinterest-engineering", IsActive: true},
	{Name: "Lyft Engineering", Company: "Lyft", FeedURL: "https://eng.lyft.com/feed", SiteURL: "https://eng.lyft.com", IsActive: true},
}

// GetAllSources returns all blog sources regardless of active status,
// ordered by name.
func (s *Store) GetAllSources(ctx context.Context) ([]models.BlogSource, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, company, feed_url, site_url, is_active, created_at
		 FROM blog_sources ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("querying all sources: %w", err)
	}
	defer rows.Close()

	return scanSources(rows)
}

// GetActiveSources returns all blog sources where is_active = 1,
// ordered by name.
func (s *Store) GetActiveSources(ctx context.Context) ([]models.BlogSource, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, company, feed_url, site_url, is_active, created_at
		 FROM blog_sources WHERE is_active = 1 ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("querying active sources: %w", err)
	}
	defer rows.Close()

	return scanSources(rows)
}

// ToggleSource sets the is_active flag for the given source ID.
// It returns ErrNotFound if no source matches the given ID.
func (s *Store) ToggleSource(ctx context.Context, id int64, active bool) error {
	activeInt := 0
	if active {
		activeInt = 1
	}

	res, err := s.db.ExecContext(ctx,
		`UPDATE blog_sources SET is_active = ? WHERE id = ?`, activeInt, id)
	if err != nil {
		return fmt.Errorf("toggling source %d: %w", id, err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected for source %d: %w", id, err)
	}
	if n == 0 {
		return ErrNotFound
	}

	return nil
}

// SeedDefaults inserts the default blog sources if the blog_sources table is
// empty. All inserts happen within a single transaction. This operation is
// idempotent: calling it on a non-empty table is a no-op.
func (s *Store) SeedDefaults(ctx context.Context) error {
	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM blog_sources`).Scan(&count); err != nil {
		return fmt.Errorf("counting blog sources: %w", err)
	}

	if count > 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning seed transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // rollback after commit is a no-op

	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO blog_sources (name, company, feed_url, site_url, is_active)
		 VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("preparing seed statement: %w", err)
	}
	defer stmt.Close()

	for _, src := range defaultSources {
		activeInt := 0
		if src.IsActive {
			activeInt = 1
		}

		if _, err := stmt.ExecContext(ctx, src.Name, src.Company, src.FeedURL, src.SiteURL, activeInt); err != nil {
			return fmt.Errorf("seeding source %q: %w", src.Name, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing seed transaction: %w", err)
	}

	return nil
}

// scanSources reads all rows from a blog_sources query into a slice.
func scanSources(rows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
},
) ([]models.BlogSource, error) {
	var sources []models.BlogSource
	for rows.Next() {
		var (
			src       models.BlogSource
			isActive  int
			createdAt string
		)
		if err := rows.Scan(
			&src.ID, &src.Name, &src.Company, &src.FeedURL,
			&src.SiteURL, &isActive, &createdAt,
		); err != nil {
			return nil, fmt.Errorf("scanning source row: %w", err)
		}
		src.IsActive = isActive == 1
		src.CreatedAt = parseTime(createdAt)
		sources = append(sources, src)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating source rows: %w", err)
	}

	// Return empty slice instead of nil for consistent JSON serialization.
	if sources == nil {
		sources = []models.BlogSource{}
	}

	return sources, nil
}

// DefaultSourceCount returns the number of default blog sources that will be
// seeded into a new database. Useful for tests.
func DefaultSourceCount() int {
	return len(defaultSources)
}
