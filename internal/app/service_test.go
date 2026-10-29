package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/alyraffauf/tg/atproto"
	"github.com/alyraffauf/tg/internal/gitutil"
	"github.com/alyraffauf/tg/internal/tangledlex"
	"github.com/alyraffauf/tg/knot"
	"github.com/alyraffauf/tg/tangled"
	"github.com/bluesky-social/indigo/atproto/atclient"
	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

type testResolver struct {
	identity   *identity.Identity
	identities map[string]*identity.Identity
	pdsURL     string
}

func (r testResolver) ResolveHandle(context.Context, string) (*identity.Identity, error) {
	if r.identity != nil {
		return r.identity, nil
	}
	return repositoryIdentity("did:plc:owner", "owner.test", "https://knot.example"), nil
}

func (r testResolver) ResolveDID(_ context.Context, did string) (*identity.Identity, error) {
	if found := r.identities[did]; found != nil {
		return found, nil
	}
	if r.identity != nil {
		return r.identity, nil
	}
	return repositoryIdentity(did, "owner.test", "https://knot.example"), nil
}
func (r testResolver) ResolvePDS(context.Context, string) (string, error) {
	if r.pdsURL != "" {
		return r.pdsURL, nil
	}
	return "https://pds.example", nil
}

func TestListOptions(t *testing.T) {
	service := testService(nil, nil, nil)

	issueOptions, err := service.issueListOptions(context.Background(), ListOptions{
		Author: "alice.test",
		State:  "open",
		Limit:  20,
		Order:  "asc",
	})
	if err != nil {
		t.Fatalf("issueListOptions() error = %v", err)
	}
	want := tangled.ListOpts{Author: "did:plc:owner", State: "open", Limit: defaultListLimit, MaxItems: 20, Order: "asc"}
	if !reflect.DeepEqual(issueOptions, want) {
		t.Fatalf("issueListOptions() = %#v, want %#v", issueOptions, want)
	}

	pullOptions, err := service.pullListOptions(context.Background(), ListOptions{Author: "did:plc:alice", State: "merged"})
	if err != nil {
		t.Fatalf("pullListOptions() error = %v", err)
	}
	if pullOptions.Author != "did:plc:alice" || pullOptions.State != "merged" {
		t.Fatalf("pullListOptions() = %#v", pullOptions)
	}

	for _, options := range []ListOptions{{State: "merged"}, {Order: "newest"}, {Limit: -1}} {
		if _, err := service.issueListOptions(context.Background(), options); err == nil {
			t.Fatalf("issueListOptions(%#v) succeeded, want validation error", options)
		}
	}
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
	getRecordHook             func(string, string) (*atproto.GetRecordOutput, error)
	claims                    map[string]string
	puts                      []atproto.PutRecordInput
	deletes                   []atproto.DeleteRecordInput
	record                    *atproto.GetRecordOutput
	getErr                    error
	records                   []atproto.RecordItem
	putErr                    error
	putHook                   func(context.Context)
	uploadBlob                *atproto.Blob
	uploadErr                 error
	listErr                   error
	listCalls                 int
	listOptions               []atproto.ListRecordsOpts
	getCalls                  int
	serviceAuthCalls          int
	serviceAuthAudiences      []string
	serviceAuthLexiconMethods []string
	serviceAuthErr            error
}

