package knot

import (
	"context"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

// SetDefaultBranchInput is the argument to org.tangled.git.setDefaultBranch.
type SetDefaultBranchInput struct {
	Repo          string `json:"repo"` // repository DID
	DefaultBranch string `json:"branch"`
}

// SetDefaultBranch repoints the default branch (bare repo HEAD) on the knot.
func (c *Client) SetDefaultBranch(ctx context.Context, input SetDefaultBranchInput) error {
	if err := c.Post(ctx, syntax.NSID("org.tangled.git.setDefaultBranch"), input, nil); err != nil {
		return fmt.Errorf("set default branch to %q: %w", input.DefaultBranch, err)
	}
	return nil
}
