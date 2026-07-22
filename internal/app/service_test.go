package app

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alyraffauf/tg/atproto"
	"github.com/alyraffauf/tg/internal/gitutil"
	"github.com/alyraffauf/tg/knot"
	"github.com/alyraffauf/tg/tangled"
	"github.com/bluesky-social/indigo/atproto/atclient"
	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

func TestCreateRepoRecordsDefaultBranchOutcome(t *testing.T) {
	tests := []struct {
		name           string
		setBranchErr   error
		wantWarnings   bool
		wantDefaultRef string
	}{
		{name: "default branch set", wantDefaultRef: "main"},
		{name: "default branch warning", setBranchErr: errors.New("knot unavailable"), wantWarnings: true, wantDefaultRef: "main"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pds := &testPDS{}
			git := &testGit{branch: "main"}
			knotClient := &testKnot{setDefaultBranchErr: tt.setBranchErr}
			service := testService(pds, git, knotClient)

			result, err := service.CreateRepo(context.Background(), CreateRepoInput{
				KnotHost: "knot.example", Name: "example", PushPath: ".", RemoteName: "origin",
			})
			if err != nil {
				t.Fatalf("CreateRepo() error = %v", err)
			}
			if !result.Pushed || result.DefaultBranch != tt.wantDefaultRef {
				t.Fatalf("CreateRepo() result = %+v", result)
			}
			if got := len(result.Warnings) > 0; got != tt.wantWarnings {
				t.Fatalf("CreateRepo() warnings = %v, want warnings %t", result.Warnings, tt.wantWarnings)
			}
			if len(pds.puts) != 1 || pds.puts[0].Collection != repoCollection {
				t.Fatalf("repository record writes = %+v", pds.puts)
			}
			if len(git.pushes) != 1 {
				t.Fatalf("git pushes = %+v", git.pushes)
			}
		})
	}
}

func TestDeleteRepoRestoresRecordWhenKnotDeleteFails(t *testing.T) {
	pds := &testPDS{record: &atproto.GetRecordOutput{Value: map[string]any{"$type": repoCollection, "knot": "knot.example"}}}
	knotClient := &testKnot{deleteErr: errors.New("knot unavailable")}
	service := testService(pds, &testGit{}, knotClient)
	service.appview = testAppview{repo: &tangled.Repo{
		URI:   "at://did:plc:owner/sh.tangled.repo/example",
		Value: tangled.RepoRecord{Knot: "knot.example"},
	}}

	_, err := service.DeleteRepo(context.Background(), Target{Handle: "owner.test", Repo: "example"})
	if err == nil || err.Error() != "knot unavailable" {
		t.Fatalf("DeleteRepo() error = %v, want knot error", err)
	}
	if len(pds.deletes) != 1 {
		t.Fatalf("DeleteRepo() deletes = %+v", pds.deletes)
	}
	if len(pds.puts) != 1 || pds.puts[0].Collection != repoCollection {
		t.Fatalf("DeleteRepo() restores = %+v", pds.puts)
	}
}

func TestForkRepoCleansUpWhenRecordWriteFails(t *testing.T) {
	pds := &testPDS{putErr: errors.New("PDS unavailable")}
	knotClient := &testKnot{}
	service := testService(pds, &testGit{}, knotClient)
	service.appview = testAppview{repo: &tangled.Repo{
		URI:   "at://did:plc:source/sh.tangled.repo/source",
		Value: tangled.RepoRecord{Knot: "knot.example", RepoDid: "did:plc:source-repo"},
	}}

	_, err := service.ForkRepo(context.Background(), Target{Handle: "source.test", Repo: "source"}, "fork")
	if err == nil || !strings.Contains(err.Error(), "write fork record") {
		t.Fatalf("ForkRepo() error = %v", err)
	}
	if knotClient.deleteCalls != 1 {
		t.Fatalf("ForkRepo() orphan cleanup calls = %d, want 1", knotClient.deleteCalls)
	}
}

