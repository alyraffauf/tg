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
	audience, err := spindle.ServiceDID(spindleHost)
	if err != nil {
		return nil, err
	}
	token, err := pds.GetServiceAuth(ctx, audience, "sh.tangled.ci.triggerPipeline")
	if err != nil {
		return nil, fmt.Errorf("mint pipeline trigger token: %w", err)
	}
	client, err := s.spindle.NewWithToken(spindleHost, token)
	if err != nil {
		return nil, fmt.Errorf("connect to pipeline spindle: %w", err)
	}
	response, err := client.TriggerPipeline(ctx, spindle.TriggerPipelineInput{
		Repo: repoDID, Workflows: workflows,
		Trigger: spindle.ManualTrigger{LexiconTypeID: "sh.tangled.ci.trigger#manual", SHA: commit, Ref: ref},
	})
	if err != nil {
		return nil, err
	}
	return &PipelineTriggerResult{Pipeline: extractRKey(response.Pipeline), Commit: commit, Workflows: workflows}, nil
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
	if len(revision) != 40 {
		return false
	}
	return strings.Trim(revision, "0123456789abcdefABCDEF") == ""
}
