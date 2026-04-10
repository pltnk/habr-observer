// Package yagpt provides a client for YandexGPT's article summarization
// service (300.ya.ru), which generates short thesis-style summaries of
// online articles.
//
// Typical usage is a two-step flow: first obtain a SummaryURL for an
// article via [Client.GetSummaryURL], then fetch the summary content
// via [Client.GetSummaryContent]. The latter prefers the JSON API and
// falls back to scraping the og:description meta tag from the sharing
// page if the API call fails.

package yagpt

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// Client is a YandexGPT summarization client. It is safe for concurrent
// use by multiple goroutines provided the underlying [http.Client]
// is also concurrent-safe. The standard library's [http.Client]
// satisfies this requirement.
//
// A Client must be created with [NewClient]; the zero value is not usable.
type Client struct {
	client *http.Client
	token  string
}

// NewClient returns a Client authenticated with the given OAuth token.
//
// The authToken is trimmed of surrounding whitespace and must be
// non-empty; an empty or whitespace-only token returns an error.
//
// If hc is nil, a default [http.Client] with a [defaultTimeout] is used.
// Callers that need custom transports, timeouts, or middleware
// (for example, retry or logging wrappers) should pass their own [http.Client].
func NewClient(authToken string, hc *http.Client) (*Client, error) {
	authToken = strings.TrimSpace(authToken)
	if authToken == "" {
		return nil, errors.New("yagpt: empty auth token")
	}

	if hc == nil {
		hc = &http.Client{Timeout: defaultTimeout}
	}

	return &Client{client: hc, token: authToken}, nil
}

// GetSummaryURL requests a sharing URL for the given article from the
// YandexGPT API and returns it as a validated [SummaryURL].
//
// The returned SummaryURL carries both the canonical sharing URL and
// the opaque token embedded in its path, which can be passed to
// [Client.GetSummaryContent] to retrieve the summary text.
//
// The context controls cancellation and deadline for the underlying
// HTTP request. Errors from the API, including non-2xx responses and
// malformed payloads, are wrapped with descriptive context.
func (c *Client) GetSummaryURL(ctx context.Context, articleURL string) (SummaryURL, error) {
	su, err := getSharingURL(ctx, c.client, c.token, articleURL)
	if err != nil {
		return SummaryURL{}, fmt.Errorf("yagpt: getting summary URL for %q: %w", articleURL, err)
	}
	return su, nil
}

// GetSummaryContent fetches the summary content for the given
// [SummaryURL] as a slice of thesis lines, one bullet per element.
//
// It first attempts the JSON API, which is the preferred path since
// it returns structured data. If the API call fails for any reason
// (network error, non-2xx status, malformed response), it transparently
// falls back to fetching the sharing page's HTML and extracting the
// summary from its og:description meta tag.
//
// If both the API and HTML paths fail, the returned error joins both
// underlying errors via [errors.Join] so callers can inspect either
// one with [errors.Is] or [errors.As] for diagnostics.
//
// The context controls cancellation and deadline for all underlying HTTP requests.
func (c *Client) GetSummaryContent(ctx context.Context, su SummaryURL) ([]string, error) {
	content, apiErr := getSummaryContentAPI(ctx, c.client, su.Token())
	if apiErr == nil {
		return content, nil
	}

	content, htmlErr := getSummaryContentHTML(ctx, c.client, su.URL())
	if htmlErr == nil {
		return content, nil
	}

	return nil, fmt.Errorf("yagpt: getting summary content from %q: %w", su.String(), errors.Join(apiErr, htmlErr))
}
