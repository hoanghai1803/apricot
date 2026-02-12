package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hoanghai1803/apricot/internal/models"
)

// seedBlog inserts a test blog into the store and returns its ID.
func seedBlog(t *testing.T, store interface {
	UpsertBlog(ctx context.Context, blog *models.Blog) (int64, error)
}) int64 {
	t.Helper()
	now := time.Now()
	blog := &models.Blog{
		SourceID:    1,
		Title:       "Test Blog Post",
		URL:         "https://example.com/test-post",
		Description: "A test blog post",
		PublishedAt: &now,
		FetchedAt:   now,
	}
	id, err := store.UpsertBlog(context.Background(), blog)
	if err != nil {
		t.Fatalf("seeding blog: %v", err)
	}
	return id
}

func TestReadingListAddAndGet(t *testing.T) {
	store := newTestStore(t)
	blogID := seedBlog(t, store)

	// POST to add item.
	body, _ := json.Marshal(map[string]int64{"blog_id": blogID})
	postR := httptest.NewRequest(http.MethodPost, "/api/reading-list", bytes.NewBuffer(body))
	postW := httptest.NewRecorder()

	AddToReadingList(store).ServeHTTP(postW, postR)

	if postW.Code != http.StatusCreated {
		t.Fatalf("POST got status %d, want %d; body: %s", postW.Code, http.StatusCreated, postW.Body.String())
	}

	var addResult map[string]string
	if err := json.NewDecoder(postW.Body).Decode(&addResult); err != nil {
		t.Fatalf("decoding POST response: %v", err)
	}
	if addResult["status"] != "added" {
		t.Errorf("got status %q, want %q", addResult["status"], "added")
	}

	// GET reading list.
	getR := httptest.NewRequest(http.MethodGet, "/api/reading-list", nil)
	getW := httptest.NewRecorder()

	GetReadingList(store).ServeHTTP(getW, getR)

	if getW.Code != http.StatusOK {
		t.Fatalf("GET got status %d, want %d", getW.Code, http.StatusOK)
	}

	var items []models.ReadingListItem
	if err := json.NewDecoder(getW.Body).Decode(&items); err != nil {
		t.Fatalf("decoding GET response: %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("got %d items, want 1", len(items))
	}

	if items[0].BlogID != blogID {
		t.Errorf("got blog_id %d, want %d", items[0].BlogID, blogID)
	}
	if items[0].Status != "unread" {
		t.Errorf("got status %q, want %q", items[0].Status, "unread")
	}
}

func TestReadingListPatchStatus(t *testing.T) {
	store := newTestStore(t)
	blogID := seedBlog(t, store)

	// Add to reading list first.
	addBody, _ := json.Marshal(map[string]int64{"blog_id": blogID})
	addR := httptest.NewRequest(http.MethodPost, "/api/reading-list", bytes.NewBuffer(addBody))
	addW := httptest.NewRecorder()
	AddToReadingList(store).ServeHTTP(addW, addR)

	if addW.Code != http.StatusCreated {
		t.Fatalf("add failed with status %d", addW.Code)
	}

	// Get the reading list to find the item ID.
	getR := httptest.NewRequest(http.MethodGet, "/api/reading-list", nil)
	getW := httptest.NewRecorder()
	GetReadingList(store).ServeHTTP(getW, getR)

	var items []models.ReadingListItem
	if err := json.NewDecoder(getW.Body).Decode(&items); err != nil {
		t.Fatalf("decoding items: %v", err)
	}
	if len(items) == 0 {
		t.Fatal("no items in reading list")
	}
	itemID := items[0].ID

	// PATCH status to "reading".
	patchBody := `{"status": "reading"}`
	patchR := httptest.NewRequest(http.MethodPatch, "/api/reading-list/1", bytes.NewBufferString(patchBody))
	patchW := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", jsonInt64(itemID))
	patchR = patchR.WithContext(context.WithValue(patchR.Context(), chi.RouteCtxKey, rctx))

	UpdateReadingListItem(store).ServeHTTP(patchW, patchR)

	if patchW.Code != http.StatusOK {
		t.Fatalf("PATCH got status %d, want %d; body: %s", patchW.Code, http.StatusOK, patchW.Body.String())
	}

	// Verify status changed.
	getR2 := httptest.NewRequest(http.MethodGet, "/api/reading-list", nil)
	getW2 := httptest.NewRecorder()
	GetReadingList(store).ServeHTTP(getW2, getR2)

	var items2 []models.ReadingListItem
	if err := json.NewDecoder(getW2.Body).Decode(&items2); err != nil {
		t.Fatalf("decoding items: %v", err)
	}
	if len(items2) == 0 {
		t.Fatal("no items after patch")
	}
	if items2[0].Status != "reading" {
		t.Errorf("got status %q, want %q", items2[0].Status, "reading")
	}
}

func TestReadingListPatchNotFound(t *testing.T) {
	store := newTestStore(t)

	patchBody := `{"status": "reading"}`
	patchR := httptest.NewRequest(http.MethodPatch, "/api/reading-list/99999", bytes.NewBufferString(patchBody))
	patchW := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "99999")
	patchR = patchR.WithContext(context.WithValue(patchR.Context(), chi.RouteCtxKey, rctx))

	UpdateReadingListItem(store).ServeHTTP(patchW, patchR)

	if patchW.Code != http.StatusNotFound {
		t.Fatalf("got status %d, want %d", patchW.Code, http.StatusNotFound)
	}
}

