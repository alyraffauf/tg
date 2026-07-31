package knot

import (
	"context"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

// DefaultBranch describes the repository's default branch and its latest commit.
type DefaultBranch struct {
	Name string `json:"name"`
	Hash string `json:"hash"`
}

// GetDefaultBranch returns the default branch for a repository DID and name.
func (c *Client) GetDefaultBranch(ctx context.Context, repo string) (*DefaultBranch, error) {
	var branch DefaultBranch
	if err := c.Get(ctx, syntax.NSID("sh.tangled.repo.getDefaultBranch"), map[string]any{"repo": repo}, &branch); err != nil {
		return nil, fmt.Errorf("get default branch for %q: %w", repo, err)
	}
	return &branch, nil
}
