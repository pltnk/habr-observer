package yagpt

import (
	"fmt"
	"net/url"
	"path"
	"strings"
)

type SummaryURL struct {
	url   url.URL
	token string
}

func (su SummaryURL) URL() url.URL {
	return su.url
}

func (su SummaryURL) Token() string {
	return su.token
}

func (su SummaryURL) String() string {
	return su.url.String()
}

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
