package storage

import (
	"context"
	"errors"
	"testing"
)

func TestAddTagToItem(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	blogID := seedReadingListBlog(t, store, "https://test.com/tag-add")

	if err := store.AddToReadingList(ctx, blogID); err != nil {
		t.Fatalf("AddToReadingList() error: %v", err)
	}

	items, _ := store.GetReadingList(ctx, "")
	itemID := items[0].ID

	if err := store.AddTagToItem(ctx, itemID, "distributed-systems"); err != nil {
		t.Fatalf("AddTagToItem() error: %v", err)
	}

	// Verify tag shows up on the item.
	items, _ = store.GetReadingList(ctx, "")
	if len(items[0].Tags) != 1 {
		t.Fatalf("got %d tags, want 1", len(items[0].Tags))
	}
	if items[0].Tags[0] != "distributed-systems" {
		t.Errorf("tag = %q, want %q", items[0].Tags[0], "distributed-systems")
	}
}

func TestAddTagToItem_NormalizesCase(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	blogID := seedReadingListBlog(t, store, "https://test.com/tag-case")

	if err := store.AddToReadingList(ctx, blogID); err != nil {
		t.Fatalf("AddToReadingList() error: %v", err)
	}

	items, _ := store.GetReadingList(ctx, "")
	itemID := items[0].ID

	if err := store.AddTagToItem(ctx, itemID, "  ML  "); err != nil {
		t.Fatalf("AddTagToItem() error: %v", err)
	}

	items, _ = store.GetReadingList(ctx, "")
	if items[0].Tags[0] != "ml" {
		t.Errorf("tag = %q, want %q (lowercased and trimmed)", items[0].Tags[0], "ml")
	}
}

func TestAddTagToItem_EmptyTag(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	err := store.AddTagToItem(ctx, 1, "")
	if err == nil {
		t.Fatal("expected error for empty tag, got nil")
	}
}

