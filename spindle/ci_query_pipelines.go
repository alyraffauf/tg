package spindle

import (
	"context"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

func (c *Client) QueryPipelines(ctx context.Context, repoDID, cursor string) (*QueryPipelinesOutput, error) {
	params := map[string]any{"repo": repoDID, "limit": 250}
	if cursor != "" {
		params["cursor"] = cursor
	}
	var output QueryPipelinesOutput
	if err := c.Get(ctx, syntax.NSID("sh.tangled.ci.queryPipelines"), params, &output); err != nil {
		return nil, fmt.Errorf("query pipelines for %q: %w", repoDID, err)
	}
	return &output, nil
}

func (c *Client) QueryLatestPipeline(ctx context.Context, repoDID string) (*QueryPipelinesOutput, error) {
	var output QueryPipelinesOutput
	if err := c.Get(ctx, syntax.NSID("sh.tangled.ci.queryPipelines"), map[string]any{"repo": repoDID, "limit": 1}, &output); err != nil {
		return nil, fmt.Errorf("query latest pipeline for %q: %w", repoDID, err)
	}
	return &output, nil
}
