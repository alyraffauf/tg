package app

import (
	"context"
	"strings"
	"testing"

	"github.com/alyraffauf/tg/internal/gitutil"
	"github.com/alyraffauf/tg/internal/tangledlex"
	"github.com/alyraffauf/tg/tangled"
)

func TestGitPushToken(t *testing.T) {
	pds := &testPDS{}
	service := testService(pds, &testGit{repoCandidates: []gitutil.RepoContext{{Handle: "owner.test", Repo: "repo"}}}, &testKnot{})
	service.appview = testAppview{repo: &tangled.Repo{Value: tangledlex.Repo{Knot: "knot.example"}}}

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
	if len(pds.serviceAuthLexiconMethods) != 1 || pds.serviceAuthLexiconMethods[0] != "sh.tangled.repo.push" {
		t.Fatalf("lexicon methods = %v, want [sh.tangled.repo.push]", pds.serviceAuthLexiconMethods)
	}
}

func TestGitPushTokenIgnoresOtherHosts(t *testing.T) {
	pds := &testPDS{}
	service := testService(pds, &testGit{repoCandidates: []gitutil.RepoContext{{Handle: "owner.test", Repo: "repo"}}}, &testKnot{})
	service.appview = testAppview{repo: &tangled.Repo{Value: tangledlex.Repo{Knot: "knot.example"}}}

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
	service.appview = testAppview{repo: &tangled.Repo{Value: tangledlex.Repo{Knot: "knot.example"}}}

	_, err := service.GitPushToken(context.Background(), "knot.example")
	if err == nil || !strings.Contains(err.Error(), `run "tg auth login" again`) {
		t.Fatalf("GitPushToken() error = %v", err)
	}
	if pds.serviceAuthCalls != 0 {
		t.Fatalf("service auth calls = %d, want 0", pds.serviceAuthCalls)
	}
}
