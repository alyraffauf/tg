package app

import (
	"context"
	"fmt"

	"github.com/alyraffauf/tg/spindle"
)

const maxPipelinePages = 100

func (s *Service) pipelineClient(ctx context.Context, target Target) (pipelineClient, string, error) {
	spindleHost, repoDID, err := s.pipelineTarget(ctx, target)
	if err != nil {
		return nil, "", err
	}
	client, err := s.spindle.New(spindleHost)
	if err != nil {
		return nil, "", fmt.Errorf("connect to pipeline spindle: %w", err)
	}
	return client, repoDID, nil
}

func fetchOwnedPipeline(ctx context.Context, client pipelineClient, target Target, repoDID, pipelineID string) (*spindle.Pipeline, error) {
	pipeline, err := client.GetPipeline(ctx, pipelineID)
	if err != nil {
		return nil, err
	}
	if pipeline == nil {
		return nil, fmt.Errorf("pipeline %q returned an empty response", pipelineID)
	}
	if pipeline.Repo == "" || pipeline.Repo != repoDID {
		return nil, fmt.Errorf("pipeline %q does not belong to repository %q", pipelineID, target.String())
	}
	return pipeline, nil
}

func (s *Service) pipelineTarget(ctx context.Context, target Target) (string, string, error) {
	repo, err := s.resolveRepo(ctx, target)
	if err != nil {
		return "", "", err
	}
	spindleHost := stringValue(repo.Value.Spindle)
	if spindleHost == "" {
		return "", "", fmt.Errorf("pipelines are not configured for repository %q", target.String())
	}
	repoDID := stringValue(repo.Value.RepoDid)
	if repoDID == "" {
		return "", "", fmt.Errorf("repository %q has no repository DID", target.String())
	}
	return spindleHost, repoDID, nil
}

func pipelineItems(pipelines []spindle.Pipeline) []Pipeline {
	items := make([]Pipeline, 0, len(pipelines))
	for _, pipeline := range pipelines {
		items = append(items, pipelineItem(pipeline))
	}
	return items
}

func pipelineItem(pipeline spindle.Pipeline) Pipeline {
	workflows := make([]PipelineWorkflow, 0, len(pipeline.Workflows))
	for _, workflow := range pipeline.Workflows {
		workflows = append(workflows, PipelineWorkflow{
			ID: workflow.ID, Name: workflow.Name, Status: workflow.Status, Error: workflow.Error,
			StartedAt: workflow.StartedAt, FinishedAt: workflow.FinishedAt,
		})
	}
	return Pipeline{
		ID: pipeline.ID, Commit: pipeline.Commit, CreatedAt: pipeline.CreatedAt,
		Repo: pipeline.Repo, SourceRepo: pipeline.SourceRepo, Trigger: pipeline.Trigger, Workflows: workflows,
	}
}
