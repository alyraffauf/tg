package spindle

import (
	"context"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

func (c *Client) GetPipeline(ctx context.Context, pipelineID string) (*Pipeline, error) {
	var pipeline Pipeline
	if err := c.Get(ctx, syntax.NSID("sh.tangled.ci.getPipeline"), map[string]any{"pipeline": pipelineID}, &pipeline); err != nil {
		return nil, fmt.Errorf("get pipeline %q: %w", pipelineID, err)
	}
	return &pipeline, nil
}
