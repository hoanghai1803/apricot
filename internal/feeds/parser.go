package feeds

import (
	"crypto/sha256"
	"fmt"
	"html"
	"regexp"
	"time"

	"github.com/hoanghai1803/apricot/internal/models"
	"github.com/mmcdole/gofeed"
)

var htmlTagPattern = regexp.MustCompile("<[^>]*>")

// parseFeedItems converts gofeed items into Blog models using the provided
// FetchOptions. In "recent_posts" mode (default) it takes the most recent
// MaxArticles posts. In "time_range" mode it filters to posts published
// within LookbackDays. Items with empty Title or URL are always skipped.
func parseFeedItems(source models.BlogSource, feed *gofeed.Feed, opts FetchOptions) []models.Blog {
	now := time.Now()

	if opts.Mode == "time_range" {
		return parseFeedItemsTimeRange(source, feed, opts.LookbackDays, now)
	}

	// Default: recent_posts mode.
	return parseFeedItemsRecent(source, feed, opts.MaxArticles, now)
}

// parseFeedItemsRecent takes the first maxArticles valid items from the feed.
// RSS feeds typically return items in reverse chronological order (newest
// first), so we simply take the first maxArticles valid items.
func parseFeedItemsRecent(source models.BlogSource, feed *gofeed.Feed, maxArticles int, now time.Time) []models.Blog {
	var blogs []models.Blog
	for _, item := range feed.Items {
		if len(blogs) >= maxArticles {
			break
		}

		if item.Title == "" || item.Link == "" {
			continue
		}

		var publishedAt *time.Time
		if item.PublishedParsed != nil {
			t := *item.PublishedParsed
			publishedAt = &t
		}

		blogs = append(blogs, models.Blog{
			SourceID:    source.ID,
			Source:      source.Name,
			Title:       item.Title,
			URL:         item.Link,
			Description: stripHTML(item.Description),
			PublishedAt: publishedAt,
			FetchedAt:   now,
			ContentHash: computeHash(item.Link),
		})
	}

	return blogs
}

// parseFeedItemsTimeRange filters feed items to those published within the
// last lookbackDays. Items without a published date are included (we cannot
// determine their age).
func parseFeedItemsTimeRange(source models.BlogSource, feed *gofeed.Feed, lookbackDays int, now time.Time) []models.Blog {
	cutoff := now.AddDate(0, 0, -lookbackDays)

	var blogs []models.Blog
	for _, item := range feed.Items {
		if item.Title == "" || item.Link == "" {
			continue
		}

		var publishedAt *time.Time
		if item.PublishedParsed != nil {
			t := *item.PublishedParsed
			publishedAt = &t

			// Skip items older than the cutoff.
			if t.Before(cutoff) {
				continue
			}
		}

		blogs = append(blogs, models.Blog{
			SourceID:    source.ID,
			Source:      source.Name,
			Title:       item.Title,
			URL:         item.Link,
			Description: stripHTML(item.Description),
			PublishedAt: publishedAt,
			FetchedAt:   now,
			ContentHash: computeHash(item.Link),
		})
	}

	return blogs
}

// computeHash returns the SHA-256 hex digest of the given string.
func computeHash(s string) string {
	h := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", h)
}

// stripHTML removes HTML tags from s and unescapes HTML entities.
func stripHTML(s string) string {
	clean := htmlTagPattern.ReplaceAllString(s, "")
	return html.UnescapeString(clean)
}
