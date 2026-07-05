package usecases

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"habr-observer/internal/domain"
	"habr-observer/internal/infrastructure/habr"
	"habr-observer/internal/infrastructure/yagpt"
)

// --- test data ---------------------------------------------------------------

const (
	// summaryURL is the sharing URL the fake summary client returns for an
	// article it freshly summarizes.
	summaryURL = "https://300.ya.ru/tok123"
	// storedSummaryURL tags a summary that came from the repository, so a test can
	// tell a reused (already-stored) summary apart from a freshly computed one.
	storedSummaryURL = "https://300.ya.ru/stored"
)

// summaryContent is the canned thesis content of a freshly computed summary;
// storedSummaryContent is the content of one that came from the repository.
var (
	summaryContent       = []string{"thesis one", "thesis two"}
	storedSummaryContent = []string{"stored"}
)

// --- fakes -------------------------------------------------------------------

// Compile-time guards that each fake still satisfies the port it stands in for,
// so a drift in an interface signature fails here rather than at a call site.
var (
	_ FeedClient    = (*fakeFeedClient)(nil)
	_ SummaryClient = (*fakeSummaryClient)(nil)
	_ Repository    = (*fakeRepo)(nil)
)

// fakeFeedClient returns the canned feed articles, or err if set. It hands back a
// deep copy so the use case's in-place mutations never touch the seed. It ignores
// the feed argument, since every test drives a single feed.
type fakeFeedClient struct {
	articles []*domain.Article
	err      error
}

func (f *fakeFeedClient) GetArticles(ctx context.Context, _ habr.RSSFeed) ([]*domain.Article, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if f.err != nil {
		return nil, f.err
	}
	return cloneArticles(f.articles), nil
}

// fakeSummaryClient summarizes an article URL, with per-URL behavior overrides.
// It is safe for concurrent use: the use case summarizes articles in parallel.
type fakeSummaryClient struct {
	mu sync.Mutex
	// urlErr maps an article URL to the error GetSummary returns for it.
	// yagpt.ErrSummaryUnavailable yields a placeholder; any other error skips it.
	urlErr map[string]error
	// panicURL marks URLs for which GetSummary panics.
	panicURL map[string]bool
	// blockURL marks URLs for which GetSummary blocks until ctx is canceled.
	blockURL map[string]bool
	// called records, in call order, the article URLs passed to GetSummary.
	called []string
}

func (f *fakeSummaryClient) GetSummary(ctx context.Context, articleURL string) (*domain.Summary, error) {
	f.mu.Lock()
	f.called = append(f.called, articleURL)
	doPanic := f.panicURL[articleURL]
	block := f.blockURL[articleURL]
	urlErr := f.urlErr[articleURL]
	f.mu.Unlock()

	switch {
	case doPanic:
		panic("fake summary client boom")
	case block:
		<-ctx.Done() // honor ctx so the goroutine is joinable at the deadline
		return nil, ctx.Err()
	case urlErr != nil:
		return nil, urlErr
	}
	return &domain.Summary{URL: summaryURL, Content: append([]string(nil), summaryContent...)}, nil
}

func (f *fakeSummaryClient) calledURLs() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.called...)
}

// fakeRepo is a stub+spy: GetArticles returns the seeded already-stored set, and
// the upsert methods capture their input (or return an injected error). The use
// case touches the repo only sequentially, so it needs no locking.
type fakeRepo struct {
	present []*domain.Article // articles treated as already stored

	getErr    error // returned by GetArticles
	upsertErr error // returned by UpsertArticles
	feedErr   error // returned by UpsertFeed

	upsertedArticles []*domain.Article
	upsertedFeeds    []*domain.Feed
}

func (r *fakeRepo) GetArticles(_ context.Context, ids []string) ([]*domain.Article, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	byID := make(map[string]*domain.Article, len(r.present))
	for _, a := range r.present {
		byID[a.ID] = a
	}
	var out []*domain.Article
	for _, id := range ids {
		if a, ok := byID[id]; ok {
			out = append(out, cloneArticle(a))
		}
	}
	return out, nil
}

func (r *fakeRepo) UpsertArticles(_ context.Context, articles []*domain.Article) error {
	if r.upsertErr != nil {
		return r.upsertErr
	}
	r.upsertedArticles = append(r.upsertedArticles, articles...)
	return nil
}

func (r *fakeRepo) UpsertFeed(_ context.Context, f *domain.Feed) error {
	if r.feedErr != nil {
		return r.feedErr
	}
	r.upsertedFeeds = append(r.upsertedFeeds, f)
	return nil
}

