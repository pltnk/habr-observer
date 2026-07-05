package yagpt

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"habr-observer/internal/domain"
)

func TestNewClient_Errors(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		token string
	}{
		{name: "empty_token", token: ""},
		{name: "whitespace_token", token: "  \t\n "},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			c, err := NewClient(tc.token, nil)
			if err == nil {
				t.Fatal("NewClient: want error got nil")
			}
			if c != nil {
				t.Fatalf("NewClient: got non-nil client %+v", c)
			}
			if !strings.Contains(err.Error(), "yagpt: empty auth token") {
				t.Fatalf("NewClient: error %q does not contain expected text %q", err, "yagpt: empty auth token")
			}
		})
	}
}

func TestNewClient_Success(t *testing.T) {
	t.Parallel()

	t.Run("default_http_client", func(t *testing.T) {
		t.Parallel()

		c, err := NewClient(testAuthToken, nil)
		if err != nil {
			t.Fatalf("NewClient: %v", err)
		}
		if c.client == nil {
			t.Fatal("NewClient: nil http client")
		}
		if c.client.Timeout != defaultTimeout {
			t.Errorf("Timeout = %v, want %v", c.client.Timeout, defaultTimeout)
		}
		if c.token != testAuthToken {
			t.Errorf("token = %q, want %q", c.token, testAuthToken)
		}
		if c.limiter == nil {
			t.Error("NewClient: nil limiter")
		}
	})

	t.Run("trims_token", func(t *testing.T) {
		t.Parallel()

		c, err := NewClient("  \t"+testAuthToken+"\n ", nil)
		if err != nil {
			t.Fatalf("NewClient: %v", err)
		}
		if c.token != testAuthToken {
			t.Errorf("token = %q, want %q", c.token, testAuthToken)
		}
	})
}

func TestClient_GetSummaryURL_Success(t *testing.T) {
	t.Parallel()

	body := readTestData(t, sharingURLFile)
	rt := RT(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodPost {
			t.Errorf("Method = %q, want %q", r.Method, http.MethodPost)
		}
		if r.URL.String() != sharingURLEndpoint {
			t.Errorf("URL = %q, want %q", r.URL, sharingURLEndpoint)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(body)),
			Request:    r,
		}, nil
	})

	c, err := NewClient(testAuthToken, newHTTPClientWithRT(t, rt))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	got, err := c.GetSummaryURL(context.Background(), testArticleURL)
	if err != nil {
		t.Fatalf("GetSummaryURL: %v", err)
	}

	want := mustSummaryURL(t, testSummaryRawURL)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("GetSummaryURL: got %+v, want %+v", got, want)
	}
}

func TestClient_GetSummaryURL_Error(t *testing.T) {
	t.Parallel()

	rt := RT(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusUnauthorized,
			Status:     fmt.Sprintf("%d %s", http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized)),
			Body:       io.NopCloser(strings.NewReader("nope")),
			Request:    r,
		}, nil
	})

	c, err := NewClient(testAuthToken, newHTTPClientWithRT(t, rt))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	got, err := c.GetSummaryURL(context.Background(), testArticleURL)
	if err == nil {
		t.Fatal("GetSummaryURL: want error got nil")
	}
	if !reflect.DeepEqual(got, SummaryURL{}) {
		t.Fatalf("GetSummaryURL: got non-zero result %+v", got)
	}

	wantSubstr := fmt.Sprintf("yagpt: getting summary URL for %q", testArticleURL)
	if !strings.Contains(err.Error(), wantSubstr) {
		t.Fatalf("GetSummaryURL: error %q does not contain %q", err, wantSubstr)
	}
	if !strings.Contains(err.Error(), "API:") {
		t.Fatalf("GetSummaryURL: error %q does not wrap the underlying API error", err)
	}
}

// TestClient_GetSummaryURL_Unavailable pins the public detection contract: a 404
// from the sharing-url endpoint must remain matchable as ErrSummaryUnavailable
// through the wrapping GetSummaryURL adds, so callers can tell "no summary exists"
// apart from a transient failure. Without this, a regression from %w to %v in
// client.go would break errors.Is for every caller, with the rest of the suite
// staying green.
func TestClient_GetSummaryURL_Unavailable(t *testing.T) {
	t.Parallel()

	rt := RT(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Status:     fmt.Sprintf("%d %s", http.StatusNotFound, http.StatusText(http.StatusNotFound)),
			Body:       io.NopCloser(strings.NewReader("")),
			Request:    r,
		}, nil
	})

	c, err := NewClient(testAuthToken, newHTTPClientWithRT(t, rt))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	got, err := c.GetSummaryURL(context.Background(), testArticleURL)
	if !errors.Is(err, ErrSummaryUnavailable) {
		t.Fatalf("GetSummaryURL: want ErrSummaryUnavailable through wrapping, got %v", err)
	}
	if !reflect.DeepEqual(got, SummaryURL{}) {
		t.Fatalf("GetSummaryURL: got non-zero result %+v", got)
	}
}

