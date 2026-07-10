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

func (t *Tangled) ListIssues(ctx context.Context, repoDid string, opts ListOpts) (*List, error) {
	var out List
	if err := t.Client.Get(ctx, syntax.NSID("sh.tangled.repo.listIssues"), opts.params(repoDid), &out); err != nil {
		return nil, fmt.Errorf("list issues for %q: %w", repoDid, err)
	}
	return &out, nil
}
