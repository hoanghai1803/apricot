package models

import "time"

// BlogSource represents an engineering blog we track via RSS.
type BlogSource struct {
	ID          int64      `json:"id"`
	Name        string     `json:"name"`
	Company     string     `json:"company"`
	FeedURL     string     `json:"feed_url"`
	SiteURL     string     `json:"site_url"`
	IsActive    bool       `json:"is_active"`
	LastFetchAt *time.Time `json:"last_fetch_at,omitempty"`
	LastFetchOK bool       `json:"last_fetch_ok"`
	LastError   string     `json:"last_error,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// Blog represents an individual blog post discovered from an RSS feed.
type Blog struct {
	ID          int64      `json:"id"`
	SourceID    int64      `json:"source_id"`
	Source      string     `json:"source,omitempty"`
	Title       string     `json:"title"`
	URL         string     `json:"url"`
	Description string     `json:"description,omitempty"`
	FullContent string     `json:"full_content,omitempty"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
	FetchedAt   time.Time  `json:"fetched_at"`
	ContentHash string     `json:"content_hash,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// BlogSummary holds a cached AI-generated summary for a blog post.
type BlogSummary struct {
	ID        int64     `json:"id"`
	BlogID    int64     `json:"blog_id"`
	Summary   string    `json:"summary"`
	ModelUsed string    `json:"model_used"`
	CreatedAt time.Time `json:"created_at"`
}

// DiscoverySession records an audit trail of each discovery run.
type DiscoverySession struct {
	ID                  int64     `json:"id"`
	PreferencesSnapshot string    `json:"preferences_snapshot"`
	BlogsConsidered     int       `json:"blogs_considered"`
	BlogsSelected       string    `json:"blogs_selected"`
	ModelUsed           string    `json:"model_used"`
	InputTokens         *int      `json:"input_tokens,omitempty"`
	OutputTokens        *int      `json:"output_tokens,omitempty"`
	ResultsJSON         string    `json:"results_json,omitempty"`
	FailedFeedsJSON     string    `json:"failed_feeds_json,omitempty"`
	CreatedAt           time.Time `json:"created_at"`
}
