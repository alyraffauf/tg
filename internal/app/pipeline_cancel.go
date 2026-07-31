package app

import (
	"context"
	"fmt"

	"github.com/alyraffauf/tg/spindle"
)

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
