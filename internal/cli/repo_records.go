package cli

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/alyraffauf/tg/tangled"
	"github.com/bluesky-social/indigo/atproto/atclient"
)

// resolveRepoRecord finds a repository record even when its rkey does not
// match the repository name.
func resolveRepoRecord(ctx context.Context, handle, name string) (*tangled.Repo, error) {
	ident, err := resolver.ResolveHandle(ctx, handle)
	if err != nil {
		return nil, fmt.Errorf("resolve handle %q: %w", handle, err)
	}

	recordURI := fmt.Sprintf("at://%s/sh.tangled.repo/%s", ident.DID, name)
	if repo, err := client.GetRepo(ctx, recordURI); err == nil {
		if repo.URI == "" {
			repo.URI = recordURI
		}
		return repo, nil
	} else if !shouldListRepoRecords(err) {
		return nil, fmt.Errorf("get repository %q: %w", name, err)
	}

	repos, err := client.ListRepos(ctx, ident.DID.String())
	if err != nil {
		return nil, fmt.Errorf("list repos for %q: %w", handle, err)
	}
	for index := range repos.Items {
		repo := &repos.Items[index]
		if repo.Value.Name == name || extractRKey(repo.URI) == name {
			return repo, nil
		}
	}
	return nil, fmt.Errorf("repo %q not found for handle %q", name, handle)
}

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

func requireOwnedRepo(ctx context.Context, handle, name, did string) (*tangled.Repo, error) {
	repo, err := resolveRepoRecord(ctx, handle, name)
	if err != nil {
		return nil, err
	}
	if extractDID(repo.URI) != did {
		return nil, fmt.Errorf("repo %q is not owned by the authenticated user", handle+"/"+name)
	}
	return repo, nil
}
