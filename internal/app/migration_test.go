package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alyraffauf/tg/atproto"
	"github.com/alyraffauf/tg/internal/gitutil"
	"github.com/alyraffauf/tg/tangled"
	"github.com/bluesky-social/indigo/atproto/atclient"
)

func migrationService(t *testing.T) (*Service, *testPDS, *testKnot) {
	t.Helper()
	pds := &testPDS{}
	origin := &testKnot{}
	service := testService(pds, &testGit{branch: "feature", commit: strings.Repeat("a", 40), pullBases: map[string]gitutil.PullBase{"": {Revision: strings.Repeat("b", 40), Branch: "main"}, "origin/main": {Revision: strings.Repeat("b", 40), Branch: "main"}}}, origin)
	service.attestationDirectory = t.TempDir()
	return service, pds, origin
}
func migrationTarget() Target { return Target{Handle: "owner.test", Repo: "example"} }
func manifestValue() map[string]any {
	return map[string]any{"$type": tangled.RepoCollection, "declaration": map[string]any{"owner": "did:plc:owner", "name": "Example"}, "createdAt": "2026-10-01T00:00:00Z"}
}
func ticketList(pull bool) *tangled.List {
	branch := ""
	if pull {
		branch = "main"
	}
	data, _ := json.Marshal(ticketRecord("Example", "Body", branch))
	return &tangled.List{Items: []tangled.ListItem{{URI: "at://did:plc:repo/" + tangled.IssueCollection + "/3m2zq4c6pmk2a", CID: testCID, Value: data, Author: "did:plc:alice", State: "open"}}}
}

