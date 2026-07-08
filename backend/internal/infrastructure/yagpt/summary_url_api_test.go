package yagpt

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"strings"
	"testing"
)

const (
	// testAuthToken is a placeholder OAuth credential for non-live tests,
	// where the request never reaches Yandex.
	testAuthToken = "test-oauth-token"

	// testArticleURL is the habr article whose sharing URL we request;
	// it is the source article behind testArticleToken's summary.
	testArticleURL = "https://habr.com/ru/articles/133473/"

	// liveAuthTokenEnv is the OAuth token for the live test. It reuses the
	// application's OBSERVER_AUTH_TOKEN (see internal/config) so a token already
	// present in your .env serves both the app and this test.
	liveAuthTokenEnv = "OBSERVER_AUTH_TOKEN"

	// liveAuthTokenPlaceholder is the .env_example placeholder for
	// OBSERVER_AUTH_TOKEN; a token still set to it isn't real, so treat it as
	// "no token" and skip the live test instead of failing.
	liveAuthTokenPlaceholder = "default"
)

func TestGetSharingURL_InputValidation(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		articleURL string
		authToken  string
		doer       httpDoer
		err        string
	}{
		{
			name:       "empty_article_url",
			articleURL: "",
			authToken:  testAuthToken,
			doer:       &http.Client{},
			err:        "API: empty articleURL",
		},
		{
			name:       "whitespace_article_url",
			articleURL: "   ",
			authToken:  testAuthToken,
			doer:       &http.Client{},
			err:        "API: empty articleURL",
		},
		{
			name:       "empty_auth_token",
			articleURL: testArticleURL,
			authToken:  "",
			doer:       &http.Client{},
			err:        "API: empty authToken",
		},
		{
			name:       "whitespace_auth_token",
			articleURL: testArticleURL,
			authToken:  "  ",
			doer:       &http.Client{},
			err:        "API: empty authToken",
		},
		{
			name:       "nil_doer",
			articleURL: testArticleURL,
			authToken:  testAuthToken,
			doer:       nil,
			err:        "API: nil httpDoer",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := getSharingURL(context.Background(), tc.doer, tc.authToken, tc.articleURL)
			if err == nil {
				t.Fatal("getSharingURL: want error got nil")
			}
			if !reflect.DeepEqual(got, SummaryURL{}) {
				t.Fatalf("getSharingURL: got non-zero result %+v", got)
			}
			if !strings.Contains(err.Error(), tc.err) {
				t.Fatalf("getSharingURL: error %q does not contain expected text %q", err, tc.err)
			}
		})
	}
}

