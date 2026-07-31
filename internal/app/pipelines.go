package app

import (
	"context"
	"fmt"

	"github.com/alyraffauf/tg/spindle"
)

const maxPipelinePages = 1000

// ListPipelines returns every pipeline configured for a repository.
func (s *Service) ListPipelines(ctx context.Context, target Target) ([]Pipeline, error) {
	client, repoDID, err := s.pipelineClient(ctx, target)
	if err != nil {
		return nil, err
	}
	return listPipelinePages(ctx, client, repoDID)
}

// CancelPipeline cancels every workflow in a pipeline, or only the selected workflows.
func (s *Service) CancelPipeline(ctx context.Context, target Target, pipelineID string, workflows []string) (*PipelineCancelResult, error) {
	spindleHost, repoDID, err := s.pipelineTarget(ctx, target)
	if err != nil {
		return nil, err
	}
	client, err := s.spindle.New(spindleHost)
	if err != nil {
		return nil, fmt.Errorf("connect to pipeline spindle: %w", err)
	}
	pipeline, err := client.GetPipeline(ctx, pipelineID)
	if err != nil {
		return nil, err
	}
	cancellableWorkflows := selectCancellableWorkflows(pipeline.Workflows, workflows)
	if len(cancellableWorkflows) == 0 {
		return &PipelineCancelResult{Pipeline: pipelineID}, nil
	}

	pds, _, err := s.authenticatedPDS(ctx)
	if err != nil {
		return nil, err
	}
	audience, err := spindle.ServiceDID(spindleHost)
	if err != nil {
		return nil, err
	}
	token, err := pds.GetServiceAuth(ctx, audience, "sh.tangled.ci.cancelPipeline")
	if err != nil {
		return nil, fmt.Errorf("mint pipeline cancel token: %w", err)
	}
	authenticatedClient, err := s.spindle.NewWithToken(spindleHost, token)
	if err != nil {
		return nil, fmt.Errorf("connect to pipeline spindle: %w", err)
	}
	workflowsForRequest := cancellableWorkflows
	if len(workflows) == 0 {
		workflowsForRequest = nil
	}
	if err := authenticatedClient.CancelPipeline(ctx, spindle.CancelPipelineInput{
		Pipeline: pipelineID, Repo: repoDID, Workflows: workflowsForRequest,
	}); err != nil {
		return nil, err
	}
	return &PipelineCancelResult{Pipeline: pipelineID, Workflows: cancellableWorkflows, CancellationRequested: true}, nil
}

func selectCancellableWorkflows(workflows []spindle.Workflow, selected []string) []string {
	selectedWorkflows := make(map[string]bool, len(selected))
	for _, workflow := range selected {
		selectedWorkflows[workflow] = true
	}

	cancellable := make([]string, 0, len(workflows))
	for _, workflow := range workflows {
		if workflow.Status != "pending" && workflow.Status != "running" {
			continue
		}
		if len(selectedWorkflows) == 0 || selectedWorkflows[workflow.Name] {
			cancellable = append(cancellable, workflow.Name)
		}
	}
	return cancellable
}

// PipelineStatus returns the most recent pipeline for a repository.
func (s *Service) PipelineStatus(ctx context.Context, target Target) (*PipelineStatusResult, error) {
	client, repoDID, err := s.pipelineClient(ctx, target)
	if err != nil {
		return nil, err
	}
	response, err := client.QueryLatestPipeline(ctx, repoDID)
	if err != nil {
		return nil, err
	}
	pipelines := pipelineItems(response.Pipelines)
	if len(pipelines) == 0 {
		return nil, fmt.Errorf("no pipelines found for repository %q", target.String())
	}
	pipeline := &pipelines[0]
	return &PipelineStatusResult{Commit: pipeline.Commit, Pipeline: pipeline, HasFailures: pipelineHasFailures(*pipeline)}, nil
}

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

// ViewPipeline finds a pipeline by its spindle-local ID.
func (s *Service) ViewPipeline(ctx context.Context, target Target, pipelineID string) (*Pipeline, error) {
	pipelines, err := s.ListPipelines(ctx, target)
	if err != nil {
		return nil, err
	}
	pipeline, err := findPipeline(pipelines, pipelineID)
	if err != nil {
		return nil, fmt.Errorf("pipeline %q not found for repository %q", pipelineID, target.String())
	}
	return pipeline, nil
}

func findPipeline(pipelines []Pipeline, pipelineID string) (*Pipeline, error) {
	for index := range pipelines {
		if pipelines[index].ID == pipelineID {
			return &pipelines[index], nil
		}
	}
	return nil, fmt.Errorf("pipeline %q not found", pipelineID)
}

func pipelineHasFailures(pipeline Pipeline) bool {
	for _, workflow := range pipeline.Workflows {
		if workflow.Status == "failed" || workflow.Status == "timeout" {
			return true
		}
	}
	return false
}

func listPipelinePages(ctx context.Context, client pipelineClient, repoDID string) ([]Pipeline, error) {
	var pipelines []Pipeline
	cursor := ""
	for page := 0; page < maxPipelinePages; page++ {
		response, err := client.QueryPipelines(ctx, repoDID, cursor)
		if err != nil {
			return nil, err
		}
		pipelines = append(pipelines, pipelineItems(response.Pipelines)...)
		if response.Cursor == "" {
			return pipelines, nil
		}
		cursor = response.Cursor
	}
	return nil, fmt.Errorf("exceeded %d pipeline pages without reaching the end of the list", maxPipelinePages)
}

func pipelineItems(pipelines []spindle.Pipeline) []Pipeline {
	items := make([]Pipeline, 0, len(pipelines))
	for _, pipeline := range pipelines {
		workflows := make([]PipelineWorkflow, 0, len(pipeline.Workflows))
		for _, workflow := range pipeline.Workflows {
			workflows = append(workflows, PipelineWorkflow{
				ID: workflow.ID, Name: workflow.Name, Status: workflow.Status, Error: workflow.Error,
				StartedAt: workflow.StartedAt, FinishedAt: workflow.FinishedAt,
			})
		}
		items = append(items, Pipeline{
			ID: pipeline.ID, Commit: pipeline.Commit, CreatedAt: pipeline.CreatedAt,
			Repo: pipeline.Repo, SourceRepo: pipeline.SourceRepo, Trigger: pipeline.Trigger, Workflows: workflows,
		})
	}
	return items
}