func TestMergePullReportsStatusWriteFailureAfterMerge(t *testing.T) {
	pds := &testPDS{putErr: errors.New("PDS unavailable")}
	knotClient := &testKnot{}
	service := testService(pds, &testGit{}, knotClient)
	service.appview = testAppview{
		repo: &tangled.Repo{
			URI:   "at://did:plc:owner/sh.tangled.repo/example",
			Value: tangled.RepoRecord{Knot: "knot.example", RepoDid: "did:plc:repo"},
		},
		pulls: &tangled.List{Items: []tangled.ListItem{{
			URI:   "at://did:plc:owner/sh.tangled.repo.pull/pr-1",
			Value: json.RawMessage(`{"title":"Example"}`),
		}}},
	}

	_, err := service.MergePull(context.Background(), Target{Handle: "owner.test", Repo: "example"}, "pr-1")
	if err == nil || !strings.Contains(err.Error(), "record merged pull request status") {
		t.Fatalf("MergePull() error = %v", err)
	}
	if knotClient.mergeCalls != 1 {
		t.Fatalf("MergePull() calls = %d, want 1", knotClient.mergeCalls)
	}
}

func TestDownloadPullPatch(t *testing.T) {
	patch := gzipContents(t, []byte("diff --git a/file b/file\n"))
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/xrpc/com.atproto.sync.getBlob" {
			t.Fatalf("request path = %q", request.URL.Path)
		}
		_, _ = writer.Write(patch)
	}))
	defer server.Close()

	service := testService(&testPDS{}, &testGit{}, &testKnot{})
	service.resolver = testResolver{identity: &identity.Identity{DID: syntax.DID("did:plc:owner")}, pdsURL: server.URL}
	service.httpClient = server.Client()

	contents, err := service.downloadPullPatch(context.Background(), "did:plc:owner", "bafycid")
	if err != nil {
		t.Fatalf("downloadPullPatch() error = %v", err)
	}
	if got, want := string(contents), "diff --git a/file b/file\n"; got != want {
		t.Fatalf("downloadPullPatch() = %q, want %q", got, want)
	}
}

func TestCallAPIValidatesEndpointBeforeAuthenticating(t *testing.T) {
	service := testService(&testPDS{}, &testGit{}, &testKnot{})
	_, err := service.CallAPI(context.Background(), APIRequestInput{Endpoint: "not an nsid", Method: http.MethodGet})
	if err == nil || !strings.HasPrefix(err.Error(), "parse NSID") {
		t.Fatalf("CallAPI() error = %v, want NSID validation error", err)
	}
}

func TestCallAPIPostsJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost || request.URL.Path != "/xrpc/com.example.test" {
			t.Fatalf("request = %s %s", request.Method, request.URL.Path)
		}
		body, err := io.ReadAll(request.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		if got, want := string(body), `{"message":"hello"}`; got != want {
			t.Fatalf("request body = %q, want %q", got, want)
		}
		writer.WriteHeader(http.StatusCreated)
		_, _ = writer.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	service := testService(&testPDS{}, &testGit{}, &testKnot{})
	service.sessions = testSessions{pds: &testPDS{}, api: &atclient.APIClient{Host: server.URL, Client: server.Client()}}

	response, err := service.CallAPI(context.Background(), APIRequestInput{
		Endpoint: "com.example.test", Method: http.MethodPost, Fields: map[string]any{"message": "hello"},
	})
	if err != nil {
		t.Fatalf("CallAPI() error = %v", err)
	}
	if response.StatusCode != http.StatusCreated || string(response.Body) != `{"ok":true}` {
		t.Fatalf("CallAPI() response = %+v", response)
	}
}

func gzipContents(t *testing.T, contents []byte) []byte {
	t.Helper()
	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	if _, err := writer.Write(contents); err != nil {
		t.Fatalf("write gzip: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close gzip: %v", err)
	}
	data, err := io.ReadAll(&compressed)
	if err != nil {
		t.Fatalf("read gzip: %v", err)
	}
	return data
}

func testService(pds *testPDS, git *testGit, knotClient *testKnot) *Service {
	resolver := testResolver{identity: &identity.Identity{
		DID:    syntax.DID("did:plc:owner"),
		Handle: syntax.Handle("owner.test"),
	}}
	return &Service{
		resolver: resolver,
		sessions: testSessions{pds: pds},
		git:      git,
		knot:     testKnotFactory{client: knotClient},
	}
}

type testResolver struct {
	identity *identity.Identity
	pdsURL   string
}

func (r testResolver) ResolveHandle(context.Context, string) (*identity.Identity, error) {
	return r.identity, nil
}

func (r testResolver) ResolveDID(context.Context, string) (*identity.Identity, error) {
	return r.identity, nil
}

func (r testResolver) ResolvePDS(context.Context, string) (string, error) {
	if r.pdsURL != "" {
		return r.pdsURL, nil
	}
	return "https://pds.example", nil
}