func TestGetSharingURL_Errors(t *testing.T) {
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
					StatusCode: http.StatusUnauthorized,
					Status:     fmt.Sprintf("%d %s", http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized)),
					Body:       io.NopCloser(strings.NewReader("error body")),
					Request:    r,
				}, nil
			}),
			err: fmt.Sprintf("HTTP %d %s: \"error body\"", http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized)),
		},
		{
			name: "read_error",
			rt: RT(func(r *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(newErrReader(io.ErrUnexpectedEOF)),
					Request:    r,
				}, nil
			}),
			err: io.ErrUnexpectedEOF.Error(),
		},
		{
			name: "bad_json",
			rt: RT(func(r *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("bad json")),
					Request:    r,
				}, nil
			}),
			err: "decoding response",
		},
		{
			name: "unsuccessful_status",
			rt: RT(func(r *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"status":"error","sharing_url":""}`)),
					Request:    r,
				}, nil
			}),
			err: "unsuccessful status in JSON response",
		},
		{
			name: "empty_sharing_url",
			rt: RT(func(r *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"status":"success","sharing_url":""}`)),
					Request:    r,
				}, nil
			}),
			err: "empty sharing_url in JSON response",
		},
		{
			name: "invalid_sharing_url",
			rt: RT(func(r *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"status":"success","sharing_url":"http://example.com/x"}`)),
					Request:    r,
				}, nil
			}),
			err: "creating summary URL",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			hc := newHTTPClientWithRT(t, tc.rt)
			got, err := getSharingURL(context.Background(), hc, testAuthToken, testArticleURL)
			if err == nil {
				t.Fatal("getSharingURL: want error got nil")
			}
			if !reflect.DeepEqual(got, SummaryURL{}) {
				t.Fatalf("getSharingURL: got non-zero result %+v", got)
			}
			if !strings.Contains(err.Error(), "API:") {
				t.Fatalf("getSharingURL: error %q missing API: prefix", err)
			}
			if !strings.Contains(err.Error(), tc.err) {
				t.Fatalf("getSharingURL: error %q does not contain expected text %q", err, tc.err)
			}
		})
	}
}

func TestGetSharingURL_NotFound(t *testing.T) {
	t.Parallel()

	rt := RT(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Status:     fmt.Sprintf("%d %s", http.StatusNotFound, http.StatusText(http.StatusNotFound)),
			Body:       io.NopCloser(strings.NewReader("")),
			Request:    r,
		}, nil
	})

	hc := newHTTPClientWithRT(t, rt)
	got, err := getSharingURL(context.Background(), hc, testAuthToken, testArticleURL)
	if !errors.Is(err, ErrSummaryUnavailable) {
		t.Fatalf("getSharingURL: want ErrSummaryUnavailable, got %v", err)
	}
	if !reflect.DeepEqual(got, SummaryURL{}) {
		t.Fatalf("getSharingURL: got non-zero result %+v", got)
	}
}

func TestGetSharingURL_Success(t *testing.T) {
	t.Parallel()

	wantType := "application/json"
	testBody := fmt.Sprintf(`{"status":"success","sharing_url":%q}`, testSummaryRawURL)

	rt := RT(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodPost {
			t.Errorf("Method = %q, want %q", r.Method, http.MethodPost)
		}
		if r.URL.String() != sharingURLEndpoint {
			t.Errorf("URL = %q, want %q", r.URL, sharingURLEndpoint)
		}
		if auth := r.Header.Get("Authorization"); auth != "OAuth "+testAuthToken {
			t.Errorf("Authorization = %q, want %q", auth, "OAuth "+testAuthToken)
		}
		if ct := r.Header.Get("Content-Type"); ct != wantType {
			t.Errorf("Content-Type = %q, want %q", ct, wantType)
		}
		if ac := r.Header.Get("Accept"); ac != wantType {
			t.Errorf("Accept = %q, want %q", ac, wantType)
		}

		var payload sharingURLRequestPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("decoding request body: %v", err)
			return nil, err
		}
		if payload.ArticleURL != testArticleURL {
			t.Errorf("ArticleURL = %q, want %q", payload.ArticleURL, testArticleURL)
		}

		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(testBody)),
			Request:    r,
		}, nil
	})

	hc := newHTTPClientWithRT(t, rt)
	got, err := getSharingURL(context.Background(), hc, testAuthToken, testArticleURL)
	if err != nil {
		t.Fatalf("getSharingURL: %v", err)
	}

	want := mustSummaryURL(t, testSummaryRawURL)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("getSharingURL: got %+v, want %+v", got, want)
	}
}

func TestGetSharingURL_HTTPIntegration(t *testing.T) {
	t.Parallel()

	testBody := readTestData(t, sharingURLFile)

	var fixture sharingURLResponse
	if err := json.Unmarshal(testBody, &fixture); err != nil {
		t.Fatalf("unmarshalling %s: %v", sharingURLFile, err)
	}

	wantURL, err := url.Parse(sharingURLEndpoint)
	if err != nil {
		t.Fatalf("parsing URL: %v", err)
	}

	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Method = %q, want %q", r.Method, http.MethodPost)
			return
		}

		if r.Host != wantURL.Host {
			t.Errorf("Host = %q, want %q", r.Host, wantURL.Host)
			return
		}

		if r.URL.EscapedPath() != wantURL.EscapedPath() {
			t.Errorf("Path = %q, want %q", r.URL.EscapedPath(), wantURL.EscapedPath())
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, wErr := w.Write(testBody)
		if wErr != nil {
			t.Errorf("writing response: %v", wErr)
		}
	}))
	t.Cleanup(ts.Close)

	hc := newPinnedHTTPClient(t, ts)
	got, err := getSharingURL(context.Background(), hc, testAuthToken, testArticleURL)
	if err != nil {
		t.Fatalf("getSharingURL: %v", err)
	}

	if got.String() != fixture.SharingURL {
		t.Fatalf("getSharingURL: got %q, want %q", got.String(), fixture.SharingURL)
	}
}

func TestGetSharingURL_Live(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live test, run without -short to include it")
	}

	token := strings.TrimSpace(os.Getenv(liveAuthTokenEnv))
	if token == "" || token == liveAuthTokenPlaceholder {
		t.Skipf("skipping live test, set %s to a real OAuth token to run it", liveAuthTokenEnv)
	}

	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	hc := &http.Client{Timeout: defaultTimeout}
	got, err := getSharingURL(ctx, hc, token, testArticleURL)
	if err != nil {
		t.Fatalf("getSharingURL: %v", err)
	}
	if got.Token() == "" {
		t.Fatal("getSharingURL: empty token in result")
	}
}
