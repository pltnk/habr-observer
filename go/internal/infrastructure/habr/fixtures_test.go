package habr

import (
	"context"
	"crypto/tls"
	"habr-observer/internal/entities"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const (
	testDataDir                = "testdata"
	testDataValidRSSFilename   = "habr_valid_rss.xml"
	testDataBadRSSFilename     = "habr_bad_rss.xml"
	testDataEmptyRSSFilename   = "habr_empty_rss.xml"
	testDataBadDateRSSFilename = "habr_bad_date_rss.xml"
	testDataExpectedItemsNum   = 3
	testTimeoutSeconds         = 2
)

func readTestData(t *testing.T, filename string) []byte {
	t.Helper()

	path := filepath.Join(testDataDir, filename)
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading test data: %v", err)
	}
	return b
}

func mustParseRFC1123(t *testing.T, s string) time.Time {
	t.Helper()

	tt, err := parseDate(s)
	if err != nil {
		t.Fatalf("bad RFC1123(_Z) date in test: %q: %v", s, err)
	}
	return tt
}

func newExpectedThreeArticles(t *testing.T) [testDataExpectedItemsNum]*entities.Article {
	t.Helper()

	return [testDataExpectedItemsNum]*entities.Article{
		{
			ID:      "https://habr.com/ru/articles/133473/",
			Title:   "[Перевод] Делаем приватный монитор из старого LCD монитора",
			Author:  "kfedorov",
			PubDate: mustParseRFC1123(t, "Sun, 27 Nov 2011 19:21:13 GMT"),
		},
		{
			ID:      "https://habr.com/ru/articles/536750/",
			Title:   "Самый беззащитный — уже не Сапсан. Всё оказалось куда хуже…",
			Author:  "LMonoceros",
			PubDate: mustParseRFC1123(t, "Wed, 13 Jan 2021 05:51:41 GMT"),
		},
		{
			ID:      "https://habr.com/ru/articles/70330/",
			Title:   "Были получены исходники 3300 глобальных интернет-проектов",
			Author:  "mobilz",
			PubDate: mustParseRFC1123(t, "Wed, 23 Sep 2009 09:17:27 GMT"),
		},
	}
}

type RT func(*http.Request) (*http.Response, error)

func (rt RT) RoundTrip(r *http.Request) (*http.Response, error) {
	return rt(r)
}

func newHTTPClientWithRT(t *testing.T, rt http.RoundTripper) *http.Client {
	t.Helper()

	return &http.Client{
		Transport: rt,
		Timeout:   testTimeoutSeconds * time.Second,
	}
}

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

type errReader struct {
	err error
}

func (e errReader) Read(p []byte) (int, error) {
	return 0, e.err
}

func newErrReader(err error) *errReader {
	return &errReader{
		err: err,
	}
}
