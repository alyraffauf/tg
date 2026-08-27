package app

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/alyraffauf/tg/internal/tangledlex"
	"github.com/alyraffauf/tg/knot"
	"github.com/alyraffauf/tg/tangled"
)

func TestCloneRepoUsesRepositoryDIDForHTTPS(t *testing.T) {
	gitClient := &testGit{}
	service := testService(&testPDS{}, gitClient, &testKnot{})
	service.appview = testAppview{repo: &tangled.Repo{Value: tangledlex.Repo{
		Knot: "knot.example", RepoDid: optionalString("did:plc:repo"),
	}}}

	_, err := service.CloneRepo(context.Background(), CloneRepoInput{
		Protocol: "https", Handle: "owner.test", Repo: "repo", Destination: "repo",
	})
	if err != nil {
		t.Fatalf("CloneRepo() error = %v", err)
	}
	if len(gitClient.clones) != 1 {
		t.Fatalf("clone calls = %v, want one", gitClient.clones)
	}
	if gitClient.clones[0].KnotHost != "" || gitClient.clones[0].Protocol != "https" || gitClient.clones[0].RepoDID != "did:plc:repo" {
		t.Fatalf("clone input = %+v", gitClient.clones[0])
	}
}

func TestCloneRepoResolvesRepositoryDIDInput(t *testing.T) {
	const repoDID = "did:plc:repository"
	gitClient := &testGit{}
	knotClient := &testKnot{description: &knot.RepoDescription{
		RepoDID: repoDID, OwnerDID: "did:plc:owner", RKey: "repo-rkey",
	}}
	service := testService(&testPDS{}, gitClient, knotClient)
	service.resolver = testResolver{
		identity: repositoryIdentity(repoDID, "owner.test", "https://knot.example"),
	}
	service.appview = testAppview{repo: &tangled.Repo{URI: "at://did:plc:owner/sh.tangled.repo/repo-rkey", Value: tangledlex.Repo{
		Name: optionalString("current-name"), Knot: "knot.example", RepoDid: optionalString(repoDID),
	}}}

	result, err := service.CloneRepo(context.Background(), CloneRepoInput{Protocol: "ssh", RepoDID: repoDID})
	if err != nil {
		t.Fatalf("CloneRepo() error = %v", err)
	}
	if result.Handle != "owner.test" || result.Repo != "current-name" || result.Destination != "current-name" || result.RepoDID != repoDID {
		t.Fatalf("CloneRepo() = %+v", result)
	}
	if len(gitClient.clones) != 1 || gitClient.clones[0].RepoDID != repoDID {
		t.Fatalf("clone calls = %+v", gitClient.clones)
	}
}

func TestCloneRepoUsesRecordKeyWhenRepositoryNameIsEmpty(t *testing.T) {
	const repoDID = "did:plc:repository"
	gitClient := &testGit{}
	service := testService(&testPDS{}, gitClient, &testKnot{description: &knot.RepoDescription{
		RepoDID: repoDID, OwnerDID: "did:plc:owner", RKey: "repo-rkey",
	}})
	service.resolver = testResolver{
		identity: repositoryIdentity(repoDID, "owner.test", "https://knot.example"),
	}
	service.appview = testAppview{repo: &tangled.Repo{URI: "at://did:plc:owner/sh.tangled.repo/repo-rkey", Value: tangledlex.Repo{
		Knot: "knot.example", RepoDid: optionalString(repoDID),
	}}}

	result, err := service.CloneRepo(context.Background(), CloneRepoInput{Protocol: "ssh", RepoDID: repoDID})
	if err != nil {
		t.Fatalf("CloneRepo() error = %v", err)
	}
	if result.Repo != "repo-rkey" || result.Destination != "repo-rkey" {
		t.Fatalf("CloneRepo() = %+v", result)
	}
}

func TestCloneRepoRejectsUnsafeDefaultDestination(t *testing.T) {
	const repoDID = "did:plc:repository"
	tests := []string{"../outside", "/tmp/outside", "--bare", ".", "..", "nested/repo", "bad\x00name"}
	for _, repoName := range tests {
		t.Run(repoName, func(t *testing.T) {
			gitClient := &testGit{}
			service := testService(&testPDS{}, gitClient, &testKnot{description: &knot.RepoDescription{
				RepoDID: repoDID, OwnerDID: "did:plc:owner", RKey: "repo-rkey",
			}})
			service.resolver = testResolver{
				identity: repositoryIdentity(repoDID, "owner.test", "https://knot.example"),
			}
			service.appview = testAppview{repo: &tangled.Repo{URI: "at://did:plc:owner/sh.tangled.repo/repo-rkey", Value: tangledlex.Repo{
				Name: optionalString(repoName), Knot: "knot.example", RepoDid: optionalString(repoDID),
			}}}

			_, err := service.CloneRepo(context.Background(), CloneRepoInput{Protocol: "ssh", RepoDID: repoDID})
			if err == nil || !strings.Contains(err.Error(), "provide an explicit directory") {
				t.Fatalf("CloneRepo() error = %v, want explicit-directory hint", err)
			}
			if len(gitClient.clones) != 0 {
				t.Fatalf("clone calls = %+v, want none", gitClient.clones)
			}
		})
	}
}

