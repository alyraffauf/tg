package knot

import (
	"context"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

// MergeInput is the argument to sh.tangled.repo.merge.
type MergeInput struct {
	Repo string `json:"repo"`
	Pull string `json:"pull"`
}

// Merge applies a pull request on the knot.
func (c *Client) Merge(ctx context.Context, input MergeInput) error {
	if err := c.Post(ctx, syntax.NSID("sh.tangled.repo.merge"), input, nil); err != nil {
		return fmt.Errorf("merge pull request: %w", err)
	}
	return nil
}
