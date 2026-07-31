package spindle

import (
	"context"
	"fmt"
)

func (c *Client) TriggerPipeline(ctx context.Context, input TriggerPipelineInput) (*TriggerPipelineOutput, error) {
	var output TriggerPipelineOutput
	if err := c.Post(ctx, nsidTriggerPipeline, input, &output); err != nil {
		return nil, fmt.Errorf("trigger pipeline: %w", err)
	}
	return &output, nil
}
