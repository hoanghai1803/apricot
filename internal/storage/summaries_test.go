package storage

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hoanghai1803/apricot/internal/models"
)

// seedTestBlog creates a source and a blog for use in summary tests, returning the blog ID.
func seedTestBlog(t *testing.T, store *Store) int64 {
	t.Helper()
	sourceID := seedTestSource(t, store)
	blog := &models.Blog{
		SourceID:  sourceID,
		Title:     "Summary Test Blog",
		URL:       "https://test.com/summary-test",
		FetchedAt: time.Now().Truncate(time.Second),
	}
	id, err := store.UpsertBlog(context.Background(), blog)
	if err != nil {
		t.Fatalf("seeding test blog: %v", err)
	}
	return id
}

func TestUpsertSummary_CreatesNew(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	blogID := seedTestBlog(t, store)

	summary := &models.BlogSummary{
		BlogID:    blogID,
		Summary:   "This is a test summary.",
		ModelUsed: "claude-haiku-4-5",
	}

	if err := store.UpsertSummary(ctx, summary); err != nil {
		t.Fatalf("UpsertSummary() error: %v", err)
	}

	got, err := store.GetSummaryByBlogID(ctx, blogID)
	if err != nil {
		t.Fatalf("GetSummaryByBlogID() error: %v", err)
	}
	if got.Summary != "This is a test summary." {
		t.Errorf("Summary = %q, want %q", got.Summary, "This is a test summary.")
	}
	if got.ModelUsed != "claude-haiku-4-5" {
		t.Errorf("ModelUsed = %q, want %q", got.ModelUsed, "claude-haiku-4-5")
	}
	if got.BlogID != blogID {
		t.Errorf("BlogID = %d, want %d", got.BlogID, blogID)
	}
}

func TestUpsertSummary_UpdatesExisting(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	blogID := seedTestBlog(t, store)

	original := &models.BlogSummary{
		BlogID:    blogID,
		Summary:   "Original summary.",
		ModelUsed: "gpt-4o",
	}
	if err := store.UpsertSummary(ctx, original); err != nil {
		t.Fatalf("first UpsertSummary() error: %v", err)
	}

	updated := &models.BlogSummary{
		BlogID:    blogID,
		Summary:   "Updated summary.",
		ModelUsed: "claude-haiku-4-5",
	}
	if err := store.UpsertSummary(ctx, updated); err != nil {
		t.Fatalf("second UpsertSummary() error: %v", err)
	}

	got, err := store.GetSummaryByBlogID(ctx, blogID)
	if err != nil {
		t.Fatalf("GetSummaryByBlogID() error: %v", err)
	}
	if got.Summary != "Updated summary." {
		t.Errorf("Summary = %q, want %q", got.Summary, "Updated summary.")
	}
	if got.ModelUsed != "claude-haiku-4-5" {
		t.Errorf("ModelUsed = %q, want %q", got.ModelUsed, "claude-haiku-4-5")
	}
}

func TestGetSummaryByBlogID_NotFound(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	_, err := store.GetSummaryByBlogID(ctx, 99999)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestHasSummary(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	blogID := seedTestBlog(t, store)

	// No summary yet.
	has, err := store.HasSummary(ctx, blogID)
	if err != nil {
		t.Fatalf("HasSummary() error: %v", err)
	}
	if has {
		t.Error("HasSummary() = true, want false before inserting summary")
	}

	// Insert summary.
	summary := &models.BlogSummary{
		BlogID:    blogID,
		Summary:   "Exists now.",
		ModelUsed: "test-model",
	}
	if err := store.UpsertSummary(ctx, summary); err != nil {
		t.Fatalf("UpsertSummary() error: %v", err)
	}

	has, err = store.HasSummary(ctx, blogID)
	if err != nil {
		t.Fatalf("HasSummary() error: %v", err)
	}
	if !has {
		t.Error("HasSummary() = false, want true after inserting summary")
	}
}

func TestHasSummary_NonExistentBlog(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	has, err := store.HasSummary(ctx, 99999)
	if err != nil {
		t.Fatalf("HasSummary() error: %v", err)
	}
	if has {
		t.Error("HasSummary() = true for non-existent blog, want false")
	}
}
