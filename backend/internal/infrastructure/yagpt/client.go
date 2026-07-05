// Package yagpt provides a client for YandexGPT's article summarization
// service (300.ya.ru), which generates short thesis-style summaries of
// online articles.
package yagpt

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"golang.org/x/time/rate"

	"habr-observer/internal/domain"
)

// Client is a YandexGPT summarization client. It is safe for concurrent
// use by multiple goroutines provided the underlying [http.Client]
// is also concurrent-safe. The standard library's [http.Client]
// satisfies this requirement.
//
// A Client must be created with [NewClient]; the zero value is not usable.
type Client struct {
	client  *http.Client
	token   string
	limiter *rate.Limiter
}

// NewClient returns a Client authenticated with the given OAuth token.
//
// The authToken is trimmed of surrounding whitespace and must be
// non-empty; an empty or whitespace-only token returns an error.
//
// If hc is nil, a default [http.Client] with a 60-second timeout is used.
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

	return &Client{
		client:  hc,
		token:   authToken,
		limiter: rate.NewLimiter(rate.Every(sharingURLRateLimitWindow/sharingURLRateLimit), sharingURLRateLimitBurst),
	}, nil
}

// GetSummaryURL requests a sharing URL for the given article from the
// YandexGPT API and returns it as a validated [SummaryURL].
//
// The returned SummaryURL carries both the canonical sharing URL and
// the opaque token embedded in its path, which can be passed to
// [Client.GetSummaryContent] to retrieve the summary text.
//
// Calls to this method are rate-limited client-side to comply with
// the API's published limits; under load, this method may block
// until a token is available or until ctx is cancelled.
//
// The context controls cancellation and deadline for the underlying
// HTTP request. Errors from the API, including non-2xx responses and
// malformed payloads, are wrapped with descriptive context.
func (c *Client) GetSummaryURL(ctx context.Context, articleURL string) (SummaryURL, error) {
	if err := c.limiter.Wait(ctx); err != nil {
		return SummaryURL{}, fmt.Errorf("yagpt: getting summary URL rate limit: %w", err)
	}

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
// summary from its og:description meta tag. The fallback is skipped
// if the context has already been cancelled or its deadline exceeded,
// in which case only the API error is returned.
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

	if ctx.Err() != nil {
		return nil, fmt.Errorf("yagpt: getting summary content from %q: %w", su.String(), apiErr)
	}

	content, htmlErr := getSummaryContentHTML(ctx, c.client, su.URL())
	if htmlErr == nil {
		return content, nil
	}

	return nil, fmt.Errorf("yagpt: getting summary content from %q: %w", su.String(), errors.Join(apiErr, htmlErr))
}

// GetSummary resolves and fetches the summary for the given article in one call,
// composing [Client.GetSummaryURL] and [Client.GetSummaryContent] into a
// [domain.Summary]. It is the convenient path for callers that just want the
// summary; those needing the intermediate sharing URL or finer control over the
// two requests can call the underlying methods directly.
//
// A 404 from the sharing-url endpoint is returned as [ErrSummaryUnavailable]
// (detectable with errors.Is) and the content fetch is skipped. The call is
// rate-limited exactly as [Client.GetSummaryURL].
func (c *Client) GetSummary(ctx context.Context, articleURL string) (*domain.Summary, error) {
	su, err := c.GetSummaryURL(ctx, articleURL)
	if err != nil {
		return nil, err
	}

	content, err := c.GetSummaryContent(ctx, su)
	if err != nil {
		return nil, err
	}

	return &domain.Summary{URL: su.String(), Content: content}, nil
}
