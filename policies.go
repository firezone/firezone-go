package firezone

import "context"

// ConditionProperty is the subject property a policy [Condition]
// evaluates.
type ConditionProperty string

// ConditionProperty values.
const (
	ConditionPropertyRemoteIPLocationRegion ConditionProperty = "remote_ip_location_region"
	ConditionPropertyRemoteIP               ConditionProperty = "remote_ip"
	ConditionPropertyAuthProviderID         ConditionProperty = "auth_provider_id"
	ConditionPropertyClientVerified         ConditionProperty = "client_verified"
)

// ConditionOperator is the comparison a policy [Condition] applies
// between the subject property and Values. Which operators are valid
// depends on Property - see the API's policy schema documentation.
type ConditionOperator string

// ConditionOperator values.
const (
	ConditionOperatorIsIn        ConditionOperator = "is_in"
	ConditionOperatorIsNotIn     ConditionOperator = "is_not_in"
	ConditionOperatorIsInCIDR    ConditionOperator = "is_in_cidr"
	ConditionOperatorIsNotInCIDR ConditionOperator = "is_not_in_cidr"
	ConditionOperatorIs          ConditionOperator = "is"
)

// Condition restricts when a Policy grants access. See the API's
// policy schema for which Operators are valid for each Property and
// how Values is interpreted.
type Condition struct {
	Property ConditionProperty `json:"property"`
	Operator ConditionOperator `json:"operator"`
	Values   []string          `json:"values"`
}

// Policy grants a Group access to a Resource, optionally restricted by
// Conditions.
type Policy struct {
	ID                    string      `json:"id"`
	GroupID               string      `json:"group_id"`
	ResourceID            string      `json:"resource_id"`
	Description           string      `json:"description"`
	FlowLogUploadsEnabled bool        `json:"flow_log_uploads_enabled"`
	Conditions            []Condition `json:"conditions"`
}

// CreatePolicyRequest is the request body for [PoliciesService.Create].
type CreatePolicyRequest struct {
	GroupID               string      `json:"group_id"`
	ResourceID            string      `json:"resource_id"`
	Description           string      `json:"description,omitempty"`
	FlowLogUploadsEnabled *bool       `json:"flow_log_uploads_enabled,omitempty"`
	Conditions            []Condition `json:"conditions,omitempty"`
}

// UpdatePolicyRequest is the request body for [PoliciesService.Update].
type UpdatePolicyRequest struct {
	GroupID               string      `json:"group_id,omitempty"`
	ResourceID            string      `json:"resource_id,omitempty"`
	Description           string      `json:"description,omitempty"`
	FlowLogUploadsEnabled *bool       `json:"flow_log_uploads_enabled,omitempty"`
	Conditions            []Condition `json:"conditions,omitempty"`
}

// PoliciesService manages Policies.
type PoliciesService struct {
	client *Client
}

// Get fetches a single Policy by ID.
func (s *PoliciesService) Get(ctx context.Context, id string) (*Policy, error) {
	var policy Policy
	if err := s.client.do(ctx, "GET", "policies/"+id, nil, nil, &policy); err != nil {
		return nil, err
	}
	return &policy, nil
}

// PolicyListOptions extends ListOptions with Policies-specific filters.
type PolicyListOptions struct {
	ListOptions
	// GroupID filters to Policies granting this Group.
	GroupID string
	// ResourceID filters to Policies granting access to this Resource.
	ResourceID string
}

// List returns a page of Policies. Pass nil for opts to use the API's
// default page size and no filters.
func (s *PoliciesService) List(ctx context.Context, opts *PolicyListOptions) (*Page[Policy], error) {
	if opts == nil {
		opts = &PolicyListOptions{}
	}
	q := filterQuery(opts.ListOptions,
		[2]string{"group_id", opts.GroupID},
		[2]string{"resource_id", opts.ResourceID},
	)
	return doList[Policy](ctx, s.client, "GET", "policies", q)
}

// Create creates a new Policy.
func (s *PoliciesService) Create(ctx context.Context, req *CreatePolicyRequest) (*Policy, error) {
	body, err := wrapBody("policy", req)
	if err != nil {
		return nil, err
	}
	var policy Policy
	if err := s.client.do(ctx, "POST", "policies", nil, body, &policy); err != nil {
		return nil, err
	}
	return &policy, nil
}

// Update updates a Policy.
func (s *PoliciesService) Update(ctx context.Context, id string, req *UpdatePolicyRequest) (*Policy, error) {
	body, err := wrapBody("policy", req)
	if err != nil {
		return nil, err
	}
	var policy Policy
	if err := s.client.do(ctx, "PUT", "policies/"+id, nil, body, &policy); err != nil {
		return nil, err
	}
	return &policy, nil
}

// Delete deletes a Policy.
func (s *PoliciesService) Delete(ctx context.Context, id string) error {
	return s.client.do(ctx, "DELETE", "policies/"+id, nil, nil, nil)
}

// Disable disables a Policy. Idempotent - disabling an already-disabled
// Policy is a no-op.
func (s *PoliciesService) Disable(ctx context.Context, id string) (*Policy, error) {
	var policy Policy
	if err := s.client.do(ctx, "POST", "policies/"+id+"/disable", nil, nil, &policy); err != nil {
		return nil, err
	}
	return &policy, nil
}

// Enable enables a Policy.
func (s *PoliciesService) Enable(ctx context.Context, id string) (*Policy, error) {
	var policy Policy
	if err := s.client.do(ctx, "POST", "policies/"+id+"/enable", nil, nil, &policy); err != nil {
		return nil, err
	}
	return &policy, nil
}
