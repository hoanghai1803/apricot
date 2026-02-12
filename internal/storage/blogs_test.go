package storage

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hoanghai1803/apricot/internal/models"
)

// seedTestSource inserts a blog source for use in blog tests and returns its ID.
func seedTestSource(t *testing.T, store *Store) int64 {
	t.Helper()
	res, err := store.db.Exec(
		`INSERT INTO blog_sources (name, company, feed_url, site_url, is_active)
		 VALUES ('Test Blog', 'TestCo', 'https://test.com/feed', 'https://test.com', 1)`)
	if err != nil {
		t.Fatalf("seeding test source: %v", err)
	}
	id, _ := res.LastInsertId()
	return id
}

func TestUpsertBlog_CreatesNew(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	sourceID := seedTestSource(t, store)

	now := time.Now().Truncate(time.Second)
	blog := &models.Blog{
		SourceID:    sourceID,
		Title:       "Test Post",
		URL:         "https://test.com/post-1",
		Description: "A test post",
		FetchedAt:   now,
	}

	id, err := store.UpsertBlog(ctx, blog)
	if err != nil {
		t.Fatalf("UpsertBlog() error: %v", err)
	}
	if id == 0 {
		t.Fatal("UpsertBlog() returned id 0")
	}

	got, err := store.GetBlogByID(ctx, id)
	if err != nil {
		t.Fatalf("GetBlogByID() error: %v", err)
	}
	if got.Title != "Test Post" {
		t.Errorf("Title = %q, want %q", got.Title, "Test Post")
	}
	if got.URL != "https://test.com/post-1" {
		t.Errorf("URL = %q, want %q", got.URL, "https://test.com/post-1")
	}
	if got.Description != "A test post" {
		t.Errorf("Description = %q, want %q", got.Description, "A test post")
	}
}

func TestUpsertBlog_UpdatesExisting(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	sourceID := seedTestSource(t, store)

	now := time.Now().Truncate(time.Second)
	blog := &models.Blog{
		SourceID:  sourceID,
		Title:     "Original Title",
		URL:       "https://test.com/post-update",
		FetchedAt: now,
	}

	id1, err := store.UpsertBlog(ctx, blog)
	if err != nil {
		t.Fatalf("first UpsertBlog() error: %v", err)
	}

	// Update with new content via same URL.
	later := now.Add(time.Hour)
	blog.FullContent = "Updated full content"
	blog.ContentHash = "abc123"
	blog.FetchedAt = later

	id2, err := store.UpsertBlog(ctx, blog)
	if err != nil {
		t.Fatalf("second UpsertBlog() error: %v", err)
	}
	if id1 != id2 {
		t.Errorf("expected same ID on upsert, got %d and %d", id1, id2)
	}

	got, err := store.GetBlogByURL(ctx, "https://test.com/post-update")
	if err != nil {
		t.Fatalf("GetBlogByURL() error: %v", err)
	}
	if got.FullContent != "Updated full content" {
		t.Errorf("FullContent = %q, want %q", got.FullContent, "Updated full content")
	}
	if got.ContentHash != "abc123" {
		t.Errorf("ContentHash = %q, want %q", got.ContentHash, "abc123")
	}
}

func TestGetBlogByURL_NotFound(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	_, err := store.GetBlogByURL(ctx, "https://nonexistent.com/post")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestGetBlogByID_NotFound(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	_, err := store.GetBlogByID(ctx, 99999)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestGetBlogByURL_Found(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	sourceID := seedTestSource(t, store)

	pub := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	blog := &models.Blog{
		SourceID:    sourceID,
		Title:       "Found Post",
		URL:         "https://test.com/found",
		Description: "desc",
		PublishedAt: &pub,
		FetchedAt:   time.Now().Truncate(time.Second),
	}

	if _, err := store.UpsertBlog(ctx, blog); err != nil {
		t.Fatalf("UpsertBlog() error: %v", err)
	}

	got, err := store.GetBlogByURL(ctx, "https://test.com/found")
	if err != nil {
		t.Fatalf("GetBlogByURL() error: %v", err)
	}
	if got.Title != "Found Post" {
		t.Errorf("Title = %q, want %q", got.Title, "Found Post")
	}
	if got.PublishedAt == nil {
		t.Fatal("PublishedAt is nil, expected non-nil")
	}
	if got.PublishedAt.Format("2006-01-02") != "2025-01-15" {
		t.Errorf("PublishedAt = %v, want 2025-01-15", got.PublishedAt)
	}
	if got.Source != "Test Blog" {
		t.Errorf("Source = %q, want %q", got.Source, "Test Blog")
	}
}

func TestGetBlogByID_Found(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	sourceID := seedTestSource(t, store)

	blog := &models.Blog{
		SourceID:  sourceID,
		Title:     "ID Lookup",
		URL:       "https://test.com/id-lookup",
		FetchedAt: time.Now().Truncate(time.Second),
	}

	id, err := store.UpsertBlog(ctx, blog)
	if err != nil {
		t.Fatalf("UpsertBlog() error: %v", err)
	}

	got, err := store.GetBlogByID(ctx, id)
	if err != nil {
		t.Fatalf("GetBlogByID() error: %v", err)
	}
	if got.Title != "ID Lookup" {
		t.Errorf("Title = %q, want %q", got.Title, "ID Lookup")
	}
}

func TestSaveBlogs_Batch(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	sourceID := seedTestSource(t, store)

	now := time.Now().Truncate(time.Second)
	blogs := []models.Blog{
		{SourceID: sourceID, Title: "Batch 1", URL: "https://test.com/batch-1", FetchedAt: now},
		{SourceID: sourceID, Title: "Batch 2", URL: "https://test.com/batch-2", FetchedAt: now},
		{SourceID: sourceID, Title: "Batch 3", URL: "https://test.com/batch-3", FetchedAt: now},
	}

	if err := store.SaveBlogs(ctx, blogs); err != nil {
		t.Fatalf("SaveBlogs() error: %v", err)
	}

	// Verify all three were inserted.
	for _, b := range blogs {
		got, err := store.GetBlogByURL(ctx, b.URL)
		if err != nil {
			t.Fatalf("GetBlogByURL(%q) error: %v", b.URL, err)
		}
		if got.Title != b.Title {
			t.Errorf("Title = %q, want %q", got.Title, b.Title)
		}
	}

	// Upsert with updated content.
	blogs[0].FullContent = "updated content"
	if err := store.SaveBlogs(ctx, blogs); err != nil {
		t.Fatalf("SaveBlogs() upsert error: %v", err)
	}

	got, err := store.GetBlogByURL(ctx, "https://test.com/batch-1")
	if err != nil {
		t.Fatalf("GetBlogByURL() error: %v", err)
	}
	if got.FullContent != "updated content" {
		t.Errorf("FullContent = %q, want %q", got.FullContent, "updated content")
	}
}
