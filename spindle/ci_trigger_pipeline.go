package spindle

import (
	"context"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

func (c *Client) TriggerPipeline(ctx context.Context, input TriggerPipelineInput) (*TriggerPipelineOutput, error) {
	var output TriggerPipelineOutput
	if err := c.Post(ctx, syntax.NSID("sh.tangled.ci.triggerPipeline"), input, &output); err != nil {
		return nil, fmt.Errorf("trigger pipeline: %w", err)
	}
	return &output, nil
}
