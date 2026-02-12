package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/hoanghai1803/apricot/internal/storage"
)

// GetSources handles GET /api/sources. It returns all blog sources.
func GetSources(store *storage.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		sources, err := store.GetAllSources(ctx)
		if err != nil {
			slog.Error("failed to get sources", "error", err)
			writeError(w, http.StatusInternalServerError, "Failed to get sources")
			return
		}

		writeJSON(w, http.StatusOK, sources)
	}
}

// ToggleSource handles PUT /api/sources/{id}. It toggles the is_active flag
// for a blog source.
func ToggleSource(store *storage.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		id, err := parseID(r, "id")
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		var body struct {
			IsActive bool `json:"is_active"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid JSON body")
			return
		}

		if err := store.ToggleSource(ctx, id, body.IsActive); err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				writeError(w, http.StatusNotFound, "Source not found")
				return
			}
			slog.Error("failed to toggle source", "id", id, "error", err)
			writeError(w, http.StatusInternalServerError, "Failed to toggle source")
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
	}
}
