package firezone_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	firezone "github.com/firezone/firezone-go"
)

func TestRetry_SucceedsAfterRateLimitedResponses(t *testing.T) {
	var attempts int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.Header().Set("Retry-After", "0")
			w.Header().Set("Content-Type", "application/problem+json")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"type":"about:blank","title":"Too Many Requests","status":429,"detail":"slow down"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":{"id":"site-1","name":"primary-dc"}}`))
	}))
	defer server.Close()

	client, err := firezone.NewClient(server.URL, "test-token", firezone.WithRetry(true, 5))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	site, err := client.Sites.Get(ctx, "site-1")
	if err != nil {
		t.Fatalf("Get returned error after retries: %v", err)
	}
	if attempts != 3 {
		t.Errorf("attempts = %d, want 3", attempts)
	}
	if site.ID != "site-1" {
		t.Errorf("site.ID = %q, want site-1", site.ID)
	}
}

func TestRetry_GivesUpAfterMaxRetries(t *testing.T) {
	var attempts int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.Header().Set("Retry-After", "0")
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"type":"about:blank","title":"Too Many Requests","status":429,"detail":"slow down"}`))
	}))
	defer server.Close()

	client, err := firezone.NewClient(server.URL, "test-token", firezone.WithRetry(true, 2))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = client.Sites.Get(context.Background(), "site-1")
	if !firezone.IsRateLimited(err) {
		t.Fatalf("IsRateLimited(err) = false, want true (err: %v)", err)
	}
	// maxRetries=2 means 3 total attempts (initial + 2 retries).
	if attempts != 3 {
		t.Errorf("attempts = %d, want 3", attempts)
	}
}

func TestRetry_DisabledDoesNotRetry(t *testing.T) {
	var attempts int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.Header().Set("Retry-After", "0")
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"type":"about:blank","title":"Too Many Requests","status":429,"detail":"slow down"}`))
	}))
	defer server.Close()

	client, err := firezone.NewClient(server.URL, "test-token", firezone.WithRetry(false, 0))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = client.Sites.Get(context.Background(), "site-1")
	if !firezone.IsRateLimited(err) {
		t.Fatalf("IsRateLimited(err) = false, want true (err: %v)", err)
	}
	if attempts != 1 {
		t.Errorf("attempts = %d, want 1 (retry disabled)", attempts)
	}
}
