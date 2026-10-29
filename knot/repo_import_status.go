package knot

import (
	"context"
	"fmt"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

type ImportStatus struct {
	Status string `json:"status,omitempty"`
	Reason string `json:"reason,omitempty"`
}

func (c *Client) GetImportStatus(ctx context.Context, repo string) (*ImportStatus, error) {
	var out ImportStatus
	if err := c.Get(ctx, syntax.NSID("org.tangled.repo.getImportStatus"), map[string]any{"repo": repo}, &out); err != nil {
		return nil, err
	}
	switch out.Status {
	case "", "fetching", "present", "partial", "failed":
	default:
		return nil, fmt.Errorf("unknown import status %q", out.Status)
	}
	return &out, nil
}
