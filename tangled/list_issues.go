package tangled

import (
	"context"
	"encoding/json"
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

type IssueListItem struct {
	URI            string          `json:"uri"`
	CID            string          `json:"cid,omitempty"`
	Value          json.RawMessage `json:"value"`
	State          string          `json:"state"`
	StateUpdatedAt string          `json:"stateUpdatedAt,omitempty"`
	CommentCount   int64           `json:"commentCount"`
}

type IssueList struct {
	Items  []IssueListItem `json:"items"`
	Cursor *string         `json:"cursor"`
}

type IssueListOpts struct {
	Author string // only issues by this DID
	State  string // "open" or "closed"
	Limit  int64  // 1-1000, default 50
	Order  string // "asc" or "desc"
}

func (t *Tangled) ListIssues(ctx context.Context, repoDid string, opts IssueListOpts) (*IssueList, error) {
	params := map[string]any{
		"subject": repoDid,
	}
	if opts.Author != "" {
		params["author"] = opts.Author
	}
	if opts.State != "" {
		params["state"] = opts.State
	}
	if opts.Limit > 0 {
		params["limit"] = opts.Limit
	} else {
		params["limit"] = 50
	}
	if opts.Order != "" {
		params["order"] = opts.Order
	}

	var out IssueList
	err := t.Client.Get(ctx, syntax.NSID("sh.tangled.repo.listIssues"), params, &out)
	if err != nil {
		return nil, fmt.Errorf("list issues for %q: %w", repoDid, err)
	}
	return &out, nil
}
