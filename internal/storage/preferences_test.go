package storage

import (
	"context"
	"errors"
	"testing"
)

func TestPreferences_SetAndGet_String(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	if err := store.SetPreference(ctx, "theme", "dark"); err != nil {
		t.Fatalf("SetPreference() error: %v", err)
	}

	var got string
	if err := store.GetPreference(ctx, "theme", &got); err != nil {
		t.Fatalf("GetPreference() error: %v", err)
	}
	if got != "dark" {
		t.Errorf("got %q, want %q", got, "dark")
	}
}

func TestPreferences_SetAndGet_Int(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	if err := store.SetPreference(ctx, "max_results", 42); err != nil {
		t.Fatalf("SetPreference() error: %v", err)
	}

	var got int
	if err := store.GetPreference(ctx, "max_results", &got); err != nil {
		t.Fatalf("GetPreference() error: %v", err)
	}
	if got != 42 {
		t.Errorf("got %d, want %d", got, 42)
	}
}

func TestPreferences_SetAndGet_IntSlice(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	ids := []int64{1, 2, 3, 42}
	if err := store.SetPreference(ctx, "source_ids", ids); err != nil {
		t.Fatalf("SetPreference() error: %v", err)
	}

	var got []int64
	if err := store.GetPreference(ctx, "source_ids", &got); err != nil {
		t.Fatalf("GetPreference() error: %v", err)
	}
	if len(got) != len(ids) {
		t.Fatalf("got length %d, want %d", len(got), len(ids))
	}
	for i, v := range got {
		if v != ids[i] {
			t.Errorf("got[%d] = %d, want %d", i, v, ids[i])
		}
	}
}

func TestPreferences_SetOverwrites(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	if err := store.SetPreference(ctx, "key", "v1"); err != nil {
		t.Fatalf("first SetPreference() error: %v", err)
	}
	if err := store.SetPreference(ctx, "key", "v2"); err != nil {
		t.Fatalf("second SetPreference() error: %v", err)
	}

	var got string
	if err := store.GetPreference(ctx, "key", &got); err != nil {
		t.Fatalf("GetPreference() error: %v", err)
	}
	if got != "v2" {
		t.Errorf("got %q, want %q", got, "v2")
	}
}

func TestPreferences_GetNotFound(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	var got string
	err := store.GetPreference(ctx, "nonexistent", &got)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestGetAllPreferences(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	if err := store.SetPreference(ctx, "a", "alpha"); err != nil {
		t.Fatalf("SetPreference(a) error: %v", err)
	}
	if err := store.SetPreference(ctx, "b", 99); err != nil {
		t.Fatalf("SetPreference(b) error: %v", err)
	}

	prefs, err := store.GetAllPreferences(ctx)
	if err != nil {
		t.Fatalf("GetAllPreferences() error: %v", err)
	}
	if len(prefs) != 2 {
		t.Fatalf("got %d preferences, want 2", len(prefs))
	}
	if string(prefs["a"]) != `"alpha"` {
		t.Errorf("prefs[a] = %s, want %q", prefs["a"], `"alpha"`)
	}
	if string(prefs["b"]) != `99` {
		t.Errorf("prefs[b] = %s, want %q", prefs["b"], `99`)
	}
}

func TestGetAllPreferences_Empty(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	prefs, err := store.GetAllPreferences(ctx)
	if err != nil {
		t.Fatalf("GetAllPreferences() error: %v", err)
	}
	if len(prefs) != 0 {
		t.Errorf("got %d preferences, want 0", len(prefs))
	}
}
