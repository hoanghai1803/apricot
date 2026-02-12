package feeds

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/hoanghai1803/apricot/internal/models"
	"golang.org/x/net/html"
)

// IsScrapeURL returns true if the feed URL uses the scrape:// scheme,
// indicating it should be fetched via HTML scraping instead of RSS.
func IsScrapeURL(feedURL string) bool {
	return strings.HasPrefix(feedURL, "scrape://")
}

// ScrapeURLToHTTPS converts a scrape:// URL to its https:// equivalent.
func ScrapeURLToHTTPS(feedURL string) string {
	return "https://" + strings.TrimPrefix(feedURL, "scrape://")
}

// scrapeBlogPage fetches a blog listing page and extracts post entries from
// the HTML. Currently supports LinkedIn Engineering's DOM structure.
func (f *Fetcher) scrapeBlogPage(source models.BlogSource, maxArticles int) ([]models.Blog, error) {
	pageURL := ScrapeURLToHTTPS(source.FeedURL)

	req, err := http.NewRequest(http.MethodGet, pageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request for %q: %w", pageURL, err)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching %q: %w", pageURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetching %q: HTTP %d", pageURL, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading body from %q: %w", pageURL, err)
	}

	return parseLinkedInHTML(source, string(body), maxArticles)
}

// parseLinkedInHTML extracts blog posts from LinkedIn Engineering's HTML.
//
// LinkedIn's blog page has this structure:
//
//	li.post-list__item
//	  a.grid-post__link  -> title text + href
//	  p.grid-post__date  -> "Jan 29, 2026"
//
// There's also a featured post:
//
//	section.featured-post
//	  a.featured-post__headline -> title + href
func parseLinkedInHTML(source models.BlogSource, body string, maxArticles int) ([]models.Blog, error) {
	doc, err := html.Parse(strings.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("parsing HTML: %w", err)
	}

	now := time.Now()
	var blogs []models.Blog

	// Walk the HTML tree and find all links that look like blog post links.
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if len(blogs) >= maxArticles {
			return
		}

		if n.Type == html.ElementNode && n.Data == "a" {
			class := getAttr(n, "class")
			href := getAttr(n, "href")

			// Match LinkedIn's blog post link classes.
			if href != "" && (strings.Contains(class, "grid-post__link") || strings.Contains(class, "featured-post__headline")) {
				title := strings.TrimSpace(textContent(n))
				if title == "" || href == "" {
					goto next
				}

				// Make relative URLs absolute.
				if strings.HasPrefix(href, "/") {
					href = "https://www.linkedin.com" + href
				}

				// Try to find the date in a sibling/nearby element.
				publishedAt := findNearbyDate(n)

				blogs = append(blogs, models.Blog{
					SourceID:    source.ID,
					Source:      source.Name,
					Title:       title,
					URL:         href,
					PublishedAt: publishedAt,
					FetchedAt:   now,
					ContentHash: computeHash(href),
				})
			}
		}

	next:
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}

	walk(doc)
	return blogs, nil
}

// getAttr returns the value of the named attribute on an HTML node.
func getAttr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

// textContent returns the concatenated text content of an HTML node and its children.
func textContent(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var sb strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		sb.WriteString(textContent(c))
	}
	return sb.String()
}

// findNearbyDate walks up from the given node to find a date string in a
// sibling or parent's child with class containing "date".
func findNearbyDate(n *html.Node) *time.Time {
	// Walk up to the parent list item or container.
	parent := n.Parent
	for i := 0; i < 5 && parent != nil; i++ {
		// Search this parent's descendants for a date element.
		if t := findDateInSubtree(parent); t != nil {
			return t
		}
		parent = parent.Parent
	}
	return nil
}

// findDateInSubtree searches an HTML subtree for elements with "date" in their
// class and tries to parse the text content as a date.
func findDateInSubtree(n *html.Node) *time.Time {
	if n.Type == html.ElementNode {
		class := getAttr(n, "class")
		if strings.Contains(class, "date") {
			text := strings.TrimSpace(textContent(n))
			if t := parseHumanDate(text); t != nil {
				return t
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if t := findDateInSubtree(c); t != nil {
			return t
		}
	}
	return nil
}

// parseHumanDate tries to parse date strings like "Jan 29, 2026" or "February 5, 2026".
func parseHumanDate(s string) *time.Time {
	layouts := []string{
		"Jan 2, 2006",
		"January 2, 2006",
		"Jan 02, 2006",
		"January 02, 2006",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return &t
		}
	}
	return nil
}
