package firezone

import "context"

// GroupMember is an Actor's minimal representation as returned by
// [MembershipsService.List].
type GroupMember struct {
	ID   string    `json:"id"`
	Name string    `json:"name"`
	Type ActorType `json:"type"`
}

// MembershipsService manages a single Group's memberships. Obtain one
// via [GroupsService.Memberships].
type MembershipsService struct {
	client  *Client
	groupID string
}

// List returns a page of the Group's members.
func (s *MembershipsService) List(ctx context.Context, opts *ListOptions) (*Page[GroupMember], error) {
	return doList[GroupMember](ctx, s.client, "GET", "groups/"+s.groupID+"/memberships", listOptionsToQuery(opts))
}

// membershipEntry is the wire shape of one entry in a ReplaceAll request.
type membershipEntry struct {
	ActorID string `json:"actor_id"`
}

// membershipPatchBody is the wire shape of a Patch request body.
type membershipPatchBody struct {
	Add    []string `json:"add,omitempty"`
	Remove []string `json:"remove,omitempty"`
}

// membershipActorIDs is the wire shape of a memberships write response.
type membershipActorIDs struct {
	ActorIDs []string `json:"actor_ids"`
}

// ReplaceAll replaces the Group's entire membership list with
// actorIDs, returning the resulting member actor IDs. Unlike [Patch],
// this is a full replace: any actor not in actorIDs is removed.
//
// Prefer [Patch] when multiple independent callers manage membership on
// the same Group - ReplaceAll from more than one caller will overwrite
// each other's changes.
func (s *MembershipsService) ReplaceAll(ctx context.Context, actorIDs []string) ([]string, error) {
	entries := make([]membershipEntry, len(actorIDs))
	for i, id := range actorIDs {
		entries[i] = membershipEntry{ActorID: id}
	}
	body, err := wrapBody("memberships", entries)
	if err != nil {
		return nil, err
	}
	var result membershipActorIDs
	if err := s.client.do(ctx, "PUT", "groups/"+s.groupID+"/memberships", nil, body, &result); err != nil {
		return nil, err
	}
	return result.ActorIDs, nil
}

// Patch adds and removes members from the Group without disturbing any
// other membership, returning the resulting member actor IDs. This is
// the safe choice when more than one caller manages membership on the
// same Group independently.
func (s *MembershipsService) Patch(ctx context.Context, add, remove []string) ([]string, error) {
	body, err := wrapBody("memberships", membershipPatchBody{Add: add, Remove: remove})
	if err != nil {
		return nil, err
	}
	var result membershipActorIDs
	if err := s.client.do(ctx, "PATCH", "groups/"+s.groupID+"/memberships", nil, body, &result); err != nil {
		return nil, err
	}
	return result.ActorIDs, nil
}
