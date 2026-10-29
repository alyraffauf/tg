package knot

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bluesky-social/indigo/atproto/atclient"
)

func TestDescribeKnotLegacyDetectionAndFailures(t *testing.T) {
	for _, test := range []struct {
		name       string
		status     int
		body       string
		wantLegacy bool
		wantError  bool
		wantCalls  int
	}{
		{"modern", 200, `{"did":"did:web:knot.example","capabilities":["knot-social-writes"]}`, false, false, 1},
		{"legacy", 404, `{"error":"XRPCNotSupported"}`, true, false, 2},
		{"unavailable", 503, `{"error":"Unavailable"}`, false, true, 1},
		{"unauthorized", 401, `{"error":"AuthRequired"}`, false, true, 1},
		{"malformed", 200, `{"did":"did:web:knot.example"}`, false, true, 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				w.Header().Set("Content-Type", "application/json")
				if calls == 2 {
					if r.URL.Path != "/xrpc/sh.tangled.knot.version" {
						t.Errorf("unexpected fallback %s", r.URL)
					}
					fmt.Fprint(w, `{"capabilities":[]}`)
					return
				}
				w.WriteHeader(test.status)
				fmt.Fprint(w, test.body)
			}))
			defer server.Close()
			client := &Client{APIClient: &atclient.APIClient{Host: server.URL, Client: server.Client()}}
			description, err := client.DescribeKnot(context.Background())
			if (err != nil) != test.wantError || calls != test.wantCalls {
				t.Fatalf("result=%+v err=%v calls=%d", description, err, calls)
			}
			if err == nil && description.Legacy != test.wantLegacy {
				t.Fatalf("description=%+v", description)
			}
		})
	}
}
func TestModernRepositoryCreateAndBranchContracts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Header.Get("Authorization") != "Bearer token" {
			t.Errorf("missing service token")
		}
		var input map[string]any
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			t.Error(err)
		}
		switch r.URL.Path {
		case "/xrpc/org.tangled.repo.createRepo":
			if input["source"].(map[string]any)["did"] != "did:plc:source" {
				t.Errorf("input=%+v", input)
			}
			fmt.Fprint(w, `{"did":"did:plc:repo"}`)
		case "/xrpc/org.tangled.git.setDefaultBranch":
			if input["branch"] != "main" || input["defaultBranch"] != nil {
				t.Errorf("input=%+v", input)
			}
			fmt.Fprint(w, `{}`)
		default:
			t.Errorf("unexpected endpoint %s", r.URL)
			fmt.Fprint(w, `{}`)
		}
	}))
	defer server.Close()
	client := NewWithClient("ignored", "token", server.Client())
	client.Host = server.URL
	did, err := client.CreateRepo(context.Background(), CreateRepoInput{Name: "Repo", Source: "did:plc:source"})
	if err != nil || did != "did:plc:repo" {
		t.Fatalf("did=%s err=%v", did, err)
	}
	if err := client.SetDefaultBranch(context.Background(), SetDefaultBranchInput{Repo: did, DefaultBranch: "main"}); err != nil {
		t.Fatal(err)
	}
}
func TestDefaultBranchUsesRefsSnapshot(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("collection") != "org.tangled.git.refs" || r.URL.Query().Get("rkey") != "self" {
			t.Errorf("query=%s", r.URL)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"value": map[string]any{"defaultBranch": "main", "refs": map[string]string{"$bytes": base64.RawStdEncoding.EncodeToString([]byte(strings.Repeat("a", 40) + " refs/heads/other\n" + strings.Repeat("b", 64) + " refs/heads/main\n"))}}})
	}))
	defer server.Close()
	client := &Client{APIClient: &atclient.APIClient{Host: server.URL, Client: server.Client()}}
	branch, err := client.GetDefaultBranch(context.Background(), "did:plc:repo")
	if err != nil || branch.Name != "main" || branch.Hash != strings.Repeat("b", 64) {
		t.Fatalf("branch=%+v err=%v", branch, err)
	}
}

func TestDefaultBranchRejectsMalformedSnapshots(t *testing.T) {
	for _, value := range []string{
		`{"defaultBranch":"main","refs":"plain-base64-is-not-lexicon-bytes"}`,
		`{"defaultBranch":"main"}`,
		`{"defaultBranch":"main","refs":{"$bytes":"!bad!"}}`,
		`{"refs":{"$bytes":""}}`,
	} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"value":%s}`, value)
		}))
		client := &Client{APIClient: &atclient.APIClient{Host: server.URL, Client: server.Client()}}
		if _, err := client.GetDefaultBranch(context.Background(), "did:plc:repo"); err == nil {
			t.Errorf("accepted malformed snapshot=%s", value)
		}
		server.Close()
	}
}
