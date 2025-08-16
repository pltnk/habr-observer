package habr

import (
	"context"
	"fmt"
	"habr-observer/internal/entities"
	"io"
	"net/http"
	"sync"
	"time"
)

const defaultTimeout = 10 * time.Second

type Client struct {
	c    *http.Client
	once sync.Once
}

func NewClient(hc *http.Client) *Client {
	if hc == nil {
		hc = &http.Client{Timeout: defaultTimeout}
	}
	return &Client{c: hc}
}

func (c *Client) http() *http.Client {
	if c == nil {
		panic("nil *habr.Client: construct with habr.NewClient(nil)")
	}

	c.once.Do(func() {
		if c.c == nil {
			c.c = &http.Client{Timeout: defaultTimeout}
		}
	})

	return c.c
}

func (c *Client) getXML(ctx context.Context, f RSSFeed) ([]byte, error) {
	url := f.URL()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("getting XML from %q: %w", url, err)
	}

	resp, err := c.http().Do(req)
	if err != nil {
		return nil, fmt.Errorf("getting XML from %q: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("getting XML from %q: HTTP %d %s", url, resp.StatusCode, resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("getting XML from %q: %w", url, err)
	}

	return data, nil
}

func (c *Client) GetArticles(ctx context.Context, f RSSFeed) ([]*entities.Article, error) {
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
