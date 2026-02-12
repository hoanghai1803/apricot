package feeds

import (
	"testing"
	"time"

	"github.com/hoanghai1803/apricot/internal/models"
	"github.com/mmcdole/gofeed"
)

func TestParseFeedItems(t *testing.T) {
	now := time.Now()
	recentTime := now.Add(-12 * time.Hour)

	source := models.BlogSource{
		ID:   42,
		Name: "Test Blog",
	}

	tests := []struct {
		name        string
		items       []*gofeed.Item
		maxArticles int
		wantCount   int
		desc        string
	}{
		{
			name: "takes up to maxArticles items",
			items: []*gofeed.Item{
				{Title: "Post 1", Link: "https://example.com/1", PublishedParsed: &recentTime},
				{Title: "Post 2", Link: "https://example.com/2", PublishedParsed: &recentTime},
				{Title: "Post 3", Link: "https://example.com/3", PublishedParsed: &recentTime},
			},
			maxArticles: 2,
			wantCount:   2,
			desc:        "should limit to maxArticles items",
		},
		{
			name: "returns all when fewer than maxArticles",
			items: []*gofeed.Item{
				{Title: "Post 1", Link: "https://example.com/1", PublishedParsed: &recentTime},
			},
			maxArticles: 10,
			wantCount:   1,
			desc:        "should return all items when fewer than limit",
		},
		{
			name: "nil published date is included",
			items: []*gofeed.Item{
				{Title: "No Date Post", Link: "https://example.com/nodate", Description: "No date"},
			},
			maxArticles: 10,
			wantCount:   1,
			desc:        "items with nil PublishedParsed should be included",
		},
		{
			name: "empty title is skipped",
			items: []*gofeed.Item{
				{Title: "", Link: "https://example.com/notitle", PublishedParsed: &recentTime},
			},
			maxArticles: 10,
			wantCount:   0,
			desc:        "items with empty title should be skipped",
		},
		{
			name: "empty URL is skipped",
			items: []*gofeed.Item{
				{Title: "No URL Post", Link: "", PublishedParsed: &recentTime},
			},
			maxArticles: 10,
			wantCount:   0,
			desc:        "items with empty URL should be skipped",
		},
		{
			name: "skips invalid items and still respects limit",
			items: []*gofeed.Item{
				{Title: "", Link: "https://example.com/notitle"},         // skipped
				{Title: "Good 1", Link: "https://example.com/good1"},     // counted
				{Title: "Good 2", Link: "https://example.com/good2"},     // counted
				{Title: "Good 3", Link: "https://example.com/good3"},     // not reached (limit=2)
			},
			maxArticles: 2,
			wantCount:   2,
			desc:        "invalid items don't count toward the limit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			feed := &gofeed.Feed{Items: tt.items}
			blogs := parseFeedItems(source, feed, tt.maxArticles)

			if got := len(blogs); got != tt.wantCount {
				t.Errorf("%s: got %d blogs, want %d", tt.desc, got, tt.wantCount)
			}
		})
	}
}

func TestParseFeedItems_FieldMapping(t *testing.T) {
	pubTime := time.Now().Add(-24 * time.Hour)
	source := models.BlogSource{
		ID:   7,
		Name: "Engineering Blog",
	}

	feed := &gofeed.Feed{
		Items: []*gofeed.Item{
			{
				Title:           "Test Article",
				Link:            "https://example.com/article",
				Description:     "A <b>bold</b> description",
				PublishedParsed: &pubTime,
			},
		},
	}

	blogs := parseFeedItems(source, feed, 10)
	if len(blogs) != 1 {
		t.Fatalf("expected 1 blog, got %d", len(blogs))
	}

	blog := blogs[0]

	if blog.Title != "Test Article" {
		t.Errorf("Title = %q, want %q", blog.Title, "Test Article")
	}
	if blog.URL != "https://example.com/article" {
		t.Errorf("URL = %q, want %q", blog.URL, "https://example.com/article")
	}
	if blog.Description != "A bold description" {
		t.Errorf("Description = %q, want %q", blog.Description, "A bold description")
	}
	if blog.SourceID != 7 {
		t.Errorf("SourceID = %d, want %d", blog.SourceID, 7)
	}
	if blog.Source != "Engineering Blog" {
		t.Errorf("Source = %q, want %q", blog.Source, "Engineering Blog")
	}
	if blog.PublishedAt == nil {
		t.Fatal("PublishedAt should not be nil")
	}
	if !blog.PublishedAt.Equal(pubTime) {
		t.Errorf("PublishedAt = %v, want %v", blog.PublishedAt, pubTime)
	}
	if blog.FetchedAt.IsZero() {
		t.Error("FetchedAt should not be zero")
	}
	if blog.ContentHash == "" {
		t.Error("ContentHash should not be empty")
	}
}

func TestComputeHash(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "non-empty string", input: "https://example.com/post"},
		{name: "empty string", input: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h1 := computeHash(tt.input)
			h2 := computeHash(tt.input)

			if h1 != h2 {
				t.Errorf("computeHash not deterministic: %q != %q", h1, h2)
			}
			if len(h1) != 64 {
				t.Errorf("expected 64-char hex string, got %d chars: %q", len(h1), h1)
			}
		})
	}

	if computeHash("a") == computeHash("b") {
		t.Error("different inputs should produce different hashes")
	}
}

func TestStripHTML(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "removes simple tags",
			input: "<p>Hello <b>world</b></p>",
			want:  "Hello world",
		},
		{
			name:  "unescapes HTML entities",
			input: "Tom &amp; Jerry &lt;3",
			want:  "Tom & Jerry <3",
		},
		{
			name:  "combined tags and entities",
			input: "<div>Price: &gt; $10 &amp; &lt; $20</div>",
			want:  "Price: > $10 & < $20",
		},
		{
			name:  "plain text unchanged",
			input: "no tags here",
			want:  "no tags here",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "self-closing tags",
			input: "line one<br/>line two",
			want:  "line oneline two",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripHTML(tt.input)
			if got != tt.want {
				t.Errorf("stripHTML(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
