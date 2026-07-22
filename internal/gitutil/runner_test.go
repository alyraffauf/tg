package gitutil

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestGitCommandStreamsDiagnosticsWithoutDuplicatingThemInErrors(t *testing.T) {
	var stderr bytes.Buffer
	client := NewClient(&bytes.Buffer{}, &stderr)

	err := client.gitCommand(context.Background(), t.TempDir(), "not-a-command")
	if err == nil {
		t.Fatal("gitCommand() error = nil")
	}
	diagnostic := strings.TrimSpace(stderr.String())
	if diagnostic == "" {
		t.Fatal("gitCommand() did not stream diagnostics")
	}
	if strings.Contains(err.Error(), diagnostic) {
		t.Fatalf("gitCommand() error duplicated diagnostics: %q", err)
	}
}

func TestApplyPatchStreamsDiagnosticsWithoutDuplicatingThemInErrors(t *testing.T) {
	repoDir := t.TempDir()
	runGit(t, repoDir, "init")
	var stderr bytes.Buffer
	client := NewClient(&bytes.Buffer{}, &stderr)

	err := client.applyPatch(context.Background(), repoDir, []byte("not a patch"))
	if err == nil {
		t.Fatal("applyPatch() error = nil")
	}
	diagnostic := strings.TrimSpace(stderr.String())
	if diagnostic == "" {
		t.Fatal("applyPatch() did not stream diagnostics")
	}
	if strings.Contains(err.Error(), diagnostic) {
		t.Fatalf("applyPatch() error duplicated diagnostics: %q", err)
	}
}
