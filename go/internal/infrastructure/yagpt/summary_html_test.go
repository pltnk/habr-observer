package yagpt

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

// testSummaryRawURL is a public 300.ya.ru summary page used as the request
// target in HTML tests (and fetched for real by the live test). It is the
// same article as testArticleToken / summaryFixtureFile.
const testSummaryRawURL = baseURL + "/" + testArticleToken

// mustSummaryURL builds a SummaryURL from raw or fails the test.
func mustSummaryURL(t *testing.T, raw string) SummaryURL {
	t.Helper()

	su, err := NewSummaryURL(raw)
	if err != nil {
		t.Fatalf("NewSummaryURL(%q): %v", raw, err)
	}
	return su
}

// parseHTML parses s into an *html.Node or fails the test.
func parseHTML(t *testing.T, s string) *html.Node {
	t.Helper()

	n, err := html.Parse(strings.NewReader(s))
	if err != nil {
		t.Fatalf("html.Parse: %v", err)
	}
	return n
}

func TestExtractSummaryContent(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		root := parseHTML(t, `<!doctype html><html><head>`+
			`<meta property="og:description" content='&#8226; First line.
      &#8226; Second line.
      &#8226; Third line.' />`+
			`</head><body></body></html>`)

		got, err := extractSummaryContent(root)
		if err != nil {
			t.Fatalf("extractSummaryContent: %v", err)
		}

		want := []string{"First line.", "Second line.", "Third line."}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("extractSummaryContent: got %v, want %v", got, want)
		}
	})

	cases := []struct {
		name string
		root *html.Node
		err  string
	}{
		{
			name: "nil_root",
			root: nil,
			err:  "nil root node",
		},
		{
			name: "no_meta",
			root: parseHTML(t, `<html><head></head><body></body></html>`),
			err:  "og:description meta tag not found",
		},
		{
			name: "empty_content",
			root: parseHTML(t, `<html><head><meta property="og:description" content="" /></head></html>`),
			err:  "og:description is empty",
		},
		{
			name: "cleaned_empty",
			root: parseHTML(t, `<html><head><meta property="og:description" content='&#8226;
      &#8226;   ' /></head></html>`),
			err: "cleaned og:description is empty",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := extractSummaryContent(tc.root)
			if err == nil {
				t.Fatal("extractSummaryContent: want error got nil")
			}
			if got != nil {
				t.Fatalf("extractSummaryContent: got non-nil result %+v", got)
			}
			if !strings.Contains(err.Error(), tc.err) {
				t.Fatalf("extractSummaryContent: error %q does not contain expected text %q", err, tc.err)
			}
		})
	}
}

func TestGetSummaryContentHTML_InputValidation(t *testing.T) {
	t.Parallel()

	u := mustSummaryURL(t, testSummaryRawURL).URL()

	got, err := getSummaryContentHTML(context.Background(), nil, u)
	if err == nil {
		t.Fatal("getSummaryContentHTML: want error got nil")
	}
	if got != nil {
		t.Fatalf("getSummaryContentHTML: got non-nil result %+v", got)
	}
	if !strings.Contains(err.Error(), "HTML: nil httpDoer") {
		t.Fatalf("getSummaryContentHTML: error %q does not contain expected text %q", err, "HTML: nil httpDoer")
	}
}

func TestGetSummaryContentHTML_Errors(t *testing.T) {
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
			name: "no_og_description",
			rt: RT(func(r *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`<html><head></head><body></body></html>`)),
					Request:    r,
				}, nil
			}),
			err: "og:description meta tag not found",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			u := mustSummaryURL(t, testSummaryRawURL).URL()
			hc := newHTTPClientWithRT(t, tc.rt)
			got, err := getSummaryContentHTML(context.Background(), hc, u)
			if err == nil {
				t.Fatal("getSummaryContentHTML: want error got nil")
			}
			if got != nil {
				t.Fatalf("getSummaryContentHTML: got non-nil result %+v", got)
			}
			if !strings.Contains(err.Error(), "HTML:") {
				t.Fatalf("getSummaryContentHTML: error %q missing HTML: prefix", err)
			}
			if !strings.Contains(err.Error(), tc.err) {
				t.Fatalf("getSummaryContentHTML: error %q does not contain expected text %q", err, tc.err)
			}
		})
	}
}

func TestGetSummaryContentHTML_Success(t *testing.T) {
	t.Parallel()

	u := mustSummaryURL(t, testSummaryRawURL).URL()

	wantResult := []string{"First line.", "Second line."}

	testBody := `<!doctype html><html><head>` +
		`<meta property="og:description" content='&#8226; First line.
      &#8226; Second line.' />` +
		`</head><body></body></html>`

	rt := RT(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodGet {
			t.Errorf("Method = %q, want %q", r.Method, http.MethodGet)
		}
		if r.URL.String() != u.String() {
			t.Errorf("URL = %q, want %q", r.URL, u.String())
		}

		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(testBody)),
			Request:    r,
		}, nil
	})

	hc := newHTTPClientWithRT(t, rt)
	got, err := getSummaryContentHTML(context.Background(), hc, u)
	if err != nil {
		t.Fatalf("getSummaryContentHTML: %v", err)
	}
	if !reflect.DeepEqual(got, wantResult) {
		t.Fatalf("getSummaryContentHTML: got %v, want %v", got, wantResult)
	}
}

func TestGetSummaryContentHTML_HTTPIntegration(t *testing.T) {
	t.Parallel()

	testBody := readTestData(t, summaryPageFile)

	wantResult := expectedSummaryLines(t)

	wantURL := mustSummaryURL(t, testSummaryRawURL).URL()

	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Method = %q, want %q", r.Method, http.MethodGet)
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

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, wErr := w.Write(testBody)
		if wErr != nil {
			t.Errorf("writing response: %v", wErr)
		}
	}))
	t.Cleanup(ts.Close)

	hc := newPinnedHTTPClient(t, ts)
	got, err := getSummaryContentHTML(context.Background(), hc, wantURL)
	if err != nil {
		t.Fatalf("getSummaryContentHTML: %v", err)
	}

	if !reflect.DeepEqual(got, wantResult) {
		t.Fatalf("getSummaryContentHTML: got %v, want %v", got, wantResult)
	}
}

func TestGetSummaryContentHTML_Live(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live test, run without -short to include it")
	}

	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	u := mustSummaryURL(t, testSummaryRawURL).URL()

	hc := &http.Client{Timeout: defaultTimeout}
	got, err := getSummaryContentHTML(ctx, hc, u)
	if err != nil {
		t.Fatalf("getSummaryContentHTML: %v", err)
	}
	if len(got) == 0 {
		t.Fatal("getSummaryContentHTML: empty result")
	}
	for i, line := range got {
		if line == "" {
			t.Errorf("getSummaryContentHTML: result[%d] is empty string", i)
		}
	}
}
