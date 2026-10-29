package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/alyraffauf/tg/spindle"
)

// TriggerPipeline starts a manual pipeline for revision. A full commit SHA can be
// used without a local Git checkout; other revisions are resolved locally.
func (s *Service) TriggerPipeline(ctx context.Context, target Target, revision string, workflows []string) (*PipelineTriggerResult, error) {
	commit, ref, err := s.resolvePipelineRevision(ctx, revision)
	if err != nil {
		return nil, err
	}
	spindleHost, repoDID, err := s.pipelineTarget(ctx, target)
	if err != nil {
		return nil, err
	}
	pds, _, err := s.authenticatedPDS(ctx)
	if err != nil {
		return nil, err
	}
	token, err := s.spindleToken(ctx, target, pds, spindleHost, repoDID, "org.tangled.event.notifyEvent")
	if err != nil {
		return nil, err
	}
	client, err := s.spindle.NewWithToken(spindleHost, token)
	if err != nil {
		return nil, fmt.Errorf("connect to pipeline spindle: %w", err)
	}
	response, err := client.TriggerPipeline(ctx, spindle.TriggerPipelineInput{
		Repo:  repoDID,
		Event: spindle.ManualTrigger{LexiconTypeID: "org.tangled.ci.trigger.manual", Repo: repoDID, SHA: commit, Ref: ref, Workflows: workflows},
	})
	if err != nil {
		return nil, err
	}
	result := &PipelineTriggerResult{Pipelines: response.Pipelines, Message: response.Message, Commit: commit, Workflows: workflows}
	if len(response.Pipelines) > 0 {
		// Preserve the existing JSON field for callers expecting a single pipeline.
		result.Pipeline = response.Pipelines[0]
	}
	return result, nil
}

func (s *Service) resolvePipelineRevision(ctx context.Context, revision string) (commit, ref string, err error) {
	if isFullCommitSHA(revision) {
		return revision, "", nil
	}
	commit, err = s.git.ResolveCommit(ctx, "", revision)
	if err != nil {
		return "", "", err
	}
	if revision == "HEAD" {
		if branch, branchErr := s.git.CurrentBranch(ctx, ""); branchErr == nil {
			return commit, "refs/heads/" + branch, nil
		}
		return commit, "", nil
	}
	return commit, revision, nil
}

func isFullCommitSHA(revision string) bool {
	if len(revision) != 40 && len(revision) != 64 {
		return false
	}
	return strings.Trim(revision, "0123456789abcdefABCDEF") == ""
}
