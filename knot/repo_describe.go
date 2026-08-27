package knot

import (
	"context"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

// RepoDescription identifies the current ATProto record for a repository DID.
type RepoDescription struct {
	RepoDID  string `json:"repoDid"`
	OwnerDID string `json:"ownerDid"`
	RKey     string `json:"rkey"`
}

// DescribeRepo returns the repository metadata reported by the Knot.
func (c *Client) DescribeRepo(ctx context.Context, repoDID string) (*RepoDescription, error) {
	var description RepoDescription
	if err := c.Get(ctx, syntax.NSID("sh.tangled.repo.describeRepo"), map[string]any{"repoDid": repoDID}, &description); err != nil {
		return nil, fmt.Errorf("describe repository %q: %w", repoDID, err)
	}
	return &description, nil
}
