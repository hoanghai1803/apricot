package storage

import (
	"context"
	"testing"
	"time"

	"github.com/hoanghai1803/apricot/internal/models"
)

func seedSearchBlog(t *testing.T, store *Store, title, description, url string) int64 {
	t.Helper()
	ctx := context.Background()

	var sourceID int64
	err := store.db.QueryRow(`SELECT id FROM blog_sources LIMIT 1`).Scan(&sourceID)
	if err != nil {
		sourceID = seedTestSource(t, store)
	}

	blog := &models.Blog{
		SourceID:    sourceID,
		Title:       title,
		URL:         url,
		Description: description,
		FetchedAt:   time.Now().Truncate(time.Second),
	}
	id, err := store.UpsertBlog(ctx, blog)
	if err != nil {
		t.Fatalf("seeding search blog: %v", err)
	}
	return id
}

func TestSearchBlogs_ByTitle(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	seedSearchBlog(t, store, "Building Distributed Systems at Scale", "How we built our distributed infrastructure", "https://test.com/distributed")
	seedSearchBlog(t, store, "Introduction to Machine Learning", "ML basics for engineers", "https://test.com/ml")
	seedSearchBlog(t, store, "Go Concurrency Patterns", "Advanced Go patterns", "https://test.com/go")

	results, err := store.SearchBlogs(ctx, "distributed", 10)
	if err != nil {
		t.Fatalf("SearchBlogs() error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].Title != "Building Distributed Systems at Scale" {
		t.Errorf("title = %q, want %q", results[0].Title, "Building Distributed Systems at Scale")
	}
}

func TestSearchBlogs_ByDescription(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	seedSearchBlog(t, store, "Some Title", "Kubernetes orchestration patterns", "https://test.com/k8s")
	seedSearchBlog(t, store, "Another Title", "React component testing", "https://test.com/react")

	results, err := store.SearchBlogs(ctx, "kubernetes", 10)
	if err != nil {
		t.Fatalf("SearchBlogs() error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
}

func TestSearchBlogs_EmptyQuery(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	results, err := store.SearchBlogs(ctx, "", 10)
	if err != nil {
		t.Fatalf("SearchBlogs() error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("got %d results, want 0 for empty query", len(results))
	}
}

func TestSearchBlogs_NoResults(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	seedSearchBlog(t, store, "Go Patterns", "Concurrency in Go", "https://test.com/go2")

	results, err := store.SearchBlogs(ctx, "python", 10)
	if err != nil {
		t.Fatalf("SearchBlogs() error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("got %d results, want 0", len(results))
	}
}

func TestSearchBlogs_IncludesSourceName(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	seedSearchBlog(t, store, "Unique Test Blog", "Unique description", "https://test.com/unique")

	results, err := store.SearchBlogs(ctx, "unique", 10)
	if err != nil {
		t.Fatalf("SearchBlogs() error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].Source == "" {
		t.Error("expected non-empty Source name")
	}
}

func TestSearchBlogs_Limit(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		seedSearchBlog(t, store, "Microservices Architecture Part", "Microservices patterns",
			"https://test.com/micro-"+string(rune('a'+i)))
	}

	results, err := store.SearchBlogs(ctx, "microservices", 2)
	if err != nil {
		t.Fatalf("SearchBlogs() error: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("got %d results, want 2 (limited)", len(results))
	}
}
