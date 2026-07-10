package tangled

import (
	"context"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/atdata"
	"github.com/bluesky-social/indigo/atproto/syntax"
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

// ListPulls fetches every pull request for repoDid, following pagination
// cursors until the listing is exhausted.
func (t *Tangled) ListPulls(ctx context.Context, repoDid string, opts ListOpts) (*List, error) {
	items, err := fetchAllPages(ctx, func(ctx context.Context, cursor string) ([]ListItem, *string, error) {
		var page List
		if err := t.Client.Get(ctx, syntax.NSID("sh.tangled.repo.listPulls"), opts.params(repoDid, cursor), &page); err != nil {
			return nil, nil, err
		}
		return page.Items, page.Cursor, nil
	})
	if err != nil {
		return nil, fmt.Errorf("list PRs for %q: %w", repoDid, err)
	}
	return &List{Items: items}, nil
}
