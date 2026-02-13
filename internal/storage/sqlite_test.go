package storage

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

// newTestDB creates an in-memory SQLite database with migrations applied.
// The database is automatically closed when the test completes.
func newTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := OpenDatabase(":memory:")
	if err != nil {
		t.Fatalf("opening test database: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	if err := RunMigrations(db); err != nil {
		t.Fatalf("running migrations: %v", err)
	}

	return db
}

// newTestStore creates an in-memory Store with migrations applied.
// The store is automatically closed when the test completes.
func newTestStore(t *testing.T) *Store {
	t.Helper()

	db := newTestDB(t)
	return NewStore(db)
}

func TestOpenDatabase_InMemory(t *testing.T) {
	db, err := OpenDatabase(":memory:")
	if err != nil {
		t.Fatalf("OpenDatabase(:memory:) error: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Fatalf("Ping() error: %v", err)
	}
}

func TestOpenDatabase_CreatesDirectoryAndFile(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "nested", "deep", "test.db")

	db, err := OpenDatabase(dbPath)
	if err != nil {
		t.Fatalf("OpenDatabase(%q) error: %v", dbPath, err)
	}
	defer db.Close()

	// Verify the file was created.
	if _, err := os.Stat(dbPath); err != nil {
		t.Fatalf("database file not created at %q: %v", dbPath, err)
	}
}

func TestRunMigrations_AppliesSchema(t *testing.T) {
	db, err := OpenDatabase(":memory:")
	if err != nil {
		t.Fatalf("OpenDatabase error: %v", err)
	}
	defer db.Close()

	if err := RunMigrations(db); err != nil {
		t.Fatalf("RunMigrations error: %v", err)
	}

	// Verify expected tables exist by querying sqlite_master.
	expectedTables := []string{
		"blog_sources",
		"blogs",
		"blog_summaries",
		"preferences",
		"reading_list",
		"discovery_sessions",
		"schema_migrations",
		"tags",
		"reading_list_tags",
	}

	for _, table := range expectedTables {
		var name string
		err := db.QueryRow(
			"SELECT name FROM sqlite_master WHERE type='table' AND name=?", table,
		).Scan(&name)
		if err != nil {
			t.Errorf("table %q not found: %v", table, err)
		}
	}

	// Verify indexes exist.
	expectedIndexes := []string{
		"idx_blogs_published",
		"idx_blogs_source",
		"idx_blogs_url",
		"idx_reading_status",
	}

	for _, idx := range expectedIndexes {
		var name string
		err := db.QueryRow(
			"SELECT name FROM sqlite_master WHERE type='index' AND name=?", idx,
		).Scan(&name)
		if err != nil {
			t.Errorf("index %q not found: %v", idx, err)
		}
	}

	// Verify migration version was recorded.
	var version int
	err = db.QueryRow("SELECT version FROM schema_migrations WHERE version = 1").Scan(&version)
	if err != nil {
		t.Fatalf("migration version 1 not recorded: %v", err)
	}
	if version != 1 {
		t.Fatalf("expected version 1, got %d", version)
	}
}

func TestRunMigrations_Idempotent(t *testing.T) {
	db, err := OpenDatabase(":memory:")
	if err != nil {
		t.Fatalf("OpenDatabase error: %v", err)
	}
	defer db.Close()

	// Run migrations twice.
	if err := RunMigrations(db); err != nil {
		t.Fatalf("first RunMigrations error: %v", err)
	}
	if err := RunMigrations(db); err != nil {
		t.Fatalf("second RunMigrations error: %v", err)
	}

	// Verify only one migration version is recorded (not duplicated).
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count); err != nil {
		t.Fatalf("counting migrations: %v", err)
	}
	if count != 4 {
		t.Fatalf("expected 4 migration records, got %d", count)
	}
}

func TestNewStore(t *testing.T) {
	db := newTestDB(t)
	store := NewStore(db)

	if store.DB() != db {
		t.Fatal("NewStore did not store the provided *sql.DB")
	}
}

func TestParseTime(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string // expected in "2006-01-02 15:04:05" format, or "zero"
	}{
		{
			name:  "sqlite format",
			input: "2025-01-15 10:30:00",
			want:  "2025-01-15 10:30:00",
		},
		{
			name:  "RFC3339",
			input: "2025-01-15T10:30:00Z",
			want:  "2025-01-15 10:30:00",
		},
		{
			name:  "invalid",
			input: "not-a-date",
			want:  "zero",
		},
		{
			name:  "empty",
			input: "",
			want:  "zero",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseTime(tt.input)
			if tt.want == "zero" {
				if !got.IsZero() {
					t.Errorf("parseTime(%q) = %v, want zero time", tt.input, got)
				}
				return
			}
			gotStr := got.Format("2006-01-02 15:04:05")
			if gotStr != tt.want {
				t.Errorf("parseTime(%q) = %q, want %q", tt.input, gotStr, tt.want)
			}
		})
	}
}

func TestParseTimePtr(t *testing.T) {
	t.Run("nil input", func(t *testing.T) {
		got := parseTimePtr(nil)
		if got != nil {
			t.Errorf("parseTimePtr(nil) = %v, want nil", got)
		}
	})

	t.Run("empty string", func(t *testing.T) {
		s := ""
		got := parseTimePtr(&s)
		if got != nil {
			t.Errorf("parseTimePtr(&\"\") = %v, want nil", got)
		}
	})

	t.Run("valid time", func(t *testing.T) {
		s := "2025-01-15 10:30:00"
		got := parseTimePtr(&s)
		if got == nil {
			t.Fatal("parseTimePtr returned nil for valid input")
		}
		want := "2025-01-15 10:30:00"
		gotStr := got.Format("2006-01-02 15:04:05")
		if gotStr != want {
			t.Errorf("parseTimePtr(%q) = %q, want %q", s, gotStr, want)
		}
	})
}
