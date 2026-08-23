package gitutil

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolvePullBaseDistinguishesLocalAndRemoteBranches(t *testing.T) {
	repoDir := newPullBaseTestRepo(t)
	rootCommit := gitOutputForTest(t, repoDir, "rev-parse", "HEAD")
	runGit(t, repoDir, "remote", "add", "origin", "https://example.com/fork.git")
	runGit(t, repoDir, "update-ref", "refs/remotes/origin/main", rootCommit)
	runGit(t, repoDir, "remote", "add", "upstream", "https://example.com/upstream.git")
	runGit(t, repoDir, "update-ref", "refs/remotes/upstream/main", rootCommit)
	runGit(t, repoDir, "update-ref", "refs/remotes/upstream/release/2026", rootCommit)

	commitTestFile(t, repoDir, "local.txt", "local\n", "local main")
	localCommit := gitOutputForTest(t, repoDir, "rev-parse", "HEAD")
	runGit(t, repoDir, "branch", "release/2026", localCommit)

	tests := []struct {
		name         string
		base         string
		wantRevision string
		wantBranch   string
	}{
		{name: "local branch", base: "main", wantRevision: localCommit, wantBranch: "main"},
		{name: "local branch with slash", base: "release/2026", wantRevision: localCommit, wantBranch: "release/2026"},
		{name: "remote branch", base: "upstream/main", wantRevision: rootCommit, wantBranch: "main"},
		{name: "remote branch with slash", base: "upstream/release/2026", wantRevision: rootCommit, wantBranch: "release/2026"},
		{name: "full local ref", base: "refs/heads/release/2026", wantRevision: localCommit, wantBranch: "release/2026"},
		{name: "full remote ref", base: "refs/remotes/upstream/release/2026", wantRevision: rootCommit, wantBranch: "release/2026"},
	}

	client := NewClient(nil, nil)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base, err := client.ResolvePullBase(context.Background(), repoDir, tt.base)
			if err != nil {
				t.Fatalf("ResolvePullBase() error = %v", err)
			}
			if base.Revision != tt.wantRevision || base.Branch != tt.wantBranch {
				t.Fatalf("ResolvePullBase() = %+v, want revision %q and branch %q", base, tt.wantRevision, tt.wantBranch)
			}
		})
	}
}

func TestResolvePullBaseRejectsAmbiguousAndNonBranchRevisions(t *testing.T) {
	repoDir := newPullBaseTestRepo(t)
	rootCommit := gitOutputForTest(t, repoDir, "rev-parse", "HEAD")
	runGit(t, repoDir, "remote", "add", "origin", "https://example.com/fork.git")
	runGit(t, repoDir, "update-ref", "refs/remotes/origin/remote-only", rootCommit)
	runGit(t, repoDir, "remote", "add", "upstream", "https://example.com/upstream.git")
	runGit(t, repoDir, "update-ref", "refs/remotes/upstream/main", rootCommit)
	runGit(t, repoDir, "branch", "upstream/main", rootCommit)
	runGit(t, repoDir, "update-ref", "refs/heads/HEAD", rootCommit)
	runGit(t, repoDir, "update-ref", "refs/remotes/origin/HEAD", rootCommit)
	runGit(t, repoDir, "tag", "v1", rootCommit)

	client := NewClient(nil, nil)
	tests := []struct {
		name      string
		base      string
		wantError string
	}{
		{name: "ambiguous local and remote", base: "upstream/main", wantError: "ambiguous"},
		{name: "tag", base: "v1", wantError: "local branch or configured remote-tracking branch"},
		{name: "commit", base: rootCommit, wantError: "local branch or configured remote-tracking branch"},
		{name: "head", base: "HEAD", wantError: "local branch or configured remote-tracking branch"},
		{name: "direct local head ref", base: "refs/heads/HEAD", wantError: "reserved branch name HEAD"},
		{name: "direct remote head ref", base: "origin/HEAD", wantError: "reserved branch name HEAD"},
		{name: "revision expression", base: "main~1", wantError: "local branch or configured remote-tracking branch"},
		{name: "unqualified remote branch", base: "remote-only", wantError: "local branch or configured remote-tracking branch"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.ResolvePullBase(context.Background(), repoDir, tt.base)
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("ResolvePullBase() error = %v, want error containing %q", err, tt.wantError)
			}
		})
	}
}

func TestDefaultPullBaseUsesOriginHead(t *testing.T) {
	repoDir := newPullBaseTestRepo(t)
	rootCommit := gitOutputForTest(t, repoDir, "rev-parse", "HEAD")
	runGit(t, repoDir, "remote", "add", "origin", "https://example.com/fork.git")
	runGit(t, repoDir, "update-ref", "refs/remotes/origin/main", rootCommit)
	runGit(t, repoDir, "symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/main")
	commitTestFile(t, repoDir, "local.txt", "local\n", "local main")

	base, err := NewClient(nil, nil).DefaultPullBase(context.Background(), repoDir)
	if err != nil {
		t.Fatalf("DefaultPullBase() error = %v", err)
	}
	if base.Revision != rootCommit || base.Branch != "main" {
		t.Fatalf("DefaultPullBase() = %+v, want revision %q and branch main", base, rootCommit)
	}
}

func newPullBaseTestRepo(t *testing.T) string {
	t.Helper()
	tempDir := t.TempDir()
	repoDir := filepath.Join(tempDir, "repo")
	runGit(t, tempDir, "init", "--initial-branch=main", repoDir)
	runGit(t, repoDir, "config", "user.name", "Test User")
	runGit(t, repoDir, "config", "user.email", "test@example.com")
	commitTestFile(t, repoDir, "root.txt", "root\n", "root")
	return repoDir
}

func commitTestFile(t *testing.T, repoDir, name, contents, message string) {
	t.Helper()
	writeTestFile(t, filepath.Join(repoDir, name), contents)
	runGit(t, repoDir, "add", name)
	runGit(t, repoDir, "commit", "-m", message)
}
