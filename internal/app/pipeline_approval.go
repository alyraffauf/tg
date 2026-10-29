package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/alyraffauf/tg/spindle"
)

// ApprovePipeline releases the workflows waiting for approval in a pipeline.
func (s *Service) ApprovePipeline(ctx context.Context, target Target, pipelineID string) (*PipelineApproveResult, error) {
	if strings.TrimSpace(pipelineID) == "" {
		return nil, fmt.Errorf("pipeline ID is required")
	}
	client, repoDID, err := s.authenticatedPipelineClient(ctx, target, "org.tangled.ci.approvePipeline")
	if err != nil {
		return nil, err
	}
	if err := client.ApprovePipeline(ctx, spindle.ApprovePipelineInput{Repo: repoDID, Pipeline: pipelineID}); err != nil {
		return nil, err
	}
	return &PipelineApproveResult{Pipeline: pipelineID, ApprovalRequested: true}, nil
}

// RetryPipeline creates a new pipeline from the original run's stored specification.
func (s *Service) RetryPipeline(ctx context.Context, target Target, pipelineID string, workflows []string) (*PipelineRetryResult, error) {
	if strings.TrimSpace(pipelineID) == "" {
		return nil, fmt.Errorf("pipeline ID is required")
	}
	client, repoDID, err := s.authenticatedPipelineClient(ctx, target, "org.tangled.ci.retryPipeline")
	if err != nil {
		return nil, err
	}
	response, err := client.RetryPipeline(ctx, spindle.RetryPipelineInput{Repo: repoDID, Pipeline: pipelineID, Workflows: workflows})
	if err != nil {
		return nil, err
	}
	return &PipelineRetryResult{Pipeline: response.Pipeline, OriginalPipeline: pipelineID, Workflows: workflows}, nil
}

func (s *Service) authenticatedPipelineClient(ctx context.Context, target Target, method string) (pipelineClient, string, error) {
	host, repoDID, err := s.pipelineTarget(ctx, target)
	if err != nil {
		return nil, "", err
	}
	pds, _, err := s.authenticatedPDS(ctx)
	if err != nil {
		return nil, "", err
	}
	token, err := s.spindleToken(ctx, target, pds, host, repoDID, method)
	if err != nil {
		return nil, "", err
	}
	client, err := s.spindle.NewWithToken(host, token)
	if err != nil {
		return nil, "", fmt.Errorf("connect to pipeline spindle: %w", err)
	}
	return client, repoDID, nil
}
