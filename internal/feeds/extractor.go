package feeds

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
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

// extractFullText fetches the web page using the given HTTP client and returns
// its main readable text content using go-readability's FromReader. Using the
// shared HTTP client ensures consistent User-Agent headers and TLS settings.
func extractFullText(client *http.Client, rawURL string) (string, error) {
	article, err := fetchAndParse(client, rawURL)
	if err != nil {
		return "", err
	}
	return article.TextContent, nil
}

// ExtractArticleMetadata fetches a web page using the fetcher's HTTP client
// and returns its full metadata (title, site name, excerpt, text content,
// published date).
func (f *Fetcher) ExtractArticleMetadata(ctx context.Context, rawURL string) (*ArticleMetadata, error) {
	domain := extractDomain(rawURL)
	f.waitForRateLimit(domain)

	article, err := fetchAndParse(f.client, rawURL)
	if err != nil {
		return nil, err
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

// fetchAndParse fetches a page using the given HTTP client and parses it with
// go-readability's FromReader. This avoids readability's internal HTTP client
// which has shorter timeouts and a bot-like User-Agent.
func fetchAndParse(client *http.Client, rawURL string) (readability.Article, error) {
	resp, err := client.Get(rawURL)
	if err != nil {
		return readability.Article{}, fmt.Errorf("readability extraction: failed to fetch the page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return readability.Article{}, fmt.Errorf("readability extraction: HTTP %d for %s", resp.StatusCode, rawURL)
	}

	pageURL, err := url.Parse(rawURL)
	if err != nil {
		return readability.Article{}, fmt.Errorf("readability extraction: invalid URL %q: %w", rawURL, err)
	}

	article, err := readability.FromReader(resp.Body, pageURL)
	if err != nil {
		return readability.Article{}, fmt.Errorf("readability extraction: %w", err)
	}

	return article, nil
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
