// Package usecases holds the application's interactors — single-operation units
// of work that wire domain and infrastructure ports together and expose their
// work through an Execute method. Read-side use cases (e.g. [GetFeedsUsecase])
// serve a delivery-layer request; write-side use cases (e.g. [UpdateFeedUsecase])
// are driven by the background services in internal/application/services.
package usecases

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"habr-observer/internal/domain"
)

// GetFeedsUsecase returns all configured feeds. It caches the full feed set in
// memory and reloads it from the repository once the snapshot is older than
// ttl, so most reads never reach the repository. If a reload fails, it serves
// the stale snapshot instead. It is safe for concurrent use.
//
// A GetFeedsUsecase must be created with [NewGetFeedsUsecase]; the zero value is
// not usable.
type GetFeedsUsecase struct {
	repo FeedRepository
	ids  []string // canonical, ordered feed ids to load
	ttl  time.Duration
	now  func() time.Time // overridable in tests; defaults to time.Now
	log  *slog.Logger

	mu        sync.Mutex
	cache     []*domain.Feed
	expiresAt time.Time
}

// NewGetFeedsUsecase returns a use case that serves the feeds identified by ids,
// in that order, caching them for ttl. Passing ids in keeps this package
// independent of the feed catalogue. If log is nil, [slog.Default] is used.
func NewGetFeedsUsecase(repo FeedRepository, ids []string, ttl time.Duration, log *slog.Logger) *GetFeedsUsecase {
	if log == nil {
		log = slog.Default()
	}

	return &GetFeedsUsecase{
		repo: repo,
		ids:  ids,
		ttl:  ttl,
		now:  time.Now,
		log:  log,
	}
}

// Execute returns every configured feed in canonical order. It serves the cached
// snapshot while it is fresh and otherwise reloads it from the repository. If a
// reload fails, the error is logged and the stale snapshot is served; Execute
// returns an error only when the repository fails before a first snapshot
// exists. The returned feeds are shared, read-only snapshots: callers must not
// mutate them.
//
// Concurrent callers that arrive on a cold or expired cache share a single
// reload rather than each hitting the repository.
func (u *GetFeedsUsecase) Execute(ctx context.Context) ([]*domain.Feed, error) {
	u.mu.Lock()
	defer u.mu.Unlock()

	if u.now().Before(u.expiresAt) {
		return u.cache, nil
	}

	feeds, err := u.repo.GetFeeds(ctx, u.ids)
	if err != nil {
		if u.expiresAt.IsZero() { // cold cache: no snapshot to fall back on
			return nil, fmt.Errorf("getting feeds: %w", err)
		}
		// Freshness is not extended, so the next call retries the reload.
		u.log.Error("getting feeds failed; serving stale cache", "err", err)
		return u.cache, nil
	}

	u.cache, u.expiresAt = feeds, u.now().Add(u.ttl)
	return feeds, nil
}
