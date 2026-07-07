package tangled

import (
	"context"
	"fmt"

	"github.com/bluesky-social/indigo/xrpc"
)

type RepoList struct {
	Items  []Repo  `json:"items"`
	Cursor *string `json:"cursor"`
}

func (t *Tangled) ListRepos(ctx context.Context, ownerDid string) (*RepoList, error) {
	var repos RepoList
	err := t.Client.Do(ctx,
		xrpc.Query, "", "sh.tangled.repo.listRepos", map[string]any{
			"subject": ownerDid,
			"limit":   100,
		},
		nil,
		&repos,
	)
	if err != nil {
		return nil, fmt.Errorf("list tangled repos for %q: %w", ownerDid, err)
	}

	return &repos, nil
}
