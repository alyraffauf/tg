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
	Protocol    string
	Handle      string
	Repo        string
	Destination string
}

// CloneRepo clones a Tangled repository into Destination.
func (s *Service) CloneRepo(ctx context.Context, in CloneRepoInput) (*RepoCloneResult, error) {
	cloneProtocol, err := validateCloneProtocol(in.Protocol)
	if err != nil {
		return nil, err
	}
	in.Protocol = cloneProtocol
	if in.Protocol == "ssh" {
		if in.SSHPort == 0 {
			in.SSHPort = 22
		}
		if in.SSHPort < 1 || in.SSHPort > 65535 {
			return nil, fmt.Errorf("SSH port must be between 1 and 65535")
		}
	}
	return s.cloneRepo(ctx, in)
}

func (s *Service) cloneRepo(ctx context.Context, in CloneRepoInput) (*RepoCloneResult, error) {
	if in.Protocol == "https" && in.KnotHost == "" {
		repo, err := s.resolveRepo(ctx, Target{Handle: in.Handle, Repo: in.Repo})
		if err != nil {
			return nil, fmt.Errorf("resolve repository knot: %w", err)
		}
		knotHost, err := parseKnotHostname(repo.Value.Knot)
		if err != nil {
			return nil, err
		}
		in.KnotHost = knotHost
	}
	if err := s.git.CloneRepo(ctx, gitutil.CloneRepoParams{
		KnotHost: in.KnotHost,
		SSHPort:  in.SSHPort,
		Protocol: in.Protocol,
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

func validateCloneProtocol(protocol string) (string, error) {
	if protocol == "" {
		return "ssh", nil
	}
	if protocol != "ssh" && protocol != "https" {
		return "", fmt.Errorf("clone protocol must be \"ssh\" or \"https\", got %q", protocol)
	}
	return protocol, nil
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
