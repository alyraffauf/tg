package spindle

import (
	"context"
	"fmt"
)

func (c *Client) TriggerPipeline(ctx context.Context, input TriggerPipelineInput) (*TriggerPipelineOutput, error) {
	var wire struct {
		CI *TriggerPipelineOutput `json:"ci"`
		// Older Spindles return one pipeline and its message at the top level.
		Pipeline string `json:"pipeline"`
		Message  string `json:"message"`
	}
	if err := c.Post(ctx, nsidNotifyEvent, input, &wire); err != nil {
		return nil, fmt.Errorf("trigger pipeline: %w", err)
	}
	if wire.CI != nil {
		return wire.CI, nil
	}
	output := &TriggerPipelineOutput{Message: wire.Message}
	if wire.Pipeline != "" {
		output.Pipelines = []string{wire.Pipeline}
	}
	return output, nil
}
