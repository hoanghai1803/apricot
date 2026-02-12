package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetPreferencesEmpty(t *testing.T) {
	store := newTestStore(t)

	handler := GetPreferences(store)
	r := httptest.NewRequest(http.MethodGet, "/api/preferences", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d", w.Code, http.StatusOK)
	}

	var prefs map[string]json.RawMessage
	if err := json.NewDecoder(w.Body).Decode(&prefs); err != nil {
		t.Fatalf("decoding response: %v", err)
	}

	if len(prefs) != 0 {
		t.Errorf("got %d preferences, want 0", len(prefs))
	}
}

func TestPreferencesRoundTrip(t *testing.T) {
	store := newTestStore(t)

	// PUT preferences.
	body := `{"topics": "distributed systems, Go programming", "selected_sources": [1, 3, 5]}`
	putR := httptest.NewRequest(http.MethodPut, "/api/preferences", bytes.NewBufferString(body))
	putW := httptest.NewRecorder()

	UpdatePreferences(store).ServeHTTP(putW, putR)

	if putW.Code != http.StatusOK {
		t.Fatalf("PUT got status %d, want %d; body: %s", putW.Code, http.StatusOK, putW.Body.String())
	}

	// GET preferences back.
	getR := httptest.NewRequest(http.MethodGet, "/api/preferences", nil)
	getW := httptest.NewRecorder()

	GetPreferences(store).ServeHTTP(getW, getR)

	if getW.Code != http.StatusOK {
		t.Fatalf("GET got status %d, want %d", getW.Code, http.StatusOK)
	}

	var prefs map[string]json.RawMessage
	if err := json.NewDecoder(getW.Body).Decode(&prefs); err != nil {
		t.Fatalf("decoding response: %v", err)
	}

	if len(prefs) != 2 {
		t.Errorf("got %d preferences, want 2", len(prefs))
	}

	// Verify topics value.
	var topics string
	if err := json.Unmarshal(prefs["topics"], &topics); err != nil {
		t.Fatalf("unmarshaling topics: %v", err)
	}
	if topics != "distributed systems, Go programming" {
		t.Errorf("got topics %q, want %q", topics, "distributed systems, Go programming")
	}

	// Verify selected_sources value.
	var sources []int
	if err := json.Unmarshal(prefs["selected_sources"], &sources); err != nil {
		t.Fatalf("unmarshaling selected_sources: %v", err)
	}
	if len(sources) != 3 {
		t.Errorf("got %d selected_sources, want 3", len(sources))
	}
}

func TestUpdatePreferencesInvalidJSON(t *testing.T) {
	store := newTestStore(t)

	r := httptest.NewRequest(http.MethodPut, "/api/preferences", bytes.NewBufferString("not json"))
	w := httptest.NewRecorder()

	UpdatePreferences(store).ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("got status %d, want %d", w.Code, http.StatusBadRequest)
	}
}
