package app

import (
	"context"
	"errors"
	"strings"
	"testing"
)

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
