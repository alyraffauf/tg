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
	"sync"
	"testing"

	"github.com/alyraffauf/tg/atproto"
	"github.com/alyraffauf/tg/internal/gitutil"
	"github.com/alyraffauf/tg/internal/tangledlex"
	"github.com/alyraffauf/tg/knot"
	"github.com/alyraffauf/tg/tangled"
	"github.com/bluesky-social/indigo/atproto/atclient"
	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/atproto/syntax"
	lexutil "github.com/bluesky-social/indigo/lex/util"
	"github.com/ipfs/go-cid"
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
				KnotHost: "knot.example", SSHPort: 2222, Name: "example", PushPath: ".", RemoteName: "origin",
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
			if git.pushes[0].KnotHost != "knot.example" || git.pushes[0].SSHPort != 2222 {
				t.Fatalf("git push destination = %+v", git.pushes[0])
			}
		})
	}
}

func TestDeleteRepoRestoresRecordWhenKnotDeleteFails(t *testing.T) {
	pds := &testPDS{record: &atproto.GetRecordOutput{Value: map[string]any{"$type": repoCollection, "knot": "knot.example", "createdAt": "2026-07-25T12:00:00Z"}}}
	knotClient := &testKnot{deleteErr: errors.New("knot unavailable")}
	service := testService(pds, &testGit{}, knotClient)
	service.appview = testAppview{repo: &tangled.Repo{
		URI:   "at://did:plc:owner/sh.tangled.repo/example",
		Value: tangledlex.Repo{Knot: "knot.example"},
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
		Value: tangledlex.Repo{Knot: "knot.example", RepoDid: optionalString("did:plc:source-repo")},
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
	patchServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write(gzipContents(t, []byte("patch")))
	}))
	defer patchServer.Close()
	service := testService(pds, &testGit{}, knotClient)
	service.resolver = testResolver{
		identity: &identity.Identity{DID: syntax.DID("did:plc:owner")},
		pdsURL:   patchServer.URL,
	}
	service.httpClient = patchServer.Client()
	patchCID := cid.MustParse("bafybeigdyrzt5m6b5nkn55vsgzzfw5cfs2tidw6zqugycdkyybf2z7kz4q")
	pullRecord := tangledlex.RepoPull{
		Title: "Example", CreatedAt: "2026-07-29T00:00:00Z",
		Target: &tangledlex.RepoPull_Target{Repo: "did:plc:repo", Branch: "master"},
		Rounds: []*tangledlex.RepoPull_Round{{PatchBlob: &lexutil.LexBlob{Ref: lexutil.LexLink(patchCID)}}},
	}
	service.appview = testAppview{
		repo: &tangled.Repo{
			URI:   "at://did:plc:owner/sh.tangled.repo/example",
			Value: tangledlex.Repo{Name: optionalString("example"), Knot: "knot.example", RepoDid: optionalString("did:plc:repo")},
		},
		pulls: &tangled.List{Items: []tangled.ListItem{{
			URI: "at://did:plc:owner/sh.tangled.repo.pull/pr-1",
			Value: func() json.RawMessage {
				value, err := json.Marshal(pullRecord)
				if err != nil {
					t.Fatal(err)
				}
				return value
			}(),
		}}},
	}

	_, err := service.MergePull(context.Background(), Target{Handle: "owner.test", Repo: "example"}, "pr-1")
	if err == nil || !strings.Contains(err.Error(), "record merged pull request status") {
		t.Fatalf("MergePull() error = %v", err)
	}
	if knotClient.mergeCalls != 1 {
		t.Fatalf("MergePull() calls = %d, want 1", knotClient.mergeCalls)
	}
	if knotClient.mergeInput.DID != "did:plc:owner" || knotClient.mergeInput.Name != "example" || knotClient.mergeInput.Repo != "did:plc:repo" || knotClient.mergeInput.Branch != "master" || knotClient.mergeInput.Patch != "patch" {
		t.Fatalf("MergeInput = %+v", knotClient.mergeInput)
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
		resolver:              resolver,
		sessions:              testSessions{pds: pds},
		git:                   git,
		knot:                  &testKnotFactory{client: knotClient},
		knotOwnershipVerifier: &testKnotOwnershipVerifier{},
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
	pds           pdsClient
	api           *atclient.APIClient
	hasScope      bool
	isOAuth       bool
	oauthScopeErr error
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

func (s testSessions) OAuthSessionHasScope(context.Context, string) (bool, bool, error) {
	return s.hasScope, s.isOAuth, s.oauthScopeErr
}

type testPDS struct {
	puts                      []atproto.PutRecordInput
	deletes                   []atproto.DeleteRecordInput
	record                    *atproto.GetRecordOutput
	records                   []atproto.RecordItem
	putErr                    error
	uploadBlob                *atproto.Blob
	uploadErr                 error
	listErr                   error
	listCalls                 int
	listOptions               []atproto.ListRecordsOpts
	serviceAuthCalls          int
	serviceAuthAudiences      []string
	serviceAuthLexiconMethods []string
}

func (p *testPDS) PutRecord(_ context.Context, input atproto.PutRecordInput) (string, string, error) {
	p.puts = append(p.puts, input)
	if p.putErr != nil {
		return "", "", p.putErr
	}
	if strings.HasPrefix(input.Collection, "sh.tangled.") {
		if err := tangledlex.ValidateRecord(input.Collection, input.Record); err != nil {
			return "", "", err
		}
	}
	return fmt.Sprintf("at://%s/%s/%s", input.Repo, input.Collection, input.Rkey), "", nil
}

func (p *testPDS) DeleteRecord(_ context.Context, input atproto.DeleteRecordInput) error {
	p.deletes = append(p.deletes, input)
	return nil
}

func (p *testPDS) UploadBlob(context.Context, []byte, string) (*atproto.Blob, error) {
	if p.uploadErr != nil {
		return nil, p.uploadErr
	}
	if p.uploadBlob != nil {
		return p.uploadBlob, nil
	}
	return nil, errors.New("not implemented")
}

func (p *testPDS) GetRecord(context.Context, string, string, string) (*atproto.GetRecordOutput, error) {
	return p.record, nil
}

func (p *testPDS) ListRecords(_ context.Context, _, _ string, opts atproto.ListRecordsOpts) (*atproto.ListRecordsOutput, error) {
	p.listCalls++
	p.listOptions = append(p.listOptions, opts)
	if p.listErr != nil {
		return nil, p.listErr
	}
	records := p.records
	var cursor *string
	if opts.Limit > 0 && int64(len(records)) > opts.Limit {
		records = records[:opts.Limit]
	}
	if len(records) > 0 {
		lastRecordKey := "last-record"
		cursor = &lastRecordKey
	}
	return &atproto.ListRecordsOutput{Records: records, Cursor: cursor}, nil
}

func (p *testPDS) ListAllRecords(context.Context, string, string, atproto.ListRecordsOpts) ([]atproto.RecordItem, error) {
	p.listCalls++
	return p.records, p.listErr
}

func (p *testPDS) GetServiceAuth(_ context.Context, audience, lexiconMethod string) (string, error) {
	p.serviceAuthCalls++
	p.serviceAuthAudiences = append(p.serviceAuthAudiences, audience)
	p.serviceAuthLexiconMethods = append(p.serviceAuthLexiconMethods, lexiconMethod)
	return "token", nil
}

type testGit struct {
	branch         string
	commit         string
	patch          []byte
	patchErr       error
	clones         []gitutil.CloneRepoParams
	pushes         []gitutil.PushNewRepoParams
	repoCandidates []gitutil.RepoContext
	detectErr      error
}

func (g *testGit) CloneRepo(_ context.Context, input gitutil.CloneRepoParams) error {
	g.clones = append(g.clones, input)
	return nil
}

func (g *testGit) PushNewRepo(_ context.Context, input gitutil.PushNewRepoParams) error {
	g.pushes = append(g.pushes, input)
	return nil
}
func (g *testGit) CheckoutPatch(context.Context, gitutil.CheckoutPatchParams) error { return nil }
func (g *testGit) GeneratePatch(context.Context, string, string, string) ([]byte, error) {
	if g.patchErr != nil {
		return nil, g.patchErr
	}
	if g.patch != nil {
		return g.patch, nil
	}
	return nil, errors.New("not implemented")
}
func (g *testGit) CurrentBranch(context.Context, string) (string, error) { return g.branch, nil }
func (g *testGit) ResolveCommit(context.Context, string, string) (string, error) {
	return g.commit, nil
}
func (g *testGit) DefaultBranch(context.Context, string) (string, error) {
	return "", errors.New("not implemented")
}
func (g *testGit) DetectRepoCandidatesFromCWD(context.Context) ([]gitutil.RepoContext, error) {
	return g.repoCandidates, g.detectErr
}

type testKnotFactory struct {
	client knotClient
	hosts  []string
}

func (f *testKnotFactory) New(host string, _ string) knotClient {
	f.hosts = append(f.hosts, host)
	return f.client
}

func (f *testKnotFactory) NewPublic(host string) knotClient {
	f.hosts = append(f.hosts, host)
	return f.client
}

type testKnotOwnershipVerifier struct {
	errors map[string]error
	hosts  []string
	mu     sync.Mutex
}

func (v *testKnotOwnershipVerifier) Verify(_ context.Context, host, _ string) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.hosts = append(v.hosts, host)
	return v.errors[host]
}

