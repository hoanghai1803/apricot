package handlers

import (
	"context"
	"testing"

	"github.com/hoanghai1803/apricot/internal/storage"
)

// newTestStore creates an in-memory SQLite store with migrations applied and
// default sources seeded. It registers a cleanup function to close the database
// when the test completes.
func newTestStore(t *testing.T) *storage.Store {
	t.Helper()

	db, err := storage.OpenDatabase(":memory:")
	if err != nil {
		t.Fatalf("opening test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	if err := storage.RunMigrations(db); err != nil {
		t.Fatalf("running migrations: %v", err)
	}

	store := storage.NewStore(db)
	if err := store.SeedDefaults(context.Background()); err != nil {
		t.Fatalf("seeding defaults: %v", err)
	}

	return store
}
