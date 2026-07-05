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
	"reflect"
	"strings"
	"testing"
)

func TestGetSummaryContentAPI_InputValidation(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		token string
		doer  httpDoer
		err   string
	}{
		{
			name:  "empty_token",
			token: "",
			doer:  &http.Client{},
			err:   "API: empty token",
		},
		{
			name:  "nil_doer",
			token: testArticleToken,
			doer:  nil,
			err:   "API: nil httpDoer",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := getSummaryContentAPI(context.Background(), tc.doer, tc.token)

			if err == nil {
				t.Fatal("getSummaryContentAPI: want error got nil")
			}

			if got != nil {
				t.Fatalf("getSummaryContentAPI: got non-nil result %+v", got)
			}

			if !strings.Contains(err.Error(), tc.err) {
				t.Fatalf("getSummaryContentAPI: error %q does not contain expected text %q", err, tc.err)
			}
		})
	}
}

func TestGetSummaryContentAPI_Errors(t *testing.T) {
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
			name: "empty_thesis",
			rt: RT(func(r *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"thesis": []}`)),
					Request:    r,
				}, nil
			}),
			err: "empty thesis",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			hc := newHTTPClientWithRT(t, tc.rt)
			got, err := getSummaryContentAPI(context.Background(), hc, testArticleToken)
			if err == nil {
				t.Fatal("getSummaryContentAPI: want error got nil")
			}

			if got != nil {
				t.Fatalf("getSummaryContentAPI: got non-nil result %+v", got)
			}

			if !strings.Contains(err.Error(), "API:") {
				t.Fatalf("getSummaryContentAPI: error %q missing API: prefix", err)
			}

			if !strings.Contains(err.Error(), tc.err) {
				t.Fatalf("getSummaryContentAPI: error %q does not contain expected text %q", err, tc.err)
			}
		})
	}
}

func TestGetSummaryContentAPI_Success(t *testing.T) {
	t.Parallel()

	wantResult := []string{
		"First thesis line.",
		"Second thesis line.",
	}
	wantType := "application/json"

	testBody := `{"thesis":[{"content":"First thesis line."},{"content":"Second thesis line."}]}`

	rt := RT(func(r *http.Request) (*http.Response, error) {

		if r.Method != http.MethodPost {
			t.Errorf("Method = %q, want %q", r.Method, http.MethodPost)
		}
		if r.URL.String() != sharingEndpoint {
			t.Errorf("URL = %q, want %q", r.URL, sharingEndpoint)
		}
		if ct := r.Header.Get("Content-Type"); ct != wantType {
			t.Errorf("Content-Type = %q, want %q", ct, wantType)
		}
		if ac := r.Header.Get("Accept"); ac != wantType {
			t.Errorf("Accept = %q, want %q", ac, wantType)
		}

		var payload sharingRequestPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("decoding request body: %v", err)
			return nil, err
		}
		if payload.Token != testArticleToken {
			t.Errorf("Token = %q, want %q", payload.Token, testArticleToken)
		}

		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(testBody)),
			Request:    r,
		}, nil
	})

	hc := newHTTPClientWithRT(t, rt)
	got, err := getSummaryContentAPI(context.Background(), hc, testArticleToken)
	if err != nil {
		t.Fatalf("getSummaryContentAPI: %v", err)
	}
	if !reflect.DeepEqual(got, wantResult) {
		t.Fatalf("getSummaryContentAPI: got %v, want %v", got, wantResult)
	}
}

func TestGetSummaryContentAPI_HTTPIntegration(t *testing.T) {
	t.Parallel()

	testBody := readTestData(t, summaryContentFile)

	wantResult := expectedSummaryLines(t)

	wantURL, err := url.Parse(sharingEndpoint)
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
	got, err := getSummaryContentAPI(context.Background(), hc, testArticleToken)
	if err != nil {
		t.Fatalf("getSummaryContentAPI: %v", err)
	}

	if !reflect.DeepEqual(got, wantResult) {
		t.Fatalf("getSummaryContentAPI: got %v, want %v", got, wantResult)
	}
}

func TestGetSummaryContentAPI_Live(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live test, run without -short to include it")
	}

	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	hc := &http.Client{Timeout: defaultTimeout}
	got, err := getSummaryContentAPI(ctx, hc, testArticleToken)
	if err != nil {
		t.Fatalf("getSummaryContentAPI: %v", err)
	}
	if len(got) == 0 {
		t.Fatal("getSummaryContentAPI: empty result")
	}
	for i, line := range got {
		if line == "" {
			t.Errorf("getSummaryContentAPI: result[%d] is empty string", i)
		}
	}
}
