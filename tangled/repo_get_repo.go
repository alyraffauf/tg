package tangled

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/alyraffauf/tg/internal/tangledlex"
	"github.com/bluesky-social/indigo/atproto/syntax"
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

	value, err := rawRecordJSON(response.Value)
	if err != nil {
		return nil, fmt.Errorf("decode tangled repo %q: %w", repoURI, err)
	}
	return decodeRepo(response.Uri, response.Cid, value)
}

// GetRepoByDID returns the repository record with repoDID.
func (t *Tangled) GetRepoByDID(ctx context.Context, repoDID string) (*Repo, error) {
	// The record stays raw: an appview serving org.tangled.repo.manifest returns
	// a lexicon type this client does not register.
	var response struct {
		CID   *string         `json:"cid,omitempty"`
		URI   string          `json:"uri"`
		Value json.RawMessage `json:"value"`
	}
	params := map[string]any{"repoDid": repoDID}

	err := t.Client.Get(ctx, syntax.NSID("org.tangled.repo.getRepoByRepoDid"), params, &response)
	if err != nil && methodNotServed(err) {
		err = t.Client.Get(ctx, syntax.NSID("sh.tangled.repo.getRepoByRepoDid"), params, &response)
	}
	if err != nil {
		return nil, fmt.Errorf("get tangled repo by DID %q: %w", repoDID, err)
	}

	return decodeRepo(response.URI, response.CID, response.Value)
}

func decodeRepo(uri string, cid *string, value json.RawMessage) (*Repo, error) {
	var record repoRecordValue
	if err := json.Unmarshal(value, &record); err != nil {
		return nil, fmt.Errorf("decode tangled repo %q: %w", uri, err)
	}

	return &Repo{URI: uri, CID: dereference(cid), Value: record.record(uri)}, nil
}

func dereference(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
