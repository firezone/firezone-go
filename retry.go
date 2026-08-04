package firezone

import (
	"context"
	"errors"
	"math"
	"math/rand/v2"
	"net/url"
	"time"
)

// maxRetryBackoff caps the exponential fallback. Without it the 8th
// attempt would sleep over two minutes, long past the point where a
// caller would rather see the error.
const maxRetryBackoff = 30 * time.Second

// maxRetryJitter caps the random padding added to every wait. The
// padding exists to break up thundering herds: Terraform applies with
// the default parallelism of 10 send bursts of concurrent requests, and
// without jitter every one of them is rate limited at the same instant,
// waits the same duration, and retries in lockstep - so the same subset
// keeps winning and the rest exhaust their retries.
const maxRetryJitter = 2 * time.Second

// requestWithRetry calls rawRequest, retrying on HTTP 429 (rate
// limited) responses. It waits for the duration in the response's
// Retry-After header when present, falling back to exponential backoff
// otherwise. Retrying is a request-level concern (not a
// http.RoundTripper-level one) because it needs structured access to
// APIError.RetryAfter, which a RoundTripper wrapper would have to
// re-parse the body to get.
func (c *Client) requestWithRetry(ctx context.Context, method, requestPath string, query url.Values, body requestBody) ([]byte, error) {
	if !c.retryEnabled {
		return c.rawRequest(ctx, method, requestPath, query, body)
	}

	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		respBody, err := c.rawRequest(ctx, method, requestPath, query, body)
		if err == nil {
			return respBody, nil
		}

		var apiErr *APIError
		if !errors.As(err, &apiErr) || !IsRateLimited(err) || attempt == c.maxRetries {
			return nil, err
		}
		lastErr = err

		// Retry-After is a floor, not a target: waiting less than the
		// server asked guarantees another 429. Jitter is therefore added
		// on top of it, never subtracted from it.
		wait := apiErr.RetryAfter
		if wait <= 0 {
			wait = exponentialBackoff(attempt)
		}
		wait += retryJitter(wait)

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(wait):
		}
	}

	return nil, lastErr
}

// exponentialBackoff returns 1s, 2s, 4s, 8s, ... for attempt 0, 1, 2, 3,
// capped at maxRetryBackoff. Used as a fallback when a 429 response
// carries no Retry-After header.
func exponentialBackoff(attempt int) time.Duration {
	// Guard the shift before math.Pow overflows into +Inf, which would
	// convert to a nonsense Duration.
	if attempt >= 30 {
		return maxRetryBackoff
	}
	backoff := time.Duration(math.Pow(2, float64(attempt))) * time.Second
	return min(backoff, maxRetryBackoff)
}

// retryJitter returns a random padding of up to half the base wait,
// capped at maxRetryJitter, so concurrent callers rate limited at the
// same moment don't all retry at the same moment too.
func retryJitter(base time.Duration) time.Duration {
	window := min(base/2, maxRetryJitter)
	if window <= 0 {
		return 0
	}
	return time.Duration(rand.Int64N(int64(window)))
}
