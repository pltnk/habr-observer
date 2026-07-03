package delivery

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"habr-observer/internal/domain"
)

// --- fakes -------------------------------------------------------------------

// Compile-time guard that the fake still satisfies the port it stands in for.
var _ FeedsGetter = fakeFeedsGetter{}

// fakeFeedsGetter is a controllable FeedsGetter returning canned feeds or an error.
type fakeFeedsGetter struct {
	feeds []*domain.Feed
	err   error
}

func (f fakeFeedsGetter) Execute(context.Context) ([]*domain.Feed, error) {
	return f.feeds, f.err
}

// --- helpers -----------------------------------------------------------------

func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// --- tests -------------------------------------------------------------------

func TestGetFeedsHandler_Success(t *testing.T) {
	t.Parallel()

	feeds := []*domain.Feed{
		{
			ID:   "https://habr.com/ru/rss/articles/top/daily/?fl=ru",
			Name: "Сутки",
			Articles: []*domain.Article{
				{
					ID:      "https://habr.com/ru/articles/1/",
					Title:   "One",
					Summary: &domain.Summary{URL: "https://300.ya.ru/x", Content: []string{"a", "b"}},
				},
			},
		},
		{ID: "https://habr.com/ru/rss/articles/top/weekly/?fl=ru", Name: "Неделя"},
	}

	h := NewGetFeedsHandler(fakeFeedsGetter{feeds: feeds}, quietLogger())

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/feeds", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q, want application/json; charset=utf-8", ct)
	}

	var got []*domain.Feed
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	// The full feed set — order, articles, summaries — survives serialization.
	if !reflect.DeepEqual(got, feeds) {
		t.Fatalf("round-tripped feeds = %+v, want %+v", got, feeds)
	}
}

func TestGetFeedsHandler_NilEncodesEmptyArray(t *testing.T) {
	t.Parallel()

	h := NewGetFeedsHandler(fakeFeedsGetter{feeds: nil}, quietLogger())

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/feeds", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if body := strings.TrimSpace(rec.Body.String()); body != "[]" {
		t.Fatalf("body = %q, want %q (nil must encode as an empty array, not null)", body, "[]")
	}
}

// TestGetFeedsHandler_Error pins that every use case failure — a real one or a
// canceled request (whose logging is suppressed) — yields the same 500 with a
// generic body that never leaks the internal error.
func TestGetFeedsHandler_Error(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		err  error
	}{
		{name: "repository_failure", err: errors.New("boom")},
		{name: "canceled_request", err: context.Canceled},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := NewGetFeedsHandler(fakeFeedsGetter{err: tc.err}, quietLogger())

			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/feeds", nil))

			if rec.Code != http.StatusInternalServerError {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
			}
			if body := rec.Body.String(); strings.Contains(body, tc.err.Error()) {
				t.Fatalf("response body %q leaks the internal error", body)
			}
		})
	}
}
