package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestWriteJSON(t *testing.T) {
	t.Run("encodes and sets content type", func(t *testing.T) {
		w := httptest.NewRecorder()
		data := map[string]string{"hello": "world"}

		writeJSON(w, http.StatusOK, data)

		if w.Code != http.StatusOK {
			t.Errorf("got status %d, want %d", w.Code, http.StatusOK)
		}

		ct := w.Header().Get("Content-Type")
		if ct != "application/json" {
			t.Errorf("got Content-Type %q, want %q", ct, "application/json")
		}

		var got map[string]string
		if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
			t.Fatalf("decoding response body: %v", err)
		}
		if got["hello"] != "world" {
			t.Errorf("got %q, want %q", got["hello"], "world")
		}
	})

	t.Run("sets custom status code", func(t *testing.T) {
		w := httptest.NewRecorder()
		writeJSON(w, http.StatusCreated, map[string]string{"ok": "true"})

		if w.Code != http.StatusCreated {
			t.Errorf("got status %d, want %d", w.Code, http.StatusCreated)
		}
	})
}

func TestWriteError(t *testing.T) {
	w := httptest.NewRecorder()
	writeError(w, http.StatusBadRequest, "something went wrong")

	if w.Code != http.StatusBadRequest {
		t.Errorf("got status %d, want %d", w.Code, http.StatusBadRequest)
	}

	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("got Content-Type %q, want %q", ct, "application/json")
	}

	var got map[string]string
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decoding response body: %v", err)
	}
	if got["error"] != "something went wrong" {
		t.Errorf("got error %q, want %q", got["error"], "something went wrong")
	}
}

func TestParseID(t *testing.T) {
	tests := []struct {
		name    string
		param   string
		value   string
		wantID  int64
		wantErr bool
	}{
		{
			name:   "valid integer",
			param:  "id",
			value:  "42",
			wantID: 42,
		},
		{
			name:   "valid large integer",
			param:  "id",
			value:  "123456789",
			wantID: 123456789,
		},
		{
			name:    "invalid string",
			param:   "id",
			value:   "abc",
			wantErr: true,
		},
		{
			name:    "empty string",
			param:   "id",
			value:   "",
			wantErr: true,
		},
		{
			name:    "float value",
			param:   "id",
			value:   "3.14",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build a chi context with the URL param set.
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add(tt.param, tt.value)
			r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

			got, err := parseID(r, tt.param)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.wantID {
				t.Errorf("got %d, want %d", got, tt.wantID)
			}
		})
	}
}

