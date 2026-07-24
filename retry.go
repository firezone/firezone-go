package firezone

import (
	"context"
	"errors"
	"math"
	"net/url"
	"time"
)

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

		wait := apiErr.RetryAfter
		if wait <= 0 {
			wait = exponentialBackoff(attempt)
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(wait):
		}
	}

	return nil, lastErr
}

// exponentialBackoff returns 1s, 2s, 4s, 8s, ... for attempt 0, 1, 2, 3,
// used as a fallback when a 429 response carries no Retry-After header.
func exponentialBackoff(attempt int) time.Duration {
	return time.Duration(math.Pow(2, float64(attempt))) * time.Second
}
