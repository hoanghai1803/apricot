package handlers

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/hoanghai1803/apricot/internal/models"
	"github.com/hoanghai1803/apricot/internal/storage"
)

// SearchBlogs handles GET /api/search?q={query}&limit={limit}. It performs
// full-text search on blogs using FTS5.
func SearchBlogs(store *storage.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		query := r.URL.Query().Get("q")
		if query == "" {
			writeJSON(w, http.StatusOK, []models.Blog{})
			return
		}

		limit := 20
		if l := r.URL.Query().Get("limit"); l != "" {
			if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
				limit = parsed
			}
		}

		blogs, err := store.SearchBlogs(ctx, query, limit)
		if err != nil {
			slog.Error("failed to search blogs", "query", query, "error", err)
			writeError(w, http.StatusInternalServerError, "Search failed")
			return
		}

		writeJSON(w, http.StatusOK, blogs)
	}
}
