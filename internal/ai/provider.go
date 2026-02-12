package ai

import (
	"context"
	"fmt"
)

// AIProvider is the interface that all LLM providers must implement.
type AIProvider interface {
	// FilterAndRank selects and ranks blogs based on user preferences.
	// It returns up to 10 blogs ranked by relevance to the given preferences.
	FilterAndRank(ctx context.Context, preferences string, blogs []BlogEntry) ([]RankedBlog, error)

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
