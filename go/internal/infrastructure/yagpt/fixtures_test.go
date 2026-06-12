package yagpt

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const (
	// testDataDir is the directory holding fixture files loaded by readTestData.
	testDataDir = "testdata"

	// testTimeoutSeconds caps all HTTP operations in tests to keep failures fast.
	testTimeoutSeconds = 2

	// testArticleToken is a valid public article sharing token
	// (the path segment from a 300.ya.ru/<token> URL), not an OAuth credential.
	testArticleToken = "BoudKEx0"

	// summaryContentFile is the canonical summary for testArticleToken, used as
	// the single source of truth for the expected lines in the integration tests.
	summaryContentFile = "summary_content.json"

	// summaryPageFile is the 300.ya.ru summary page for testArticleToken,
	// served by the HTML integration test.
	summaryPageFile = "summary_page.html"

	// sharingURLFile is a successful /api/sharing-url response,
	// served by the sharing-URL integration test.
	sharingURLFile = "sharing_url.json"
)

// readTestData returns the contents of testdata/filename or fails the test.
func readTestData(t *testing.T, filename string) []byte {
	t.Helper()

	path := filepath.Join(testDataDir, filename)
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading test data: %v", err)
	}
	return b
}

// expectedSummaryLines returns the thesis contents from summaryFixtureFile.
// Both the API and HTML integration tests assert against these, since both the
// JSON sharing response and the page's og:description describe the same article.
func expectedSummaryLines(t *testing.T) []string {
	t.Helper()

	var resp sharingResponse
	if err := json.Unmarshal(readTestData(t, summaryContentFile), &resp); err != nil {
		t.Fatalf("unmarshalling %s: %v", summaryContentFile, err)
	}

	lines := make([]string, len(resp.Thesis))
	for i, el := range resp.Thesis {
		lines[i] = el.Content
	}
	return lines
}

// RT adapts a plain function to the [http.RoundTripper] interface,
// so tests can inject fake HTTP responses without defining a new type.
type RT func(*http.Request) (*http.Response, error)

// RoundTrip implements [http.RoundTripper] by invoking rt.
func (rt RT) RoundTrip(r *http.Request) (*http.Response, error) {
	return rt(r)
}

// newHTTPClientWithRT returns an [http.Client] whose transport is rt
// and whose timeout matches testTimeoutSeconds.
func newHTTPClientWithRT(t *testing.T, rt http.RoundTripper) *http.Client {
	t.Helper()

	return &http.Client{
		Transport: rt,
		Timeout:   testTimeoutSeconds * time.Second,
	}
}

// newPinnedHTTPClient returns an [http.Client] that always dials ts,
// regardless of the request URL's host. This lets production code
// use real endpoint URLs in tests without rewriting them.
//
// TLS verification is disabled because [httptest.NewTLSServer] uses
// a self-signed certificate; this is safe in tests only.
func newPinnedHTTPClient(t *testing.T, ts *httptest.Server) *http.Client {
	t.Helper()

	addr := ts.Listener.Addr().String()

	dialer := &net.Dialer{
		Timeout:   5 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	tr := &http.Transport{
		Proxy: nil,
		DialContext: func(ctx context.Context, network string, _ string) (net.Conn, error) {
			// Ignore the requested address; always dial the test server.
			return dialer.DialContext(ctx, network, addr)
		},
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	return &http.Client{
		Transport: tr,
		Timeout:   testTimeoutSeconds * time.Second,
	}
}

// errReader is an [io.Reader] that always fails with err,
// used to simulate mid-read body failures.
type errReader struct {
	err error
}

// Read always returns 0 and the configured error.
func (e errReader) Read(p []byte) (int, error) {
	return 0, e.err
}

// newErrReader returns an [errReader] that fails every Read with err.
func newErrReader(err error) *errReader {
	return &errReader{
		err: err,
	}
}
