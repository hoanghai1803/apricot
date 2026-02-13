package handlers

import (
	"context"
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

// DiscoverResponse is the full response for discovery endpoints.
type DiscoverResponse struct {
	Results     []DiscoverResult `json:"results"`
	FailedFeeds []feeds.FailedFeed `json:"failed_feeds"`
	SessionID   int64            `json:"session_id"`
	CreatedAt   string           `json:"created_at"`
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

		// 3. Load max discovery results preference.
		maxResults := 10
		var maxResultsPref int
		if err := store.GetPreference(ctx, "max_results", &maxResultsPref); err == nil {
			if maxResultsPref >= 5 && maxResultsPref <= 20 {
				maxResults = maxResultsPref
			}
		}

		// 4. Load feed preferences for mode, max articles, and lookback days.
		fetchOpts := buildFetchOptions(store, cfg, ctx)

		// 5. Get active sources.
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

		// 6. Fetch feeds.
		slog.Info("fetching feeds", "sources", len(sources), "mode", fetchOpts.Mode)
		fetchResult, err := fetcher.FetchAll(ctx, sources, fetchOpts)
		if err != nil {
			slog.Error("failed to fetch feeds", "error", err)
			writeError(w, http.StatusInternalServerError, "Failed to fetch feeds")
			return
		}

		blogs := fetchResult.Blogs
		failedFeeds := fetchResult.Failed

		slog.Info("fetched blogs", "count", len(blogs), "failed", len(failedFeeds))

		if len(blogs) == 0 {
			resp := DiscoverResponse{
				Results:     []DiscoverResult{},
				FailedFeeds: ensureFailedFeeds(failedFeeds),
			}
			writeJSON(w, http.StatusOK, resp)
			return
		}

		// 7. Save fetched blogs to storage.
		if err := store.SaveBlogs(ctx, blogs); err != nil {
			slog.Error("failed to save blogs", "error", err)
			writeError(w, http.StatusInternalServerError, "Failed to save blogs")
			return
		}

		// 8. Convert to AI blog entries.
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

		// 9. Filter and rank with AI.
		slog.Info("ranking blogs with AI", "entries", len(blogEntries))
		ranked, err := aiProvider.FilterAndRank(ctx, topics, blogEntries, maxResults)
		if err != nil {
			slog.Error("failed to rank blogs", "error", err)
			writeError(w, http.StatusInternalServerError, "Failed to rank blogs with AI")
			return
		}

		// Limit to configured max.
		if len(ranked) > maxResults {
			ranked = ranked[:maxResults]
		}

		slog.Info("ranked blogs", "count", len(ranked))

		// 10. Enrich each ranked blog: extract full content if missing, summarize.
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

			// 11. Summarize if not cached.
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

			// 12. Build result.
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

		// 13. Create audit session with full results.
		selectedJSON, _ := json.Marshal(selectedIDs)
		resultsJSON, _ := json.Marshal(results)
		failedFeedsJSON, _ := json.Marshal(ensureFailedFeeds(failedFeeds))

		session := &models.DiscoverySession{
			PreferencesSnapshot: topics,
			BlogsConsidered:     len(blogEntries),
			BlogsSelected:       string(selectedJSON),
			ModelUsed:            cfg.AI.Model,
			ResultsJSON:         string(resultsJSON),
			FailedFeedsJSON:     string(failedFeedsJSON),
		}
		sessionID, err := store.CreateSession(ctx, session)
		if err != nil {
			slog.Warn("failed to create discovery session", "error", err)
		}

		// 14. Return response.
		resp := DiscoverResponse{
			Results:     results,
			FailedFeeds: ensureFailedFeeds(failedFeeds),
			SessionID:   sessionID,
			CreatedAt:   session.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}

		writeJSON(w, http.StatusOK, resp)
	}
}

// GetLatestDiscovery handles GET /api/discover/latest. It returns the most
// recent discovery session's stored results without triggering a new discovery.
func GetLatestDiscovery(store *storage.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		session, err := store.GetLatestSession(ctx)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				writeJSON(w, http.StatusOK, DiscoverResponse{
					Results:     []DiscoverResult{},
					FailedFeeds: []feeds.FailedFeed{},
				})
				return
			}
			slog.Error("failed to get latest session", "error", err)
			writeError(w, http.StatusInternalServerError, "Failed to load latest discovery")
			return
		}

		var results []DiscoverResult
		if session.ResultsJSON != "" {
			if err := json.Unmarshal([]byte(session.ResultsJSON), &results); err != nil {
				slog.Error("failed to unmarshal session results", "error", err)
				writeError(w, http.StatusInternalServerError, "Failed to parse stored results")
				return
			}
		}
		if results == nil {
			results = []DiscoverResult{}
		}

		var failedFeeds []feeds.FailedFeed
		if session.FailedFeedsJSON != "" {
			if err := json.Unmarshal([]byte(session.FailedFeedsJSON), &failedFeeds); err != nil {
				slog.Warn("failed to unmarshal failed feeds", "error", err)
			}
		}
		if failedFeeds == nil {
			failedFeeds = []feeds.FailedFeed{}
		}

		resp := DiscoverResponse{
			Results:     results,
			FailedFeeds: failedFeeds,
			SessionID:   session.ID,
			CreatedAt:   session.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}

		writeJSON(w, http.StatusOK, resp)
	}
}

// buildFetchOptions reads user feed preferences and falls back to config defaults.
func buildFetchOptions(store *storage.Store, cfg *config.Config, ctx context.Context) feeds.FetchOptions {
	opts := feeds.FetchOptions{
		Mode:         "recent_posts",
		MaxArticles:  cfg.Feeds.MaxArticlesPerFeed,
		LookbackDays: cfg.Feeds.LookbackDays,
	}

	var feedMode string
	if err := store.GetPreference(ctx, "feed_mode", &feedMode); err == nil {
		if feedMode == "recent_posts" || feedMode == "time_range" {
			opts.Mode = feedMode
		}
	}

	var maxArticles int
	if err := store.GetPreference(ctx, "max_articles_per_feed", &maxArticles); err == nil {
		if maxArticles >= 5 && maxArticles <= 20 {
			opts.MaxArticles = maxArticles
		}
	}

	var lookbackDays int
	if err := store.GetPreference(ctx, "lookback_days", &lookbackDays); err == nil {
		if lookbackDays >= 1 && lookbackDays <= 30 {
			opts.LookbackDays = lookbackDays
		}
	}

	return opts
}

// ensureFailedFeeds returns an empty slice instead of nil for consistent
// JSON serialization.
func ensureFailedFeeds(ff []feeds.FailedFeed) []feeds.FailedFeed {
	if ff == nil {
		return []feeds.FailedFeed{}
	}
	return ff
}
