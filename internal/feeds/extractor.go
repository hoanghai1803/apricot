package feeds

import (
	"fmt"
	"strings"
	"time"

	readability "github.com/go-shiori/go-readability"
)

// extractFullText fetches the web page at the given URL and returns its main
// readable text content using go-readability.
func extractFullText(url string, timeout time.Duration) (string, error) {
	article, err := readability.FromURL(url, timeout)
	if err != nil {
		return "", fmt.Errorf("readability extraction: %w", err)
	}
	return article.TextContent, nil
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
