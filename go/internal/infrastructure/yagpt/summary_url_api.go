package yagpt

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type sharingURLRequestPayload struct {
	ArticleURL string `json:"article_url"`
}

type sharingURLResponse struct {
	Status     string `json:"status"`
	SharingURL string `json:"sharing_url"`
}

func getSharingURL(ctx context.Context, doer httpDoer, authToken, articleURL string) (SummaryURL, error) {
	articleURL = strings.TrimSpace(articleURL)
	if articleURL == "" {
		return SummaryURL{}, errors.New("API: empty articleURL")
	}

	authToken = strings.TrimSpace(authToken)
	if authToken == "" {
		return SummaryURL{}, errors.New("API: empty authToken")
	}

	if doer == nil {
		return SummaryURL{}, errors.New("API: nil httpDoer")
	}

	reqBody, err := json.Marshal(sharingURLRequestPayload{ArticleURL: articleURL})
	if err != nil {
		return SummaryURL{}, fmt.Errorf("API: marshalling request payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, sharingURLEndpoint, bytes.NewReader(reqBody))
	if err != nil {
		return SummaryURL{}, fmt.Errorf("API: creating request: %w", err)
	}
	req.Header.Set("Authorization", "OAuth "+authToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := doer.Do(req)
	if err != nil {
		return SummaryURL{}, fmt.Errorf("API: doing request: %w", err)
	}
	defer func() {
		io.Copy(io.Discard, io.LimitReader(resp.Body, maxDrainBodySize))
		resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrSnippetBodySize))
		return SummaryURL{}, fmt.Errorf("API: HTTP %s: %q", resp.Status, string(bytes.TrimSpace(body)))
	}

	var parsed sharingURLResponse
	err = json.NewDecoder(io.LimitReader(resp.Body, maxReadBodySize)).Decode(&parsed)
	if err != nil {
		return SummaryURL{}, fmt.Errorf("API: decoding response: %w", err)
	}

	if parsed.Status != "success" {
		return SummaryURL{}, fmt.Errorf("API: unsuccessful status in JSON response: %q", parsed.Status)
	}

	if parsed.SharingURL == "" {
		return SummaryURL{}, errors.New("API: empty sharing_url in JSON response")
	}

	su, err := NewSummaryURL(parsed.SharingURL)
	if err != nil {
		return SummaryURL{}, fmt.Errorf("API: creating summary URL: %w", err)
	}

	return su, nil
}
