package tangled

import (
	"context"
	"encoding/json"
	"fmt"
)

// maxPaginationPages caps how many pages fetchAllPages will follow, as a
// safety net against a server that never returns an empty cursor.
const maxPaginationPages = 1000

// ListItem is one item in an issue or pull-request listing.
type ListItem struct {
	URI            string          `json:"uri"`
	CID            string          `json:"cid,omitempty"`
	Value          json.RawMessage `json:"value"`
	State          string          `json:"state"`
	StateUpdatedAt string          `json:"stateUpdatedAt,omitempty"`
	CommentCount   int64           `json:"commentCount"`
}

// List is a page of issues or pull requests.
type List struct {
	Items  []ListItem `json:"items"`
	Cursor *string    `json:"cursor"`
}

// ListOpts are the query parameters shared by ListIssues and ListPulls.
type ListOpts struct {
	Author string // only items by this DID
	State  string // "open" or "closed"
	Limit  int64  // 1-1000, default 50
	Order  string // "asc" or "desc"
}

// params builds the XRPC query parameters for subject, requesting the page
// after cursor (the first page if cursor is empty).
func (o ListOpts) params(subject, cursor string) map[string]any {
	params := map[string]any{"subject": subject}
	if o.Author != "" {
		params["author"] = o.Author
	}
	if o.State != "" {
		params["state"] = o.State
	}
	if o.Limit > 0 {
		params["limit"] = o.Limit
	} else {
		params["limit"] = int64(50)
	}
	if o.Order != "" {
		params["order"] = o.Order
	}
	if cursor != "" {
		params["cursor"] = cursor
	}
	return params
}

// fetchAllPages calls fetch for successive pages, advancing the cursor it
// returns, until a page reports no further cursor. It returns every item
// across all pages combined.
func fetchAllPages[T any](ctx context.Context, fetch func(ctx context.Context, cursor string) (items []T, nextCursor *string, err error)) ([]T, error) {
	var all []T
	cursor := ""

	for page := 0; page < maxPaginationPages; page++ {
		items, nextCursor, err := fetch(ctx, cursor)
		if err != nil {
			return nil, err
		}
		all = append(all, items...)

		if nextCursor == nil || *nextCursor == "" {
			return all, nil
		}
		cursor = *nextCursor
	}

	return nil, fmt.Errorf("exceeded %d pages without reaching the end of the list", maxPaginationPages)
}
