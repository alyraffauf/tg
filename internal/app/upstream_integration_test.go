//go:build upstream_integration

package app

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/alyraffauf/tg/atproto"
	"github.com/alyraffauf/tg/tangled"
	"github.com/bluesky-social/indigo/atproto/atclient"
	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"golang.org/x/crypto/ssh"
)

// These endpoints belong to upstream's disposable localinfra network namespace.
// Never substitute production endpoints: this test creates and edits real records.
const stackPDS = "https://pds.tngl.boltless.dev"

type stackBearer string

func (token stackBearer) DoWithAuth(client *http.Client, request *http.Request, _ syntax.NSID) (*http.Response, error) {
	request.Header.Set("Authorization", "Bearer "+string(token))
	return client.Do(request)
}

type stackSessions struct {
	pds      pdsClient
	api      *atclient.APIClient
	actor    string
	resolver identityResolver
	client   *http.Client
}

func (s stackSessions) AuthenticatedPDS(context.Context) (pdsClient, string, error) {
	return s.pds, s.actor, nil
}
func (s stackSessions) PublicPDS(ctx context.Context, actor string) (pdsClient, string, error) {
	return (productionSessions{resolver: s.resolver, httpClient: s.client}).PublicPDS(ctx, actor)
}
func (s stackSessions) CurrentDID(context.Context) (string, error)             { return s.actor, nil }
func (s stackSessions) APIClient(context.Context) (*atclient.APIClient, error) { return s.api, nil }
func (s stackSessions) OAuthSessionHasScope(context.Context, string) (bool, bool, error) {
	return false, false, nil
}

func upstreamService(t *testing.T) (*Service, context.Context, string) {
	t.Helper()
	if os.Getenv("TG_UPSTREAM_INTEGRATION") != "1" {
		t.Skip("requires upstream localinfra namespace and seeded Alice account")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(cancel)
	client := &http.Client{Timeout: 30 * time.Second}
	data := []byte(`{"identifier":"alice.pds.tngl.boltless.dev","password":"password"}`)
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, stackPDS+"/xrpc/com.atproto.server.createSession", bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		t.Fatalf("local fixture login returned HTTP %d", response.StatusCode)
	}
	var session struct {
		DID       string `json:"did"`
		AccessJWT string `json:"accessJwt"`
	}
	if err := json.NewDecoder(response.Body).Decode(&session); err != nil {
		t.Fatal(err)
	}
	if session.DID == "" || session.AccessJWT == "" {
		t.Fatal("local fixture omitted session identity")
	}
	pds := &atproto.ATProto{Client: &atclient.APIClient{Host: stackPDS, Client: client, Auth: stackBearer(session.AccessJWT)}}
	resolver := &atproto.Resolver{Directory: &identity.BaseDirectory{PLCURL: "https://plc.tngl.boltless.dev", HTTPClient: *client, PLCClient: client}}
	service := NewWithStreams("https://bobbin.tngl.boltless.dev", io.Discard, io.Discard)
	service.httpClient = client
	service.resolver = resolver
	service.sessions = stackSessions{pds: pds, api: pds.Client, actor: session.DID, resolver: resolver, client: client}
	service.attestationDirectory = t.TempDir()
	return service, ctx, session.DID
}