func TestResolveRepositorySeparatesGitAndRecordEndpoints(t *testing.T) {
	service, _, _ := migrationService(t)
	repo, err := service.resolveRepo(context.Background(), migrationTarget())
	if err != nil {
		t.Fatal(err)
	}
	if repo.RecordURL != "https://knot.example" || repo.GitURL != "https://knot.example/repo/tag" || repo.OwnerDID != "did:plc:owner" {
		t.Fatalf("resolved repository: %+v", repo)
	}
}
func TestResolveRepositoryRejectsBrokenOwnershipClaim(t *testing.T) {
	service, pds, _ := migrationService(t)
	pds.record = &atproto.GetRecordOutput{Value: map[string]any{"repo": "did:plc:someoneelse"}}
	if _, err := service.resolveVerifiedRepo(context.Background(), migrationTarget()); err == nil {
		t.Fatal("accepted inconsistent declaration")
	}
}
func TestSocialWritesBlockLegacyAndFailedProbes(t *testing.T) {
	for _, test := range []struct {
		name   string
		legacy bool
		probe  error
	}{{name: "legacy", legacy: true}, {name: "probe unavailable", probe: errors.New("network unavailable")}} {
		t.Run(test.name, func(t *testing.T) {
			service, pds, origin := migrationService(t)
			origin.legacy = test.legacy
			origin.probeErr = test.probe
			service.fosterKnot = "foster.example"
			if _, err := service.CreateIssue(context.Background(), migrationTarget(), "Title", "Body"); err == nil {
				t.Fatal("unsupported write accepted")
			}
			if len(origin.procedureMethods) != 0 || len(pds.puts) != 0 {
				t.Fatal("unsupported write reached an authority")
			}
		})
	}
}
func TestCreateIssueAttestsAndReportsSavedIndexingFailure(t *testing.T) {
	service, pds, origin := migrationService(t)
	view := service.appview.(testAppview)
	view.awaitErr = errors.New("NotSettled")
	service.appview = view
	result, err := service.CreateIssue(context.Background(), migrationTarget(), "Title", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Warnings) != 1 || result.URI == "" || len(origin.procedureMethods) != 1 {
		t.Fatalf("result=%+v methods=%v", result, origin.procedureMethods)
	}
	if pds.serviceAuthAudiences[0] != "did:plc:repo" || pds.serviceAuthLexiconMethods[0] != "com.atproto.repo.createRecord" {
		t.Fatalf("incorrect auth: %+v", pds)
	}
	if len(pds.puts) != 1 || pds.puts[0].Collection != tangled.AttestationCollection {
		t.Fatalf("attestations=%+v", pds.puts)
	}
	input := origin.procedureInputs[0]
	if _, present := input["rkey"]; present {
		t.Fatal("client tried to mint knot rkey")
	}
	record := input["record"].(map[string]any)
	if _, present := record["body"]; present {
		t.Fatal("blank markdown body should be omitted")
	}
}
func TestAttestationSurvivesRestartAndRetriesSameKey(t *testing.T) {
	service, pds, _ := migrationService(t)
	pds.putErr = errors.New("PDS offline")
	result, err := service.CreateIssue(context.Background(), migrationTarget(), "Title", "Body")
	if err != nil || len(result.Warnings) != 1 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	pending, err := service.ListPendingAttestations(context.Background())
	if err != nil || len(pending.Items) != 1 {
		t.Fatalf("pending=%+v err=%v", pending, err)
	}
	key := pending.Items[0].Rkey
	restored, _, _ := migrationService(t)
	restored.attestationDirectory = service.attestationDirectory
	restored.sessions = service.sessions
	pds.putErr = nil
	retry, err := restored.RetryPendingAttestations(context.Background())
	if err != nil || retry.Retried != 1 || retry.Pending != 0 {
		t.Fatalf("retry=%+v err=%v", retry, err)
	}
	if pds.puts[len(pds.puts)-1].Rkey != key {
		t.Fatal("retry minted another attestation key")
	}
	pending, err = restored.ListPendingAttestations(context.Background())
	if err != nil || len(pending.Items) != 0 {
		t.Fatalf("pending=%+v err=%v", pending, err)
	}
}
func TestAttestationQueueIsPrivateAndAccountSpecific(t *testing.T) {
	service, pds, _ := migrationService(t)
	pds.putErr = errors.New("offline")
	_, err := service.CreateIssue(context.Background(), migrationTarget(), "Title", "Body")
	if err != nil {
		t.Fatal(err)
	}
	dir, err := service.attestationDir("did:plc:owner")
	if err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("entries=%v err=%v", entries, err)
	}
	info, err := os.Stat(filepath.Join(dir, entries[0].Name()))
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("mode=%v err=%v", info, err)
	}
	other, _, err := service.pendingAttestations("did:plc:other")
	if err != nil || len(other) != 0 {
		t.Fatal("account queues overlap")
	}
}
func TestTicketEditUsesCASAndPreservesUnrelatedFields(t *testing.T) {
	service, _, origin := migrationService(t)
	value := map[string]any{"$type": tangled.IssueCollection, "title": map[string]string{"$type": "org.tangled.markup.markdown#inline", "text": "Original"}, "state": "open", "createdAt": "2026-10-01T00:00:00Z", "assignees": []string{"did:plc:alice"}}
	origin.records = map[string]*atproto.GetRecordOutput{tangled.IssueCollection + "/ticket": {CID: cidPointer(), Value: value}}
	title := "Updated"
	_, err := service.EditIssue(context.Background(), migrationTarget(), "ticket", &title, nil)
	if err != nil {
		t.Fatal(err)
	}
	input := origin.procedureInputs[0]
	if input["swapRecord"] != testCID {
		t.Fatalf("no CID guard: %+v", input)
	}
	record := input["record"].(map[string]any)
	if record["assignees"] == nil {
		t.Fatal("edit removed unrelated fields")
	}
	origin.procedureErr = &atclient.APIError{StatusCode: 409, Name: "InvalidSwap"}
	if _, err := service.EditIssue(context.Background(), migrationTarget(), "ticket", &title, nil); err == nil {
		t.Fatal("CAS conflict swallowed")
	}
}
func TestCreateAndUpdatePullUsePublishedCommitRanges(t *testing.T) {
	service, pds, origin := migrationService(t)
	result, err := service.CreatePull(context.Background(), CreatePullInput{Target: migrationTarget(), Title: "PR", Body: "Body"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Base != "main" || len(origin.procedureMethods) != 2 || origin.procedureMethods[1] != "org.tangled.review.submitPatch" {
		t.Fatalf("result=%+v methods=%v", result, origin.procedureMethods)
	}
	patch := origin.procedureInputs[1]["patch"].(map[string]any)
	if patch["sourceRepo"] != "did:plc:repo" || patch["head"].(map[string]any)["oid"] != strings.Repeat("a", 40) {
		t.Fatalf("patch=%+v", patch)
	}
	if pds.serviceAuthCalls != 2 {
		t.Fatalf("tokens reused: %d", pds.serviceAuthCalls)
	}
	view := service.appview.(testAppview)
	view.ticket = &ticketList(true).Items[0]
	service.appview = view
	if _, err := service.UpdatePullRound(context.Background(), UpdatePullRoundInput{Target: migrationTarget(), Rkey: "3m2zq4c6pmk2a"}); err != nil {
		t.Fatal(err)
	}
}
func TestCreatePullRejectsUnpublishedHeadBeforeCreatingTicket(t *testing.T) {
	service, _, origin := migrationService(t)
	view := service.appview.(testAppview)
	view.ref = strings.Repeat("c", 40)
	service.appview = view
	_, err := service.CreatePull(context.Background(), CreatePullInput{Target: migrationTarget(), Title: "PR"})
	if err == nil || !strings.Contains(err.Error(), "push that branch") {
		t.Fatalf("err=%v", err)
	}
	if len(origin.procedureMethods) != 0 {
		t.Fatal("created ticket for unpublished head")
	}
	if len(service.git.(*testGit).pushes) != 0 {
		t.Fatal("implicitly pushed")
	}
}
func TestMergeReportsGitSuccessWhenTicketCompletionFails(t *testing.T) {
	service, _, origin := migrationService(t)
	view := service.appview.(testAppview)
	view.ticket = &ticketList(true).Items[0]
	merged := false
	view.mergeHook = func(token string, input any) {
		merged = true
		body := input.(map[string]any)
		if body["style"] != "rebase" {
			t.Errorf("merge=%+v", body)
		}
	}
	service.appview = view
	origin.procedureErr = &atclient.APIError{StatusCode: 409, Name: "InvalidSwap"}
	result, err := service.MergePull(context.Background(), migrationTarget(), "3m2zq4c6pmk2a")
	if err != nil || !merged || !result.Merged || result.StatusRecorded || len(result.Warnings) != 1 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}
func TestRepositoryCreationUsesDeclarationAndSourceDID(t *testing.T) {
	service, pds, origin := migrationService(t)
	pds.getErr = &atclient.APIError{StatusCode: 404, Name: "RecordNotFound"}
	result, err := service.CreateRepo(context.Background(), CreateRepoInput{KnotHost: "knot.example", Name: "Example"})
	if err != nil {
		t.Fatal(err)
	}
	if result.URI != "at://did:plc:repo/org.tangled.repo.manifest/self" || origin.createInput.Name != "Example" {
		t.Fatalf("result=%+v create=%+v", result, origin.createInput)
	}
	if len(pds.puts) != 1 || pds.puts[0].Collection != tangled.DeclarationCollection {
		t.Fatalf("writes=%+v", pds.puts)
	}
}
func TestRepositoryCreationCleansUpOnlyDefiniteDeclarationRejection(t *testing.T) {
	for _, test := range []struct {
		name    string
		failure error
		cleanup int
	}{{"rejected", &atclient.APIError{StatusCode: 400, Name: "InvalidRequest"}, 1}, {"ambiguous", errors.New("connection lost"), 0}} {
		t.Run(test.name, func(t *testing.T) {
			service, pds, origin := migrationService(t)
			pds.getErr = &atclient.APIError{StatusCode: 404, Name: "RecordNotFound"}
			pds.putErr = test.failure
			_, err := service.CreateRepo(context.Background(), CreateRepoInput{Name: "Example"})
			if err == nil || origin.deleteCalls != test.cleanup {
				t.Fatalf("err=%v cleanup=%d", err, origin.deleteCalls)
			}
		})
	}
}
func TestRepositoryDeletionRestoresDeclarationOnRejection(t *testing.T) {
	service, pds, origin := migrationService(t)
	origin.deleteErr = &atclient.APIError{StatusCode: 403, Name: "AccessDenied"}
	if _, err := service.DeleteRepo(context.Background(), migrationTarget()); err == nil {
		t.Fatal("expected deletion failure")
	}
	if len(pds.deletes) != 1 || len(pds.puts) != 1 || pds.deletes[0].SwapRecord == nil {
		t.Fatalf("deletes=%+v restore=%+v", pds.deletes, pds.puts)
	}
}
func TestManifestEditPreservesPolicyAndUsesTopics(t *testing.T) {
	service, _, origin := migrationService(t)
	record := manifestValue()
	record["contributionPolicy"] = "members"
	origin.records = map[string]*atproto.GetRecordOutput{tangled.RepoCollection + "/self": {CID: cidPointer(), Value: record}}
	result, err := service.EditRepo(context.Background(), migrationTarget(), EditRepoInput{AddLabels: []string{"go"}})
	if err != nil {
		t.Fatal(err)
	}
	if result.URI == "" {
		t.Fatal("missing result")
	}
	saved := origin.procedureInputs[0]["record"].(map[string]any)
	if saved["contributionPolicy"] != "members" || saved["topics"] == nil {
		t.Fatalf("manifest=%+v", saved)
	}
}

func TestIssueAndPullCommentsUseCorrectStrongReferences(t *testing.T) {
	for _, pull := range []bool{false, true} {
		t.Run(fmt.Sprint(pull), func(t *testing.T) {
			service, _, origin := migrationService(t)
			view := service.appview.(testAppview)
			view.ticket = &ticketList(pull).Items[0]
			service.appview = view
			var err error
			if pull {
				_, err = service.CommentPull(context.Background(), migrationTarget(), "3m2zq4c6pmk2a", "Comment")
			} else {
				_, err = service.CommentIssue(context.Background(), migrationTarget(), "3m2zq4c6pmk2a", "Comment")
			}
			if err != nil {
				t.Fatal(err)
			}
			input := origin.procedureInputs[0]
			record := input["record"].(map[string]any)
			subject := record["subject"].(map[string]any)
			collection := tangled.IssueCollection
			if pull {
				collection = "org.tangled.review.patch"
			}
			if !strings.Contains(subject["uri"].(string), "/"+collection+"/") || subject["cid"] != testCID {
				t.Fatalf("subject=%+v", subject)
			}
			if record["pullRoundIdx"] != nil || record["body"].(map[string]any)["original"] != nil {
				t.Fatalf("legacy comment fields leaked: %+v", record)
			}
		})
	}
}
func TestForkPullUsesSourceDIDAndExplicitTargetBase(t *testing.T) {
	service, pds, origin := migrationService(t)
	pds.claims = map[string]string{"fork": "did:plc:fork"}
	view := service.appview.(testAppview)
	view.repos = map[string]*tangled.Repo{"at://did:plc:owner/org.tangled.repo.declaration/fork": {OwnerDID: "did:plc:owner", URI: "at://did:plc:fork/org.tangled.repo.manifest/self", Value: tangled.RepoMetadata{Name: optionalString("fork"), RepoDid: optionalString("did:plc:fork"), Knot: "knot.example"}}}
	service.appview = view
	source := Target{Handle: "owner.test", Repo: "fork"}
	service.git.(*testGit).pullBases["upstream/main"] = gitutil.PullBase{Revision: strings.Repeat("b", 40), Branch: "main"}
	if _, err := service.CreatePull(context.Background(), CreatePullInput{Target: migrationTarget(), Source: &source, Title: "Fork"}); err == nil || !strings.Contains(err.Error(), "require --base") {
		t.Fatalf("missing base: %v", err)
	}
	_, err := service.CreatePull(context.Background(), CreatePullInput{Target: migrationTarget(), Source: &source, Title: "Fork", Base: "upstream/main"})
	if err != nil {
		t.Fatal(err)
	}
	patch := origin.procedureInputs[1]["patch"].(map[string]any)
	if patch["sourceRepo"] != "did:plc:fork" {
		t.Fatalf("patch=%+v", patch)
	}
}
func TestSpindlePrefersRepositoryTokenAndPropagatesDenial(t *testing.T) {
	service, pds, origin := migrationService(t)
	origin.capabilities = []string{"knot-social-writes", "knot-repo-token-mint"}
	token, err := service.spindleToken(context.Background(), migrationTarget(), pds, "spindle.example", "did:plc:repo", "org.tangled.ci.cancelPipeline")
	if err != nil || token != "repo-token" {
		t.Fatalf("token=%s err=%v", token, err)
	}
	if pds.serviceAuthAudiences[0] != "did:plc:repo" || pds.serviceAuthLexiconMethods[0] != "org.tangled.repo.mintToken" || origin.procedureInputs[0]["aud"] != "did:web:spindle.example" {
		t.Fatalf("auth=%+v input=%+v", pds.serviceAuthAudiences, origin.procedureInputs)
	}
	origin.procedureErr = &atclient.APIError{StatusCode: 403, Name: "AccessDenied"}
	if _, err := service.spindleToken(context.Background(), migrationTarget(), pds, "spindle.example", "did:plc:repo", "org.tangled.ci.cancelPipeline"); err == nil {
		t.Fatal("knot authorization denial was hidden")
	}
}
func TestStringBlobRoundTripAndSSHKeySchema(t *testing.T) {
	service, pds, _ := migrationService(t)
	pds.uploadBlob = &atproto.Blob{Type: "blob", Ref: map[string]string{"$link": testCID}, MimeType: "text/plain", Size: 6}
	created, err := service.CreateString(context.Background(), CreateStringInput{Filename: "note.txt", Contents: "hello\n"})
	if err != nil {
		t.Fatal(err)
	}
	if created.CID != testCID {
		t.Fatalf("string receipt CID = %q, want %q", created.CID, testCID)
	}
	saved := pds.puts[0]
	if saved.Collection != tangled.StringCollection {
		t.Fatalf("collection=%s", saved.Collection)
	}
	pds.record = &atproto.GetRecordOutput{URI: created.URI, CID: cidPointer(), Value: saved.Record}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/xrpc/com.atproto.sync.getBlob" || r.URL.Query().Get("cid") != testCID {
			t.Errorf("request=%s", r.URL)
		}
		fmt.Fprint(w, "hello\n")
	}))
	defer server.Close()
	service.httpClient = server.Client()
	service.resolver = testResolver{pdsURL: server.URL}
	viewed, err := service.ViewString(context.Background(), "owner.test", created.Rkey)
	if err != nil || viewed.Contents != "hello\n" || viewed.Filename != "note.txt" {
		t.Fatalf("viewed=%+v err=%v", viewed, err)
	}
	if _, err := service.AddSSHKey(context.Background(), "laptop", "ssh-ed25519 AAAA"); err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(pds.puts[len(pds.puts)-1].Record)
	var key map[string]any
	_ = json.Unmarshal(data, &key)
	if key["$type"] != tangled.SSHKeyCollection || key["publicKey"] == nil || key["key"] != nil {
		t.Fatalf("key=%+v", key)
	}
}

