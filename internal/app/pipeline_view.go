package app

import "context"

// ViewPipeline fetches a pipeline by its spindle-local ID.
func (s *Service) ViewPipeline(ctx context.Context, target Target, pipelineID string) (*Pipeline, error) {
	client, repoDID, err := s.pipelineClient(ctx, target)
	if err != nil {
		return nil, err
	}
	pipeline, err := fetchOwnedPipeline(ctx, client, target, repoDID, pipelineID)
	if err != nil {
		return nil, err
	}
	item := pipelineItem(*pipeline)
	return &item, nil
}
