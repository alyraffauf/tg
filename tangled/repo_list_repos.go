package tangled

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/alyraffauf/tg/internal/tangledlex"
)

type RepoList struct {
	Items    []Repo          `json:"items"`
	Cursor   *string         `json:"cursor"`
	Warnings []RecordWarning `json:"warnings,omitempty"`
}

// ListRepos fetches every repo owned by ownerDid, following pagination
// cursors until the listing is exhausted.
func (t *Tangled) ListRepos(ctx context.Context, ownerDid string) (*RepoList, error) {
	var warnings []RecordWarning
	items, err := fetchAllPages(ctx, 0, func(ctx context.Context, cursor string) ([]Repo, *string, error) {
		page, err := tangledlex.RepoListRepos(ctx, t.lexClient(), cursor, 100, "", ownerDid)
		if err != nil {
			return nil, nil, err
		}
		items, itemWarnings := repoListItems(page.Items)
		warnings = append(warnings, itemWarnings...)
		return items, page.Cursor, nil
	})
	if err != nil {
		return nil, fmt.Errorf("list tangled repos for %q: %w", ownerDid, err)
	}

	return &RepoList{Items: items, Warnings: warnings}, nil
}

func repoListItems(items []*tangledlex.RepoListRepos_ListItem) ([]Repo, []RecordWarning) {
	decoded := make([]Repo, 0, len(items))
	var warnings []RecordWarning
	for _, item := range items {
		if item == nil {
			warnings = append(warnings, RecordWarning{Error: "repository list contained a null item"})
			continue
		}
		value, err := recordJSON(item.Value, &tangledlex.Repo{})
		if err != nil {
			warnings = append(warnings, RecordWarning{URI: item.Uri, Error: err.Error()})
			continue
		}
		var record tangledlex.Repo
		if err := json.Unmarshal(value, &record); err != nil {
			warnings = append(warnings, RecordWarning{URI: item.Uri, Error: fmt.Sprintf("decode repository record: %v", err)})
			continue
		}
		decoded = append(decoded, Repo{URI: item.Uri, CID: dereference(item.Cid), Value: record})
	}
	return decoded, warnings
}
