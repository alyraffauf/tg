package app

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/alyraffauf/tg/internal/gitutil"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

// CloneRepoInput configures a repository clone.
type CloneRepoInput struct {
	KnotHost    string
	SSHPort     int
	Protocol    string
	Handle      string
	Repo        string
	RepoDID     string
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
	if in.RepoDID != "" && (in.Handle != "" || in.Repo != "") {
		return nil, errors.New("repository DID cannot be combined with a handle or repository name")
	}
	if in.RepoDID == "" && (in.Handle == "" || in.Repo == "") {
		return nil, errors.New("clone requires either a repository DID or both a handle and repository name")
	}
	resolved, err := s.resolveCloneRepo(ctx, in)
	if err != nil {
		return nil, err
	}
	return s.cloneResolvedRepo(ctx, resolved)
}

type resolvedCloneRepoInput struct {
	KnotHost    string
	SSHPort     int
	Protocol    string
	Handle      string
	Repo        string
	RepoDID     string
	Destination string
	Warnings    []string
}

func (s *Service) resolveCloneRepo(ctx context.Context, in CloneRepoInput) (resolvedCloneRepoInput, error) {
	resolved := resolvedCloneRepoInput{
		KnotHost: in.KnotHost, SSHPort: in.SSHPort, Protocol: in.Protocol,
		Handle: in.Handle, Repo: in.Repo, RepoDID: in.RepoDID, Destination: in.Destination,
	}
	var warnings []string
	if in.RepoDID != "" {
		target, _, err := s.resolveRepoDID(ctx, in.RepoDID, in.KnotHost)
		if err != nil {
			return resolvedCloneRepoInput{}, err
		}
		resolved.Handle = target.Handle
		resolved.Repo = target.Repo
	} else {
		repo, err := s.resolveRepo(ctx, Target{Handle: in.Handle, Repo: in.Repo})
		if err != nil {
			if in.Protocol != "ssh" {
				return resolvedCloneRepoInput{}, fmt.Errorf("resolve repository Knot: %w", err)
			}
			warnings = append(warnings, fmt.Sprintf("could not resolve repository DID. Using handle-based remote: %v", err))
		} else {
			resolved.RepoDID = stringValue(repo.Value.RepoDid)
			if resolved.RepoDID == "" {
				warnings = append(warnings, "repository record has no repository DID. Using handle-based remote")
			} else if _, err := syntax.ParseDID(resolved.RepoDID); err != nil {
				return resolvedCloneRepoInput{}, fmt.Errorf("invalid repository DID %q: %w", resolved.RepoDID, err)
			}
			if in.Protocol == "https" && resolved.RepoDID == "" {
				resolved.KnotHost, err = parseKnotHostname(repo.Value.Knot)
				if err != nil {
					return resolvedCloneRepoInput{}, err
				}
			}
		}
	}
	if resolved.Destination == "" {
		resolved.Destination = resolved.Repo
		if err := validateDefaultCloneDestination(resolved.Destination); err != nil {
			return resolvedCloneRepoInput{}, err
		}
	}
	resolved.Warnings = warnings
	return resolved, nil
}

func validateDefaultCloneDestination(destination string) error {
	unsafe := destination == "" ||
		filepath.IsAbs(destination) ||
		destination == "." ||
		destination == ".." ||
		strings.ContainsRune(destination, filepath.Separator) ||
		strings.ContainsRune(destination, '\x00') ||
		strings.HasPrefix(destination, "-")
	if unsafe {
		return fmt.Errorf("repository name %q cannot be used as the default clone directory; provide an explicit directory", destination)
	}
	return nil
}

func (s *Service) cloneResolvedRepo(ctx context.Context, in resolvedCloneRepoInput) (*RepoCloneResult, error) {
	if in.RepoDID != "" {
		if _, err := syntax.ParseDID(in.RepoDID); err != nil {
			return nil, fmt.Errorf("invalid repository DID %q: %w", in.RepoDID, err)
		}
	}
	if err := s.git.CloneRepo(ctx, gitutil.CloneRepoParams{
		KnotHost: in.KnotHost,
		SSHPort:  in.SSHPort,
		Protocol: in.Protocol,
		Handle:   in.Handle,
		Repo:     in.Repo,
		RepoDID:  in.RepoDID,
		RepoDir:  in.Destination,
	}); err != nil {
		return nil, err
	}
	return &RepoCloneResult{
		Handle:      in.Handle,
		Repo:        in.Repo,
		RepoDID:     in.RepoDID,
		Destination: in.Destination,
		Warnings:    in.Warnings,
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
