package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/alyraffauf/tg/tangled"
)

// ListRepos lists every repository owned by handle.
func (s *Service) ListRepos(ctx context.Context, handle string) (*RepoListResult, error) {
	ident, err := s.resolver.ResolveHandle(ctx, handle)
	if err != nil {
		return nil, fmt.Errorf("resolve handle %q: %w", handle, err)
	}
	repos, err := s.appview.ListRepos(ctx, ident.DID.String())
	if err != nil {
		return nil, fmt.Errorf("list repos for %q: %w", handle, err)
	}
	warnings := recordWarnings(repos.Warnings)
	for index := range repos.Items {
		repo := &repos.Items[index]
		host, err := s.repositoryHost(ctx, stringValue(repo.Value.RepoDid))
		if err != nil {
			warnings = append(warnings, RecordWarning{URI: repo.URI, Error: err.Error()})
		} else {
			repo.Value.Knot = host
		}
	}
	return &RepoListResult{Items: buildRepoItems(repos.Items, handle), Warnings: warnings}, nil
}

func buildRepoItems(items []tangled.Repo, author string) []RepoItem {
	canonicalRepos := canonicalRepoItems(items)
	result := make([]RepoItem, 0, len(canonicalRepos))
	for _, tangledRepo := range canonicalRepos {
		name := stringValue(tangledRepo.Value.Name)
		if name == "" {
			// Fall back to the rkey segment of the at:// URI.
			if idx := strings.LastIndex(tangledRepo.URI, "/"); idx != -1 {
				name = tangledRepo.URI[idx+1:]
			}
		}
		result = append(result, RepoItem{
			Name:        name,
			URI:         tangledRepo.URI,
			Author:      author,
			Knot:        tangledRepo.Value.Knot,
			Description: stringValue(tangledRepo.Value.Description),
			CreatedAt:   tangledRepo.Value.CreatedAt,
			RepoDid:     stringValue(tangledRepo.Value.RepoDid),
		})
	}
	return result
}

// canonicalRepoItems returns one record per repository DID. Renamed repositories
// retain old records as aliases; the record keyed by its name is current.
func canonicalRepoItems(items []tangled.Repo) []tangled.Repo {
	result := make([]tangled.Repo, 0, len(items))
	indexesByRepoDID := make(map[string]int)
	for _, repo := range items {
		repoDID := stringValue(repo.Value.RepoDid)
		if repoDID == "" {
			result = append(result, repo)
			continue
		}

		resultIndex, found := indexesByRepoDID[repoDID]
		if !found {
			indexesByRepoDID[repoDID] = len(result)
			result = append(result, repo)
			continue
		}
		if isCanonicalRepoRecord(repo) && !isCanonicalRepoRecord(result[resultIndex]) {
			result[resultIndex] = repo
		}
	}
	return result
}

func isCanonicalRepoRecord(repo tangled.Repo) bool {
	name := stringValue(repo.Value.Name)
	return name != "" && extractRKey(repo.URI) == "self"
}
