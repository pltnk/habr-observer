package repository

import (
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"habr-observer/internal/domain"
)

// feedDailyID and feedWeeklyID are real Habr feed URLs used as feed _ids.
const (
	feedDailyID  = "https://habr.com/ru/rss/articles/top/daily/?fl=ru"
	feedWeeklyID = "https://habr.com/ru/rss/articles/top/weekly/?fl=ru"
)

// TestMongoStorage_Feeds_RoundTrip verifies UpsertFeed is idempotent and that a
// feed — including its embedded articles, in order — survives the BSON round-trip.
func TestMongoStorage_Feeds_RoundTrip(t *testing.T) {
	t.Parallel()

	m := newTestStore(t)
	ctx := testContext(t)

	a1, a2 := sampleArticles()
	feed := &domain.Feed{
		ID:       feedDailyID,
		Name:     "Сутки",
		Articles: []*domain.Article{a1, a2},
	}

	for i := range 2 {
		if err := m.UpsertFeed(ctx, feed); err != nil {
			t.Fatalf("UpsertFeed (pass %d): %v", i, err)
		}
	}

	var got domain.Feed
	if err := m.feeds.FindOne(ctx, bson.M{"_id": feed.ID}).Decode(&got); err != nil {
		t.Fatalf("FindOne feed: %v", err)
	}
	if got.ID != feed.ID || got.Name != feed.Name {
		t.Errorf("feed scalar mismatch: got {ID:%q Name:%q}, want {ID:%q Name:%q}", got.ID, got.Name, feed.ID, feed.Name)
	}
	if len(got.Articles) != len(feed.Articles) {
		t.Fatalf("feed articles length: got %d, want %d", len(got.Articles), len(feed.Articles))
	}
	for i := range feed.Articles {
		assertArticleEqual(t, got.Articles[i], feed.Articles[i])
	}
}

// TestMongoStorage_Feeds_GetFeedsOrder verifies GetFeeds returns feeds in the
// requested id order — not insertion or natural order — and skips ids with no feed.
func TestMongoStorage_Feeds_GetFeedsOrder(t *testing.T) {
	t.Parallel()

	m := newTestStore(t)
	ctx := testContext(t)

	a1, a2 := sampleArticles()
	daily := &domain.Feed{ID: feedDailyID, Name: "Сутки", Articles: []*domain.Article{a1, a2}}
	weekly := &domain.Feed{ID: feedWeeklyID, Name: "Неделя", Articles: []*domain.Article{a2}}

	if err := m.UpsertFeed(ctx, daily); err != nil {
		t.Fatalf("UpsertFeed daily: %v", err)
	}
	if err := m.UpsertFeed(ctx, weekly); err != nil {
		t.Fatalf("UpsertFeed weekly: %v", err)
	}

	// Request weekly before daily (the reverse of insertion order), plus a
	// missing id that must be skipped.
	const missing = "https://habr.com/ru/rss/articles/top/missing/?fl=ru"
	got, err := m.GetFeeds(ctx, []string{weekly.ID, daily.ID, missing})
	if err != nil {
		t.Fatalf("GetFeeds: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("GetFeeds: got %d feeds, want 2 (missing id must be skipped)", len(got))
	}
	if got[0].ID != weekly.ID || got[1].ID != daily.ID {
		t.Errorf("GetFeeds order: got [%q %q], want [%q %q]", got[0].ID, got[1].ID, weekly.ID, daily.ID)
	}
	if len(got[1].Articles) != len(daily.Articles) {
		t.Fatalf("GetFeeds daily articles length: got %d, want %d", len(got[1].Articles), len(daily.Articles))
	}
	for i := range daily.Articles {
		assertArticleEqual(t, got[1].Articles[i], daily.Articles[i])
	}
}

// TestMongoStorage_Feeds_DecodeError verifies that a stored document which does
// not fit domain.Feed surfaces as a wrapped decode error, not a panic. It seeds
// a feed whose "articles" field is a string instead of an array — valid BSON,
// but undecodable — then reads it back through GetFeeds, exercising the decode
// error branch (the analog of habr's bad-RSS parse test).
func TestMongoStorage_Feeds_DecodeError(t *testing.T) {
	t.Parallel()

	m := newTestStore(t)
	ctx := testContext(t)

	// Seed through the raw collection so the malformed shape bypasses UpsertFeed.
	if _, err := m.feeds.InsertOne(ctx, bson.M{"_id": feedDailyID, "name": "broken", "articles": "not-an-array"}); err != nil {
		t.Fatalf("seeding malformed feed: %v", err)
	}

	got, err := m.GetFeeds(ctx, []string{feedDailyID})
	if err == nil {
		t.Fatalf("GetFeeds: want decode error, got nil (feeds=%+v)", got)
	}
	if got != nil {
		t.Fatalf("GetFeeds: got non-nil result %+v", got)
	}
	if !strings.Contains(err.Error(), "repository: decoding feeds") {
		t.Fatalf("GetFeeds: error %q does not contain %q", err, "repository: decoding feeds")
	}
}
