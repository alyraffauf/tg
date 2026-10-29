package app

import (
	"context"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

func (s *Service) ForkRepo(ctx context.Context, source Target, name string) (*RepoForkResult, error) {
	return s.ForkRepoOnKnot(ctx, source, name, "")
}
func (s *Service) ForkRepoOnKnot(ctx context.Context, source Target, name, host string) (*RepoForkResult, error) {
	if _, err := syntax.ParseRecordKey(name); err != nil {
		return nil, err
	}
	repo, err := s.resolveRepo(ctx, source)
	if err != nil {
		return nil, err
	}
	if host == "" {
		host = repo.Value.Knot
	} else {
		host, err = parseKnotHostname(host)
		if err != nil {
			return nil, err
		}
	}
	result, err := s.provisionRepo(ctx, provisionRepoInput{KnotHost: host, Name: name, Source: stringValue(repo.Value.RepoDid)})
	if err != nil {
		return nil, fmt.Errorf("fork repository: %w", err)
	}
	out := &RepoForkResult{Handle: result.Handle, Name: name, URI: result.URI, Knot: host, Warnings: result.Warnings}
	status, err := s.knot.NewPublic(host).GetImportStatus(ctx, result.RepoDID)
	if err != nil {
		out.Warnings = append(out.Warnings, fmt.Sprintf("repository created, but import status is unavailable: %v", err))
	} else {
		out.ImportStatus, out.ImportReason = status.Status, status.Reason
		if status.Status != "present" {
			out.Warnings = append(out.Warnings, fmt.Sprintf("repository created; import status %q: %s", status.Status, status.Reason))
		}
	}

	return out, nil
}
