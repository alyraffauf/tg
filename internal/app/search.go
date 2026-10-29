package app

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/alyraffauf/tg/tangled"
)

// SearchRepos searches Tangled's indexed repositories.
func (s *Service) SearchRepos(ctx context.Context, query string, limit int64) (*RepoListResult, error) {
	if query == "" {
		return nil, fmt.Errorf("search query cannot be empty")
	}
	if limit < 0 {
		return nil, fmt.Errorf("search limit cannot be negative")
	}
	result, err := s.appview.Search(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	items := make([]RepoItem, 0, len(result.Hits))
	var warnings []RecordWarning
	for _, hit := range result.Hits {
		if hit.URI == "" || hit.Value == nil {
			warnings = append(warnings, RecordWarning{URI: hit.URI, Error: "search result is null or malformed"})
			continue
		}
		if hit.NSID != tangled.RepoCollection {
			continue
		}
		data, err := json.Marshal(hit.Value)
		if err != nil {
			warnings = append(warnings, RecordWarning{URI: hit.URI, Error: fmt.Sprintf("encode repository search result: %v", err)})
			continue
		}
		var record tangled.Manifest
		if err := json.Unmarshal(data, &record); err != nil {
			warnings = append(warnings, RecordWarning{URI: hit.URI, Error: fmt.Sprintf("decode repository search result: %v", err)})
			continue
		}
		name := record.Declaration.Name
		if name == "" {
			name = extractRKey(hit.URI)
		}
		ownerDID := record.Declaration.Owner
		author := ownerDID
		if ident, err := s.resolver.ResolveDID(ctx, ownerDID); err == nil {
			author = ident.Handle.String()
		}
		items = append(items, RepoItem{
			Name: name, URI: hit.URI, Author: author,
			Description: stringValue(record.Description),
			CreatedAt:   record.CreatedAt, RepoDid: extractDID(hit.URI),
		})
	}
	for i := range items {
		host, err := s.repositoryHost(ctx, items[i].RepoDid)
		if err != nil {
			warnings = append(warnings, RecordWarning{URI: items[i].URI, Error: err.Error()})
		} else {
			items[i].Knot = host
		}
		stars, err := s.repoStars(ctx, items[i].RepoDid)
		if err != nil {
			warnings = append(warnings, RecordWarning{URI: items[i].URI, Error: fmt.Sprintf("count stars: %v", err)})
			continue
		}
		items[i].Stars = stars
	}
	return &RepoListResult{Items: items, Warnings: warnings}, nil
}
