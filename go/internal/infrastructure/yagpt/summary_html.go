package yagpt

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

const (
	maxBodySize  = 5 * 1024 * 1024 // 5 MiB – hard cap on what we'll parse
	maxDrainSize = 2 * maxBodySize // 10 MiB – drain cap for connection reuse
)

func getSummaryContentHTML(ctx context.Context, doer httpDoer, u url.URL) ([]string, error) {
	if doer == nil {
		return nil, errors.New("HTML: nil httpDoer")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("HTML: creating request: %w", err)
	}

	resp, err := doer.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTML: doing request: %w", err)
	}
	defer func() {
		io.Copy(io.Discard, io.LimitReader(resp.Body, maxDrainSize))
		resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrSnippet))
		return nil, fmt.Errorf("HTML: HTTP %s: %q", resp.Status, string(bytes.TrimSpace(body)))
	}

	parsed, err := html.Parse(io.LimitReader(resp.Body, maxBodySize))
	if err != nil {
		return nil, fmt.Errorf("HTML: parsing HTML: %w", err)
	}

	result, err := extractSummaryContent(parsed)
	if err != nil {
		return nil, fmt.Errorf("HTML: extracting summary content: %w", err)
	}

	return result, nil
}

func extractSummaryContent(root *html.Node) ([]string, error) {
	if root == nil {
		return nil, errors.New("nil root node")
	}

	meta := findFirst(root, func(n *html.Node) bool {
		if n.Type != html.ElementNode || n.Data != "meta" {
			return false
		}
		for _, a := range n.Attr {
			if a.Key == "property" && a.Val == "og:description" {
				return true
			}
		}
		return false
	})
	if meta == nil {
		return nil, errors.New("og:description meta tag not found")
	}

	var content string
	for _, a := range meta.Attr {
		if a.Key == "content" {
			content = a.Val
			break
		}
	}
	if content == "" {
		return nil, errors.New("og:description is empty")
	}

	lines := strings.Split(content, "\n")
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		line = cleanRawLine(line)
		if line != "" {
			result = append(result, line)
		}
	}
	if len(result) == 0 {
		return nil, errors.New("cleaned og:description is empty")
	}

	return result, nil
}

type nodeMatcherFunc func(*html.Node) bool

func findFirst(n *html.Node, match nodeMatcherFunc) *html.Node {
	if n == nil {
		return nil
	}

	if match(n) {
		return n
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if m := findFirst(c, match); m != nil {
			return m
		}
	}

	return nil
}

func cleanRawLine(line string) string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "•")
	line = strings.TrimSpace(line)
	return line
}
