package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/alyraffauf/tg/tangled"
	"github.com/bluesky-social/indigo/atproto/atclient"
)

// Target identifies a repository by owner handle (or DID) and repo name.
type Target struct {
	Handle string
	Repo   string
}

func (t Target) String() string { return t.Handle + "/" + t.Repo }

// ParseTarget parses a "handle/repo" (or "did:plc:.../repo") argument.
func ParseTarget(arg string) (Target, error) {
	parts := strings.SplitN(arg, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" || strings.Contains(parts[1], "/") {
		return Target{}, fmt.Errorf("expected handle/repo, got %q", arg)
	}
	return Target{Handle: parts[0], Repo: parts[1]}, nil
}

// TargetFromCWD detects the target using the service's Git client.
func (s *Service) TargetFromCWD(ctx context.Context) (Target, error) {
	return targetFromCWD(ctx, s.git)
}

func targetFromCWD(ctx context.Context, git gitClient) (Target, error) {
	rc, err := git.DetectRepoFromCWD(ctx)
	if err != nil {
		return Target{}, fmt.Errorf("detect repo from current directory: %w", err)
	}
	return Target{Handle: rc.Handle, Repo: rc.Repo}, nil
}

// resolveRepo finds a repository record even when its rkey does not match
// the repository name.
func (s *Service) resolveRepo(ctx context.Context, t Target) (*tangled.Repo, error) {
	ident, err := s.resolver.ResolveHandle(ctx, t.Handle)
	if err != nil {
		return nil, fmt.Errorf("resolve handle %q: %w", t.Handle, err)
	}

	recordURI := fmt.Sprintf("at://%s/sh.tangled.repo/%s", ident.DID, t.Repo)
	if repo, err := s.appview.GetRepo(ctx, recordURI); err == nil {
		if repo.URI == "" {
			repo.URI = recordURI
		}
		if isCanonicalRepoRecord(*repo) || stringValue(repo.Value.Name) == "" {
			return repo, nil
		}
		return s.resolveCanonicalRepo(ctx, ident.DID.String(), t, repo)
	} else if !shouldListRepoRecords(err) {
		return nil, fmt.Errorf("get repository %q: %w", t.Repo, err)
	}

	return s.resolveCanonicalRepo(ctx, ident.DID.String(), t, nil)
}

func (s *Service) resolveCanonicalRepo(ctx context.Context, ownerDID string, t Target, directRepo *tangled.Repo) (*tangled.Repo, error) {
	repos, err := s.appview.ListRepos(ctx, ownerDID)
	if err != nil {
		return nil, fmt.Errorf("list repos for %q: %w", t.Handle, err)
	}
	if directRepo != nil {
		return canonicalRepoForAlias(repos.Items, *directRepo), nil
	}
	for _, repo := range repos.Items {
		if stringValue(repo.Value.Name) == t.Repo || extractRKey(repo.URI) == t.Repo {
			return canonicalRepoForAlias(repos.Items, repo), nil
		}
	}
	return nil, fmt.Errorf("repo %q not found for handle %q", t.Repo, t.Handle)
}

func canonicalRepoForAlias(items []tangled.Repo, alias tangled.Repo) *tangled.Repo {
	aliasName := stringValue(alias.Value.Name)
	aliasRepoDID := stringValue(alias.Value.RepoDid)
	if aliasName == "" || aliasRepoDID == "" {
		return &alias
	}
	for index := range items {
		candidate := &items[index]
		if stringValue(candidate.Value.RepoDid) == aliasRepoDID && isCanonicalRepoRecord(*candidate) && stringValue(candidate.Value.Name) == aliasName {
			return candidate
		}
	}
	return &alias
}

// repoDID resolves a target to the key used by issue and pull-request listings.
func (s *Service) repoDID(ctx context.Context, t Target) (string, error) {
	record, err := s.resolveRepo(ctx, t)
	if err != nil {
		return "", err
	}
	return stringValue(record.Value.RepoDid), nil
}

// requireOwnedRepo resolves a target and verifies it is owned by did.
func (s *Service) requireOwnedRepo(ctx context.Context, t Target, did string) (*tangled.Repo, error) {
	repo, err := s.resolveRepo(ctx, t)
	if err != nil {
		return nil, err
	}
	if extractDID(repo.URI) != did {
		return nil, fmt.Errorf("repo %q is not owned by the authenticated user", t.String())
	}
	return repo, nil
}

// shouldListRepoRecords reports whether err indicates the record is absent at
// the name-derived rkey and so a full listing is required to find it.
func shouldListRepoRecords(err error) bool {
	var apiError *atclient.APIError
	if !errors.As(err, &apiError) {
		return false
	}
	if apiError.StatusCode == http.StatusNotFound {
		return true
	}

	// Bobbin wraps an upstream PDS 400 as a 502 when no record exists at the
	// name-derived rkey. Listing is required to find the record's actual rkey.
	return apiError.StatusCode == http.StatusBadGateway &&
		apiError.Name == "UpstreamFailed" &&
		strings.Contains(apiError.Message, "upstream returned status 400 Bad Request")
}