// --- helpers -----------------------------------------------------------------

func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// art builds a feed article carrying no summary, as the feed client returns it.
func art(id string) *domain.Article {
	return &domain.Article{ID: id, Title: "title", Author: "author"}
}

// storedArt builds an already-stored article, carrying a summary tagged with
// storedSummaryURL so a test can tell it apart from a freshly computed one.
func storedArt(id string) *domain.Article {
	a := art(id)
	a.Summary = &domain.Summary{URL: storedSummaryURL, Content: append([]string(nil), storedSummaryContent...)}
	return a
}

func cloneArticle(a *domain.Article) *domain.Article {
	if a == nil {
		return nil
	}
	c := &domain.Article{ID: a.ID, Title: a.Title, PubDate: a.PubDate, Author: a.Author}
	if a.Summary != nil {
		c.Summary = &domain.Summary{URL: a.Summary.URL, Content: append([]string(nil), a.Summary.Content...)}
	}
	return c
}

func cloneArticles(in []*domain.Article) []*domain.Article {
	if in == nil {
		return nil
	}
	out := make([]*domain.Article, len(in))
	for i, a := range in {
		out[i] = cloneArticle(a)
	}
	return out
}

// idsOf returns the IDs of arts in order. Defined here (rather than reusing the
// production articleIDs) so the assertions don't lean on the code under test.
func idsOf(arts []*domain.Article) []string {
	out := make([]string, len(arts))
	for i, a := range arts {
		out[i] = a.ID
	}
	return out
}

// feedWith builds a fake feed client serving the given articles.
func feedWith(arts ...*domain.Article) *fakeFeedClient {
	return &fakeFeedClient{articles: arts}
}

// --- tests -------------------------------------------------------------------

// TestUpdateFeedUsecase_Execute_IOErrors pins that each of the four I/O failure
// points returns an error wrapped with its stage context and still unwrappable to
// the underlying error. These are the only conditions under which Execute fails.
func TestUpdateFeedUsecase_Execute_IOErrors(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("io boom")

	cases := []struct {
		name    string
		feeds   *fakeFeedClient
		repo    *fakeRepo
		wantCtx string
	}{
		{
			name:    "fetching_feed",
			feeds:   &fakeFeedClient{err: wantErr},
			repo:    &fakeRepo{},
			wantCtx: "fetching feed",
		},
		{
			name:    "loading_summarized_articles",
			feeds:   feedWith(art("https://habr.com/a/")),
			repo:    &fakeRepo{getErr: wantErr},
			wantCtx: "loading summarized articles",
		},
		{
			name:    "upserting_articles",
			feeds:   feedWith(art("https://habr.com/a/")),
			repo:    &fakeRepo{upsertErr: wantErr},
			wantCtx: "upserting articles",
		},
		{
			name:    "upserting_feed",
			feeds:   feedWith(art("https://habr.com/a/")),
			repo:    &fakeRepo{feedErr: wantErr},
			wantCtx: "upserting feed",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			uc := NewUpdateFeedUsecase(tc.feeds, &fakeSummaryClient{}, tc.repo, quietLogger())

			err := uc.Execute(context.Background(), habr.FeedDaily)
			if err == nil {
				t.Fatalf("Execute: want error, got nil")
			}
			if !strings.Contains(err.Error(), tc.wantCtx) {
				t.Fatalf("Execute error = %q, want it to contain %q", err, tc.wantCtx)
			}
			if !errors.Is(err, wantErr) {
				t.Fatalf("Execute error = %q, want it to wrap the injected error", err)
			}
		})
	}
}

func TestUpdateFeedUsecase_Execute_ContextCanceled(t *testing.T) {
	t.Parallel()

	uc := NewUpdateFeedUsecase(feedWith(art("https://habr.com/a/")), &fakeSummaryClient{}, &fakeRepo{}, quietLogger())

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already canceled

	if err := uc.Execute(ctx, habr.FeedDaily); !errors.Is(err, context.Canceled) {
		t.Fatalf("Execute: want context.Canceled, got %v", err)
	}
}

