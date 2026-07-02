package updater

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"reflect"
	"strings"
	"testing"

	"habr-observer/internal/infrastructure/habr"
)

// --- fakes -------------------------------------------------------------------

// fakeFeedUpdater records which feeds Execute was called for and can be told to
// fail or panic for specific feeds. UpdateAllFeeds drives feeds sequentially, so
// no locking is needed.
type fakeFeedUpdater struct {
	calls     []habr.RSSFeed
	errFeed   map[habr.RSSFeed]error
	panicFeed map[habr.RSSFeed]bool
}

func (u *fakeFeedUpdater) Execute(ctx context.Context, f habr.RSSFeed) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	u.calls = append(u.calls, f)
	if u.panicFeed[f] {
		panic("fake feed updater boom")
	}
	return u.errFeed[f]
}

// --- helpers -----------------------------------------------------------------

func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// --- tests -------------------------------------------------------------------

func TestUpdateAllFeeds_RunsEveryFeed(t *testing.T) {
	t.Parallel()

	upd := &fakeFeedUpdater{}
	s := NewService(upd, quietLogger())

	if err := s.UpdateAllFeeds(context.Background()); err != nil {
		t.Fatalf("UpdateAllFeeds: %v", err)
	}
	if got, want := upd.calls, habr.AllFeeds(); !reflect.DeepEqual(got, want) {
		t.Fatalf("feeds updated = %v, want %v (each once, in order)", got, want)
	}
}

// TestUpdateAllFeeds_HandlesEachFeedIndependently pins how one feed's outcome is
// handled: a real failure or panic is reported in the joined error, a context
// cancellation is suppressed as expected teardown, and in every case the remaining
// feeds are still attempted.
func TestUpdateAllFeeds_HandlesEachFeedIndependently(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name         string
		upd          *fakeFeedUpdater
		wantReported bool // the offending feed appears in the returned error
	}{
		{
			name:         "error_is_reported",
			upd:          &fakeFeedUpdater{errFeed: map[habr.RSSFeed]error{habr.FeedWeekly: errors.New("feed down")}},
			wantReported: true,
		},
		{
			name:         "panic_is_recovered_and_reported",
			upd:          &fakeFeedUpdater{panicFeed: map[habr.RSSFeed]bool{habr.FeedWeekly: true}},
			wantReported: true,
		},
		{
			name:         "cancellation_is_suppressed",
			upd:          &fakeFeedUpdater{errFeed: map[habr.RSSFeed]error{habr.FeedWeekly: context.Canceled}},
			wantReported: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			s := NewService(tc.upd, quietLogger())

			err := s.UpdateAllFeeds(context.Background()) // a panicking feed must not crash the run
			if tc.wantReported {
				if err == nil || !strings.Contains(err.Error(), habr.FeedWeekly.Name()) {
					t.Fatalf("error = %v, want it to mention feed %q", err, habr.FeedWeekly.Name())
				}
			} else if err != nil {
				t.Fatalf("error = %v, want nil (cancellation suppressed)", err)
			}
			// The offending feed never aborts the batch: every feed is still attempted.
			if got, want := tc.upd.calls, habr.AllFeeds(); !reflect.DeepEqual(got, want) {
				t.Fatalf("feeds updated = %v, want all %v attempted", got, want)
			}
		})
	}
}

func TestUpdateAllFeeds_StopsOnCancellation(t *testing.T) {
	t.Parallel()

	upd := &fakeFeedUpdater{}
	s := NewService(upd, quietLogger())

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // canceled before the loop starts

	if err := s.UpdateAllFeeds(ctx); err != nil {
		t.Fatalf("UpdateAllFeeds: want nil (cancellation suppressed), got %v", err)
	}
	if got := upd.calls; len(got) != 0 {
		t.Fatalf("feeds updated = %v, want none (ctx already canceled)", got)
	}
}
