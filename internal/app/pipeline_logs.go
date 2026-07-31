package app

import (
	"context"
	"fmt"

	"github.com/alyraffauf/tg/spindle"
)

// PipelineLogs streams log events from a pipeline. A non-empty workflows list
// filters to the named workflows.
func (s *Service) PipelineLogs(ctx context.Context, target Target, pipelineID string, workflows []string, onEvent func(PipelineLogEvent) error) error {
	spindleHost, _, err := s.pipelineTarget(ctx, target)
	if err != nil {
		return err
	}
	client, err := s.spindle.New(spindleHost)
	if err != nil {
		return fmt.Errorf("connect to pipeline spindle: %w", err)
	}
	return client.SubscribePipelineLogs(ctx, pipelineID, workflows, func(event spindle.PipelineLogEvent) error {
		return onEvent(PipelineLogEvent{
			Type:    event.Type,
			Control: convertControl(event.Control),
			Data:    convertData(event.Data),
		})
	})
}

func convertControl(c *spindle.PipelineLogControl) *PipelineLogControl {
	if c == nil {
		return nil
	}
	return &PipelineLogControl{
		Kind: c.Kind, Step: c.Step, Time: c.Time, Status: c.Status,
		Content: c.Content, Workflow: c.Workflow, Command: c.Command,
	}
}

func convertData(d *spindle.PipelineLogData) *PipelineLogData {
	if d == nil {
		return nil
	}
	return &PipelineLogData{
		Step: d.Step, Time: d.Time, Stream: d.Stream,
		Content: d.Content, Workflow: d.Workflow,
	}
}