func TestClient_GetSummaryURL_RateLimited(t *testing.T) {
	t.Parallel()

	rt := RT(func(r *http.Request) (*http.Response, error) {
		t.Error("unexpected HTTP request: a canceled context should short-circuit in the rate limiter")
		return nil, errors.New("unreachable")
	})

	c, err := NewClient(testAuthToken, newHTTPClientWithRT(t, rt))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	got, err := c.GetSummaryURL(ctx, testArticleURL)
	if err == nil {
		t.Fatal("GetSummaryURL: want error got nil")
	}
	if !reflect.DeepEqual(got, SummaryURL{}) {
		t.Fatalf("GetSummaryURL: got non-zero result %+v", got)
	}
	if !strings.Contains(err.Error(), "yagpt: getting summary URL rate limit") {
		t.Fatalf("GetSummaryURL: error %q does not contain the rate-limit context", err)
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("GetSummaryURL: error %q is not context.Canceled", err)
	}
}

func TestClient_GetSummaryContent_APISuccess(t *testing.T) {
	t.Parallel()

	apiBody := readTestData(t, summaryContentFile)
	rt := RT(func(r *http.Request) (*http.Response, error) {
		switch r.Method {
		case http.MethodPost:
			if r.URL.String() != sharingEndpoint {
				t.Errorf("API URL = %q, want %q", r.URL, sharingEndpoint)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(apiBody)),
				Request:    r,
			}, nil
		default:
			t.Errorf("unexpected %s request to %q: HTML fallback must not run when the API succeeds", r.Method, r.URL)
			return nil, errors.New("unexpected fallback")
		}
	})

	c, err := NewClient(testAuthToken, newHTTPClientWithRT(t, rt))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	su := mustSummaryURL(t, testSummaryRawURL)
	got, err := c.GetSummaryContent(context.Background(), su)
	if err != nil {
		t.Fatalf("GetSummaryContent: %v", err)
	}

	want := expectedSummaryLines(t)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("GetSummaryContent: got %v, want %v", got, want)
	}
}

func TestClient_GetSummaryContent_HTMLFallback(t *testing.T) {
	t.Parallel()

	htmlBody := readTestData(t, summaryPageFile)
	rt := RT(func(r *http.Request) (*http.Response, error) {
		switch r.Method {
		case http.MethodPost:
			// API path fails, forcing the HTML fallback.
			return nil, errors.New("api unavailable")
		case http.MethodGet:
			if r.URL.String() != testSummaryRawURL {
				t.Errorf("HTML URL = %q, want %q", r.URL, testSummaryRawURL)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(htmlBody)),
				Request:    r,
			}, nil
		default:
			t.Errorf("unexpected %s request to %q", r.Method, r.URL)
			return nil, errors.New("unexpected method")
		}
	})

	c, err := NewClient(testAuthToken, newHTTPClientWithRT(t, rt))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	su := mustSummaryURL(t, testSummaryRawURL)
	got, err := c.GetSummaryContent(context.Background(), su)
	if err != nil {
		t.Fatalf("GetSummaryContent: %v", err)
	}

	// The page's og:description carries the same theses as the JSON API,
	// so both paths yield expectedSummaryLines.
	want := expectedSummaryLines(t)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("GetSummaryContent: got %v, want %v", got, want)
	}
}

