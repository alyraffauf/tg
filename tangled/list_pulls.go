package tangled

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/atdata"
	"github.com/bluesky-social/indigo/xrpc"
)

type PullRecord struct {
	Type        string      `json:"$type"`
	Title       string      `json:"title"`
	Body        string      `json:"body,omitempty"`
	CreatedAt   string      `json:"createdAt"`
	Mentions    []string    `json:"mentions,omitempty"`
	References  []string    `json:"references,omitempty"`
	DependentOn string      `json:"dependentOn,omitempty"`
	Target      PullTarget  `json:"target"`
	Source      PullSource  `json:"source"`
	Rounds      []PullRound `json:"rounds"`
}

type PullTarget struct {
	Repo   string `json:"repo"`
	Branch string `json:"branch"`
}

type PullSource struct {
	Repo   string `json:"repo,omitempty"`
	Branch string `json:"branch"`
}

type PullRound struct {
	CreatedAt string    `json:"createdAt"`
	PatchBlob PatchBlob `json:"patchBlob"`
}

// PatchBlob is a gzipped git patch referenced by CID.
type PatchBlob struct {
	Type     string         `json:"$type"`
	Ref      atdata.CIDLink `json:"ref"`
	MimeType string         `json:"mimeType"`
	Size     int64          `json:"size"`
}

type PullListItem struct {
	URI            string          `json:"uri"`
	CID            string          `json:"cid,omitempty"`
	Value          json.RawMessage `json:"value"`
	State          string          `json:"state"`
	StateUpdatedAt string          `json:"stateUpdatedAt,omitempty"`
	CommentCount   int64           `json:"commentCount"`
}

type PullList struct {
	Items  []PullListItem `json:"items"`
	Cursor *string        `json:"cursor"`
}

type PullListOpts struct {
	Author string // only pulls by this DID
	State  string // "open" or "closed"
	Limit  int64  // 1-1000, default 50
	Order  string // "asc" or "desc"
}

func (t *Tangled) ListPulls(ctx context.Context, repoDid string, opts PullListOpts) (*PullList, error) {
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

	var out PullList
	err := t.Client.Do(ctx, xrpc.Query, "", "sh.tangled.repo.listPulls", params, nil, &out)
	if err != nil {
		return nil, fmt.Errorf("list PRs for %q: %w", repoDid, err)
	}
	return &out, nil
}
