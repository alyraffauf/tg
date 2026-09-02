package gitutil

import (
	"context"
	"strings"
	"testing"
)

func TestPushNewRepoRejectsUnsafeRemoteNames(t *testing.T) {
	client := NewClient(nil, nil)
	for _, name := range []string{"", "--upload-pack=bad", "bad\nname"} {
		if err := client.PushNewRepo(context.Background(), PushNewRepoParams{RemoteName: name}); err == nil {
			t.Fatalf("PushNewRepo() accepted remote name %q", name)
		}
	}
}

func TestPushNewRepoRemovesRemoteAfterFailedPush(t *testing.T) {
	repository := t.TempDir()
	runGit(t, repository, "init")
	client := NewClient(nil, nil)
	err := client.PushNewRepo(context.Background(), PushNewRepoParams{
		Dir: repository, KnotHost: "127.0.0.1", SSHPort: 1,
		RepoDID: "did:plc:abc123", RemoteName: "upstream",
	})
	if err == nil || !strings.Contains(err.Error(), "push to \"upstream\"") {
		t.Fatalf("PushNewRepo() error = %v", err)
	}
	remotes, err := client.gitLines(context.Background(), "-C", repository, "remote")
	if err != nil {
		t.Fatalf("list remotes: %v", err)
	}
	if len(remotes) != 0 {
		t.Fatalf("remotes after rollback = %q", remotes)
	}
}

func TestParseRepoCandidateRejectsMalformedHTTPSIdentity(t *testing.T) {
	for _, remote := range []string{
		"https:///owner/repo",
		"https://user@knot.example/owner/repo",
		"https://knot.example:8443/owner/repo",
		"https://knot.example/owner/repo?token=secret",
	} {
		if candidate, ok := parseRepoCandidate(remote); ok {
			t.Fatalf("parseRepoCandidate(%q) = %+v", remote, candidate)
		}
	}
}
