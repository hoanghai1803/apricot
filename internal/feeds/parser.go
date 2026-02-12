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

// parseFeedItems converts gofeed items into Blog models, taking the most
// recent maxArticles posts. Items with empty Title or URL are skipped.
// RSS feeds typically return items in reverse chronological order (newest
// first), so we simply take the first maxArticles valid items.
func parseFeedItems(source models.BlogSource, feed *gofeed.Feed, maxArticles int) []models.Blog {
	now := time.Now()

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
