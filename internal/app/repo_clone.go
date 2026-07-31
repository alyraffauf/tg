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