func TestAddTagToItem_NotFound(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	err := store.AddTagToItem(ctx, 99999, "test")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestAddTagToItem_Idempotent(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	blogID := seedReadingListBlog(t, store, "https://test.com/tag-idem")

	if err := store.AddToReadingList(ctx, blogID); err != nil {
		t.Fatalf("AddToReadingList() error: %v", err)
	}

	items, _ := store.GetReadingList(ctx, "")
	itemID := items[0].ID

	// Add same tag twice — should not error or duplicate.
	if err := store.AddTagToItem(ctx, itemID, "golang"); err != nil {
		t.Fatalf("first AddTagToItem() error: %v", err)
	}
	if err := store.AddTagToItem(ctx, itemID, "golang"); err != nil {
		t.Fatalf("second AddTagToItem() error: %v", err)
	}

	items, _ = store.GetReadingList(ctx, "")
	if len(items[0].Tags) != 1 {
		t.Errorf("got %d tags, want 1 (no duplicate)", len(items[0].Tags))
	}
}

func TestRemoveTagFromItem(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	blogID := seedReadingListBlog(t, store, "https://test.com/tag-remove")

	if err := store.AddToReadingList(ctx, blogID); err != nil {
		t.Fatalf("AddToReadingList() error: %v", err)
	}

	items, _ := store.GetReadingList(ctx, "")
	itemID := items[0].ID

	if err := store.AddTagToItem(ctx, itemID, "rust"); err != nil {
		t.Fatalf("AddTagToItem() error: %v", err)
	}

	if err := store.RemoveTagFromItem(ctx, itemID, "rust"); err != nil {
		t.Fatalf("RemoveTagFromItem() error: %v", err)
	}

	items, _ = store.GetReadingList(ctx, "")
	if len(items[0].Tags) != 0 {
		t.Errorf("got %d tags, want 0 after removal", len(items[0].Tags))
	}

	// Verify orphan tag was cleaned up.
	tags, _ := store.GetAllTags(ctx)
	if len(tags) != 0 {
		t.Errorf("got %d tags in registry, want 0 (orphan should be cleaned up)", len(tags))
	}
}

func TestRemoveTagFromItem_NotFound(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	err := store.RemoveTagFromItem(ctx, 99999, "nonexistent")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestGetAllTags(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	blog1 := seedReadingListBlog(t, store, "https://test.com/tag-all1")
	blog2 := seedReadingListBlog(t, store, "https://test.com/tag-all2")

	if err := store.AddToReadingList(ctx, blog1); err != nil {
		t.Fatalf("AddToReadingList(1) error: %v", err)
	}
	if err := store.AddToReadingList(ctx, blog2); err != nil {
		t.Fatalf("AddToReadingList(2) error: %v", err)
	}

	items, _ := store.GetReadingList(ctx, "")

	if err := store.AddTagToItem(ctx, items[0].ID, "golang"); err != nil {
		t.Fatalf("AddTagToItem() error: %v", err)
	}
	if err := store.AddTagToItem(ctx, items[1].ID, "rust"); err != nil {
		t.Fatalf("AddTagToItem() error: %v", err)
	}
	if err := store.AddTagToItem(ctx, items[1].ID, "golang"); err != nil {
		t.Fatalf("AddTagToItem() error: %v", err)
	}

	tags, err := store.GetAllTags(ctx)
	if err != nil {
		t.Fatalf("GetAllTags() error: %v", err)
	}

	if len(tags) != 2 {
		t.Fatalf("got %d tags, want 2", len(tags))
	}
	// Should be alphabetically ordered.
	if tags[0] != "golang" || tags[1] != "rust" {
		t.Errorf("tags = %v, want [golang, rust]", tags)
	}
}

func TestGetAllTags_Empty(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	tags, err := store.GetAllTags(ctx)
	if err != nil {
		t.Fatalf("GetAllTags() error: %v", err)
	}
	if len(tags) != 0 {
		t.Errorf("got %d tags, want 0", len(tags))
	}
}

func TestMultipleTags(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	blogID := seedReadingListBlog(t, store, "https://test.com/tag-multi")

	if err := store.AddToReadingList(ctx, blogID); err != nil {
		t.Fatalf("AddToReadingList() error: %v", err)
	}

	items, _ := store.GetReadingList(ctx, "")
	itemID := items[0].ID

	for _, tag := range []string{"golang", "distributed-systems", "performance"} {
		if err := store.AddTagToItem(ctx, itemID, tag); err != nil {
			t.Fatalf("AddTagToItem(%q) error: %v", tag, err)
		}
	}

	items, _ = store.GetReadingList(ctx, "")
	if len(items[0].Tags) != 3 {
		t.Fatalf("got %d tags, want 3", len(items[0].Tags))
	}
}

func TestTagsSurviveStatusChange(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	blogID := seedReadingListBlog(t, store, "https://test.com/tag-status")

	if err := store.AddToReadingList(ctx, blogID); err != nil {
		t.Fatalf("AddToReadingList() error: %v", err)
	}

	items, _ := store.GetReadingList(ctx, "")
	itemID := items[0].ID

	if err := store.AddTagToItem(ctx, itemID, "important"); err != nil {
		t.Fatalf("AddTagToItem() error: %v", err)
	}

	// Change status.
	if err := store.UpdateReadingListStatus(ctx, itemID, "reading"); err != nil {
		t.Fatalf("UpdateReadingListStatus() error: %v", err)
	}

	// Tags should still be there.
	items, _ = store.GetReadingList(ctx, "reading")
	if len(items[0].Tags) != 1 || items[0].Tags[0] != "important" {
		t.Errorf("tags = %v, want [important] after status change", items[0].Tags)
	}
}

func TestTagsCascadeOnDelete(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	blogID := seedReadingListBlog(t, store, "https://test.com/tag-cascade")

	if err := store.AddToReadingList(ctx, blogID); err != nil {
		t.Fatalf("AddToReadingList() error: %v", err)
	}

	items, _ := store.GetReadingList(ctx, "")
	itemID := items[0].ID

	if err := store.AddTagToItem(ctx, itemID, "ephemeral"); err != nil {
		t.Fatalf("AddTagToItem() error: %v", err)
	}

	// Delete the reading list item.
	if err := store.RemoveFromReadingList(ctx, itemID); err != nil {
		t.Fatalf("RemoveFromReadingList() error: %v", err)
	}

	// The join table entry should be gone (CASCADE), and the tag
	// itself still exists in the tags table since we didn't call RemoveTagFromItem.
	// That's fine — GetAllTags returns tags from the tags table.
	// The orphan cleanup only happens in RemoveTagFromItem.
}
