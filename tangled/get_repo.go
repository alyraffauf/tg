package tangled

import (
	"context"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

// RepoRecord is the value of a sh.tangled.repo lexicon record.
type RepoRecord struct {
	Type        string   `json:"$type"`
	Name        string   `json:"name,omitempty"`
	Description string   `json:"description,omitempty"`
	Knot        string   `json:"knot"`
	CreatedAt   string   `json:"createdAt"`
	Owner       string   `json:"owner,omitempty"`
	AddedAt     string   `json:"addedAt,omitempty"`
	RepoDid     string   `json:"repoDid,omitempty"`
	Spindle     string   `json:"spindle,omitempty"`
	Website     string   `json:"website,omitempty"`
	Labels      []string `json:"labels,omitempty"`
}

type Repo struct {
	URI   string     `json:"uri"`
	CID   string     `json:"cid"`
	Value RepoRecord `json:"value"`
}

func (t *Tangled) GetRepo(ctx context.Context, repoURI string) (*Repo, error) {
	var repo Repo
	err := t.Client.Get(ctx, syntax.NSID("sh.tangled.repo.getRepo"), map[string]any{"repo": repoURI}, &repo)
	if err != nil {
		return nil, fmt.Errorf("get tangled repo %q: %w", repoURI, err)
	}

	return &repo, nil
}
