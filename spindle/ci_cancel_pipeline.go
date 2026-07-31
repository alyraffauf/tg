package spindle

import (
	"context"
	"fmt"
)

func (c *Client) CancelPipeline(ctx context.Context, input CancelPipelineInput) error {
	if err := c.Post(ctx, nsidCancelPipeline, input, nil); err != nil {
		return fmt.Errorf("cancel pipeline %q: %w", input.Pipeline, err)
	}
	return nil
}
