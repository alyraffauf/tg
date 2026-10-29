package spindle

import (
	"context"
	"fmt"
)

func (c *Client) GetPipeline(ctx context.Context, pipelineID string, repo ...string) (*Pipeline, error) {
	params := map[string]any{"pipeline": pipelineID}
	if len(repo) > 0 {
		params["repo"] = repo[0]
	}
	var pipeline Pipeline
	if err := c.Get(ctx, nsidGetPipeline, params, &pipeline); err != nil {
		return nil, fmt.Errorf("get pipeline %q: %w", pipelineID, err)
	}
	return &pipeline, nil
}