func TestReadingListDelete(t *testing.T) {
	store := newTestStore(t)
	blogID := seedBlog(t, store)

	// Add.
	addBody, _ := json.Marshal(map[string]int64{"blog_id": blogID})
	addR := httptest.NewRequest(http.MethodPost, "/api/reading-list", bytes.NewBuffer(addBody))
	addW := httptest.NewRecorder()
	AddToReadingList(store).ServeHTTP(addW, addR)

	// Get ID.
	getR := httptest.NewRequest(http.MethodGet, "/api/reading-list", nil)
	getW := httptest.NewRecorder()
	GetReadingList(store).ServeHTTP(getW, getR)

	var items []models.ReadingListItem
	if err := json.NewDecoder(getW.Body).Decode(&items); err != nil {
		t.Fatalf("decoding items: %v", err)
	}
	if len(items) == 0 {
		t.Fatal("no items in reading list")
	}

	// DELETE.
	delR := httptest.NewRequest(http.MethodDelete, "/api/reading-list/1", nil)
	delW := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", jsonInt64(items[0].ID))
	delR = delR.WithContext(context.WithValue(delR.Context(), chi.RouteCtxKey, rctx))

	DeleteReadingListItem(store).ServeHTTP(delW, delR)

	if delW.Code != http.StatusOK {
		t.Fatalf("DELETE got status %d, want %d; body: %s", delW.Code, http.StatusOK, delW.Body.String())
	}

	// Verify removed.
	getR2 := httptest.NewRequest(http.MethodGet, "/api/reading-list", nil)
	getW2 := httptest.NewRecorder()
	GetReadingList(store).ServeHTTP(getW2, getR2)

	var items2 []models.ReadingListItem
	if err := json.NewDecoder(getW2.Body).Decode(&items2); err != nil {
		t.Fatalf("decoding items: %v", err)
	}
	if len(items2) != 0 {
		t.Errorf("got %d items, want 0 after delete", len(items2))
	}
}

func TestReadingListDeleteNotFound(t *testing.T) {
	store := newTestStore(t)

	delR := httptest.NewRequest(http.MethodDelete, "/api/reading-list/99999", nil)
	delW := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "99999")
	delR = delR.WithContext(context.WithValue(delR.Context(), chi.RouteCtxKey, rctx))

	DeleteReadingListItem(store).ServeHTTP(delW, delR)

	if delW.Code != http.StatusNotFound {
		t.Fatalf("got status %d, want %d", delW.Code, http.StatusNotFound)
	}
}

func TestReadingListDuplicate(t *testing.T) {
	store := newTestStore(t)
	blogID := seedBlog(t, store)

	// Add first time.
	body, _ := json.Marshal(map[string]int64{"blog_id": blogID})
	r1 := httptest.NewRequest(http.MethodPost, "/api/reading-list", bytes.NewBuffer(body))
	w1 := httptest.NewRecorder()
	AddToReadingList(store).ServeHTTP(w1, r1)

	if w1.Code != http.StatusCreated {
		t.Fatalf("first add got status %d", w1.Code)
	}

	// Add second time — should be a duplicate error.
	body2, _ := json.Marshal(map[string]int64{"blog_id": blogID})
	r2 := httptest.NewRequest(http.MethodPost, "/api/reading-list", bytes.NewBuffer(body2))
	w2 := httptest.NewRecorder()
	AddToReadingList(store).ServeHTTP(w2, r2)

	if w2.Code != http.StatusBadRequest {
		t.Fatalf("duplicate add got status %d, want %d; body: %s", w2.Code, http.StatusBadRequest, w2.Body.String())
	}
}

func TestReadingListMissingBlogID(t *testing.T) {
	store := newTestStore(t)

	body := `{}`
	r := httptest.NewRequest(http.MethodPost, "/api/reading-list", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	AddToReadingList(store).ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("got status %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// jsonInt64 converts an int64 to its string representation for URL params.
func jsonInt64(n int64) string {
	b, _ := json.Marshal(n)
	return string(b)
}
