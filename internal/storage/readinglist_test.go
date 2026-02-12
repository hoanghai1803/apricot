package storage

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/hoanghai1803/apricot/internal/models"
)

// seedReadingListBlog creates a source and blog suitable for reading list tests.
func seedReadingListBlog(t *testing.T, store *Store, url string) int64 {
	t.Helper()
	ctx := context.Background()

	// Reuse existing source if possible; create one otherwise.
	var sourceID int64
	err := store.db.QueryRow(`SELECT id FROM blog_sources LIMIT 1`).Scan(&sourceID)
	if err != nil {
		sourceID = seedTestSource(t, store)
	}

	blog := &models.Blog{
		SourceID:  sourceID,
		Title:     "RL Post: " + url,
		URL:       url,
		FetchedAt: time.Now().Truncate(time.Second),
	}
	id, err := store.UpsertBlog(ctx, blog)
	if err != nil {
		t.Fatalf("seeding reading list blog: %v", err)
	}
	return id
}

func TestAddToReadingList(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	blogID := seedReadingListBlog(t, store, "https://test.com/rl-add")

	if err := store.AddToReadingList(ctx, blogID); err != nil {
		t.Fatalf("AddToReadingList() error: %v", err)
	}

	items, err := store.GetReadingList(ctx, "")
	if err != nil {
		t.Fatalf("GetReadingList() error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("got %d items, want 1", len(items))
	}
	if items[0].BlogID != blogID {
		t.Errorf("BlogID = %d, want %d", items[0].BlogID, blogID)
	}
	if items[0].Status != "unread" {
		t.Errorf("Status = %q, want %q", items[0].Status, "unread")
	}
	if items[0].Blog == nil {
		t.Fatal("Blog is nil, expected non-nil")
	}
	if items[0].Blog.URL != "https://test.com/rl-add" {
		t.Errorf("Blog.URL = %q, want %q", items[0].Blog.URL, "https://test.com/rl-add")
	}
}

func TestAddToReadingList_DuplicateError(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	blogID := seedReadingListBlog(t, store, "https://test.com/rl-dup")

	if err := store.AddToReadingList(ctx, blogID); err != nil {
		t.Fatalf("first AddToReadingList() error: %v", err)
	}

	err := store.AddToReadingList(ctx, blogID)
	if err == nil {
		t.Fatal("expected error for duplicate, got nil")
	}
	if !strings.Contains(err.Error(), "already on the reading list") {
		t.Errorf("expected 'already on the reading list' error, got: %v", err)
	}
}

func TestAddToReadingList_NonExistentBlog(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	err := store.AddToReadingList(ctx, 99999)
	if err == nil {
		t.Fatal("expected error for non-existent blog, got nil")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("expected 'does not exist' error, got: %v", err)
	}
}

func TestGetReadingList_FilterByStatus(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	blog1 := seedReadingListBlog(t, store, "https://test.com/rl-s1")
	blog2 := seedReadingListBlog(t, store, "https://test.com/rl-s2")

	if err := store.AddToReadingList(ctx, blog1); err != nil {
		t.Fatalf("AddToReadingList(1) error: %v", err)
	}
	if err := store.AddToReadingList(ctx, blog2); err != nil {
		t.Fatalf("AddToReadingList(2) error: %v", err)
	}

	// Get the reading list item ID for blog1 so we can update its status.
	items, _ := store.GetReadingList(ctx, "")
	var blog1ItemID int64
	for _, item := range items {
		if item.BlogID == blog1 {
			blog1ItemID = item.ID
			break
		}
	}

	if err := store.UpdateReadingListStatus(ctx, blog1ItemID, "reading"); err != nil {
		t.Fatalf("UpdateReadingListStatus() error: %v", err)
	}

	// Filter by "unread".
	unread, err := store.GetReadingList(ctx, "unread")
	if err != nil {
		t.Fatalf("GetReadingList(unread) error: %v", err)
	}
	if len(unread) != 1 {
		t.Errorf("unread count = %d, want 1", len(unread))
	}

	// Filter by "reading".
	reading, err := store.GetReadingList(ctx, "reading")
	if err != nil {
		t.Fatalf("GetReadingList(reading) error: %v", err)
	}
	if len(reading) != 1 {
		t.Errorf("reading count = %d, want 1", len(reading))
	}

	// All items.
	all, err := store.GetReadingList(ctx, "")
	if err != nil {
		t.Fatalf("GetReadingList('') error: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("all count = %d, want 2", len(all))
	}
}

func TestUpdateReadingListStatus_Transitions(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	blogID := seedReadingListBlog(t, store, "https://test.com/rl-transitions")

	if err := store.AddToReadingList(ctx, blogID); err != nil {
		t.Fatalf("AddToReadingList() error: %v", err)
	}

	items, _ := store.GetReadingList(ctx, "")
	itemID := items[0].ID

	// Transition to "reading".
	if err := store.UpdateReadingListStatus(ctx, itemID, "reading"); err != nil {
		t.Fatalf("UpdateReadingListStatus(reading) error: %v", err)
	}
	items, _ = store.GetReadingList(ctx, "")
	if items[0].Status != "reading" {
		t.Errorf("Status = %q, want %q", items[0].Status, "reading")
	}
	if items[0].ReadAt != nil {
		t.Error("ReadAt should be nil for 'reading' status")
	}

	// Transition to "read" -- should set read_at.
	if err := store.UpdateReadingListStatus(ctx, itemID, "read"); err != nil {
		t.Fatalf("UpdateReadingListStatus(read) error: %v", err)
	}
	items, _ = store.GetReadingList(ctx, "")
	if items[0].Status != "read" {
		t.Errorf("Status = %q, want %q", items[0].Status, "read")
	}
	if items[0].ReadAt == nil {
		t.Error("ReadAt should be set for 'read' status")
	}

	// Transition back to "unread" -- should clear read_at.
	if err := store.UpdateReadingListStatus(ctx, itemID, "unread"); err != nil {
		t.Fatalf("UpdateReadingListStatus(unread) error: %v", err)
	}
	items, _ = store.GetReadingList(ctx, "")
	if items[0].Status != "unread" {
		t.Errorf("Status = %q, want %q", items[0].Status, "unread")
	}
	if items[0].ReadAt != nil {
		t.Error("ReadAt should be nil after reverting to 'unread'")
	}
}

func TestUpdateReadingListStatus_InvalidStatus(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	err := store.UpdateReadingListStatus(ctx, 1, "invalid")
	if err == nil {
		t.Fatal("expected error for invalid status, got nil")
	}
	if !strings.Contains(err.Error(), "invalid reading list status") {
		t.Errorf("expected 'invalid reading list status' error, got: %v", err)
	}
}

func TestUpdateReadingListStatus_NotFound(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	err := store.UpdateReadingListStatus(ctx, 99999, "read")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestUpdateReadingListNotes(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	blogID := seedReadingListBlog(t, store, "https://test.com/rl-notes")

	if err := store.AddToReadingList(ctx, blogID); err != nil {
		t.Fatalf("AddToReadingList() error: %v", err)
	}

	items, _ := store.GetReadingList(ctx, "")
	itemID := items[0].ID

	if err := store.UpdateReadingListNotes(ctx, itemID, "Great article!"); err != nil {
		t.Fatalf("UpdateReadingListNotes() error: %v", err)
	}

	items, _ = store.GetReadingList(ctx, "")
	if items[0].Notes == nil || *items[0].Notes != "Great article!" {
		t.Errorf("Notes = %v, want %q", items[0].Notes, "Great article!")
	}
}

func TestUpdateReadingListNotes_NotFound(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	err := store.UpdateReadingListNotes(ctx, 99999, "test")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestRemoveFromReadingList(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	blogID := seedReadingListBlog(t, store, "https://test.com/rl-remove")

	if err := store.AddToReadingList(ctx, blogID); err != nil {
		t.Fatalf("AddToReadingList() error: %v", err)
	}

	items, _ := store.GetReadingList(ctx, "")
	if len(items) != 1 {
		t.Fatalf("expected 1 item before removal, got %d", len(items))
	}
	itemID := items[0].ID

	if err := store.RemoveFromReadingList(ctx, itemID); err != nil {
		t.Fatalf("RemoveFromReadingList() error: %v", err)
	}

	items, _ = store.GetReadingList(ctx, "")
	if len(items) != 0 {
		t.Errorf("expected 0 items after removal, got %d", len(items))
	}
}

func TestRemoveFromReadingList_NotFound(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	err := store.RemoveFromReadingList(ctx, 99999)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestGetReadingList_WithSummary(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	blogID := seedReadingListBlog(t, store, "https://test.com/rl-summary")

	// Add summary for the blog.
	summary := &models.BlogSummary{
		BlogID:    blogID,
		Summary:   "Test summary for reading list.",
		ModelUsed: "test-model",
	}
	if err := store.UpsertSummary(ctx, summary); err != nil {
		t.Fatalf("UpsertSummary() error: %v", err)
	}

	if err := store.AddToReadingList(ctx, blogID); err != nil {
		t.Fatalf("AddToReadingList() error: %v", err)
	}

	items, err := store.GetReadingList(ctx, "")
	if err != nil {
		t.Fatalf("GetReadingList() error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("got %d items, want 1", len(items))
	}
	if items[0].Summary == nil {
		t.Fatal("Summary is nil, expected non-nil")
	}
	if *items[0].Summary != "Test summary for reading list." {
		t.Errorf("Summary = %q, want %q", *items[0].Summary, "Test summary for reading list.")
	}
}