func TestRepositoryRemoteVerificationIncludesHTTPSPort(t *testing.T) {
	for _, test := range []struct {
		endpoint  string
		remote    string
		wantError bool
	}{
		{"https://knot.example:8443", "knot.example:8443", false},
		{"https://knot.example:8443", "knot.example", true},
		{"https://knot.example:443", "knot.example", false},
	} {
		t.Run(test.endpoint+"/"+test.remote, func(t *testing.T) {
			service, _, _ := migrationService(t)
			service.resolver = testResolver{identity: repositoryIdentity("did:plc:repo", "owner.test", test.endpoint)}
			_, repo, err := service.resolveRepoDID(context.Background(), "did:plc:repo", test.remote)
			if (err != nil) != test.wantError {
				t.Fatalf("repo=%+v err=%v", repo, err)
			}
		})
	}
}

func TestResolveRepositoryRejectsDifferentNamedDeclaration(t *testing.T) {
	service, _, _ := migrationService(t)
	view := service.appview.(testAppview)
	view.repo.Value.Name = optionalString("another-repository")
	service.appview = view
	if _, err := service.resolveVerifiedRepo(context.Background(), migrationTarget()); err == nil {
		t.Fatal("accepted another repository's valid declaration")
	}
}

func TestSavedWriteBoundsFollowUpRequests(t *testing.T) {
	service, pds, _ := migrationService(t)
	pds.putHook = func(ctx context.Context) {
		if _, ok := ctx.Deadline(); !ok {
			t.Fatal("attestation request has no bounded deadline")
		}
	}
	if _, err := service.CreateIssue(context.Background(), migrationTarget(), "Title", ""); err != nil {
		t.Fatal(err)
	}
}

