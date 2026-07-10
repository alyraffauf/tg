package tangled

import "encoding/json"

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

// params builds the XRPC query parameters for subject.
func (o ListOpts) params(subject string) map[string]any {
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
		params["limit"] = 50
	}
	if o.Order != "" {
		params["order"] = o.Order
	}
	return params
}
