package gitutil

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateAndCheckoutPatch(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	originDir := filepath.Join(tempDir, "origin.git")
	repoDir := filepath.Join(tempDir, "repo")
	runGit(t, tempDir, "init", "--bare", originDir)
	runGit(t, tempDir, "clone", originDir, repoDir)
	runGit(t, repoDir, "config", "user.name", "Test User")
	runGit(t, repoDir, "config", "user.email", "test@example.com")
	runGit(t, repoDir, "switch", "-c", "main")

	writeTestFile(t, filepath.Join(repoDir, "message.txt"), "base\n")
	runGit(t, repoDir, "add", "message.txt")
	runGit(t, repoDir, "commit", "-m", "base")
	runGit(t, repoDir, "push", "-u", "origin", "main")

	runGit(t, repoDir, "switch", "-c", "feature")
	writeTestFile(t, filepath.Join(repoDir, "message.txt"), "feature\n")
	runGit(t, repoDir, "add", "message.txt")
	runGit(t, repoDir, "commit", "-m", "feature")
	compressedPatch, err := GeneratePatch(ctx, repoDir, "main", "feature")
	if err != nil {
		t.Fatalf("GeneratePatch() error = %v", err)
	}
	patch := decompressTestPatch(t, compressedPatch)

	err = CheckoutPatch(ctx, CheckoutPatchParams{
		RepoDir:      repoDir,
		Branch:       "review",
		TargetBranch: "main",
		Patch:        patch,
	})
	if err != nil {
		t.Fatalf("CheckoutPatch() error = %v", err)
	}
	if branch := gitOutputForTest(t, repoDir, "branch", "--show-current"); branch != "review" {
		t.Fatalf("checked out branch = %q, want review", branch)
	}
	contents, err := os.ReadFile(filepath.Join(repoDir, "message.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != "feature\n" {
		t.Fatalf("checked out contents = %q, want feature", contents)
	}
}

func TestCheckoutPatchRejectsDirtyWorktree(t *testing.T) {
	repoDir := t.TempDir()
	runGit(t, repoDir, "init")
	writeTestFile(t, filepath.Join(repoDir, "untracked.txt"), "dirty\n")

	err := CheckoutPatch(context.Background(), CheckoutPatchParams{RepoDir: repoDir})
	if err == nil || !strings.Contains(err.Error(), "uncommitted changes") {
		t.Fatalf("CheckoutPatch() error = %v, want uncommitted changes error", err)
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, output)
	}
}

func gitOutputForTest(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, output)
	}
	return strings.TrimSpace(string(output))
}

func writeTestFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}

func decompressTestPatch(t *testing.T, compressed []byte) []byte {
	t.Helper()
	reader, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	patch, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	return patch
}
