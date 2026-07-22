package app

import (
	"context"
	"fmt"

	"github.com/alyraffauf/tg/internal/gitutil"
)

// CloneRepoInput configures a repository clone.
type CloneRepoInput struct {
	Handle      string
	Repo        string
	Destination string
}

// CloneRepo clones a Tangled repository into Destination.
func (s *Service) CloneRepo(ctx context.Context, in CloneRepoInput) (*RepoCloneResult, error) {
	if err := s.Git.CloneRepo(ctx, gitutil.CloneRepoParams{
		Handle:  in.Handle,
		Repo:    in.Repo,
		RepoDir: in.Destination,
	}); err != nil {
		return nil, err
	}
	return &RepoCloneResult{
		Handle:      in.Handle,
		Repo:        in.Repo,
		Destination: in.Destination,
	}, nil
}

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
	localRepo, err := s.Git.DetectRepoFromCWD(ctx)
	if err != nil {
		return nil, fmt.Errorf("detect local repository: %w", err)
	}
	localTarget := Target{Handle: localRepo.Handle, Repo: localRepo.Repo}
	localRecord, err := s.ResolveRepo(ctx, localTarget)
	if err != nil {
		return nil, err
	}

	target := localTarget
	if in.Target != nil {
		target = *in.Target
	}
	targetRecord := localRecord
	if target != localTarget {
		targetRecord, err = s.ResolveRepo(ctx, target)
		if err != nil {
			return nil, err
		}
	}
	if targetRecord.Value.RepoDid != localRecord.Value.RepoDid {
		return nil, fmt.Errorf("pull request target %s does not match the current repository", target)
	}

	patch, err := s.PullPatch(ctx, target, in.Rkey)
	if err != nil {
		return nil, err
	}
	if patch.Record.Target.Branch == "" {
		return nil, fmt.Errorf("pull request %q has no target branch", in.Rkey)
	}

	branch := in.Branch
	if branch == "" {
		branch = "pr-" + in.Rkey
	}
	if err := s.Git.CheckoutPatch(ctx, gitutil.CheckoutPatchParams{
		RepoDir:      in.RepoDir,
		Branch:       branch,
		TargetBranch: patch.Record.Target.Branch,
		Patch:        patch.Patch,
		Force:        in.Force,
	}); err != nil {
		return nil, err
	}
	return &PRCheckoutResult{Rkey: in.Rkey, Branch: branch}, nil
}
