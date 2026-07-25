package tangled

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/alyraffauf/tg/internal/tangledlex"
)

type RepoList struct {
	Items  []Repo  `json:"items"`
	Cursor *string `json:"cursor"`
}

// ListRepos fetches every repo owned by ownerDid, following pagination
// cursors until the listing is exhausted.
func (t *Tangled) ListRepos(ctx context.Context, ownerDid string) (*RepoList, error) {
	items, err := fetchAllPages(ctx, func(ctx context.Context, cursor string) ([]Repo, *string, error) {
		page, err := tangledlex.RepoListRepos(ctx, t.lexClient(), cursor, 100, "", ownerDid)
		if err != nil {
			return nil, nil, err
		}
		items, err := repoListItems(page.Items)
		return items, page.Cursor, err
	})
	if err != nil {
		return nil, fmt.Errorf("list tangled repos for %q: %w", ownerDid, err)
	}

	return &RepoList{Items: items}, nil
}

func repoListItems(items []*tangledlex.RepoListRepos_ListItem) ([]Repo, error) {
	decoded := make([]Repo, 0, len(items))
	for _, item := range items {
		value, err := recordJSON(item.Value, &tangledlex.Repo{})
		if err != nil {
			return nil, err
		}
		var record tangledlex.Repo
		if err := json.Unmarshal(value, &record); err != nil {
			return nil, fmt.Errorf("decode repository record: %w", err)
		}
		decoded = append(decoded, Repo{URI: item.Uri, CID: dereference(item.Cid), Value: record})
	}
	return decoded, nil
}
