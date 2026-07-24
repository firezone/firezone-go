package firezone_test

import (
	"context"
	"net/http"
	"testing"

	firezone "github.com/firezone/firezone-go"
	"github.com/firezone/firezone-go/internal/testutil"
)

func TestResourcesService_Create(t *testing.T) {
	t.Run("happy path with filters", func(t *testing.T) {
		var gotBody map[string]any
		client := testutil.NewClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			decodeJSONBody(t, r, &gotBody)
			testutil.JSONResponse(http.StatusCreated, map[string]any{
				"data": map[string]any{
					"id": "res-1", "name": "postgres-prod", "type": "cidr",
					"address": "10.0.1.0/24", "address_description": "Production Postgres subnet",
					"site_id": "site-1",
					"filters": []map[string]any{
						{"protocol": "tcp", "ports": []string{"5432"}},
					},
				},
			})(w, r)
		}))

		resource, err := client.Resources.Create(context.Background(), &firezone.CreateResourceRequest{
			Name:    "postgres-prod",
			Type:    firezone.ResourceTypeCIDR,
			Address: "10.0.1.0/24",
			SiteID:  "site-1",
			Filters: []firezone.Filter{{Protocol: firezone.FilterProtocolTCP, Ports: []string{"5432"}}},
		})
		if err != nil {
			t.Fatalf("Create returned error: %v", err)
		}

		reqResource, ok := gotBody["resource"].(map[string]any)
		if !ok {
			t.Fatalf("body[resource] = %v, want an object", gotBody["resource"])
		}
		if reqResource["type"] != "cidr" {
			t.Errorf("body.resource.type = %v, want cidr", reqResource["type"])
		}

		if resource.Type != firezone.ResourceTypeCIDR {
			t.Errorf("resource.Type = %q, want cidr", resource.Type)
		}
		if len(resource.Filters) != 1 || resource.Filters[0].Protocol != firezone.FilterProtocolTCP {
			t.Errorf("resource.Filters = %+v, want one tcp filter", resource.Filters)
		}
	})

	t.Run("internet type is forbidden", func(t *testing.T) {
		client := testutil.NewClient(t, testutil.ProblemResponse(http.StatusForbidden, "Internet Resource cannot be modified"))

		_, err := client.Resources.Create(context.Background(), &firezone.CreateResourceRequest{Name: "internet"})
		if !firezone.IsForbidden(err) {
			t.Fatalf("IsForbidden(err) = false, want true (err: %v)", err)
		}
	})
}

func TestResourcesService_Delete(t *testing.T) {
	client := testutil.NewClient(t, testutil.JSONResponse(http.StatusOK, map[string]any{
		"data": map[string]any{"id": "res-1", "name": "postgres-prod"},
	}))

	if err := client.Resources.Delete(context.Background(), "res-1"); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
}

func TestResourcesService_List_Filters(t *testing.T) {
	tests := []struct {
		name      string
		opts      *firezone.ResourceListOptions
		wantQuery string
	}{
		{name: "by name", opts: &firezone.ResourceListOptions{Name: "postgres-prod"}, wantQuery: "name=postgres-prod"},
		{name: "by type", opts: &firezone.ResourceListOptions{Type: firezone.ResourceTypeDNS}, wantQuery: "type=dns"},
		{name: "by site_id", opts: &firezone.ResourceListOptions{SiteID: "site-1"}, wantQuery: "site_id=site-1"},
		{name: "by address", opts: &firezone.ResourceListOptions{Address: "10.0.0.10"}, wantQuery: "address=10.0.0.10"},
		{name: "by ip_stack", opts: &firezone.ResourceListOptions{IPStack: firezone.IPStackDual}, wantQuery: "ip_stack=dual"},
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

			_, err := client.Resources.List(context.Background(), tt.opts)
			if err != nil {
				t.Fatalf("List returned error: %v", err)
			}
			if gotQuery != tt.wantQuery {
				t.Errorf("query = %q, want %q", gotQuery, tt.wantQuery)
			}
		})
	}
}
