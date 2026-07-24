package firezone_test

import (
	"context"
	"net/http"
	"testing"

	firezone "github.com/firezone/firezone-go"
	"github.com/firezone/firezone-go/internal/testutil"
)

func TestEntraDirectoriesService_Get(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		var gotPath string
		client := testutil.NewClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotPath = r.URL.Path
			testutil.JSONResponse(http.StatusOK, map[string]any{
				"data": map[string]any{"id": "dir-1", "name": "Entra", "tenant_id": "tenant-1"},
			})(w, r)
		}))

		dir, err := client.EntraDirectories.Get(context.Background(), "dir-1")
		if err != nil {
			t.Fatalf("Get returned error: %v", err)
		}
		if gotPath != "/v1/entra_directories/dir-1" {
			t.Errorf("path = %q, want /v1/entra_directories/dir-1", gotPath)
		}
		if dir.ID != "dir-1" || dir.Name != "Entra" || dir.TenantID != "tenant-1" {
			t.Errorf("dir = %+v, want {ID: dir-1, Name: Entra, TenantID: tenant-1}", dir)
		}
	})

	t.Run("not found", func(t *testing.T) {
		client := testutil.NewClient(t, testutil.ProblemResponse(http.StatusNotFound, "The requested resource could not be found."))

		_, err := client.EntraDirectories.Get(context.Background(), "missing")
		if !firezone.IsNotFound(err) {
			t.Fatalf("IsNotFound(err) = false, want true (err: %v)", err)
		}
	})
}

func TestEntraDirectoriesService_List(t *testing.T) {
	var gotPath string
	client := testutil.NewClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		testutil.JSONResponse(http.StatusOK, map[string]any{
			"data":     []map[string]any{{"id": "dir-1", "name": "Entra"}},
			"metadata": map[string]any{"count": 1, "limit": 50, "next_page": "", "prev_page": ""},
		})(w, r)
	}))

	page, err := client.EntraDirectories.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if gotPath != "/v1/entra_directories" {
		t.Errorf("path = %q, want /v1/entra_directories", gotPath)
	}
	if len(page.Data) != 1 || page.Data[0].ID != "dir-1" {
		t.Errorf("page.Data = %+v, want one directory with ID dir-1", page.Data)
	}
}

func TestGoogleDirectoriesService_Get(t *testing.T) {
	var gotPath string
	client := testutil.NewClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		testutil.JSONResponse(http.StatusOK, map[string]any{
			"data": map[string]any{"id": "dir-2", "name": "Google", "domain": "example.com"},
		})(w, r)
	}))

	dir, err := client.GoogleDirectories.Get(context.Background(), "dir-2")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if gotPath != "/v1/google_directories/dir-2" {
		t.Errorf("path = %q, want /v1/google_directories/dir-2", gotPath)
	}
	if dir.ID != "dir-2" || dir.Name != "Google" || dir.Domain != "example.com" {
		t.Errorf("dir = %+v, want {ID: dir-2, Name: Google, Domain: example.com}", dir)
	}
}

func TestOktaDirectoriesService_Get(t *testing.T) {
	var gotPath string
	client := testutil.NewClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		testutil.JSONResponse(http.StatusOK, map[string]any{
			"data": map[string]any{"id": "dir-3", "name": "Okta", "okta_domain": "example.okta.com"},
		})(w, r)
	}))

	dir, err := client.OktaDirectories.Get(context.Background(), "dir-3")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if gotPath != "/v1/okta_directories/dir-3" {
		t.Errorf("path = %q, want /v1/okta_directories/dir-3", gotPath)
	}
	if dir.ID != "dir-3" || dir.Name != "Okta" || dir.OktaDomain != "example.okta.com" {
		t.Errorf("dir = %+v, want {ID: dir-3, Name: Okta, OktaDomain: example.okta.com}", dir)
	}
}
