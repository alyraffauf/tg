package tangled

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/alyraffauf/tg/internal/tangledlex"
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
	value, err := recordJSON(response.Value, &tangledlex.Repo{})
	if err != nil {
		return nil, fmt.Errorf("decode tangled repo %q: %w", repoURI, err)
	}

	var record tangledlex.Repo
	if err := json.Unmarshal(value, &record); err != nil {
		return nil, fmt.Errorf("decode tangled repo %q: %w", repoURI, err)
	}

	return &Repo{URI: response.Uri, CID: dereference(response.Cid), Value: record}, nil
}

func dereference(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
