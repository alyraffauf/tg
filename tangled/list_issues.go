package tangled

import (
	"context"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

type IssueRecord struct {
	Type       string   `json:"$type"`
	Repo       string   `json:"repo"`
	Title      string   `json:"title"`
	Body       string   `json:"body,omitempty"`
	CreatedAt  string   `json:"createdAt"`
	Mentions   []string `json:"mentions,omitempty"`
	References []string `json:"references,omitempty"`
}

// ListIssues fetches every issue for repoDid, following pagination
// cursors until the listing is exhausted.
func (t *Tangled) ListIssues(ctx context.Context, repoDid string, opts ListOpts) (*List, error) {
	items, err := fetchAllPages(ctx, func(ctx context.Context, cursor string) ([]ListItem, *string, error) {
		var page List
		if err := t.Client.Get(ctx, syntax.NSID("sh.tangled.repo.listIssues"), opts.params(repoDid, cursor), &page); err != nil {
			return nil, nil, err
		}
		return page.Items, page.Cursor, nil
	})
	if err != nil {
		return nil, fmt.Errorf("list issues for %q: %w", repoDid, err)
	}
	return &List{Items: items}, nil
}
