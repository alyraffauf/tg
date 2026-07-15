package knot

import (
	"context"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

// CreateRepoInput is the argument to sh.tangled.repo.create.
type CreateRepoInput struct {
	Name          string `json:"name"`
	Rkey          string `json:"rkey"`
	Source        string `json:"source,omitempty"`
	DefaultBranch string `json:"defaultBranch,omitempty"`
}

// CreateRepo creates the repo via sh.tangled.repo.create, returning the minted
// repoDid. Use the repo name as the rkey (current schema).
func (c *Client) CreateRepo(ctx context.Context, input CreateRepoInput) (string, error) {
	var out struct {
		RepoDid *string `json:"repoDid,omitempty"`
	}
	if err := c.Post(ctx, syntax.NSID("sh.tangled.repo.create"), input, &out); err != nil {
		return "", fmt.Errorf("create repo on knot: %w", err)
	}
	if out.RepoDid == nil || *out.RepoDid == "" {
		return "", fmt.Errorf("knot did not return a repoDid")
	}
	return *out.RepoDid, nil
}
