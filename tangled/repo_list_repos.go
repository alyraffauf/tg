package tangled

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

type RepoList struct {
	Items    []Repo          `json:"items"`
	Cursor   *string         `json:"cursor"`
	Warnings []RecordWarning `json:"warnings,omitempty"`
}

func (t *Tangled) ListRepos(ctx context.Context, owner string) (*RepoList, error) {
	var warnings []RecordWarning
	items, err := fetchAllPages(ctx, 0, func(ctx context.Context, cursor string) ([]Repo, *string, error) {
		var page struct {
			Items  []json.RawMessage `json:"items"`
			Cursor *string           `json:"cursor"`
		}
		params := map[string]any{"subject": owner, "limit": 100}
		if cursor != "" {
			params["cursor"] = cursor
		}
		if err := t.Client.Get(ctx, syntax.NSID("org.tangled.repo.listRepos"), params, &page); err != nil {
			return nil, nil, err
		}
		var decoded []Repo
		for _, raw := range page.Items {
			var repo Repo
			if err := json.Unmarshal(raw, &repo); err != nil {
				warnings = append(warnings, RecordWarning{Error: err.Error()})
				continue
			}
			decoded = append(decoded, repo)
		}
		return decoded, page.Cursor, nil
	})
	if err != nil {
		return nil, fmt.Errorf("list repositories: %w", err)
	}
	return &RepoList{Items: items, Warnings: warnings}, nil
}
