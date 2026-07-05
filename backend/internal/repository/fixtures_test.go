package repository

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"

	"habr-observer/internal/config"
	"habr-observer/internal/domain"
)

const (
	// testMongoImage pins the MongoDB image the integration tests run against; it
	// matches the production image in db/Dockerfile and the two must move together.
	//
	// 8.0.21+ refuse to start on Linux kernels 6.19+ (SERVER-121912) even where
	// the incompatibility is fixed, so this pins the newest patch without that
	// guard. Switch to a floating mongo:8.0 once SERVER-125742 relaxes it.
	testMongoImage = "mongo:8.0.20"

	// containerMongoUser and containerMongoPass are the root credentials created
	// in the throwaway test container. They are test-only and never leave the
	// test process. The user is a superuser in the admin database, matching the
	// AuthSource [NewMongoStorage] uses.
	containerMongoUser = "root"
	containerMongoPass = "test-password"

	// testMongoURIEnv lets CI or a developer point the suite at an
	// already-running MongoDB (e.g. a CI service container) instead of starting
	// one via Docker. It is test-only; locally it is typically
	// mongodb://<user>:<pass>@localhost:27017.
	testMongoURIEnv = "OBSERVER_TEST_MONGO_URI"

	// requireMongoEnv forces a missing MongoDB to fail the suite instead of
	// skipping it. Common CI systems set CI automatically, so the integration
	// tests fail loudly on a build server rather than passing green at the
	// hermetic-only coverage floor; set OBSERVER_TEST_REQUIRE_MONGO to demand
	// the same anywhere else.
	requireMongoEnv = "OBSERVER_TEST_REQUIRE_MONGO"

	// testOpTimeout bounds each test's MongoDB operations so an unreachable or
	// misconfigured server fails fast instead of hanging.
	testOpTimeout = 30 * time.Second
)

// The MongoDB the integration tests run against, resolved once by [TestMain].
// testMongoSkip is non-empty when no server could be reached — under -short, or
// because Docker is absent and OBSERVER_TEST_MONGO_URI is unset — in which case
// [connectTestStore] skips the calling test with it as the reason.
var (
	testMongoHost string
	testMongoUser string
	testMongoPass string
	testMongoSkip string
)

// TestMain resolves a MongoDB for the package's integration tests: it prefers an
// externally provided OBSERVER_TEST_MONGO_URI and otherwise starts a throwaway
// container (skipped under -short). The container, if any, lives for the whole
// package run and is shared across tests; each test isolates itself on its own
// database (see [newTestStore]). When no MongoDB can be reached the integration
// tests skip locally but fail the run in CI (see [skipOrFail]).
func TestMain(m *testing.M) {
	os.Exit(runMain(m))
}

func runMain(m *testing.M) int {
	flag.Parse() // parse test flags so testing.Short() is meaningful here

	if testing.Short() {
		testMongoSkip = "skipping mongo integration test in -short mode"
		return m.Run()
	}

	// Prefer an externally provided instance; it carries its own credentials.
	if uri := strings.TrimSpace(os.Getenv(testMongoURIEnv)); uri != "" {
		u, err := url.Parse(uri)
		if err != nil {
			return skipOrFail(m, fmt.Sprintf("invalid %s: %v", testMongoURIEnv, err))
		}
		testMongoHost = u.Host
		testMongoUser = u.User.Username()
		testMongoPass, _ = u.User.Password()
		return m.Run()
	}

	// No external instance: start a throwaway container. This needs Docker; if it
	// is unavailable, skip the integration tests rather than fail the build.
	ctx := context.Background()
	ctr, err := mongodb.Run(ctx, testMongoImage,
		mongodb.WithUsername(containerMongoUser),
		mongodb.WithPassword(containerMongoPass),
	)
	if err != nil {
		log.Printf("repository test: starting mongo container: %v", err)
		return skipOrFail(m, fmt.Sprintf("no MongoDB available: set %s or start Docker (%v)", testMongoURIEnv, err))
	}
	defer func() {
		if err := testcontainers.TerminateContainer(ctr); err != nil {
			log.Printf("repository test: terminating mongo container: %v", err)
		}
	}()

	uri, err := ctr.ConnectionString(ctx)
	if err != nil {
		return skipOrFail(m, fmt.Sprintf("mongo container connection string: %v", err))
	}
	u, err := url.Parse(uri)
	if err != nil {
		return skipOrFail(m, fmt.Sprintf("parsing mongo connection string %q: %v", uri, err))
	}
	testMongoHost = u.Host
	testMongoUser = containerMongoUser
	testMongoPass = containerMongoPass

	return m.Run()
}

