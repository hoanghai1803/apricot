package storage

import (
	"context"
	"errors"
	"testing"
)

func TestSeedDefaults_Inserts20Sources(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	if err := store.SeedDefaults(ctx); err != nil {
		t.Fatalf("SeedDefaults error: %v", err)
	}

	sources, err := store.GetAllSources(ctx)
	if err != nil {
		t.Fatalf("GetAllSources error: %v", err)
	}

	want := DefaultSourceCount()
	if len(sources) != want {
		t.Fatalf("got %d sources, want %d", len(sources), want)
	}
}

func TestSeedDefaults_Idempotent(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	// Seed twice.
	if err := store.SeedDefaults(ctx); err != nil {
		t.Fatalf("first SeedDefaults error: %v", err)
	}
	if err := store.SeedDefaults(ctx); err != nil {
		t.Fatalf("second SeedDefaults error: %v", err)
	}

	sources, err := store.GetAllSources(ctx)
	if err != nil {
		t.Fatalf("GetAllSources error: %v", err)
	}

	want := DefaultSourceCount()
	if len(sources) != want {
		t.Fatalf("got %d sources after double seed, want %d", len(sources), want)
	}
}

func TestGetAllSources(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	// Empty table should return empty slice, not nil.
	sources, err := store.GetAllSources(ctx)
	if err != nil {
		t.Fatalf("GetAllSources error on empty table: %v", err)
	}
	if sources == nil {
		t.Fatal("GetAllSources returned nil, want empty slice")
	}
	if len(sources) != 0 {
		t.Fatalf("got %d sources on empty table, want 0", len(sources))
	}

	// Seed and verify all are returned.
	if err := store.SeedDefaults(ctx); err != nil {
		t.Fatalf("SeedDefaults error: %v", err)
	}

	sources, err = store.GetAllSources(ctx)
	if err != nil {
		t.Fatalf("GetAllSources error after seed: %v", err)
	}

	want := DefaultSourceCount()
	if len(sources) != want {
		t.Fatalf("got %d sources, want %d", len(sources), want)
	}

	// Verify sources have expected fields populated.
	for _, src := range sources {
		if src.ID == 0 {
			t.Error("source has zero ID")
		}
		if src.Name == "" {
			t.Error("source has empty Name")
		}
		if src.Company == "" {
			t.Error("source has empty Company")
		}
		if src.FeedURL == "" {
			t.Error("source has empty FeedURL")
		}
		if src.SiteURL == "" {
			t.Error("source has empty SiteURL")
		}
		if src.CreatedAt.IsZero() {
			t.Errorf("source %q has zero CreatedAt", src.Name)
		}
	}

	// Verify sources are ordered by name.
	for i := 1; i < len(sources); i++ {
		if sources[i].Name < sources[i-1].Name {
			t.Errorf("sources not ordered by name: %q came after %q",
				sources[i].Name, sources[i-1].Name)
		}
	}
}

func TestGetActiveSources(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	if err := store.SeedDefaults(ctx); err != nil {
		t.Fatalf("SeedDefaults error: %v", err)
	}

	// All default sources are active.
	active, err := store.GetActiveSources(ctx)
	if err != nil {
		t.Fatalf("GetActiveSources error: %v", err)
	}

	want := DefaultSourceCount()
	if len(active) != want {
		t.Fatalf("got %d active sources, want %d", len(active), want)
	}

	// Deactivate the first source.
	if err := store.ToggleSource(ctx, active[0].ID, false); err != nil {
		t.Fatalf("ToggleSource error: %v", err)
	}

	// Now we should have one fewer active source.
	active, err = store.GetActiveSources(ctx)
	if err != nil {
		t.Fatalf("GetActiveSources error after toggle: %v", err)
	}
	if len(active) != want-1 {
		t.Fatalf("got %d active sources after deactivation, want %d", len(active), want-1)
	}

	// Verify IsActive is true for all returned sources.
	for _, src := range active {
		if !src.IsActive {
			t.Errorf("GetActiveSources returned inactive source %q", src.Name)
		}
	}
}

func TestToggleSource(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	if err := store.SeedDefaults(ctx); err != nil {
		t.Fatalf("SeedDefaults error: %v", err)
	}

	all, err := store.GetAllSources(ctx)
	if err != nil {
		t.Fatalf("GetAllSources error: %v", err)
	}

	targetID := all[0].ID

	// Deactivate.
	if err := store.ToggleSource(ctx, targetID, false); err != nil {
		t.Fatalf("ToggleSource(false) error: %v", err)
	}

	// Verify the source is now inactive by checking GetActiveSources.
	active, err := store.GetActiveSources(ctx)
	if err != nil {
		t.Fatalf("GetActiveSources error: %v", err)
	}
	for _, src := range active {
		if src.ID == targetID {
			t.Fatalf("source %d still active after deactivation", targetID)
		}
	}

	// Reactivate.
	if err := store.ToggleSource(ctx, targetID, true); err != nil {
		t.Fatalf("ToggleSource(true) error: %v", err)
	}

	active, err = store.GetActiveSources(ctx)
	if err != nil {
		t.Fatalf("GetActiveSources error after reactivation: %v", err)
	}

	found := false
	for _, src := range active {
		if src.ID == targetID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("source %d not found in active sources after reactivation", targetID)
	}
}

func TestToggleSource_NotFound(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	err := store.ToggleSource(ctx, 99999, false)
	if err == nil {
		t.Fatal("expected error for non-existent source, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got: %v", err)
	}
}

func TestDefaultSourceCount(t *testing.T) {
	got := DefaultSourceCount()
	if got != 21 {
		t.Fatalf("DefaultSourceCount() = %d, want 21", got)
	}
}
