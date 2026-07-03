package habr

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
)

func TestNewClient_DefaultHTTPClient(t *testing.T) {
	t.Parallel()

	c := NewClient(nil)

	if c.c == nil {
		t.Fatal("NewClient(nil): nil http client")
	}
	if c.c.Timeout != defaultTimeout {
		t.Fatalf("default Timeout = %v, want %v", c.c.Timeout, defaultTimeout)
	}
}

func TestGetArticles_Errors(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		rt   http.RoundTripper
		err  string
	}{
		{
			name: "transport_error",
			rt: RT(func(r *http.Request) (*http.Response, error) {
				return nil, errors.New("unable to get response")
			}),
			err: "unable to get response",
		},
		{
			name: "context_canceled",
			rt: RT(func(r *http.Request) (*http.Response, error) {
				return nil, context.Canceled
			}),
			err: context.Canceled.Error(),
		},
		{
			name: "context_deadline_exceeded",
			rt: RT(func(r *http.Request) (*http.Response, error) {
				return nil, context.DeadlineExceeded
			}),
			err: context.DeadlineExceeded.Error(),
		},
		{
			name: "HTTP_error",
			rt: RT(func(r *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusInternalServerError,
					Status:     fmt.Sprintf("%d %s", http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError)),
					Body:       io.NopCloser(strings.NewReader("error body")),
					Request:    r,
				}, nil
			}),
			err: fmt.Sprintf("HTTP %d %s: \"error body\"", http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError)),
		},
		{
			name: "read_error",
			rt: RT(func(r *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Status:     http.StatusText(http.StatusOK),
					Body:       io.NopCloser(newErrReader(io.ErrUnexpectedEOF)),
					Request:    r,
				}, nil
			}),
			err: io.ErrUnexpectedEOF.Error(),
		},
		{
			name: "parsing_error",
			rt: RT(func(r *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Status:     http.StatusText(http.StatusOK),
					Body:       io.NopCloser(bytes.NewReader(readTestData(t, testDataBadRSSFilename))),
					Request:    r,
				}, nil
			}),
			err: "parsing XML",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			hc := newHTTPClientWithRT(t, tc.rt)
			c := NewClient(hc)

			got, err := c.GetArticles(context.Background(), FeedAllTime)
			if err == nil {
				t.Fatalf("GetArticles: want error got nil")
			}
			if got != nil {
				t.Fatalf("GetArticles: got non-nil articles %+v", got)
			}
			if !strings.Contains(err.Error(), "getting articles for") || !strings.Contains(err.Error(), tc.err) {
				t.Fatalf("GetArticles: error %q does not contain expected text %q", err, tc.err)
			}
		})
	}
}

func TestGetArticles_Pinned_AllFeeds(t *testing.T) {
	t.Parallel()

	newTestServer := func(t *testing.T, f RSSFeed, testDataFilename string) *httptest.Server {
		t.Helper()

		return httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Fatalf("Method = %q, want %q", r.Method, http.MethodGet)
			}

			wantURL, err := url.Parse(f.URL())
			if err != nil {
				t.Fatalf("parsing URL: %v", err)
			}

			if r.Host != wantURL.Host {
				t.Fatalf("Host = %q, want %q", r.Host, wantURL.Host)
			}

			if r.URL.EscapedPath() != wantURL.EscapedPath() {
				t.Fatalf("Path = %q, want %q", r.URL.EscapedPath(), wantURL.EscapedPath())
			}

			if r.URL.RawQuery != wantURL.RawQuery {
				t.Fatalf("Query = %q, want %q", r.URL.RawQuery, wantURL.RawQuery)
			}

			w.Header().Set("Content-Type", "text/xml; charset=utf-8")
			w.WriteHeader(http.StatusOK)

			_, err = w.Write(readTestData(t, testDataFilename))
			if err != nil {
				t.Fatalf("writing response: %v", err)
			}
		}))
	}

	for _, f := range AllFeeds() {
		t.Run(f.String(), func(t *testing.T) {
			t.Parallel()

			ts := newTestServer(t, f, testDataValidRSSFilename)
			t.Cleanup(ts.Close)

			hc := newPinnedHTTPClient(t, ts)
			c := NewClient(hc)

			got, err := c.GetArticles(context.Background(), f)
			if err != nil {
				t.Fatalf("GetArticles(%s): %v", f, err)
			}

			if len(got) != testDataExpectedItemsNum {
				t.Fatalf("got %d articles, want %d", len(got), testDataExpectedItemsNum)
			}

			want := newExpectedThreeArticles(t)

			for i := range testDataExpectedItemsNum {
				if !reflect.DeepEqual(got[i], want[i]) {
					t.Errorf("got %+v, want %+v", got[i], want[i])
				}
			}
		})
	}

}

func TestGetArticles_Live_AllFeeds(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live test, run without -short to include it")
	}

	t.Parallel()

	client := NewClient(nil)

	for _, f := range AllFeeds() {
		t.Run(f.String(), func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
			defer cancel()

			articles, err := client.GetArticles(ctx, f)
			if err != nil {
				t.Fatalf("GetArticles(%s): %v", f, err)
			}

			if len(articles) == 0 {
				t.Fatalf("GetArticles(%s): got 0 articles", f)
			}

			for i := 0; i < len(articles) && i < testDataExpectedItemsNum; i++ {
				a := articles[i]
				if a.ID == "" || a.Title == "" || a.Author == "" || a.PubDate.IsZero() {
					t.Fatalf("%s article[%d] has empty required fields: %+v", f, i, a)
				}
			}
		})
	}
}
