package knot

import (
	"context"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

// DeleteRepoInput is the argument to org.tangled.repo.deleteRepo.
type DeleteRepoInput struct {
	Repo string `json:"repo"` // repository DID
}

// DeleteRepo removes a repository from the knot.
func (c *Client) DeleteRepo(ctx context.Context, input DeleteRepoInput) error {
	if err := c.Post(ctx, syntax.NSID("org.tangled.repo.deleteRepo"), input, nil); err != nil {
		return fmt.Errorf("delete repo on knot: %w", err)
	}
	return nil
}
