package usecases

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"habr-observer/internal/domain"
)

// --- fakes -------------------------------------------------------------------

// Compile-time guard that the fake still satisfies the port it stands in for.
var _ FeedRepository = (*fakeFeedRepo)(nil)

// fakeFeedRepo is a controllable FeedRepository: it records each call and
// returns canned feeds or a fixed error.
type fakeFeedRepo struct {
	feeds  []*domain.Feed
	err    error
	calls  int
	gotIDs [][]string
}

func (r *fakeFeedRepo) GetFeeds(_ context.Context, ids []string) ([]*domain.Feed, error) {
	r.calls++
	r.gotIDs = append(r.gotIDs, append([]string(nil), ids...))
	if r.err != nil {
		return nil, r.err
	}
	return r.feeds, nil
}

// --- helpers -----------------------------------------------------------------

func sampleFeeds() []*domain.Feed {
	return []*domain.Feed{
		{ID: "https://habr.com/ru/rss/articles/top/daily/?fl=ru", Name: "Сутки"},
		{ID: "https://habr.com/ru/rss/articles/top/weekly/?fl=ru", Name: "Неделя"},
	}
}

// --- tests -------------------------------------------------------------------

func TestGetFeedsUsecase_Execute_CachesWithinTTL(t *testing.T) {
	t.Parallel()

	repo := &fakeFeedRepo{feeds: sampleFeeds()}
	ids := []string{"daily", "weekly"}
	uc := NewGetFeedsUsecase(repo, ids, time.Minute, quietLogger())

	clock := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	uc.now = func() time.Time { return clock }

	got, err := uc.Execute(context.Background())
	if err != nil {
		t.Fatalf("Execute (cold): %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d feeds, want 2", len(got))
	}
	if repo.calls != 1 {
		t.Fatalf("repo calls = %d, want 1 on cold cache", repo.calls)
	}
	if len(repo.gotIDs) == 0 || !reflect.DeepEqual(repo.gotIDs[0], ids) {
		t.Fatalf("repo got ids %v, want %v", repo.gotIDs, ids)
	}

	// A second call within the ttl serves the same snapshot from cache; the
	// repo is untouched.
	warm, err := uc.Execute(context.Background())
	if err != nil {
		t.Fatalf("Execute (warm): %v", err)
	}
	if !reflect.DeepEqual(warm, got) {
		t.Fatalf("Execute (warm) = %+v, want the cached snapshot %+v", warm, got)
	}
	if repo.calls != 1 {
		t.Fatalf("repo calls = %d, want 1 (served from cache)", repo.calls)
	}
}

func TestGetFeedsUsecase_Execute_RefreshesAfterTTL(t *testing.T) {
	t.Parallel()

	repo := &fakeFeedRepo{feeds: sampleFeeds()}
	uc := NewGetFeedsUsecase(repo, []string{"daily"}, time.Minute, quietLogger())

	clock := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	uc.now = func() time.Time { return clock }

	if _, err := uc.Execute(context.Background()); err != nil {
		t.Fatalf("Execute (cold): %v", err)
	}
	if repo.calls != 1 {
		t.Fatalf("repo calls = %d, want 1", repo.calls)
	}

	// Past the ttl, the next call reloads from the repository.
	clock = clock.Add(time.Minute + time.Second)
	if _, err := uc.Execute(context.Background()); err != nil {
		t.Fatalf("Execute (expired): %v", err)
	}
	if repo.calls != 2 {
		t.Fatalf("repo calls = %d, want 2 after expiry", repo.calls)
	}
}

// TestGetFeedsUsecase_Execute_ColdCacheError pins the only failing path: a
// repository error before a first snapshot exists, with nothing stale to serve.
func TestGetFeedsUsecase_Execute_ColdCacheError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("boom")
	repo := &fakeFeedRepo{err: wantErr}
	uc := NewGetFeedsUsecase(repo, []string{"daily"}, time.Minute, quietLogger())

	got, err := uc.Execute(context.Background())
	if !errors.Is(err, wantErr) {
		t.Fatalf("Execute error = %v, want a wrap of %v", err, wantErr)
	}
	if !strings.Contains(err.Error(), "getting feeds") {
		t.Fatalf("Execute error = %q, want it to contain %q", err, "getting feeds")
	}
	if got != nil {
		t.Fatalf("Execute returned %v, want nil on error", got)
	}

	// A failed load must not be cached: the next call retries the repository.
	if _, err := uc.Execute(context.Background()); !errors.Is(err, wantErr) {
		t.Fatalf("Execute (retry) error = %v, want a wrap of %v", err, wantErr)
	}
	if repo.calls != 2 {
		t.Fatalf("repo calls = %d, want 2 (an error must not be cached)", repo.calls)
	}
}

// TestGetFeedsUsecase_Execute_ServesStaleOnReloadError pins the degraded mode: a
// failed reload of an expired cache — a repository failure or a cancellation
// alike — serves the previous snapshot instead of erroring, and does not extend
// its freshness, so every later call keeps retrying until one succeeds.
func TestGetFeedsUsecase_Execute_ServesStaleOnReloadError(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		err  error
	}{
		{name: "transient", err: errors.New("boom")},
		{name: "cancellation", err: context.Canceled},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			repo := &fakeFeedRepo{feeds: sampleFeeds()}
			uc := NewGetFeedsUsecase(repo, []string{"daily"}, time.Minute, quietLogger())

			clock := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
			uc.now = func() time.Time { return clock }

			want, err := uc.Execute(context.Background())
			if err != nil {
				t.Fatalf("Execute (cold): %v", err)
			}

			// The cache expires and the reload starts failing.
			clock = clock.Add(time.Minute + time.Second)
			repo.err = tc.err

			got, err := uc.Execute(context.Background())
			if err != nil {
				t.Fatalf("Execute (stale): %v, want nil (stale snapshot served)", err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("Execute (stale) = %+v, want the previous snapshot %+v", got, want)
			}

			// The failure did not extend freshness: the next call retries the
			// reload, and once the repository recovers the snapshot is refreshed.
			repo.err = nil
			if _, err := uc.Execute(context.Background()); err != nil {
				t.Fatalf("Execute (recovered): %v", err)
			}
			if repo.calls != 3 {
				t.Fatalf("repo calls = %d, want 3 (cold, failed reload, retry)", repo.calls)
			}
		})
	}
}
