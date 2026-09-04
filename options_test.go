package firezone_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	firezone "github.com/firezone/firezone-go"
	"github.com/firezone/firezone-go/internal/testutil"
)

// TestNewClientRejectsNilHTTPClient checks that a nil *http.Client is
// caught at construction. An Option can't report an error itself, so
// without this check the nil survives into the Client and the first
// request panics with a nil dereference, pointing at the request
// instead of at the option that caused it.
func TestNewClientRejectsNilHTTPClient(t *testing.T) {
	client, err := firezone.NewClient("https://api.example.com", "token",
		firezone.WithHTTPClient(nil))
	if err == nil {
		t.Fatalf("NewClient with a nil *http.Client returned no error, client = %v", client != nil)
	}
	if client != nil {
		t.Errorf("NewClient returned a non-nil client alongside an error")
	}
}

// TestWithRetryClampsNegativeBudget is a regression test for a retry
// loop that made zero attempts.
//
// The loop runs `attempt <= maxRetries`, so a negative budget skipped
// the body entirely and fell through to a nil body with a nil error.
// That reads to the caller as a successful request that returned
// nothing: no HTTP request was made, and the destination struct was
// left at its zero value with no error to say so.
func TestWithRetryClampsNegativeBudget(t *testing.T) {
	client := testutil.NewClientWithOptions(t,
		testutil.JSONResponse(http.StatusOK, map[string]any{
			"data": map[string]any{"id": "site-id", "name": "primary-dc"},
		}),
		firezone.WithRetry(true, -1),
	)

	// The assertion is on the decoded result rather than a request
	// counter: Name is only populated if a request actually happened.
	site, err := client.Sites.Get(context.Background(), "site-id")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if site.Name != "primary-dc" {
		t.Errorf("Name = %q, want %q - a zero value here means no request was made", site.Name, "primary-dc")
	}
}

// TestNilRequestBody checks that a nil request struct is rejected
// before a request is made, for every method that takes one.
//
// A nil *CreateSiteRequest reaches wrapBody as an any holding a typed
// nil pointer, so a plain v == nil check reads false and the body
// encodes as {"site": null} - a request the API rejects for a reason
// that names nothing useful.
func TestNilRequestBody(t *testing.T) {
	const id = "00000000-0000-0000-0000-000000000000"

	client := testutil.NewClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusInternalServerError)
	}))

	ctx := context.Background()
	calls := map[string]func() error{
		"Sites.Create":     func() error { _, err := client.Sites.Create(ctx, nil); return err },
		"Sites.Update":     func() error { _, err := client.Sites.Update(ctx, id, nil); return err },
		"Resources.Create": func() error { _, err := client.Resources.Create(ctx, nil); return err },
		"Resources.Update": func() error { _, err := client.Resources.Update(ctx, id, nil); return err },
		"Policies.Create":  func() error { _, err := client.Policies.Create(ctx, nil); return err },
		"Policies.Update":  func() error { _, err := client.Policies.Update(ctx, id, nil); return err },
		"Groups.Create":    func() error { _, err := client.Groups.Create(ctx, nil); return err },
		"Groups.Update":    func() error { _, err := client.Groups.Update(ctx, id, nil); return err },
		"Actors.Create":    func() error { _, err := client.Actors.Create(ctx, nil); return err },
		"Actors.Update":    func() error { _, err := client.Actors.Update(ctx, id, nil); return err },
		"ClientDevices.Update": func() error {
			_, err := client.ClientDevices.Update(ctx, id, nil)
			return err
		},
		"Gateways.Provision": func() error {
			_, err := client.Sites.Gateways(id).Provision(ctx, nil)
			return err
		},
		"Gateways.Update": func() error {
			_, err := client.Sites.Gateways(id).Update(ctx, id, nil)
			return err
		},
	}

	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			err := call()
			if err == nil {
				t.Fatal("nil request body returned no error")
			}
			if !errors.Is(err, firezone.ErrNilRequest) {
				t.Errorf("error = %v, want one matching ErrNilRequest", err)
			}
		})
	}
}
