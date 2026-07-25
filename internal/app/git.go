package app

import (
	"context"
	"fmt"

	"github.com/alyraffauf/tg/internal/gitutil"
)

// CloneRepoInput configures a repository clone.
type CloneRepoInput struct {
	KnotHost    string
	SSHPort     int
	Handle      string
	Repo        string
	Destination string
}

// CloneRepo clones a Tangled repository into Destination.
func (s *Service) CloneRepo(ctx context.Context, in CloneRepoInput) (*RepoCloneResult, error) {
	if in.SSHPort == 0 {
		in.SSHPort = 22
	}
	if in.SSHPort < 1 || in.SSHPort > 65535 {
		return nil, fmt.Errorf("SSH port must be between 1 and 65535")
	}
	if err := s.git.CloneRepo(ctx, gitutil.CloneRepoParams{
		KnotHost: in.KnotHost,
		SSHPort:  in.SSHPort,
		Handle:   in.Handle,
		Repo:     in.Repo,
		RepoDir:  in.Destination,
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
	localRepo, err := s.git.DetectRepoFromCWD(ctx)
	if err != nil {
		return nil, fmt.Errorf("detect local repository: %w", err)
	}
	localTarget := Target{Handle: localRepo.Handle, Repo: localRepo.Repo}
	localRecord, err := s.resolveRepo(ctx, localTarget)
	if err != nil {
		return nil, err
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
	if stringValue(targetRecord.Value.RepoDid) != stringValue(localRecord.Value.RepoDid) {
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
