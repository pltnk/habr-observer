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
	// Rate limiting for the /api/sharing-url endpoint. The API permits 20 req/min
	// per token; we refill at 19/min with burst 1, so steady state is 19/min and
	// even a cold-start backlog tops out at 1 + 19 = 20 in the first minute — at
	// the cap, never over it. Applied via rate.Limiter in NewClient.
	sharingURLRateLimit       = 19
	sharingURLRateLimitWindow = 60 * time.Second
	sharingURLRateLimitBurst  = 1

	// continuationThesis is the notice 300.ya.ru appends as the last thesis of a
	// long article's summary, telling the reader to switch to the full detailed
	// summary. The app strips it (see stripContinuationThesis).
	continuationThesis = "Пересказана только часть. Для продолжения перейдите в режим подробного пересказа."
)
