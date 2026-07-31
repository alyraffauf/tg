package spindle

import (
	"context"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

func (c *Client) CancelPipeline(ctx context.Context, input CancelPipelineInput) error {
	if err := c.Post(ctx, syntax.NSID("sh.tangled.ci.cancelPipeline"), input, nil); err != nil {
		return fmt.Errorf("cancel pipeline %q: %w", input.Pipeline, err)
	}
	return nil
}
