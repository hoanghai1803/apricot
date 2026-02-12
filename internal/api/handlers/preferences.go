package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/hoanghai1803/apricot/internal/storage"
)

// GetPreferences handles GET /api/preferences. It returns all user
// preferences as a JSON object.
func GetPreferences(store *storage.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		prefs, err := store.GetAllPreferences(ctx)
		if err != nil {
			slog.Error("failed to get preferences", "error", err)
			writeError(w, http.StatusInternalServerError, "Failed to get preferences")
			return
		}

		writeJSON(w, http.StatusOK, prefs)
	}
}

// UpdatePreferences handles PUT /api/preferences. It accepts a JSON object
// where each key-value pair is saved as a separate preference.
func UpdatePreferences(store *storage.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var body map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid JSON body")
			return
		}

		for key, value := range body {
			if err := store.SetPreference(ctx, key, json.RawMessage(value)); err != nil {
				slog.Error("failed to set preference", "key", key, "error", err)
				writeError(w, http.StatusInternalServerError, "Failed to save preferences")
				return
			}
		}

		// Return the saved preferences.
		prefs, err := store.GetAllPreferences(ctx)
		if err != nil {
			slog.Error("failed to get preferences after save", "error", err)
			writeError(w, http.StatusInternalServerError, "Failed to get preferences")
			return
		}

		writeJSON(w, http.StatusOK, prefs)
	}
}
