package firezone

import "context"

// Gateway is a Firezone Gateway - a host that exposes a Site's
// Resources to Clients. Gateways self-register their IP addresses on
// first connect, so IPv4/IPv6 are empty until then.
type Gateway struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	IPv4   string `json:"ipv4"`
	IPv6   string `json:"ipv6"`
	Online bool   `json:"online"`
}

// ProvisionedGateway is a newly provisioned Gateway along with its
// one-time token secret, returned only from [GatewaysService.Provision].
// The API never re-exposes the token after creation - see
// [GatewaysService.Get], which returns a plain [Gateway] with no token
// field at all, so the type system itself prevents a caller from
// expecting a token after a refresh.
type ProvisionedGateway struct {
	Gateway
	// Token is the one-time Gateway token secret. Store it securely -
	// it cannot be retrieved again.
	Token string `json:"token"`
}

// ProvisionGatewayRequest is the request body for
// [GatewaysService.Provision]. Name is optional; the API generates a
// random name when omitted.
type ProvisionGatewayRequest struct {
	Name string `json:"name,omitempty"`
}

// UpdateGatewayRequest is the request body for [GatewaysService.Update].
// Name is the only mutable field - a Gateway's Site is permanent.
type UpdateGatewayRequest struct {
	Name string `json:"name"`
}

// GatewaysService manages the Gateways belonging to a single Site.
// Obtain one via [SitesService.Gateways].
type GatewaysService struct {
	client *Client
	siteID string
}

func (s *GatewaysService) basePath() string {
	return "sites/" + s.siteID + "/gateways"
}

// Get fetches a single Gateway by ID.
func (s *GatewaysService) Get(ctx context.Context, id string) (*Gateway, error) {
	var gateway Gateway
	if err := s.client.do(ctx, "GET", s.basePath()+"/"+id, nil, nil, &gateway); err != nil {
		return nil, err
	}
	return &gateway, nil
}

// GatewayListOptions extends ListOptions with Gateways-specific
// filters. There's no SiteID filter here - the Site is already fixed
// by which [GatewaysService] you called List on (see
// [SitesService.Gateways]).
type GatewayListOptions struct {
	ListOptions
	// Name filters to the Gateway with this exact name.
	Name string
	// IPv4 filters to the Gateway with this exact IPv4 address.
	IPv4 string
	// IPv6 filters to the Gateway with this exact IPv6 address.
	IPv6 string
}

// List returns a page of the Site's Gateways. Pass nil for opts to use
// the API's default page size and no filters.
func (s *GatewaysService) List(ctx context.Context, opts *GatewayListOptions) (*Page[Gateway], error) {
	if opts == nil {
		opts = &GatewayListOptions{}
	}
	q := filterQuery(opts.ListOptions,
		[2]string{"name", opts.Name},
		[2]string{"ipv4", opts.IPv4},
		[2]string{"ipv6", opts.IPv6},
	)
	return doList[Gateway](ctx, s.client, "GET", s.basePath(), q)
}

// Provision creates a new Gateway and mints its single-owner token in
// one call. The returned Token is shown once - store it securely.
func (s *GatewaysService) Provision(ctx context.Context, req *ProvisionGatewayRequest) (*ProvisionedGateway, error) {
	body, err := wrapBody("gateway", req)
	if err != nil {
		return nil, err
	}
	var provisioned ProvisionedGateway
	if err := s.client.do(ctx, "POST", s.basePath(), nil, body, &provisioned); err != nil {
		return nil, err
	}
	return &provisioned, nil
}

// Update renames a Gateway.
func (s *GatewaysService) Update(ctx context.Context, id string, req *UpdateGatewayRequest) (*Gateway, error) {
	body, err := wrapBody("gateway", req)
	if err != nil {
		return nil, err
	}
	var gateway Gateway
	if err := s.client.do(ctx, "PUT", s.basePath()+"/"+id, nil, body, &gateway); err != nil {
		return nil, err
	}
	return &gateway, nil
}

// Delete deletes a Gateway, revoking its token.
func (s *GatewaysService) Delete(ctx context.Context, id string) error {
	return s.client.do(ctx, "DELETE", s.basePath()+"/"+id, nil, nil, nil)
}
