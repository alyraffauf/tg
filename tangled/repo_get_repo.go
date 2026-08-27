package tangled

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/alyraffauf/tg/internal/tangledlex"
	"github.com/bluesky-social/indigo/atproto/syntax"
	lexutil "github.com/bluesky-social/indigo/lex/util"
)

type Repo struct {
	URI   string          `json:"uri"`
	CID   string          `json:"cid"`
	Value tangledlex.Repo `json:"value"`
}

func (t *Tangled) GetRepo(ctx context.Context, repoURI string) (*Repo, error) {
	response, err := tangledlex.RepoGetRepo(ctx, t.lexClient(), repoURI)
	if err != nil {
		return nil, fmt.Errorf("get tangled repo %q: %w", repoURI, err)
	}
	return decodeRepo(response.Uri, response.Cid, response.Value)
}

// GetRepoByDID returns the repository record with repoDID.
func (t *Tangled) GetRepoByDID(ctx context.Context, repoDID string) (*Repo, error) {
	var response struct {
		CID   *string                     `json:"cid,omitempty"`
		URI   string                      `json:"uri"`
		Value *lexutil.LexiconTypeDecoder `json:"value"`
	}
	if err := t.Client.Get(ctx, syntax.NSID("sh.tangled.repo.getRepoByRepoDid"), map[string]any{"repoDid": repoDID}, &response); err != nil {
		return nil, fmt.Errorf("get tangled repo by DID %q: %w", repoDID, err)
	}
	return decodeRepo(response.URI, response.CID, response.Value)
}

func decodeRepo(uri string, cid *string, valueDecoder *lexutil.LexiconTypeDecoder) (*Repo, error) {
	value, err := recordJSON(valueDecoder, &tangledlex.Repo{})
	if err != nil {
		return nil, fmt.Errorf("decode tangled repo %q: %w", uri, err)
	}

	var record tangledlex.Repo
	if err := json.Unmarshal(value, &record); err != nil {
		return nil, fmt.Errorf("decode tangled repo %q: %w", uri, err)
	}

	return &Repo{URI: uri, CID: dereference(cid), Value: record}, nil
}

func dereference(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
