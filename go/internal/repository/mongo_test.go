package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"habr-observer/internal/config"
)

// TestNewMongoStorage_ConnectError verifies that construction fails with
// ErrMongoStorageCreation rather than returning a half-built store. A
// non-numeric port makes an invalid connection string, so Connect fails at
// parse time — fast and with no network I/O. The short ctx is only a backstop.
func TestNewMongoStorage_ConnectError(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := NewMongoStorage(ctx, config.MongoConfig{
		Host: "localhost:notaport", User: "user", Password: "pass",
		DB: "db", ArticlesColl: "articles", FeedsColl: "feeds",
	})
	if !errors.Is(err, ErrMongoStorageCreation) {
		t.Fatalf("NewMongoStorage: want ErrMongoStorageCreation, got %v", err)
	}
}

// TestMongoStorage_EmptyInputs verifies the read/write methods short-circuit on
// empty input without issuing a query. It runs against a zero-value store whose
// collections are nil, so any DB access would panic — proving none happens.
// Hermetic: it needs no MongoDB and so runs even under -short.
func TestMongoStorage_EmptyInputs(t *testing.T) {
	t.Parallel()

	m := &MongoStorage{} // nil collections: any DB access would panic
	ctx := context.Background()

	t.Run("GetArticles_nil", func(t *testing.T) {
		t.Parallel()
		got, err := m.GetArticles(ctx, nil)
		if err != nil || got != nil {
			t.Fatalf("GetArticles(nil) = (%v, %v), want (nil, nil)", got, err)
		}
	})

	t.Run("GetFeeds_empty", func(t *testing.T) {
		t.Parallel()
		got, err := m.GetFeeds(ctx, []string{})
		if err != nil || got != nil {
			t.Fatalf("GetFeeds([]) = (%v, %v), want (nil, nil)", got, err)
		}
	})

	t.Run("UpsertArticles_empty", func(t *testing.T) {
		t.Parallel()
		if err := m.UpsertArticles(ctx, nil); err != nil {
			t.Fatalf("UpsertArticles(nil) = %v, want nil", err)
		}
	})
}
