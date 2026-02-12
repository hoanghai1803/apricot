package storage

import (
	"context"
	"errors"
	"testing"

	"github.com/hoanghai1803/apricot/internal/models"
)

func TestCreateSession(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	inputTokens := 100
	outputTokens := 50
	session := &models.DiscoverySession{
		PreferencesSnapshot: `{"topics":["go","rust"]}`,
		BlogsConsidered:     25,
		BlogsSelected:       `[1,2,3]`,
		ModelUsed:           "claude-haiku-4-5",
		InputTokens:         &inputTokens,
		OutputTokens:        &outputTokens,
		ResultsJSON:         `[{"id":1,"title":"test"}]`,
		FailedFeedsJSON:     `[{"source":"Bad Blog","error":"timeout"}]`,
	}

	id, err := store.CreateSession(ctx, session)
	if err != nil {
		t.Fatalf("CreateSession() error: %v", err)
	}
	if id == 0 {
		t.Fatal("CreateSession() returned id 0")
	}

	sessions, err := store.GetRecentSessions(ctx, 10)
	if err != nil {
		t.Fatalf("GetRecentSessions() error: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("got %d sessions, want 1", len(sessions))
	}
	got := sessions[0]
	if got.ID != id {
		t.Errorf("ID = %d, want %d", got.ID, id)
	}
	if got.PreferencesSnapshot != `{"topics":["go","rust"]}` {
		t.Errorf("PreferencesSnapshot = %q, want %q", got.PreferencesSnapshot, `{"topics":["go","rust"]}`)
	}
	if got.BlogsConsidered != 25 {
		t.Errorf("BlogsConsidered = %d, want %d", got.BlogsConsidered, 25)
	}
	if got.BlogsSelected != `[1,2,3]` {
		t.Errorf("BlogsSelected = %q, want %q", got.BlogsSelected, `[1,2,3]`)
	}
	if got.ModelUsed != "claude-haiku-4-5" {
		t.Errorf("ModelUsed = %q, want %q", got.ModelUsed, "claude-haiku-4-5")
	}
	if got.InputTokens == nil || *got.InputTokens != 100 {
		t.Errorf("InputTokens = %v, want 100", got.InputTokens)
	}
	if got.OutputTokens == nil || *got.OutputTokens != 50 {
		t.Errorf("OutputTokens = %v, want 50", got.OutputTokens)
	}
	if got.ResultsJSON != `[{"id":1,"title":"test"}]` {
		t.Errorf("ResultsJSON = %q, want %q", got.ResultsJSON, `[{"id":1,"title":"test"}]`)
	}
	if got.FailedFeedsJSON != `[{"source":"Bad Blog","error":"timeout"}]` {
		t.Errorf("FailedFeedsJSON = %q, want %q", got.FailedFeedsJSON, `[{"source":"Bad Blog","error":"timeout"}]`)
	}
	if got.CreatedAt.IsZero() {
		t.Error("CreatedAt is zero")
	}
}

func TestCreateSession_NilTokens(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	session := &models.DiscoverySession{
		PreferencesSnapshot: "{}",
		BlogsConsidered:     0,
		BlogsSelected:       "[]",
		ModelUsed:           "test",
	}

	id, err := store.CreateSession(ctx, session)
	if err != nil {
		t.Fatalf("CreateSession() error: %v", err)
	}

	sessions, err := store.GetRecentSessions(ctx, 10)
	if err != nil {
		t.Fatalf("GetRecentSessions() error: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("got %d sessions, want 1", len(sessions))
	}
	got := sessions[0]
	if got.ID != id {
		t.Errorf("ID = %d, want %d", got.ID, id)
	}
	if got.InputTokens != nil {
		t.Errorf("InputTokens = %v, want nil", got.InputTokens)
	}
	if got.OutputTokens != nil {
		t.Errorf("OutputTokens = %v, want nil", got.OutputTokens)
	}
	if got.ResultsJSON != "" {
		t.Errorf("ResultsJSON = %q, want empty", got.ResultsJSON)
	}
	if got.FailedFeedsJSON != "" {
		t.Errorf("FailedFeedsJSON = %q, want empty", got.FailedFeedsJSON)
	}
}

func TestGetRecentSessions_Limit(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	// Insert 5 sessions.
	for i := range 5 {
		session := &models.DiscoverySession{
			PreferencesSnapshot: "{}",
			BlogsConsidered:     i,
			BlogsSelected:       "[]",
			ModelUsed:           "test",
		}
		if _, err := store.CreateSession(ctx, session); err != nil {
			t.Fatalf("CreateSession(%d) error: %v", i, err)
		}
	}

	// Limit to 3.
	sessions, err := store.GetRecentSessions(ctx, 3)
	if err != nil {
		t.Fatalf("GetRecentSessions() error: %v", err)
	}
	if len(sessions) != 3 {
		t.Fatalf("got %d sessions, want 3", len(sessions))
	}

	// Verify DESC ordering (most recent first). Since all were inserted
	// at roughly the same time, we check that the IDs are descending.
	for i := 1; i < len(sessions); i++ {
		if sessions[i].ID >= sessions[i-1].ID {
			t.Errorf("sessions not ordered DESC: id[%d]=%d >= id[%d]=%d",
				i, sessions[i].ID, i-1, sessions[i-1].ID)
		}
	}
}

func TestGetRecentSessions_Empty(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	sessions, err := store.GetRecentSessions(ctx, 10)
	if err != nil {
		t.Fatalf("GetRecentSessions() error: %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("got %d sessions, want 0", len(sessions))
	}
}

func TestGetLatestSession(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	// No sessions yet.
	_, err := store.GetLatestSession(ctx)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got: %v", err)
	}

	// Insert a session.
	session := &models.DiscoverySession{
		PreferencesSnapshot: "topics",
		BlogsConsidered:     10,
		BlogsSelected:       "[1,2]",
		ModelUsed:           "test-model",
		ResultsJSON:         `[{"id":1}]`,
		FailedFeedsJSON:     `[]`,
	}
	id, err := store.CreateSession(ctx, session)
	if err != nil {
		t.Fatalf("CreateSession() error: %v", err)
	}

	// Retrieve it.
	got, err := store.GetLatestSession(ctx)
	if err != nil {
		t.Fatalf("GetLatestSession() error: %v", err)
	}
	if got.ID != id {
		t.Errorf("ID = %d, want %d", got.ID, id)
	}
	if got.ResultsJSON != `[{"id":1}]` {
		t.Errorf("ResultsJSON = %q, want %q", got.ResultsJSON, `[{"id":1}]`)
	}

	// Insert another and verify we get the newest.
	session2 := &models.DiscoverySession{
		PreferencesSnapshot: "topics2",
		BlogsConsidered:     20,
		BlogsSelected:       "[3,4]",
		ModelUsed:           "test-model-2",
		ResultsJSON:         `[{"id":3}]`,
	}
	id2, err := store.CreateSession(ctx, session2)
	if err != nil {
		t.Fatalf("CreateSession() error: %v", err)
	}

	got2, err := store.GetLatestSession(ctx)
	if err != nil {
		t.Fatalf("GetLatestSession() error: %v", err)
	}
	if got2.ID != id2 {
		t.Errorf("latest ID = %d, want %d", got2.ID, id2)
	}
}
