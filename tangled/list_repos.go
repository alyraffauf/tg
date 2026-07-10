package tangled

import (
	"context"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

type RepoList struct {
	Items  []Repo  `json:"items"`
	Cursor *string `json:"cursor"`
}

// ListRepos fetches every repo owned by ownerDid, following pagination
// cursors until the listing is exhausted.
func (t *Tangled) ListRepos(ctx context.Context, ownerDid string) (*RepoList, error) {
	items, err := fetchAllPages(ctx, func(ctx context.Context, cursor string) ([]Repo, *string, error) {
		params := map[string]any{"subject": ownerDid, "limit": 100}
		if cursor != "" {
			params["cursor"] = cursor
		}
		var page RepoList
		if err := t.Client.Get(ctx, syntax.NSID("sh.tangled.repo.listRepos"), params, &page); err != nil {
			return nil, nil, err
		}
		return page.Items, page.Cursor, nil
	})
	if err != nil {
		return nil, fmt.Errorf("list tangled repos for %q: %w", ownerDid, err)
	}

	return &RepoList{Items: items}, nil
}
