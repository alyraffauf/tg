package app

import (
	"context"
	"strings"
	"testing"

	"github.com/alyraffauf/tg/internal/gitutil"
	"github.com/alyraffauf/tg/tangled"
)

func TestGitPushToken(t *testing.T) {
	pds := &testPDS{}
	service := testService(pds, &testGit{repoCandidates: []gitutil.RepoContext{{Handle: "owner.test", Repo: "repo"}}}, &testKnot{})
	service.appview = testAppview{repo: &tangled.Repo{Value: tangled.RepoMetadata{Knot: "knot.example", Name: optionalString("repo")}}}

	credentials, err := service.GitPushToken(context.Background(), "KNOT.EXAMPLE")
	if err != nil {
		t.Fatalf("GitPushToken() error = %v", err)
	}
	if !credentials.MatchesRequestedHost {
		t.Fatal("GitPushToken() did not match the repository knot")
	}
	if credentials.Token != "token" {
		t.Fatalf("token = %q, want %q", credentials.Token, "token")
	}
	if credentials.Handle != "owner.test" {
		t.Fatalf("handle = %q, want %q", credentials.Handle, "owner.test")
	}
	if len(pds.serviceAuthAudiences) != 1 || pds.serviceAuthAudiences[0] != "did:web:knot.example" {
		t.Fatalf("audiences = %v, want [did:web:knot.example]", pds.serviceAuthAudiences)
	}
	if len(pds.serviceAuthLexiconMethods) != 1 || pds.serviceAuthLexiconMethods[0] != "org.tangled.git.receivePack" {
		t.Fatalf("lexicon methods = %v, want [org.tangled.git.receivePack]", pds.serviceAuthLexiconMethods)
	}
}

func TestGitPushTokenAcceptsHostedGitProxy(t *testing.T) {
	pds := &testPDS{}
	service := testService(pds, &testGit{repoCandidates: []gitutil.RepoContext{{Handle: "owner.test", Repo: "repo"}}}, &testKnot{})
	service.appview = testAppview{repo: &tangled.Repo{Value: tangled.RepoMetadata{Knot: "knot.example", Name: optionalString("repo")}}}

	credentials, err := service.GitPushToken(context.Background(), gitutil.HostedGitHost)
	if err != nil {
		t.Fatalf("GitPushToken() error = %v", err)
	}
	if !credentials.MatchesRequestedHost || credentials.Token != "token" {
		t.Fatalf("GitPushToken() = %+v", credentials)
	}
	if len(pds.serviceAuthAudiences) != 1 || pds.serviceAuthAudiences[0] != "did:web:knot.example" {
		t.Fatalf("audiences = %v, want [did:web:knot.example]", pds.serviceAuthAudiences)
	}
}

func TestGitPushTokenIgnoresOtherHosts(t *testing.T) {
	pds := &testPDS{}
	service := testService(pds, &testGit{repoCandidates: []gitutil.RepoContext{{Handle: "owner.test", Repo: "repo"}}}, &testKnot{})
	service.appview = testAppview{repo: &tangled.Repo{Value: tangled.RepoMetadata{Knot: "knot.example", Name: optionalString("repo")}}}

	credentials, err := service.GitPushToken(context.Background(), "other.example")
	if err != nil {
		t.Fatalf("GitPushToken() error = %v", err)
	}
	if credentials.MatchesRequestedHost {
		t.Fatal("GitPushToken() matched an unrelated host")
	}
	if credentials.Token != "" || credentials.Handle != "" {
		t.Fatalf("credentials = %+v, want empty", credentials)
	}
	if pds.serviceAuthCalls != 0 {
		t.Fatalf("service auth calls = %d, want 0", pds.serviceAuthCalls)
	}
}

func TestGitPushTokenReportsMissingOAuthPushScope(t *testing.T) {
	pds := &testPDS{}
	service := testService(pds, &testGit{repoCandidates: []gitutil.RepoContext{{Handle: "owner.test", Repo: "repo"}}}, &testKnot{})
	service.sessions = testSessions{pds: pds, isOAuth: true}
	service.appview = testAppview{repo: &tangled.Repo{Value: tangled.RepoMetadata{Knot: "knot.example", Name: optionalString("repo")}}}

	_, err := service.GitPushToken(context.Background(), "knot.example")
	if err == nil || !strings.Contains(err.Error(), `run "tg auth login" again`) {
		t.Fatalf("GitPushToken() error = %v", err)
	}
	if pds.serviceAuthCalls != 0 {
		t.Fatalf("service auth calls = %d, want 0", pds.serviceAuthCalls)
	}
}

func TestGitPushTokenPreservesHTTPSAuthority(t *testing.T) {
	for _, requested := range []string{"knot.example:8443", "knot.example"} {
		service, pds, _ := migrationService(t)
		service.git = &testGit{repoCandidates: []gitutil.RepoContext{{RepoDID: "did:plc:repo", KnotHost: "knot.example:8443"}}}
		service.resolver = testResolver{identity: repositoryIdentity("did:plc:repo", "owner.test", "https://knot.example:8443")}
		credentials, err := service.GitPushToken(context.Background(), requested)
		if err != nil {
			t.Fatal(err)
		}
		want := requested == "knot.example:8443"
		if credentials.MatchesRequestedHost != want {
			t.Fatalf("credentials=%+v", credentials)
		}
		if want && (len(pds.serviceAuthAudiences) != 1 || pds.serviceAuthAudiences[0] != "did:web:knot.example%3A8443") {
			t.Fatalf("auth=%v", pds.serviceAuthAudiences)
		}
		if !want && pds.serviceAuthCalls != 0 {
			t.Fatal("issued credentials for a different HTTPS port")
		}
	}
}