func TestCloneRepoPreservesExplicitDestination(t *testing.T) {
	const repoDID = "did:plc:repository"
	tests := []string{"../work", "/tmp/work", "--bare"}
	for _, destination := range tests {
		t.Run(destination, func(t *testing.T) {
			gitClient := &testGit{}
			service := testService(&testPDS{}, gitClient, &testKnot{description: &knot.RepoDescription{
				RepoDID: repoDID, OwnerDID: "did:plc:owner", RKey: "repo-rkey",
			}})
			service.resolver = testResolver{
				identity: repositoryIdentity(repoDID, "owner.test", "https://knot.example"),
			}
			service.appview = testAppview{repo: &tangled.Repo{URI: "at://did:plc:owner/sh.tangled.repo/repo-rkey", Value: tangledlex.Repo{
				Name: optionalString(destination), Knot: "knot.example", RepoDid: optionalString(repoDID),
			}}}

			result, err := service.CloneRepo(context.Background(), CloneRepoInput{
				Protocol: "ssh", RepoDID: repoDID, Destination: destination,
			})
			if err != nil {
				t.Fatalf("CloneRepo() error = %v", err)
			}
			if result.Destination != destination || len(gitClient.clones) != 1 || gitClient.clones[0].RepoDir != destination {
				t.Fatalf("CloneRepo() = %+v, clone calls = %+v", result, gitClient.clones)
			}
		})
	}
}

func TestCloneRepoFallsBackToHandleForSSH(t *testing.T) {
	gitClient := &testGit{}
	service := testService(&testPDS{}, gitClient, &testKnot{})
	service.appview = testAppview{repoErr: errors.New("appview unavailable")}

	result, err := service.CloneRepo(context.Background(), CloneRepoInput{
		Protocol: "ssh", Handle: "owner.test", Repo: "repo", Destination: "repo",
	})
	if err != nil {
		t.Fatalf("CloneRepo() error = %v", err)
	}
	if len(result.Warnings) != 1 || !strings.Contains(result.Warnings[0], "handle-based remote") {
		t.Fatalf("CloneRepo() warnings = %v", result.Warnings)
	}
	if len(gitClient.clones) != 1 || gitClient.clones[0].RepoDID != "" {
		t.Fatalf("clone calls = %+v", gitClient.clones)
	}
}

func TestCloneRepoRejectsRepositoryRecordWithInvalidDID(t *testing.T) {
	gitClient := &testGit{}
	service := testService(&testPDS{}, gitClient, &testKnot{})
	service.appview = testAppview{repo: &tangled.Repo{Value: tangledlex.Repo{
		RepoDid: optionalString("not-a-did"),
	}}}

	_, err := service.CloneRepo(context.Background(), CloneRepoInput{
		Protocol: "ssh", Handle: "owner.test", Repo: "repo",
	})
	if err == nil || !strings.Contains(err.Error(), `invalid repository DID "not-a-did"`) {
		t.Fatalf("CloneRepo() error = %v", err)
	}
	if len(gitClient.clones) != 0 {
		t.Fatalf("clone calls = %+v, want none", gitClient.clones)
	}
}

func TestCloneRepoRejectsUnsupportedProtocol(t *testing.T) {
	service := testService(&testPDS{}, &testGit{}, &testKnot{})

	_, err := service.CloneRepo(context.Background(), CloneRepoInput{Protocol: "git", Handle: "owner.test", Repo: "repo"})
	if err == nil || err.Error() != "clone protocol must be \"ssh\" or \"https\", got \"git\"" {
		t.Fatalf("CloneRepo() error = %v", err)
	}
}

func TestCloneRepoRejectsMixedOrIncompleteIdentity(t *testing.T) {
	tests := []struct {
		name  string
		input CloneRepoInput
		want  string
	}{
		{
			name:  "DID and handle name",
			input: CloneRepoInput{Protocol: "ssh", RepoDID: "did:plc:repo", Handle: "owner.test", Repo: "repo"},
			want:  "cannot be combined",
		},
		{
			name:  "missing repository name",
			input: CloneRepoInput{Protocol: "ssh", Handle: "owner.test"},
			want:  "either a repository DID or both a handle and repository name",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gitClient := &testGit{}
			service := testService(&testPDS{}, gitClient, &testKnot{})
			_, err := service.CloneRepo(context.Background(), tt.input)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("CloneRepo() error = %v, want containing %q", err, tt.want)
			}
			if len(gitClient.clones) != 0 {
				t.Fatalf("clone calls = %+v, want none", gitClient.clones)
			}
		})
	}
}
