package habr

import (
	"bytes"
	"context"
	"fmt"
	"habr-observer/internal/domain"
	"io"
	"net/http"
	"time"
)

const defaultTimeout = 10 * time.Second
const maxErrSnippet = 2048 // 2KiB

type Client struct {
	c *http.Client
}

// NewClient returns a Client backed by hc. If hc is nil, a default
// [http.Client] with a 10-second timeout is used. A Client must be created
// with NewClient; the zero value is not usable.
func NewClient(hc *http.Client) *Client {
	if hc == nil {
		hc = &http.Client{Timeout: defaultTimeout}
	}
	return &Client{c: hc}
}

func (c *Client) getXML(ctx context.Context, f RSSFeed) ([]byte, error) {
	url := f.URL()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("getting XML from %q: %w", url, err)
	}

	resp, err := c.c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("getting XML from %q: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrSnippet))
		return nil, fmt.Errorf("getting XML from %q: HTTP %s: %q", url, resp.Status, string(bytes.TrimSpace(body)))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("getting XML from %q: %w", url, err)
	}

	return data, nil
}

func (c *Client) GetArticles(ctx context.Context, f RSSFeed) ([]*domain.Article, error) {
	xml, err := c.getXML(ctx, f)
	if err != nil {
		return nil, fmt.Errorf("getting articles for %s: %w", f, err)
	}

	articles, err := parseXML(xml)
	if err != nil {
		return nil, fmt.Errorf("getting articles for %s: %w", f, err)
	}

	return articles, nil
}
