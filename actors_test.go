package firezone_test

import (
	"context"
	"net/http"
	"testing"

	firezone "github.com/firezone/firezone-go"
	"github.com/firezone/firezone-go/internal/testutil"
)

func TestActorsService_DisableEnable(t *testing.T) {
	t.Run("disable", func(t *testing.T) {
		var gotMethod, gotPath string
		client := testutil.NewClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotMethod, gotPath = r.Method, r.URL.Path
			testutil.JSONResponse(http.StatusOK, map[string]any{
				"data": map[string]any{"id": "actor-1", "name": "svc", "type": "service_account", "disabled_at": "2026-01-01T00:00:00Z"},
			})(w, r)
		}))

		actor, err := client.Actors.Disable(context.Background(), "actor-1")
		if err != nil {
			t.Fatalf("Disable returned error: %v", err)
		}
		if gotMethod != http.MethodPost || gotPath != "/actors/actor-1/disable" {
			t.Errorf("request = %s %s, want POST /actors/actor-1/disable", gotMethod, gotPath)
		}
		if !actor.IsDisabled() {
			t.Error("actor.IsDisabled() = false, want true")
		}
	})

	t.Run("cannot disable self", func(t *testing.T) {
		client := testutil.NewClient(t, testutil.ProblemResponse(http.StatusForbidden, "You cannot disable yourself"))
		_, err := client.Actors.Disable(context.Background(), "self")
		if !firezone.IsForbidden(err) {
			t.Fatalf("IsForbidden(err) = false, want true (err: %v)", err)
		}
	})
}

func TestActorsService_Create(t *testing.T) {
	t.Run("service account limit reached", func(t *testing.T) {
		client := testutil.NewClient(t, testutil.ProblemResponse(http.StatusForbidden, "Service accounts limit reached"))

		_, err := client.Actors.Create(context.Background(), &firezone.CreateActorRequest{
			Name: "svc", Type: firezone.ActorTypeServiceAccount,
		})
		if !firezone.IsForbidden(err) {
			t.Fatalf("IsForbidden(err) = false, want true (err: %v)", err)
		}
	})
}

func TestActorsService_List_Filters(t *testing.T) {
	tests := []struct {
		name      string
		opts      *firezone.ActorListOptions
		wantQuery string
	}{
		{name: "by name", opts: &firezone.ActorListOptions{Name: "alice"}, wantQuery: "name=alice"},
		{
			name:      "by email",
			opts:      &firezone.ActorListOptions{Email: "alice@example.com"},
			wantQuery: "email=alice%40example.com",
		},
		{
			name:      "by type",
			opts:      &firezone.ActorListOptions{Type: firezone.ActorTypeServiceAccount},
			wantQuery: "type=service_account",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotQuery string
			client := testutil.NewClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotQuery = r.URL.RawQuery
				testutil.JSONResponse(http.StatusOK, map[string]any{
					"data":     []map[string]any{},
					"metadata": map[string]any{"count": 0, "limit": 50, "next_page": "", "prev_page": ""},
				})(w, r)
			}))

			_, err := client.Actors.List(context.Background(), tt.opts)
			if err != nil {
				t.Fatalf("List returned error: %v", err)
			}
			if gotQuery != tt.wantQuery {
				t.Errorf("query = %q, want %q", gotQuery, tt.wantQuery)
			}
		})
	}
}
