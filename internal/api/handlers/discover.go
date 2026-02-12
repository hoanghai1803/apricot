package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/hoanghai1803/apricot/internal/ai"
	"github.com/hoanghai1803/apricot/internal/config"
	"github.com/hoanghai1803/apricot/internal/feeds"
	"github.com/hoanghai1803/apricot/internal/models"
	"github.com/hoanghai1803/apricot/internal/storage"
)

// DiscoverResult is a single item in the discovery response.
type DiscoverResult struct {
	ID          int64   `json:"id"`
	Title       string  `json:"title"`
	URL         string  `json:"url"`
	Source      string  `json:"source"`
	PublishedAt *string `json:"published_at,omitempty"`
	Summary     string  `json:"summary"`
	Reason      string  `json:"reason"`
}

// Discover handles POST /api/discover. It orchestrates the full discovery
// pipeline: fetch feeds, rank with AI, extract full content, summarize, and
// return the top results.
func Discover(store *storage.Store, aiProvider ai.AIProvider, fetcher *feeds.Fetcher, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// 1. Check if AI provider is configured.
		if aiProvider == nil {
			writeError(w, http.StatusServiceUnavailable,
				"AI provider not configured. Add your API key to config.toml")
			return
		}

		// 2. Load user preferences.
		var topics string
		if err := store.GetPreference(ctx, "topics", &topics); err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				writeError(w, http.StatusBadRequest,
					"No preferences set. Please set your interests first.")
				return
			}
			slog.Error("failed to load preferences", "error", err)
			writeError(w, http.StatusInternalServerError, "Failed to load preferences")
			return
		}

		// 3. Get active sources.
		sources, err := store.GetActiveSources(ctx)
		if err != nil {
			slog.Error("failed to get sources", "error", err)
			writeError(w, http.StatusInternalServerError, "Failed to get sources")
			return
		}

		if len(sources) == 0 {
			writeError(w, http.StatusBadRequest, "No active sources configured")
			return
		}

		// 4. Fetch feeds.
		slog.Info("fetching feeds", "sources", len(sources))
		blogs, err := fetcher.FetchAll(ctx, sources, cfg.Feeds.MaxArticlesPerFeed)
		if err != nil {
			slog.Error("failed to fetch feeds", "error", err)
			writeError(w, http.StatusInternalServerError, "Failed to fetch feeds")
			return
		}

		slog.Info("fetched blogs", "count", len(blogs))

		if len(blogs) == 0 {
			writeJSON(w, http.StatusOK, []DiscoverResult{})
			return
		}

		// 5. Save fetched blogs to storage.
		if err := store.SaveBlogs(ctx, blogs); err != nil {
			slog.Error("failed to save blogs", "error", err)
			writeError(w, http.StatusInternalServerError, "Failed to save blogs")
			return
		}

		// 6. Convert to AI blog entries.
		blogEntries := make([]ai.BlogEntry, len(blogs))
		for i, b := range blogs {
			var publishedAt string
			if b.PublishedAt != nil {
				publishedAt = b.PublishedAt.Format("2006-01-02")
			}
			blogEntries[i] = ai.BlogEntry{
				ID:          b.ID,
				Title:       b.Title,
				Source:      b.Source,
				PublishedAt: publishedAt,
				Description: b.Description,
				FullContent: b.FullContent,
			}
		}

		// We need to look up blogs by URL to get their stored IDs, since
		// SaveBlogs does upserts and we need the database IDs for ranking.
		blogByURL := make(map[string]*models.Blog, len(blogs))
		for i := range blogs {
			blogByURL[blogs[i].URL] = &blogs[i]
		}

		// Refresh blog entries with stored IDs.
		for i, entry := range blogEntries {
			if entry.ID == 0 {
				// Look up the stored blog by URL to get the real ID.
				if b, ok := blogByURL[blogs[i].URL]; ok {
					stored, err := store.GetBlogByURL(ctx, b.URL)
					if err == nil {
						blogEntries[i].ID = stored.ID
					}
				}
			}
		}

		// 7. Filter and rank with AI.
		slog.Info("ranking blogs with AI", "entries", len(blogEntries))
		ranked, err := aiProvider.FilterAndRank(ctx, topics, blogEntries)
		if err != nil {
			slog.Error("failed to rank blogs", "error", err)
			writeError(w, http.StatusInternalServerError, "Failed to rank blogs with AI")
			return
		}

		// Limit to top 10.
		if len(ranked) > 10 {
			ranked = ranked[:10]
		}

		slog.Info("ranked blogs", "count", len(ranked))

		// 8. Enrich each ranked blog: extract full content if missing, summarize.
		results := make([]DiscoverResult, 0, len(ranked))
		selectedIDs := make([]int64, 0, len(ranked))

		for _, rb := range ranked {
			blog, err := store.GetBlogByID(ctx, rb.ID)
			if err != nil {
				slog.Warn("ranked blog not found in storage", "id", rb.ID, "error", err)
				continue
			}

			// Extract full content if missing.
			if blog.FullContent == "" {
				slog.Info("extracting article", "url", blog.URL)
				content, err := fetcher.ExtractArticle(ctx, blog.URL)
				if err != nil {
					slog.Warn("failed to extract article", "url", blog.URL, "error", err)
				} else {
					blog.FullContent = content
					if _, err := store.UpsertBlog(ctx, blog); err != nil {
						slog.Warn("failed to update blog content", "id", blog.ID, "error", err)
					}
				}
			}

			// 9. Summarize if not cached.
			var summary string
			hasSummary, err := store.HasSummary(ctx, blog.ID)
			if err != nil {
				slog.Warn("failed to check summary cache", "id", blog.ID, "error", err)
			}

			if hasSummary {
				cached, err := store.GetSummaryByBlogID(ctx, blog.ID)
				if err == nil {
					summary = cached.Summary
				}
			} else {
				slog.Info("summarizing blog", "id", blog.ID, "title", blog.Title)
				var publishedAt string
				if blog.PublishedAt != nil {
					publishedAt = blog.PublishedAt.Format("2006-01-02")
				}
				entry := ai.BlogEntry{
					ID:          blog.ID,
					Title:       blog.Title,
					Source:      blog.Source,
					PublishedAt: publishedAt,
					Description: blog.Description,
					FullContent: blog.FullContent,
				}
				aiSummary, err := aiProvider.Summarize(ctx, entry)
				if err != nil {
					slog.Warn("failed to summarize blog", "id", blog.ID, "error", err)
					aiSummary = blog.Description // fallback to description
				}
				summary = aiSummary

				// Cache the summary.
				if err := store.UpsertSummary(ctx, &models.BlogSummary{
					BlogID:    blog.ID,
					Summary:   summary,
					ModelUsed: cfg.AI.Model,
				}); err != nil {
					slog.Warn("failed to cache summary", "id", blog.ID, "error", err)
				}
			}

			// 10. Build result.
			var pubAt *string
			if blog.PublishedAt != nil {
				v := blog.PublishedAt.Format("2006-01-02T15:04:05Z")
				pubAt = &v
			}

			results = append(results, DiscoverResult{
				ID:          blog.ID,
				Title:       blog.Title,
				URL:         blog.URL,
				Source:      blog.Source,
				PublishedAt: pubAt,
				Summary:     summary,
				Reason:      rb.Reason,
			})

			selectedIDs = append(selectedIDs, blog.ID)
		}

		// 11. Create audit session.
		selectedJSON, _ := json.Marshal(selectedIDs)
		session := &models.DiscoverySession{
			PreferencesSnapshot: topics,
			BlogsConsidered:     len(blogEntries),
			BlogsSelected:       string(selectedJSON),
			ModelUsed:            cfg.AI.Model,
		}
		if _, err := store.CreateSession(ctx, session); err != nil {
			slog.Warn("failed to create discovery session", "error", err)
		}

		// 12. Return response.
		writeJSON(w, http.StatusOK, results)
	}
}