func TestUpstreamNativeRecordRoundTrip(t *testing.T) {
	service, ctx, actor := upstreamService(t)
	name := fmt.Sprintf("tg-verify-%d", time.Now().UnixNano())
	created, err := service.CreateRepo(ctx, CreateRepoInput{Name: name, KnotHost: "knot2.tngl.boltless.dev", ContributionPolicy: "members"})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("created disposable repository %s", created.URI)
	target := Target{Handle: actor, Repo: name}
	manifest, err := service.knot.NewPublic("knot2.tngl.boltless.dev").GetRecord(ctx, strings.Split(created.URI, "/")[2], tangled.RepoCollection, "self")
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(manifest.Value)
	if err != nil {
		t.Fatal(err)
	}
	var policy tangled.Manifest
	if err := json.Unmarshal(data, &policy); err != nil {
		t.Fatal(err)
	}
	if policy.ContributionPolicy != "members" {
		t.Fatalf("explicit policy was not retained: %+v", policy)
	}
	description := "Updated description"
	edited, err := service.EditRepo(ctx, target, EditRepoInput{Description: &description, AddLabels: []string{"verification"}})
	if err != nil || len(edited.Warnings) != 0 {
		t.Fatalf("repository edit=%+v err=%v", edited, err)
	}
	repo, err := service.ViewRepo(ctx, target)
	if err != nil || repo.Description != description {
		t.Fatalf("repository=%+v err=%v", repo, err)
	}
	repos, err := service.ListRepos(ctx, "alice.pds.tngl.boltless.dev")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, item := range repos.Items {
		if item.URI == created.URI {
			found = true
		}
	}
	if !found {
		t.Fatal("created repository missing from owner listing")
	}
	deadline := time.Now().Add(10 * time.Second)
	for {
		matches, err := service.SearchRepos(ctx, `"`+name+`"`, 10)
		if err != nil {
			t.Fatal(err)
		}
		found = false
		for _, item := range matches.Items {
			if item.URI == created.URI {
				found = true
			}
		}
		if found {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("created repository missing from search")
		}
		select {
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		case <-time.After(200 * time.Millisecond):
		}
	}
	issue, err := service.CreateIssue(ctx, target, "Original title", "Original body")
	if err != nil {
		t.Fatal(err)
	}
	if issue.CID == "" || issue.URI == "" || len(issue.Warnings) != 0 {
		t.Fatalf("issue receipt or indexing failed: %+v", issue)
	}
	viewed, err := service.ViewIssue(ctx, target, issue.Rkey)
	if err != nil {
		t.Fatal(err)
	}
	if viewed.URI != issue.URI || viewed.CID != issue.CID || viewed.Title != "Original title" {
		t.Fatalf("view does not match receipt: %+v", viewed)
	}
	resolved, err := service.resolveRepo(ctx, target)
	if err != nil {
		t.Fatal(err)
	}
	ticket := ticketRecord("Original title", "Original body", "")
	ticket.CreatedAt = viewed.CreatedAt
	record := struct {
		tangled.Ticket
		Assignees []string `json:"assignees"`
	}{Ticket: ticket, Assignees: []string{actor}}
	assigned, warnings, err := service.socialProcedure(ctx, resolved, "com.atproto.repo.putRecord", map[string]any{"collection": "org.tangled.track.ticket", "rkey": issue.Rkey, "swapRecord": viewed.CID, "record": record})
	if err != nil || len(warnings) != 0 {
		t.Fatalf("assignment=%+v warnings=%v err=%v", assigned, warnings, err)
	}
	viewed.CID = assigned.CID
	title := "Updated title"
	updated, err := service.EditIssue(ctx, target, issue.Rkey, &title, nil, viewed.CID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.URI != issue.URI || updated.CID == issue.CID || len(updated.Warnings) != 0 {
		t.Fatalf("edit receipt failed: %+v", updated)
	}
	stored, err := service.knot.NewPublic("knot2.tngl.boltless.dev").GetRecord(ctx, repo.RepoDid, "org.tangled.track.ticket", issue.Rkey)
	if err != nil {
		t.Fatal(err)
	}
	data, err = json.Marshal(stored.Value)
	if err != nil {
		t.Fatal(err)
	}
	var preserved struct {
		Assignees []string `json:"assignees"`
	}
	if err := json.Unmarshal(data, &preserved); err != nil {
		t.Fatal(err)
	}
	if len(preserved.Assignees) != 1 || preserved.Assignees[0] != actor {
		t.Fatalf("edit lost assignees: %+v", preserved)
	}
	if _, err := service.EditIssue(ctx, target, issue.Rkey, &title, nil, issue.CID); err == nil {
		t.Fatal("upstream accepted stale edit")
	}
	comment, err := service.CommentIssue(ctx, target, issue.Rkey, "Verified comment")
	if err != nil || comment.CID == "" || len(comment.Warnings) != 0 {
		t.Fatalf("comment=%+v err=%v", comment, err)
	}
	closed, err := service.SetIssueState(ctx, target, issue.Rkey, "closed", updated.CID)
	if err != nil || closed.CID == "" || len(closed.Warnings) != 0 {
		t.Fatalf("close=%+v err=%v", closed, err)
	}
	reopened, err := service.SetIssueState(ctx, target, issue.Rkey, "open", closed.CID)
	if err != nil || reopened.CID == "" || len(reopened.Warnings) != 0 {
		t.Fatalf("reopen=%+v err=%v", reopened, err)
	}
	listed, err := service.ListIssues(ctx, target, ListOptions{State: "open", Author: actor})
	if err != nil || len(listed.Items) != 1 || listed.Items[0].Title != title || len(listed.Warnings) != 0 {
		t.Fatalf("issues=%+v err=%v", listed, err)
	}
	pending, err := service.ListPendingAttestations(ctx)
	if err != nil || len(pending.Items) != 0 {
		t.Fatalf("attestations=%+v err=%v", pending, err)
	}
	// Keep failures available for inspection, but clean up a fully verified run.
	if _, err := service.DeleteRepo(ctx, target); err != nil {
		t.Fatal(err)
	}
}

func TestUpstreamAccountRecords(t *testing.T) {
	service, ctx, actor := upstreamService(t)
	created, err := service.CreateString(ctx, CreateStringInput{Filename: "verified.txt", Description: "Disposable verification", Contents: "Verified blob\n"})
	if err != nil || created.CID == "" || len(created.Warnings) != 0 {
		t.Fatalf("string=%+v err=%v", created, err)
	}
	viewed, err := service.ViewString(ctx, actor, created.Rkey)
	if err != nil || viewed.Contents != "Verified blob\n" || viewed.Filename != "verified.txt" {
		t.Fatalf("string view=%+v err=%v", viewed, err)
	}
	listed, err := service.ListStrings(ctx, actor)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, item := range listed.Items {
		if item.URI == created.URI {
			found = true
		}
	}
	if !found {
		t.Fatal("created string missing from listing")
	}
	if _, err := service.DeleteString(ctx, created.Rkey); err != nil {
		t.Fatal(err)
	}
	public, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	key, err := ssh.NewPublicKey(public)
	if err != nil {
		t.Fatal(err)
	}
	keyText := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(key)))
	added, err := service.AddSSHKey(ctx, "tg verification", keyText)
	if err != nil {
		t.Fatal(err)
	}
	keys, err := service.ListSSHKeys(ctx, actor)
	if err != nil {
		t.Fatal(err)
	}
	found = false
	for _, item := range keys.Items {
		if item.URI == added.URI && item.Key == keyText {
			found = true
		}
	}
	if !found {
		t.Fatal("created key missing from listing")
	}
	if _, err := service.DeleteSSHKey(ctx, extractRKey(added.URI)); err != nil {
		t.Fatal(err)
	}
}

