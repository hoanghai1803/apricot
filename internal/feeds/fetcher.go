package feeds

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/hoanghai1803/apricot/internal/models"
	"github.com/mmcdole/gofeed"
	"golang.org/x/sync/errgroup"
)

const (
	httpTimeout    = 30 * time.Second
	maxConcurrent  = 10
	rateLimitDelay = 1 * time.Second
	maxWords       = 5000
)

// Fetcher handles RSS feed fetching with per-domain rate limiting and
// bounded concurrency.
type Fetcher struct {
	client      *http.Client
	rateLimiter map[string]time.Time // per-domain last request time
	mu          sync.Mutex           // protects rateLimiter
}

// NewFetcher creates a Fetcher with a custom HTTP client configured with a
// 30-second timeout and the Apricot user agent.
func NewFetcher() *Fetcher {
	return &Fetcher{
		client: &http.Client{
			Timeout: httpTimeout,
			Transport: &userAgentTransport{
				base: http.DefaultTransport,
			},
		},
		rateLimiter: make(map[string]time.Time),
	}
}

// userAgentTransport wraps an http.RoundTripper to inject a custom User-Agent
// header on every request.
type userAgentTransport struct {
	base http.RoundTripper
}

func (t *userAgentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	// Use a browser-like User-Agent to avoid bot detection on some sites.
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	return t.base.RoundTrip(req)
}

// FetchAll fetches RSS feeds from all sources concurrently with a maximum of
// 10 goroutines. Each feed is limited to maxArticles most recent posts.
// Individual source failures are logged and skipped rather than failing the
// entire batch.
func (f *Fetcher) FetchAll(ctx context.Context, sources []models.BlogSource, maxArticles int) ([]models.Blog, error) {
	var (
		results []models.Blog
		mu      sync.Mutex
	)

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(maxConcurrent)

	for _, src := range sources {
		g.Go(func() error {
			blogs, err := f.fetchSingleFeed(ctx, src, maxArticles)
			if err != nil {
				slog.Warn("failed to fetch feed",
					"source", src.Name,
					"url", src.FeedURL,
					"error", err,
				)
				return nil // skip failures, don't fail the batch
			}

			mu.Lock()
			results = append(results, blogs...)
			mu.Unlock()

			slog.Info("fetched feed",
				"source", src.Name,
				"items", len(blogs),
			)
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("fetching feeds: %w", err)
	}

	return results, nil
}

// fetchSingleFeed retrieves and parses a feed from a single source. Sources
// with a "scrape://" feed URL are fetched via HTML scraping; all others use
// standard RSS/Atom parsing.
func (f *Fetcher) fetchSingleFeed(ctx context.Context, source models.BlogSource, maxArticles int) ([]models.Blog, error) {
	if IsScrapeURL(source.FeedURL) {
		return f.scrapeBlogPage(source, maxArticles)
	}

	domain := extractDomain(source.FeedURL)
	f.waitForRateLimit(domain)

	fp := gofeed.NewParser()
	fp.Client = f.client

	feed, err := fp.ParseURLWithContext(source.FeedURL, ctx)
	if err != nil {
		return nil, fmt.Errorf("parsing feed %q: %w", source.FeedURL, err)
	}

	blogs := parseFeedItems(source, feed, maxArticles)
	return blogs, nil
}

// ExtractArticle fetches the full article text from the given URL using
// go-readability. The returned text is truncated to 5000 words maximum.
func (f *Fetcher) ExtractArticle(ctx context.Context, articleURL string) (string, error) {
	domain := extractDomain(articleURL)
	f.waitForRateLimit(domain)

	text, err := extractFullText(articleURL, httpTimeout)
	if err != nil {
		return "", fmt.Errorf("extracting article from %q: %w", articleURL, err)
	}

	return truncateWords(text, maxWords), nil
}

// waitForRateLimit enforces a minimum delay of 1 second between requests to
// the same domain. It blocks until the delay has elapsed.
func (f *Fetcher) waitForRateLimit(domain string) {
	f.mu.Lock()
	lastReq, ok := f.rateLimiter[domain]
	if ok {
		elapsed := time.Since(lastReq)
		if elapsed < rateLimitDelay {
			f.mu.Unlock()
			time.Sleep(rateLimitDelay - elapsed)
			f.mu.Lock()
		}
	}
	f.rateLimiter[domain] = time.Now()
	f.mu.Unlock()
}

// extractDomain parses a URL and returns its hostname. If parsing fails, it
// returns the raw URL as a fallback key.
func extractDomain(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	return u.Hostname()
}
