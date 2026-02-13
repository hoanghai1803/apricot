package feeds

import (
	"fmt"
	"strings"
	"time"

	readability "github.com/go-shiori/go-readability"
)

// ArticleMetadata holds full metadata extracted from a web page.
type ArticleMetadata struct {
	Title       string
	SiteName    string
	Excerpt     string
	TextContent string
	PublishedAt *time.Time
}

// extractFullText fetches the web page at the given URL and returns its main
// readable text content using go-readability.
func extractFullText(url string, timeout time.Duration) (string, error) {
	article, err := readability.FromURL(url, timeout)
	if err != nil {
		return "", fmt.Errorf("readability extraction: %w", err)
	}
	return article.TextContent, nil
}

// ExtractArticleMetadata fetches a web page and returns its full metadata
// (title, site name, excerpt, text content, published date).
func ExtractArticleMetadata(url string, timeout time.Duration) (*ArticleMetadata, error) {
	article, err := readability.FromURL(url, timeout)
	if err != nil {
		return nil, fmt.Errorf("readability extraction: %w", err)
	}

	meta := &ArticleMetadata{
		Title:       article.Title,
		SiteName:    article.SiteName,
		Excerpt:     article.Excerpt,
		TextContent: article.TextContent,
	}
	if article.PublishedTime != nil {
		meta.PublishedAt = article.PublishedTime
	}
	return meta, nil
}

// truncateWords returns the first maxWords whitespace-delimited words from s.
// If s contains fewer than maxWords words, it is returned unchanged.
func truncateWords(s string, maxWords int) string {
	words := strings.Fields(s)
	if len(words) <= maxWords {
		return s
	}
	return strings.Join(words[:maxWords], " ")
}
