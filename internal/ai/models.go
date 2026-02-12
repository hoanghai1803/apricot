package ai

// ProviderConfig holds the configuration needed to create an AI provider.
type ProviderConfig struct {
	Provider string // "anthropic" | "openai"
	APIKey   string
	Model    string
}

// BlogEntry is a simplified blog representation for AI prompts.
type BlogEntry struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	Source      string `json:"source"`
	PublishedAt string `json:"published_at"`
	Description string `json:"description"`
	FullContent string `json:"full_content,omitempty"`
}

// RankedBlog is a single result from the filter-and-rank operation.
type RankedBlog struct {
	ID     int64  `json:"id"`
	Reason string `json:"reason"`
}
