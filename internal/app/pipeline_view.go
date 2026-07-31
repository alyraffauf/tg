package app

import (
	"context"
	"fmt"
)

// ViewPipeline fetches a pipeline by its spindle-local ID.
func (s *Service) ViewPipeline(ctx context.Context, target Target, pipelineID string) (*Pipeline, error) {
	client, repoDID, err := s.pipelineClient(ctx, target)
	if err != nil {
		return nil, err
	}
	pipeline, err := client.GetPipeline(ctx, pipelineID)
	if err != nil {
		return nil, err
	}
	if pipeline.Repo != repoDID {
		return nil, fmt.Errorf("pipeline %q does not belong to repository %q", pipelineID, target.String())
	}
	item := pipelineItem(*pipeline)
	return &item, nil
}
