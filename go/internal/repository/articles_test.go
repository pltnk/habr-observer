package repository

import (
	"strings"
	"testing"

	"habr-observer/internal/domain"
)

// TestMongoStorage_Articles_RoundTrip exercises the article write/read path
// against a real MongoDB: upserts are idempotent (safe to repeat), GetArticles
// returns only the stored ids, and articles survive the BSON round-trip intact.
func TestMongoStorage_Articles_RoundTrip(t *testing.T) {
	t.Parallel()

	m := newTestStore(t)
	ctx := testContext(t)

	a1, a2 := sampleArticles()

	// Upsert twice to prove idempotency (no duplicate-key error on re-write).
	for i := range 2 {
		if err := m.UpsertArticles(ctx, []*domain.Article{a1, a2}); err != nil {
			t.Fatalf("UpsertArticles (pass %d): %v", i, err)
		}
	}

	const missing = "https://habr.com/ru/articles/missing/"
	got, err := m.GetArticles(ctx, []string{a1.ID, a2.ID, missing})
	if err != nil {
		t.Fatalf("GetArticles: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("GetArticles: got %d articles, want 2 (missing id must be skipped)", len(got))
	}

	byID := make(map[string]*domain.Article, len(got))
	for _, a := range got {
		byID[a.ID] = a
	}
	assertArticleEqual(t, byID[a1.ID], a1)
	assertArticleEqual(t, byID[a2.ID], a2)
}

// TestMongoStorage_Articles_OperationError verifies a query failure surfaces as
// a wrapped repository error rather than a panic or a bare driver error. It
// closes the store first: the driver then rejects operations on the
// disconnected client, exercising GetArticles' Find error branch. Nothing is
// written, so there is no database to drop and no teardown to register.
func TestMongoStorage_Articles_OperationError(t *testing.T) {
	t.Parallel()

	ctx := testContext(t)
	m := connectTestStore(t, testDBName(t))

	if err := m.Close(ctx); err != nil {
		t.Fatalf("Close: %v", err)
	}

	got, err := m.GetArticles(ctx, []string{"https://habr.com/ru/articles/1/"})
	if err == nil {
		t.Fatalf("GetArticles after Close: want error, got nil (articles=%+v)", got)
	}
	if got != nil {
		t.Fatalf("GetArticles after Close: got non-nil result %+v", got)
	}
	if !strings.Contains(err.Error(), "repository: finding articles") {
		t.Fatalf("GetArticles after Close: error %q does not contain %q", err, "repository: finding articles")
	}
}
