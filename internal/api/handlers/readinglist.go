package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/hoanghai1803/apricot/internal/feeds"
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

// AddCustomBlog handles POST /api/reading-list/custom. It fetches article
// metadata from a user-provided URL and adds it to the reading list.
func AddCustomBlog(store *storage.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var body struct {
			URL    string `json:"url"`
			Source string `json:"source"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid JSON body")
			return
		}

		body.URL = strings.TrimSpace(body.URL)
		body.Source = strings.TrimSpace(body.Source)

		if body.URL == "" {
			writeError(w, http.StatusBadRequest, "url is required")
			return
		}

		parsed, err := url.ParseRequestURI(body.URL)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			writeError(w, http.StatusBadRequest, "url must be a valid HTTP or HTTPS URL")
			return
		}

		// Check if blog already exists by URL.
		existing, err := store.GetBlogByURL(ctx, body.URL)
		if err != nil && !errors.Is(err, storage.ErrNotFound) {
			slog.Error("failed to check existing blog", "url", body.URL, "error", err)
			writeError(w, http.StatusInternalServerError, "Failed to check existing blog")
			return
		}

		var blogID int64
		if existing != nil {
			blogID = existing.ID
		} else {
			// Fetch article metadata from URL.
			meta, err := feeds.ExtractArticleMetadata(body.URL, 30*time.Second)
			if err != nil {
				slog.Warn("failed to extract article metadata", "url", body.URL, "error", err)
				writeError(w, http.StatusUnprocessableEntity, "Could not fetch article from URL")
				return
			}

			title := meta.Title
			if title == "" {
				title = body.URL
			}

			// Determine source display name.
			customSource := body.Source
			if customSource == "" {
				customSource = meta.SiteName
			}
			if customSource == "" {
				customSource = parsed.Hostname()
			}

			blogID, err = store.CreateCustomBlog(ctx, body.URL, title, meta.Excerpt, meta.TextContent, customSource)
			if err != nil {
				if strings.Contains(err.Error(), "UNIQUE constraint failed") {
					// Race condition: blog was inserted between our check and insert.
					existing, err2 := store.GetBlogByURL(ctx, body.URL)
					if err2 != nil {
						slog.Error("failed to get existing blog after conflict", "url", body.URL, "error", err2)
						writeError(w, http.StatusInternalServerError, "Failed to add blog")
						return
					}
					blogID = existing.ID
				} else {
					slog.Error("failed to create custom blog", "url", body.URL, "error", err)
					writeError(w, http.StatusInternalServerError, "Failed to save blog")
					return
				}
			}
		}

		if err := store.AddToReadingList(ctx, blogID); err != nil {
			if strings.Contains(err.Error(), "already on the reading list") {
				writeError(w, http.StatusConflict, err.Error())
				return
			}
			slog.Error("failed to add custom blog to reading list", "blog_id", blogID, "error", err)
			writeError(w, http.StatusInternalServerError, "Failed to add to reading list")
			return
		}

		writeJSON(w, http.StatusCreated, map[string]any{
			"status":  "added",
			"blog_id": blogID,
		})
	}
}