// TestUpdateFeedUsecase_Execute_BuildsSnapshotInFeedOrder is the core happy path:
// the snapshot is the feed in RSS order, already-stored articles are reused with
// their stored summary (never re-summarized), new ones are freshly summarized,
// and only the new ones are persisted.
func TestUpdateFeedUsecase_Execute_BuildsSnapshotInFeedOrder(t *testing.T) {
	t.Parallel()

	aID, bID, cID := "https://habr.com/a/", "https://habr.com/b/", "https://habr.com/c/"

	feeds := feedWith(art(aID), art(bID), art(cID))
	repo := &fakeRepo{present: []*domain.Article{storedArt(bID)}} // b already stored
	summarizer := &fakeSummaryClient{}

	uc := NewUpdateFeedUsecase(feeds, summarizer, repo, quietLogger())
	if err := uc.Execute(context.Background(), habr.FeedDaily); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if len(repo.upsertedFeeds) != 1 {
		t.Fatalf("UpsertFeed calls = %d, want 1", len(repo.upsertedFeeds))
	}
	feed := repo.upsertedFeeds[0]

	// The snapshot is identified by the source feed's URL and name.
	if got, want := feed.ID, habr.FeedDaily.URL(); got != want {
		t.Fatalf("feed ID = %q, want %q", got, want)
	}
	if got, want := feed.Name, habr.FeedDaily.Name(); got != want {
		t.Fatalf("feed Name = %q, want %q", got, want)
	}

	// The whole feed, in order.
	snapshot := feed.Articles
	if got, want := idsOf(snapshot), []string{aID, bID, cID}; !reflect.DeepEqual(got, want) {
		t.Fatalf("snapshot order = %v, want %v", got, want)
	}

	// The stored article was not re-summarized; the new ones were.
	called := summarizer.calledURLs()
	if slices.Contains(called, bID) {
		t.Fatalf("summary client was called for the already-stored article %q", bID)
	}
	for _, id := range []string{aID, cID} {
		if !slices.Contains(called, id) {
			t.Fatalf("new article %q was not summarized (called = %v)", id, called)
		}
	}

	// Each snapshot article carries the expected summary in full: the stored one
	// keeps its stored summary, the new ones get the freshly computed one.
	wantSummary := map[string]*domain.Summary{
		aID: {URL: summaryURL, Content: summaryContent},
		bID: {URL: storedSummaryURL, Content: storedSummaryContent},
		cID: {URL: summaryURL, Content: summaryContent},
	}
	for _, a := range snapshot {
		if !reflect.DeepEqual(a.Summary, wantSummary[a.ID]) {
			t.Fatalf("snapshot article %q summary = %+v, want %+v", a.ID, a.Summary, wantSummary[a.ID])
		}
	}

	// Only the two new articles are persisted.
	if got, want := idsOf(repo.upsertedArticles), []string{aID, cID}; !reflect.DeepEqual(got, want) {
		t.Fatalf("upserted articles = %v, want %v", got, want)
	}
}

func TestUpdateFeedUsecase_Execute_SkipsAlreadyStored(t *testing.T) {
	t.Parallel()

	aID, bID := "https://habr.com/a/", "https://habr.com/b/"

	feeds := feedWith(art(aID), art(bID))
	repo := &fakeRepo{present: []*domain.Article{storedArt(aID), storedArt(bID)}}
	summarizer := &fakeSummaryClient{}

	uc := NewUpdateFeedUsecase(feeds, summarizer, repo, quietLogger())
	if err := uc.Execute(context.Background(), habr.FeedDaily); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if got := summarizer.calledURLs(); len(got) != 0 {
		t.Fatalf("summary client called for %v, want no calls", got)
	}
	if got := len(repo.upsertedArticles); got != 0 {
		t.Fatalf("upserted articles = %d, want 0 (nothing new)", got)
	}
	if got, want := idsOf(repo.upsertedFeeds[0].Articles), []string{aID, bID}; !reflect.DeepEqual(got, want) {
		t.Fatalf("snapshot = %v, want %v", got, want)
	}
}

func TestUpdateFeedUsecase_Execute_StoresPlaceholderForUnavailable(t *testing.T) {
	t.Parallel()

	aID := "https://habr.com/imageonly/"

	feeds := feedWith(art(aID))
	repo := &fakeRepo{}
	summarizer := &fakeSummaryClient{urlErr: map[string]error{aID: yagpt.ErrSummaryUnavailable}}

	uc := NewUpdateFeedUsecase(feeds, summarizer, repo, quietLogger())
	if err := uc.Execute(context.Background(), habr.FeedDaily); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	// The article is persisted with the placeholder summary...
	if len(repo.upsertedArticles) != 1 {
		t.Fatalf("upserted articles = %d, want 1", len(repo.upsertedArticles))
	}
	want := &domain.Summary{URL: placeholderSummaryURL, Content: []string{placeholderSummaryMessage}}
	if got := repo.upsertedArticles[0].Summary; !reflect.DeepEqual(got, want) {
		t.Fatalf("placeholder summary = %+v, want %+v", got, want)
	}

	// ...and still appears in the feed snapshot.
	if got, want := idsOf(repo.upsertedFeeds[0].Articles), []string{aID}; !reflect.DeepEqual(got, want) {
		t.Fatalf("snapshot = %v, want %v", got, want)
	}
}

