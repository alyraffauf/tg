package app

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/alyraffauf/tg/internal/tangledlex"
)

// SearchRepos searches Tangled's indexed repositories.
func (s *Service) SearchRepos(ctx context.Context, query string, limit int64) ([]RepoItem, error) {
	if query == "" {
		return nil, fmt.Errorf("search query cannot be empty")
	}
	result, err := s.appview.Search(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	items := make([]RepoItem, 0, len(result.Hits))
	for _, hit := range result.Hits {
		if hit.NSID != "sh.tangled.repo" {
			continue
		}
		data, err := json.Marshal(hit.Value)
		if err != nil {
			continue
		}
		var record tangledlex.Repo
		if err := json.Unmarshal(data, &record); err != nil {
			continue
		}
		name := stringValue(record.Name)
		if name == "" {
			name = extractRKey(hit.URI)
		}
		ownerDID := extractDID(hit.URI)
		author := ownerDID
		if ident, err := s.resolver.ResolveDID(ctx, ownerDID); err == nil {
			author = ident.Handle.String()
		}
		items = append(items, RepoItem{
			Name: name, URI: hit.URI, Author: author,
			Knot: record.Knot, Description: stringValue(record.Description),
			CreatedAt: record.CreatedAt, RepoDid: stringValue(record.RepoDid),
		})
	}
	for i := range items {
		items[i].Stars = s.repoStars(ctx, items[i].RepoDid)
	}
	return items, nil
}
