package ai

import (
	"strings"
	"testing"
)

func TestFilterAndRankPrompt(t *testing.T) {
	preferences := "distributed systems, Kubernetes, Go"
	blogs := []BlogEntry{
		{
			ID:          1,
			Title:       "Building Scalable Microservices with Go",
			Source:      "Engineering Blog",
			PublishedAt: "2026-01-15",
			Description: "A deep dive into microservice patterns using Go.",
		},
		{
			ID:          2,
			Title:       "Kubernetes Autoscaling Best Practices",
			Source:      "Cloud Native Blog",
			PublishedAt: "2026-01-16",
			Description: "How we scaled to 10k pods with custom metrics.",
		},
	}

	t.Run("returns non-empty prompts", func(t *testing.T) {
		systemPrompt, userPrompt := FilterAndRankPrompt(preferences, blogs)

		if systemPrompt == "" {
			t.Error("expected non-empty system prompt")
		}
		if userPrompt == "" {
			t.Error("expected non-empty user prompt")
		}
	})

	t.Run("user prompt contains preferences", func(t *testing.T) {
		_, userPrompt := FilterAndRankPrompt(preferences, blogs)

		if !strings.Contains(userPrompt, preferences) {
			t.Errorf("user prompt should contain preferences %q", preferences)
		}
	})

	t.Run("user prompt contains blog titles", func(t *testing.T) {
		_, userPrompt := FilterAndRankPrompt(preferences, blogs)

		for _, blog := range blogs {
			if !strings.Contains(userPrompt, blog.Title) {
				t.Errorf("user prompt should contain blog title %q", blog.Title)
			}
		}
	})

	t.Run("user prompt contains blog metadata", func(t *testing.T) {
		_, userPrompt := FilterAndRankPrompt(preferences, blogs)

		for _, blog := range blogs {
			if !strings.Contains(userPrompt, blog.Source) {
				t.Errorf("user prompt should contain blog source %q", blog.Source)
			}
			if !strings.Contains(userPrompt, blog.PublishedAt) {
				t.Errorf("user prompt should contain blog published date %q", blog.PublishedAt)
			}
			if !strings.Contains(userPrompt, blog.Description) {
				t.Errorf("user prompt should contain blog description %q", blog.Description)
			}
		}
	})

	t.Run("system prompt contains ranking instructions", func(t *testing.T) {
		systemPrompt, _ := FilterAndRankPrompt(preferences, blogs)

		if !strings.Contains(systemPrompt, "10") {
			t.Error("system prompt should mention selecting 10 posts")
		}
		if !strings.Contains(systemPrompt, "JSON") {
			t.Error("system prompt should mention JSON output format")
		}
	})

	t.Run("handles empty blog list", func(t *testing.T) {
		systemPrompt, userPrompt := FilterAndRankPrompt(preferences, nil)

		if systemPrompt == "" {
			t.Error("system prompt should be non-empty even with no blogs")
		}
		if userPrompt == "" {
			t.Error("user prompt should be non-empty even with no blogs")
		}
		if !strings.Contains(userPrompt, preferences) {
			t.Error("user prompt should still contain preferences")
		}
	})
}

func TestSummarizePrompt(t *testing.T) {
	title := "How We Reduced Latency by 90% with io_uring"
	source := "Netflix Tech Blog"
	content := "In this post, we discuss how we migrated our data plane from epoll to io_uring, reducing p99 latency from 50ms to 5ms."

	t.Run("returns non-empty prompts", func(t *testing.T) {
		systemPrompt, userPrompt := SummarizePrompt(title, source, content)

		if systemPrompt == "" {
			t.Error("expected non-empty system prompt")
		}
		if userPrompt == "" {
			t.Error("expected non-empty user prompt")
		}
	})

	t.Run("user prompt contains title", func(t *testing.T) {
		_, userPrompt := SummarizePrompt(title, source, content)

		if !strings.Contains(userPrompt, title) {
			t.Errorf("user prompt should contain title %q", title)
		}
	})

	t.Run("user prompt contains source", func(t *testing.T) {
		_, userPrompt := SummarizePrompt(title, source, content)

		if !strings.Contains(userPrompt, source) {
			t.Errorf("user prompt should contain source %q", source)
		}
	})

	t.Run("user prompt contains content", func(t *testing.T) {
		_, userPrompt := SummarizePrompt(title, source, content)

		if !strings.Contains(userPrompt, content) {
			t.Errorf("user prompt should contain content")
		}
	})

	t.Run("system prompt contains summarization instructions", func(t *testing.T) {
		systemPrompt, _ := SummarizePrompt(title, source, content)

		if !strings.Contains(systemPrompt, "4-5 sentences") {
			t.Error("system prompt should mention 4-5 sentences")
		}
		if !strings.Contains(systemPrompt, "senior engineer") {
			t.Error("system prompt should mention target audience")
		}
	})
}

func TestExtractJSON(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "plain JSON array",
			input: `[{"id": 1, "reason": "relevant"}]`,
			want:  `[{"id": 1, "reason": "relevant"}]`,
		},
		{
			name:  "JSON wrapped in json code fence",
			input: "```json\n[{\"id\": 1, \"reason\": \"relevant\"}]\n```",
			want:  `[{"id": 1, "reason": "relevant"}]`,
		},
		{
			name:  "JSON wrapped in plain code fence",
			input: "```\n[{\"id\": 1, \"reason\": \"relevant\"}]\n```",
			want:  `[{"id": 1, "reason": "relevant"}]`,
		},
		{
			name:  "JSON with surrounding whitespace",
			input: "  \n  [{\"id\": 1}]  \n  ",
			want:  `[{"id": 1}]`,
		},
		{
			name:  "code fence with extra whitespace",
			input: "```json\n\n  [{\"id\": 1}]\n\n```",
			want:  `[{"id": 1}]`,
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractJSON(tt.input)
			if got != tt.want {
				t.Errorf("extractJSON() = %q, want %q", got, tt.want)
			}
		})
	}
}
