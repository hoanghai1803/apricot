package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/hoanghai1803/apricot/internal/models"
	"github.com/hoanghai1803/apricot/internal/storage"
)

// GetReadingList handles GET /api/reading-list. It returns all reading list
// items, optionally filtered by the "status" query parameter.
func GetReadingList(store *storage.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		status := r.URL.Query().Get("status")

		items, err := store.GetReadingList(ctx, status)
		if err != nil {
			slog.Error("failed to get reading list", "error", err)
			writeError(w, http.StatusInternalServerError, "Failed to get reading list")
			return
		}

		if items == nil {
			items = []models.ReadingListItem{}
		}

		writeJSON(w, http.StatusOK, items)
	}
}

// AddToReadingList handles POST /api/reading-list. It adds a blog post to
// the reading list by blog_id.
func AddToReadingList(store *storage.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var body struct {
			BlogID int64 `json:"blog_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid JSON body")
			return
		}

		if body.BlogID == 0 {
			writeError(w, http.StatusBadRequest, "blog_id is required")
			return
		}

		if err := store.AddToReadingList(ctx, body.BlogID); err != nil {
			slog.Warn("failed to add to reading list", "blog_id", body.BlogID, "error", err)
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		writeJSON(w, http.StatusCreated, map[string]string{"status": "added"})
	}
}

// UpdateReadingListItem handles PATCH /api/reading-list/{id}. It updates the
// status and/or notes of a reading list item.
func UpdateReadingListItem(store *storage.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		id, err := parseID(r, "id")
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		var body struct {
			Status *string `json:"status"`
			Notes  *string `json:"notes"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid JSON body")
			return
		}

		if body.Status != nil {
			if err := store.UpdateReadingListStatus(ctx, id, *body.Status); err != nil {
				if errors.Is(err, storage.ErrNotFound) {
					writeError(w, http.StatusNotFound, "Reading list item not found")
					return
				}
				slog.Error("failed to update reading list status", "id", id, "error", err)
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
		}

		if body.Notes != nil {
			if err := store.UpdateReadingListNotes(ctx, id, *body.Notes); err != nil {
				if errors.Is(err, storage.ErrNotFound) {
					writeError(w, http.StatusNotFound, "Reading list item not found")
					return
				}
				slog.Error("failed to update reading list notes", "id", id, "error", err)
				writeError(w, http.StatusInternalServerError, "Failed to update notes")
				return
			}
		}

		writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
	}
}

// DeleteReadingListItem handles DELETE /api/reading-list/{id}. It removes
// a reading list item by its ID.
func DeleteReadingListItem(store *storage.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		id, err := parseID(r, "id")
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		if err := store.RemoveFromReadingList(ctx, id); err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				writeError(w, http.StatusNotFound, "Reading list item not found")
				return
			}
			slog.Error("failed to remove from reading list", "id", id, "error", err)
			writeError(w, http.StatusInternalServerError, "Failed to remove from reading list")
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{"status": "removed"})
	}
}
