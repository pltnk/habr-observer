package yagpt

import (
	"fmt"
	"net/url"
	"path"
	"strings"
)

// SummaryURL is a validated 300.ya.ru sharing URL. It pairs the parsed URL with
// the opaque token taken from its final path segment, so callers can reuse the
// token without re-parsing. Construct one with [NewSummaryURL]; the zero value
// is not a usable URL.
type SummaryURL struct {
	url   url.URL
	token string
}

// URL returns the parsed sharing URL (a copy), for example to fetch the sharing
// page directly during the HTML fallback.
func (su SummaryURL) URL() url.URL {
	return su.url
}

// Token returns the opaque identifier from the URL's final path segment, used as
// the request key for the summary-content API.
func (su SummaryURL) Token() string {
	return su.token
}

// String returns the sharing URL in its canonical string form, implementing
// [fmt.Stringer].
func (su SummaryURL) String() string {
	return su.url.String()
}

// NewSummaryURL parses rawURL and validates it as a 300.ya.ru sharing URL: it
// must use https, be hosted on the service's domain, and end in a non-empty path
// segment, which becomes the [SummaryURL.Token]. Surrounding whitespace is
// trimmed. It returns an error describing the first check that fails.
func NewSummaryURL(rawURL string) (SummaryURL, error) {
	rawURL = strings.TrimSpace(rawURL)

	u, err := url.Parse(rawURL)
	if err != nil {
		return SummaryURL{}, fmt.Errorf("creating summary URL from %q: %w", rawURL, err)
	}

	if !strings.EqualFold(u.Scheme, "https") {
		return SummaryURL{}, fmt.Errorf("creating summary URL from %q: want Scheme=%q, got %q", rawURL, "https", u.Scheme)
	}

	if !strings.EqualFold(u.Hostname(), baseHostname) {
		return SummaryURL{}, fmt.Errorf("creating summary URL from %q: want Hostname=%q, got %q", rawURL, baseHostname, u.Hostname())
	}

	t := path.Base(u.Path)
	if t == "" || t == "/" || t == "." {
		return SummaryURL{}, fmt.Errorf("creating summary URL from %q: no token", rawURL)
	}

	su := SummaryURL{
		url:   *u,
		token: t,
	}

	return su, nil
}
