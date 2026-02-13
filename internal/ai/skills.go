package ai

import (
	"fmt"
	"strings"
)

const filterAndRankSystemPromptTmpl = `You are a tech blog curator. Given the user's interests and a list of recent blog posts, select exactly %d posts that best match the user's interests. Return ONLY valid JSON: an array of objects with "id" (the post ID) and "reason" (one sentence explaining why this post matches). Rank by relevance, most relevant first. If there are fewer than %d posts, select all of them.`

const serendipitySystemPromptTmpl = `You are a tech blog curator focused on broadening horizons. Given the user's stated interests and a list of recent blog posts, select exactly %d posts that are OUTSIDE the user's usual interests but are still high-quality, surprising, and educational. Deliberately avoid posts that directly match the user's interests. Instead, pick posts from different domains, unexpected topics, or novel approaches that a curious engineer would find fascinating. Return ONLY valid JSON: an array of objects with "id" (the post ID) and "reason" (one sentence explaining why this post is a surprising but worthwhile read). Rank by how interesting and unexpected the post would be. If there are fewer than %d posts, select all of them.`

const summarizeSystemPrompt = `You are a technical writer. Summarize the following blog post in exactly 4-5 sentences. Focus on: the problem being solved, the approach taken, key technical decisions, and the outcome or results. Write for a senior engineer audience. Be specific about technologies and numbers mentioned in the post. Do NOT include any prefix like "# Summary" or "Summary:" — start directly with the first sentence.`

// FilterAndRankPrompt builds the system and user prompts for the
// filter-and-rank operation. When serendipity is true, the prompt
// deliberately selects posts outside the user's stated interests.
func FilterAndRankPrompt(preferences string, blogs []BlogEntry, maxResults int, serendipity bool) (systemPrompt string, userPrompt string) {
	if serendipity {
		systemPrompt = fmt.Sprintf(serendipitySystemPromptTmpl, maxResults, maxResults)
	} else {
		systemPrompt = fmt.Sprintf(filterAndRankSystemPromptTmpl, maxResults, maxResults)
	}

	var b strings.Builder
	b.WriteString("User Preferences:\n")
	b.WriteString(preferences)
	b.WriteString("\n\nRecent Blog Posts:\n")

	for i, blog := range blogs {
		fmt.Fprintf(&b, "%d. ID: %d | Title: %s | Source: %s | Published: %s | Description: %s\n",
			i+1, blog.ID, blog.Title, blog.Source, blog.PublishedAt, blog.Description)
	}

	userPrompt = b.String()
	return systemPrompt, userPrompt
}

// SummarizePrompt builds the system and user prompts for the blog
// summarization operation.
func SummarizePrompt(title, source, content string) (systemPrompt string, userPrompt string) {
	systemPrompt = summarizeSystemPrompt

	var b strings.Builder
	fmt.Fprintf(&b, "Blog Title: %s\n", title)
	fmt.Fprintf(&b, "Blog Source: %s\n", source)
	b.WriteString("Blog Content:\n")
	b.WriteString(content)

	userPrompt = b.String()
	return systemPrompt, userPrompt
}

// extractJSON strips markdown code fences from a string that may contain
// JSON wrapped in ```json ... ``` or ``` ... ``` blocks. This handles the
// common case where LLMs return JSON inside code fences.
func extractJSON(s string) string {
	s = strings.TrimSpace(s)

	// Try ```json ... ``` first.
	if after, found := strings.CutPrefix(s, "```json"); found {
		if idx := strings.LastIndex(after, "```"); idx >= 0 {
			after = after[:idx]
		}
		return strings.TrimSpace(after)
	}

	// Try plain ``` ... ```.
	if after, found := strings.CutPrefix(s, "```"); found {
		if idx := strings.LastIndex(after, "```"); idx >= 0 {
			after = after[:idx]
		}
		return strings.TrimSpace(after)
	}

	return s
}
