package app

import (
	"context"
	"fmt"
	"strings"
)

// PipelineStatus returns the most recent pipeline for a repository.
func (s *Service) PipelineStatus(ctx context.Context, target Target) (*PipelineStatusResult, error) {
	repo, err := s.resolveRepo(ctx, target)
	if err != nil {
		return nil, err
	}
	spindleHost := stringValue(repo.Value.Spindle)
	repoDID := stringValue(repo.Value.RepoDid)
	if spindleHost == "" {
		return nil, fmt.Errorf("pipelines are not configured for repository %q", target.String())
	}
	if repoDID == "" {
		return nil, fmt.Errorf("repository %q has no repository DID", target.String())
	}
	branchName := stringValue(repo.Value.Name)
	if branchName == "" {
		branchName = target.Repo
	}
	defaultBranch, err := s.knot.NewPublic(repo.Value.Knot).GetDefaultBranch(ctx, extractDID(repo.URI)+"/"+branchName)
	if err != nil {
		return nil, err
	}
	client, err := s.spindle.New(spindleHost)
	if err != nil {
		return nil, fmt.Errorf("connect to pipeline spindle: %w", err)
	}
	response, err := client.QueryPipelines(ctx, repoDID, "")
	if err != nil {
		return nil, err
	}
	pipelines := pipelineItems(response.Pipelines)
	pipeline := latestDefaultBranchPipeline(pipelines, defaultBranch.Name, defaultBranch.Hash)
	if pipeline == nil {
		return nil, fmt.Errorf("no pipeline found for the latest %s commit on the default branch", target.String())
	}
	return &PipelineStatusResult{Commit: pipeline.Commit, Pipeline: pipeline, HasFailures: pipelineHasFailures(*pipeline)}, nil
}

func latestDefaultBranchPipeline(pipelines []Pipeline, branchName, branchHash string) *Pipeline {
	for index := range pipelines {
		pipeline := &pipelines[index]
		if branchHash != "" && pipeline.Commit == branchHash {
			return pipeline
		}
		if branchHash == "" && pipelineTargetsBranch(*pipeline, branchName) {
			return pipeline
		}
	}
	return nil
}

func pipelineTargetsBranch(pipeline Pipeline, branchName string) bool {
	triggerType, _ := pipeline.Trigger["$type"].(string)
	if triggerType == "sh.tangled.ci.trigger#pullRequest" {
		targetBranch, _ := pipeline.Trigger["targetBranch"].(string)
		return targetBranch == branchName
	}
	ref, _ := pipeline.Trigger["ref"].(string)
	return strings.TrimPrefix(ref, "refs/heads/") == branchName
}

func pipelineHasFailures(pipeline Pipeline) bool {
	for _, workflow := range pipeline.Workflows {
		if workflow.Status == "failed" || workflow.Status == "timeout" {
			return true
		}
	}
	return false
}
