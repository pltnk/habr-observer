package yagpt

import "time"

const (
	baseHostname          = "300.ya.ru"
	baseURL               = "https://" + baseHostname
	sharingURLEndpoint    = baseURL + "/api/sharing-url"
	sharingEndpoint       = baseURL + "/api/sharing"
	defaultTimeout        = 60 * time.Second
	maxErrSnippetBodySize = 2048                // 2 KiB – hard cap for error snippets
	maxReadBodySize       = 5 * 1024 * 1024     // 5 MiB – hard cap on what we'll parse
	maxDrainBodySize      = 2 * maxReadBodySize // 10 MiB – drain cap for connection reuse
	// Rate limit for the /api/sharing-url endpoint: 20 req/min per token,
	// implemented client-side as a token bucket.
	sharingURLRateLimit       = 18 // requests per time window (20 with safety margin)
	sharingURLRateLimitWindow = 60 * time.Second
	sharingURLRateLimitBurst  = 6

	// continuationThesis is the notice 300.ya.ru appends as the last thesis of a
	// long article's summary, telling the reader to switch to the full detailed
	// summary. The app strips it (see stripContinuationThesis).
	continuationThesis = "Пересказана только часть. Для продолжения перейдите в режим подробного пересказа."
)
