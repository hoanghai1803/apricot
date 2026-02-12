package feeds

import (
	"testing"

	"github.com/hoanghai1803/apricot/internal/models"
)

func TestIsScrapeURL(t *testing.T) {
	tests := []struct {
		url  string
		want bool
	}{
		{"scrape://www.linkedin.com/blog/engineering", true},
		{"https://example.com/feed", false},
		{"scrape://", true},
		{"", false},
	}
	for _, tt := range tests {
		if got := IsScrapeURL(tt.url); got != tt.want {
			t.Errorf("IsScrapeURL(%q) = %v, want %v", tt.url, got, tt.want)
		}
	}
}

func TestScrapeURLToHTTPS(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"scrape://www.linkedin.com/blog/engineering", "https://www.linkedin.com/blog/engineering"},
		{"scrape://example.com", "https://example.com"},
	}
	for _, tt := range tests {
		if got := ScrapeURLToHTTPS(tt.input); got != tt.want {
			t.Errorf("ScrapeURLToHTTPS(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestParseLinkedInHTML(t *testing.T) {
	source := models.BlogSource{ID: 1, Name: "LinkedIn Engineering"}

	html := `
	<html><body>
		<section class="featured-post">
			<div class="post-container">
				<a class="featured-post__headline" href="/blog/engineering/featured-post">Featured Article Title</a>
				<p class="featured-post__date">Feb 10, 2026</p>
			</div>
		</section>
		<ul>
			<li class="post-list__item">
				<a class="grid-post__link" href="/blog/engineering/post-one">First Post Title</a>
				<p class="grid-post__date">Jan 29, 2026</p>
			</li>
			<li class="post-list__item">
				<a class="grid-post__link" href="/blog/engineering/post-two">Second Post Title</a>
				<p class="grid-post__date">Jan 15, 2026</p>
			</li>
			<li class="post-list__item">
				<a class="grid-post__link" href="">Empty URL Post</a>
				<p class="grid-post__date">Jan 10, 2026</p>
			</li>
		</ul>
	</body></html>`

	blogs, err := parseLinkedInHTML(source, html, 10)
	if err != nil {
		t.Fatalf("parseLinkedInHTML error: %v", err)
	}

	if len(blogs) != 3 {
		t.Fatalf("got %d blogs, want 3", len(blogs))
	}

	// Check featured post.
	if blogs[0].Title != "Featured Article Title" {
		t.Errorf("blogs[0].Title = %q, want %q", blogs[0].Title, "Featured Article Title")
	}
	if blogs[0].URL != "https://www.linkedin.com/blog/engineering/featured-post" {
		t.Errorf("blogs[0].URL = %q", blogs[0].URL)
	}
	if blogs[0].PublishedAt == nil {
		t.Error("blogs[0].PublishedAt should not be nil")
	}

	// Check regular posts.
	if blogs[1].Title != "First Post Title" {
		t.Errorf("blogs[1].Title = %q, want %q", blogs[1].Title, "First Post Title")
	}
	if blogs[2].Title != "Second Post Title" {
		t.Errorf("blogs[2].Title = %q, want %q", blogs[2].Title, "Second Post Title")
	}

	// Check source info is set.
	for _, b := range blogs {
		if b.SourceID != 1 {
			t.Errorf("SourceID = %d, want 1", b.SourceID)
		}
		if b.Source != "LinkedIn Engineering" {
			t.Errorf("Source = %q, want %q", b.Source, "LinkedIn Engineering")
		}
		if b.ContentHash == "" {
			t.Error("ContentHash should not be empty")
		}
	}
}

func TestParseLinkedInHTML_MaxArticles(t *testing.T) {
	source := models.BlogSource{ID: 1, Name: "Test"}

	html := `
	<html><body>
		<ul>
			<li><a class="grid-post__link" href="/post/1">Post 1</a></li>
			<li><a class="grid-post__link" href="/post/2">Post 2</a></li>
			<li><a class="grid-post__link" href="/post/3">Post 3</a></li>
		</ul>
	</body></html>`

	blogs, err := parseLinkedInHTML(source, html, 2)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(blogs) != 2 {
		t.Errorf("got %d blogs, want 2 (limited by maxArticles)", len(blogs))
	}
}

func TestParseHumanDate(t *testing.T) {
	tests := []struct {
		input   string
		wantNil bool
		wantDay int
	}{
		{"Jan 29, 2026", false, 29},
		{"February 5, 2026", false, 5},
		{"2026-01-15", false, 15},
		{"not a date", true, 0},
		{"", true, 0},
	}
	for _, tt := range tests {
		got := parseHumanDate(tt.input)
		if tt.wantNil && got != nil {
			t.Errorf("parseHumanDate(%q) = %v, want nil", tt.input, got)
		}
		if !tt.wantNil && got == nil {
			t.Errorf("parseHumanDate(%q) = nil, want day %d", tt.input, tt.wantDay)
		}
		if !tt.wantNil && got != nil && got.Day() != tt.wantDay {
			t.Errorf("parseHumanDate(%q).Day() = %d, want %d", tt.input, got.Day(), tt.wantDay)
		}
	}
}
