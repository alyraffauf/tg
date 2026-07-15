package knot

import (
	"context"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

// DeleteRepoInput is the argument to sh.tangled.repo.delete.
type DeleteRepoInput struct {
	DID  string `json:"did"`
	Name string `json:"name"`
	Rkey string `json:"rkey"`
}

// DeleteRepo removes a repository from the knot.
func (c *Client) DeleteRepo(ctx context.Context, input DeleteRepoInput) error {
	if err := c.Post(ctx, syntax.NSID("sh.tangled.repo.delete"), input, nil); err != nil {
		return fmt.Errorf("delete repo on knot: %w", err)
	}
	return nil
}
