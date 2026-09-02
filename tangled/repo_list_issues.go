package tangled

import (
	"context"
	"fmt"

	"github.com/alyraffauf/tg/internal/tangledlex"
)

// ListIssues fetches every issue for repoDid, following pagination
// cursors until the listing is exhausted.
func (t *Tangled) ListIssues(ctx context.Context, repoDid string, opts ListOpts) (*List, error) {
	var warnings []RecordWarning
	items, err := fetchAllPages(ctx, opts.MaxItems, func(ctx context.Context, cursor string) ([]ListItem, *string, error) {
		page, err := tangledlex.RepoListIssues(ctx, t.lexClient(), opts.Author, cursor, opts.limit(), opts.Order, opts.State, repoDid)
		if err != nil {
			return nil, nil, err
		}
		items, itemWarnings := issueListItems(page.Items)
		warnings = append(warnings, itemWarnings...)
		return items, page.Cursor, nil
	})
	if err != nil {
		return nil, fmt.Errorf("list issues for %q: %w", repoDid, err)
	}
	return &List{Items: items, Warnings: warnings}, nil
}

func issueListItems(items []*tangledlex.RepoListIssues_IssueListItem) ([]ListItem, []RecordWarning) {
	decoded := make([]ListItem, 0, len(items))
	var warnings []RecordWarning
	for _, item := range items {
		if item == nil {
			warnings = append(warnings, RecordWarning{Error: "issue list contained a null item"})
			continue
		}
		value, err := recordJSON(item.Value, &tangledlex.RepoIssue{})
		if err != nil {
			warnings = append(warnings, RecordWarning{URI: item.Uri, Error: err.Error()})
			continue
		}
		decoded = append(decoded, ListItem{URI: item.Uri, CID: dereference(item.Cid), Value: value, State: item.State, StateUpdatedAt: dereference(item.StateUpdatedAt), CommentCount: item.CommentCount})
	}
	return decoded, warnings
}
