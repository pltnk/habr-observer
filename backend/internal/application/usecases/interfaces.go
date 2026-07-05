package usecases

import (
	"context"

	"habr-observer/internal/domain"
	"habr-observer/internal/infrastructure/habr"
)

// FeedRepository reads stored feed snapshots. GetFeeds returns the feeds for the
// given ids in id order, omitting any that are absent.
type FeedRepository interface {
	GetFeeds(ctx context.Context, ids []string) ([]*domain.Feed, error)
}

// FeedClient fetches the current articles of a Habr RSS feed; the returned
// articles carry no summary.
type FeedClient interface {
	GetArticles(ctx context.Context, f habr.RSSFeed) ([]*domain.Article, error)
}

// SummaryClient produces an article's summary via YandexGPT. A 404 from the
// service surfaces as [yagpt.ErrSummaryUnavailable].
type SummaryClient interface {
	GetSummary(ctx context.Context, articleURL string) (*domain.Summary, error)
}

// Repository is the write-side store [UpdateFeedUsecase] persists to. GetArticles
// returns the stored articles for the given ids, omitting any absent; the upsert
// methods are idempotent and treat empty input as a no-op.
type Repository interface {
	GetArticles(ctx context.Context, ids []string) ([]*domain.Article, error)
	UpsertArticles(ctx context.Context, articles []*domain.Article) error
	UpsertFeed(ctx context.Context, f *domain.Feed) error
}
