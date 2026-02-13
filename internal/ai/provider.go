package ai

import (
	"context"
	"fmt"
)

// AIProvider is the interface that all LLM providers must implement.
type AIProvider interface {
	// FilterAndRank selects and ranks blogs based on user preferences.
	// It returns up to maxResults blogs ranked by relevance to the given preferences.
	// When serendipity is true, it deliberately picks posts outside the user's interests.
	FilterAndRank(ctx context.Context, preferences string, blogs []BlogEntry, maxResults int, serendipity bool) ([]RankedBlog, error)

	// Summarize generates a concise summary of the given blog post.
	Summarize(ctx context.Context, blog BlogEntry) (string, error)
}

// NewProvider creates the appropriate provider based on config.
func NewProvider(cfg ProviderConfig) (AIProvider, error) {
	switch cfg.Provider {
	case "anthropic":
		return NewAnthropicProvider(cfg.APIKey, cfg.Model), nil
	case "openai":
		return NewOpenAIProvider(cfg.APIKey, cfg.Model), nil
	default:
		return nil, fmt.Errorf("unsupported AI provider: %s", cfg.Provider)
	}
}
