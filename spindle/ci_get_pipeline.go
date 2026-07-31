package spindle

import (
	"context"
	"fmt"
)

func (c *Client) GetPipeline(ctx context.Context, pipelineID string) (*Pipeline, error) {
	var pipeline Pipeline
	if err := c.Get(ctx, nsidGetPipeline, map[string]any{"pipeline": pipelineID}, &pipeline); err != nil {
		return nil, fmt.Errorf("get pipeline %q: %w", pipelineID, err)
	}
	return &pipeline, nil
}
