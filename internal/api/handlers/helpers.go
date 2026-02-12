package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// writeJSON encodes v as JSON and writes it to the response with the given
// HTTP status code. Content-Type is always set to application/json.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		// At this point headers are already sent; log but cannot change status.
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

// writeError writes a JSON error response with the given HTTP status code.
// The response body is {"error": "message"}.
func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// parseID extracts an int64 from a chi URL parameter.
func parseID(r *http.Request, param string) (int64, error) {
	raw := chi.URLParam(r, param)
	if raw == "" {
		return 0, fmt.Errorf("missing URL parameter %q", param)
	}

	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %q parameter: %w", param, err)
	}
	return id, nil
}
