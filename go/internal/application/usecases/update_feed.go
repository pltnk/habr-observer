package usecases

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime/debug"
	"sync"

	"habr-observer/internal/domain"
	"habr-observer/internal/infrastructure/habr"
	"habr-observer/internal/infrastructure/yagpt"
)

const (
	// placeholderSummaryURL and placeholderSummaryMessage are stored for articles
	// the service cannot summarize (signalled by [yagpt.ErrSummaryUnavailable],
	// e.g. an image-only post with little summarizable text), so the article is
	// recorded once and never re-attempted on later runs.
	placeholderSummaryURL     = "https://300.ya.ru"
	placeholderSummaryMessage = "Не получилось пересказать статью, лучше сразу посмотреть оригинал 🙂"
)

// UpdateFeedUsecase refreshes a single Habr feed: it fetches the feed's current
// articles, summarizes the ones not yet stored, and persists both the new
// articles and a denormalized per-feed snapshot. It holds no per-call state, so
// a single instance may be reused and is safe for concurrent use as long as its
// dependencies are.
//
// An UpdateFeedUsecase must be created with [NewUpdateFeedUsecase]; the zero
// value is not usable.
type UpdateFeedUsecase struct {
	feeds   FeedClient
	summary SummaryClient
	repo    Repository
	log     *slog.Logger
}

// NewUpdateFeedUsecase returns a use case backed by the given ports. If log is
// nil, [slog.Default] is used.
func NewUpdateFeedUsecase(feeds FeedClient, summary SummaryClient, repo Repository, log *slog.Logger) *UpdateFeedUsecase {
	if log == nil {
		log = slog.Default()
	}

	return &UpdateFeedUsecase{feeds: feeds, summary: summary, repo: repo, log: log}
}

// Execute refreshes one feed: fetch its current articles, determine which are
// new, summarize and persist those, then write a feed snapshot whose articles
// preserve the feed's original (RSS) order.
//
// It returns an error only when the feed source or the repository fails. A
// single article that cannot be summarized is logged and skipped, and a 404
// from the summary service is stored as a placeholder — neither fails the feed.
func (u *UpdateFeedUsecase) Execute(ctx context.Context, f habr.RSSFeed) error {
	feedArticles, err := u.feeds.GetArticles(ctx, f)
	if err != nil {
		return fmt.Errorf("updating feed %q: fetching feed: %w", f.Name(), err)
	}

	summarized, err := u.repo.GetArticles(ctx, articleIDs(feedArticles))
	if err != nil {
		return fmt.Errorf("updating feed %q: loading summarized articles: %w", f.Name(), err)
	}

	// For articles we already have summarized, swap the stored version into the
	// feed slot (it carries its summary); collect the rest to summarize.
	summarizedByID := make(map[string]*domain.Article)
	for _, a := range summarized {
		summarizedByID[a.ID] = a
	}
	var toSummarize []*domain.Article
	for i, a := range feedArticles {
		if s, ok := summarizedByID[a.ID]; ok {
			feedArticles[i] = s
		} else {
			toSummarize = append(toSummarize, a)
		}
	}

	// Summarize the rest; each success sets the article's Summary in place.
	toStore := u.summarizeArticles(ctx, toSummarize)
	if err := u.repo.UpsertArticles(ctx, toStore); err != nil {
		return fmt.Errorf("updating feed %q: upserting articles: %w", f.Name(), err)
	}

	// feedArticles is already in feed order, so the snapshot is simply the ones
	// that now carry a summary. A new article that failed to summarize has none,
	// so it is left out.
	snapshot := make([]*domain.Article, 0, len(feedArticles))
	for _, a := range feedArticles {
		if a.Summary != nil {
			snapshot = append(snapshot, a)
		}
	}

	feed := &domain.Feed{ID: f.URL(), Name: f.Name(), Articles: snapshot}
	if err := u.repo.UpsertFeed(ctx, feed); err != nil {
		return fmt.Errorf("updating feed %q: upserting feed: %w", f.Name(), err)
	}

	return nil
}

// articleIDs returns the IDs of the given articles, in the same order.
func articleIDs(arts []*domain.Article) []string {
	ids := make([]string, len(arts))
	for i, a := range arts {
		ids[i] = a.ID
	}
	return ids
}

// summarizeArticles summarizes the given articles concurrently (one goroutine
// per article) and returns the ones to persist — those that got a real
// summary or the placeholder, each with its Summary set. An article whose
// summary fails for a real reason is logged (with its URL) and dropped;
// cancellation is suppressed. A panic skips only that article. Every spawned
// goroutine is joined before returning, so none leaks.
func (u *UpdateFeedUsecase) summarizeArticles(ctx context.Context, articles []*domain.Article) []*domain.Article {
	if len(articles) == 0 {
		return nil
	}

	// Index-aligned output: each goroutine owns one slot, so no lock is needed.
	// A nil slot means the summary was skipped this run.
	out := make([]*domain.Article, len(articles))

	var wg sync.WaitGroup
	for i, a := range articles {
		wg.Go(func() {
			// Recover so one article's panic skips only that article.
			defer func() {
				if r := recover(); r != nil {
					u.log.Error("updating feed: summary panicked; skipping article",
						"url", a.ID, "panic", r, "stack", string(debug.Stack()))
				}
			}()

			summary, err := u.summarize(ctx, a)
			if err != nil {
				if !isCancellation(err) {
					u.log.Error("updating feed: summary failed; skipping article", "url", a.ID, "err", err)
				}
				return // skip this article; never abort the batch
			}

			a.Summary = summary
			out[i] = a
		})
	}
	wg.Wait()

	toStore := make([]*domain.Article, 0, len(out))
	for _, a := range out {
		if a != nil {
			toStore = append(toStore, a)
		}
	}

	return toStore
}

// summarize produces the summary for a single article and applies the app-level
// placeholder policy: a service-reported "unavailable" ([yagpt.ErrSummaryUnavailable])
// yields the stored placeholder. Any other error is transient: it is returned so
// the caller skips the article and a later run can retry it.
func (u *UpdateFeedUsecase) summarize(ctx context.Context, a *domain.Article) (*domain.Summary, error) {
	summary, err := u.summary.GetSummary(ctx, a.ID)

	if errors.Is(err, yagpt.ErrSummaryUnavailable) {
		return &domain.Summary{
			URL:     placeholderSummaryURL,
			Content: []string{placeholderSummaryMessage},
		}, nil
	}

	if err != nil {
		return nil, err
	}

	return summary, nil
}

// isCancellation reports whether err is context cancellation — the run's
// deadline or a shutdown. Such errors are expected, not failures, so a canceled
// per-article summary is suppressed from the logs.
func isCancellation(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}