type testSessions struct {
	pds pdsClient
	api *atclient.APIClient
}

func (s testSessions) AuthenticatedPDS(context.Context) (pdsClient, string, error) {
	return s.pds, "did:plc:owner", nil
}

func (s testSessions) PublicPDS(context.Context, string) (pdsClient, string, error) {
	return s.pds, "did:plc:owner", nil
}

func (s testSessions) APIClient(context.Context) (*atclient.APIClient, error) {
	if s.api == nil {
		return nil, errors.New("not implemented")
	}
	return s.api, nil
}

type testPDS struct {
	puts    []atproto.PutRecordInput
	deletes []atproto.DeleteRecordInput
	record  *atproto.GetRecordOutput
	putErr  error
}

func (p *testPDS) PutRecord(_ context.Context, input atproto.PutRecordInput) (string, string, error) {
	p.puts = append(p.puts, input)
	if p.putErr != nil {
		return "", "", p.putErr
	}
	return fmt.Sprintf("at://%s/%s/%s", input.Repo, input.Collection, input.Rkey), "", nil
}

func (p *testPDS) DeleteRecord(_ context.Context, input atproto.DeleteRecordInput) error {
	p.deletes = append(p.deletes, input)
	return nil
}

func (p *testPDS) UploadBlob(context.Context, []byte, string) (*atproto.Blob, error) {
	return nil, errors.New("not implemented")
}

func (p *testPDS) GetRecord(context.Context, string, string, string) (*atproto.GetRecordOutput, error) {
	return p.record, nil
}

func (p *testPDS) ListAllRecords(context.Context, string, string, atproto.ListRecordsOpts) ([]atproto.RecordItem, error) {
	return nil, errors.New("not implemented")
}

func (p *testPDS) GetServiceAuth(context.Context, string, string) (string, error) {
	return "token", nil
}

type testGit struct {
	branch string
	pushes []gitutil.PushNewRepoParams
}

func (g *testGit) CloneRepo(context.Context, gitutil.CloneRepoParams) error { return nil }
func (g *testGit) PushNewRepo(_ context.Context, input gitutil.PushNewRepoParams) error {
	g.pushes = append(g.pushes, input)
	return nil
}
func (g *testGit) CheckoutPatch(context.Context, gitutil.CheckoutPatchParams) error { return nil }
func (g *testGit) GeneratePatch(context.Context, string, string, string) ([]byte, error) {
	return nil, errors.New("not implemented")
}
func (g *testGit) CurrentBranch(context.Context, string) (string, error) { return g.branch, nil }
func (g *testGit) DefaultBranch(context.Context, string) (string, error) {
	return "", errors.New("not implemented")
}
func (g *testGit) DetectRepoFromCWD(context.Context) (*gitutil.RepoContext, error) {
	return nil, errors.New("not implemented")
}

type testKnotFactory struct {
	client knotClient
}

func (f testKnotFactory) New(string, string) knotClient { return f.client }

type testKnot struct {
	setDefaultBranchErr error
	deleteErr           error
	deleteCalls         int
	mergeCalls          int
}

func (k *testKnot) CreateRepo(context.Context, knot.CreateRepoInput) (string, error) {
	return "did:plc:repo", nil
}
func (k *testKnot) DeleteRepo(context.Context, knot.DeleteRepoInput) error {
	k.deleteCalls++
	return k.deleteErr
}
func (k *testKnot) SetDefaultBranch(context.Context, knot.SetDefaultBranchInput) error {
	return k.setDefaultBranchErr
}
func (k *testKnot) Merge(context.Context, knot.MergeInput) error {
	k.mergeCalls++
	return nil
}

type testAppview struct {
	repo  *tangled.Repo
	pulls *tangled.List
}

func (a testAppview) GetRepo(context.Context, string) (*tangled.Repo, error) { return a.repo, nil }
func (testAppview) ListRepos(context.Context, string) (*tangled.RepoList, error) {
	return nil, errors.New("not implemented")
}
func (testAppview) ListIssues(context.Context, string, tangled.ListOpts) (*tangled.List, error) {
	return nil, errors.New("not implemented")
}
func (a testAppview) ListPulls(context.Context, string, tangled.ListOpts) (*tangled.List, error) {
	if a.pulls == nil {
		return nil, errors.New("not implemented")
	}
	return a.pulls, nil
}