func stackGit(t *testing.T, ctx context.Context, dir string, args ...string) string {
	t.Helper()
	command := exec.CommandContext(ctx, "git", args...)
	command.Dir = dir
	command.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1", "GIT_TERMINAL_PROMPT=0", "GIT_SSL_CAINFO="+os.Getenv("SSL_CERT_FILE"))
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("fixture git failed: %v\n%s", err, output)
	}
	return strings.TrimSpace(string(output))
}

func TestUpstreamPullRoundTrip(t *testing.T) {
	service, ctx, actor := upstreamService(t)
	t.Setenv("GIT_SSL_CAINFO", os.Getenv("SSL_CERT_FILE"))
	t.Setenv("GIT_CONFIG_GLOBAL", "/dev/null")
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	name := fmt.Sprintf("tg-pull-%d", time.Now().UnixNano())
	created, err := service.CreateRepo(ctx, CreateRepoInput{Name: name, KnotHost: "knot2.tngl.boltless.dev"})
	if err != nil {
		t.Fatal(err)
	}
	target := Target{Handle: actor, Repo: name}
	did := strings.Split(created.URI, "/")[2]
	dir := t.TempDir()
	t.Chdir(dir)
	stackGit(t, ctx, dir, "init", "--object-format=sha256", "-b", "main")
	stackGit(t, ctx, dir, "config", "user.name", "Alice")
	stackGit(t, ctx, dir, "config", "user.email", "alice@example.test")
	write := func(contents string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte(contents), 0600); err != nil {
			t.Fatal(err)
		}
		stackGit(t, ctx, dir, "add", "README.md")
		stackGit(t, ctx, dir, "-c", "commit.gpgsign=false", "commit", "-m", "fixture change")
	}
	if os.Getenv("TG_UPSTREAM_SPINDLE_INTEGRATION") == "1" {
		configureUpstreamPipeline(t, service, ctx, target, dir)
	}
	write("Base\n")
	stackGit(t, ctx, dir, "checkout", "-b", "proposal")
	write("Base\nProposal\n")
	stackGit(t, ctx, dir, "remote", "add", "origin", "https://knot2.tngl.boltless.dev/"+did)
	push := func() {
		t.Helper()
		credential, err := service.GitPushToken(ctx, "knot2.tngl.boltless.dev")
		if err != nil {
			t.Fatal(err)
		}
		// Pass the short-lived credential through the environment, never the command line or logs.
		command := exec.CommandContext(ctx, "git", "push", "origin", "main", "proposal")
		command.Dir = dir
		command.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1", "GIT_TERMINAL_PROMPT=0", "GIT_SSL_CAINFO="+os.Getenv("SSL_CERT_FILE"), "GIT_CONFIG_COUNT=1", "GIT_CONFIG_KEY_0=http.extraHeader", "GIT_CONFIG_VALUE_0=Authorization: Bearer "+credential.Token)
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("fixture push failed: %v\n%s", err, output)
		}
		head := stackGit(t, ctx, dir, "rev-parse", "proposal")
		deadline := time.Now().Add(30 * time.Second)
		for {
			published, err := service.appview.GetRef(ctx, did, "refs/heads/proposal")
			if err == nil && published == head {
				break
			}
			if time.Now().After(deadline) {
				t.Fatalf("mirror did not publish proposal: %v", err)
			}
			select {
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			case <-time.After(200 * time.Millisecond):
			}
		}
	}
	push()
	if _, err := service.SetRepoDefaultBranch(ctx, target, "main"); err != nil {
		t.Fatal(err)
	}
	if os.Getenv("TG_UPSTREAM_SPINDLE_INTEGRATION") == "1" {
		verifyUpstreamPipeline(t, service, ctx, target, dir)
	}
	pull, err := service.CreatePull(ctx, CreatePullInput{Target: target, RepoDir: dir, Title: "Verify pull", Body: "Disposable proposal", Base: "main", Head: "proposal"})
	if err != nil || pull.CID == "" || len(pull.Warnings) != 0 {
		t.Fatalf("pull=%+v err=%v", pull, err)
	}
	if os.Getenv("TG_UPSTREAM_SPINDLE_INTEGRATION") == "1" {
		verifyUpstreamPullPipeline(t, service, ctx, target, pull.URI, stackGit(t, ctx, dir, "rev-parse", "proposal"))
	}
	key := extractRKey(pull.URI)
	view, err := service.ViewPull(ctx, target, key)
	if err != nil || view.SourceBranch != "proposal" || view.TargetBranch != "main" {
		t.Fatalf("pull view=%+v err=%v", view, err)
	}
	patch, err := service.PullPatch(ctx, target, key)
	if err != nil || !bytes.Contains(patch.Patch, []byte("+Proposal")) {
		t.Fatalf("patch=%+v err=%v", patch, err)
	}
	editedTitle := "Edited pull"
	edited, err := service.EditPull(ctx, target, key, &editedTitle, nil, pull.CID)
	if err != nil || len(edited.Warnings) != 0 {
		t.Fatalf("pull edit=%+v err=%v", edited, err)
	}
	commented, err := service.CommentPull(ctx, target, key, "Verified patch comment")
	if err != nil || len(commented.Warnings) != 0 {
		t.Fatalf("pull comment=%+v err=%v", commented, err)
	}
	closed, err := service.SetPullState(ctx, target, key, "closed", edited.CID)
	if err != nil || len(closed.Warnings) != 0 {
		t.Fatalf("pull close=%+v err=%v", closed, err)
	}
	reopened, err := service.SetPullState(ctx, target, key, "open", closed.CID)
	if err != nil || len(reopened.Warnings) != 0 {
		t.Fatalf("pull reopen=%+v err=%v", reopened, err)
	}
	checked, err := service.CheckoutPull(ctx, CheckoutPullInput{Target: &target, RepoDir: dir, Rkey: key, Branch: "verified-checkout"})
	if err != nil || checked.Branch != "verified-checkout" {
		t.Fatalf("checkout=%+v err=%v", checked, err)
	}
	stackGit(t, ctx, dir, "checkout", "proposal")
	forkName := name + "-fork"
	fork, err := service.ForkRepo(ctx, target, forkName)
	if err != nil || fork.ImportStatus == "" {
		t.Fatalf("fork=%+v err=%v", fork, err)
	}
	forkDID := strings.Split(fork.URI, "/")[2]
	deadline := time.Now().Add(30 * time.Second)
	for {
		status, err := service.knot.NewPublic("knot2.tngl.boltless.dev").GetImportStatus(ctx, forkDID)
		if err != nil {
			t.Fatal(err)
		}
		if status.Status == "present" {
			break
		}
		if status.Status == "failed" || time.Now().After(deadline) {
			t.Fatalf("fork import=%+v", status)
		}
		select {
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		case <-time.After(200 * time.Millisecond):
		}
	}
	cloneDir := filepath.Join(t.TempDir(), "clone")
	cloned, err := service.CloneRepo(ctx, CloneRepoInput{RepoDID: forkDID, KnotHost: "knot2.tngl.boltless.dev", Protocol: "https", Destination: cloneDir})
	if err != nil || cloned.RepoDID != forkDID {
		t.Fatalf("clone=%+v err=%v", cloned, err)
	}
	if format := stackGit(t, ctx, cloneDir, "rev-parse", "--show-object-format"); format != "sha256" {
		t.Fatalf("fork changed object format: %s", format)
	}
	if _, err := service.DeleteRepo(ctx, Target{Handle: actor, Repo: forkName}); err != nil {
		t.Fatal(err)
	}
	write("Base\nProposal\nUpdated\n")
	push()
	updated, err := service.UpdatePullRound(ctx, UpdatePullRoundInput{Target: target, RepoDir: dir, Rkey: key, Base: "main", Head: "proposal"})
	if err != nil || len(updated.Warnings) != 0 {
		t.Fatalf("pull update=%+v err=%v", updated, err)
	}
	merged, err := service.MergePull(ctx, target, key)
	if err != nil || !merged.Merged || !merged.StatusRecorded || len(merged.Warnings) != 0 {
		t.Fatalf("merge=%+v err=%v", merged, err)
	}
	pending, err := service.ListPendingAttestations(ctx)
	if err != nil || len(pending.Items) != 0 {
		t.Fatalf("pull attestations=%+v err=%v", pending, err)
	}
	if _, err := service.DeleteRepo(ctx, target); err != nil {
		t.Fatal(err)
	}
}
