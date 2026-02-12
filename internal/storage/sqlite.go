// Package storage provides the SQLite persistence layer for Apricot.
//
// It manages database connections, schema migrations, and CRUD operations
// for all domain entities. The database uses WAL journal mode for concurrent
// reads and a single-writer model.
package storage

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"time"

	_ "modernc.org/sqlite" // Pure Go SQLite driver.
)

// Store wraps a SQL database connection and provides typed query methods
// for all Apricot domain entities.
type Store struct {
	db *sql.DB
}

// NewStore creates a Store backed by the given database connection.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// Close closes the underlying database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// DB returns the underlying *sql.DB for advanced use cases.
func (s *Store) DB() *sql.DB {
	return s.db
}

// OpenDatabase opens (or creates) a SQLite database at the given path.
// It configures the connection for WAL journal mode, a 5-second busy timeout,
// and foreign key enforcement. Parent directories are created if missing.
//
// The returned *sql.DB is limited to a single connection because SQLite
// supports only one concurrent writer.
func OpenDatabase(path string) (*sql.DB, error) {
	// For in-memory databases, skip directory creation.
	if path != ":memory:" {
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("creating database directory %q: %w", dir, err)
		}
	}

	dsn := path + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)"

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening database %q: %w", path, err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	// Verify the connection is usable.
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("pinging database %q: %w", path, err)
	}

	slog.Info("opened sqlite database", "path", path)
	return db, nil
}

// migration defines a single schema migration with a version number and SQL.
type migration struct {
	Version int
	SQL     string
}

// migrations is the ordered list of all schema migrations. Migrations are
// stored as Go constants (rather than embedded files) for simplicity and to
// avoid go:embed path limitations. The corresponding SQL is also maintained
// in migrations/*.sql for documentation.
var migrations = []migration{
	{Version: 1, SQL: migrationV1},
}

const migrationV1 = `
-- Blog sources (the engineering blogs we track)
CREATE TABLE blog_sources (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT NOT NULL,
    company     TEXT NOT NULL,
    feed_url    TEXT NOT NULL UNIQUE,
    site_url    TEXT NOT NULL,
    is_active   INTEGER NOT NULL DEFAULT 1,
    created_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

-- Individual blog posts discovered from RSS feeds
CREATE TABLE blogs (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    source_id       INTEGER NOT NULL REFERENCES blog_sources(id),
    title           TEXT NOT NULL,
    url             TEXT NOT NULL UNIQUE,
    description     TEXT,
    full_content    TEXT,
    published_at    TEXT,
    fetched_at      TEXT NOT NULL DEFAULT (datetime('now')),
    content_hash    TEXT,
    created_at      TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_blogs_published ON blogs(published_at DESC);
CREATE INDEX idx_blogs_source ON blogs(source_id);
CREATE INDEX idx_blogs_url ON blogs(url);

-- Cached AI-generated summaries
CREATE TABLE blog_summaries (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    blog_id     INTEGER NOT NULL UNIQUE REFERENCES blogs(id),
    summary     TEXT NOT NULL,
    model_used  TEXT NOT NULL,
    created_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

-- User preferences (single user, so no user_id needed)
CREATE TABLE preferences (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    key         TEXT NOT NULL UNIQUE,
    value       TEXT NOT NULL,
    updated_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

-- Reading list / wishlist
CREATE TABLE reading_list (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    blog_id     INTEGER NOT NULL UNIQUE REFERENCES blogs(id),
    status      TEXT NOT NULL DEFAULT 'unread',
    notes       TEXT,
    added_at    TEXT NOT NULL DEFAULT (datetime('now')),
    read_at     TEXT
);

CREATE INDEX idx_reading_status ON reading_list(status);

-- Discovery sessions (audit trail of each discovery run)
CREATE TABLE discovery_sessions (
    id                   INTEGER PRIMARY KEY AUTOINCREMENT,
    preferences_snapshot TEXT NOT NULL,
    blogs_considered     INTEGER NOT NULL,
    blogs_selected       TEXT NOT NULL,
    model_used           TEXT NOT NULL,
    input_tokens         INTEGER,
    output_tokens        INTEGER,
    created_at           TEXT NOT NULL DEFAULT (datetime('now'))
);
`

// RunMigrations applies any unapplied schema migrations to the database.
// It first ensures the schema_migrations tracking table exists, then applies
// each migration whose version has not yet been recorded. Each migration
// runs inside its own transaction for atomicity.
func RunMigrations(db *sql.DB) error {
	// Ensure the tracking table exists.
	const createTracker = `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    INTEGER PRIMARY KEY,
			applied_at TEXT NOT NULL DEFAULT (datetime('now'))
		);
	`
	if _, err := db.Exec(createTracker); err != nil {
		return fmt.Errorf("creating schema_migrations table: %w", err)
	}

	// Load already-applied versions.
	applied, err := appliedVersions(db)
	if err != nil {
		return fmt.Errorf("reading applied migrations: %w", err)
	}

	// Sort migrations by version to guarantee ordering.
	sorted := make([]migration, len(migrations))
	copy(sorted, migrations)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Version < sorted[j].Version
	})

	for _, m := range sorted {
		if applied[m.Version] {
			continue
		}

		if err := applyMigration(db, m); err != nil {
			return fmt.Errorf("applying migration v%d: %w", m.Version, err)
		}

		slog.Info("applied migration", "version", m.Version)
	}

	return nil
}

// appliedVersions returns a set of migration versions that have already been
// applied to the database.
func appliedVersions(db *sql.DB) (map[int]bool, error) {
	rows, err := db.Query("SELECT version FROM schema_migrations ORDER BY version")
	if err != nil {
		return nil, fmt.Errorf("querying schema_migrations: %w", err)
	}
	defer rows.Close()

	versions := make(map[int]bool)
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			return nil, fmt.Errorf("scanning migration version: %w", err)
		}
		versions[v] = true
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating migration versions: %w", err)
	}

	return versions, nil
}

// applyMigration executes a single migration's SQL and records its version,
// all within a single transaction.
func applyMigration(db *sql.DB, m migration) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // rollback after commit is a no-op

	if _, err := tx.Exec(m.SQL); err != nil {
		return fmt.Errorf("executing migration SQL: %w", err)
	}

	if _, err := tx.Exec(
		"INSERT INTO schema_migrations (version) VALUES (?)", m.Version,
	); err != nil {
		return fmt.Errorf("recording migration version: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing migration: %w", err)
	}

	return nil
}

// parseTime attempts to parse a SQLite datetime string in common formats.
// It returns the zero time if parsing fails.
func parseTime(s string) time.Time {
	for _, layout := range []string{
		"2006-01-02 15:04:05",
		time.RFC3339,
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

// parseTimePtr is like parseTime but returns nil for empty strings.
func parseTimePtr(s *string) *time.Time {
	if s == nil || *s == "" {
		return nil
	}
	t := parseTime(*s)
	if t.IsZero() {
		return nil
	}
	return &t
}
