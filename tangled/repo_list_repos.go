package tangled

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/alyraffauf/tg/internal/tangledlex"
	"github.com/bluesky-social/indigo/atproto/atclient"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

type RepoList struct {
	Items    []Repo          `json:"items"`
	Cursor   *string         `json:"cursor"`
	Warnings []RecordWarning `json:"warnings,omitempty"`
}

// repoRecordValue decodes every record shape a repository listing returns: the
// legacy sh.tangled.repo record and the org.tangled.repo.manifest that replaced
// it, where the name lives in declaration.name and the repo DID is the authority
// of the record URI.
type repoRecordValue struct {
	CreatedAt   string  `json:"createdAt"`
	Description *string `json:"description"`
	Knot        string  `json:"knot"`
	Name        *string `json:"name"`
	RepoDid     *string `json:"repoDid"`
	Declaration *struct {
		Name  string `json:"name"`
		Owner string `json:"owner"`
	} `json:"declaration"`
}

type repoListItem struct {
	URI   string          `json:"uri"`
	CID   *string         `json:"cid,omitempty"`
	Value json.RawMessage `json:"value"`
}

type repoListPage struct {
	Items  []repoListItem `json:"items"`
	Cursor *string        `json:"cursor,omitempty"`
}

// ListRepos fetches every repo owned by ownerDid, following pagination
// cursors until the listing is exhausted.
func (t *Tangled) ListRepos(ctx context.Context, ownerDid string) (*RepoList, error) {
	var warnings []RecordWarning
	items, err := fetchAllPages(ctx, 0, func(ctx context.Context, cursor string) ([]Repo, *string, error) {
		page, next, pageWarnings, err := t.listReposPage(ctx, ownerDid, cursor)
		if err != nil {
			return nil, nil, err
		}
		warnings = append(warnings, pageWarnings...)
		return page, next, nil
	})
	if err != nil {
		return nil, fmt.Errorf("list tangled repos for %q: %w", ownerDid, err)
	}

	return &RepoList{Items: items, Warnings: warnings}, nil
}

// listReposPage reads one page of a repository listing. The appview serves
// org.tangled.repo.listRepos; an appview that predates that method still serves
// sh.tangled.repo.listRepos.
func (t *Tangled) listReposPage(ctx context.Context, ownerDid, cursor string) ([]Repo, *string, []RecordWarning, error) {
	params := map[string]any{"subject": ownerDid, "limit": 100}
	if cursor != "" {
		params["cursor"] = cursor
	}

	var page repoListPage
	err := t.Client.Get(ctx, syntax.NSID("org.tangled.repo.listRepos"), params, &page)
	if err == nil {
		items, warnings := repoListItems(page.Items)
		return items, page.Cursor, warnings, nil
	}
	if !methodNotServed(err) {
		return nil, nil, nil, err
	}

	legacy, err := tangledlex.RepoListRepos(ctx, t.lexClient(), cursor, 100, "", ownerDid)
	if err != nil {
		return nil, nil, nil, err
	}

	items := make([]repoListItem, 0, len(legacy.Items))
	for _, item := range legacy.Items {
		if item == nil {
			continue
		}
		raw, marshalErr := json.Marshal(item.Value)
		if marshalErr != nil {
			return nil, nil, nil, fmt.Errorf("encode repository record %q: %w", item.Uri, marshalErr)
		}
		items = append(items, repoListItem{URI: item.Uri, CID: item.Cid, Value: raw})
	}

	decoded, warnings := repoListItems(items)
	return decoded, legacy.Cursor, warnings, nil
}

func repoListItems(items []repoListItem) ([]Repo, []RecordWarning) {
	decoded := make([]Repo, 0, len(items))
	var warnings []RecordWarning
	for _, item := range items {
		var value repoRecordValue
		if err := json.Unmarshal(item.Value, &value); err != nil {
			warnings = append(warnings, RecordWarning{URI: item.URI, Error: fmt.Sprintf("decode repository record: %v", err)})
			continue
		}

		decoded = append(decoded, Repo{URI: item.URI, CID: dereference(item.CID), Value: value.record(item.URI)})
	}
	return decoded, warnings
}

// record converts the decoded record into the legacy shape the rest of tg reads.
func (v repoRecordValue) record(uri string) tangledlex.Repo {
	record := tangledlex.Repo{
		CreatedAt:   v.CreatedAt,
		Description: v.Description,
		Knot:        v.Knot,
		Name:        v.Name,
		RepoDid:     v.RepoDid,
	}
	if v.Declaration != nil {
		name := v.Declaration.Name
		record.Name = &name
		if did := repoDidFromURI(uri); did != "" {
			record.RepoDid = &did
		}
	}
	return record
}

// repoDidFromURI extracts the DID from at://<did>/org.tangled.repo.manifest/self.
func repoDidFromURI(uri string) string {
	rest, ok := strings.CutPrefix(uri, "at://")
	if !ok {
		return ""
	}
	did, _, ok := strings.Cut(rest, "/")
	if !ok {
		return ""
	}
	return did
}

// methodNotServed reports whether err says the appview does not serve the
// method that was asked for.
func methodNotServed(err error) bool {
	var apiError *atclient.APIError
	if !errors.As(err, &apiError) {
		return false
	}
	switch apiError.StatusCode {
	case http.StatusNotFound:
		return apiError.Name == "XRPCNotSupported"
	case http.StatusNotImplemented:
		return true
	}
	return false
}
