package spindle

import (
	"context"
	"fmt"
)

func (c *Client) ApprovePipeline(ctx context.Context, input ApprovePipelineInput) error {
	if err := c.Post(ctx, nsidApprovePipeline, input, nil); err != nil {
		return fmt.Errorf("approve pipeline: %w", err)
	}
	return nil
}

func (c *Client) RetryPipeline(ctx context.Context, input RetryPipelineInput) (*RetryPipelineOutput, error) {
	var output RetryPipelineOutput
	if err := c.Post(ctx, nsidRetryPipeline, input, &output); err != nil {
		return nil, fmt.Errorf("retry pipeline: %w", err)
	}
	if output.Pipeline == "" {
		return nil, fmt.Errorf("retry accepted without a pipeline ID; inspect the repository's pipelines before retrying")
	}
	return &output, nil
}