// skipOrFail records reason as the package-wide skip and runs the suite — but
// when MongoDB is required (see [mongoRequired]) it aborts the whole run
// instead, so a build server fails loudly rather than passing green at the
// hermetic-only coverage floor. The -short opt-out never reaches here, so it
// always skips, even in CI.
func skipOrFail(m *testing.M, reason string) int {
	if mongoRequired() {
		log.Fatalf("repository: MongoDB required but unavailable: %s", reason)
	}
	testMongoSkip = reason
	return m.Run()
}

// mongoRequired reports whether a missing MongoDB must fail the suite rather
// than skip it: true in CI (the CI env var, set by common CI systems) or when
// OBSERVER_TEST_REQUIRE_MONGO is set.
func mongoRequired() bool {
	return os.Getenv("CI") != "" || os.Getenv(requireMongoEnv) != ""
}

// newTestStore returns a MongoStorage bound to a database unique to the calling
// test, so tests — including parallel ones — never see each other's data. It
// skips the test when no MongoDB is available. The database and connection are
// dropped when the test ends.
func newTestStore(t *testing.T) *MongoStorage {
	t.Helper()

	db := testDBName(t)
	m := connectTestStore(t, db)

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), testOpTimeout)
		defer cancel()
		if err := m.client.Database(db).Drop(ctx); err != nil {
			t.Errorf("dropping test database %q: %v", db, err)
		}
		if err := m.Close(ctx); err != nil {
			t.Errorf("closing store: %v", err)
		}
	})

	return m
}

// connectTestStore connects a MongoStorage to database db, skipping the test
// when no MongoDB is available. Unlike [newTestStore] it registers no teardown,
// so the caller owns the store's lifecycle — used by tests that close the store
// themselves to exercise post-Close behavior.
func connectTestStore(t *testing.T, db string) *MongoStorage {
	t.Helper()

	if testMongoSkip != "" {
		t.Skip(testMongoSkip)
	}

	ctx, cancel := context.WithTimeout(context.Background(), testOpTimeout)
	defer cancel()

	m, err := NewMongoStorage(ctx, config.MongoConfig{
		Host:         testMongoHost,
		User:         testMongoUser,
		Password:     testMongoPass,
		DB:           db,
		ArticlesColl: "articles",
		FeedsColl:    "feeds",
	})
	if err != nil {
		t.Fatalf("NewMongoStorage: %v", err)
	}

	return m
}

// testContext returns a context bounded by testOpTimeout and cancelled when the
// test ends.
func testContext(t *testing.T) context.Context {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), testOpTimeout)
	t.Cleanup(cancel)
	return ctx
}

// testDBName maps the test name to a valid, unique MongoDB database name:
// letters, digits and underscores only, capped at MongoDB's 63-byte limit.
func testDBName(t *testing.T) string {
	t.Helper()

	safe := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			return r
		default:
			return '_'
		}
	}, t.Name())

	if len(safe) > 63 {
		safe = safe[:63]
	}
	return safe
}

// sampleArticles returns two distinct, fully populated articles. Each call
// returns fresh values, so tests never share mutable state.
func sampleArticles() (*domain.Article, *domain.Article) {
	pub := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	return &domain.Article{
			ID:      "https://habr.com/ru/articles/1/",
			Title:   "One",
			PubDate: pub,
			Author:  "alice",
			Summary: &domain.Summary{URL: "https://300.ya.ru/x", Content: []string{"a", "b"}},
		}, &domain.Article{
			ID:      "https://habr.com/ru/articles/2/",
			Title:   "Two",
			PubDate: pub,
			Author:  "bob",
			Summary: &domain.Summary{URL: "https://300.ya.ru/y", Content: []string{"c"}},
		}
}

// assertArticleEqual compares articles field by field. PubDate is compared with
// Time.Equal because a Mongo round-trip strips the monotonic clock and may
// change the time's internal representation, which would defeat DeepEqual.
func assertArticleEqual(t *testing.T, got, want *domain.Article) {
	t.Helper()

	if got == nil {
		t.Fatalf("article %q: got nil", want.ID)
	}
	if got.ID != want.ID || got.Title != want.Title || got.Author != want.Author {
		t.Errorf("article scalar fields mismatch:\n got %+v\nwant %+v", got, want)
	}
	if !got.PubDate.Equal(want.PubDate) {
		t.Errorf("article %q PubDate mismatch: got %v, want %v", want.ID, got.PubDate, want.PubDate)
	}
	if !reflect.DeepEqual(got.Summary, want.Summary) {
		t.Errorf("article %q Summary mismatch:\n got %+v\nwant %+v", want.ID, got.Summary, want.Summary)
	}
}
