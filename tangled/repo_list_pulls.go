package tangled

import (
	"context"
	"fmt"

	"github.com/alyraffauf/tg/internal/tangledlex"
)

// ListPulls fetches every pull request for repoDid, following pagination
// cursors until the listing is exhausted.
func (t *Tangled) ListPulls(ctx context.Context, repoDid string, opts ListOpts) (*List, error) {
	items, err := fetchAllPages(ctx, opts.MaxItems, func(ctx context.Context, cursor string) ([]ListItem, *string, error) {
		page, err := tangledlex.RepoListPulls(ctx, t.lexClient(), opts.Author, cursor, opts.limit(), opts.Order, opts.State, repoDid)
		if err != nil {
			return nil, nil, err
		}
		items, err := pullListItems(page.Items)
		return items, page.Cursor, err
	})
	if err != nil {
		return nil, fmt.Errorf("list PRs for %q: %w", repoDid, err)
	}
	return &List{Items: items}, nil
}

func pullListItems(items []*tangledlex.RepoListPulls_PullListItem) ([]ListItem, error) {
	decoded := make([]ListItem, 0, len(items))
	for _, item := range items {
		value, err := recordJSON(item.Value, &tangledlex.RepoPull{})
		if err != nil {
			return nil, err
		}
		decoded = append(decoded, ListItem{URI: item.Uri, CID: dereference(item.Cid), Value: value, State: item.State, StateUpdatedAt: dereference(item.StateUpdatedAt), CommentCount: item.CommentCount})
	}
	return decoded, nil
}
