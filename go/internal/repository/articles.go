package repository

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"habr-observer/internal/domain"
)

// GetArticles returns the subset of ids (article URLs) that already exist in
// the store, decoded as articles. The result order is unspecified — MongoDB
// does not preserve $in order — so callers needing a particular order must sort
// the results themselves. An empty ids slice returns nil without querying.
func (m *MongoStorage) GetArticles(ctx context.Context, ids []string) ([]*domain.Article, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	cur, err := m.articles.Find(ctx, bson.M{"_id": bson.M{"$in": ids}})
	if err != nil {
		return nil, fmt.Errorf("repository: finding articles: %w", err)
	}

	var articles []*domain.Article
	if err := cur.All(ctx, &articles); err != nil {
		return nil, fmt.Errorf("repository: decoding articles: %w", err)
	}

	return articles, nil
}

// UpsertArticles writes each article by its _id, inserting or replacing. It is
// idempotent, which keeps the updater race-safe when several feeds surface the
// same new article concurrently. Writes are unordered, so one failing write
// does not abort the batch. An empty slice is a no-op.
func (m *MongoStorage) UpsertArticles(ctx context.Context, articles []*domain.Article) error {
	if len(articles) == 0 {
		return nil
	}

	models := make([]mongo.WriteModel, len(articles))
	for i, a := range articles {
		models[i] = mongo.NewReplaceOneModel().
			SetFilter(bson.M{"_id": a.ID}).
			SetReplacement(a).
			SetUpsert(true)
	}

	opts := options.BulkWrite().SetOrdered(false)
	if _, err := m.articles.BulkWrite(ctx, models, opts); err != nil {
		return fmt.Errorf("repository: upserting %d articles: %w", len(articles), err)
	}

	return nil
}
