package app

import (
	"context"
	"fmt"

	"github.com/alyraffauf/tg/internal/gitutil"
)

// CheckoutPullInput configures reconstructing a pull request in a local
// repository. Target may be empty, in which case the current repository is
// used.
type CheckoutPullInput struct {
	RepoDir string
	Rkey    string
	Target  *Target
	Branch  string
	Force   bool
}

// CheckoutPull downloads and applies the latest pull request patch.
func (s *Service) CheckoutPull(ctx context.Context, in CheckoutPullInput) (*PRCheckoutResult, error) {
	localTarget, localRecord, err := s.repoFromCWD(ctx)
	if err != nil {
		return nil, fmt.Errorf("detect local repository: %w", err)
	}

	target := localTarget
	if in.Target != nil {
		target = *in.Target
	}
	targetRecord := localRecord
	if target != localTarget {
		targetRecord, err = s.resolveRepo(ctx, target)
		if err != nil {
			return nil, err
		}
	}
	localRepoDID := stringValue(localRecord.Value.RepoDid)
	if localRepoDID == "" {
		return nil, fmt.Errorf("repository %q has no repository DID", localTarget.String())
	}
	targetRepoDID := stringValue(targetRecord.Value.RepoDid)
	if targetRepoDID == "" {
		return nil, fmt.Errorf("repository %q has no repository DID", target.String())
	}
	if targetRepoDID != localRepoDID {
		return nil, fmt.Errorf("pull request target %s does not match the current repository", target)
	}

	patch, err := s.PullPatch(ctx, target, in.Rkey)
	if err != nil {
		return nil, err
	}
	if patch.TargetBranch == "" {
		return nil, fmt.Errorf("pull request %q has no target branch", in.Rkey)
	}

	branch := in.Branch
	if branch == "" {
		branch = "pr-" + in.Rkey
	}
	if err := s.git.CheckoutPatch(ctx, gitutil.CheckoutPatchParams{
		RepoDir:      in.RepoDir,
		Branch:       branch,
		TargetBranch: patch.TargetBranch,
		Patch:        patch.Patch,
		Force:        in.Force,
	}); err != nil {
		return nil, err
	}
	return &PRCheckoutResult{Rkey: in.Rkey, Branch: branch}, nil
}
