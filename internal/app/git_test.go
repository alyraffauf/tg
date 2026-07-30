package app

import (
	"context"
	"testing"

	"github.com/alyraffauf/tg/internal/tangledlex"
	"github.com/alyraffauf/tg/tangled"
)

func TestCloneRepoResolvesKnotForHTTPS(t *testing.T) {
	gitClient := &testGit{}
	service := testService(&testPDS{}, gitClient, &testKnot{})
	service.appview = testAppview{repo: &tangled.Repo{Value: tangledlex.Repo{Knot: "knot.example"}}}

	_, err := service.CloneRepo(context.Background(), CloneRepoInput{
		Protocol: "https", Handle: "owner.test", Repo: "repo", Destination: "repo",
	})
	if err != nil {
		t.Fatalf("CloneRepo() error = %v", err)
	}
	if len(gitClient.clones) != 1 {
		t.Fatalf("clone calls = %v, want one", gitClient.clones)
	}
	if gitClient.clones[0].KnotHost != "knot.example" || gitClient.clones[0].Protocol != "https" {
		t.Fatalf("clone input = %+v", gitClient.clones[0])
	}
}

func TestCloneRepoRejectsUnsupportedProtocol(t *testing.T) {
	service := testService(&testPDS{}, &testGit{}, &testKnot{})

	_, err := service.CloneRepo(context.Background(), CloneRepoInput{Protocol: "git", Handle: "owner.test", Repo: "repo"})
	if err == nil || err.Error() != "clone protocol must be \"ssh\" or \"https\", got \"git\"" {
		t.Fatalf("CloneRepo() error = %v", err)
	}
}
