package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/hoanghai1803/apricot/internal/models"
	"github.com/hoanghai1803/apricot/internal/storage"
)

func TestGetSources(t *testing.T) {
	store := newTestStore(t)

	handler := GetSources(store)
	r := httptest.NewRequest(http.MethodGet, "/api/sources", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d", w.Code, http.StatusOK)
	}

	var sources []models.BlogSource
	if err := json.NewDecoder(w.Body).Decode(&sources); err != nil {
		t.Fatalf("decoding response: %v", err)
	}

	expectedCount := storage.DefaultSourceCount()
	if len(sources) != expectedCount {
		t.Errorf("got %d sources, want %d", len(sources), expectedCount)
	}

	// Verify sources have expected fields populated.
	for _, s := range sources {
		if s.ID == 0 {
			t.Error("source ID should not be zero")
		}
		if s.Name == "" {
			t.Error("source name should not be empty")
		}
		if s.FeedURL == "" {
			t.Error("source feed_url should not be empty")
		}
	}
}

func TestToggleSource(t *testing.T) {
	store := newTestStore(t)

	t.Run("deactivate source", func(t *testing.T) {
		body := `{"is_active": false}`
		r := httptest.NewRequest(http.MethodPut, "/api/sources/1", bytes.NewBufferString(body))
		w := httptest.NewRecorder()

		// Set up chi URL param.
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", "1")
		r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

		handler := ToggleSource(store)
		handler.ServeHTTP(w, r)

		if w.Code != http.StatusOK {
			t.Fatalf("got status %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
		}

		// Verify the source is now inactive by fetching all sources.
		sourcesW := httptest.NewRecorder()
		sourcesR := httptest.NewRequest(http.MethodGet, "/api/sources", nil)
		GetSources(store).ServeHTTP(sourcesW, sourcesR)

		var sources []models.BlogSource
		if err := json.NewDecoder(sourcesW.Body).Decode(&sources); err != nil {
			t.Fatalf("decoding sources: %v", err)
		}

		for _, s := range sources {
			if s.ID == 1 && s.IsActive {
				t.Error("source 1 should be inactive after toggle")
			}
		}
	})

	t.Run("reactivate source", func(t *testing.T) {
		body := `{"is_active": true}`
		r := httptest.NewRequest(http.MethodPut, "/api/sources/1", bytes.NewBufferString(body))
		w := httptest.NewRecorder()

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", "1")
		r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

		ToggleSource(store).ServeHTTP(w, r)

		if w.Code != http.StatusOK {
			t.Fatalf("got status %d, want %d", w.Code, http.StatusOK)
		}
	})

	t.Run("not found", func(t *testing.T) {
		body := `{"is_active": true}`
		r := httptest.NewRequest(http.MethodPut, "/api/sources/99999", bytes.NewBufferString(body))
		w := httptest.NewRecorder()

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", "99999")
		r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

		ToggleSource(store).ServeHTTP(w, r)

		if w.Code != http.StatusNotFound {
			t.Fatalf("got status %d, want %d", w.Code, http.StatusNotFound)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		body := `{"is_active": true}`
		r := httptest.NewRequest(http.MethodPut, "/api/sources/abc", bytes.NewBufferString(body))
		w := httptest.NewRecorder()

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", "abc")
		r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

		ToggleSource(store).ServeHTTP(w, r)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("got status %d, want %d", w.Code, http.StatusBadRequest)
		}
	})

}
