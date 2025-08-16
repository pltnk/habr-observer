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
	"sync"
	"testing"
	"time"
)

func TestClient_http_DefaultHTTPClient(t *testing.T) {
	t.Parallel()

	var c Client
	hc := c.http()

	if hc == nil {
		t.Fatal("http() returned nil")
	}

	if c.c == nil {
		t.Fatal("http() did not store default client into c.c")
	}

	if hc != c.c {
		t.Fatal("http() did not return the same client as stored in c.c")
	}

	if hc.Timeout != defaultTimeout {
		t.Fatalf("default Timeout = %v, want %v", hc.Timeout, defaultTimeout)
	}

	hc2 := c.http()
	if hc2 != hc {
		t.Fatal("http() returned a different client on second call")
	}
}

func TestClient_http_PreservesProvidedClient(t *testing.T) {
	t.Parallel()

	to := 123 * time.Millisecond
	hc := &http.Client{Timeout: to}
	c := NewClient(hc)

	got := c.http()
	if got != hc {
		t.Fatal("http() did not return the provided http.Client")
	}
	if got.Timeout != to {
		t.Fatalf("timeout mutated: got %v, want %v", got.Timeout, to)
	}
}

func TestClient_http_ConcurrentSingleInit(t *testing.T) {
	t.Parallel()

	var c Client

	const N = 64
	var wg sync.WaitGroup
	wg.Add(N)

	// Barrier to start all goroutines simultaneously.
	start := make(chan struct{})
	results := make(chan *http.Client, N)

	for range N {
		go func() {
			defer wg.Done()
			<-start
			results <- c.http()
		}()
	}

	close(start)
	wg.Wait()
	close(results)

	// All returned pointers must be identical, and c.c must be that same pointer.
	var first *http.Client
	for hc := range results {
		if first == nil {
			first = hc
			continue
		}
		if hc != first {
			t.Fatal("http() returned different *http.Client instances across concurrent calls")
		}
	}
	if first == nil {
		t.Fatal("no client returned")
	}
	if c.c != first {
		t.Fatal("c.c not set to the initialized *http.Client")
	}
	if first.Timeout != defaultTimeout {
		t.Fatalf("initialized Timeout = %v, want %v", first.Timeout, defaultTimeout)
	}
}

func TestClient_http_PanicsOnNilReceiver(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for nil *Client")
		}
	}()

	var c *Client
	_ = c.http()
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
					Status:     http.StatusText(http.StatusInternalServerError),
					Body:       io.NopCloser(strings.NewReader("error")),
					Request:    r,
				}, nil
			}),
			err: fmt.Sprintf("HTTP %d %s", http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError)),
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
				t.Fatalf("GetArticles: error does not contain expected text: %q", err)
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
