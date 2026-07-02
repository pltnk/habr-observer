// Package updater runs the periodic worker that refreshes every Habr feed on a
// fixed interval and reports liveness through a heartbeat file. Feeds are
// refreshed one at a time: the rate-limited summary service is the throughput
// bottleneck, so concurrency would not help.
package updater

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime/debug"

	"habr-observer/internal/infrastructure/habr"
)

// isCancellation reports whether err is context cancellation — shutdown or a
// cycle's deadline — which is expected and so suppressed from the logs.
func isCancellation(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

// Service refreshes every Habr feed once per cycle, sequentially. It holds no
// per-cycle state, so one instance may be reused and is safe for concurrent use
// if its dependencies are.
type Service struct {
	updater feedUpdater
	log     *slog.Logger
}

// New returns a [Service]. If log is nil, [slog.Default] is used.
func New(updater feedUpdater, log *slog.Logger) *Service {
	if log == nil {
		log = slog.Default()
	}
	return &Service{updater: updater, log: log}
}

// UpdateAllFeeds refreshes every Habr feed sequentially, isolating each: a real
// failure or panic is logged and collected but never aborts the rest, and
// cancellation is suppressed. It returns the joined per-feed failures, or nil if
// none occurred.
func (s *Service) UpdateAllFeeds(ctx context.Context) error {
	var errs []error

	for _, f := range habr.AllFeeds() {
		if ctx.Err() != nil {
			break // shutdown or the cycle's deadline: stop starting new feeds
		}

		if err := s.updateFeed(ctx, f); err != nil && !isCancellation(err) {
			s.log.Error("updater: feed failed", "feed", f.Name(), "err", err)
			errs = append(errs, fmt.Errorf("%s: %w", f.Name(), err))
		}
	}

	return errors.Join(errs...)
}

// updateFeed runs the feed update, recovering a panic into a logged error so one
// feed can never crash the worker.
func (s *Service) updateFeed(ctx context.Context, f habr.RSSFeed) (err error) {
	defer func() {
		if r := recover(); r != nil {
			s.log.Error("updater: feed update panicked",
				"feed", f.Name(), "panic", r, "stack", string(debug.Stack()))
			err = fmt.Errorf("panic: %v", r)
		}
	}()

	return s.updater.Execute(ctx, f)
}
