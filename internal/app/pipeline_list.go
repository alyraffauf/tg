package app

import (
	"context"
	"fmt"
)

// ListPipelines returns every pipeline configured for a repository.
func (s *Service) ListPipelines(ctx context.Context, target Target) ([]Pipeline, error) {
	client, repoDID, err := s.pipelineClient(ctx, target)
	if err != nil {
		return nil, err
	}
	return listPipelinePages(ctx, client, repoDID)
}

func listPipelinePages(ctx context.Context, client pipelineClient, repoDID string) ([]Pipeline, error) {
	var pipelines []Pipeline
	cursor := ""
	seenCursors := make(map[string]bool)
	for page := 0; page < maxPipelinePages; page++ {
		response, err := client.QueryPipelines(ctx, repoDID, cursor)
		if err != nil {
			return nil, err
		}
		pipelines = append(pipelines, pipelineItems(response.Pipelines)...)
		if response.Cursor == "" {
			return pipelines, nil
		}
		if seenCursors[response.Cursor] {
			return nil, fmt.Errorf("pipeline pagination repeated cursor %q", response.Cursor)
		}
		seenCursors[response.Cursor] = true
		cursor = response.Cursor
	}
	return nil, fmt.Errorf("exceeded %d pipeline pages without reaching the end of the list", maxPipelinePages)
}