func TestCreatePullPreservesSourceMirrorFailure(t *testing.T) {
	service, _, origin := migrationService(t)
	view := service.appview.(testAppview)
	failure := errors.New("mirror offline")
	view.refErr = failure
	service.appview = view
	_, err := service.CreatePull(context.Background(), CreatePullInput{Target: migrationTarget(), Title: "PR"})
	if !errors.Is(err, failure) || len(origin.procedureMethods) != 0 {
		t.Fatalf("err=%v writes=%v", err, origin.procedureMethods)
	}
}

func TestMergeCompletesTicketWithCAS(t *testing.T) {
	service, _, origin := migrationService(t)
	view := service.appview.(testAppview)
	view.ticket = &ticketList(true).Items[0]
	service.appview = view
	origin.records = map[string]*atproto.GetRecordOutput{tangled.IssueCollection + "/3m2zq4c6pmk2a": {CID: cidPointer(), Value: ticketRecord("PR", "", "main")}}
	result, err := service.MergePull(context.Background(), migrationTarget(), "3m2zq4c6pmk2a")
	if err != nil || !result.Merged || !result.StatusRecorded {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	input := origin.procedureInputs[0]
	if input["swapRecord"] != testCID || input["record"].(map[string]any)["state"] != "completed" {
		t.Fatalf("completion=%+v", input)
	}
}

func TestExplicitCreatePolicyPreservesPendingAttestationWarning(t *testing.T) {
	service, pds, origin := migrationService(t)
	pds.getErr = &atclient.APIError{StatusCode: 404, Name: "RecordNotFound"}
	pds.putHook = func(_ context.Context) {
		if len(pds.puts) == 0 {
			pds.putErr = errors.New("PDS unavailable for attestation")
		} else {
			pds.putErr = nil
		}
	}
	origin.records = map[string]*atproto.GetRecordOutput{tangled.RepoCollection + "/self": {CID: cidPointer(), Value: manifestValue()}}
	result, err := service.CreateRepo(context.Background(), CreateRepoInput{Name: "Example", ContributionPolicy: "members"})
	if err != nil || len(result.Warnings) != 1 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if origin.procedureInputs[0]["record"].(map[string]any)["contributionPolicy"] != "members" {
		t.Fatal("explicit policy was not applied")
	}
	pending, err := service.ListPendingAttestations(context.Background())
	if err != nil || len(pending.Items) != 1 {
		t.Fatalf("pending=%+v err=%v", pending, err)
	}
}

func TestFosterReadsStopAfterOriginUpgradeAndNeverHideProbeFailures(t *testing.T) {
	for _, test := range []struct {
		name       string
		legacy     bool
		probeErr   error
		wantFoster bool
	}{
		{name: "legacy origin", legacy: true, wantFoster: true},
		{name: "upgraded origin"},
		{name: "unavailable origin", probeErr: errors.New("origin offline")},
	} {
		t.Run(test.name, func(t *testing.T) {
			service, _, origin := migrationService(t)
			service.fosterKnot = "foster.example"
			origin.legacy = test.legacy
			origin.probeErr = test.probeErr
			item := ticketList(false).Items[0]
			view := service.appview.(testAppview)
			view.ticket = &item
			service.appview = view
			origin.records = map[string]*atproto.GetRecordOutput{tangled.IssueCollection + "/3m2zq4c6pmk2a": {URI: item.URI, CID: cidPointer(), Value: ticketRecord("Foster title", "", "")}}
			result, err := service.ViewIssue(context.Background(), migrationTarget(), "3m2zq4c6pmk2a")
			if test.probeErr != nil {
				if !errors.Is(err, test.probeErr) {
					t.Fatalf("err=%v", err)
				}
			} else {
				if err != nil || result.Author.DID != "did:plc:alice" {
					t.Fatalf("result=%+v err=%v", result, err)
				}
				if (result.Title == "Foster title") != test.wantFoster {
					t.Fatalf("unexpected read source: %+v", result)
				}
			}
			usedFoster := false
			for _, host := range service.knot.(*testKnotFactory).hosts {
				usedFoster = usedFoster || host == "foster.example"
			}
			if usedFoster != test.wantFoster || len(origin.procedureMethods) != 0 {
				t.Fatalf("foster=%v writes=%v", usedFoster, origin.procedureMethods)
			}
		})
	}
}

func TestForkPullReadsAndMergesFromSourceRepository(t *testing.T) {
	service, _, origin := migrationService(t)
	view := service.appview.(testAppview)
	view.ticket = &ticketList(true).Items[0]
	var patch tangled.Patch
	if err := json.Unmarshal([]byte(`{"uri":"at://did:plc:repo/org.tangled.review.patch/patch","cid":"`+testCID+`","value":{"base":{"oid":"base"},"head":{"oid":"head"},"source":{"repo":"did:plc:fork","branch":"feature"}}}`), &patch); err != nil {
		t.Fatal(err)
	}
	view.patches = []tangled.Patch{patch}
	view.formatHook = func(repo, base, head string) {
		if repo != "did:plc:fork" || base != "base" || head != "head" {
			t.Errorf("patch repo=%s base=%s head=%s", repo, base, head)
		}
	}
	view.mergeHook = func(_ string, input any) {
		body := input.(map[string]any)
		if body["source"].(map[string]string)["repo"] != "did:plc:fork" || body["target"].(map[string]string)["repo"] != "did:plc:repo" {
			t.Errorf("merge=%+v", body)
		}
	}
	service.appview = view
	if _, err := service.PullPatch(context.Background(), migrationTarget(), "3m2zq4c6pmk2a"); err != nil {
		t.Fatal(err)
	}
	origin.records = map[string]*atproto.GetRecordOutput{tangled.IssueCollection + "/3m2zq4c6pmk2a": {CID: cidPointer(), Value: ticketRecord("PR", "", "main")}}
	result, err := service.MergePull(context.Background(), migrationTarget(), "3m2zq4c6pmk2a")
	if err != nil || !result.Merged || !result.StatusRecorded {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestMirrorLagPreservesErrorsAndDoesNotCompletePull(t *testing.T) {
	service, _, origin := migrationService(t)
	view := service.appview.(testAppview)
	view.ticket = &ticketList(true).Items[0]
	failure := &atclient.APIError{StatusCode: 404, Name: "RevisionNotFound"}
	view.formatErr, view.mergeErr = failure, failure
	service.appview = view
	_, diffErr := service.PullPatch(context.Background(), migrationTarget(), "3m2zq4c6pmk2a")
	_, mergeErr := service.MergePull(context.Background(), migrationTarget(), "3m2zq4c6pmk2a")
	for _, err := range []error{diffErr, mergeErr} {
		if !errors.Is(err, failure) || !strings.Contains(err.Error(), "mirror has not caught up") {
			t.Fatalf("err=%v", err)
		}
	}
	if len(origin.procedureMethods) != 0 {
		t.Fatalf("completed unmerged ticket: %v", origin.procedureMethods)
	}
}
