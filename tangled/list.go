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
	Items    []ListItem      `json:"items"`
	Cursor   *string         `json:"cursor"`
	Warnings []RecordWarning `json:"warnings,omitempty"`
}

type RecordWarning struct {
	URI   string `json:"uri,omitempty"`
	Error string `json:"error"`
}

// ListOpts are the query parameters shared by ListIssues and ListPulls.
type ListOpts struct {
	Author   string // only items by this DID
	State    string // "open" or "closed"
	Limit    int64  // page size: 1-1000, default 50
	MaxItems int64  // maximum results to return; zero returns every item
	Order    string // "asc" or "desc"
}

func (o ListOpts) limit() int64 {
	limit := int64(50)
	if o.Limit > 0 {
		limit = o.Limit
	}
	if o.MaxItems > 0 && o.MaxItems < limit {
		return o.MaxItems
	}
	return limit
}

// params builds query parameters for a paginated issue or pull request list.
func (o ListOpts) params(subject, cursor string) map[string]any {
	params := map[string]any{"subject": subject, "limit": o.limit()}
	if o.Author != "" {
		params["author"] = o.Author
	}
	if o.State != "" {
		params["state"] = o.State
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
func fetchAllPages[T any](ctx context.Context, maximumItems int64, fetch func(ctx context.Context, cursor string) (items []T, nextCursor *string, err error)) ([]T, error) {
	var all []T
	cursor := ""

	for page := 0; page < maxPaginationPages; page++ {
		items, nextCursor, err := fetch(ctx, cursor)
		if err != nil {
			return nil, err
		}
		if maximumItems > 0 && int64(len(items)) > maximumItems-int64(len(all)) {
			items = items[:maximumItems-int64(len(all))]
		}
		all = append(all, items...)

		if maximumItems > 0 && int64(len(all)) == maximumItems {
			return all, nil
		}

		if nextCursor == nil || *nextCursor == "" {
			return all, nil
		}
		cursor = *nextCursor
	}

	return nil, fmt.Errorf("exceeded %d pages without reaching the end of the list", maxPaginationPages)
}
