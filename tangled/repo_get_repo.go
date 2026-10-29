package tangled

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

// RepoMetadata is normalized repository metadata, independent of record storage.
type RepoMetadata struct {
	Name        *string
	Knot        string
	RepoDid     *string
	Spindle     *string
	Description *string
	Website     *string
	Topics      []string
	CreatedAt   string
}

type Repo struct {
	URI            string       `json:"uri"`
	CID            string       `json:"cid"`
	OwnerDID       string       `json:"ownerDid"`
	DeclarationURI string       `json:"declarationUri"`
	GitURL         string       `json:"gitUrl"`
	RecordURL      string       `json:"recordUrl"`
	DefaultBranch  string       `json:"defaultBranch"`
	Value          RepoMetadata `json:"value"`
	Manifest       Manifest     `json:"manifest"`
}

func (r *Repo) UnmarshalJSON(data []byte) error {
	var wire struct {
		URI           string   `json:"uri"`
		CID           string   `json:"cid"`
		Value         Manifest `json:"value"`
		Knot          string   `json:"knot"`
		DefaultBranch string   `json:"defaultBranch"`
		Git           *struct {
			DefaultBranch string `json:"defaultBranch"`
		} `json:"git"`
	}
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	uri, err := syntax.ParseATURI(wire.URI)
	if err != nil || uri.Collection().String() != RepoCollection || uri.RecordKey().String() != "self" || wire.Value.Type != RepoCollection {
		return fmt.Errorf("invalid repository manifest %q", wire.URI)
	}
	if _, err := syntax.ParseDID(uri.Authority().String()); err != nil {
		return err
	}
	if _, err := syntax.ParseDID(wire.Value.Declaration.Owner); err != nil {
		return fmt.Errorf("invalid manifest owner: %w", err)
	}
	if _, err := syntax.ParseRecordKey(wire.Value.Declaration.Name); err != nil {
		return fmt.Errorf("invalid manifest name: %w", err)
	}
	did := uri.Authority().String()
	*r = Repo{URI: wire.URI, CID: wire.CID, OwnerDID: wire.Value.Declaration.Owner, Manifest: wire.Value, DefaultBranch: wire.DefaultBranch}
	if wire.Git != nil {
		r.DefaultBranch = wire.Git.DefaultBranch
	}
	r.DeclarationURI = "at://" + r.OwnerDID + "/" + DeclarationCollection + "/" + strings.ToLower(wire.Value.Declaration.Name)
	r.Value = RepoMetadata{Name: &r.Manifest.Declaration.Name, Knot: wire.Knot, RepoDid: &did, Description: wire.Value.Description, Website: wire.Value.Website, Spindle: wire.Value.Spindle, Topics: wire.Value.Topics, CreatedAt: wire.Value.CreatedAt}
	return nil
}

// GetRepo resolves an owner's declaration by its name through Bobbin.
func (t *Tangled) GetRepo(ctx context.Context, reference string) (*Repo, error) {
	uri, err := syntax.ParseATURI(reference)
	if err != nil {
		return nil, err
	}
	params := map[string]any{"owner": uri.Authority().String(), "name": uri.RecordKey().String()}
	if uri.Collection().String() == RepoCollection {
		return t.GetRepoByDID(ctx, uri.Authority().String())
	}
	var result Repo
	if err := t.Client.Get(ctx, syntax.NSID("org.tangled.temp.repo.getRepoByName"), params, &result); err != nil {
		return nil, fmt.Errorf("resolve repository %q: %w", reference, err)
	}
	return &result, nil
}

func (t *Tangled) GetRepoByDID(ctx context.Context, did string) (*Repo, error) {
	var result Repo
	if err := t.Client.Get(ctx, syntax.NSID("org.tangled.repo.getRepoByRepoDid"), map[string]any{"repoDid": did}, &result); err != nil {
		return nil, fmt.Errorf("get repository %q: %w", did, err)
	}
	return &result, nil
}
func dereference(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
