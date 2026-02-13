// Package storage provides the SQLite persistence layer for Apricot.
//
// It manages database connections, schema migrations, and CRUD operations
// for all domain entities. The database uses WAL journal mode for concurrent
// reads and a single-writer model.
package storage

import (
	"database/sql"
	"embed"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
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

//go:embed migrations/*.sql
var migrationsFS embed.FS

// RunMigrations applies any unapplied schema migrations to the database.
// Migration SQL files are read from the embedded migrations/ directory.
// Each file must be named NNN_description.sql where NNN is the version number.
// Each migration runs inside its own transaction for atomicity.
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

	// Read migration files from the embedded filesystem.
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("reading migrations directory: %w", err)
	}

	// Parse and sort migration files by version number.
	type migrationFile struct {
		version  int
		filename string
	}
	var files []migrationFile
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		version := parseVersion(entry.Name())
		if version <= 0 {
			continue
		}
		files = append(files, migrationFile{version: version, filename: entry.Name()})
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].version < files[j].version
	})

	// Apply each unapplied migration.
	for _, mf := range files {
		if applied[mf.version] {
			continue
		}

		sqlBytes, err := migrationsFS.ReadFile("migrations/" + mf.filename)
		if err != nil {
			return fmt.Errorf("reading migration file %q: %w", mf.filename, err)
		}

		if err := applyMigration(db, mf.version, string(sqlBytes)); err != nil {
			return fmt.Errorf("applying migration %s: %w", mf.filename, err)
		}

		slog.Info("applied migration", "version", mf.version, "file", mf.filename)
	}

	return nil
}

// parseVersion extracts the version number from a migration filename like
// "001_initial_schema.sql" → 1, "002_update_sources.sql" → 2.
func parseVersion(filename string) int {
	parts := strings.SplitN(filename, "_", 2)
	if len(parts) == 0 {
		return 0
	}
	v, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0
	}
	return v
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
func applyMigration(db *sql.DB, version int, sql string) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // rollback after commit is a no-op

	if _, err := tx.Exec(sql); err != nil {
		return fmt.Errorf("executing migration SQL: %w", err)
	}

	if _, err := tx.Exec(
		"INSERT INTO schema_migrations (version) VALUES (?)", version,
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
