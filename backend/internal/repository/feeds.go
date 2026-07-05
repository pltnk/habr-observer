package repository

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"habr-observer/internal/domain"
)

// UpsertFeed replaces the feed document identified by its _id (the feed URL),
// inserting it if absent. The entire denormalized feed — including its embedded
// articles, in their given order — is written as a single document.
func (m *MongoStorage) UpsertFeed(ctx context.Context, f *domain.Feed) error {
	opts := options.Replace().SetUpsert(true)
	if _, err := m.feeds.ReplaceOne(ctx, bson.M{"_id": f.ID}, f, opts); err != nil {
		return fmt.Errorf("repository: upserting feed %q: %w", f.ID, err)
	}
	return nil
}

// GetFeeds returns the feeds whose _id (the feed URL) is in ids, each with its
// embedded articles. The result preserves the order of ids — unlike
// [MongoStorage.GetArticles] — and skips ids with no matching feed, so it may
// be shorter than ids. An empty ids slice returns nil without querying.
func (m *MongoStorage) GetFeeds(ctx context.Context, ids []string) ([]*domain.Feed, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	// The __order helper field is a decode-time artifact; the BSON decoder
	// drops it when populating domain.Feed, which has no matching field.
	pipeline := mongo.Pipeline{
		bson.D{{Key: "$match", Value: bson.M{"_id": bson.M{"$in": ids}}}},
		bson.D{{Key: "$addFields", Value: bson.M{"__order": bson.M{"$indexOfArray": bson.A{ids, "$_id"}}}}},
		bson.D{{Key: "$sort", Value: bson.M{"__order": 1}}},
	}

	cur, err := m.feeds.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("repository: aggregating feeds: %w", err)
	}

	var feeds []*domain.Feed
	if err := cur.All(ctx, &feeds); err != nil {
		return nil, fmt.Errorf("repository: decoding feeds: %w", err)
	}

	return feeds, nil
}