func (p *testPDS) PutRecord(ctx context.Context, input atproto.PutRecordInput) (string, string, error) {
	if p.putHook != nil {
		p.putHook(ctx)
	}
	p.puts = append(p.puts, input)
	if p.putErr != nil {
		return "", "", p.putErr
	}
	if strings.HasPrefix(input.Collection, "org.tangled.") {
		if err := tangledlex.ValidateRecord(input.Collection, input.Record); err != nil {
			return "", "", err
		}
	}
	return fmt.Sprintf("at://%s/%s/%s", input.Repo, input.Collection, input.Rkey), testCID, nil
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

func (p *testPDS) GetRecord(_ context.Context, _ string, collection, key string) (*atproto.GetRecordOutput, error) {
	p.getCalls++
	if p.getRecordHook != nil {
		return p.getRecordHook(collection, key)
	}
	if p.getErr != nil {
		return nil, p.getErr
	}
	if p.record != nil {
		return p.record, nil
	}
	return &atproto.GetRecordOutput{CID: cidPointer(), Value: map[string]any{"$type": tangled.DeclarationCollection, "repo": func() string {
		if did := p.claims[key]; did != "" {
			return did
		}
		return "did:plc:repo"
	}(), "createdAt": "2026-10-01T00:00:00Z"}}, nil
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
	return "token", p.serviceAuthErr
}

type testGit struct {
	branch         string
	commit         string
	patch          []byte
	patchErr       error
	patchBase      string
	patchHead      string
	pullBases      map[string]gitutil.PullBase
	pullBaseInputs []string
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
func (g *testGit) GeneratePatch(_ context.Context, _, base, head string) ([]byte, error) {
	g.patchBase = base
	g.patchHead = head
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
func (g *testGit) ResolvePullBase(_ context.Context, _, base string) (gitutil.PullBase, error) {
	g.pullBaseInputs = append(g.pullBaseInputs, base)
	resolved, found := g.pullBases[base]
	if !found {
		return gitutil.PullBase{}, errors.New("not implemented")
	}
	return resolved, nil
}
func (g *testGit) DefaultPullBase(context.Context, string) (gitutil.PullBase, error) {
	resolved, found := g.pullBases[""]
	if !found {
		return gitutil.PullBase{}, errors.New("not implemented")
	}
	return resolved, nil
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

type testKnot struct {
	importStatus          *knot.ImportStatus
	importErr             error
	importCalls           int
	setDefaultBranchErr   error
	setDefaultBranchInput knot.SetDefaultBranchInput
	defaultBranch         *knot.DefaultBranch
	deleteErr             error
	deleteCalls           int
	deleteInput           knot.DeleteRepoInput
	mergeCalls            int
	mergeErr              error
	procedureErr          error
	procedureInputs       []map[string]any
	procedureMethods      []string
	records               map[string]*atproto.GetRecordOutput
	recordErr             error
	capabilities          []string
	legacy                bool
	probeErr              error
	receipt               knot.Receipt
	createInput           knot.CreateRepoInput
	createCalls           int
	createErr             error
}

func (k *testKnot) CreateRepo(_ context.Context, input knot.CreateRepoInput) (string, error) {
	k.createCalls++
	k.createInput = input
	return "did:plc:repo", k.createErr
}
func (k *testKnot) DeleteRepo(_ context.Context, input knot.DeleteRepoInput) error {
	k.deleteCalls++
	k.deleteInput = input
	return k.deleteErr
}
func (k *testKnot) SetDefaultBranch(_ context.Context, input knot.SetDefaultBranchInput) error {
	k.setDefaultBranchInput = input
	return k.setDefaultBranchErr
}
func (k *testKnot) GetDefaultBranch(context.Context, string) (*knot.DefaultBranch, error) {
	if k.defaultBranch == nil {
		return &knot.DefaultBranch{Name: "main"}, nil
	}
	return k.defaultBranch, nil
}
func (k *testKnot) DescribeKnot(context.Context) (*knot.Description, error) {
	caps := k.capabilities
	if caps == nil {
		caps = []string{"knot-social-writes"}
	}
	return &knot.Description{DID: "did:web:knot.example", Capabilities: caps, Legacy: k.legacy}, k.probeErr
}
func (k *testKnot) GetRecord(_ context.Context, repo, collection, rkey string) (*atproto.GetRecordOutput, error) {
	if k.recordErr != nil {
		return nil, k.recordErr
	}
	if record := k.records[collection+"/"+rkey]; record != nil {
		return record, nil
	}
	return &atproto.GetRecordOutput{URI: "at://" + repo + "/" + collection + "/" + rkey, CID: cidPointer(), Value: map[string]any{"$type": collection, "title": map[string]any{"$type": "org.tangled.markup.markdown#inline", "text": "Original"}, "state": "open", "createdAt": "2026-10-01T00:00:00Z"}}, nil
}
func (k *testKnot) Procedure(_ context.Context, method string, input, out any) error {
	data, _ := json.Marshal(input)
	var body map[string]any
	_ = json.Unmarshal(data, &body)
	k.procedureInputs = append(k.procedureInputs, body)
	k.procedureMethods = append(k.procedureMethods, method)
	if k.procedureErr != nil {
		return k.procedureErr
	}
	if method == "org.tangled.repo.mintToken" {
		return json.Unmarshal([]byte(`{"token":"repo-token"}`), out)
	}
	receipt := k.receipt
	if receipt.URI == "" {
		collection, _ := body["collection"].(string)
		if method == "org.tangled.review.submitPatch" {
			collection = "org.tangled.review.patch"
		}
		key, _ := body["rkey"].(string)
		if key == "" {
			key = "3m2zq4c6pmk2a"
		}
		receipt = knot.Receipt{URI: "at://did:plc:repo/" + collection + "/" + key, CID: testCID, Change: strings.Repeat("a", 40)}
	}
	data, _ = json.Marshal(receipt)
	if out != nil {
		return json.Unmarshal(data, out)
	}
	return nil
}

type testAppview struct {
	awaitErr         error
	formatHook       func(string, string, string)
	formatErr        error
	patches          []tangled.Patch
	ref              string
	refErr           error
	mergeErr         error
	mergeHook        func(string, any)
	ticket           *tangled.ListItem
	repo             *tangled.Repo
	repos            map[string]*tangled.Repo
	repoErr          error
	getRepoHook      func(string)
	getRepoByDIDHook func(string)
	repoByDID        *tangled.Repo
	repoByDIDErr     error
	issues           *tangled.List
	pulls            *tangled.List
	search           *tangled.SearchResult
	stars            int64
}

func (a testAppview) GetRepo(_ context.Context, uri string) (*tangled.Repo, error) {
	if a.getRepoHook != nil {
		a.getRepoHook(uri)
	}
	if repo, found := a.repos[uri]; found {
		return repo, nil
	}
	return normalizedTestRepo(a.repo), a.repoErr
}
func (a testAppview) GetRepoByDID(_ context.Context, repoDID string) (*tangled.Repo, error) {
	if a.getRepoByDIDHook != nil {
		a.getRepoByDIDHook(repoDID)
	}
	if a.repoByDID != nil || a.repoByDIDErr != nil {
		return normalizedTestRepo(a.repoByDID), a.repoByDIDErr
	}
	return normalizedTestRepo(a.repo), a.repoErr
}
func (testAppview) ListRepos(context.Context, string) (*tangled.RepoList, error) {
	return nil, errors.New("not implemented")
}
func (a testAppview) ListIssues(context.Context, string, tangled.ListOpts) (*tangled.List, error) {
	if a.issues == nil {
		return nil, errors.New("not implemented")
	}
	return a.issues, nil
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

const testCID = "bafybeigdyrzt5m6b5nkn55vsgzzfw5cfs2tidw6zqugycdkyybf2z7kz4q"

func cidPointer() *syntax.CID { value := syntax.CID(testCID); return &value }
func (p *testPDS) CreateRecord(ctx context.Context, in atproto.PutRecordInput) (string, string, error) {
	return p.PutRecord(ctx, in)
}
func (g *testGit) MergeBase(context.Context, string, string, string) (string, error) {
	return strings.Repeat("b", 40), nil
}
func (a testAppview) AwaitRecord(context.Context, string, string) error { return a.awaitErr }
func (a testAppview) ListPatches(context.Context, string) ([]tangled.Patch, error) {
	if a.patches != nil {
		return a.patches, nil
	}
	patch := tangled.Patch{URI: "at://did:plc:repo/org.tangled.review.patch/" + testCID, CID: testCID}
	patch.Value.Base.OID = strings.Repeat("b", 40)
	patch.Value.Head.OID = strings.Repeat("a", 40)
	return []tangled.Patch{patch}, nil
}
func (a testAppview) FormatPatch(_ context.Context, repo, base, head string) ([]byte, error) {
	if a.formatHook != nil {
		a.formatHook(repo, base, head)
	}
	if a.formatErr != nil {
		return nil, a.formatErr
	}
	return []byte("diff --git a/file b/file\n"), nil
}
func (a testAppview) GetRef(context.Context, string, string) (string, error) {
	if a.refErr != nil {
		return "", a.refErr
	}
	if a.ref != "" {
		return a.ref, nil
	}
	return strings.Repeat("a", 40), nil
}
func (a testAppview) Merge(_ context.Context, token string, input any) error {
	if a.mergeHook != nil {
		a.mergeHook(token, input)
	}
	return a.mergeErr
}
func (a testAppview) GetTicket(context.Context, string) (*tangled.ListItem, error) {
	if a.ticket != nil {
		return a.ticket, nil
	}
	return nil, errors.New("not implemented")
}
func normalizedTestRepo(repo *tangled.Repo) *tangled.Repo {
	if repo == nil {
		return nil
	}
	copy := *repo
	if copy.OwnerDID == "" {
		copy.OwnerDID = "did:plc:owner"
	}
	if copy.Value.Name == nil {
		copy.Value.Name = optionalString("example")
	}
	if copy.Value.RepoDid == nil {
		copy.Value.RepoDid = optionalString("did:plc:repo")
	}
	if copy.Value.Knot == "" {
		copy.Value.Knot = "knot.example"
	}
	return &copy
}
func repositoryIdentity(did, handle, host string) *identity.Identity {
	return &identity.Identity{DID: syntax.DID(did), Handle: syntax.Handle(handle), Services: map[string]identity.ServiceEndpoint{"atproto_pds": {Type: "AtprotoPersonalDataServer", URL: host}, "tangled_knot": {Type: "TangledKnot", URL: host + "/repo/tag"}}}
}
func testService(pds *testPDS, git *testGit, client *testKnot) *Service {
	if pds == nil {
		pds = &testPDS{}
	}
	if git == nil {
		git = &testGit{}
	}
	if client == nil {
		client = &testKnot{}
	}
	return &Service{resolver: testResolver{}, sessions: testSessions{pds: pds}, git: git, knot: &testKnotFactory{client: client}, appview: testAppview{repo: &tangled.Repo{URI: "at://did:plc:repo/org.tangled.repo.manifest/self", Value: tangled.RepoMetadata{RepoDid: optionalString("did:plc:repo"), Knot: "knot.example"}}}, attestationDirectory: ""}
}

func (s testSessions) CurrentDID(context.Context) (string, error) { return "did:plc:owner", nil }

func (k *testKnot) GetImportStatus(context.Context, string) (*knot.ImportStatus, error) {
	k.importCalls++
	if k.importStatus == nil {
		return &knot.ImportStatus{Status: "present"}, k.importErr
	}
	return k.importStatus, k.importErr
}