type testKnot struct {
	setDefaultBranchErr error
	defaultBranch       *knot.DefaultBranch
	deleteErr           error
	deleteCalls         int
	mergeCalls          int
	mergeInput          knot.MergeInput
	createCalls         int
}

func (k *testKnot) CreateRepo(context.Context, knot.CreateRepoInput) (string, error) {
	k.createCalls++
	return "did:plc:repo", nil
}
func (k *testKnot) DeleteRepo(context.Context, knot.DeleteRepoInput) error {
	k.deleteCalls++
	return k.deleteErr
}
func (k *testKnot) SetDefaultBranch(context.Context, knot.SetDefaultBranchInput) error {
	return k.setDefaultBranchErr
}
func (k *testKnot) GetDefaultBranch(context.Context, string) (*knot.DefaultBranch, error) {
	if k.defaultBranch == nil {
		return &knot.DefaultBranch{Name: "main"}, nil
	}
	return k.defaultBranch, nil
}
func (k *testKnot) Merge(_ context.Context, input knot.MergeInput) error {
	k.mergeCalls++
	k.mergeInput = input
	return nil
}

type testAppview struct {
	repo   *tangled.Repo
	pulls  *tangled.List
	search *tangled.SearchResult
	stars  int64
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
func (a testAppview) Search(context.Context, string, int64) (*tangled.SearchResult, error) {
	if a.search == nil {
		return nil, errors.New("not implemented")
	}
	return a.search, nil
}
func (a testAppview) CountStars(context.Context, string) (int64, error) { return a.stars, nil }
