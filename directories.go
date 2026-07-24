package firezone

import (
	"context"
	"time"
)

// EntraDirectory is a Microsoft Entra directory connection. Directories
// are read-only via this API - they're managed through the Firezone
// dashboard's identity provider setup, not created or updated here.
type EntraDirectory struct {
	ID              string     `json:"id"`
	AccountID       string     `json:"account_id"`
	Name            string     `json:"name"`
	TenantID        string     `json:"tenant_id"`
	ErrorEmailCount int        `json:"error_email_count"`
	IsDisabled      bool       `json:"is_disabled"`
	DisabledReason  string     `json:"disabled_reason,omitempty"`
	SyncedAt        *time.Time `json:"synced_at,omitempty"`
	ErrorMessage    string     `json:"error_message,omitempty"`
	ErroredAt       *time.Time `json:"errored_at,omitempty"`
	EmailField      string     `json:"email_field"`
	SyncAllGroups   bool       `json:"sync_all_groups"`
	InsertedAt      time.Time  `json:"inserted_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// GoogleDirectory is a Google Workspace directory connection. Directories
// are read-only via this API - they're managed through the Firezone
// dashboard's identity provider setup, not created or updated here.
type GoogleDirectory struct {
	ID                 string     `json:"id"`
	AccountID          string     `json:"account_id"`
	Name               string     `json:"name"`
	Domain             string     `json:"domain"`
	ImpersonationEmail string     `json:"impersonation_email"`
	ErrorEmailCount    int        `json:"error_email_count"`
	IsDisabled         bool       `json:"is_disabled"`
	DisabledReason     string     `json:"disabled_reason,omitempty"`
	SyncedAt           *time.Time `json:"synced_at,omitempty"`
	ErrorMessage       string     `json:"error_message,omitempty"`
	ErroredAt          *time.Time `json:"errored_at,omitempty"`
	GroupSyncMode      string     `json:"group_sync_mode"`
	OrgUnitSyncEnabled bool       `json:"orgunit_sync_enabled"`
	InsertedAt         time.Time  `json:"inserted_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

// OktaDirectory is an Okta directory connection. Directories are
// read-only via this API - they're managed through the Firezone
// dashboard's identity provider setup, not created or updated here.
type OktaDirectory struct {
	ID              string     `json:"id"`
	AccountID       string     `json:"account_id"`
	Name            string     `json:"name"`
	ClientID        string     `json:"client_id"`
	Kid             string     `json:"kid"`
	OktaDomain      string     `json:"okta_domain"`
	ErrorEmailCount int        `json:"error_email_count"`
	IsDisabled      bool       `json:"is_disabled"`
	DisabledReason  string     `json:"disabled_reason,omitempty"`
	SyncedAt        *time.Time `json:"synced_at,omitempty"`
	ErrorMessage    string     `json:"error_message,omitempty"`
	ErroredAt       *time.Time `json:"errored_at,omitempty"`
	InsertedAt      time.Time  `json:"inserted_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// EntraDirectoriesService reads Entra directory connections. Read-only:
// there is no Create, Update, or Delete - see [EntraDirectory].
type EntraDirectoriesService struct {
	client *Client
}

// Get fetches a single Entra directory by ID.
func (s *EntraDirectoriesService) Get(ctx context.Context, id string) (*EntraDirectory, error) {
	var d EntraDirectory
	if err := s.client.do(ctx, "GET", "entra_directories/"+id, nil, nil, &d); err != nil {
		return nil, err
	}
	return &d, nil
}

// List returns a page of Entra directories. Pass nil for opts to use
// the API's default page size.
func (s *EntraDirectoriesService) List(ctx context.Context, opts *ListOptions) (*Page[EntraDirectory], error) {
	return doList[EntraDirectory](ctx, s.client, "GET", "entra_directories", listOptionsToQuery(opts))
}

// GoogleDirectoriesService reads Google Workspace directory connections.
// Read-only: there is no Create, Update, or Delete - see [GoogleDirectory].
type GoogleDirectoriesService struct {
	client *Client
}

// Get fetches a single Google Workspace directory by ID.
func (s *GoogleDirectoriesService) Get(ctx context.Context, id string) (*GoogleDirectory, error) {
	var d GoogleDirectory
	if err := s.client.do(ctx, "GET", "google_directories/"+id, nil, nil, &d); err != nil {
		return nil, err
	}
	return &d, nil
}

// List returns a page of Google Workspace directories. Pass nil for
// opts to use the API's default page size.
func (s *GoogleDirectoriesService) List(ctx context.Context, opts *ListOptions) (*Page[GoogleDirectory], error) {
	return doList[GoogleDirectory](ctx, s.client, "GET", "google_directories", listOptionsToQuery(opts))
}

// OktaDirectoriesService reads Okta directory connections. Read-only:
// there is no Create, Update, or Delete - see [OktaDirectory].
type OktaDirectoriesService struct {
	client *Client
}

// Get fetches a single Okta directory by ID.
func (s *OktaDirectoriesService) Get(ctx context.Context, id string) (*OktaDirectory, error) {
	var d OktaDirectory
	if err := s.client.do(ctx, "GET", "okta_directories/"+id, nil, nil, &d); err != nil {
		return nil, err
	}
	return &d, nil
}

// List returns a page of Okta directories. Pass nil for opts to use the
// API's default page size.
func (s *OktaDirectoriesService) List(ctx context.Context, opts *ListOptions) (*Page[OktaDirectory], error) {
	return doList[OktaDirectory](ctx, s.client, "GET", "okta_directories", listOptionsToQuery(opts))
}
