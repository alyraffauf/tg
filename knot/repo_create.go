package knot

import (
	"context"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

type CreateRepoInput struct {
	Name          string `json:"name"`
	Source        string `json:"-"`
	Description   string `json:"description,omitempty"`
	DefaultBranch string `json:"defaultBranch,omitempty"`
}

func (c *Client) CreateRepo(ctx context.Context, in CreateRepoInput) (string, error) {
	input := map[string]any{"name": in.Name}
	if in.Source != "" {
		input["source"] = map[string]string{"$type": "org.tangled.repo.defs#sourceTangled", "did": in.Source}
	}
	if in.Description != "" {
		input["description"] = in.Description
	}
	if in.DefaultBranch != "" {
		input["defaultBranch"] = in.DefaultBranch
	}
	var out struct {
		DID string `json:"did"`
	}
	if err := c.Post(ctx, syntax.NSID("org.tangled.repo.createRepo"), input, &out); err != nil {
		return "", err
	}
	if _, err := syntax.ParseDID(out.DID); err != nil {
		return "", fmt.Errorf("knot returned no valid repository DID: %w", err)
	}
	return out.DID, nil
}