// TestUpdateFeedUsecase_Execute_SkipsFailedArticle pins that a per-article summary
// failure — a transient error or a context cancellation alike — drops only that
// article and never aborts the batch: the healthy sibling is still summarized and
// persisted.
func TestUpdateFeedUsecase_Execute_SkipsFailedArticle(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		err  error
	}{
		{name: "transient", err: errors.New("network boom")},
		{name: "cancellation", err: context.Canceled},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			badID, goodID := "https://habr.com/bad/", "https://habr.com/good/"

			feeds := feedWith(art(badID), art(goodID))
			repo := &fakeRepo{}
			summarizer := &fakeSummaryClient{urlErr: map[string]error{badID: tc.err}}

			uc := NewUpdateFeedUsecase(feeds, summarizer, repo, quietLogger())
			if err := uc.Execute(context.Background(), habr.FeedDaily); err != nil {
				t.Fatalf("Execute: %v", err)
			}

			// The failed article is dropped from the store and the snapshot; the good one stays.
			if got, want := idsOf(repo.upsertedArticles), []string{goodID}; !reflect.DeepEqual(got, want) {
				t.Fatalf("upserted articles = %v, want %v", got, want)
			}
			if got, want := idsOf(repo.upsertedFeeds[0].Articles), []string{goodID}; !reflect.DeepEqual(got, want) {
				t.Fatalf("snapshot = %v, want %v", got, want)
			}
		})
	}
}

func TestUpdateFeedUsecase_Execute_RecoversFromSummaryPanic(t *testing.T) {
	t.Parallel()

	panicID, goodID := "https://habr.com/panic/", "https://habr.com/good/"

	feeds := feedWith(art(panicID), art(goodID))
	repo := &fakeRepo{}
	summarizer := &fakeSummaryClient{panicURL: map[string]bool{panicID: true}}

	uc := NewUpdateFeedUsecase(feeds, summarizer, repo, quietLogger())
	if err := uc.Execute(context.Background(), habr.FeedDaily); err != nil { // recovered, so no crash
		t.Fatalf("Execute: %v", err)
	}

	// The panicking article is isolated and dropped; the good one is unaffected.
	// That Execute returns at all proves the panic was recovered — an unrecovered
	// panic in a summary goroutine would crash the test binary.
	if got, want := idsOf(repo.upsertedArticles), []string{goodID}; !reflect.DeepEqual(got, want) {
		t.Fatalf("upserted articles = %v, want %v", got, want)
	}
}

// TestUpdateFeedUsecase_Execute_NoGoroutineLeak proves Execute joins every summary
// goroutine it spawns even when a summary call hangs: the deadline cancels the
// blocked call, Execute returns rather than wedging, and the goroutine count
// settles back to baseline. It is not parallel because it reads the process-wide
// goroutine count.
func TestUpdateFeedUsecase_Execute_NoGoroutineLeak(t *testing.T) {
	blockID := "https://habr.com/blocking/"

	feeds := feedWith(art(blockID))
	summarizer := &fakeSummaryClient{blockURL: map[string]bool{blockID: true}}

	uc := NewUpdateFeedUsecase(feeds, summarizer, &fakeRepo{}, quietLogger())

	before := runtime.NumGoroutine()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	go func() {
		_ = uc.Execute(ctx, habr.FeedDaily)
		close(done)
	}()

	select {
	case <-done: // returned: the blocked goroutine was canceled at the deadline and joined
	case <-time.After(2 * time.Second):
		t.Fatal("Execute did not return: a summary goroutine leaked past the deadline")
	}

	// All spawned goroutines should have exited; allow a brief settle.
	deadline := time.Now().Add(2 * time.Second)
	for runtime.NumGoroutine() > before && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if after := runtime.NumGoroutine(); after > before {
		t.Fatalf("goroutine leak: before=%d after=%d", before, after)
	}
}
