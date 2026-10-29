package tangled

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/bluesky-social/indigo/atproto/atclient"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

func (t *Tangled) AwaitRecord(ctx context.Context, uri, cid string) error {
	var out struct {
		Status string `json:"status"`
		Detail string `json:"detail"`
	}
	params := map[string]any{"uri": uri}
	if cid != "" {
		params["cid"] = cid
	}
	if err := t.Client.Get(ctx, syntax.NSID("org.tangled.bobbin.awaitRecord"), params, &out); err != nil {
		return err
	}
	if out.Status != "indexed" && out.Status != "deleted" {
		return fmt.Errorf("record rejected by Bobbin: %s", out.Detail)
	}
	return nil
}
func (t *Tangled) FormatPatch(ctx context.Context, repo, base, head string) ([]byte, error) {
	req := atclient.NewAPIRequest(http.MethodGet, syntax.NSID("org.tangled.temp.git.formatPatch"), nil)
	req.QueryParams.Set("repo", repo)
	req.QueryParams.Set("from", base)
	req.QueryParams.Set("to", head)
	resp, err := t.Client.Do(ctx, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		var body atclient.ErrorBody
		if err := json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&body); err != nil {
			return nil, &atclient.APIError{StatusCode: resp.StatusCode}
		}
		return nil, body.APIError(resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, (100<<20)+1))
	if len(data) > 100<<20 {
		return nil, fmt.Errorf("patch exceeds 100 MiB")
	}
	return data, err
}
func (t *Tangled) GetRef(ctx context.Context, repo, ref string) (string, error) {
	var out struct {
		OID string `json:"oid"`
		Ref struct {
			OID string `json:"oid"`
		} `json:"ref"`
	}
	err := t.Client.Get(ctx, syntax.NSID("org.tangled.temp.git.getRef"), map[string]any{"repo": repo, "ref": ref}, &out)
	if err != nil {
		return "", err
	}
	if out.OID == "" {
		out.OID = out.Ref.OID
	}
	if out.OID == "" {
		return "", fmt.Errorf("ref %q response has no commit", ref)
	}
	return out.OID, nil
}
func (t *Tangled) Merge(ctx context.Context, token string, input any) error {
	client := *t.Client
	client.Auth = serviceBearer(token)
	return client.Post(ctx, syntax.NSID("sh.tangled.git.mergeCommit"), input, nil)
}

type serviceBearer string

func (b serviceBearer) DoWithAuth(c *http.Client, r *http.Request, _ syntax.NSID) (*http.Response, error) {
	r.Header.Set("Authorization", "Bearer "+string(b))
	return c.Do(r)
}
