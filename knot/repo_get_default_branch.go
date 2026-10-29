package knot

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	lexutil "github.com/bluesky-social/indigo/lex/util"
)

type DefaultBranch struct {
	Name string `json:"name"`
	Hash string `json:"hash"`
}

func (c *Client) GetDefaultBranch(ctx context.Context, repo string) (*DefaultBranch, error) {
	record, err := c.GetRecord(ctx, repo, "org.tangled.git.refs", "self")
	if err != nil {
		return nil, err
	}
	var refs struct {
		DefaultBranch string           `json:"defaultBranch"`
		Refs          lexutil.LexBytes `json:"refs"`
	}
	data, err := json.Marshal(record.Value)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &refs); err != nil {
		return nil, err
	}
	if refs.DefaultBranch == "" || refs.Refs == nil {
		return nil, fmt.Errorf("Git refs record omitted its default branch or refs bytes")
	}
	hash := ""
	for _, line := range strings.Split(string(refs.Refs), "\n") {
		oid, ref, found := strings.Cut(line, " ")
		if found && ref == "refs/heads/"+refs.DefaultBranch {
			hash = oid
			break
		}
	}
	return &DefaultBranch{Name: refs.DefaultBranch, Hash: hash}, nil
}
