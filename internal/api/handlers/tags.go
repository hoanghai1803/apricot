package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hoanghai1803/apricot/internal/storage"
)

// AddTagToItem handles POST /api/reading-list/{id}/tags. It adds a tag to a
// reading list item. The tag is created if it doesn't exist.
func AddTagToItem(store *storage.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		id, err := parseID(r, "id")
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		var body struct {
			Tag string `json:"tag"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid JSON body")
			return
		}

		if body.Tag == "" {
			writeError(w, http.StatusBadRequest, "tag is required")
			return
		}

		if err := store.AddTagToItem(ctx, id, body.Tag); err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				writeError(w, http.StatusNotFound, "Reading list item not found")
				return
			}
			slog.Error("failed to add tag", "id", id, "tag", body.Tag, "error", err)
			writeError(w, http.StatusInternalServerError, "Failed to add tag")
			return
		}

		writeJSON(w, http.StatusCreated, map[string]string{"status": "added"})
	}
}

// RemoveTagFromItem handles DELETE /api/reading-list/{id}/tags/{tag}. It
// removes a tag from a reading list item and cleans up unused tags.
func RemoveTagFromItem(store *storage.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		id, err := parseID(r, "id")
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		tag := chi.URLParam(r, "tag")
		if tag == "" {
			writeError(w, http.StatusBadRequest, "tag parameter is required")
			return
		}

		if err := store.RemoveTagFromItem(ctx, id, tag); err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				writeError(w, http.StatusNotFound, "Tag not found on this item")
				return
			}
			slog.Error("failed to remove tag", "id", id, "tag", tag, "error", err)
			writeError(w, http.StatusInternalServerError, "Failed to remove tag")
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{"status": "removed"})
	}
}

// GetAllTags handles GET /api/tags. It returns all distinct tag names for
// autocomplete.
func GetAllTags(store *storage.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		tags, err := store.GetAllTags(ctx)
		if err != nil {
			slog.Error("failed to get tags", "error", err)
			writeError(w, http.StatusInternalServerError, "Failed to get tags")
			return
		}

		writeJSON(w, http.StatusOK, tags)
	}
}
