package yagpt

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

type sharingRequestPayload struct {
	Token string `json:"token"`
}

type sharingResponse struct {
	Thesis []struct {
		Content string `json:"content"`
	} `json:"thesis"`
}

func getSummaryContentAPI(ctx context.Context, doer httpDoer, token string) ([]string, error) {
	if token == "" {
		return nil, errors.New("API: empty token")
	}

	if doer == nil {
		return nil, errors.New("API: nil httpDoer")
	}

	reqBody, err := json.Marshal(sharingRequestPayload{Token: token})
	if err != nil {
		return nil, fmt.Errorf("API: marshalling request payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, sharingEndpoint, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("API: creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := doer.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API: doing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrSnippetBodySize))
		return nil, fmt.Errorf("API: HTTP %s: %q", resp.Status, string(bytes.TrimSpace(body)))
	}

	var parsed sharingResponse
	err = json.NewDecoder(resp.Body).Decode(&parsed)
	if err != nil {
		return nil, fmt.Errorf("API: decoding response: %w", err)
	}

	result := make([]string, len(parsed.Thesis))
	for i := range parsed.Thesis {
		result[i] = parsed.Thesis[i].Content
	}

	return result, nil
}