func TestClient_GetSummaryContent_ContextCanceledSkipsFallback(t *testing.T) {
	t.Parallel()

	// A custom RoundTripper does not honor context cancellation the way the
	// real transport does, so the API request still reaches it; the point of
	// this test is that the HTML fallback (a GET) is skipped once ctx is done.
	apiErr := errors.New("api boom")
	rt := RT(func(r *http.Request) (*http.Response, error) {
		if r.Method == http.MethodGet {
			t.Errorf("HTML fallback must not run after the context is canceled (got GET %q)", r.URL)
		}
		return nil, apiErr
	})

	c, err := NewClient(testAuthToken, newHTTPClientWithRT(t, rt))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	su := mustSummaryURL(t, testSummaryRawURL)
	got, err := c.GetSummaryContent(ctx, su)
	if err == nil {
		t.Fatal("GetSummaryContent: want error got nil")
	}
	if got != nil {
		t.Fatalf("GetSummaryContent: got non-nil result %+v", got)
	}
	if !strings.Contains(err.Error(), "yagpt: getting summary content from") {
		t.Fatalf("GetSummaryContent: error %q missing wrapping context", err)
	}
	// Only the API error is returned; the HTML path (errors prefixed "HTML:")
	// is skipped, so the join branch is not taken.
	if !errors.Is(err, apiErr) {
		t.Fatalf("GetSummaryContent: error %q does not wrap the API error", err)
	}
	if strings.Contains(err.Error(), "HTML:") {
		t.Fatalf("GetSummaryContent: fallback should be skipped on cancellation, but error mentions the HTML path: %q", err)
	}
}

func TestClient_GetSummaryContent_BothFail(t *testing.T) {
	t.Parallel()

	apiErr := errors.New("api boom")
	htmlErr := errors.New("html boom")

	rt := RT(func(r *http.Request) (*http.Response, error) {
		switch r.Method {
		case http.MethodPost:
			return nil, apiErr
		case http.MethodGet:
			return nil, htmlErr
		default:
			t.Errorf("unexpected %s request to %q", r.Method, r.URL)
			return nil, errors.New("unexpected method")
		}
	})

	c, err := NewClient(testAuthToken, newHTTPClientWithRT(t, rt))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	su := mustSummaryURL(t, testSummaryRawURL)
	got, err := c.GetSummaryContent(context.Background(), su)
	if err == nil {
		t.Fatal("GetSummaryContent: want error got nil")
	}
	if got != nil {
		t.Fatalf("GetSummaryContent: got non-nil result %+v", got)
	}
	if !strings.Contains(err.Error(), "yagpt: getting summary content from") {
		t.Fatalf("GetSummaryContent: error %q missing wrapping context", err)
	}

	// Both underlying errors are joined and remain inspectable via errors.Is.
	if !errors.Is(err, apiErr) {
		t.Errorf("GetSummaryContent: joined error %q does not wrap the API error", err)
	}
	if !errors.Is(err, htmlErr) {
		t.Errorf("GetSummaryContent: joined error %q does not wrap the HTML error", err)
	}
}

func TestClient_GetSummary_Success(t *testing.T) {
	t.Parallel()

	urlBody := readTestData(t, sharingURLFile)
	contentBody := readTestData(t, summaryContentFile)
	rt := RT(func(r *http.Request) (*http.Response, error) {
		switch r.URL.String() {
		case sharingURLEndpoint:
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader(urlBody)), Request: r}, nil
		case sharingEndpoint:
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader(contentBody)), Request: r}, nil
		default:
			t.Errorf("unexpected request to %q", r.URL)
			return nil, errors.New("unexpected URL")
		}
	})

	c, err := NewClient(testAuthToken, newHTTPClientWithRT(t, rt))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	got, err := c.GetSummary(context.Background(), testArticleURL)
	if err != nil {
		t.Fatalf("GetSummary: %v", err)
	}

	want := &domain.Summary{URL: mustSummaryURL(t, testSummaryRawURL).String(), Content: expectedSummaryLines(t)}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("GetSummary: got %+v, want %+v", got, want)
	}
}

// TestClient_GetSummary_Unavailable pins that a 404 from the sharing-url step
// propagates as ErrSummaryUnavailable and the content fetch is skipped, so the
// caller can apply its own "no summary" policy.
func TestClient_GetSummary_Unavailable(t *testing.T) {
	t.Parallel()

	rt := RT(func(r *http.Request) (*http.Response, error) {
		if r.URL.String() != sharingURLEndpoint {
			t.Errorf("content fetch must not run when the sharing URL is unavailable (got %q)", r.URL)
		}
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Status:     fmt.Sprintf("%d %s", http.StatusNotFound, http.StatusText(http.StatusNotFound)),
			Body:       io.NopCloser(strings.NewReader("")),
			Request:    r,
		}, nil
	})

	c, err := NewClient(testAuthToken, newHTTPClientWithRT(t, rt))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	got, err := c.GetSummary(context.Background(), testArticleURL)
	if !errors.Is(err, ErrSummaryUnavailable) {
		t.Fatalf("GetSummary: want ErrSummaryUnavailable, got %v", err)
	}
	if got != nil {
		t.Fatalf("GetSummary: got non-nil result %+v", got)
	}
}
