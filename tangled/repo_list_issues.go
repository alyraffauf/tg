package tangled

import (
	"context"
	"fmt"

	"github.com/alyraffauf/tg/internal/tangledlex"
)

// ListIssues fetches every issue for repoDid, following pagination
// cursors until the listing is exhausted.
func (t *Tangled) ListIssues(ctx context.Context, repoDid string, opts ListOpts) (*List, error) {
	items, err := fetchAllPages(ctx, func(ctx context.Context, cursor string) ([]ListItem, *string, error) {
		page, err := tangledlex.RepoListIssues(ctx, t.lexClient(), opts.Author, cursor, opts.limit(), opts.Order, opts.State, repoDid)
		if err != nil {
			return nil, nil, err
		}
		items, err := issueListItems(page.Items)
		return items, page.Cursor, err
	})
	if err != nil {
		return nil, fmt.Errorf("list issues for %q: %w", repoDid, err)
	}
	return &List{Items: items}, nil
}

func issueListItems(items []*tangledlex.RepoListIssues_IssueListItem) ([]ListItem, error) {
	decoded := make([]ListItem, 0, len(items))
	for _, item := range items {
		value, err := recordJSON(item.Value, &tangledlex.RepoIssue{})
		if err != nil {
			return nil, err
		}
		decoded = append(decoded, ListItem{URI: item.Uri, CID: dereference(item.Cid), Value: value, State: item.State, StateUpdatedAt: dereference(item.StateUpdatedAt), CommentCount: item.CommentCount})
	}
	return decoded, nil
}
